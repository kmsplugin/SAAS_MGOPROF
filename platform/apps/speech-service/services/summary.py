"""
Summary — генерация резюме, highlights, chapters, action_items через Claude API.
Если ANTHROPIC_API_KEY не задан — возвращает None для всех полей.
"""

from __future__ import annotations
import asyncio
import httpx
import json
import logging
from typing import Optional
from config import settings

log = logging.getLogger(__name__)

_CLAUDE_URL = "https://api.anthropic.com/v1/messages"
_API_VER    = "2023-06-01"
_SYS_PROMPT = (
    "Ты — эксперт-аналитик встреч и вебинаров. "
    "Анализируй транскрипцию строго по теме. "
    "Отвечай на том же языке, что транскрипция. "
    "Без вводных фраз типа 'Вот анализ:' или 'Конечно!'."
)


async def _call_claude(prompt: str) -> Optional[str]:
    if not settings.anthropic_api_key:
        return None

    payload = {
        "model":      settings.anthropic_model,
        "max_tokens": 4096,
        "system":     _SYS_PROMPT,
        "messages":   [{"role": "user", "content": prompt}],
    }
    headers = {
        "x-api-key":          settings.anthropic_api_key,
        "anthropic-version":  _API_VER,
        "content-type":       "application/json",
    }

    async with httpx.AsyncClient(timeout=300) as client:
        resp = await client.post(_CLAUDE_URL, json=payload, headers=headers)
        if resp.status_code != 200:
            log.error("Claude API error %d: %s", resp.status_code, resp.text[:300])
            return None
        data = resp.json()
        return data["content"][0]["text"]


async def generate_all(transcript: str) -> dict:
    """
    Параллельно генерирует summary / highlights / chapters / action_items.

    Returns:
        {"summary": str|None, "highlights": str|None, "chapters": str|None, "action_items": str|None}
    """
    if not settings.anthropic_api_key:
        log.warning("Summary: ANTHROPIC_API_KEY not set, skipping Claude analysis")
        return {"summary": None, "highlights": None, "chapters": None, "action_items": None}

    prompts = {
        "summary": (
            "Создай краткое и информативное резюме встречи (3-5 абзацев). "
            "Включи: главную тему, ключевые обсуждения, принятые решения, итоги.\n\n"
            f"ТРАНСКРИПЦИЯ:\n{transcript}"
        ),
        "highlights": (
            "Извлеки 5-10 ключевых моментов и инсайтов. "
            "Формат: нумерованный список. Каждый пункт — 1-2 предложения. "
            "Только реально важные моменты, факты, цифры, цитаты.\n\n"
            f"ТРАНСКРИПЦИЯ:\n{transcript}"
        ),
        "chapters": (
            "Разбей встречу на логические главы/разделы.\n"
            "Для каждой главы:\n"
            "## [Название]\n"
            "**Позиция:** ~X% от начала\n"
            "**Содержание:** 2-3 предложения о чём шла речь.\n\n"
            f"ТРАНСКРИПЦИЯ:\n{transcript}"
        ),
        "action_items": (
            "Извлеки все задачи, договорённости и дедлайны.\n"
            "Формат каждой задачи:\n"
            "- [ ] **Задача:** описание\n"
            "  **Ответственный:** имя или 'не указан'\n"
            "  **Срок:** дата или 'не указан'\n\n"
            "Если задач нет — напиши 'Явных задач не выявлено.'\n\n"
            f"ТРАНСКРИПЦИЯ:\n{transcript}"
        ),
    }

    tasks = {key: _call_claude(prompt) for key, prompt in prompts.items()}
    results = await asyncio.gather(*tasks.values(), return_exceptions=True)

    output = {}
    for key, result in zip(tasks.keys(), results):
        if isinstance(result, Exception):
            log.error("Summary[%s] failed: %s", key, result)
            output[key] = None
        else:
            output[key] = result

    return output
