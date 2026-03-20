-- Migration 009: add event details required for:
--   · RF 152-ФЗ ст.9  — субъект должен знать цель обработки (что за мероприятие, когда, где)
--   · GDPR Art.13      — transparency: controller must disclose what data is collected and for what purpose
--   · CCPA §1798.100   — right to know before collection
--   · capacity + waitlist — functional requirement for any real event platform

ALTER TABLE reg_events
    ADD COLUMN IF NOT EXISTS start_at          TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS venue             TEXT        DEFAULT '',
    ADD COLUMN IF NOT EXISTS address           TEXT        DEFAULT '',
    ADD COLUMN IF NOT EXISTS capacity          INT         DEFAULT 0,    -- 0 = unlimited
    ADD COLUMN IF NOT EXISTS cover_url         TEXT        DEFAULT '';

-- Index for sorting by start time on public event list
CREATE INDEX IF NOT EXISTS idx_reg_events_start_at ON reg_events(start_at);
