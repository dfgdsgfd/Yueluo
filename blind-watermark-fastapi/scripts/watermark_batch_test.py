from __future__ import annotations

import argparse
import csv
import hashlib
import io
import json
import random
import sys
import time
import urllib.request
from dataclasses import asdict, dataclass
from pathlib import Path
from typing import Any

from PIL import Image, ImageOps

PROJECT_ROOT = Path(__file__).resolve().parents[1]
if str(PROJECT_ROOT) not in sys.path:
    sys.path.insert(0, str(PROJECT_ROOT))

from blind_watermark_fastapi.watermark import (  # noqa: E402
    WatermarkError,
    WatermarkOptions,
    embed_payload,
    extract_payload_candidate_groups,
)

PICSUM_LIST_URL = "https://picsum.photos/v2/list?page={page}&limit={limit}"
PICSUM_IMAGE_URL = "https://picsum.photos/id/{id}/{width}/{height}"

RESOLUTION_PRESETS: dict[str, tuple[int, int]] = {
    "4k": (3840, 2160),
    "2k": (2560, 1440),
    "1080p": (1920, 1080),
    "720p": (1280, 720),
}

SOURCE_FORMATS = (
    ("jpeg_q92", "JPEG", {"quality": 92, "subsampling": 0, "optimize": True}),
    ("png", "PNG", {"optimize": True}),
    ("webp_lossless", "WEBP", {"lossless": True, "quality": 100, "method": 6}),
    ("webp_q90", "WEBP", {"quality": 90, "method": 6}),
)

ATTACKS = ("lossless_webp_self", "jpeg75", "webp85", "resize82_jpeg75_restore")


@dataclass(frozen=True)
class SourceSample:
    id: str
    author: str
    width: int
    height: int
    url: str
    download_url: str


@dataclass
class CaseResult:
    sample_id: str
    author: str
    source_url: str
    resolution: str
    orientation: str
    source_format: str
    attack: str
    payload_hex: str
    target_width: int
    target_height: int
    attacked_width: int
    attacked_height: int
    success: bool
    exact_4_byte_success: bool
    prefix_3_byte_success: bool
    prefix_2_byte_success: bool
    candidate_count: int
    candidate_hex: str
    embed_ms: int
    extract_ms: int
    error: str


