-- name: GetRecurringIncomePatterns :many
SELECT
    COALESCE(t.merchant_name, t.name) as merchant,
    CAST(COUNT(DISTINCT strftime('%Y-%m', t.date)) AS INTEGER) as months_seen,
    CAST(AVG(t.amount) AS REAL) as avg_amount,
    CAST(AVG(CAST(strftime('%d', t.date) AS INTEGER)) AS REAL) as avg_day
FROM transactions t
JOIN accounts a ON t.account_id = a.id
WHERE t.amount < 0
  AND a.type = 'depository'
  AND t.category_id LIKE 'INCOME_WAGES%'
  AND t.pending = 0
GROUP BY COALESCE(t.merchant_name, t.name)
HAVING months_seen >= 3
ORDER BY avg_amount ASC;

-- name: UpsertSalaryEstimate :exec
INSERT INTO salary_estimates (id, merchant_name, avg_amount, avg_day_of_month, months_seen)
VALUES (lower(hex(randomblob(8))), sqlc.arg(merchant_name), sqlc.arg(avg_amount), sqlc.arg(avg_day_of_month), sqlc.arg(months_seen))
ON CONFLICT(merchant_name) DO UPDATE SET
    avg_amount       = excluded.avg_amount,
    avg_day_of_month = excluded.avg_day_of_month,
    months_seen      = excluded.months_seen,
    last_detected_at = CURRENT_TIMESTAMP;

-- name: ListSalaryEstimates :many
SELECT * FROM salary_estimates ORDER BY avg_amount ASC;

-- name: DeleteAllSalaryEstimates :exec
DELETE FROM salary_estimates;
