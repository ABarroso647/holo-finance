-- name: InsertRule :one
INSERT INTO rules (id, match_type, match_field, match_value, category_id, priority)
VALUES (?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: ListRules :many
SELECT r.*, c.name as category_name FROM rules r
JOIN categories c ON r.category_id = c.id
ORDER BY r.priority ASC, r.created_at ASC;

-- name: DeleteRule :exec
DELETE FROM rules WHERE id = ?;