def main() -> int:
    parser = argparse.ArgumentParser(
        description="Batch-test the DWT/DCT/SVD short-code watermark engine."
    )
    parser.add_argument("--sources", type=int, default=10)
    parser.add_argument("--picsum-page", type=int, default=2)
    parser.add_argument("--seed", type=int, default=20260620)
    parser.add_argument(
        "--output-dir",
        type=Path,
        default=Path(".tmp-watermark-batch") / time.strftime("%Y%m%d-%H%M%S"),
    )
    parser.add_argument("--password-wm", type=int, default=1)
    parser.add_argument("--password-img", type=int, default=1)
    parser.add_argument("--repeat", type=int, default=9)
    parser.add_argument("--scale", type=int, default=64)
    parser.add_argument(
        "--resolutions",
        default=",".join(RESOLUTION_PRESETS.keys()),
        help="Comma-separated resolution keys: 4k,2k,1080p,720p.",
    )
    parser.add_argument(
        "--attacks",
        default=",".join(ATTACKS),
        help="Comma-separated attacks: lossless_webp_self,jpeg75,webp85,resize82_jpeg75_restore.",
    )
    parser.add_argument("--download-width", type=int, default=4096)
    parser.add_argument("--download-height", type=int, default=4096)
    parser.add_argument("--save-failures", action="store_true")
    parser.add_argument(
        "--common-recover-candidates",
        action="store_true",
        help="Also try common 720/960/1280/2048-style dimensions for resized inputs.",
    )
    args = parser.parse_args()

    selected_resolutions = select_mapping_keys(args.resolutions, RESOLUTION_PRESETS, "resolution")
    selected_attacks = select_values(args.attacks, ATTACKS, "attack")

    random.seed(args.seed)
    output_dir = args.output_dir
    output_dir.mkdir(parents=True, exist_ok=True)
    failures_dir = output_dir / "failures"
    if args.save_failures:
        failures_dir.mkdir(parents=True, exist_ok=True)

    results_path = output_dir / "results.csv"
    summary_path = output_dir / "summary.json"
    sources_path = output_dir / "sources.json"

    samples = fetch_picsum_samples(
        page=args.picsum_page,
        limit=max(args.sources * 3, args.sources),
        count=args.sources,
    )
    sources_path.write_text(
        json.dumps([asdict(sample) for sample in samples], ensure_ascii=False, indent=2),
        encoding="utf-8",
    )

    print(
        f"batch_start sources={len(samples)} resolutions={','.join(selected_resolutions)} "
        f"attacks={','.join(selected_attacks)} out={output_dir}",
        flush=True,
    )

    options = WatermarkOptions(
        password_wm=args.password_wm,
        password_img=args.password_img,
        engine="dwt_dct_svd",
        dwt_dct_svd_repeat=args.repeat,
        dwt_dct_svd_scale=args.scale,
    )

    rows: list[CaseResult] = []
    fieldnames = list(CaseResult.__dataclass_fields__.keys())
    with results_path.open("w", newline="", encoding="utf-8") as csv_file:
        writer = csv.DictWriter(csv_file, fieldnames=fieldnames)
        writer.writeheader()

        for sample_index, sample in enumerate(samples):
            try:
                source = download_sample_image(
                    sample,
                    width=args.download_width,
                    height=args.download_height,
                )
            except Exception as exc:  # noqa: BLE001 - keep batch moving.
                print(f"sample_download_failed id={sample.id} error={exc}", flush=True)
                continue

            portrait = sample_index % 2 == 1
            for resolution_index, resolution in enumerate(selected_resolutions):
                base_size = RESOLUTION_PRESETS[resolution]
                target_size = base_size[::-1] if portrait else base_size
                orientation = "portrait" if portrait else "landscape"
                source_format = SOURCE_FORMATS[
                    (sample_index + resolution_index) % len(SOURCE_FORMATS)
                ]
                source_format_name = source_format[0]
                target = fit_cover(source, target_size)
                source_bytes = encode_image(target, source_format[1], source_format[2])
                payload = short_code(sample.id, resolution, source_format_name, target_size)
                started = time.perf_counter()
                embed_ms = 0
                protected_bytes = b""
                embed_error = ""
                try:
                    embedded = embed_payload(source_bytes, payload, options)
                    protected_image = Image.open(io.BytesIO(embedded.data)).convert("RGB")
                    protected_bytes = encode_image(
                        protected_image,
                        "WEBP",
                        {"lossless": True, "quality": 100, "method": 6},
                    )
                except Exception as exc:  # noqa: BLE001 - report and continue.
                    embed_error = f"{type(exc).__name__}: {exc}"
                finally:
                    embed_ms = round((time.perf_counter() - started) * 1000)

                if embed_error:
                    for attack in selected_attacks:
                        row = CaseResult(
                            sample_id=sample.id,
                            author=sample.author,
                            source_url=sample.url,
                            resolution=resolution,
                            orientation=orientation,
                            source_format=source_format_name,
                            attack=attack,
                            payload_hex=payload.hex(),
                            target_width=target_size[0],
                            target_height=target_size[1],
                            attacked_width=0,
                            attacked_height=0,
                            success=False,
                            exact_4_byte_success=False,
                            prefix_3_byte_success=False,
                            prefix_2_byte_success=False,
                            candidate_count=0,
                            candidate_hex="",
                            embed_ms=embed_ms,
                            extract_ms=0,
                            error=embed_error,
                        )
                        write_row(writer, csv_file, rows, row)
                    print(
                        f"embed_failed sample={sample.id} res={resolution} "
                        f"fmt={source_format_name} "
                        f"ms={embed_ms} error={embed_error}",
                        flush=True,
                    )
                    write_summary(summary_path, rows)
                    continue

                protected_image = Image.open(io.BytesIO(protected_bytes)).convert("RGB")
                for attack in selected_attacks:
                    attacked_bytes, attacked_size = make_attack(
                        protected_image,
                        protected_bytes,
                        attack,
                    )
                    extract_started = time.perf_counter()
                    error = ""
                    groups: dict[int, list[bytes]] = {}
                    try:
                        recover_dimensions = recover_dimension_candidates(
                            protected_image.size,
                            attacked_size,
                            include_common=(
                                args.common_recover_candidates
                                and attack == "resize82_jpeg75_restore"
                            ),
                        )
                        extract_options = WatermarkOptions(
                            password_wm=args.password_wm,
                            password_img=args.password_img,
                            engine="dwt_dct_svd",
                            dwt_dct_svd_repeat=args.repeat,
                            dwt_dct_svd_scale=args.scale,
                            recover_dimensions=recover_dimensions,
                        )
                        groups = extract_payload_candidate_groups(
                            attacked_bytes,
                            [4, 3, 2],
                            extract_options,
                        )
                    except (WatermarkError, Exception) as exc:  # noqa: BLE001
                        error = f"{type(exc).__name__}: {exc}"
                    extract_ms = round((time.perf_counter() - extract_started) * 1000)

                    candidates = flatten_candidates(groups)
                    exact_4 = payload in groups.get(4, [])
                    prefix_3 = payload[:3] in groups.get(3, [])
                    prefix_2 = payload[:2] in groups.get(2, [])
                    success = exact_4
                    row = CaseResult(
                        sample_id=sample.id,
                        author=sample.author,
                        source_url=sample.url,
                        resolution=resolution,
                        orientation=orientation,
                        source_format=source_format_name,
                        attack=attack,
                        payload_hex=payload.hex(),
                        target_width=protected_image.width,
                        target_height=protected_image.height,
                        attacked_width=attacked_size[0],
                        attacked_height=attacked_size[1],
                        success=success,
                        exact_4_byte_success=exact_4,
                        prefix_3_byte_success=prefix_3,
                        prefix_2_byte_success=prefix_2,
                        candidate_count=sum(len(values) for values in groups.values()),
                        candidate_hex=" ".join(candidates[:12]),
                        embed_ms=embed_ms,
                        extract_ms=extract_ms,
                        error=error,
                    )
                    write_row(writer, csv_file, rows, row)
                    if args.save_failures and not success:
                        failure_stem = (
                            f"{sample.id}_{resolution}_{orientation}_{source_format_name}_{attack}"
                        )
                        (failures_dir / f"{failure_stem}.json").write_text(
                            json.dumps(asdict(row), ensure_ascii=False, indent=2),
                            encoding="utf-8",
                        )
                        (failures_dir / f"{failure_stem}.webp").write_bytes(protected_bytes)
                        (failures_dir / f"{failure_stem}_attacked.bin").write_bytes(attacked_bytes)
                    print(
                        f"case sample={sample.id} res={resolution} {orientation} "
                        f"fmt={source_format_name} attack={attack} "
                        f"ok={int(success)} embed_ms={embed_ms} extract_ms={extract_ms} "
                        f"candidates={row.candidate_hex or '-'} error={error or '-'}",
                        flush=True,
                    )
                    write_summary(summary_path, rows)

    write_summary(summary_path, rows)
    print_summary(rows)
    print(f"batch_done results={results_path} summary={summary_path}", flush=True)
    return 0


