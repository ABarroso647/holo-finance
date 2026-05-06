-- name: UpsertTransaction :one
INSERT INTO transactions (id, account_id, plaid_transaction_id, date, authorized_date, name, merchant_name, amount, currency, category_id, category_source, category_confidence, pending, is_recurring)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(plaid_transaction_id) DO UPDATE SET
    date                = excluded.date,
    authorized_date     = excluded.authorized_date,
    name                = excluded.name,
    merchant_name       = excluded.merchant_name,
    amount              = excluded.amount,
    pending             = excluded.pending,
    is_recurring        = excluded.is_recurring,
    category_id         = CASE WHEN transactions.category_source = 'uncategorized' THEN excluded.category_id         ELSE transactions.category_id         END,
    category_source     = CASE WHEN transactions.category_source = 'uncategorized' THEN excluded.category_source     ELSE transactions.category_source     END,
    category_confidence = CASE WHEN transactions.category_source = 'uncategorized' THEN excluded.category_confidence ELSE transactions.category_confidence END,
    updated_at          = CURRENT_TIMESTAMP
RETURNING *;

-- name: DeleteTransaction :exec
DELETE FROM transactions WHERE plaid_transaction_id = ?;

-- name: ListTransactions :many
SELECT t.*,
    COALESCE(a.display_name, a.name) as account_name,
    i.name as institution_name,
    c.name as category_name,
    c.color as category_color
FROM transactions t
LEFT JOIN accounts a ON t.account_id = a.id
LEFT JOIN institutions i ON a.institution_id = i.id
LEFT JOIN categories c ON t.category_id = c.id
ORDER BY t.date DESC, t.created_at DESC
LIMIT ? OFFSET ?;

-- name: ListTransactionsByAccount :many
SELECT t.*,
    COALESCE(a.display_name, a.name) as account_name,
    i.name as institution_name,
    c.name as category_name,
    c.color as category_color
FROM transactions t
LEFT JOIN accounts a ON t.account_id = a.id
LEFT JOIN institutions i ON a.institution_id = i.id
LEFT JOIN categories c ON t.category_id = c.id
WHERE t.account_id = ?
ORDER BY t.date DESC
LIMIT ? OFFSET ?;

-- name: ListUncategorizedTransactions :many
SELECT * FROM transactions
WHERE category_source = 'uncategorized'
ORDER BY date DESC;

-- name: UpdateTransactionCategory :exec
UPDATE transactions
SET category_id = ?, category_source = ?, category_confidence = NULL, updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: CountTransactions :one
SELECT COUNT(*) FROM transactions;

-- name: ResetRecurringForInstitution :exec
UPDATE transactions SET is_recurring = 0
WHERE account_id IN (SELECT id FROM accounts WHERE institution_id = ?);

-- name: SetTransactionRecurring :exec
UPDATE transactions SET is_recurring = 1 WHERE plaid_transaction_id = ?;

-- name: ListNonManualTransactions :many
SELECT * FROM transactions
WHERE category_source != 'manual'
ORDER BY date DESC;

-- name: UpdateTransactionCategoryBySource :exec
UPDATE transactions
SET category_id = ?, category_source = 'rule', category_confidence = NULL, updated_at = CURRENT_TIMESTAMP
WHERE id = ? AND category_source != 'manual';

-- name: SearchTransactions :many
SELECT t.*,
    COALESCE(a.display_name, a.name) as account_name,
    i.name as institution_name,
    c.name as category_name,
    c.color as category_color
FROM transactions t
LEFT JOIN accounts a ON t.account_id = a.id
LEFT JOIN institutions i ON a.institution_id = i.id
LEFT JOIN categories c ON t.category_id = c.id
WHERE (sqlc.arg(search) = '' OR LOWER(t.name) LIKE '%' || LOWER(sqlc.arg(search)) || '%' OR LOWER(COALESCE(t.merchant_name,'')) LIKE '%' || LOWER(sqlc.arg(search)) || '%')
AND (sqlc.arg(account_id) = '' OR t.account_id = sqlc.arg(account_id))
AND (sqlc.arg(category_id) = '' OR t.category_id = sqlc.arg(category_id)
     OR t.category_id IN (SELECT id FROM categories WHERE parent_id = sqlc.arg(category_id)))
AND (sqlc.arg(date_from) = '' OR t.date >= sqlc.arg(date_from))
AND (sqlc.arg(date_to) = '' OR t.date <= sqlc.arg(date_to))
AND (sqlc.arg(recurring) = '' OR t.is_recurring = 1)
AND (sqlc.arg(tag_id) = '' OR t.id IN (SELECT transaction_id FROM transaction_tags WHERE tag_id = sqlc.arg(tag_id)))
ORDER BY t.date DESC, t.created_at DESC
LIMIT sqlc.arg(limit) OFFSET sqlc.arg(offset);

