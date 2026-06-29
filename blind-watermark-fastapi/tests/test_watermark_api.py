import base64
import io
import json
import multiprocessing
import os
import subprocess
import sys
import time
from pathlib import Path

import numpy as np
import pytest
from fastapi.testclient import TestClient
from PIL import Image, ImageDraw

from blind_watermark_fastapi import main as watermark_main
from blind_watermark_fastapi._blind_watermark_compat import (
    install_blind_watermark_multiprocessing_guard,
)
from blind_watermark_fastapi.main import app
from blind_watermark_fastapi.settings import get_settings
from blind_watermark_fastapi.watermark import (
    EmbeddedImage,
    WatermarkError,
    WatermarkOptions,
    _normalized_bits,
    extract_payload_candidate_groups,
)


def test_blind_watermark_guard_ignores_redundant_fork_context(monkeypatch):
    calls: list[tuple[str | None, bool]] = []

    def fake_set_start_method(method=None, force=False):
        calls.append((method, force))
        raise RuntimeError("context has already been set")

    monkeypatch.setattr(multiprocessing, "set_start_method", fake_set_start_method)
    monkeypatch.setattr(sys, "platform", "linux")

    install_blind_watermark_multiprocessing_guard()

    multiprocessing.set_start_method("fork")
    assert calls == [("fork", False)]

    with pytest.raises(RuntimeError):
        multiprocessing.set_start_method("spawn")


def test_main_import_survives_spawn_context_subprocess():
    project_root = Path(__file__).resolve().parents[1]
    env = os.environ.copy()
    env["PYTHONPATH"] = (
        str(project_root)
        if not env.get("PYTHONPATH")
        else str(project_root) + os.pathsep + env["PYTHONPATH"]
    )
    completed = subprocess.run(
        [
            sys.executable,
            "-c",
            (
                "import multiprocessing as mp; "
                "mp.set_start_method('spawn'); "
                "import blind_watermark_fastapi.main; "
                "print('ok')"
            ),
        ],
        cwd=project_root,
        env=env,
        capture_output=True,
        text=True,
        timeout=30,
        check=False,
    )

    assert completed.returncode == 0, completed.stderr
    assert completed.stdout.strip().endswith("ok")


def test_embed_extract_round_trip(monkeypatch):
    monkeypatch.setenv("BLIND_WATERMARK_API_KEY", "secret")
    get_settings.cache_clear()
    client = TestClient(app)
    payload = bytes(range(52))

    embed_response = client.post(
        "/v1/watermark/embed",
        headers={"X-Internal-API-Key": "secret"},
        files={"file": ("source.png", _image_bytes(), "image/png")},
        data={"payload_b64": base64.b64encode(payload).decode("ascii")},
    )
    assert embed_response.status_code == 200, embed_response.text
    assert embed_response.headers["content-type"] == "image/png"

    extract_response = client.post(
        "/v1/watermark/extract",
        headers={"X-Internal-API-Key": "secret"},
        files={"file": ("marked.png", embed_response.content, "image/png")},
        data={"payload_bytes_candidates": f"{len(payload)},52,68"},
    )
    assert extract_response.status_code == 200, extract_response.text
    payload_groups = {
        group["payload_bytes"]: {
            base64.b64decode(value) for value in group["payload_candidates_b64"]
        }
        for group in extract_response.json()["payload_groups"]
    }
    assert payload in payload_groups[len(payload)]


