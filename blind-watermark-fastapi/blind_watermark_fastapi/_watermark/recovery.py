import math

import cv2
import numpy as np

from .models import (
    _FAST_RESIZE_ASPECT_DELTA,
    _FAST_RESIZE_MIN_AREA_RATIO,
    _MAX_DIRECT_CANDIDATES,
    _MAX_FAST_RESIZE_CANDIDATES,
    _MAX_RECOVERED_CANDIDATES,
    _RECOVERY_GOOD_SCORE,
    _RECOVERY_MATCHES_TO_EXPAND,
    _RECOVERY_PAIR_LIMIT,
    ProgressCallback,
    WatermarkError,
)
from .runtime import estimate_crop_parameters, recover_crop


def _recover_screenshot_candidates(
    screenshot: np.ndarray,
    reference: np.ndarray,
    search_num: int,
    progress: ProgressCallback | None = None,
) -> list[np.ndarray]:
    fast_candidates = _reference_resized_candidates(
        screenshot=screenshot,
        reference=reference,
        limit=_MAX_FAST_RESIZE_CANDIDATES,
    )
    if fast_candidates:
        _report_progress(progress, "recovering", 55, completed=1, total=1)
        return fast_candidates

    matches: list[tuple[float, tuple[int, int, int, int], tuple[int, ...], np.ndarray]] = []
    _report_progress(progress, "recovering", 20, completed=0, total=1)
    quick_match = _estimate_recovery_match(
        template=_bgr_uint8_image(screenshot),
        original=_bgr_uint8_image(reference),
        search_num=search_num,
    )
    _report_progress(progress, "recovering", 32, completed=1, total=1)
    if quick_match is not None and quick_match[0] >= _RECOVERY_GOOD_SCORE:
        _report_progress(progress, "recovering", 55, completed=1, total=1)
        return _expand_recovery_matches([quick_match])

    screenshot_candidates = _candidate_image_variants(screenshot, limit=_MAX_DIRECT_CANDIDATES)
    reference_candidates = _candidate_image_variants(
        reference,
        include_rotations=False,
        limit=4,
    )
    pairs = _candidate_match_pairs(
        screenshot_candidates, reference_candidates, _RECOVERY_PAIR_LIMIT
    )
    total_pairs = max(1, len(pairs))
    variant_search_num = max(20, min(search_num, 60))
    for pair_index, (template, original) in enumerate(pairs, start=1):
        match = _estimate_recovery_match(
            template=template,
            original=original,
            search_num=variant_search_num,
        )
        _report_progress(
            progress,
            "recovering",
            32 + round(23 * pair_index / total_pairs),
            completed=pair_index,
            total=total_pairs,
        )
        if match is not None:
            matches.append(match)
            if match[0] >= _RECOVERY_GOOD_SCORE:
                return _expand_recovery_matches([match])
    if not matches:
        return []

    return _expand_recovery_matches(matches)


def _report_progress(
    callback: ProgressCallback | None,
    stage: str,
    percent: int,
    *,
    completed: int = 0,
    total: int = 0,
) -> None:
    if callback is None:
        return
    callback(
        {
            "stage": stage,
            "percent": max(0, min(100, int(percent))),
            "completed": max(0, int(completed)),
            "total": max(0, int(total)),
        }
    )


def _monotonic_progress(callback: ProgressCallback | None) -> ProgressCallback | None:
    if callback is None:
        return None
    max_percent = 0

    def wrapped(event: dict[str, object]) -> None:
        nonlocal max_percent
        next_event = dict(event)
        percent = int(next_event.get("percent", 0) or 0)
        percent = max(max_percent, percent)
        max_percent = percent
        next_event["percent"] = percent
        callback(next_event)

    return wrapped


def _reference_resized_candidates(
    *,
    screenshot: np.ndarray,
    reference: np.ndarray,
    limit: int,
) -> list[np.ndarray]:
    screenshot_candidates = _candidate_image_variants(screenshot, limit=_MAX_DIRECT_CANDIDATES)
    reference_base = _bgr_uint8_image(reference)
    pairs = _candidate_match_pairs(
        screenshot_candidates, [reference_base], len(screenshot_candidates)
    )
    candidates: list[np.ndarray] = []
    reference_area = float(reference_base.shape[0] * reference_base.shape[1])
    for template, original in pairs:
        if _aspect_delta(template, original) > _FAST_RESIZE_ASPECT_DELTA:
            continue
        area_ratio = (template.shape[0] * template.shape[1]) / reference_area
        if area_ratio < _FAST_RESIZE_MIN_AREA_RATIO:
            continue
        target_height, target_width = original.shape[:2]
        resized = cv2.resize(
            template,
            (target_width, target_height),
            interpolation=cv2.INTER_CUBIC if template.shape[0] < target_height else cv2.INTER_AREA,
        )
        _append_unique_image(candidates, resized, limit)
        if len(candidates) >= limit:
            break
    return candidates


