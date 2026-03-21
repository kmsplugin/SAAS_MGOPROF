-- Migration 020: extend reg_events with event type, check-in mode, registration window,
-- and badge template link.

ALTER TABLE reg_events
    ADD COLUMN IF NOT EXISTS event_type              TEXT    NOT NULL DEFAULT 'offline'
        CHECK (event_type IN ('online', 'offline', 'hybrid')),
    ADD COLUMN IF NOT EXISTS check_in_mode           TEXT    NOT NULL DEFAULT 'none'
        CHECK (check_in_mode IN ('none', 'entry_only', 'entry_exit')),
    ADD COLUMN IF NOT EXISTS registration_opens_at   TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS registration_closes_at  TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS badge_template_id       INTEGER REFERENCES badge_templates(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS max_scans_per_ticket    INTEGER NOT NULL DEFAULT 1,  -- 1 = single-entry
    ADD COLUMN IF NOT EXISTS end_at                  TIMESTAMPTZ;

-- Sync event_type with existing is_online flag
UPDATE reg_events
    SET event_type = CASE WHEN is_online THEN 'online' ELSE 'offline' END
WHERE event_type = 'offline';

CREATE INDEX IF NOT EXISTS idx_reg_events_type
    ON reg_events(event_type);

CREATE INDEX IF NOT EXISTS idx_reg_events_badge_tpl
    ON reg_events(badge_template_id)
    WHERE badge_template_id IS NOT NULL;
