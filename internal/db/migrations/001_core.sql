CREATE TABLE IF NOT EXISTS settings (
    key   TEXT PRIMARY KEY,
    value INT NOT NULL
);
INSERT INTO settings (key, value) VALUES ('max_active_appeals', 5)
ON CONFLICT (key) DO NOTHING;

CREATE TABLE IF NOT EXISTS users (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    login            TEXT NOT NULL UNIQUE,
    password_hash    TEXT NOT NULL,
    role             TEXT NOT NULL CHECK (role IN ('operator', 'expert', 'admin')),
    specialist_group TEXT NOT NULL DEFAULT '',
    active           BOOLEAN NOT NULL DEFAULT TRUE,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS staff_sessions (
    token_hash TEXT PRIMARY KEY,
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS staff_sessions_user_idx ON staff_sessions (user_id);

CREATE TABLE IF NOT EXISTS applicant_sessions (
    token_hash TEXT PRIMARY KEY,
    appeal_id  UUID NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS categories (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name             TEXT NOT NULL UNIQUE,
    specialist_group TEXT NOT NULL DEFAULT '',
    active           BOOLEAN NOT NULL DEFAULT TRUE,
    free_form        BOOLEAN NOT NULL DEFAULT FALSE,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);
