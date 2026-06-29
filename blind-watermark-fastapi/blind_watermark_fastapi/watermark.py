"""Compatibility facade for the watermark implementation."""

from ._watermark import engine as _engine
from ._watermark.models import (
    EmbeddedImage,
    ProgressCallback,
    WatermarkError,
    WatermarkOptions,
    decode_payload_b64,
    encode_payload_b64,
    options_from_settings,
)
from ._watermark.recovery import _normalized_bits

_new_watermark = _engine._new_watermark


def _sync_compat_overrides() -> None:
    _engine._new_watermark = _new_watermark


def embed_payload(
    image_bytes: bytes,
    payload: bytes,
    options: WatermarkOptions,
    progress: ProgressCallback | None = None,
) -> EmbeddedImage:
    _sync_compat_overrides()
    return _engine.embed_payload(image_bytes, payload, options, progress)


def extract_payload(
    image_bytes: bytes,
    payload_bytes: int,
    options: WatermarkOptions,
    reference_image_bytes: bytes | None = None,
    recovery_search_num: int = 200,
) -> bytes:
    _sync_compat_overrides()
    return _engine.extract_payload(
        image_bytes, payload_bytes, options, reference_image_bytes, recovery_search_num
    )


def extract_payload_candidates(
    image_bytes: bytes,
    payload_bytes: int,
    options: WatermarkOptions,
    reference_image_bytes: bytes | None = None,
    recovery_search_num: int = 200,
) -> list[bytes]:
    _sync_compat_overrides()
    return _engine.extract_payload_candidates(
        image_bytes, payload_bytes, options, reference_image_bytes, recovery_search_num
    )


def extract_payload_candidate_groups(
    image_bytes: bytes,
    payload_bytes_candidates: list[int],
    options: WatermarkOptions,
    reference_image_bytes: bytes | None = None,
    recovery_search_num: int = 200,
    progress: ProgressCallback | None = None,
) -> dict[int, list[bytes]]:
    _sync_compat_overrides()
    return _engine.extract_payload_candidate_groups(
        image_bytes,
        payload_bytes_candidates,
        options,
        reference_image_bytes,
        recovery_search_num,
        progress,
    )

__all__ = [
    "EmbeddedImage",
    "ProgressCallback",
    "WatermarkError",
    "WatermarkOptions",
    "_normalized_bits",
    "decode_payload_b64",
    "embed_payload",
    "encode_payload_b64",
    "extract_payload",
    "extract_payload_candidate_groups",
    "extract_payload_candidates",
    "options_from_settings",
]
