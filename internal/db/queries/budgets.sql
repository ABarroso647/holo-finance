-- name: UpsertBudget :one
INSERT INTO category_budgets (id, category_id, monthly_limit)
VALUES (lower(hex(randomblob(8))), sqlc.arg(category_id), sqlc.arg(monthly_limit))
ON CONFLICT(category_id) DO UPDATE SET
    monthly_limit = excluded.monthly_limit
RETURNING *;

-- name: ListBudgets :many
SELECT
    cb.id,
    cb.category_id,
    cb.monthly_limit,
    cb.created_at,
    c.name  as category_name,
    c.color as category_color
FROM category_budgets cb
JOIN categories c ON cb.category_id = c.id
ORDER BY c.name;

-- name: DeleteBudget :exec
DELETE FROM category_budgets WHERE id = sqlc.arg(id);

-- name: GetBudgetByCategoryID :one
SELECT * FROM category_budgets WHERE category_id = sqlc.arg(category_id);