def test_official_password_options_control_remote_watermark(monkeypatch):
    monkeypatch.setenv("BLIND_WATERMARK_API_KEY", "")
    get_settings.cache_clear()
    client = TestClient(app)
    payload = bytes(range(52))
    tuned = {
        "payload_b64": base64.b64encode(payload).decode("ascii"),
        "password_wm": "77",
        "password_img": "88",
    }

    embed_response = client.post(
        "/v1/watermark/embed",
        files={"file": ("source.png", _image_bytes(), "image/png")},
        data=tuned,
    )
    assert embed_response.status_code == 200, embed_response.text

    extract_response = client.post(
        "/v1/watermark/extract",
        files={"file": ("marked.png", embed_response.content, "image/png")},
        data={key: value for key, value in tuned.items() if key != "payload_b64"}
        | {"payload_bytes": str(len(payload))},
    )
    assert extract_response.status_code == 200, extract_response.text
    assert base64.b64decode(extract_response.json()["payload_b64"]) == payload

    mismatched_response = client.post(
        "/v1/watermark/extract",
        files={"file": ("marked.png", embed_response.content, "image/png")},
        data={"payload_bytes": str(len(payload)), "password_wm": "78", "password_img": "88"},
    )
    assert mismatched_response.status_code == 200, mismatched_response.text
    assert base64.b64decode(mismatched_response.json()["payload_b64"]) != payload


def test_operation_timeout_form_overrides_settings(monkeypatch):
    monkeypatch.setenv("BLIND_WATERMARK_API_KEY", "")
    get_settings.cache_clear()
    seen: list[int] = []

    async def fake_run_watermark(
        operation, args, settings, operation_timeout_seconds, progress=None
    ):
        seen.append(operation_timeout_seconds)
        return EmbeddedImage(data=b"marked", media_type="image/png")

    monkeypatch.setattr(watermark_main, "_run_watermark", fake_run_watermark)
    client = TestClient(app)

    response = client.post(
        "/v1/watermark/embed",
        files={"file": ("source.png", _image_bytes(), "image/png")},
        data={
            "payload_b64": base64.b64encode(b"trace-id").decode("ascii"),
            "operation_timeout_seconds": "66",
        },
    )

    assert response.status_code == 200, response.text
    assert seen == [66]


def test_dwt_dct_svd_short_code_round_trip(monkeypatch):
    monkeypatch.setenv("BLIND_WATERMARK_API_KEY", "")
    get_settings.cache_clear()
    client = TestClient(app)
    payload = b"\xa1\xb2\xc3\xd4"

    embed_response = client.post(
        "/v1/watermark/embed",
        files={"file": ("source.png", _image_bytes(720, 720), "image/png")},
        data={
            "payload_b64": base64.b64encode(payload).decode("ascii"),
            "engine": "dwt_dct_svd",
            "dwt_dct_svd_repeat": "9",
            "dwt_dct_svd_scale": "64",
            "watermark_width": "720",
            "watermark_height": "720",
        },
    )
    assert embed_response.status_code == 200, embed_response.text

    extract_response = client.post(
        "/v1/watermark/extract",
        files={"file": ("marked.png", embed_response.content, "image/png")},
        data={
            "payload_bytes": str(len(payload)),
            "engine": "dwt_dct_svd",
            "dwt_dct_svd_repeat": "9",
            "dwt_dct_svd_scale": "64",
            "recover_dimensions": "720x720",
        },
    )
    assert extract_response.status_code == 200, extract_response.text
    assert base64.b64decode(extract_response.json()["payload_b64"]) == payload


def test_normalized_bits_rejects_nan_values():
    bits = np.zeros(16)
    bits[3] = np.nan

    with pytest.raises(WatermarkError, match="error.watermark_extract_unstable"):
        _normalized_bits(bits, 2)


def test_blind_watermark_runtime_warning_is_reported_as_unstable(monkeypatch):
    class WarningWatermark:
        bwm_core = type("Core", (), {"d1": 0, "d2": 0})()

        def extract(self, **_kwargs):
            import warnings

            warnings.warn("Mean of empty slice", RuntimeWarning, stacklevel=2)
            return np.zeros(32)

    monkeypatch.setattr(
        "blind_watermark_fastapi.watermark._new_watermark",
        lambda _options: WarningWatermark(),
    )

    with pytest.raises(WatermarkError, match="error.watermark_extract_unstable"):
        extract_payload_candidate_groups(
            _image_bytes(),
            [4],
            WatermarkOptions(password_wm=1, password_img=1, engine="blind_watermark"),
        )


