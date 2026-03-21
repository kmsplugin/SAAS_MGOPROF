from pydantic_settings import BaseSettings
from typing import Literal


class Settings(BaseSettings):
    # HTTP
    port: int = 8030
    workers: int = 1  # uvicorn workers (1 = single process, GPU memory shared)

    # Database
    database_url: str = "postgresql+asyncpg://platform:platform_secret@localhost:5432/platform"

    # Whisper model
    whisper_model: str = "large-v3"       # tiny/base/small/medium/large-v2/large-v3
    whisper_device: str = "cpu"            # "cpu" | "cuda"
    whisper_compute_type: str = "int8"     # int8 / float16 / float32
    whisper_language: str = "ru"           # default language

    # Diarization
    hf_token: str = ""                    # HuggingFace token для pyannote.audio
    diarization_enabled: bool = True

    # Translation
    translation_enabled: bool = True
    translation_target_lang: str = "ru"   # целевой язык перевода

    # Summary via Claude API (опционально — если не задан, summary пропускается)
    anthropic_api_key: str = ""
    anthropic_model: str = "claude-sonnet-4-6"

    # Processing
    max_audio_size_mb: int = 500
    temp_dir: str = "/tmp/speech"

    # Internal auth (от ai-service или main API)
    internal_token: str = "dev_internal_token_change_in_prod"

    class Config:
        env_file = ".env"
        extra = "ignore"


settings = Settings()
