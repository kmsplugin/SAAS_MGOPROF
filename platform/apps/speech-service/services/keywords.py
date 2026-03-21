"""
Keyword extraction — извлечение ключевых слов и фраз (KeyBERT).
Работает локально без API, поддерживает русский и английский.
"""

from __future__ import annotations
import logging
from typing import Optional

log = logging.getLogger(__name__)

_model = None


def _load_model():
    global _model
    if _model is None:
        from keybert import KeyBERT
        from sentence_transformers import SentenceTransformer

        log.info("Keywords: loading KeyBERT model (paraphrase-multilingual-MiniLM)...")
        # Многоязычная модель для ru/en
        st_model = SentenceTransformer("paraphrase-multilingual-MiniLM-L12-v2")
        _model = KeyBERT(model=st_model)
        log.info("Keywords: model loaded")
    return _model


def extract_keywords(
    text: str,
    top_n: int = 20,
    ngram_range: tuple[int, int] = (1, 3),
    diversity: float = 0.7,
) -> list[dict]:
    """
    Извлекает ключевые слова и фразы из текста.

    Returns:
        [{"keyword": str, "score": float}, ...]  — отсортировано по score DESC
    """
    if not text.strip():
        return []

    try:
        kw_model = _load_model()
    except Exception as e:
        log.warning("Keywords: model load failed: %s", e)
        return _fallback_keywords(text, top_n)

    try:
        keywords = kw_model.extract_keywords(
            text,
            keyphrase_ngram_range=ngram_range,
            stop_words=None,           # автоопределение языка
            use_mmr=True,              # Maximal Marginal Relevance для разнообразия
            diversity=diversity,
            top_n=top_n,
        )
        return [{"keyword": kw, "score": round(score, 4)} for kw, score in keywords]

    except Exception as e:
        log.error("Keywords: extraction failed: %s", e)
        return _fallback_keywords(text, top_n)


def _fallback_keywords(text: str, top_n: int) -> list[dict]:
    """
    Запасной вариант: TF-IDF на основе встроенных средств.
    Работает без GPU и тяжёлых моделей.
    """
    import re
    from collections import Counter
    from math import log as math_log

    words = re.findall(r"\b[а-яёa-zA-Z]{4,}\b", text.lower())
    freq = Counter(words)

    # Простой IDF-подобный score (чем реже слово в общем, тем интереснее)
    total = sum(freq.values())
    scored = {
        word: count / total * math_log(total / (count + 1) + 1)
        for word, count in freq.items()
    }

    top = sorted(scored.items(), key=lambda x: -x[1])[:top_n]
    return [{"keyword": kw, "score": round(score, 4)} for kw, score in top]
