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
