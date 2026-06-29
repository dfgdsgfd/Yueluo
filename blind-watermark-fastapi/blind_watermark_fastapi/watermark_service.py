import asyncio
import json
import multiprocessing
import threading
import time
from collections.abc import Callable
from queue import Empty

import anyio
from fastapi import HTTPException, UploadFile
from fastapi.responses import StreamingResponse

from .settings import Settings
from .watermark import (
    WatermarkError,
    embed_payload,
    encode_payload_b64,
    extract_payload_candidate_groups,
)

_limiter_lock = threading.Lock()
_limiter_workers = 0
_limiter: anyio.CapacityLimiter | None = None

def _operation_stream(
    operation: str,
    args: tuple[object, ...],
    settings: Settings,
    operation_timeout_seconds: int,
    response_builder: Callable[[object], dict[str, object]],
    fallback_error: str,
) -> StreamingResponse:
    events: asyncio.Queue[dict[str, object]] = asyncio.Queue()
    started_at = time.monotonic()
    latest: dict[str, object] = {
        "stage": "queued",
        "percent": 1,
        "completed": 0,
        "total": 0,
    }

    async def report(event: dict[str, object]) -> None:
        await events.put({"type": "progress", **event})

    async def run() -> None:
        try:
            result = await _run_watermark(
                operation,
                args,
                settings,
                operation_timeout_seconds,
                progress=report,
            )
        except TimeoutError:
            await events.put(
                {
                    "type": "error",
                    "error": "error.hidden_watermark_remote_timeout",
                    "retryable": True,
                }
            )
        except WatermarkError as exc:
            await events.put({"type": "error", "error": str(exc), "retryable": False})
        except asyncio.CancelledError:
            raise
        except Exception:
            await events.put({"type": "error", "error": fallback_error, "retryable": True})
        else:
            await events.put({"type": "result", **response_builder(result)})

    task = asyncio.create_task(run())

    async def stream():
        nonlocal latest
        yield _ndjson({"type": "progress", **latest, "elapsed_ms": 0})
        try:
            while True:
                try:
                    event = await asyncio.wait_for(events.get(), timeout=2)
                except TimeoutError:
                    event = {"type": "heartbeat", **latest}
                if event.get("type") == "progress":
                    latest = {
                        "stage": event.get("stage", latest["stage"]),
                        "percent": event.get("percent", latest["percent"]),
                        "completed": event.get("completed", latest["completed"]),
                        "total": event.get("total", latest["total"]),
                    }
                event["elapsed_ms"] = round((time.monotonic() - started_at) * 1000)
                yield _ndjson(event)
                if event.get("type") in {"result", "error"}:
                    break
        finally:
            if not task.done():
                task.cancel()

    return StreamingResponse(
        stream(),
        media_type="application/x-ndjson",
        headers={
            "Cache-Control": "no-cache, no-store",
            "X-Accel-Buffering": "no",
        },
    )


def _extraction_response(
    groups: dict[int, list[bytes]],
    payload_sizes: list[int],
) -> dict[str, object]:
    first_size = next(size for size in payload_sizes if size in groups)
    payloads = groups[first_size]
    return {
        "payload_b64": encode_payload_b64(payloads[0]),
        "payload_candidates_b64": [encode_payload_b64(payload) for payload in payloads],
        "payload_bytes": len(payloads[0]),
        "payload_groups": [
            {
                "payload_bytes": size,
                "payload_candidates_b64": [encode_payload_b64(payload) for payload in groups[size]],
            }
            for size in payload_sizes
            if size in groups
        ],
    }


def _ndjson(event: dict[str, object]) -> bytes:
    return (json.dumps(event, separators=(",", ":")) + "\n").encode("utf-8")


def _payload_sizes(payload_bytes: int | None, payload_bytes_candidates: str | None) -> list[int]:
    values: list[int] = []
    if payload_bytes_candidates:
        try:
            values.extend(
                int(part.strip()) for part in payload_bytes_candidates.split(",") if part.strip()
            )
        except ValueError as exc:
            raise HTTPException(status_code=422, detail="error.payload_size_invalid") from exc
    if payload_bytes is not None:
        values.append(payload_bytes)
    values = list(dict.fromkeys(values))
    if not values or any(value <= 0 or value > 4096 for value in values):
        raise HTTPException(status_code=422, detail="error.payload_size_invalid")
    return values


async def _read_limited(file: UploadFile, max_bytes: int) -> bytes:
    chunks: list[bytes] = []
    total = 0
    while True:
        chunk = await file.read(1024 * 1024)
        if not chunk:
            break
        total += len(chunk)
        if total > max_bytes:
            raise HTTPException(status_code=413, detail="error.image_too_large")
        chunks.append(chunk)
    return b"".join(chunks)


def _watermark_process_entry(
    operation: str,
    args: tuple[object, ...],
    output: multiprocessing.Queue,
) -> None:
    def report(event: dict[str, object]) -> None:
        output.put({"type": "progress", "event": event})

    try:
        if operation == "embed":
            image_bytes, payload, options = args
            result = embed_payload(image_bytes, payload, options, progress=report)
        elif operation == "extract":
            image_bytes, payload_sizes, options, reference_image_bytes, search_num = args
            result = extract_payload_candidate_groups(
                image_bytes,
                payload_sizes,
                options,
                reference_image_bytes=reference_image_bytes,
                recovery_search_num=search_num,
                progress=report,
            )
        else:
            raise RuntimeError("unsupported watermark operation")
    except WatermarkError as exc:
        output.put({"type": "error", "error": str(exc), "watermark_error": True})
    except BaseException:
        output.put({"type": "error", "error": "error.watermark_process_failed"})
    else:
        output.put({"type": "result", "result": result})


async def _run_watermark[T](
    operation: str,
    args: tuple[object, ...],
    settings: Settings,
    operation_timeout_seconds: int,
    progress: Callable[[dict[str, object]], object] | None = None,
) -> T:
    async with _watermark_limiter(settings):
        context = multiprocessing.get_context("spawn")
        output = context.Queue()
        process = context.Process(
            target=_watermark_process_entry,
            args=(operation, args, output),
            daemon=True,
        )
        process.start()
        deadline = time.monotonic() + operation_timeout_seconds
        dead_checks = 0
        try:
            while True:
                if time.monotonic() >= deadline:
                    raise TimeoutError
                try:
                    event = output.get_nowait()
                except Empty:
                    if not process.is_alive():
                        dead_checks += 1
                        if dead_checks >= 5:
                            raise RuntimeError("watermark worker exited without a result") from None
                    await asyncio.sleep(0.05)
                    continue
                dead_checks = 0
                if event["type"] == "progress":
                    if progress is not None:
                        await progress(event["event"])
                    continue
                if event["type"] == "error":
                    if event.get("watermark_error"):
                        raise WatermarkError(event["error"])
                    raise RuntimeError(event["error"])
                return event["result"]
        finally:
            if process.is_alive():
                process.terminate()
            await anyio.to_thread.run_sync(process.join, 2)
            if process.is_alive():
                process.kill()
                await anyio.to_thread.run_sync(process.join, 2)
            output.close()


def _watermark_limiter(settings: Settings) -> anyio.CapacityLimiter:
    global _limiter, _limiter_workers
    with _limiter_lock:
        if _limiter is None or _limiter_workers != settings.max_workers:
            _limiter = anyio.CapacityLimiter(settings.max_workers)
            _limiter_workers = settings.max_workers
        return _limiter
