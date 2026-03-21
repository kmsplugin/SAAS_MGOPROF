"""
speech-service — self-hosted AI speech subsystem.
Port: 8030

Endpoints:
  GET  /health
  POST /process              — запустить обработку (async, возвращает job_id)
  GET  /jobs/{job_id}        — статус задачи
  GET  /results/{event_id}   — получить результаты обработки
  GET  /results/{event_id}/export/{format}  — скачать экспорт
  POST /webhooks/egress      — вебхук от LiveKit Egress
"""

from __future__ import annotations
import asyncio
import json
import logging
import os
import uuid
from contextlib import asynccontextmanager
from typing import Optional

from fastapi import FastAPI, HTTPException, Header, BackgroundTasks, Response
from fastapi.responses import FileResponse, JSONResponse
from pydantic import BaseModel

from config import settings
from models.schemas import ProcessRequest, ProcessResult, JobStatus, ExportFormat, WebhookPayload
import pipeline as pl

log = logging.getLogger(__name__)
logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(name)s: %(message)s")


# ── In-memory job store (заменить на Redis/DB для multi-replica) ──────────────
_jobs:    dict[str, JobStatus] = {}
_results: dict[str, ProcessResult] = {}  # event_id → result


# ── Lifespan ──────────────────────────────────────────────────────────────────
@asynccontextmanager
async def lifespan(_: FastAPI):
    os.makedirs(settings.temp_dir, exist_ok=True)
    log.info("speech-service started on port %d", settings.port)
    yield
    log.info("speech-service stopped")


app = FastAPI(
    title="Speech Service",
    description="Self-hosted STT / Diarization / NLP pipeline",
    version="1.0.0",
    lifespan=lifespan,
)


# ── Background worker ─────────────────────────────────────────────────────────

async def _process_job(job_id: str, req: ProcessRequest):
    """Запускает pipeline в фоне и обновляет статус."""
    async def _update(step: str):
        if job_id in _jobs:
            _jobs[job_id].step = step

    try:
        _jobs[job_id] = JobStatus(job_id=job_id, event_id=req.event_id, status="processing")
        result = await pl.run(req, status_callback=_update)
        _results[req.event_id] = result
        _jobs[job_id].status = "done"
        _jobs[job_id].step   = "done"
        log.info("Job %s done for event %s", job_id, req.event_id)

    except Exception as e:
        log.exception("Job %s failed: %s", job_id, e)
        if job_id in _jobs:
            _jobs[job_id].status = "failed"
            _jobs[job_id].error  = str(e)


# ── Routes ────────────────────────────────────────────────────────────────────

@app.get("/health")
def health():
    return {"status": "ok", "service": "speech-service"}


@app.post("/process", status_code=202)
async def process(
    req: ProcessRequest,
    background_tasks: BackgroundTasks,
    x_internal_token: Optional[str] = Header(None),
):
    """Принимает задачу обработки и ставит её в фон."""
    if x_internal_token != settings.internal_token:
        raise HTTPException(status_code=401, detail="Unauthorized")

    job_id = str(uuid.uuid4())
    _jobs[job_id] = JobStatus(job_id=job_id, event_id=req.event_id, status="queued")
    background_tasks.add_task(_process_job, job_id, req)

    return {"job_id": job_id, "event_id": req.event_id, "status": "queued"}


@app.get("/jobs/{job_id}", response_model=JobStatus)
def get_job(job_id: str):
    job = _jobs.get(job_id)
    if not job:
        raise HTTPException(status_code=404, detail="Job not found")
    return job


@app.get("/results/{event_id}")
def get_result(event_id: str):
    result = _results.get(event_id)
    if not result:
        raise HTTPException(status_code=404, detail="No result for this event yet")
    return result


@app.get("/results/{event_id}/export/{fmt}")
def get_export(event_id: str, fmt: ExportFormat):
    """Отдаёт экспорт в указанном формате."""
    result = _results.get(event_id)
    if not result:
        raise HTTPException(status_code=404, detail="No result")

    content = result.exports.get(fmt.value)
    if not content:
        raise HTTPException(status_code=404, detail=f"Export format '{fmt}' not available")

    media_types = {
        ExportFormat.srt:  "text/srt",
        ExportFormat.vtt:  "text/vtt",
        ExportFormat.csv:  "text/csv",
        ExportFormat.json: "application/json",
        ExportFormat.docx: "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
    }

    if fmt == ExportFormat.docx:
        # content — путь к файлу
        if not os.path.exists(content):
            raise HTTPException(status_code=404, detail="DOCX file not found")
        return FileResponse(
            content,
            media_type=media_types[fmt],
            filename=f"meeting_{event_id}.docx",
        )

    return Response(
        content=content.encode("utf-8"),
        media_type=media_types.get(fmt, "text/plain"),
        headers={"Content-Disposition": f'attachment; filename="meeting_{event_id}.{fmt.value}"'},
    )


@app.post("/webhooks/egress")
async def egress_webhook(
    payload: WebhookPayload,
    background_tasks: BackgroundTasks,
    x_internal_token: Optional[str] = Header(None),
):
    """Получает уведомление об окончании записи и автоматически запускает pipeline."""
    if x_internal_token != settings.internal_token:
        raise HTTPException(status_code=401, detail="Unauthorized")

    req = ProcessRequest(
        event_id=payload.event_id,
        tenant_id=payload.tenant_id,
        audio_url=payload.audio_url,
        language=payload.language,
        export_formats=[ExportFormat.json, ExportFormat.srt, ExportFormat.vtt, ExportFormat.docx],
        diarize=True,
    )

    job_id = str(uuid.uuid4())
    _jobs[job_id] = JobStatus(job_id=job_id, event_id=req.event_id, status="queued")
    background_tasks.add_task(_process_job, job_id, req)

    return {"job_id": job_id, "event_id": req.event_id, "status": "queued"}


if __name__ == "__main__":
    import uvicorn
    uvicorn.run(
        "main:app",
        host="0.0.0.0",
        port=settings.port,
        workers=settings.workers,
        log_level="info",
    )
