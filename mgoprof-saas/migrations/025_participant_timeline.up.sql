-- Migration 025: extend participant timeline
-- Adds welcome_email_sent_at + otp_send_count to registrations,
-- and cabinet_first_login_at + cabinet_last_login_at to users.

ALTER TABLE reg_registrations
    ADD COLUMN welcome_email_sent_at TIMESTAMPTZ,
    ADD COLUMN otp_send_count        INT NOT NULL DEFAULT 0;

-- Backfill: treat existing verified rows as having had the welcome email sent
-- at otp_verified_at time (best approximation we can make without real data).
UPDATE reg_registrations
   SET welcome_email_sent_at = otp_verified_at
 WHERE status = 'verified' AND otp_verified_at IS NOT NULL;

-- Backfill otp_send_count = 1 for all rows that have an OTP expiry,
-- i.e. at least one OTP was ever sent.
UPDATE reg_registrations
   SET otp_send_count = 1
 WHERE otp_expires_at IS NOT NULL;

ALTER TABLE reg_users
    ADD COLUMN cabinet_first_login_at TIMESTAMPTZ,
    ADD COLUMN cabinet_last_login_at  TIMESTAMPTZ;
