from __future__ import annotations
from pydantic import BaseModel, Field
from typing import Optional
from enum import Enum
import uuid


class ExportFormat(str, Enum):
    srt  = "srt"
    vtt  = "vtt"
    json = "json"
    docx = "docx"
    csv  = "csv"


class ProcessRequest(BaseModel):
    event_id:  str
    tenant_id: str
    audio_url: str
    language:  str = "ru"
    translate_to: Optional[str] = None      # язык перевода, None = не переводить
    export_formats: list[ExportFormat] = [ExportFormat.json]
    diarize: bool = True


class WordSegment(BaseModel):
    word:       str
    start:      float
    end:        float
    score:      float = 1.0
    speaker:    Optional[str] = None


class Segment(BaseModel):
    id:         int
    start:      float
    end:        float
    text:       str
    speaker:    Optional[str] = None
    words:      list[WordSegment] = []
    confidence: float = 1.0


class SpeakerStats(BaseModel):
    speaker:       str
    total_seconds: float
    segment_count: int
    word_count:    int
    percentage:    float


class Keyword(BaseModel):
    keyword: str
    score:   float


class ProcessResult(BaseModel):
    event_id:         str
    tenant_id:        str
    language_detected: str
    duration_seconds: float

    # Основной контент
    transcript:   str                    # plain text
    segments:     list[Segment]          # с таймкодами и спикерами
    words:        list[WordSegment]      # все слова с таймкодами

    # Расширенный анализ
    translation:  Optional[str]  = None  # перевод
    summary:      Optional[str]  = None  # резюме (Claude)
    highlights:   Optional[str]  = None  # ключевые моменты
    chapters:     Optional[str]  = None  # главы
    action_items: Optional[str]  = None  # задачи

    keywords:     list[Keyword]  = []
    speaker_stats: list[SpeakerStats] = []

    # Экспорты
    exports:      dict[str, str] = {}    # format → content/path


class JobStatus(BaseModel):
    job_id:    str
    event_id:  str
    status:    str   # queued | processing | done | failed
    step:      str = ""
    error:     str = ""


class WebhookPayload(BaseModel):
    event_id:  str
    tenant_id: str
    audio_url: str
    language:  str = "ru"
