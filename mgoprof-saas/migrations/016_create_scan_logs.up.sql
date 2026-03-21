-- Migration 016: scan_logs — immutable audit log of every QR scan attempt.
-- Every scan (entry, exit, verify, duplicate, error) is recorded here.

CREATE TABLE IF NOT EXISTS scan_logs (
    id              BIGSERIAL   PRIMARY KEY,
    event_id        INTEGER     NOT NULL REFERENCES reg_events(id) ON DELETE CASCADE,
    registration_id INTEGER     REFERENCES reg_registrations(id)  ON DELETE SET NULL,
    scanned_token   TEXT        NOT NULL,  -- raw token from QR
    scan_mode       TEXT        NOT NULL
        CHECK (scan_mode IN ('entry', 'exit', 'verify')),
    scan_result     TEXT        NOT NULL
        CHECK (scan_result IN ('ok', 'duplicate', 'not_found', 'wrong_event', 'cancelled', 'error')),
    operator_id     INTEGER     REFERENCES reg_admins(id) ON DELETE SET NULL,
    device_info     TEXT        DEFAULT '',  -- e.g. "iPhone 14 / Chrome 121"
    ip_address      TEXT,
    note            TEXT,
    scanned_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_scan_logs_event       ON scan_logs(event_id, scanned_at DESC);
CREATE INDEX IF NOT EXISTS idx_scan_logs_registration ON scan_logs(registration_id)
    WHERE registration_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_scan_logs_token        ON scan_logs(scanned_token);
CREATE INDEX IF NOT EXISTS idx_scan_logs_operator     ON scan_logs(operator_id, scanned_at DESC)
    WHERE operator_id IS NOT NULL;
