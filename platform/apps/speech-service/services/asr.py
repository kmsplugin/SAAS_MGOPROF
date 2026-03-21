"""
ASR — Automatic Speech Recognition (faster-whisper).
Транскрибирует аудио, возвращает сегменты с таймкодами и confidence score.
"""

from __future__ import annotations
import logging
from typing import Optional
from faster_whisper import WhisperModel
from config import settings

log = logging.getLogger(__name__)

_model: Optional[WhisperModel] = None


def _load_model() -> WhisperModel:
    global _model
    if _model is None:
        log.info(
            "ASR: loading faster-whisper model=%s device=%s compute=%s",
            settings.whisper_model,
            settings.whisper_device,
            settings.whisper_compute_type,
        )
        _model = WhisperModel(
            model_size_or_path=settings.whisper_model,
            device=settings.whisper_device,
            compute_type=settings.whisper_compute_type,
        )
        log.info("ASR: model loaded")
    return _model


def transcribe(
    audio_path: str,
    language: Optional[str] = None,
    initial_prompt: Optional[str] = None,
    vad_filter: bool = True,
) -> dict:
    """
    Запускает faster-whisper и возвращает словарь:
    {
        "language": "ru",
        "language_probability": 0.99,
        "duration": 1234.5,
        "segments": [
            {
                "id": 0,
                "start": 0.0,
                "end": 3.5,
                "text": "Привет мир",
                "words": [{"word": "Привет", "start": 0.0, "end": 1.2, "probability": 0.98}, ...],
                "avg_logprob": -0.2,
                "no_speech_prob": 0.01,
            },
            ...
        ],
        "transcript": "Привет мир ...",
    }
    """
    model = _load_model()

    lang = language or settings.whisper_language or None

    segments_iter, info = model.transcribe(
        audio_path,
        language=lang,
        word_timestamps=True,
        vad_filter=vad_filter,
        initial_prompt=initial_prompt,
        beam_size=5,
        best_of=5,
        temperature=0.0,
        condition_on_previous_text=True,
        no_speech_threshold=0.6,
        log_prob_threshold=-1.0,
        compression_ratio_threshold=2.4,
    )

    segments = []
    all_words = []

    for i, seg in enumerate(segments_iter):
        words = []
        if seg.words:
            for w in seg.words:
                word_d = {
                    "word":  w.word.strip(),
                    "start": round(w.start, 3),
                    "end":   round(w.end,   3),
                    "score": round(w.probability, 4),
                }
                words.append(word_d)
                all_words.append(word_d)

        segments.append({
            "id":          i,
            "start":       round(seg.start, 3),
            "end":         round(seg.end,   3),
            "text":        seg.text.strip(),
            "words":       words,
            "confidence":  round(max(0.0, seg.avg_logprob + 1.0), 4),  # нормализованный
            "no_speech_prob": round(seg.no_speech_prob, 4),
        })

    transcript = " ".join(s["text"] for s in segments)

    result = {
        "language":             info.language,
        "language_probability": round(info.language_probability, 4),
        "duration":             round(info.duration, 3),
        "segments":             segments,
        "words":                all_words,
        "transcript":           transcript,
    }

    log.info(
        "ASR: done lang=%s prob=%.2f duration=%.1fs segments=%d",
        info.language,
        info.language_probability,
        info.duration,
        len(segments),
    )
    return result
