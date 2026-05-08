-- name: GetInstitutionSyncStats :many
SELECT i.name, i.plaid_item_id,
       MIN(t.date) as earliest_date,
       MAX(t.date) as latest_date,
       COUNT(t.id) as total_txns,
       COUNT(DISTINCT a.id) as account_count
FROM institutions i
LEFT JOIN accounts a ON a.institution_id = i.id
LEFT JOIN transactions t ON t.account_id = a.id
GROUP BY i.id;

-- name: GetCursorByItemID :one
SELECT * FROM plaid_cursors WHERE item_id = sqlc.arg(item_id);

-- name: ListAllCursors :many
SELECT * FROM plaid_cursors;

-- name: ResetCursor :exec
UPDATE plaid_cursors SET cursor = NULL WHERE item_id = sqlc.arg(item_id);

-- name: ListAccountsForDebug :many
SELECT a.id, a.plaid_account_id, a.name, a.type, a.subtype, a.institution_id,
       i.plaid_item_id, i.plaid_access_token, i.name as institution_name
FROM accounts a
JOIN institutions i ON a.institution_id = i.id
ORDER BY i.name, a.name;

-- name: DeleteAccountByID :exec
DELETE FROM accounts WHERE id = sqlc.arg(id);
