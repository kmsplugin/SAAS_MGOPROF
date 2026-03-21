"""
Translation — перевод транскрипции.
Использует Helsinki-NLP/opus-mt модели через Hugging Face transformers.
Поддерживает ru↔en и другие пары без API.

Если язык источника и цель совпадают — возвращает исходный текст.
"""

from __future__ import annotations
import logging
from typing import Optional

log = logging.getLogger(__name__)

# Кэш пайплайнов: (src, tgt) → pipeline
_pipelines: dict[tuple[str, str], object] = {}


def _get_model_name(src: str, tgt: str) -> Optional[str]:
    """Возвращает имя модели Helsinki-NLP для пары языков."""
    # Прямые пары
    direct = {
        ("ru", "en"): "Helsinki-NLP/opus-mt-ru-en",
        ("en", "ru"): "Helsinki-NLP/opus-mt-en-ru",
        ("ru", "de"): "Helsinki-NLP/opus-mt-ru-de",
        ("de", "ru"): "Helsinki-NLP/opus-mt-de-ru",
        ("en", "de"): "Helsinki-NLP/opus-mt-en-de",
        ("de", "en"): "Helsinki-NLP/opus-mt-de-en",
        ("en", "fr"): "Helsinki-NLP/opus-mt-en-fr",
        ("fr", "en"): "Helsinki-NLP/opus-mt-fr-en",
        ("en", "es"): "Helsinki-NLP/opus-mt-en-es",
        ("es", "en"): "Helsinki-NLP/opus-mt-es-en",
        ("en", "zh"): "Helsinki-NLP/opus-mt-en-zh",
        ("zh", "en"): "Helsinki-NLP/opus-mt-zh-en",
    }
    return direct.get((src, tgt))


def _load_pipeline(src: str, tgt: str):
    key = (src, tgt)
    if key not in _pipelines:
        from transformers import pipeline as hf_pipeline
        model_name = _get_model_name(src, tgt)
        if not model_name:
            log.warning("Translation: no model for %s→%s", src, tgt)
            return None
        log.info("Translation: loading model %s", model_name)
        _pipelines[key] = hf_pipeline(
            "translation",
            model=model_name,
            device=-1,  # CPU; use 0 for GPU
        )
        log.info("Translation: model loaded for %s→%s", src, tgt)
    return _pipelines[key]


def translate(text: str, src_lang: str, tgt_lang: str, chunk_size: int = 400) -> str:
    """
    Переводит текст блоками (модели ограничены по длине ввода).

    Returns:
        Переведённый текст или исходный при ошибке/совпадении языков.
    """
    if not text.strip():
        return text
    if src_lang == tgt_lang:
        return text

    pipe = _load_pipeline(src_lang, tgt_lang)
    if pipe is None:
        log.warning("Translation: skipped %s→%s (no model)", src_lang, tgt_lang)
        return text

    # Разбиваем на предложения/блоки
    sentences = _split_into_chunks(text, chunk_size)
    translated_parts = []

    for chunk in sentences:
        try:
            result = pipe(chunk, max_length=512)
            translated_parts.append(result[0]["translation_text"])
        except Exception as e:
            log.error("Translation: chunk error: %s", e)
            translated_parts.append(chunk)

    return " ".join(translated_parts)


def _split_into_chunks(text: str, max_words: int) -> list[str]:
    """Разбивает текст на блоки по max_words слов, стараясь не резать предложения."""
    import re
    sentences = re.split(r"(?<=[.!?])\s+", text)
    chunks = []
    current = []
    current_len = 0

    for sent in sentences:
        words = sent.split()
        if current_len + len(words) > max_words and current:
            chunks.append(" ".join(current))
            current = words
            current_len = len(words)
        else:
            current.extend(words)
            current_len += len(words)

    if current:
        chunks.append(" ".join(current))

    return chunks if chunks else [text]
