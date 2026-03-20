DROP INDEX IF EXISTS idx_reg_participant_token;

ALTER TABLE reg_registrations
    DROP COLUMN IF EXISTS participant_token,
    DROP COLUMN IF EXISTS checked_in_at;

ALTER TABLE reg_events
    DROP COLUMN IF EXISTS is_online;
