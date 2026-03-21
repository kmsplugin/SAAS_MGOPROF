"""
Pipeline — главный оркестратор обработки аудио.

Последовательность шагов:
  1. Скачать аудио (если URL)
  2. VAD — обнаружение речи
  3. ASR (faster-whisper) — транскрипция с таймкодами
  4. Alignment (WhisperX) — уточнение слов до мс
  5. Diarization (pyannote) — кто говорил
  6. Merge — присвоить спикеров сегментам
  7. Punctuation — восстановить знаки препинания
  8. Translation — перевод (опционально)
  9. Summary/Highlights/Chapters/ActionItems (Claude API, опционально)
 10. Keywords (KeyBERT)
 11. Speaker Analytics
 12. Export (SRT/VTT/JSON/DOCX/CSV)
 13. Сохранить результаты в БД
"""

from __future__ import annotations
import asyncio
import json
import logging
import os
import tempfile
from pathlib import Path
from typing import Optional

import httpx

from config import settings
from models.schemas import ProcessRequest, ProcessResult, Segment, WordSegment, Keyword, SpeakerStats, ExportFormat
from services import vad, asr, alignment, diarization, punctuation, translation, keywords, analytics
from services.summary import generate_all as generate_summaries
from exporters import to_srt, to_vtt, to_csv, words_to_csv, to_docx

log = logging.getLogger(__name__)


async def _download_audio(url: str) -> str:
    """Скачивает аудио по URL во временный файл. Возвращает путь."""
    ext = Path(url.split("?")[0]).suffix or ".mp4"
    tmp = tempfile.NamedTemporaryFile(
        delete=False,
        suffix=ext,
        dir=settings.temp_dir,
    )
    tmp.close()

    log.info("Pipeline: downloading %s → %s", url, tmp.name)
    async with httpx.AsyncClient(timeout=1800) as client:
        async with client.stream("GET", url) as resp:
            resp.raise_for_status()
            with open(tmp.name, "wb") as f:
                async for chunk in resp.aiter_bytes(chunk_size=1 << 20):
                    f.write(chunk)

    log.info("Pipeline: downloaded %.1f MB", os.path.getsize(tmp.name) / 1e6)
    return tmp.name


def _build_segments(raw_segments: list[dict]) -> list[Segment]:
    result = []
    for seg in raw_segments:
        words = [
            WordSegment(
                word=w.get("word", ""),
                start=w.get("start", 0),
                end=w.get("end", 0),
                score=w.get("score", w.get("probability", 1.0)),
                speaker=w.get("speaker"),
            )
            for w in seg.get("words", [])
        ]
        result.append(Segment(
            id=seg.get("id", 0),
            start=seg.get("start", 0),
            end=seg.get("end", 0),
            text=seg.get("text", "").strip(),
            speaker=seg.get("speaker"),
            words=words,
            confidence=seg.get("confidence", 1.0),
        ))
    return result


