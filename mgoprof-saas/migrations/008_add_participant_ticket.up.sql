-- Each offline participant gets a unique scannable token and check-in timestamp.
-- Events get an is_online flag so the system knows when to show QR tickets.

-- Add to registrations
ALTER TABLE reg_registrations
    ADD COLUMN IF NOT EXISTS participant_token TEXT UNIQUE,
    ADD COLUMN IF NOT EXISTS checked_in_at     TIMESTAMPTZ;

CREATE UNIQUE INDEX IF NOT EXISTS idx_reg_participant_token ON reg_registrations(participant_token)
    WHERE participant_token IS NOT NULL;

-- Add to events
ALTER TABLE reg_events
    ADD COLUMN IF NOT EXISTS is_online BOOLEAN NOT NULL DEFAULT TRUE;
