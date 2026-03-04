-- +goose Up
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS groups (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS users (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    telegram_id BIGINT NOT NULL UNIQUE,
    name        TEXT NOT NULL,
    username    TEXT NOT NULL DEFAULT '',
    role        TEXT NOT NULL DEFAULT 'organizer' CHECK (role IN ('organizer', 'curator', 'admin', 'root')),
    group_id    UUID REFERENCES groups(id) ON DELETE SET NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS media (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    file_path     TEXT NOT NULL,
    mime_type     TEXT NOT NULL,
    original_name TEXT NOT NULL DEFAULT '',
    size_bytes    BIGINT NOT NULL DEFAULT 0,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS participants (
    id                UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name              TEXT NOT NULL,
    telegram_id       BIGINT UNIQUE,
    custom_identifier TEXT,
    group_id          UUID REFERENCES groups(id) ON DELETE SET NULL,
    photo_media_id    UUID REFERENCES media(id) ON DELETE SET NULL,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS notes (
    id             UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    author_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    participant_id UUID REFERENCES participants(id) ON DELETE SET NULL,
    text           TEXT NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_users_telegram_id ON users(telegram_id);
CREATE INDEX IF NOT EXISTS idx_participants_group_id ON participants(group_id);
CREATE INDEX IF NOT EXISTS idx_notes_author_id ON notes(author_id);
CREATE INDEX IF NOT EXISTS idx_notes_participant_id ON notes(participant_id);

-- +goose Down
DROP TABLE IF EXISTS notes;
DROP TABLE IF EXISTS participants;
DROP TABLE IF EXISTS media;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS groups;
