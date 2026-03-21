-- Migration 014: reference lists (справочники) — admin-managed dropdown lists
-- used in form builder for select/multiselect fields.

CREATE TABLE IF NOT EXISTS ref_lists (
    id          SERIAL PRIMARY KEY,
    slug        TEXT        NOT NULL UNIQUE,  -- machine name, e.g. 'moscow_districts'
    title       TEXT        NOT NULL,         -- display name
    description TEXT        DEFAULT '',
    is_active   BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS ref_list_items (
    id         SERIAL PRIMARY KEY,
    list_id    INTEGER NOT NULL REFERENCES ref_lists(id) ON DELETE CASCADE,
    value      TEXT    NOT NULL,              -- stored value
    label      TEXT    NOT NULL,              -- display label
    sort_order INTEGER NOT NULL DEFAULT 0,
    is_active  BOOLEAN NOT NULL DEFAULT TRUE,
    UNIQUE(list_id, value)
);

CREATE INDEX IF NOT EXISTS idx_ref_items_list ON ref_list_items(list_id, sort_order);

-- ── Prefill: Московские районы (10 округов + внегородские территории) ─────────

INSERT INTO ref_lists (slug, title, description) VALUES
    ('moscow_districts', 'Округа Москвы', 'Административные округа г. Москвы');

INSERT INTO ref_list_items (list_id, value, label, sort_order)
SELECT l.id, v.value, v.label, v.sort_order
FROM ref_lists l,
(VALUES
    ('CAO',  'Центральный административный округ',      1),
    ('SAO',  'Северный административный округ',         2),
    ('SVAO', 'Северо-Восточный административный округ', 3),
    ('VAO',  'Восточный административный округ',        4),
    ('UVAO', 'Юго-Восточный административный округ',    5),
    ('UAO',  'Южный административный округ',            6),
    ('UZAO', 'Юго-Западный административный округ',     7),
    ('ZAO',  'Западный административный округ',         8),
    ('SZAO', 'Северо-Западный административный округ',  9),
    ('ZelAO','Зеленоградский административный округ',  10),
    ('TroNov','Троицкий и Новомосковский округа',       11),
    ('OTHER','Другой регион',                           99)
) AS v(value, label, sort_order)
WHERE l.slug = 'moscow_districts';

-- ── Prefill: типы участников ──────────────────────────────────────────────────

INSERT INTO ref_lists (slug, title, description) VALUES
    ('participant_roles', 'Роли участников', 'Тип/роль участника мероприятия');

INSERT INTO ref_list_items (list_id, value, label, sort_order)
SELECT l.id, v.value, v.label, v.sort_order
FROM ref_lists l,
(VALUES
    ('delegate',  'Делегат',        1),
    ('speaker',   'Докладчик',      2),
    ('moderator', 'Модератор',      3),
    ('observer',  'Наблюдатель',    4),
    ('press',     'Пресса',         5),
    ('vip',       'VIP-гость',      6),
    ('staff',     'Организатор',    7),
    ('volunteer', 'Волонтёр',       8)
) AS v(value, label, sort_order)
WHERE l.slug = 'participant_roles';
