ALTER TABLE reg_events
    DROP COLUMN IF EXISTS event_type,
    DROP COLUMN IF EXISTS check_in_mode,
    DROP COLUMN IF EXISTS registration_opens_at,
    DROP COLUMN IF EXISTS registration_closes_at,
    DROP COLUMN IF EXISTS badge_template_id,
    DROP COLUMN IF EXISTS max_scans_per_ticket,
    DROP COLUMN IF EXISTS end_at;
