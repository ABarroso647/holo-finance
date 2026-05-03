-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS institutions (
    id                 TEXT PRIMARY KEY,
    plaid_item_id      TEXT UNIQUE NOT NULL,
    plaid_access_token TEXT NOT NULL,
    name               TEXT NOT NULL,
    created_at         DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at         DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS accounts (
    id                TEXT PRIMARY KEY,
    institution_id    TEXT NOT NULL REFERENCES institutions(id),
    plaid_account_id  TEXT UNIQUE NOT NULL,
    name              TEXT NOT NULL,
    official_name     TEXT,
    type              TEXT NOT NULL,
    subtype           TEXT,
    currency          TEXT NOT NULL DEFAULT 'CAD',
    current_balance   REAL,
    available_balance REAL,
    last_synced_at    DATETIME,
    created_at        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS categories (
    id        TEXT PRIMARY KEY,
    name      TEXT NOT NULL,
    parent_id TEXT REFERENCES categories(id),
    kind      TEXT NOT NULL,
    color     TEXT,
    icon      TEXT
);

CREATE TABLE IF NOT EXISTS transactions (
    id                   TEXT PRIMARY KEY,
    account_id           TEXT NOT NULL REFERENCES accounts(id),
    plaid_transaction_id TEXT UNIQUE NOT NULL,
    date                 TEXT NOT NULL,
    authorized_date      TEXT,
    name                 TEXT NOT NULL,
    merchant_name        TEXT,
    amount               REAL NOT NULL,
    currency             TEXT NOT NULL DEFAULT 'CAD',
    category_id          TEXT REFERENCES categories(id),
    category_source      TEXT NOT NULL DEFAULT 'uncategorized',
    pending              INTEGER NOT NULL DEFAULT 0,
    is_recurring         INTEGER NOT NULL DEFAULT 0,
    notes                TEXT,
    created_at           DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at           DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS rules (
    id          TEXT PRIMARY KEY,
    match_type  TEXT NOT NULL,
    match_field TEXT NOT NULL,
    match_value TEXT NOT NULL,
    category_id TEXT NOT NULL REFERENCES categories(id),
    priority    INTEGER NOT NULL DEFAULT 100,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS plaid_cursors (
    item_id TEXT PRIMARY KEY,
    cursor  TEXT
);

CREATE TABLE IF NOT EXISTS webhook_events (
    id           TEXT PRIMARY KEY,
    item_id      TEXT,
    webhook_type TEXT,
    webhook_code TEXT,
    payload      TEXT,
    received_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS credit_card_liabilities (
    account_id                TEXT PRIMARY KEY REFERENCES accounts(id),
    last_statement_balance    REAL,
    last_statement_issue_date TEXT,
    last_payment_date         TEXT,
    last_payment_amount       REAL,
    minimum_payment_amount    REAL,
    next_payment_due_date     TEXT,
    is_overdue                INTEGER NOT NULL DEFAULT 0,
    updated_at                DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS card_config (
    account_id     TEXT PRIMARY KEY REFERENCES accounts(id),
    points_program TEXT,
    reward_type    TEXT NOT NULL DEFAULT 'cashback',
    points_cpp     REAL NOT NULL DEFAULT 1.0,
    cpp_overridden INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS cpp_cache (
    program    TEXT PRIMARY KEY,
    cpp        REAL NOT NULL,
    source     TEXT NOT NULL,
    fetched_at DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS card_reward_rates (
    id          TEXT PRIMARY KEY,
    account_id  TEXT NOT NULL REFERENCES accounts(id),
    category_id TEXT REFERENCES categories(id),
    reward_rate REAL NOT NULL,
    notes       TEXT,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS webauthn_credentials (
    id            TEXT PRIMARY KEY,
    credential_id BLOB UNIQUE NOT NULL,
    public_key    BLOB NOT NULL,
    sign_count    INTEGER NOT NULL DEFAULT 0,
    created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS webauthn_credentials;
DROP TABLE IF EXISTS card_reward_rates;
DROP TABLE IF EXISTS cpp_cache;
DROP TABLE IF EXISTS card_config;
DROP TABLE IF EXISTS credit_card_liabilities;
DROP TABLE IF EXISTS webhook_events;
DROP TABLE IF EXISTS plaid_cursors;
DROP TABLE IF EXISTS rules;
DROP TABLE IF EXISTS transactions;
DROP TABLE IF EXISTS categories;
DROP TABLE IF EXISTS accounts;
DROP TABLE IF EXISTS institutions;
-- +goose StatementEnd
