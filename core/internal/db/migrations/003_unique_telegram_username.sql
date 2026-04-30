-- +goose Up
-- Remove duplicates first: keep the earliest record per telegram_username.
DELETE FROM participants a
  USING participants b
  WHERE a.telegram_username = b.telegram_username
    AND a.telegram_username != ''
    AND a.created_at > b.created_at;

CREATE UNIQUE INDEX IF NOT EXISTS idx_participants_telegram_username
  ON participants (telegram_username)
  WHERE telegram_username != '';

-- Also make group name unique.
CREATE UNIQUE INDEX IF NOT EXISTS idx_groups_name ON groups (name);

-- +goose Down
DROP INDEX IF EXISTS idx_participants_telegram_username;
DROP INDEX IF EXISTS idx_groups_name;
