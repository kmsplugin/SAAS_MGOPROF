-- Migration 011: admin_users — administrators separated from reg_users
-- Each admin has a role; super_admin is seeded via env, not stored as a row.
CREATE TABLE IF NOT EXISTS admin_users (
    id                SERIAL PRIMARY KEY,
    email             VARCHAR(255) NOT NULL UNIQUE,
    name              VARCHAR(200) NOT NULL,
    password_hash     TEXT         NOT NULL,
    role              VARCHAR(50)  NOT NULL DEFAULT 'support',
    -- Possible values: super_admin | admin | operator | support | viewer
    is_active         BOOL         NOT NULL DEFAULT TRUE,
    created_by        INT          REFERENCES admin_users(id) ON DELETE SET NULL,
    last_login_at     TIMESTAMPTZ,
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ,
    CONSTRAINT admin_users_role_check CHECK (
        role IN ('super_admin','admin','operator','support','viewer')
    )
);

-- Granular permissions catalogue (≤ 30 keys)
CREATE TABLE IF NOT EXISTS permissions (
    id          SERIAL PRIMARY KEY,
    key         VARCHAR(100) NOT NULL UNIQUE,
    description TEXT,
    category    VARCHAR(50)
);

-- Seed built-in permissions
INSERT INTO permissions (key, description, category) VALUES
  ('users.view',              'Просмотр пользователей',          'users'),
  ('users.edit',              'Редактирование пользователей',     'users'),
  ('users.anonymize',         'Анонимизация аккаунта',           'users'),
  ('registrations.view',      'Просмотр регистраций',            'registrations'),
  ('registrations.edit',      'Изменение статуса/данных рег.',   'registrations'),
  ('registrations.manual',    'Ручная регистрация',              'registrations'),
  ('registrations.export',    'Экспорт регистраций',             'registrations'),
  ('events.view',             'Просмотр мероприятий',            'events'),
  ('events.create',           'Создание мероприятий',            'events'),
  ('events.edit',             'Редактирование мероприятий',      'events'),
  ('events.deactivate',       'Деактивация мероприятий',         'events'),
  ('questions.view',          'Просмотр вопросов',               'questions'),
  ('questions.answer',        'Ответы на вопросы',               'questions'),
  ('questions.assign',        'Назначение оператора',            'questions'),
  ('questions.close',         'Закрытие вопросов',               'questions'),
  ('analytics.view',          'Просмотр аналитики',              'analytics'),
  ('analytics.export',        'Экспорт аналитики',               'analytics'),
  ('logs.view',               'Просмотр системных логов',        'audit'),
  ('audit.view',              'Просмотр аудита администраторов', 'audit'),
  ('roles.manage',            'Управление ролями и правами',     'system'),
  ('admins.create',           'Создание администраторов',        'system'),
  ('admins.deactivate',       'Деактивация администраторов',     'system'),
  ('settings.view',           'Просмотр настроек системы',      'system'),
  ('settings.edit',           'Изменение настроек системы',     'system')
ON CONFLICT (key) DO NOTHING;

-- Role → Permission mapping (system defaults, can be overridden by super_admin)
CREATE TABLE IF NOT EXISTS role_permissions (
    role            VARCHAR(50) NOT NULL,
    permission_id   INT         NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    granted_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    granted_by      INT         REFERENCES admin_users(id) ON DELETE SET NULL,
    PRIMARY KEY (role, permission_id)
);

-- Seed default role permissions
INSERT INTO role_permissions (role, permission_id)
SELECT 'super_admin', id FROM permissions
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role, permission_id)
SELECT 'admin', id FROM permissions
WHERE key IN (
  'users.view','users.edit',
  'registrations.view','registrations.edit','registrations.manual','registrations.export',
  'events.view','events.create','events.edit','events.deactivate',
  'questions.view','questions.answer','questions.assign','questions.close',
  'analytics.view','analytics.export',
  'logs.view'
) ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role, permission_id)
SELECT 'operator', id FROM permissions
WHERE key IN (
  'users.view','users.edit',
  'registrations.view','registrations.edit','registrations.export',
  'events.view',
  'questions.view','questions.answer','questions.close',
  'analytics.view'
) ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role, permission_id)
SELECT 'support', id FROM permissions
WHERE key IN (
  'users.view',
  'registrations.view',
  'events.view',
  'questions.view','questions.answer'
) ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role, permission_id)
SELECT 'viewer', id FROM permissions
WHERE key IN (
  'users.view',
  'registrations.view',
  'events.view',
  'questions.view',
  'analytics.view'
) ON CONFLICT DO NOTHING;

-- Immutable admin action audit log
-- IMPORTANT: no UPDATE/DELETE should ever be run on this table.
-- NOTE: PRIMARY KEY must include partition key (created_at) per PostgreSQL requirement.
CREATE TABLE IF NOT EXISTS admin_action_logs (
    id            BIGSERIAL    NOT NULL,
    admin_id      INT          REFERENCES admin_users(id) ON DELETE SET NULL,
    admin_email   VARCHAR(255),                       -- snapshot, never changes
    action        VARCHAR(100) NOT NULL,              -- 'user.edit', 'reg.status_change'
    target_type   VARCHAR(50)  NOT NULL,              -- 'user'|'registration'|'event'|...
    target_id     INT,
    old_value     JSONB,
    new_value     JSONB,
    ip            VARCHAR(64),
    user_agent    TEXT,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);

-- Quarterly partitions for 2026 (no gaps)
CREATE TABLE IF NOT EXISTS admin_action_logs_2026_q1
    PARTITION OF admin_action_logs
    FOR VALUES FROM ('2026-01-01') TO ('2026-04-01');
CREATE TABLE IF NOT EXISTS admin_action_logs_2026_q2
    PARTITION OF admin_action_logs
    FOR VALUES FROM ('2026-04-01') TO ('2026-07-01');
CREATE TABLE IF NOT EXISTS admin_action_logs_2026_q3
    PARTITION OF admin_action_logs
    FOR VALUES FROM ('2026-07-01') TO ('2026-10-01');
CREATE TABLE IF NOT EXISTS admin_action_logs_2026_q4
    PARTITION OF admin_action_logs
    FOR VALUES FROM ('2026-10-01') TO ('2027-01-01');
CREATE TABLE IF NOT EXISTS admin_action_logs_2027_q1
    PARTITION OF admin_action_logs
    FOR VALUES FROM ('2027-01-01') TO ('2027-04-01');

CREATE INDEX IF NOT EXISTS idx_admin_action_logs_admin
    ON admin_action_logs(admin_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_admin_action_logs_target
    ON admin_action_logs(target_type, target_id);

-- Per-field user data change log (who changed what in user profile)
CREATE TABLE IF NOT EXISTS user_profile_change_logs (
    id            BIGSERIAL    PRIMARY KEY,
    user_id       INT          NOT NULL REFERENCES reg_users(id) ON DELETE CASCADE,
    changed_by    INT          REFERENCES admin_users(id) ON DELETE SET NULL,
    -- NULL changed_by = user changed it themselves
    field_name    VARCHAR(100) NOT NULL,
    old_value     TEXT,
    new_value     TEXT,
    changed_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_user_profile_changes_user
    ON user_profile_change_logs(user_id, changed_at DESC);
