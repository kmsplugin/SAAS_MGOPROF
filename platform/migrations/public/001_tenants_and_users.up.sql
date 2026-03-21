-- ── Extension ──────────────────────────────────────────────────────────────
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ── Tenants ──────────────────────────────────────────────────────────────────

CREATE TABLE tenants (
  id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  slug            TEXT        NOT NULL UNIQUE,
  name            TEXT        NOT NULL,
  plan            TEXT        NOT NULL DEFAULT 'starter' CHECK (plan IN ('starter','pro','enterprise')),
  logo_url        TEXT,
  primary_color   VARCHAR(7)  DEFAULT '#22c55e',
  custom_domain   TEXT,
  is_active       BOOLEAN     NOT NULL DEFAULT TRUE,
  -- Feature flags (tenant-level)
  ai_enabled           BOOLEAN NOT NULL DEFAULT FALSE,
  recording_enabled    BOOLEAN NOT NULL DEFAULT FALSE,
  live_stream_enabled  BOOLEAN NOT NULL DEFAULT FALSE,
  custom_branding      BOOLEAN NOT NULL DEFAULT FALSE,
  white_label          BOOLEAN NOT NULL DEFAULT FALSE,
  advanced_analytics   BOOLEAN NOT NULL DEFAULT FALSE,
  max_participants     INT     NOT NULL DEFAULT 100,
  max_viewers          INT     NOT NULL DEFAULT 1000,
  -- Billing
  stripe_customer_id   TEXT,
  trial_ends_at        TIMESTAMPTZ,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at      TIMESTAMPTZ
);

-- ── Users ─────────────────────────────────────────────────────────────────────

CREATE TABLE users (
  id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id     UUID        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  email         TEXT        NOT NULL,
  email_verified BOOLEAN    NOT NULL DEFAULT FALSE,
  first_name    TEXT        NOT NULL DEFAULT '',
  last_name     TEXT        NOT NULL DEFAULT '',
  avatar_url    TEXT,
  password_hash TEXT        NOT NULL DEFAULT '',
  role          TEXT        NOT NULL DEFAULT 'participant'
                            CHECK (role IN ('super_admin','tenant_owner','event_admin',
                                            'moderator','speaker','support','analyst',
                                            'viewer','participant','guest')),
  is_active     BOOLEAN     NOT NULL DEFAULT TRUE,
  last_login_at TIMESTAMPTZ,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at    TIMESTAMPTZ,
  UNIQUE (tenant_id, email)
);

CREATE INDEX idx_users_tenant_email ON users (tenant_id, lower(email));
CREATE INDEX idx_users_tenant_role  ON users (tenant_id, role);

-- ── Consent / GDPR ───────────────────────────────────────────────────────────

CREATE TABLE user_consents (
  id            BIGSERIAL   PRIMARY KEY,
  user_id       UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  tenant_id     UUID        NOT NULL,
  consent_type  TEXT        NOT NULL, -- 'registration' | 'marketing' | 'analytics'
  consent_text  TEXT        NOT NULL,
  version       TEXT        NOT NULL DEFAULT '1.0',
  ip            INET,
  user_agent    TEXT,
  consented_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  withdrawn_at  TIMESTAMPTZ
);

CREATE INDEX idx_user_consents_user ON user_consents (user_id);

-- ── Events ───────────────────────────────────────────────────────────────────

CREATE TABLE events (
  id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id           UUID        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  title               TEXT        NOT NULL,
  description         TEXT        NOT NULL DEFAULT '',
  type                TEXT        NOT NULL DEFAULT 'webinar'
                                  CHECK (type IN ('webinar','conference','broadcast','hybrid','meeting')),
  status              TEXT        NOT NULL DEFAULT 'draft'
                                  CHECK (status IN ('draft','published','live','ended','archived')),
  start_at            TIMESTAMPTZ,
  end_at              TIMESTAMPTZ,
  timezone            TEXT        NOT NULL DEFAULT 'UTC',
  cover_url           TEXT,
  capacity            INT         NOT NULL DEFAULT 0, -- 0 = unlimited for participants
  viewer_capacity     INT         NOT NULL DEFAULT 10000,
  is_public           BOOLEAN     NOT NULL DEFAULT TRUE,
  registration_required BOOLEAN   NOT NULL DEFAULT TRUE,
  created_by          UUID        REFERENCES users(id),
  created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at          TIMESTAMPTZ
);

CREATE INDEX idx_events_tenant        ON events (tenant_id);
CREATE INDEX idx_events_tenant_status ON events (tenant_id, status);
CREATE INDEX idx_events_start_at      ON events (start_at);

-- ── Rooms ─────────────────────────────────────────────────────────────────────

