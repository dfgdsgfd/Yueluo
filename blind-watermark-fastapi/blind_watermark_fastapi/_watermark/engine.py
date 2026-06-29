import warnings

import cv2
import numpy as np
from imwatermark import WatermarkDecoder, WatermarkEncoder

from .models import (
    _MAX_DIRECT_CANDIDATES,
    _MAX_RECOVERED_CANDIDATES,
    EmbeddedImage,
    ProgressCallback,
    WatermarkError,
    WatermarkOptions,
)
from .recovery import (
    _append_unique_image,
    _bgr_uint8_image,
    _bits_to_bytes,
    _bytes_to_bits,
    _candidate_image_variants,
    _consensus_bit_candidates,
    _decode_image,
    _monotonic_progress,
    _normalized_bits,
    _recover_screenshot_candidates,
    _report_progress,
    _uint8_image,
    _unique_images,
)
from .runtime import WaterMark


def embed_payload(
    image_bytes: bytes,
    payload: bytes,
    options: WatermarkOptions,
    progress: ProgressCallback | None = None,
) -> EmbeddedImage:
    progress = _monotonic_progress(progress)
    if not payload:
        raise WatermarkError("error.payload_required")
    _report_progress(progress, "decoding", 5)
    img = _decode_image(image_bytes)
    if options.engine == "dwt_dct_svd":
        return _embed_dwt_dct_svd(img, payload, options, progress)
    try:
        return _embed_blind_watermark(img, payload, options, progress)
    except WatermarkError:
        if options.engine != "auto":
            raise
    return _embed_dwt_dct_svd(img, payload, options, progress)


def _embed_blind_watermark(
    img: np.ndarray,
    payload: bytes,
    options: WatermarkOptions,
    progress: ProgressCallback | None,
) -> EmbeddedImage:
    _report_progress(progress, "embedding", 15)
    wm = _new_watermark(options)
    bits = _bytes_to_bits(payload)
    try:
        wm.read_img(img=img)
        wm.read_wm(bits, mode="bit")
        marked = wm.embed()
    except (AssertionError, IndexError, ValueError) as exc:
        raise WatermarkError(str(exc) or "error.watermark_embed_failed") from exc

    _report_progress(progress, "embedding", 88)
    marked = _uint8_image(marked)
    ok, encoded = cv2.imencode(".png", marked)
    if not ok:
        raise WatermarkError("error.image_encode_failed")
    _report_progress(progress, "encoding", 98)
    return EmbeddedImage(data=encoded.tobytes(), media_type="image/png")


def extract_payload(
    image_bytes: bytes,
    payload_bytes: int,
    options: WatermarkOptions,
    reference_image_bytes: bytes | None = None,
    recovery_search_num: int = 200,
) -> bytes:

    return extract_payload_candidates(
        image_bytes,
        payload_bytes,
        options,
        reference_image_bytes,
        recovery_search_num,
    )[0]


def extract_payload_candidates(
    image_bytes: bytes,
    payload_bytes: int,
    options: WatermarkOptions,
    reference_image_bytes: bytes | None = None,
    recovery_search_num: int = 200,
) -> list[bytes]:
    groups = extract_payload_candidate_groups(
        image_bytes,
        [payload_bytes],
        options,
        reference_image_bytes,
        recovery_search_num,
    )
    return groups[payload_bytes]


