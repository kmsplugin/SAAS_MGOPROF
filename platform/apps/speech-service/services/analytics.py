"""
Speaker Analytics — анализ активности спикеров.
Считает время речи, количество сегментов, слов и долю эфира каждого спикера.
"""

from __future__ import annotations
from collections import defaultdict


def compute_speaker_stats(
    segments: list[dict],
    total_duration: float,
) -> list[dict]:
    """
    Вычисляет статистику по каждому спикеру.

    Args:
        segments: список сегментов с полем "speaker" (может быть None)
        total_duration: общая длительность аудио в секундах

    Returns:
        [
            {
                "speaker":       "SPEAKER_00",
                "total_seconds": 120.5,
                "segment_count": 15,
                "word_count":    250,
                "percentage":    48.2,
                "avg_segment_duration": 8.0,
                "wpm":           124.3,  # слов в минуту
            },
            ...
        ]  — отсортировано по total_seconds DESC
    """
    stats: dict[str, dict] = defaultdict(lambda: {
        "total_seconds": 0.0,
        "segment_count": 0,
        "word_count": 0,
    })

    for seg in segments:
        speaker = seg.get("speaker") or "UNKNOWN"
        duration = max(0.0, seg.get("end", 0) - seg.get("start", 0))
        words = len(seg.get("words", []) or seg.get("text", "").split())

        stats[speaker]["total_seconds"] += duration
        stats[speaker]["segment_count"] += 1
        stats[speaker]["word_count"] += words

    result = []
    for speaker, data in stats.items():
        secs = data["total_seconds"]
        pct  = round(secs / total_duration * 100, 1) if total_duration > 0 else 0.0
        avg  = round(secs / data["segment_count"], 2) if data["segment_count"] else 0.0
        wpm  = round(data["word_count"] / (secs / 60), 1) if secs > 0 else 0.0

        result.append({
            "speaker":              speaker,
            "total_seconds":        round(secs, 2),
            "segment_count":        data["segment_count"],
            "word_count":           data["word_count"],
            "percentage":           pct,
            "avg_segment_duration": avg,
            "wpm":                  wpm,
        })

    return sorted(result, key=lambda x: -x["total_seconds"])


def speaking_timeline(segments: list[dict]) -> list[dict]:
    """
    Возвращает хронологическую шкалу говорящих.
    Удобно для визуализации "кто когда говорил".

    Returns:
        [{"speaker": str, "start": float, "end": float}, ...]
    """
    timeline = []
    for seg in segments:
        speaker = seg.get("speaker") or "UNKNOWN"
        timeline.append({
            "speaker": speaker,
            "start":   seg.get("start", 0.0),
            "end":     seg.get("end",   0.0),
        })
    return sorted(timeline, key=lambda x: x["start"])
