import asyncio
import importlib
import time
from typing import Annotated

from fastapi import Depends, FastAPI, File, Form, Header, HTTPException, UploadFile
from fastapi.responses import Response, StreamingResponse

from ._blind_watermark_compat import install_blind_watermark_multiprocessing_guard
from .settings import Settings, get_settings
from .watermark import (
    WatermarkError,
    WatermarkOptions,
    decode_payload_b64,
    encode_payload_b64,
    options_from_settings,
)
from .watermark_service import (
    _extraction_response,
    _ndjson,
    _operation_stream,
    _payload_sizes,
    _read_limited,
    _run_watermark,
)

install_blind_watermark_multiprocessing_guard()
blind_watermark = importlib.import_module("blind_watermark")

app = FastAPI(
    title="Yuem Blind Watermark Service",
    version="0.1.0",
    docs_url="/docs",
    redoc_url=None,
)

SettingsDep = Annotated[Settings, Depends(get_settings)]
_DEFAULT_OPERATION_TIMEOUT_SECONDS = 45

def watermark_operation_timeout(
    operation_timeout_seconds: Annotated[int | None, Form(ge=10, le=300)] = None,
) -> int:
    return (
        operation_timeout_seconds
        if operation_timeout_seconds is not None
        else _DEFAULT_OPERATION_TIMEOUT_SECONDS
    )


async def require_internal_api_key(
    x_internal_api_key: Annotated[str | None, Header(alias="X-Internal-API-Key")] = None,
    settings: SettingsDep = None,
) -> None:
    assert settings is not None
    if settings.api_key and x_internal_api_key != settings.api_key:
        raise HTTPException(status_code=401, detail="error.invalid_api_key")


@app.get("/healthz")
async def healthz(settings: SettingsDep) -> dict[str, object]:
    return {
        "ok": True,
        "service": "yuem-blind-watermark-fastapi",
        "maxImageBytes": settings.max_image_bytes,
        "maxWorkers": settings.max_workers,
        "blindWatermarkVersion": blind_watermark.__version__,
    }


def watermark_options(
    settings: SettingsDep,
    password_wm: Annotated[int | None, Form(ge=0, le=2_147_483_647)] = None,
    password_img: Annotated[int | None, Form(ge=0, le=2_147_483_647)] = None,
    d1: Annotated[int | None, Form(ge=1, le=64)] = None,
    d2: Annotated[int | None, Form(ge=0, le=64)] = None,
    engine: Annotated[str | None, Form()] = None,
    dwt_dct_svd_repeat: Annotated[int | None, Form(ge=1, le=21)] = None,
    dwt_dct_svd_scale: Annotated[int | None, Form(ge=1, le=128)] = None,
    watermark_width: Annotated[int | None, Form(gt=0, le=16384)] = None,
    watermark_height: Annotated[int | None, Form(gt=0, le=16384)] = None,
    recover_dimensions: Annotated[str | None, Form()] = None,
) -> WatermarkOptions:
    base = options_from_settings(settings)
    selected_engine = (engine or "blind_watermark").strip()
    if selected_engine not in {"auto", "blind_watermark", "dwt_dct_svd"}:
        raise HTTPException(status_code=422, detail="error.watermark_engine_invalid")
    dimensions = _recover_dimensions(recover_dimensions)
    if watermark_width is not None and watermark_height is not None:
        dimensions = ((watermark_width, watermark_height), *dimensions)
    return WatermarkOptions(
        password_wm=password_wm if password_wm is not None else base.password_wm,
        password_img=password_img if password_img is not None else base.password_img,
        d1=d1 if d1 is not None else base.d1,
        d2=d2 if d2 is not None else base.d2,
        engine=selected_engine,
        dwt_dct_svd_repeat=dwt_dct_svd_repeat if dwt_dct_svd_repeat is not None else 9,
        dwt_dct_svd_scale=dwt_dct_svd_scale if dwt_dct_svd_scale is not None else 64,
        recover_dimensions=dimensions,
    )


def _recover_dimensions(value: str | None) -> tuple[tuple[int, int], ...]:
    if not value:
        return ()
    dimensions: list[tuple[int, int]] = []
    for part in value.split(","):
        text = part.strip().lower()
        if not text:
            continue
        pieces = text.split("x", 1)
        if len(pieces) != 2:
            raise HTTPException(
                status_code=422, detail="error.watermark_recover_dimensions_invalid"
            )
        try:
            width, height = int(pieces[0]), int(pieces[1])
        except ValueError as exc:
            raise HTTPException(
                status_code=422,
                detail="error.watermark_recover_dimensions_invalid",
            ) from exc
        if width <= 0 or height <= 0 or width > 16384 or height > 16384:
            raise HTTPException(
                status_code=422, detail="error.watermark_recover_dimensions_invalid"
            )
        dimension = (width, height)
        if dimension not in dimensions:
            dimensions.append(dimension)
    return tuple(dimensions)


