-- Migration 015: extend event_fields with new types, validation, badge/report flags,
-- and a link to ref_lists for select/multiselect dropdowns.

ALTER TABLE event_fields
    -- Expand allowed field types
    DROP CONSTRAINT IF EXISTS event_fields_field_type_check;

ALTER TABLE event_fields
    ADD CONSTRAINT event_fields_field_type_check
    CHECK (field_type IN (
        'text', 'textarea', 'select', 'multiselect', 'checkbox', 'radio',
        'phone', 'number', 'date', 'hidden', 'masked', 'file'
    ));

ALTER TABLE event_fields
    ADD COLUMN IF NOT EXISTS helper_text      TEXT,
    ADD COLUMN IF NOT EXISTS validation_regex TEXT,        -- optional JS-compatible regex
    ADD COLUMN IF NOT EXISTS in_badge         BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS in_report        BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN IF NOT EXISTS in_export        BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN IF NOT EXISTS list_id          INTEGER REFERENCES ref_lists(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS min_value        NUMERIC,     -- for 'number' type
    ADD COLUMN IF NOT EXISTS max_value        NUMERIC,     -- for 'number' type
    ADD COLUMN IF NOT EXISTS max_length       INTEGER;     -- for text/textarea

CREATE INDEX IF NOT EXISTS idx_fields_list_id ON event_fields(list_id)
    WHERE list_id IS NOT NULL;
