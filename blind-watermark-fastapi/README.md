# Yuem Blind Watermark FastAPI

Internal FastAPI wrapper around the latest released [`guofei9987/blind_watermark`](https://github.com/guofei9987/blind_watermark) (`0.4.4`). The Go backend keeps generating and validating the signed `YWM1` / `YPW1` payloads; this service only embeds or extracts opaque payload bytes.

## Run

Container/Linux startup:

```bash
cd blind-watermark-fastapi
export BLIND_WATERMARK_API_KEY="change_me_internal_only"
./run.sh
```

`run.sh` installs `uv` when missing, syncs production dependencies with `uv sync --no-dev`,
and starts:

```bash
uv run uvicorn blind_watermark_fastapi.main:app --host 0.0.0.0 --port 8090
```

The project excludes GUI `opencv-python` and installs `opencv-python-headless` to avoid the
common container error `libGL.so.1: cannot open shared object file`. On Debian/Ubuntu images the
script also installs `libgl1` and `libglib2.0-0` as a runtime fallback when `apt-get` is available.

The service keeps the upstream `block_shape=(4, 4)` and supports the protected-image strength
profiles used by the Go backend: `18/8`, `24/12`, `30/15`, and the upstream-compatible `36/20`.
Embedding and extraction must use the same `password_wm`, `password_img`, `d1`, and `d2`.

```powershell
cd blind-watermark-fastapi
uv sync
$env:BLIND_WATERMARK_API_KEY="change_me_internal_only"
uv run python -m blind_watermark_fastapi
```

Equivalent explicit uvicorn command:

```powershell
uv run uvicorn blind_watermark_fastapi.main:app --host 0.0.0.0 --port 8090
```

## Endpoints

- `GET /healthz`
- `POST /v1/watermark/embed`
  - multipart fields: `file`, `payload_b64`, optional `password_wm`, `password_img`, `d1`, `d2`, `operation_timeout_seconds`
  - returns PNG bytes
- `POST /v1/watermark/embed-stream`
  - same multipart fields as `embed`
  - returns NDJSON progress, heartbeat, and the final base64 image
- `POST /v1/watermark/extract`
  - multipart fields: `file`, `payload_bytes` or `payload_bytes_candidates`, optional
    `reference_file`, `password_wm`, `password_img`, `d1`, `d2`, `operation_timeout_seconds`
  - returns `{ "payload_b64": "..." }`
- `POST /v1/watermark/extract-stream`
  - same extraction fields and NDJSON progress/result events

For a cropped or resized screenshot, `reference_file` must be the complete watermarked image.
The service then runs the upstream `estimate_crop_parameters()` and `recover_crop()` flow before
extraction. This reference is required because the upstream algorithm cannot infer an arbitrary
crop and scale from the screenshot alone.

Every CPU-bound operation runs in an isolated child process behind a process-wide capacity
limiter. Timeout, client disconnect, or cancellation terminates and joins that child so abandoned
watermark calculations cannot continue consuming workers and trigger later gateway `554` errors.
