-- name: GetSetting :one
SELECT value FROM app_settings WHERE key = ?;

-- name: UpsertSetting :exec
INSERT INTO app_settings (key, value, updated_at)
VALUES (?, ?, CURRENT_TIMESTAMP)
ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = CURRENT_TIMESTAMP;
