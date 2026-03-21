-- Migration 019: registration_status_log — immutable FSM transition log.
-- Records every status change for a registration with who made the change and why.

CREATE TABLE IF NOT EXISTS registration_status_log (
    id              BIGSERIAL   PRIMARY KEY,
    registration_id INTEGER     NOT NULL REFERENCES reg_registrations(id) ON DELETE CASCADE,
    event_id        INTEGER     NOT NULL REFERENCES reg_events(id)        ON DELETE CASCADE,
    prev_status     TEXT        NOT NULL,
    new_status      TEXT        NOT NULL,
    changed_by      INTEGER     REFERENCES reg_admins(id) ON DELETE SET NULL,  -- NULL = system
    change_source   TEXT        NOT NULL DEFAULT 'system'
        CHECK (change_source IN ('system', 'admin', 'scanner', 'api', 'user')),
    note            TEXT,
    changed_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_status_log_reg
    ON registration_status_log(registration_id, changed_at DESC);
CREATE INDEX IF NOT EXISTS idx_status_log_event
    ON registration_status_log(event_id, changed_at DESC);
