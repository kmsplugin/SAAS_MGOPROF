CREATE TABLE reg_registrations (
    id              SERIAL PRIMARY KEY,
    event_id        INTEGER NOT NULL REFERENCES reg_events(id) ON DELETE CASCADE,
    user_id         INTEGER NOT NULL REFERENCES reg_users(id) ON DELETE CASCADE,
    otp_code        TEXT NOT NULL,
    otp_expires_at  TIMESTAMPTZ NOT NULL,
    otp_verified_at TIMESTAMPTZ,
    status          TEXT NOT NULL DEFAULT 'pending'
                    CHECK (status IN ('pending', 'verified', 'cancelled')),
    ip_address      TEXT,
    geo_country     TEXT,
    geo_region      TEXT,
    geo_city        TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ,
    UNIQUE(event_id, user_id)
);

CREATE INDEX idx_reg_event_user  ON reg_registrations(event_id, user_id);
CREATE INDEX idx_reg_status      ON reg_registrations(status);
CREATE INDEX idx_reg_created_at  ON reg_registrations(created_at DESC);
