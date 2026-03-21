-- Migration 021: online session tracking for virtual events.
-- Replaces loose tracking rows with structured sessions that have a clear
-- start/end/duration so admin can see actual participation time.

CREATE TABLE IF NOT EXISTS online_sessions (
    id              BIGSERIAL    PRIMARY KEY,
    event_id        INTEGER      NOT NULL REFERENCES reg_events(id) ON DELETE CASCADE,
    user_id         INTEGER      NOT NULL REFERENCES reg_users(id)  ON DELETE CASCADE,
    registration_id INTEGER      REFERENCES reg_registrations(id)   ON DELETE SET NULL,
    session_uuid    UUID         NOT NULL DEFAULT gen_random_uuid() UNIQUE,
    started_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    last_ping_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    ended_at        TIMESTAMPTZ,
    duration_seconds INTEGER,    -- filled on explicit disconnect or timeout job
    end_reason      TEXT         CHECK (end_reason IN ('explicit', 'timeout', 'error'))
);

CREATE INDEX IF NOT EXISTS idx_online_sessions_event
    ON online_sessions(event_id, started_at DESC);

CREATE INDEX IF NOT EXISTS idx_online_sessions_user
    ON online_sessions(user_id);

CREATE INDEX IF NOT EXISTS idx_online_sessions_reg
    ON online_sessions(registration_id)
    WHERE registration_id IS NOT NULL;

-- Stale-session cleanup: sessions with last_ping_at older than 90 seconds are
-- marked as timed-out. Run periodically (e.g. every minute via pg_cron or app cron).
-- A helper view makes it easy for the admin API to query.
CREATE OR REPLACE VIEW online_session_summary AS
SELECT
    os.event_id,
    os.user_id,
    os.registration_id,
    COUNT(*)                                    AS session_count,
    MIN(os.started_at)                          AS first_join_at,
    MAX(COALESCE(os.ended_at, os.last_ping_at)) AS last_seen_at,
    COALESCE(SUM(os.duration_seconds), 0)       AS total_seconds
FROM online_sessions os
GROUP BY os.event_id, os.user_id, os.registration_id;