def _candidate_match_pairs(
    templates: list[np.ndarray],
    originals: list[np.ndarray],
    limit: int,
) -> list[tuple[np.ndarray, np.ndarray]]:
    ranked = [
        (_aspect_delta(template, original), index, template, original)
        for index, (template, original) in enumerate(
            (template, original) for template in templates for original in originals
        )
    ]
    ranked.sort(key=lambda item: (item[0], item[1]))
    return [(template, original) for _delta, _index, template, original in ranked[:limit]]


def _aspect_delta(first: np.ndarray, second: np.ndarray) -> float:
    first_height, first_width = first.shape[:2]
    second_height, second_width = second.shape[:2]
    if first_height <= 0 or first_width <= 0 or second_height <= 0 or second_width <= 0:
        return math.inf
    return abs(math.log((first_width / first_height) / (second_width / second_height)))


def _estimate_recovery_match(
    *,
    template: np.ndarray,
    original: np.ndarray,
    search_num: int,
) -> tuple[float, tuple[int, int, int, int], tuple[int, ...], np.ndarray] | None:
    try:
        loc, image_shape, score, _scale = estimate_crop_parameters(
            ori_img=original,
            tem_img=template,
            scale=(0.25, 4.0),
            search_num=max(20, min(search_num, 40)),
        )
    except AssertionError, IndexError, TypeError, ValueError, cv2.error:
        return None
    return (
        float(score),
        tuple(int(value) for value in loc),
        tuple(int(value) for value in image_shape),
        template,
    )


def _expand_recovery_matches(
    matches: list[tuple[float, tuple[int, int, int, int], tuple[int, ...], np.ndarray]],
) -> list[np.ndarray]:
    candidates: list[np.ndarray] = []
    for _score, loc, image_shape, template in sorted(
        matches, key=lambda item: item[0], reverse=True
    )[:_RECOVERY_MATCHES_TO_EXPAND]:
        x1, y1, x2, y2 = loc
        for x2_delta in (0, 1, -1):
            for y2_delta in (0, 1, -1):
                candidate_loc = (x1, y1, x2 + x2_delta, y2 + y2_delta)
                try:
                    candidates.append(
                        _uint8_image(
                            recover_crop(
                                tem_img=template,
                                loc=candidate_loc,
                                image_o_shape=image_shape,
                            )
                        )
                    )
                except AssertionError, IndexError, TypeError, ValueError, cv2.error:
                    continue
    return _unique_images(candidates, limit=_MAX_RECOVERED_CANDIDATES)


def _decode_image(data: bytes) -> np.ndarray:
    if not data:
        raise WatermarkError("error.image_required")
    raw = np.frombuffer(data, dtype=np.uint8)
    img = cv2.imdecode(raw, cv2.IMREAD_UNCHANGED)
    if img is None or img.ndim < 2:
        raise WatermarkError("error.image_invalid")
    return _bgr_uint8_image(img)


def _bgr_uint8_image(img: np.ndarray) -> np.ndarray:
    img = _uint8_image(img)
    if img.ndim == 2:
        return cv2.cvtColor(img, cv2.COLOR_GRAY2BGR)
    if img.ndim != 3:
        raise WatermarkError("error.image_invalid")
    if img.shape[2] == 1:
        return cv2.cvtColor(img, cv2.COLOR_GRAY2BGR)
    if img.shape[2] == 4:
        alpha = img[:, :, 3:4].astype(np.float32) / 255.0
        color = img[:, :, :3].astype(np.float32)
        return np.ascontiguousarray(np.rint(color * alpha + 255.0 * (1.0 - alpha)).astype(np.uint8))
    return np.ascontiguousarray(img[:, :, :3])


def _uint8_image(img: np.ndarray) -> np.ndarray:
    arr = np.asarray(img)
    if arr.dtype == np.uint8:
        return np.ascontiguousarray(arr)
    arr = np.nan_to_num(arr)
    if np.issubdtype(arr.dtype, np.floating) and arr.size:
        minimum = float(np.min(arr))
        maximum = float(np.max(arr))
        if minimum >= 0.0 and maximum <= 1.0:
            arr = arr * 255.0
    return np.ascontiguousarray(np.clip(np.rint(arr), 0, 255).astype(np.uint8))


def _candidate_image_variants(
    img: np.ndarray,
    *,
    include_rotations: bool = True,
    limit: int,
) -> list[np.ndarray]:
    base = _bgr_uint8_image(img)
    rotations = [base]
    if include_rotations:
        rotations.extend(
            [
                cv2.rotate(base, cv2.ROTATE_90_CLOCKWISE),
                cv2.rotate(base, cv2.ROTATE_90_COUNTERCLOCKWISE),
                cv2.rotate(base, cv2.ROTATE_180),
            ]
        )

    candidates: list[np.ndarray] = []
    for variant in rotations:
        _append_unique_image(candidates, variant, limit)
        cropped = _content_crop_candidate(variant)
        if cropped is not None:
            _append_unique_image(candidates, cropped, limit)
        if len(candidates) >= limit:
            break
    return candidates


