-- AI summaries: stores transcript + all AI-generated content per event
CREATE TABLE IF NOT EXISTS ai_summaries (
    id            UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id      UUID         NOT NULL,
    tenant_id     UUID         NOT NULL,
    type          VARCHAR(50)  NOT NULL, -- transcript|summary|highlights|chapters|action_items
    content       TEXT         NOT NULL,
    language      VARCHAR(10)  NOT NULL DEFAULT 'ru',
    model_version VARCHAR(100) NOT NULL DEFAULT '',
    generated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    UNIQUE (event_id, type)
);

CREATE INDEX IF NOT EXISTS idx_ai_summaries_tenant_event
    ON ai_summaries (tenant_id, event_id);

-- AI jobs: persistent queue for recording → AI pipeline
-- Survives service restarts; prevents duplicate processing
CREATE TABLE IF NOT EXISTS ai_jobs (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id     UUID        NOT NULL,
    tenant_id    UUID        NOT NULL,
    audio_url    TEXT        NOT NULL,
    language     VARCHAR(10) NOT NULL DEFAULT 'ru',
    recording_id VARCHAR(255) NOT NULL DEFAULT '',
    status       VARCHAR(20) NOT NULL DEFAULT 'queued', -- queued|processing|done|failed
    error_msg    TEXT        NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ai_jobs_tenant_event
    ON ai_jobs (tenant_id, event_id);
CREATE INDEX IF NOT EXISTS idx_ai_jobs_active
    ON ai_jobs (status) WHERE status IN ('queued', 'processing');
