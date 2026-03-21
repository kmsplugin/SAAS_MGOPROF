-- Migration 018: badge_templates — per-event badge design stored in DB.
-- Stores HTML/CSS template + configuration (accent color, logo, background).

CREATE TABLE IF NOT EXISTS badge_templates (
    id              SERIAL      PRIMARY KEY,
    event_id        INTEGER     REFERENCES reg_events(id) ON DELETE SET NULL,
    name            TEXT        NOT NULL,         -- template name, e.g. "Standard A6"
    is_default      BOOLEAN     NOT NULL DEFAULT FALSE,
    -- Design settings (JSON for flexibility)
    accent_color    TEXT        NOT NULL DEFAULT '#1a56db',
    logo_url        TEXT        DEFAULT '',
    background_url  TEXT        DEFAULT '',
    paper_size      TEXT        NOT NULL DEFAULT 'A6'
        CHECK (paper_size IN ('A4', 'A5', 'A6', '90x60', '105x74')),
    orientation     TEXT        NOT NULL DEFAULT 'landscape'
        CHECK (orientation IN ('landscape', 'portrait')),
    -- HTML template with {{placeholders}}
    html_template   TEXT        NOT NULL DEFAULT '',
    -- Which fields to show on badge (jsonb array of field names / custom field IDs)
    fields_config   JSONB       NOT NULL DEFAULT '["last_name","first_name","organization","district","qr_code"]',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_badge_tpl_event
    ON badge_templates(event_id)
    WHERE event_id IS NOT NULL;