def extract_payload_candidate_groups(
    image_bytes: bytes,
    payload_bytes_candidates: list[int],
    options: WatermarkOptions,
    reference_image_bytes: bytes | None = None,
    recovery_search_num: int = 200,
    progress: ProgressCallback | None = None,
) -> dict[int, list[bytes]]:
    progress = _monotonic_progress(progress)
    payload_sizes = list(dict.fromkeys(payload_bytes_candidates))
    if not payload_sizes or any(payload_bytes <= 0 for payload_bytes in payload_sizes):
        raise WatermarkError("error.payload_size_invalid")
    _report_progress(progress, "decoding", 5)
    img = _decode_image(image_bytes)
    dwt_groups: dict[int, list[bytes]] = {}
    if options.engine == "dwt_dct_svd":
        return _extract_dwt_dct_svd_groups(img, payload_sizes, options, progress)
    direct_candidates = _candidate_image_variants(img, limit=_MAX_DIRECT_CANDIDATES)
    _report_progress(progress, "decoding", 12)
    candidates = direct_candidates
    if reference_image_bytes:
        _report_progress(progress, "recovering", 16)
        recovered_candidates = _recover_screenshot_candidates(
            screenshot=img,
            reference=_decode_image(reference_image_bytes),
            search_num=recovery_search_num,
            progress=progress,
        )
        candidates = _unique_images(
            [*recovered_candidates, *direct_candidates],
            limit=_MAX_RECOVERED_CANDIDATES + _MAX_DIRECT_CANDIDATES,
        )
    extraction_start = 60 if reference_image_bytes else 20
    extraction_range = 34 if reference_image_bytes else 74
    total_units = max(1, len(payload_sizes) * len(candidates))
    completed_units = 0
    _report_progress(
        progress,
        "extracting",
        extraction_start,
        completed=completed_units,
        total=total_units,
    )
    groups: dict[int, list[bytes]] = {}
    last_error: Exception | None = None
    for payload_bytes in payload_sizes:
        payloads: list[bytes] = []
        bit_candidates: list[np.ndarray] = []
        for candidate in candidates:
            wm = _new_watermark(options)
            try:
                bits = _extract_blind_watermark_bits(wm, candidate, payload_bytes)
                normalized_bits = _normalized_bits(bits, payload_bytes)
                payload = _bits_to_bytes(normalized_bits, payload_bytes)
            except (AssertionError, IndexError, RuntimeWarning, ValueError) as exc:
                last_error = exc
            else:
                bit_candidates.append(normalized_bits)
                if payload not in payloads:
                    payloads.append(payload)
            completed_units += 1
            _report_progress(
                progress,
                "extracting",
                extraction_start + round(extraction_range * completed_units / total_units),
                completed=completed_units,
                total=total_units,
            )
        for consensus_bits in _consensus_bit_candidates(bit_candidates):
            consensus_payload = _bits_to_bytes(consensus_bits, payload_bytes)
            if consensus_payload not in payloads:
                payloads.append(consensus_payload)
        if payloads:
            groups[payload_bytes] = payloads
    _report_progress(progress, "verifying", 96, completed=completed_units, total=total_units)
    if dwt_groups:
        groups = _merge_payload_groups(dwt_groups, groups)
    if options.engine == "auto":
        try:
            dwt_groups = _extract_dwt_dct_svd_groups(img, payload_sizes, options, progress)
        except WatermarkError as exc:
            last_error = exc
        else:
            groups = _merge_payload_groups(groups, dwt_groups)
    if not groups:
        raise WatermarkError(str(last_error) if last_error else "error.watermark_extract_failed")
    return groups


def _extract_blind_watermark_bits(
    wm: WaterMark,
    candidate: np.ndarray,
    payload_bytes: int,
) -> np.ndarray:
    try:
        with warnings.catch_warnings():
            warnings.filterwarnings("error", category=RuntimeWarning)
            return wm.extract(embed_img=candidate, wm_shape=(payload_bytes * 8,), mode="bit")
    except RuntimeWarning as exc:
        raise WatermarkError("error.watermark_extract_unstable") from exc


def _new_watermark(options: WatermarkOptions) -> WaterMark:
    if options.d1 <= 0 or options.d2 < 0 or options.d2 > options.d1:
        raise WatermarkError("error.watermark_strength_invalid")
    watermark = WaterMark(
        password_wm=options.password_wm,
        password_img=options.password_img,
        processes=1,
    )
    watermark.bwm_core.d1 = options.d1
    watermark.bwm_core.d2 = options.d2
    return watermark