def fetch_picsum_samples(*, page: int, limit: int, count: int) -> list[SourceSample]:
    with urllib.request.urlopen(
        PICSUM_LIST_URL.format(page=page, limit=limit),
        timeout=30,
    ) as response:
        payload = json.loads(response.read().decode("utf-8"))
    samples = [
        SourceSample(
            id=str(item["id"]),
            author=str(item.get("author", "")),
            width=int(item.get("width", 0)),
            height=int(item.get("height", 0)),
            url=str(item.get("url", "")),
            download_url=str(item.get("download_url", "")),
        )
        for item in payload
        if int(item.get("width", 0)) >= 1600 and int(item.get("height", 0)) >= 1000
    ]
    random.shuffle(samples)
    return samples[:count]


def select_mapping_keys(value: str, choices: dict[str, Any], label: str) -> list[str]:
    selected = [part.strip() for part in value.split(",") if part.strip()]
    if not selected:
        raise ValueError(f"at least one {label} is required")
    unknown = [item for item in selected if item not in choices]
    if unknown:
        raise ValueError(f"unknown {label}: {', '.join(unknown)}")
    return list(dict.fromkeys(selected))


def select_values(value: str, choices: tuple[str, ...], label: str) -> list[str]:
    selected = [part.strip() for part in value.split(",") if part.strip()]
    if not selected:
        raise ValueError(f"at least one {label} is required")
    unknown = [item for item in selected if item not in choices]
    if unknown:
        raise ValueError(f"unknown {label}: {', '.join(unknown)}")
    return list(dict.fromkeys(selected))


