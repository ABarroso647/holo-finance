-- name: UpsertSecurity :exec
INSERT INTO securities (id, plaid_security_id, ticker_symbol, name, type, currency, close_price, close_price_as_of)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(plaid_security_id) DO UPDATE SET
    ticker_symbol     = excluded.ticker_symbol,
    name              = excluded.name,
    type              = excluded.type,
    close_price       = excluded.close_price,
    close_price_as_of = excluded.close_price_as_of,
    updated_at        = CURRENT_TIMESTAMP;

-- name: UpsertHolding :exec
INSERT INTO holdings (id, account_id, security_id, quantity, cost_basis, institution_price, institution_value, currency)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(account_id, security_id) DO UPDATE SET
    quantity          = excluded.quantity,
    cost_basis        = excluded.cost_basis,
    institution_price = excluded.institution_price,
    institution_value = excluded.institution_value,
    updated_at        = CURRENT_TIMESTAMP;

-- name: ListHoldingsForAccount :many
SELECT h.*, s.ticker_symbol, s.name as security_name, s.type as security_type, s.currency as security_currency
FROM holdings h
JOIN securities s ON h.security_id = s.id
WHERE h.account_id = ?
ORDER BY h.institution_value DESC NULLS LAST;

-- name: ListAllHoldings :many
SELECT h.*,
    s.ticker_symbol, s.name as security_name, s.type as security_type,
    COALESCE(a.display_name, a.name) as account_name,
    i.name as institution_name
FROM holdings h
JOIN securities s ON h.security_id = s.id
JOIN accounts a ON h.account_id = a.id
JOIN institutions i ON a.institution_id = i.id
ORDER BY h.institution_value DESC NULLS LAST;

-- name: GetSecurityByPlaidID :one
SELECT * FROM securities WHERE plaid_security_id = ?;

-- name: GetPortfolioSummary :one
SELECT
    CAST(COALESCE(SUM(h.institution_value), 0.0) AS REAL) as total_value,
    CAST(COALESCE(SUM(
        CASE WHEN h.cost_basis IS NOT NULL AND h.quantity IS NOT NULL
        THEN h.institution_value - (h.quantity * h.cost_basis)
        ELSE 0 END
    ), 0.0) AS REAL) as total_gain,
    CAST(COUNT(DISTINCT h.account_id) AS INTEGER) as account_count
FROM holdings h;

-- name: GetAllocationByType :many
SELECT
    COALESCE(NULLIF(s.type, ''), 'other') as security_type,
    CAST(COALESCE(SUM(h.institution_value), 0.0) AS REAL) as value
FROM holdings h
JOIN securities s ON h.security_id = s.id
GROUP BY COALESCE(NULLIF(s.type, ''), 'other')
ORDER BY value DESC;

-- name: GetAllocationByAccount :many
SELECT
    COALESCE(a.display_name, a.name) as account_name,
    i.name as institution_name,
    CAST(COALESCE(SUM(h.institution_value), 0.0) AS REAL) as value
FROM holdings h
JOIN accounts a ON h.account_id = a.id
JOIN institutions i ON a.institution_id = i.id
GROUP BY h.account_id
ORDER BY value DESC;
