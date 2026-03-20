CREATE TABLE reg_users (
    id                  SERIAL PRIMARY KEY,
    email               TEXT NOT NULL UNIQUE,
    last_name           TEXT NOT NULL,
    first_name          TEXT NOT NULL,
    patronymic          TEXT,
    organization        TEXT NOT NULL,
    district            TEXT NOT NULL,
    is_union_member     BOOLEAN NOT NULL DEFAULT FALSE,
    union_ticket        TEXT,
    extra_info          TEXT,
    last_ip             TEXT,
    geo_country         TEXT,
    geo_region          TEXT,
    geo_city            TEXT,
    user_agent          TEXT,
    password_hash       TEXT,
    password_updated_at TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ
);

CREATE INDEX idx_users_email ON reg_users(email);
