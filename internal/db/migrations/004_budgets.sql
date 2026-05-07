-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS category_budgets (
    id            TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(8)))),
    category_id   TEXT NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    monthly_limit REAL NOT NULL CHECK(monthly_limit > 0),
    created_at    TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(category_id)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS category_budgets;
-- +goose StatementEnd
