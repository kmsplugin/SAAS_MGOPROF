-- Migration 012: Q&A module — user questions and threaded replies

CREATE TYPE question_status AS ENUM ('new','in_progress','answered','closed');

CREATE TABLE IF NOT EXISTS user_questions (
    id              SERIAL         PRIMARY KEY,
    user_id         INT            NOT NULL REFERENCES reg_users(id)  ON DELETE CASCADE,
    event_id        INT            NOT NULL REFERENCES reg_events(id) ON DELETE CASCADE,
    subject         VARCHAR(200)   NOT NULL,
    status          question_status NOT NULL DEFAULT 'new',
    priority        SMALLINT       NOT NULL DEFAULT 0, -- 0=normal, 1=high
    assigned_to     INT            REFERENCES admin_users(id) ON DELETE SET NULL,
    first_reply_at  TIMESTAMPTZ,   -- filled on first admin reply (SLA metric)
    created_at      TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ,
    closed_at       TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_user_questions_user    ON user_questions(user_id);
CREATE INDEX IF NOT EXISTS idx_user_questions_event   ON user_questions(event_id);
CREATE INDEX IF NOT EXISTS idx_user_questions_status  ON user_questions(status);
CREATE INDEX IF NOT EXISTS idx_user_questions_assigned ON user_questions(assigned_to)
    WHERE assigned_to IS NOT NULL;

-- Threaded messages per question
CREATE TABLE IF NOT EXISTS question_messages (
    id              SERIAL      PRIMARY KEY,
    question_id     INT         NOT NULL REFERENCES user_questions(id) ON DELETE CASCADE,
    sender_id       INT         NOT NULL, -- reg_users.id or admin_users.id
    sender_role     VARCHAR(20) NOT NULL, -- 'user' | 'admin'
    body            TEXT        NOT NULL,
    is_read         BOOL        NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT qm_sender_role_check CHECK (sender_role IN ('user','admin'))
);

CREATE INDEX IF NOT EXISTS idx_question_messages_question
    ON question_messages(question_id, created_at ASC);
CREATE INDEX IF NOT EXISTS idx_question_messages_unread
    ON question_messages(question_id, is_read)
    WHERE is_read = FALSE;

-- Status transition log
CREATE TABLE IF NOT EXISTS question_status_log (
    id          SERIAL         PRIMARY KEY,
    question_id INT            NOT NULL REFERENCES user_questions(id) ON DELETE CASCADE,
    old_status  question_status,
    new_status  question_status NOT NULL,
    changed_by  INT            REFERENCES admin_users(id) ON DELETE SET NULL,
    comment     TEXT,
    changed_at  TIMESTAMPTZ    NOT NULL DEFAULT NOW()
);

-- Email notification queue
CREATE TABLE IF NOT EXISTS question_notifications (
    id          SERIAL      PRIMARY KEY,
    question_id INT         NOT NULL REFERENCES user_questions(id) ON DELETE CASCADE,
    user_id     INT         NOT NULL REFERENCES reg_users(id) ON DELETE CASCADE,
    type        VARCHAR(50) NOT NULL,
    -- 'new_reply' | 'status_changed' | 'assigned' | 'closed'
    sent_at     TIMESTAMPTZ,        -- NULL = pending
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT qn_type_check CHECK (
        type IN ('new_reply','status_changed','assigned','closed')
    )
);

CREATE INDEX IF NOT EXISTS idx_question_notifications_pending
    ON question_notifications(sent_at, created_at)
    WHERE sent_at IS NULL;

-- Async export jobs table (questions Excel, registrations CSV, etc.)
CREATE TABLE IF NOT EXISTS export_jobs (
    id          SERIAL       PRIMARY KEY,
    admin_id    INT          REFERENCES admin_users(id) ON DELETE SET NULL,
    type        VARCHAR(50)  NOT NULL, -- 'registrations'|'questions'|'analytics'
    params      JSONB,                 -- filter params (event_id, date_from, ...)
    status      VARCHAR(20)  NOT NULL DEFAULT 'pending',
    -- pending | running | done | failed
    file_url    TEXT,                  -- presigned URL to generated file
    error       TEXT,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    done_at     TIMESTAMPTZ,
    expires_at  TIMESTAMPTZ,          -- file URL expires after 24h
    CONSTRAINT ej_status_check CHECK (status IN ('pending','running','done','failed'))
);

CREATE INDEX IF NOT EXISTS idx_export_jobs_admin
    ON export_jobs(admin_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_export_jobs_pending
    ON export_jobs(status, created_at)
    WHERE status = 'pending';
