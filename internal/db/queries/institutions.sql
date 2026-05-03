-- name: UpsertInstitution :one
INSERT INTO institutions (id, plaid_item_id, plaid_access_token, name, updated_at)
VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)
ON CONFLICT(plaid_item_id) DO UPDATE SET
    plaid_access_token = excluded.plaid_access_token,
    name               = excluded.name,
    updated_at         = CURRENT_TIMESTAMP
RETURNING *;

-- name: GetInstitutionByItemID :one
SELECT * FROM institutions WHERE plaid_item_id = ?;

-- name: ListInstitutions :many
SELECT * FROM institutions ORDER BY name;

-- name: GetInstitutionByID :one
SELECT * FROM institutions WHERE id = ?;

-- name: UpdateInstitutionToken :exec
UPDATE institutions SET plaid_access_token = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?;
