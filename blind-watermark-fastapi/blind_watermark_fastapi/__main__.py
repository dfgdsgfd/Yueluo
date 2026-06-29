import uvicorn

from .settings import get_settings


def main() -> None:
    settings = get_settings()
    uvicorn.run(
        "blind_watermark_fastapi.main:app",
        host=settings.host,
        port=settings.port,
        factory=False,
    )


if __name__ == "__main__":
    main()
