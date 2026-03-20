CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE reg_events (
    id          SERIAL PRIMARY KEY,
    title       TEXT NOT NULL,
    description TEXT,
    event_date  DATE NOT NULL,
    event_time  TIME NOT NULL,
    cabinet_link TEXT,
    is_active   BOOLEAN NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ
);

CREATE INDEX idx_events_is_active ON reg_events(is_active);
CREATE INDEX idx_events_date ON reg_events(event_date DESC, event_time DESC);
