"""SRT subtitle export."""

from __future__ import annotations


def _fmt_time(seconds: float) -> str:
    """00:01:23,456"""
    h = int(seconds // 3600)
    m = int((seconds % 3600) // 60)
    s = int(seconds % 60)
    ms = int(round((seconds - int(seconds)) * 1000))
    return f"{h:02d}:{m:02d}:{s:02d},{ms:03d}"


def to_srt(segments: list[dict]) -> str:
    lines = []
    for i, seg in enumerate(segments, 1):
        speaker = seg.get("speaker")
        text    = seg.get("text", "").strip()
        start   = _fmt_time(seg.get("start", 0))
        end     = _fmt_time(seg.get("end",   0))

        if speaker:
            text = f"[{speaker}] {text}"

        lines.append(str(i))
        lines.append(f"{start} --> {end}")
        lines.append(text)
        lines.append("")

    return "\n".join(lines)
