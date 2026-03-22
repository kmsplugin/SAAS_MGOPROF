-- Track when OTP was sent so the admin timeline shows the full sequence:
-- registered → otp_sent → otp_verified → welcome_email_sent → cabinet_login
ALTER TABLE reg_registrations ADD COLUMN otp_sent_at TIMESTAMPTZ;

-- Backfill: for rows that already have an OTP expiry, approximate sent_at
-- as 10 minutes before expiry (the configured OTP TTL).
UPDATE reg_registrations
   SET otp_sent_at = otp_expires_at - INTERVAL '10 minutes'
 WHERE otp_expires_at IS NOT NULL AND otp_sent_at IS NULL;