-- name: CountSearchTransactions :one
SELECT COUNT(*) FROM transactions t
WHERE (sqlc.arg(search) = '' OR LOWER(t.name) LIKE '%' || LOWER(sqlc.arg(search)) || '%' OR LOWER(COALESCE(t.merchant_name,'')) LIKE '%' || LOWER(sqlc.arg(search)) || '%')
AND (sqlc.arg(account_id) = '' OR t.account_id = sqlc.arg(account_id))
AND (sqlc.arg(category_id) = '' OR t.category_id = sqlc.arg(category_id)
     OR t.category_id IN (SELECT id FROM categories WHERE parent_id = sqlc.arg(category_id)))
AND (sqlc.arg(date_from) = '' OR t.date >= sqlc.arg(date_from))
AND (sqlc.arg(date_to) = '' OR t.date <= sqlc.arg(date_to))
AND (sqlc.arg(recurring) = '' OR t.is_recurring = 1)
AND (sqlc.arg(tag_id) = '' OR t.id IN (SELECT transaction_id FROM transaction_tags WHERE tag_id = sqlc.arg(tag_id)));

-- name: SumFilteredTransactions :one
-- Returns spending total, income total, and count for the same filter set as SearchTransactions.
-- Spending = positive non-transfer amounts. Income = negative depository amounts (non-transfer).
SELECT
    CAST(COALESCE(SUM(CASE WHEN t.amount > 0
        AND (t.category_id IS NULL
            OR (t.category_id NOT LIKE 'TRANSFER%'
                AND t.category_id NOT IN ('cat_transfer', 'LOAN_PAYMENTS_CREDIT_CARD_PAYMENT')))
        THEN t.amount ELSE 0 END), 0.0) AS REAL) as spending,
    CAST(COALESCE(SUM(CASE WHEN t.amount < 0
        AND a.type = 'depository'
        AND (t.category_id IS NULL
            OR (t.category_id NOT LIKE 'TRANSFER%'
                AND t.category_id NOT IN ('cat_transfer', 'LOAN_PAYMENTS_CREDIT_CARD_PAYMENT')))
        THEN ABS(t.amount) ELSE 0 END), 0.0) AS REAL) as income,
    CAST(COUNT(*) AS INTEGER) as count
FROM transactions t
LEFT JOIN accounts a ON t.account_id = a.id
WHERE (sqlc.arg(search) = '' OR LOWER(t.name) LIKE '%' || LOWER(sqlc.arg(search)) || '%' OR LOWER(COALESCE(t.merchant_name,'')) LIKE '%' || LOWER(sqlc.arg(search)) || '%')
AND (sqlc.arg(account_id) = '' OR t.account_id = sqlc.arg(account_id))
AND (sqlc.arg(category_id) = '' OR t.category_id = sqlc.arg(category_id)
     OR t.category_id IN (SELECT id FROM categories WHERE parent_id = sqlc.arg(category_id)))
AND (sqlc.arg(date_from) = '' OR t.date >= sqlc.arg(date_from))
AND (sqlc.arg(date_to) = '' OR t.date <= sqlc.arg(date_to))
AND (sqlc.arg(recurring) = '' OR t.is_recurring = 1)
AND (sqlc.arg(tag_id) = '' OR t.id IN (SELECT transaction_id FROM transaction_tags WHERE tag_id = sqlc.arg(tag_id)));

-- name: GetRecentTransactionsByAccount :many
SELECT t.*,
    COALESCE(a.display_name, a.name) as account_name,
    i.name as institution_name,
    c.name as category_name,
    c.color as category_color
FROM transactions t
LEFT JOIN accounts a ON t.account_id = a.id
LEFT JOIN institutions i ON a.institution_id = i.id
LEFT JOIN categories c ON t.category_id = c.id
WHERE t.account_id = ?
AND t.date >= ?
ORDER BY t.date DESC, t.created_at DESC
LIMIT 20;

-- name: GetTransaction :one
SELECT t.*,
    COALESCE(a.display_name, a.name) as account_name,
    i.name as institution_name,
    c.name as category_name,
    c.color as category_color
FROM transactions t
LEFT JOIN accounts a ON t.account_id = a.id
LEFT JOIN institutions i ON a.institution_id = i.id
LEFT JOIN categories c ON t.category_id = c.id
WHERE t.id = ?;

-- name: ExportTransactions :many
SELECT t.*,
    COALESCE(a.display_name, a.name) as account_name,
    i.name as institution_name,
    c.name as category_name,
    c.color as category_color
FROM transactions t
LEFT JOIN accounts a ON t.account_id = a.id
LEFT JOIN institutions i ON a.institution_id = i.id
LEFT JOIN categories c ON t.category_id = c.id
WHERE (sqlc.arg(date_from) = '' OR t.date >= sqlc.arg(date_from))
AND (sqlc.arg(date_to) = '' OR t.date <= sqlc.arg(date_to))
AND t.pending = 0
ORDER BY t.date DESC, t.created_at DESC;
