# ── Builder ───────────────────────────────────────────────────────────────────
FROM python:3.11-slim AS base

ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1 \
    PIP_NO_CACHE_DIR=1 \
    PIP_DISABLE_PIP_VERSION_CHECK=1

# Системные зависимости: ffmpeg (аудиодекодирование), libsndfile (torchaudio)
RUN apt-get update && apt-get install -y --no-install-recommends \
    ffmpeg \
    libsndfile1 \
    libgomp1 \
    git \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Зависимости
COPY requirements.txt .

# Torch CPU (замените URL на CUDA-версию при деплое на GPU)
RUN pip install --extra-index-url https://download.pytorch.org/whl/cpu \
    torch==2.3.0+cpu \
    torchaudio==2.3.0+cpu \
    && pip install -r requirements.txt

# Исходники
COPY . .

EXPOSE 8030

# Temp директория для аудио
RUN mkdir -p /tmp/speech

# Модели скачиваются при первом запросе в /tmp/huggingface
ENV HF_HOME=/data/huggingface \
    TORCH_HOME=/data/torch \
    SILERO_VAD_CACHE=/data/silero

HEALTHCHECK --interval=30s --timeout=10s --start-period=60s --retries=3 \
    CMD python -c "import httpx; httpx.get('http://localhost:8030/health').raise_for_status()"

CMD ["python", "-m", "uvicorn", "main:app", \
     "--host", "0.0.0.0", \
     "--port", "8030", \
     "--workers", "1", \
     "--log-level", "info"]