@app.post("/v1/watermark/embed", dependencies=[Depends(require_internal_api_key)])
async def embed(
    file: Annotated[UploadFile, File()],
    payload_b64: Annotated[str, Form()],
    settings: SettingsDep,
    operation_timeout_seconds: Annotated[int, Depends(watermark_operation_timeout)],
    options: Annotated[WatermarkOptions, Depends(watermark_options)] = None,
) -> Response:
    assert options is not None
    image_bytes = await _read_limited(file, settings.max_image_bytes)
    payload = decode_payload_b64(payload_b64)
    try:
        result = await _run_watermark(
            "embed", (image_bytes, payload, options), settings, operation_timeout_seconds
        )
    except TimeoutError as exc:
        raise HTTPException(
            status_code=504, detail="error.hidden_watermark_remote_timeout"
        ) from exc
    except WatermarkError as exc:
        raise HTTPException(status_code=422, detail=str(exc)) from exc
    return Response(
        content=result.data,
        media_type=result.media_type,
        headers={"X-Watermark-Payload-Bytes": str(len(payload))},
    )


@app.post("/v1/watermark/embed-stream", dependencies=[Depends(require_internal_api_key)])
async def embed_stream(
    file: Annotated[UploadFile, File()],
    payload_b64: Annotated[str, Form()],
    settings: SettingsDep,
    operation_timeout_seconds: Annotated[int, Depends(watermark_operation_timeout)],
    options: Annotated[WatermarkOptions, Depends(watermark_options)] = None,
) -> StreamingResponse:
    assert options is not None
    image_bytes = await _read_limited(file, settings.max_image_bytes)
    payload = decode_payload_b64(payload_b64)
    return _operation_stream(
        "embed",
        (image_bytes, payload, options),
        settings,
        operation_timeout_seconds,
        lambda result: {
            "image_b64": encode_payload_b64(result.data),
            "media_type": result.media_type,
            "payload_bytes": len(payload),
        },
        "error.watermark_embed_failed",
    )


@app.post("/v1/watermark/extract", dependencies=[Depends(require_internal_api_key)])
async def extract(
    file: Annotated[UploadFile, File()],
    settings: SettingsDep,
    operation_timeout_seconds: Annotated[int, Depends(watermark_operation_timeout)],
    payload_bytes: Annotated[int | None, Form(gt=0, le=4096)] = None,
    payload_bytes_candidates: Annotated[str | None, Form()] = None,
    reference_file: Annotated[UploadFile | None, File()] = None,
    options: Annotated[WatermarkOptions, Depends(watermark_options)] = None,
) -> dict[str, object]:
    assert options is not None
    image_bytes = await _read_limited(file, settings.max_image_bytes)
    reference_image_bytes = (
        await _read_limited(reference_file, settings.max_image_bytes)
        if reference_file is not None
        else None
    )
    payload_sizes = _payload_sizes(payload_bytes, payload_bytes_candidates)
    try:
        groups = await _run_watermark(
            "extract",
            (
                image_bytes,
                payload_sizes,
                options,
                reference_image_bytes,
                settings.recovery_search_num,
            ),
            settings,
            operation_timeout_seconds,
        )
    except TimeoutError as exc:
        raise HTTPException(
            status_code=504, detail="error.hidden_watermark_remote_timeout"
        ) from exc
    except WatermarkError as exc:
        raise HTTPException(status_code=422, detail=str(exc)) from exc
    return _extraction_response(groups, payload_sizes)


@app.post("/v1/watermark/extract-stream", dependencies=[Depends(require_internal_api_key)])
async def extract_stream(
    file: Annotated[UploadFile, File()],
    settings: SettingsDep,
    operation_timeout_seconds: Annotated[int, Depends(watermark_operation_timeout)],
    payload_bytes: Annotated[int | None, Form(gt=0, le=4096)] = None,
    payload_bytes_candidates: Annotated[str | None, Form()] = None,
    reference_file: Annotated[UploadFile | None, File()] = None,
    options: Annotated[WatermarkOptions, Depends(watermark_options)] = None,
) -> StreamingResponse:
    assert options is not None
    image_bytes = await _read_limited(file, settings.max_image_bytes)
    reference_image_bytes = (
        await _read_limited(reference_file, settings.max_image_bytes)
        if reference_file is not None
        else None
    )
    payload_sizes = _payload_sizes(payload_bytes, payload_bytes_candidates)
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
            async with asyncio.timeout(operation_timeout_seconds):
                groups = await _run_watermark(
                    "extract",
                    (
                        image_bytes,
                        payload_sizes,
                        options,
                        reference_image_bytes,
                        settings.recovery_search_num,
                    ),
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
            await events.put(
                {
                    "type": "error",
                    "error": "error.watermark_extract_failed",
                    "retryable": True,
                }
            )
        else:
            await events.put({"type": "result", **_extraction_response(groups, payload_sizes)})

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
