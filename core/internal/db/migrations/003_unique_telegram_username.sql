-- +goose Up

-- Remove duplicate participants: keep the earliest record per telegram_username.
DELETE FROM participants a
  USING participants b
  WHERE a.telegram_username = b.telegram_username
    AND a.telegram_username != ''
    AND a.created_at > b.created_at;

CREATE UNIQUE INDEX IF NOT EXISTS idx_participants_telegram_username
  ON participants (telegram_username)
  WHERE telegram_username != '';

-- Remove duplicate groups: reassign participants to the kept group, then delete dupes.
UPDATE participants SET group_id = kept.id
FROM (
  SELECT name, MIN(created_at) AS min_created, (
    SELECT id FROM groups g2 WHERE g2.name = g1.name ORDER BY created_at LIMIT 1
  ) AS id
  FROM groups g1 GROUP BY name HAVING COUNT(*) > 1
) AS kept
WHERE participants.group_id IN (
  SELECT id FROM groups WHERE name = kept.name AND id != kept.id
);

DELETE FROM groups a
  USING groups b
  WHERE a.name = b.name
    AND a.created_at > b.created_at;

CREATE UNIQUE INDEX IF NOT EXISTS idx_groups_name ON groups (name);

-- +goose Down
DROP INDEX IF EXISTS idx_participants_telegram_username;
DROP INDEX IF EXISTS idx_groups_name;
