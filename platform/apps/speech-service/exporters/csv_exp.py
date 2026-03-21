"""CSV export — каждый сегмент как строка."""

from __future__ import annotations
import csv
import io


def to_csv(segments: list[dict]) -> str:
    buf = io.StringIO()
    writer = csv.writer(buf, quoting=csv.QUOTE_ALL)
    writer.writerow(["id", "start", "end", "speaker", "text", "confidence"])

    for seg in segments:
        writer.writerow([
            seg.get("id", ""),
            seg.get("start", ""),
            seg.get("end",   ""),
            seg.get("speaker") or "",
            seg.get("text",  "").strip(),
            seg.get("confidence", ""),
        ])

    return buf.getvalue()


def words_to_csv(words: list[dict]) -> str:
    """Экспорт отдельных слов с таймкодами."""
    buf = io.StringIO()
    writer = csv.writer(buf, quoting=csv.QUOTE_ALL)
    writer.writerow(["word", "start", "end", "score", "speaker"])

    for w in words:
        writer.writerow([
            w.get("word",    ""),
            w.get("start",   ""),
            w.get("end",     ""),
            w.get("score",   ""),
            w.get("speaker") or "",
        ])

    return buf.getvalue()
