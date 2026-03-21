"""
Diarization — кто говорил, когда (pyannote.audio).
Требует HuggingFace токен и принятые условия использования моделей:
  https://huggingface.co/pyannote/speaker-diarization-3.1
  https://huggingface.co/pyannote/segmentation-3.0
"""

from __future__ import annotations
import logging
from typing import Optional
from config import settings

log = logging.getLogger(__name__)

_pipeline = None


def _load_pipeline():
    global _pipeline
    if _pipeline is not None:
        return _pipeline

    if not settings.hf_token:
        log.warning("Diarization: HF_TOKEN not set, diarization disabled")
        return None

    from pyannote.audio import Pipeline
    import torch

    log.info("Diarization: loading pyannote pipeline...")
    _pipeline = Pipeline.from_pretrained(
        "pyannote/speaker-diarization-3.1",
        use_auth_token=settings.hf_token,
    )

    device = torch.device(settings.whisper_device)
    _pipeline = _pipeline.to(device)
    log.info("Diarization: pipeline loaded on %s", settings.whisper_device)
    return _pipeline


def diarize(audio_path: str, num_speakers: Optional[int] = None) -> list[dict]:
    """
    Запускает pyannote.audio diarization.

    Returns:
        [{"speaker": "SPEAKER_00", "start": 0.0, "end": 3.5}, ...]
    """
    pipe = _load_pipeline()
    if pipe is None:
        return []

    kwargs = {}
    if num_speakers:
        kwargs["num_speakers"] = num_speakers

    log.info("Diarization: processing %s...", audio_path)
    diarization = pipe(audio_path, **kwargs)

    turns = []
    for turn, _, speaker in diarization.itertracks(yield_label=True):
        turns.append({
            "speaker": speaker,
            "start":   round(turn.start, 3),
            "end":     round(turn.end,   3),
        })

    log.info("Diarization: found %d turns, speakers: %s",
             len(turns),
             {t["speaker"] for t in turns})
    return turns


def assign_speakers_to_segments(
    segments: list[dict],
    diarization_turns: list[dict],
) -> list[dict]:
    """
    Сопоставляет каждый сегмент транскрипции с говорящим.
    Использует метод максимального перекрытия по времени.
    """
    if not diarization_turns:
        return segments

    for seg in segments:
        seg_start = seg["start"]
        seg_end   = seg["end"]

        best_speaker = None
        best_overlap = 0.0

        for turn in diarization_turns:
            overlap = min(seg_end, turn["end"]) - max(seg_start, turn["start"])
            if overlap > best_overlap:
                best_overlap = overlap
                best_speaker = turn["speaker"]

        seg["speaker"] = best_speaker

        # Назначаем спикера отдельным словам
        for word in seg.get("words", []):
            w_start = word.get("start", seg_start)
            w_end   = word.get("end",   seg_end)
            w_best_speaker = None
            w_best_overlap = 0.0
            for turn in diarization_turns:
                ov = min(w_end, turn["end"]) - max(w_start, turn["start"])
                if ov > w_best_overlap:
                    w_best_overlap = ov
                    w_best_speaker = turn["speaker"]
            word["speaker"] = w_best_speaker

    return segments
