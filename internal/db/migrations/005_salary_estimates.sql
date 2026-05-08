-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS salary_estimates (
    id               TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(8)))),
    merchant_name    TEXT NOT NULL,
    avg_amount       REAL NOT NULL,
    avg_day_of_month REAL NOT NULL,
    months_seen      INTEGER NOT NULL,
    last_detected_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(merchant_name)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS salary_estimates;
-- +goose StatementEnd
