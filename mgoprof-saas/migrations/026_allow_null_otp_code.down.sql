-- Revert: set otp_code back to NOT NULL (fill NULLs with empty string first).
UPDATE reg_registrations SET otp_code = '' WHERE otp_code IS NULL;
ALTER TABLE reg_registrations ALTER COLUMN otp_code SET NOT NULL;
