-- +goose Up
ALTER TABLE transactions ADD COLUMN category_confidence TEXT;

-- +goose Down
ALTER TABLE transactions DROP COLUMN category_confidence;
