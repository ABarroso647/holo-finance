-- name: UpsertAccount :one
INSERT INTO accounts (id, institution_id, plaid_account_id, name, official_name, type, subtype, currency, current_balance, available_balance, last_synced_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
ON CONFLICT(plaid_account_id) DO UPDATE SET
    name              = excluded.name,
    official_name     = excluded.official_name,
    current_balance   = excluded.current_balance,
    available_balance = excluded.available_balance,
    last_synced_at    = CURRENT_TIMESTAMP
RETURNING *;

-- name: ListAccounts :many
SELECT * FROM accounts ORDER BY name;

-- name: ListAccountsByInstitution :many
SELECT * FROM accounts WHERE institution_id = ? ORDER BY name;

-- name: GetAccountByPlaidID :one
SELECT * FROM accounts WHERE plaid_account_id = ?;

-- name: GetAccountByID :one
SELECT * FROM accounts WHERE id = ?;

-- name: UpdateAccountDisplayName :exec
UPDATE accounts SET display_name = ? WHERE id = ?;

-- name: ListAccountsWithInstitution :many
SELECT a.*, i.name as institution_name
FROM accounts a
JOIN institutions i ON a.institution_id = i.id
ORDER BY i.name, a.type, a.name;
