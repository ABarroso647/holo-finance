-- name: ListCardRewardRatesForAccount :many
SELECT
    r.id,
    r.account_id,
    r.category_id,
    r.raw_category,
    r.reward_rate,
    r.cap_amount,
    r.cap_period,
    r.notes,
    r.created_at,
    COALESCE(c.name, '') AS category_name,
    COALESCE(c.color, '') AS category_color
FROM card_reward_rates r
LEFT JOIN categories c ON r.category_id = c.id
WHERE r.account_id = sqlc.arg(account_id)
ORDER BY r.category_id IS NOT NULL DESC, r.reward_rate DESC;

-- name: UpsertCardRewardRate :one
INSERT INTO card_reward_rates (id, account_id, category_id, raw_category, reward_rate, cap_amount, cap_period, notes)
VALUES (sqlc.arg(id), sqlc.arg(account_id), sqlc.arg(category_id), sqlc.arg(raw_category), sqlc.arg(reward_rate), sqlc.arg(cap_amount), sqlc.arg(cap_period), sqlc.arg(notes))
ON CONFLICT(id) DO UPDATE SET
    category_id  = excluded.category_id,
    raw_category = excluded.raw_category,
    reward_rate  = excluded.reward_rate,
    cap_amount   = excluded.cap_amount,
    cap_period   = excluded.cap_period,
    notes        = excluded.notes
RETURNING *;

-- name: DeleteCardRewardRatesForAccount :exec
DELETE FROM card_reward_rates WHERE account_id = sqlc.arg(account_id);

-- name: DeleteCardRewardRate :exec
DELETE FROM card_reward_rates WHERE id = sqlc.arg(id);

-- name: UpdateCardRewardRateCategoryID :exec
UPDATE card_reward_rates
SET category_id = sqlc.arg(category_id)
WHERE id = sqlc.arg(id);

-- name: ListAllCardRewardRatesWithNullCategory :many
SELECT * FROM card_reward_rates WHERE category_id IS NULL AND raw_category IS NOT NULL;

-- name: CountCardRewardRates :one
SELECT CAST(COUNT(*) AS INTEGER) FROM card_reward_rates;

-- name: GetBestRateForCategory :one
SELECT r.reward_rate, r.account_id, r.notes
FROM card_reward_rates r
WHERE (r.category_id = sqlc.arg(category_id)
       OR r.category_id = (SELECT c.parent_id FROM categories c WHERE c.id = sqlc.arg(category_id))
       OR r.category_id IS NULL)
  AND r.account_id IN (
    SELECT id FROM accounts WHERE type = 'credit'
  )
ORDER BY
    CASE WHEN r.category_id = ?1 THEN 0
         WHEN r.category_id IS NOT NULL THEN 1
         ELSE 2 END ASC,
    r.reward_rate DESC
LIMIT 1;
