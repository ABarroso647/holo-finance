-- name: GetNetWorth :one
SELECT COALESCE(SUM(current_balance), 0.0) FROM accounts;

-- name: GetThisMonthSummary :one
-- Spending = outflows across all accounts, excluding transfers.
-- Income = inflows on depository accounts only, excluding transfers.
-- Cashback = INCOME% credits on credit accounts (statement credits, rewards).
SELECT
    COALESCE(SUM(CASE WHEN t.amount > 0
        AND t.category_id NOT LIKE 'TRANSFER%'
        AND t.category_id NOT IN ('cat_transfer', 'LOAN_PAYMENTS_CREDIT_CARD_PAYMENT')
        THEN t.amount ELSE 0 END), 0.0) as spending,
    COALESCE(SUM(CASE WHEN t.amount < 0
        AND a.type = 'depository'
        AND t.category_id NOT LIKE 'TRANSFER%'
        AND t.category_id NOT IN ('cat_transfer', 'LOAN_PAYMENTS_CREDIT_CARD_PAYMENT')
        THEN ABS(t.amount) ELSE 0 END), 0.0) as income,
    COALESCE(SUM(CASE WHEN t.amount < 0
        AND a.type = 'depository'
        AND t.category_id LIKE 'INCOME_WAGES%'
        THEN ABS(t.amount) ELSE 0 END), 0.0) as salary,
    COALESCE(SUM(CASE WHEN t.amount < 0
        AND a.type = 'depository'
        AND (t.category_id LIKE 'INCOME_INTEREST%' OR t.category_id = 'INCOME_INTEREST')
        THEN ABS(t.amount) ELSE 0 END), 0.0) as interest,
    COALESCE(SUM(CASE WHEN t.amount < 0
        AND a.type = 'credit'
        AND t.category_id LIKE 'INCOME%'
        THEN ABS(t.amount) ELSE 0 END), 0.0) as cashback
FROM transactions t
JOIN accounts a ON t.account_id = a.id
WHERE t.date >= ? AND t.date <= ? AND t.pending = 0;

-- name: GetMonthlyTotals :many
-- date param is start date YYYY-MM-DD string, returns last 12 months
SELECT
    strftime('%Y-%m', t.date) as month,
    COALESCE(SUM(CASE WHEN t.amount > 0
        AND t.category_id NOT LIKE 'TRANSFER%'
        AND t.category_id NOT IN ('cat_transfer', 'LOAN_PAYMENTS_CREDIT_CARD_PAYMENT')
        THEN t.amount ELSE 0 END), 0.0) as spending,
    COALESCE(SUM(CASE WHEN t.amount < 0
        AND a.type = 'depository'
        AND t.category_id NOT LIKE 'TRANSFER%'
        AND t.category_id NOT IN ('cat_transfer', 'LOAN_PAYMENTS_CREDIT_CARD_PAYMENT')
        THEN ABS(t.amount) ELSE 0 END), 0.0) as income
FROM transactions t
JOIN accounts a ON t.account_id = a.id
WHERE t.date >= ? AND t.pending = 0
GROUP BY strftime('%Y-%m', t.date)
ORDER BY month ASC;

-- name: GetMonthlySpendingByCategory :many
-- Returns spending per parent category per month for stacked trend chart.
SELECT
    strftime('%Y-%m', t.date) as month,
    COALESCE(p.id, c.id, 'uncategorized') as category_id,
    COALESCE(p.name, c.name, 'Uncategorized') as category_name,
    COALESCE(p.color, c.color, '#64748b') as category_color,
    CAST(SUM(CASE WHEN t.amount > 0 THEN t.amount ELSE 0 END) -
         SUM(CASE WHEN t.amount < 0 THEN ABS(t.amount) ELSE 0 END)
         AS REAL) as total
FROM transactions t
LEFT JOIN categories c ON t.category_id = c.id
LEFT JOIN categories p ON c.parent_id = p.id
WHERE t.pending = 0
    AND t.date >= ?
    AND (t.category_id IS NULL
        OR (t.category_id NOT LIKE 'TRANSFER%'
            AND t.category_id NOT LIKE 'INCOME%'
            AND t.category_id NOT IN ('cat_transfer', 'cat_income', 'cat_investment', 'LOAN_PAYMENTS_CREDIT_CARD_PAYMENT')))
GROUP BY month, COALESCE(c.parent_id, c.id, 'uncategorized')
HAVING total > 0
ORDER BY month ASC, total DESC;

-- name: GetAccountSpendSince :one
SELECT CAST(COALESCE(SUM(amount), 0.0) AS REAL) as total
FROM transactions
WHERE account_id = ?
AND amount > 0
AND pending = 0
AND date >= ?
AND (category_id IS NULL
    OR (category_id NOT LIKE 'TRANSFER%'
        AND category_id NOT IN ('cat_transfer', 'LOAN_PAYMENTS_CREDIT_CARD_PAYMENT')));

