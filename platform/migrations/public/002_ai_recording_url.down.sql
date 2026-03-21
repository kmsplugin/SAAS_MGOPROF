ALTER TABLE ai_summaries DROP CONSTRAINT IF EXISTS ai_summaries_event_type_uq;
ALTER TABLE rooms DROP COLUMN IF EXISTS recording_url;