CREATE TABLE rooms (
  id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  event_id            UUID        NOT NULL REFERENCES events(id) ON DELETE CASCADE,
  tenant_id           UUID        NOT NULL,
  livekit_room_name   TEXT        NOT NULL UNIQUE,
  mode                TEXT        NOT NULL DEFAULT 'webinar'
                                  CHECK (mode IN ('meeting','webinar','broadcast','stage')),
  status              TEXT        NOT NULL DEFAULT 'pending'
                                  CHECK (status IN ('pending','active','ended')),
  max_participants    INT         NOT NULL DEFAULT 100,
  is_recording        BOOLEAN     NOT NULL DEFAULT FALSE,
  hls_url             TEXT,
  rtmp_url            TEXT,
  started_at          TIMESTAMPTZ,
  ended_at            TIMESTAMPTZ,
  created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_rooms_event ON rooms (event_id);

-- ── Registrations ─────────────────────────────────────────────────────────────

CREATE TABLE registrations (
  id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  event_id          UUID        NOT NULL REFERENCES events(id) ON DELETE CASCADE,
  user_id           UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  tenant_id         UUID        NOT NULL,
  status            TEXT        NOT NULL DEFAULT 'pending'
                                CHECK (status IN ('pending','confirmed','cancelled','attended')),
  ticket_code       TEXT        NOT NULL UNIQUE DEFAULT gen_random_uuid()::TEXT,
  check_in_at       TIMESTAMPTZ,
  consent_given     BOOLEAN     NOT NULL DEFAULT FALSE,
  consent_version   TEXT        NOT NULL DEFAULT '1.0',
  custom_fields     JSONB       NOT NULL DEFAULT '{}',
  created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (event_id, user_id)
);

CREATE INDEX idx_registrations_event  ON registrations (event_id);
CREATE INDEX idx_registrations_user   ON registrations (user_id);
CREATE INDEX idx_registrations_status ON registrations (event_id, status);

-- ── Q&A ───────────────────────────────────────────────────────────────────────

CREATE TYPE question_status AS ENUM ('new','in_progress','answered','closed');

CREATE TABLE questions (
  id            UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
  event_id      UUID            NOT NULL REFERENCES events(id) ON DELETE CASCADE,
  user_id       UUID            NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  tenant_id     UUID            NOT NULL,
  subject       TEXT            NOT NULL,
  status        question_status NOT NULL DEFAULT 'new',
  is_public     BOOLEAN         NOT NULL DEFAULT FALSE,
  priority      INT             NOT NULL DEFAULT 0,
  assigned_to   UUID            REFERENCES users(id),
  first_reply_at TIMESTAMPTZ,
  created_at    TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
  updated_at    TIMESTAMPTZ,
  closed_at     TIMESTAMPTZ
);

CREATE INDEX idx_questions_event  ON questions (event_id);
CREATE INDEX idx_questions_tenant ON questions (tenant_id, status);

CREATE TABLE question_messages (
  id          BIGSERIAL   PRIMARY KEY,
  question_id UUID        NOT NULL REFERENCES questions(id) ON DELETE CASCADE,
  sender_id   UUID        NOT NULL REFERENCES users(id),
  sender_role TEXT        NOT NULL CHECK (sender_role IN ('user','admin','moderator')),
  body        TEXT        NOT NULL,
  is_read     BOOLEAN     NOT NULL DEFAULT FALSE,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_qmessages_question ON question_messages (question_id);

-- ── Audit log ─────────────────────────────────────────────────────────────────

CREATE TABLE audit_logs (
  id          BIGSERIAL   PRIMARY KEY,
  tenant_id   UUID,
  user_id     UUID,
  user_email  TEXT        NOT NULL DEFAULT '',
  action      TEXT        NOT NULL,
  target_type TEXT        NOT NULL DEFAULT '',
  target_id   TEXT,
  old_value   JSONB,
  new_value   JSONB,
  ip          INET,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
) PARTITION BY RANGE (created_at);

-- Monthly partitions (add more as needed)
CREATE TABLE audit_logs_2026_01 PARTITION OF audit_logs
  FOR VALUES FROM ('2026-01-01') TO ('2026-02-01');
CREATE TABLE audit_logs_2026_02 PARTITION OF audit_logs
  FOR VALUES FROM ('2026-02-01') TO ('2026-03-01');
CREATE TABLE audit_logs_2026_03 PARTITION OF audit_logs
  FOR VALUES FROM ('2026-03-01') TO ('2026-04-01');
CREATE TABLE audit_logs_2026_04 PARTITION OF audit_logs
  FOR VALUES FROM ('2026-04-01') TO ('2026-05-01');
CREATE TABLE audit_logs_2026_05 PARTITION OF audit_logs
  FOR VALUES FROM ('2026-05-01') TO ('2026-06-01');
CREATE TABLE audit_logs_2026_06 PARTITION OF audit_logs
  FOR VALUES FROM ('2026-06-01') TO ('2026-07-01');
CREATE TABLE audit_logs_2026_07 PARTITION OF audit_logs
  FOR VALUES FROM ('2026-07-01') TO ('2026-08-01');
CREATE TABLE audit_logs_2026_08 PARTITION OF audit_logs
  FOR VALUES FROM ('2026-08-01') TO ('2026-09-01');
CREATE TABLE audit_logs_2026_09 PARTITION OF audit_logs
  FOR VALUES FROM ('2026-09-01') TO ('2026-10-01');
CREATE TABLE audit_logs_2026_10 PARTITION OF audit_logs
  FOR VALUES FROM ('2026-10-01') TO ('2026-11-01');
CREATE TABLE audit_logs_2026_11 PARTITION OF audit_logs
  FOR VALUES FROM ('2026-11-01') TO ('2026-12-01');
CREATE TABLE audit_logs_2026_12 PARTITION OF audit_logs
  FOR VALUES FROM ('2026-12-01') TO ('2027-01-01');

CREATE INDEX idx_audit_tenant ON audit_logs (tenant_id, created_at DESC);
CREATE INDEX idx_audit_user   ON audit_logs (user_id, created_at DESC);

-- ── AI summaries ──────────────────────────────────────────────────────────────

CREATE TABLE ai_summaries (
  id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  event_id        UUID        NOT NULL REFERENCES events(id) ON DELETE CASCADE,
  tenant_id       UUID        NOT NULL,
  type            TEXT        NOT NULL CHECK (type IN ('transcript','summary','highlights','chapters','action_items')),
  content         TEXT        NOT NULL,
  language        VARCHAR(10) NOT NULL DEFAULT 'ru',
  model_version   TEXT        NOT NULL DEFAULT '',
  generated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_ai_summaries_event ON ai_summaries (event_id);
