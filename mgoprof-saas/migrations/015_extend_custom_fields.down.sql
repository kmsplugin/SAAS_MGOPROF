ALTER TABLE event_fields
    DROP COLUMN IF EXISTS helper_text,
    DROP COLUMN IF EXISTS validation_regex,
    DROP COLUMN IF EXISTS in_badge,
    DROP COLUMN IF EXISTS in_report,
    DROP COLUMN IF EXISTS in_export,
    DROP COLUMN IF EXISTS list_id,
    DROP COLUMN IF EXISTS min_value,
    DROP COLUMN IF EXISTS max_value,
    DROP COLUMN IF EXISTS max_length;

ALTER TABLE event_fields
    DROP CONSTRAINT IF EXISTS event_fields_field_type_check;

ALTER TABLE event_fields
    ADD CONSTRAINT event_fields_field_type_check
    CHECK (field_type IN ('text','textarea','select','checkbox','radio'));
