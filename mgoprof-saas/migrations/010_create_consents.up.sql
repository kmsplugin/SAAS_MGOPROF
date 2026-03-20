-- Migration 010: personal data consent tracking
--
-- Legal basis:
--   · RF 152-ФЗ ст.9   — operator MUST obtain written (incl. electronic) consent
--                         and be able to prove it at any time (burden of proof on operator)
--   · GDPR Art.7        — controller must demonstrate consent was given; it must be
--                         as easy to withdraw as to give; records of consent required
--   · CCPA §1798.135    — businesses must disclose and obtain opt-in for sensitive data
--
-- Design decisions:
--   · consent_text stores the exact wording shown to the user at the time of consent
--     (GDPR Art.7(1) requires the controller to demonstrate *what* the user agreed to)
--   · version allows re-consent on policy update without losing old audit trail
--   · withdrawn_at tracks revocations (152-ФЗ ст.9 ч.2, GDPR Art.7(3))
--   · UNIQUE(user_id, event_id, version) prevents duplicate consent records

CREATE TABLE IF NOT EXISTS reg_consents (
    id              SERIAL PRIMARY KEY,
    user_id         INT          NOT NULL REFERENCES reg_users(id)  ON DELETE CASCADE,
    event_id        INT          NOT NULL REFERENCES reg_events(id) ON DELETE CASCADE,
    consent_text    TEXT         NOT NULL,
    version         VARCHAR(20)  NOT NULL DEFAULT '1.0',
    ip              VARCHAR(64),
    user_agent      TEXT,
    consented_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    withdrawn_at    TIMESTAMPTZ,          -- NULL = active consent
    UNIQUE(user_id, event_id, version)
);

CREATE INDEX IF NOT EXISTS idx_reg_consents_user  ON reg_consents(user_id);
CREATE INDEX IF NOT EXISTS idx_reg_consents_event ON reg_consents(event_id);

-- Deletion requests table (152-ФЗ ст.21, GDPR Art.17, CCPA §1798.105)
-- Tracks subject requests for data deletion; actual deletion is async.
CREATE TABLE IF NOT EXISTS reg_deletion_requests (
    id              SERIAL PRIMARY KEY,
    user_id         INT          NOT NULL REFERENCES reg_users(id) ON DELETE CASCADE,
    reason          TEXT,
    requested_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    processed_at    TIMESTAMPTZ,          -- NULL = pending
    processed_by    VARCHAR(200),         -- admin email or "auto"
    UNIQUE(user_id)
);

CREATE INDEX IF NOT EXISTS idx_reg_deletion_requests_pending
    ON reg_deletion_requests(processed_at)
    WHERE processed_at IS NULL;