def _content_crop_candidate(img: np.ndarray) -> np.ndarray | None:
    img = _bgr_uint8_image(img)
    height, width = img.shape[:2]
    if height < 64 or width < 64:
        return None

    border_width = max(2, min(height, width) // 40)
    border = np.concatenate(
        [
            img[:border_width, :, :].reshape(-1, 3),
            img[-border_width:, :, :].reshape(-1, 3),
            img[:, :border_width, :].reshape(-1, 3),
            img[:, -border_width:, :].reshape(-1, 3),
        ],
        axis=0,
    ).astype(np.int16)
    background = np.median(border, axis=0)
    border_distance = np.linalg.norm(border - background, axis=1)
    if float(np.mean(border_distance < 18.0)) < 0.45:
        return None

    diff = np.linalg.norm(img.astype(np.int16) - background, axis=2)
    gray = cv2.cvtColor(img, cv2.COLOR_BGR2GRAY)
    gray_background = float(np.mean(background))
    mask = ((diff > 26.0) | (gray > gray_background + 35.0)).astype(np.uint8) * 255
    mask = cv2.morphologyEx(mask, cv2.MORPH_CLOSE, np.ones((5, 5), np.uint8), iterations=1) > 0

    column_run = _longest_run(np.where(np.mean(mask, axis=0) > 0.45)[0])
    if column_run is None:
        return None
    x1, x2 = column_run
    row_run = _longest_run(np.where(np.mean(mask[:, x1:x2], axis=1) > 0.75)[0])
    if row_run is None:
        return None
    y1, y2 = row_run
    padding = 2
    x1 = max(0, x1 - padding)
    y1 = max(0, y1 - padding)
    x2 = min(width, x2 + padding)
    y2 = min(height, y2 + padding)
    crop_area = (x2 - x1) * (y2 - y1)
    image_area = width * height
    if crop_area < image_area * 0.15 or crop_area > image_area * 0.98:
        return None
    return np.ascontiguousarray(img[y1:y2, x1:x2])


def _longest_run(values: np.ndarray) -> tuple[int, int] | None:
    if values.size == 0:
        return None
    start = previous = int(values[0])
    runs: list[tuple[int, int]] = []
    for raw_value in values[1:]:
        value = int(raw_value)
        if value == previous + 1:
            previous = value
            continue
        runs.append((start, previous + 1))
        start = previous = value
    runs.append((start, previous + 1))
    return max(runs, key=lambda run: run[1] - run[0])


def _unique_images(images: list[np.ndarray], *, limit: int) -> list[np.ndarray]:
    unique: list[np.ndarray] = []
    for image in images:
        _append_unique_image(unique, image, limit)
        if len(unique) >= limit:
            break
    return unique


def _append_unique_image(images: list[np.ndarray], image: np.ndarray, limit: int) -> None:
    if len(images) >= limit:
        return
    candidate = _bgr_uint8_image(image)
    key = _image_fingerprint(candidate)
    if any(_image_fingerprint(existing) == key for existing in images):
        return
    images.append(candidate)


def _image_fingerprint(img: np.ndarray) -> tuple[tuple[int, ...], bytes]:
    y_step = max(1, img.shape[0] // 32)
    x_step = max(1, img.shape[1] // 32)
    return img.shape, img[::y_step, ::x_step].tobytes()


def _bytes_to_bits(payload: bytes) -> np.ndarray:
    return np.unpackbits(np.frombuffer(payload, dtype=np.uint8), bitorder="big").astype(bool)


def _bits_to_bytes(bits: np.ndarray, payload_bytes: int) -> bytes:
    normalized = _normalized_bits(bits, payload_bytes) >= 0.5
    packed = np.packbits(normalized.astype(np.uint8), bitorder="big")
    return packed[:payload_bytes].tobytes()


def _normalized_bits(bits: np.ndarray, payload_bytes: int) -> np.ndarray:
    normalized = np.asarray(bits, dtype=np.float64).reshape(-1)[: payload_bytes * 8]
    if normalized.size != payload_bytes * 8:
        raise WatermarkError("error.payload_size_mismatch")
    if not np.all(np.isfinite(normalized)):
        raise WatermarkError("error.watermark_extract_unstable")
    return normalized


def _consensus_bit_candidates(bit_candidates: list[np.ndarray]) -> list[np.ndarray]:
    if len(bit_candidates) < 2:
        return []
    stacked = np.stack(bit_candidates)
    confidence = np.mean(np.abs(stacked - 0.5), axis=1) + 1e-9
    strongest = np.take_along_axis(
        stacked,
        np.argmax(np.abs(stacked - 0.5), axis=0, keepdims=True),
        axis=0,
    )[0]
    candidates = [
        np.mean(stacked, axis=0),
        np.median(stacked, axis=0),
        np.average(stacked, axis=0, weights=confidence),
        strongest,
    ]
    unique: list[np.ndarray] = []
    seen: set[bytes] = set()
    payload_bytes = bit_candidates[0].size // 8
    for candidate in candidates:
        packed = _bits_to_bytes(candidate, payload_bytes)
        if packed not in seen:
            seen.add(packed)
            unique.append(candidate)
    return unique
