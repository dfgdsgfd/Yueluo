import base64
import binascii
from collections.abc import Callable
from dataclasses import dataclass

from ..settings import Settings


class WatermarkError(ValueError):
    pass


@dataclass(frozen=True)
class EmbeddedImage:
    data: bytes
    media_type: str


@dataclass(frozen=True)
class WatermarkOptions:
    password_wm: int
    password_img: int
    d1: int = 36
    d2: int = 20
    engine: str = "blind_watermark"
    dwt_dct_svd_repeat: int = 9
    dwt_dct_svd_scale: int = 64
    recover_dimensions: tuple[tuple[int, int], ...] = ()


ProgressCallback = Callable[[dict[str, object]], None]

_MAX_DIRECT_CANDIDATES = 8
_MAX_RECOVERED_CANDIDATES = 10
_RECOVERY_MATCHES_TO_EXPAND = 2
_RECOVERY_PAIR_LIMIT = 2
_RECOVERY_GOOD_SCORE = 0.65
_FAST_RESIZE_ASPECT_DELTA = 0.08
_FAST_RESIZE_MIN_AREA_RATIO = 0.35
_MAX_FAST_RESIZE_CANDIDATES = 4


def options_from_settings(settings: Settings) -> WatermarkOptions:
    return WatermarkOptions(
        password_wm=settings.password_wm,
        password_img=settings.password_img,
        d1=36,
        d2=20,
    )


def decode_payload_b64(value: str) -> bytes:
    try:
        return base64.b64decode(value, validate=True)
    except (binascii.Error, ValueError) as exc:
        raise WatermarkError("error.payload_invalid") from exc


def encode_payload_b64(value: bytes) -> str:
    return base64.b64encode(value).decode("ascii")
