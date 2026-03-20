-- Add ISP, ASN, and device-type columns to reg_registrations
ALTER TABLE reg_registrations
    ADD COLUMN IF NOT EXISTS isp_name    TEXT,
    ADD COLUMN IF NOT EXISTS isp_asn     TEXT,
    ADD COLUMN IF NOT EXISTS device_type TEXT,   -- mobile | tablet | desktop | bot | unknown
    ADD COLUMN IF NOT EXISTS os_name     TEXT,   -- Windows | macOS | Linux | Android | iOS | …
    ADD COLUMN IF NOT EXISTS browser_name TEXT,  -- Chrome | Firefox | Safari | Edge | …
    ADD COLUMN IF NOT EXISTS user_agent  TEXT;

-- Convenience indexes for analytics queries
CREATE INDEX IF NOT EXISTS idx_reg_device_type ON reg_registrations(device_type);
CREATE INDEX IF NOT EXISTS idx_reg_os_name     ON reg_registrations(os_name);
