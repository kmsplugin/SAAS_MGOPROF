-- Tracks link clicks, stream connections, and disconnections per participant.
CREATE TABLE reg_tracking (
    id              SERIAL PRIMARY KEY,
    event_id        INTEGER NOT NULL REFERENCES reg_events(id)        ON DELETE CASCADE,
    user_id         INTEGER NOT NULL REFERENCES reg_users(id)         ON DELETE CASCADE,
    registration_id INTEGER          REFERENCES reg_registrations(id) ON DELETE SET NULL,
    action          TEXT    NOT NULL,   -- visit | stream_connect | stream_disconnect | stream_error
    ip_address      TEXT,
    geo_country     TEXT,
    geo_region      TEXT,
    geo_city        TEXT,
    isp_name        TEXT,
    isp_asn         TEXT,
    device_type     TEXT,
    os_name         TEXT,
    browser_name    TEXT,
    user_agent      TEXT,
    occurred_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_tracking_event   ON reg_tracking(event_id);
CREATE INDEX idx_tracking_user    ON reg_tracking(user_id);
CREATE INDEX idx_tracking_action  ON reg_tracking(action);
CREATE INDEX idx_tracking_time    ON reg_tracking(occurred_at DESC);
