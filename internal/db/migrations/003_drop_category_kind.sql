-- +goose Up
ALTER TABLE categories DROP COLUMN kind;

-- +goose Down
ALTER TABLE categories ADD COLUMN kind TEXT NOT NULL DEFAULT 'variable';