def download_sample_image(sample: SourceSample, *, width: int, height: int) -> Image.Image:
    url = PICSUM_IMAGE_URL.format(id=sample.id, width=width, height=height)
    request = urllib.request.Request(url, headers={"User-Agent": "yuem-watermark-batch-test/1.0"})
    with urllib.request.urlopen(request, timeout=45) as response:
        data = response.read()
    return ImageOps.exif_transpose(Image.open(io.BytesIO(data))).convert("RGB")


def fit_cover(image: Image.Image, size: tuple[int, int]) -> Image.Image:
    return ImageOps.fit(
        image.convert("RGB"),
        size,
        method=Image.Resampling.LANCZOS,
        centering=(0.5, 0.5),
    )


def encode_image(image: Image.Image, image_format: str, options: dict[str, Any]) -> bytes:
    out = io.BytesIO()
    save_image = image.convert("RGB")
    save_image.save(out, format=image_format, **options)
    return out.getvalue()


def short_code(
    sample_id: str,
    resolution: str,
    source_format: str,
    target_size: tuple[int, int],
) -> bytes:
    seed = f"{sample_id}:{resolution}:{source_format}:{target_size[0]}x{target_size[1]}".encode()
    return hashlib.blake2s(seed, digest_size=4).digest()


def make_attack(
    protected_image: Image.Image,
    protected_bytes: bytes,
    attack: str,
) -> tuple[bytes, tuple[int, int]]:
    if attack == "lossless_webp_self":
        return protected_bytes, protected_image.size
    if attack == "jpeg75":
        return (
            encode_image(protected_image, "JPEG", {"quality": 75, "optimize": True}),
            protected_image.size,
        )
    if attack == "webp85":
        return (
            encode_image(protected_image, "WEBP", {"quality": 85, "method": 6}),
            protected_image.size,
        )
    if attack == "resize82_jpeg75_restore":
        resized_size = (
            max(1, round(protected_image.width * 0.82)),
            max(1, round(protected_image.height * 0.82)),
        )
        resized = protected_image.resize(resized_size, Image.Resampling.LANCZOS)
        return encode_image(resized, "JPEG", {"quality": 75, "optimize": True}), resized_size
    raise ValueError(f"unknown attack: {attack}")


def recover_dimension_candidates(
    protected_size: tuple[int, int],
    attacked_size: tuple[int, int],
    *,
    include_common: bool,
) -> tuple[tuple[int, int], ...]:
    width, height = protected_size
    candidates = [(width, height)]
    if attacked_size != protected_size:
        candidates.append(attacked_size)

    if not include_common:
        return tuple(candidates)

    landscape = width >= height
    common_edges = (720, 960, 1080, 1280, 1440, 1920, 2048, 2160, 2560, 3840)
    ratio = width / height
    for edge in common_edges:
        if landscape:
            candidate = (edge, max(1, round(edge / ratio)))
        else:
            candidate = (max(1, round(edge / ratio)), edge)
        candidates.append(candidate)

    unique: list[tuple[int, int]] = []
    for candidate in candidates:
        if candidate[0] <= 0 or candidate[1] <= 0:
            continue
        if candidate not in unique:
            unique.append(candidate)
    return tuple(unique)