def test_screenshot_recovery_matches_upstream_reference_flow(monkeypatch):
    monkeypatch.setenv("BLIND_WATERMARK_API_KEY", "")
    monkeypatch.setenv("BLIND_WATERMARK_RECOVERY_SEARCH_NUM", "200")
    get_settings.cache_clear()
    client = TestClient(app)
    payload = b"trace-id"

    embed_response = client.post(
        "/v1/watermark/embed",
        files={"file": ("source.png", _image_bytes(), "image/png")},
        data={"payload_b64": base64.b64encode(payload).decode("ascii")},
    )
    assert embed_response.status_code == 200, embed_response.text

    reference = Image.open(io.BytesIO(embed_response.content))
    width, height = reference.size
    crop = (
        int(width * 0.1),
        int(height * 0.1),
        int(width * 0.7),
        int(height * 0.7),
    )
    screenshot = reference.crop(crop)
    screenshot = screenshot.resize((round(screenshot.width * 0.7), round(screenshot.height * 0.7)))
    for image_format, quality in (("PNG", None), ("JPEG", 65), ("WEBP", 90)):
        screenshot_out = io.BytesIO()
        save_options = {} if quality is None else {"quality": quality}
        screenshot.save(screenshot_out, format=image_format, **save_options)
        extension = image_format.lower().replace("jpeg", "jpg")
        content_type = "image/jpeg" if image_format == "JPEG" else f"image/{image_format.lower()}"
        extract_response = client.post(
            "/v1/watermark/extract",
            files={
                "file": (
                    f"screenshot.{extension}",
                    screenshot_out.getvalue(),
                    content_type,
                ),
                "reference_file": ("reference.png", embed_response.content, "image/png"),
            },
            data={"payload_bytes_candidates": f"{len(payload)},52,68"},
        )

        assert extract_response.status_code == 200, (
            image_format,
            extract_response.text,
        )
        payload_groups = {
            group["payload_bytes"]: {
                base64.b64decode(value) for value in group["payload_candidates_b64"]
            }
            for group in extract_response.json()["payload_groups"]
        }
        assert payload in payload_groups[len(payload)], image_format


def test_jpeg_recompression_keeps_short_trace_token(monkeypatch):
    monkeypatch.setenv("BLIND_WATERMARK_API_KEY", "")
    get_settings.cache_clear()
    client = TestClient(app)
    payload = b"trace-id"

    embed_response = client.post(
        "/v1/watermark/embed",
        files={"file": ("source.png", _image_bytes(), "image/png")},
        data={"payload_b64": base64.b64encode(payload).decode("ascii")},
    )
    assert embed_response.status_code == 200, embed_response.text

    marked = Image.open(io.BytesIO(embed_response.content)).convert("RGB")
    transmitted = io.BytesIO()
    marked.save(transmitted, format="JPEG", quality=55)
    extract_response = client.post(
        "/v1/watermark/extract",
        files={"file": ("wechat.jpg", transmitted.getvalue(), "image/jpeg")},
        data={"payload_bytes_candidates": str(len(payload))},
    )
    assert extract_response.status_code == 200, extract_response.text
    payloads = {
        base64.b64decode(value)
        for group in extract_response.json()["payload_groups"]
        if group["payload_bytes"] == len(payload)
        for value in group["payload_candidates_b64"]
    }
    assert payload in payloads


