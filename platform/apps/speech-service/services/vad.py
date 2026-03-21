"""
VAD — Voice Activity Detection (Silero VAD).
Сегментирует аудиофайл: возвращает список интервалов [start, end] в секундах,
где обнаружена речь. Отфильтровывает тишину и фоновый шум.
"""

from __future__ import annotations
import torch
import torchaudio
import logging
from pathlib import Path

log = logging.getLogger(__name__)

# Silero VAD модель грузится один раз на всё время жизни процесса
_model = None
_utils = None


def _load_model():
    global _model, _utils
    if _model is None:
        log.info("VAD: loading Silero VAD model...")
        _model, _utils = torch.hub.load(
            repo_or_dir="snakers4/silero-vad",
            model="silero_vad",
            force_reload=False,
            onnx=False,
        )
        _model.eval()
        log.info("VAD: model loaded")
    return _model, _utils


def detect_speech_segments(
    audio_path: str,
    threshold: float = 0.5,
    min_speech_ms: int = 250,
    min_silence_ms: int = 100,
    speech_pad_ms: int = 30,
) -> list[dict]:
    """
    Запускает Silero VAD и возвращает список сегментов со речью.

    Returns:
        [{"start": float, "end": float}, ...]  — секунды
    """
    model, utils = _load_model()
    (get_speech_timestamps, _, read_audio, _, _) = utils

    # Silero работает с 16kHz mono
    wav = read_audio(audio_path, sampling_rate=16000)

    speech_ts = get_speech_timestamps(
        wav,
        model,
        threshold=threshold,
        min_speech_duration_ms=min_speech_ms,
        min_silence_duration_ms=min_silence_ms,
        speech_pad_ms=speech_pad_ms,
        return_seconds=True,
    )

    segments = [{"start": float(s["start"]), "end": float(s["end"])} for s in speech_ts]
    log.info("VAD: found %d speech segments in %s", len(segments), audio_path)
    return segments


def speech_ratio(segments: list[dict], total_duration: float) -> float:
    """Доля времени с речью от общей длительности."""
    if total_duration <= 0:
        return 0.0
    speech_time = sum(s["end"] - s["start"] for s in segments)
    return speech_time / total_duration
