-- +goose Up
ALTER TABLE participants ADD COLUMN telegram_username TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE participants DROP COLUMN telegram_username;
