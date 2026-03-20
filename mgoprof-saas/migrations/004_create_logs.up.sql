CREATE TABLE reg_logs (
    id          SERIAL PRIMARY KEY,
    event_type  TEXT,
    user_email  TEXT,
    ip_address  TEXT,
    message     TEXT,
    user_agent  TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_logs_created_at ON reg_logs(created_at DESC);
CREATE INDEX idx_logs_user_email ON reg_logs(user_email);
CREATE INDEX idx_logs_event_type ON reg_logs(event_type);
