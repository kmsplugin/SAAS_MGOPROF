"""
Punctuation restoration + text normalization.
Использует deepmultilingualpunctuation (поддерживает ru/en/de/fr/...).
"""

from __future__ import annotations
import logging
import re
from typing import Optional

log = logging.getLogger(__name__)

_model = None


def _load_model():
    global _model
    if _model is None:
        from deepmultilingualpunctuation import PunctuationModel
        log.info("Punctuation: loading model...")
        _model = PunctuationModel(model="kredor/punctuate-all")
        log.info("Punctuation: model loaded")
    return _model


def restore_punctuation(text: str) -> str:
    """
    Восстанавливает знаки препинания в тексте без пунктуации.
    Обрабатывает большие тексты блоками по 1000 слов.
    """
    if not text.strip():
        return text

    try:
        model = _load_model()
    except Exception as e:
        log.warning("Punctuation: model load failed, returning raw text: %s", e)
        return text

    # Разбиваем на блоки по 1000 слов (лимит модели)
    words = text.split()
    chunk_size = 1000
    chunks = [words[i:i + chunk_size] for i in range(0, len(words), chunk_size)]

    restored_chunks = []
    for chunk in chunks:
        chunk_text = " ".join(chunk)
        try:
            result = model.restore_punctuation(chunk_text)
            restored_chunks.append(result)
        except Exception as e:
            log.warning("Punctuation: chunk failed: %s", e)
            restored_chunks.append(chunk_text)

    return " ".join(restored_chunks)


def normalize_text(text: str) -> str:
    """
    Базовая нормализация текста:
    - убирает лишние пробелы
    - исправляет пробелы вокруг знаков препинания
    - капитализирует начало предложений
    """
    # Убираем множественные пробелы
    text = re.sub(r" {2,}", " ", text)

    # Пробел перед знаком препинания убираем
    text = re.sub(r" ([,\.\!\?\;\:])", r"\1", text)

    # После точки/! / ? — пробел и заглавная буква
    text = re.sub(r"([\.!\?])\s*([а-яёa-z])", lambda m: m.group(1) + " " + m.group(2).upper(), text)

    # Первая буква
    text = text[:1].upper() + text[1:] if text else text

    return text.strip()
