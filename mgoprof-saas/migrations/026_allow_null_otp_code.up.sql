-- Allow otp_code to be NULL so SetVerified can clear it after successful
-- OTP verification (defense-in-depth: even if status check is bypassed,
-- a NULL code blocks OTP replay attacks).
ALTER TABLE reg_registrations ALTER COLUMN otp_code DROP NOT NULL;
