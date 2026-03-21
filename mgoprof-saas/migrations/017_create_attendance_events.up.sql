-- Migration 017: attendance_events — lightweight event stream for entry/exit tracking.
-- Each row represents a single entry or exit action for a participant.
-- Aggregate queries over this table produce total presence time.

CREATE TABLE IF NOT EXISTS attendance_events (
    id              BIGSERIAL   PRIMARY KEY,
    event_id        INTEGER     NOT NULL REFERENCES reg_events(id)          ON DELETE CASCADE,
    registration_id INTEGER     NOT NULL REFERENCES reg_registrations(id)   ON DELETE CASCADE,
    action          TEXT        NOT NULL CHECK (action IN ('entry', 'exit')),
    scan_log_id     BIGINT      REFERENCES scan_logs(id) ON DELETE SET NULL,
    occurred_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_att_event_reg
    ON attendance_events(event_id, registration_id, occurred_at);
CREATE INDEX IF NOT EXISTS idx_att_occurred_at
    ON attendance_events(occurred_at DESC);
