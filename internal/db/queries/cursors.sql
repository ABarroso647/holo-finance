-- name: GetCursor :one
SELECT cursor FROM plaid_cursors WHERE item_id = ?;

-- name: UpsertCursor :exec
INSERT INTO plaid_cursors (item_id, cursor)
VALUES (?, ?)
ON CONFLICT(item_id) DO UPDATE SET cursor = excluded.cursor;
