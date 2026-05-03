-- +goose Up
CREATE TABLE IF NOT EXISTS securities (
    id                TEXT PRIMARY KEY,
    plaid_security_id TEXT UNIQUE NOT NULL,
    ticker_symbol     TEXT,
    name              TEXT NOT NULL,
    type              TEXT,
    currency          TEXT NOT NULL DEFAULT 'CAD',
    close_price       REAL,
    close_price_as_of TEXT,
    created_at        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS holdings (
    id                 TEXT PRIMARY KEY,
    account_id         TEXT NOT NULL REFERENCES accounts(id),
    security_id        TEXT NOT NULL REFERENCES securities(id),
    quantity           REAL NOT NULL,
    cost_basis         REAL,
    institution_price  REAL,
    institution_value  REAL,
    currency           TEXT NOT NULL DEFAULT 'CAD',
    updated_at         DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(account_id, security_id)
);

-- +goose Down
DROP TABLE IF EXISTS holdings;
DROP TABLE IF EXISTS securities;
