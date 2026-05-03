-- name: ListTags :many
SELECT * FROM tags ORDER BY name;

-- name: SearchTags :many
SELECT * FROM tags WHERE LOWER(name) LIKE '%' || LOWER(?) || '%' ORDER BY name LIMIT 10;

-- name: UpsertTag :one
INSERT INTO tags (id, name, color)
VALUES (?, ?, ?)
ON CONFLICT(name) DO UPDATE SET color = excluded.color
RETURNING *;

-- name: GetTagByName :one
SELECT * FROM tags WHERE name = ?;

-- name: ListTagsForTransaction :many
SELECT t.* FROM tags t
JOIN transaction_tags tt ON tt.tag_id = t.id
WHERE tt.transaction_id = ?
ORDER BY t.name;

-- name: AddTagToTransaction :exec
INSERT INTO transaction_tags (transaction_id, tag_id) VALUES (?, ?)
ON CONFLICT DO NOTHING;

-- name: RemoveTagFromTransaction :exec
DELETE FROM transaction_tags WHERE transaction_id = ? AND tag_id = ?;

-- name: ListTransactionIDsForTag :many
SELECT transaction_id FROM transaction_tags WHERE tag_id = ?;

-- name: DeleteTag :exec
DELETE FROM tags WHERE id = ?;

-- name: CreateTag :one
INSERT INTO tags (id, name, color) VALUES (?, ?, ?) RETURNING *;
