-- +goose Up
ALTER TABLE card_reward_rates ADD COLUMN raw_category TEXT;
ALTER TABLE card_reward_rates ADD COLUMN cap_amount REAL;
ALTER TABLE card_reward_rates ADD COLUMN cap_period TEXT;

-- +goose Down
-- SQLite doesn't support DROP COLUMN in older versions; no-op
