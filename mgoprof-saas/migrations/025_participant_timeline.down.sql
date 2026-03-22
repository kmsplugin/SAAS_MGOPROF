ALTER TABLE reg_registrations
    DROP COLUMN IF EXISTS welcome_email_sent_at,
    DROP COLUMN IF EXISTS otp_send_count;

ALTER TABLE reg_users
    DROP COLUMN IF EXISTS cabinet_first_login_at,
    DROP COLUMN IF EXISTS cabinet_last_login_at;