def test_extract_stream_reports_server_progress_and_result(monkeypatch):
    monkeypatch.setenv("BLIND_WATERMARK_API_KEY", "")
    get_settings.cache_clear()
    client = TestClient(app)
    payload = b"trace-id"

    embed_response = client.post(
        "/v1/watermark/embed",
        files={"file": ("source.png", _image_bytes(), "image/png")},
        data={"payload_b64": base64.b64encode(payload).decode("ascii")},
    )
    assert embed_response.status_code == 200, embed_response.text

    with client.stream(
        "POST",
        "/v1/watermark/extract-stream",
        files={"file": ("marked.png", embed_response.content, "image/png")},
        data={"payload_bytes_candidates": str(len(payload))},
    ) as response:
        assert response.status_code == 200
        assert response.headers["content-type"].startswith("application/x-ndjson")
        events = [json.loads(line) for line in response.iter_lines() if line]

    progress_events = [event for event in events if event["type"] == "progress"]
    assert any(
        event.get("stage") == "extracting"
        and event.get("total", 0) > 0
        and event.get("completed", 0) > 0
        for event in progress_events
    )
    result = next(event for event in events if event["type"] == "result")
    payloads = {
        base64.b64decode(value)
        for group in result["payload_groups"]
        for value in group["payload_candidates_b64"]
    }
    assert payload in payloads


def test_embed_stream_reports_progress_and_reaps_worker(monkeypatch):
    monkeypatch.setenv("BLIND_WATERMARK_API_KEY", "")
    get_settings.cache_clear()
    client = TestClient(app)
    payload = b"trace-id"

    with client.stream(
        "POST",
        "/v1/watermark/embed-stream",
        files={"file": ("source.png", _image_bytes(), "image/png")},
        data={
            "payload_b64": base64.b64encode(payload).decode("ascii"),
            "d1": "18",
            "d2": "8",
        },
    ) as response:
        assert response.status_code == 200
        events = [json.loads(line) for line in response.iter_lines() if line]

    assert any(event.get("stage") == "embedding" for event in events)
    result = next(event for event in events if event["type"] == "result")
    assert base64.b64decode(result["image_b64"])
    for _ in range(20):
        if not multiprocessing.active_children():
            break
        time.sleep(0.05)
    assert not multiprocessing.active_children()


def test_api_key_is_enforced(monkeypatch):
    monkeypatch.setenv("BLIND_WATERMARK_API_KEY", "secret")
    get_settings.cache_clear()
    client = TestClient(app)

    response = client.post(
        "/v1/watermark/embed",
        files={"file": ("source.png", _image_bytes(), "image/png")},
        data={"payload_b64": base64.b64encode(b"payload").decode("ascii")},
    )

    assert response.status_code == 401


def test_image_size_limit(monkeypatch):
    monkeypatch.setenv("BLIND_WATERMARK_API_KEY", "")
    monkeypatch.setenv("BLIND_WATERMARK_MAX_IMAGE_BYTES", "8")
    get_settings.cache_clear()
    client = TestClient(app)

    response = client.post(
        "/v1/watermark/extract",
        files={"file": ("source.png", _image_bytes(), "image/png")},
        data={"payload_bytes": "1"},
    )

    assert response.status_code == 413


def _image_bytes(width: int = 738, height: int = 1108) -> bytes:
    y, x = np.mgrid[:height, :width]
    pixels = (
        np.stack(
            (
                35 + 170 * x / (width - 1) + 20 * np.sin(y / 37),
                25 + 165 * y / (height - 1) + 18 * np.cos(x / 43),
                55 + 90 * (x + y) / (width + height - 2) + 25 * np.sin((x - y) / 61),
            ),
            axis=2,
        )
        .clip(0, 255)
        .astype(np.uint8)
    )
    image = Image.fromarray(pixels)
    draw = ImageDraw.Draw(image)
    sx, sy = width / 738, height / 1108
    draw.ellipse((68 * sx, 126 * sy, 338 * sx, 548 * sy), fill=(210, 92, 58))
    draw.polygon(
        ((416 * sx, 78 * sy), (682 * sx, 386 * sy), (458 * sx, 702 * sy)),
        fill=(45, 132, 196),
    )
    draw.rounded_rectangle(
        (118 * sx, 802 * sy, 642 * sx, 1028 * sy),
        radius=max(1, round(42 * min(sx, sy))),
        fill=(236, 202, 84),
    )
    out = io.BytesIO()
    image.save(out, format="PNG")
    return out.getvalue()
