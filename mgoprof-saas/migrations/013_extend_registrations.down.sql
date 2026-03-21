ALTER TABLE reg_registrations
    DROP COLUMN IF EXISTS status_extended,
    DROP COLUMN IF EXISTS first_entry_at,
    DROP COLUMN IF EXISTS last_exit_at,
    DROP COLUMN IF EXISTS participation_seconds,
    DROP COLUMN IF EXISTS scan_count,
    DROP COLUMN IF EXISTS public_uuid;
