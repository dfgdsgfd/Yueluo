import importlib

from .._blind_watermark_compat import install_blind_watermark_multiprocessing_guard

install_blind_watermark_multiprocessing_guard()
blind_watermark = importlib.import_module("blind_watermark")
WaterMark = blind_watermark.WaterMark
recover_module = importlib.import_module("blind_watermark.recover")
estimate_crop_parameters = recover_module.estimate_crop_parameters
recover_crop = recover_module.recover_crop
blind_watermark.bw_notes.close()
