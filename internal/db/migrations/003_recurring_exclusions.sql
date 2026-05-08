-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS recurring_exclusions (
    merchant TEXT PRIMARY KEY,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS recurring_exclusions;
-- +goose StatementEnd
