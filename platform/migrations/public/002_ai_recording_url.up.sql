-- ── AI Service: добавляем recording_url в rooms ──────────────────────────────

ALTER TABLE rooms ADD COLUMN IF NOT EXISTS recording_url TEXT;

-- Уникальное ограничение для UPSERT в ai_summaries (event_id, type)
-- ON CONFLICT (event_id, type) DO UPDATE требует уникального индекса
ALTER TABLE ai_summaries
  ADD CONSTRAINT ai_summaries_event_type_uq UNIQUE (event_id, type);
