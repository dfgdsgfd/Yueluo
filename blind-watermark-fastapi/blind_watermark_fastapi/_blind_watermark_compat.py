from __future__ import annotations

import multiprocessing
import sys
from collections.abc import Callable

_GUARD_ATTR = "_yuem_blind_watermark_guard"


def install_blind_watermark_multiprocessing_guard() -> None:
    current = multiprocessing.set_start_method
    if getattr(current, _GUARD_ATTR, False):
        return

    original: Callable[..., None] = current

    def guarded_set_start_method(method: str | None, force: bool = False) -> None:
        try:
            original(method, force=force)
        except RuntimeError as exc:
            if (
                sys.platform != "win32"
                and method == "fork"
                and not force
                and "context has already been set" in str(exc)
            ):
                return
            raise

    setattr(guarded_set_start_method, _GUARD_ATTR, True)
    multiprocessing.set_start_method = guarded_set_start_method
