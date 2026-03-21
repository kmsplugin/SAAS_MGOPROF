-- Migration 023: participant role per registration.
-- Allows the system to provide different event links depending on role.

ALTER TABLE reg_registrations
    ADD COLUMN IF NOT EXISTS participant_role TEXT NOT NULL DEFAULT 'delegate'
        CHECK (participant_role IN ('delegate', 'speaker', 'moderator', 'observer', 'vip'));

CREATE INDEX IF NOT EXISTS idx_reg_registrations_role
    ON reg_registrations(event_id, participant_role)
    WHERE participant_role != 'delegate';

COMMENT ON COLUMN reg_registrations.participant_role IS
    'Role determines which event link (speaker_link vs viewer_link) is shown in cabinet.';