-- name: GetTopCategoriesForAccount :many
SELECT
    COALESCE(t.category_id, 'uncategorized') as category_id,
    COALESCE(c.name, 'Uncategorized') as category_name,
    COALESCE(c.color, '#64748b') as category_color,
    CAST(SUM(t.amount) AS REAL) as total
FROM transactions t
LEFT JOIN categories c ON t.category_id = c.id
WHERE t.account_id = ?
AND t.amount > 0
AND t.pending = 0
AND t.date >= ?
AND (t.category_id IS NULL
    OR (t.category_id NOT LIKE 'TRANSFER%'
        AND t.category_id NOT IN ('cat_transfer', 'LOAN_PAYMENTS_CREDIT_CARD_PAYMENT')))
GROUP BY t.category_id
ORDER BY total DESC
LIMIT 5;

-- name: GetSpendingByCategory :many
-- params: start_date, end_date as YYYY-MM-DD strings
-- Groups by parent category when present, deduping sub-categories.
-- Subtracts refunds (negative amounts) from spending. Excludes net-credit groups.
SELECT
    COALESCE(p.id, c.id, 'uncategorized') as category_id,
    COALESCE(p.name, c.name, 'Uncategorized') as category_name,
    COALESCE(p.color, c.color, '#64748b') as category_color,
    CAST(SUM(CASE WHEN t.amount > 0 THEN t.amount ELSE 0 END) -
         SUM(CASE WHEN t.amount < 0 THEN ABS(t.amount) ELSE 0 END)
         AS REAL) as total
FROM transactions t
LEFT JOIN categories c ON t.category_id = c.id
LEFT JOIN categories p ON c.parent_id = p.id
WHERE t.pending = 0
    AND t.date >= ?
    AND t.date <= ?
    AND (t.category_id IS NULL
        OR (t.category_id NOT LIKE 'TRANSFER%'
            AND t.category_id NOT LIKE 'INCOME%'
            AND t.category_id NOT IN ('cat_transfer', 'cat_income', 'cat_investment', 'LOAN_PAYMENTS_CREDIT_CARD_PAYMENT')))
GROUP BY COALESCE(c.parent_id, c.id, 'uncategorized')
HAVING total > 0
ORDER BY total DESC;

-- name: GetRecurringSpendForPeriod :one
-- Sum of positive recurring transactions (non-transfer) for a date range.
SELECT CAST(COALESCE(SUM(t.amount), 0.0) AS REAL) as total
FROM transactions t
WHERE t.is_recurring = 1
  AND t.amount > 0
  AND t.pending = 0
  AND t.date >= ?
  AND t.date <= ?
  AND (t.category_id IS NULL
      OR (t.category_id NOT LIKE 'TRANSFER%'
          AND t.category_id NOT IN ('cat_transfer', 'LOAN_PAYMENTS_CREDIT_CARD_PAYMENT')));

-- name: GetSpendByTag :many
SELECT
    tg.id,
    tg.name,
    tg.color,
    CAST(COALESCE(SUM(t.amount), 0.0) AS REAL) as total,
    CAST(COUNT(DISTINCT t.id) AS INTEGER) as txn_count
FROM tags tg
JOIN transaction_tags tt ON tt.tag_id = tg.id
JOIN transactions t ON t.id = tt.transaction_id
WHERE t.amount > 0
    AND t.pending = 0
    AND (sqlc.arg(date_from) = '' OR t.date >= sqlc.arg(date_from))
    AND (sqlc.arg(date_to) = '' OR t.date <= sqlc.arg(date_to))
GROUP BY tg.id
ORDER BY total DESC;

-- name: GetMonthlyFlows :many
-- Returns income and spending per month (ordered oldest first).
-- date param is start YYYY-MM-DD string.
SELECT
    strftime('%Y-%m', t.date) as month,
    CAST(COALESCE(SUM(CASE WHEN t.amount < 0
        AND a.type = 'depository'
        AND t.category_id NOT LIKE 'TRANSFER%'
        AND t.category_id NOT IN ('cat_transfer', 'LOAN_PAYMENTS_CREDIT_CARD_PAYMENT')
        THEN ABS(t.amount) ELSE 0 END), 0.0) AS REAL) as income,
    CAST(COALESCE(SUM(CASE WHEN t.amount > 0
        AND t.category_id NOT LIKE 'TRANSFER%'
        AND t.category_id NOT IN ('cat_transfer', 'LOAN_PAYMENTS_CREDIT_CARD_PAYMENT')
        THEN t.amount ELSE 0 END), 0.0) AS REAL) as spending
FROM transactions t
JOIN accounts a ON t.account_id = a.id
WHERE t.date >= ? AND t.pending = 0
GROUP BY strftime('%Y-%m', t.date)
ORDER BY month ASC;
