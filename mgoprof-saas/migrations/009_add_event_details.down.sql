DROP INDEX IF EXISTS idx_reg_events_start_at;

ALTER TABLE reg_events
    DROP COLUMN IF EXISTS start_at,
    DROP COLUMN IF EXISTS venue,
    DROP COLUMN IF EXISTS address,
    DROP COLUMN IF EXISTS capacity,
    DROP COLUMN IF EXISTS cover_url;
