-- name: UpsertCreditCardLiability :exec
INSERT INTO credit_card_liabilities (
    account_id, last_statement_balance, last_statement_issue_date,
    last_payment_date, last_payment_amount, minimum_payment_amount,
    next_payment_due_date, is_overdue
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(account_id) DO UPDATE SET
    last_statement_balance    = excluded.last_statement_balance,
    last_statement_issue_date = excluded.last_statement_issue_date,
    last_payment_date         = excluded.last_payment_date,
    last_payment_amount       = excluded.last_payment_amount,
    minimum_payment_amount    = excluded.minimum_payment_amount,
    next_payment_due_date     = excluded.next_payment_due_date,
    is_overdue                = excluded.is_overdue,
    updated_at                = CURRENT_TIMESTAMP;

-- name: GetCreditCardLiability :one
SELECT * FROM credit_card_liabilities WHERE account_id = ?;

-- name: ListCreditCardLiabilities :many
SELECT * FROM credit_card_liabilities;
