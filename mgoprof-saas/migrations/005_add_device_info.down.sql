DROP INDEX IF EXISTS idx_reg_device_type;
DROP INDEX IF EXISTS idx_reg_os_name;

ALTER TABLE reg_registrations
    DROP COLUMN IF EXISTS isp_name,
    DROP COLUMN IF EXISTS isp_asn,
    DROP COLUMN IF EXISTS device_type,
    DROP COLUMN IF EXISTS os_name,
    DROP COLUMN IF EXISTS browser_name,
    DROP COLUMN IF EXISTS user_agent;
