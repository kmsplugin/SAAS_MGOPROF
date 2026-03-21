"""
Alignment — word-level forced alignment (WhisperX).
Уточняет таймкоды до уровня отдельных слов с высокой точностью.
"""

from __future__ import annotations
import logging
from typing import Optional
import whisperx
from config import settings

log = logging.getLogger(__name__)

# Кэш alignment моделей по языку
_align_models: dict[str, tuple] = {}


def _load_align_model(language: str):
    if language not in _align_models:
        log.info("Alignment: loading model for lang=%s", language)
        model, metadata = whisperx.load_align_model(
            language_code=language,
            device=settings.whisper_device,
        )
        _align_models[language] = (model, metadata)
        log.info("Alignment: model loaded for lang=%s", language)
    return _align_models[language]


def align_segments(
    segments: list[dict],
    audio_path: str,
    language: str,
) -> list[dict]:
    """
    Запускает WhisperX forced alignment.
    Возвращает сегменты с уточнёнными word-level таймкодами.

    Args:
        segments: сегменты из faster-whisper (с полем "words")
        audio_path: путь к аудиофайлу
        language: код языка ("ru", "en", ...)

    Returns:
        Обновлённые сегменты с уточнёнными таймкодами слов.
    """
    try:
        model, metadata = _load_align_model(language)
    except Exception as e:
        log.warning("Alignment: cannot load model for lang=%s, skipping: %s", language, e)
        return segments  # возвращаем исходные без alignment

    import whisperx.audio as wa
    audio = wa.load_audio(audio_path)

    result = whisperx.align(
        segments,
        model,
        metadata,
        audio,
        device=settings.whisper_device,
        return_char_alignments=False,
    )

    aligned = result.get("segments", segments)
    log.info("Alignment: done, %d segments", len(aligned))
    return aligned