def _embed_dwt_dct_svd(
    img: np.ndarray,
    payload: bytes,
    options: WatermarkOptions,
    progress: ProgressCallback | None,
) -> EmbeddedImage:
    _report_progress(progress, "embedding", 15)
    encoder = WatermarkEncoder()
    encoder.set_watermark("bits", _repeat_bits(_bytes_to_bits(payload), options.dwt_dct_svd_repeat))
    try:
        marked = encoder.encode(
            _bgr_uint8_image(img),
            "dwtDctSvd",
            scales=[0, options.dwt_dct_svd_scale, 0],
        )
    except (AssertionError, IndexError, ValueError, cv2.error) as exc:
        raise WatermarkError(str(exc) or "error.watermark_embed_failed") from exc
    _report_progress(progress, "encoding", 98)
    ok, encoded = cv2.imencode(".png", _bgr_uint8_image(marked))
    if not ok:
        raise WatermarkError("error.image_encode_failed")
    return EmbeddedImage(data=encoded.tobytes(), media_type="image/png")


def _extract_dwt_dct_svd_groups(
    img: np.ndarray,
    payload_sizes: list[int],
    options: WatermarkOptions,
    progress: ProgressCallback | None,
) -> dict[int, list[bytes]]:
    candidates = _dwt_recover_dimension_candidates(img, options.recover_dimensions)
    total_units = max(1, len(payload_sizes) * len(candidates))
    completed_units = 0
    groups: dict[int, list[bytes]] = {}
    for payload_bytes in payload_sizes:
        payloads: list[bytes] = []
        for candidate in candidates:
            try:
                decoder = WatermarkDecoder(
                    "bits",
                    payload_bytes * 8 * options.dwt_dct_svd_repeat,
                )
                bits = decoder.decode(
                    _bgr_uint8_image(candidate),
                    "dwtDctSvd",
                    scales=[0, options.dwt_dct_svd_scale, 0],
                )
                payload = _bits_to_bytes(
                    _majority_bits(
                        [1 if bool(bit) else 0 for bit in bits],
                        options.dwt_dct_svd_repeat,
                    ),
                    payload_bytes,
                )
            except AssertionError, IndexError, ValueError, cv2.error:
                payload = b""
            if payload and payload not in payloads:
                payloads.append(payload)
            completed_units += 1
            _report_progress(
                progress,
                "extracting",
                20 + round(76 * completed_units / total_units),
                completed=completed_units,
                total=total_units,
            )
        if payloads:
            groups[payload_bytes] = payloads
    if not groups:
        raise WatermarkError("error.watermark_extract_failed")
    return groups


def _dwt_recover_dimension_candidates(
    img: np.ndarray,
    dimensions: tuple[tuple[int, int], ...],
) -> list[np.ndarray]:
    base = _bgr_uint8_image(img)
    candidates = [base]
    height, width = base.shape[:2]
    for target_width, target_height in dimensions:
        if target_width <= 0 or target_height <= 0:
            continue
        if target_width == width and target_height == height:
            continue
        resized = cv2.resize(base, (target_width, target_height), interpolation=cv2.INTER_LANCZOS4)
        _append_unique_image(candidates, resized, 12)
    return candidates


def _repeat_bits(bits: list[int], repeat: int) -> list[int]:
    if repeat <= 1:
        return bits
    repeated: list[int] = []
    for bit in bits:
        repeated.extend([bit] * repeat)
    return repeated


def _majority_bits(bits: list[int], repeat: int) -> list[int]:
    if repeat <= 1:
        return bits
    restored: list[int] = []
    for index in range(0, len(bits), repeat):
        chunk = bits[index : index + repeat]
        if len(chunk) < repeat:
            break
        restored.append(1 if sum(chunk) >= ((repeat // 2) + 1) else 0)
    return restored


def _merge_payload_groups(
    first: dict[int, list[bytes]],
    second: dict[int, list[bytes]],
) -> dict[int, list[bytes]]:
    merged: dict[int, list[bytes]] = {}
    for group in (first, second):
        for payload_bytes, payloads in group.items():
            merged.setdefault(payload_bytes, [])
            for payload in payloads:
                if payload not in merged[payload_bytes]:
                    merged[payload_bytes].append(payload)
    return merged
