-- name: UpsertCategory :one
INSERT INTO categories (id, name, parent_id, color, icon)
VALUES (?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
    name      = excluded.name,
    parent_id = excluded.parent_id,
    color     = excluded.color,
    icon      = excluded.icon
RETURNING *;

-- name: ListCategories :many
SELECT * FROM categories ORDER BY name;

-- name: GetCategoryByID :one
SELECT * FROM categories WHERE id = ?;

-- name: GetCategoryByName :one
SELECT * FROM categories WHERE name = ?;

-- name: UpdateCategory :one
UPDATE categories SET name = ?, color = ? WHERE id = ? RETURNING *;
