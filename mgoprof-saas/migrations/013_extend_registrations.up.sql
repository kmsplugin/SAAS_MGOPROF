-- Migration 013: extend registrations with detailed participant tracking fields.
-- Adds: extended status FSM, entry/exit timestamps, participation duration,
-- scan counter, and a public UUID for external references (QR deeplinks).

ALTER TABLE reg_registrations
    ADD COLUMN IF NOT EXISTS status_extended      TEXT        DEFAULT 'registered'
        CHECK (status_extended IN (
            'registered','confirmed','qr_issued','arrived',
            'entered','exited','participated','no_show','cancelled','duplicate'
        )),
    ADD COLUMN IF NOT EXISTS first_entry_at       TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS last_exit_at         TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS participation_seconds INTEGER     DEFAULT 0,
    ADD COLUMN IF NOT EXISTS scan_count           INTEGER     NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS public_uuid          UUID        UNIQUE DEFAULT gen_random_uuid();

-- Set initial extended status from existing binary status
UPDATE reg_registrations
    SET status_extended = CASE
        WHEN status = 'verified'   THEN 'confirmed'
        WHEN status = 'cancelled'  THEN 'cancelled'
        ELSE 'registered'
    END
WHERE status_extended = 'registered' OR status_extended IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_reg_public_uuid
    ON reg_registrations(public_uuid)
    WHERE public_uuid IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_reg_status_extended
    ON reg_registrations(status_extended);

CREATE INDEX IF NOT EXISTS idx_reg_first_entry_at
    ON reg_registrations(first_entry_at DESC)
    WHERE first_entry_at IS NOT NULL;
