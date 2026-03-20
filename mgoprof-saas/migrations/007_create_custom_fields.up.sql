-- Custom registration fields defined by admin per event.
CREATE TABLE event_fields (
    id          SERIAL PRIMARY KEY,
    event_id    INTEGER NOT NULL REFERENCES reg_events(id) ON DELETE CASCADE,
    label       TEXT    NOT NULL,
    field_type  TEXT    NOT NULL DEFAULT 'text'
                CHECK (field_type IN ('text','textarea','select','checkbox','radio')),
    options     JSONB,          -- For select/radio/checkbox: ["Option A","Option B",…]
    placeholder TEXT,
    is_required BOOLEAN NOT NULL DEFAULT FALSE,
    sort_order  INTEGER NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_fields_event ON event_fields(event_id, sort_order);

-- Answers submitted by registrants for custom fields.
CREATE TABLE reg_answers (
    id              SERIAL PRIMARY KEY,
    registration_id INTEGER NOT NULL REFERENCES reg_registrations(id) ON DELETE CASCADE,
    field_id        INTEGER NOT NULL REFERENCES event_fields(id)       ON DELETE CASCADE,
    value           TEXT    NOT NULL DEFAULT '',
    UNIQUE(registration_id, field_id)
);

CREATE INDEX idx_answers_reg   ON reg_answers(registration_id);
CREATE INDEX idx_answers_field ON reg_answers(field_id);