def flatten_candidates(groups: dict[int, list[bytes]]) -> list[str]:
    values: list[str] = []
    for payload_bytes in sorted(groups, reverse=True):
        for payload in groups[payload_bytes]:
            values.append(f"{payload_bytes}:{payload.hex()}")
    return values


def write_row(
    writer: csv.DictWriter,
    csv_file: Any,
    rows: list[CaseResult],
    row: CaseResult,
) -> None:
    rows.append(row)
    writer.writerow(asdict(row))
    csv_file.flush()


def write_summary(path: Path, rows: list[CaseResult]) -> None:
    summary = build_summary(rows)
    path.write_text(json.dumps(summary, ensure_ascii=False, indent=2), encoding="utf-8")


def build_summary(rows: list[CaseResult]) -> dict[str, Any]:
    groups: dict[str, dict[str, Any]] = {}
    dimensions = {
        "overall": lambda row: "overall",
        "by_attack": lambda row: row.attack,
        "by_resolution": lambda row: row.resolution,
        "by_source_format": lambda row: row.source_format,
        "by_orientation": lambda row: row.orientation,
        "by_resolution_attack": lambda row: f"{row.resolution}|{row.attack}",
    }
    for group_name, key_func in dimensions.items():
        stats: dict[str, dict[str, Any]] = {}
        for row in rows:
            key = key_func(row)
            bucket = stats.setdefault(
                key,
                {
                    "total": 0,
                    "success": 0,
                    "prefix_3": 0,
                    "prefix_2": 0,
                    "avg_embed_ms": 0.0,
                    "avg_extract_ms": 0.0,
                },
            )
            bucket["total"] += 1
            bucket["success"] += 1 if row.success else 0
            bucket["prefix_3"] += 1 if row.prefix_3_byte_success else 0
            bucket["prefix_2"] += 1 if row.prefix_2_byte_success else 0
            bucket["avg_embed_ms"] += row.embed_ms
            bucket["avg_extract_ms"] += row.extract_ms
        for bucket in stats.values():
            total = max(1, int(bucket["total"]))
            bucket["success_rate"] = round(bucket["success"] / total, 4)
            bucket["prefix_3_rate"] = round(bucket["prefix_3"] / total, 4)
            bucket["prefix_2_rate"] = round(bucket["prefix_2"] / total, 4)
            bucket["avg_embed_ms"] = round(bucket["avg_embed_ms"] / total)
            bucket["avg_extract_ms"] = round(bucket["avg_extract_ms"] / total)
        groups[group_name] = stats
    return {
        "generated_at": time.strftime("%Y-%m-%dT%H:%M:%S%z"),
        "row_count": len(rows),
        "payload_bytes_embedded": 4,
        "payload_bytes_extracted": [4, 3, 2],
        "engine": "dwt_dct_svd",
        "repeat": 9,
        "scale": 64,
        "groups": groups,
    }


def print_summary(rows: list[CaseResult]) -> None:
    summary = build_summary(rows)
    overall = summary["groups"]["overall"].get("overall", {})
    print(
        "summary overall "
        f"success={overall.get('success', 0)}/{overall.get('total', 0)} "
        f"rate={overall.get('success_rate', 0)}",
        flush=True,
    )
    for key, value in summary["groups"]["by_attack"].items():
        print(
            f"summary attack={key} success={value['success']}/{value['total']} "
            f"rate={value['success_rate']} avg_extract_ms={value['avg_extract_ms']}",
            flush=True,
        )
    for key, value in summary["groups"]["by_resolution"].items():
        print(
            f"summary resolution={key} success={value['success']}/{value['total']} "
            f"rate={value['success_rate']} avg_embed_ms={value['avg_embed_ms']}",
            flush=True,
        )


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except KeyboardInterrupt:
        print("interrupted", file=sys.stderr, flush=True)
        raise
