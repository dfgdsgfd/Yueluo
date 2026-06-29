from functools import lru_cache

from pydantic import Field
from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    model_config = SettingsConfigDict(env_prefix="BLIND_WATERMARK_", env_file=".env")

    api_key: str = ""
    host: str = "0.0.0.0"
    port: int = Field(default=8090, ge=1, le=65535)
    max_image_bytes: int = Field(default=10 * 1024 * 1024, ge=1, le=100 * 1024 * 1024)
    max_workers: int = Field(default=2, ge=1, le=16)
    password_wm: int = 1
    password_img: int = 1
    recovery_search_num: int = Field(default=80, ge=20, le=500)


@lru_cache
def get_settings() -> Settings:
    return Settings()