async def run(req: ProcessRequest, status_callback=None) -> ProcessResult:
    """
    Запускает полный pipeline.

    Args:
        req: параметры задачи
        status_callback: async callable(step: str) для обновления статуса

    Returns:
        ProcessResult со всеми данными
    """
    os.makedirs(settings.temp_dir, exist_ok=True)
    audio_path = req.audio_url
    is_temp = False

    async def _step(name: str):
        log.info("Pipeline[%s]: %s", req.event_id, name)
        if status_callback:
            await status_callback(name)

    try:
        # 1. Скачать если URL
        if req.audio_url.startswith(("http://", "https://")):
            await _step("downloading")
            audio_path = await _download_audio(req.audio_url)
            is_temp = True

        # 2. VAD
        await _step("vad")
        vad_segments = await asyncio.to_thread(
            vad.detect_speech_segments, audio_path
        )

        # 3. ASR
        await _step("transcribing")
        asr_result = await asyncio.to_thread(
            asr.transcribe, audio_path, req.language
        )

        raw_segments = asr_result["segments"]
        detected_lang = asr_result["language"]
        duration = asr_result["duration"]

        # 4. Alignment
        await _step("aligning")
        try:
            raw_segments = await asyncio.to_thread(
                alignment.align_segments, raw_segments, audio_path, detected_lang
            )
        except Exception as e:
            log.warning("Pipeline: alignment failed, continuing: %s", e)

        # 5. Diarization
        if req.diarize and settings.hf_token:
            await _step("diarizing")
            try:
                turns = await asyncio.to_thread(diarization.diarize, audio_path)
                raw_segments = diarization.assign_speakers_to_segments(raw_segments, turns)
            except Exception as e:
                log.warning("Pipeline: diarization failed, continuing: %s", e)
        else:
            if req.diarize:
                log.warning("Pipeline: diarization requested but HF_TOKEN not set")

        # 6. Punctuation
        await _step("punctuation")
        transcript_raw = " ".join(s.get("text", "") for s in raw_segments)
        try:
            transcript_punct = await asyncio.to_thread(
                punctuation.restore_punctuation, transcript_raw
            )
            transcript = punctuation.normalize_text(transcript_punct)
        except Exception as e:
            log.warning("Pipeline: punctuation failed: %s", e)
            transcript = transcript_raw

        # 7. Translation
        translation_text: Optional[str] = None
        if req.translate_to and req.translate_to != detected_lang:
            await _step("translating")
            try:
                translation_text = await asyncio.to_thread(
                    translation.translate, transcript, detected_lang, req.translate_to
                )
            except Exception as e:
                log.warning("Pipeline: translation failed: %s", e)

        # 8. Summary (параллельно с остальными NLP)
        await _step("analyzing")
        summary_data = await generate_summaries(transcript)

        # 9. Keywords
        kw_raw = await asyncio.to_thread(keywords.extract_keywords, transcript, 20)
        kw_list = [Keyword(keyword=k["keyword"], score=k["score"]) for k in kw_raw]

        # 10. Speaker Analytics
        seg_dicts = raw_segments  # уже dict из модулей
        stats_raw = analytics.compute_speaker_stats(seg_dicts, duration)
        speaker_stats = [
            SpeakerStats(
                speaker=s["speaker"],
                total_seconds=s["total_seconds"],
                segment_count=s["segment_count"],
                word_count=s["word_count"],
                percentage=s["percentage"],
            )
            for s in stats_raw
        ]

        # 11. Build typed segments
        segments = _build_segments(raw_segments)
        all_words = [w for seg in segments for w in seg.words]

        # 12. Exports
        await _step("exporting")
        exports: dict[str, str] = {}
        seg_dicts_for_export = [s.model_dump() for s in segments]

        for fmt in req.export_formats:
            try:
                if fmt == ExportFormat.srt:
                    exports["srt"] = to_srt(seg_dicts_for_export)
                elif fmt == ExportFormat.vtt:
                    exports["vtt"] = to_vtt(seg_dicts_for_export)
                elif fmt == ExportFormat.csv:
                    exports["csv"] = to_csv(seg_dicts_for_export)
                elif fmt == ExportFormat.json:
                    exports["json"] = json.dumps({
                        "segments": seg_dicts_for_export,
                        "words":    [w.model_dump() for w in all_words],
                        "speaker_stats": [s.model_dump() for s in speaker_stats],
                    }, ensure_ascii=False, indent=2)
                elif fmt == ExportFormat.docx:
                    docx_bytes = to_docx(
                        segments=seg_dicts_for_export,
                        transcript=transcript,
                        summary=summary_data.get("summary") or "",
                        highlights=summary_data.get("highlights") or "",
                        chapters=summary_data.get("chapters") or "",
                        action_items=summary_data.get("action_items") or "",
                        keywords=kw_raw,
                        speaker_stats=stats_raw,
                    )
                    # DOCX сохраняем как файл, в exports — путь
                    docx_path = os.path.join(
                        settings.temp_dir,
                        f"meeting_{req.event_id}.docx",
                    )
                    with open(docx_path, "wb") as f:
                        f.write(docx_bytes)
                    exports["docx"] = docx_path
            except Exception as e:
                log.error("Pipeline: export[%s] failed: %s", fmt, e)

        result = ProcessResult(
            event_id=req.event_id,
            tenant_id=req.tenant_id,
            language_detected=detected_lang,
            duration_seconds=duration,
            transcript=transcript,
            segments=segments,
            words=all_words,
            translation=translation_text,
            summary=summary_data.get("summary"),
            highlights=summary_data.get("highlights"),
            chapters=summary_data.get("chapters"),
            action_items=summary_data.get("action_items"),
            keywords=kw_list,
            speaker_stats=speaker_stats,
            exports=exports,
        )

        await _step("done")
        return result

    finally:
        if is_temp and audio_path and os.path.exists(audio_path):
            os.remove(audio_path)
            log.debug("Pipeline: removed temp file %s", audio_path)
