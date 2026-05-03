-- Amount convention: positive = debit/spending, negative = credit/income

CREATE TABLE IF NOT EXISTS institutions (
    id         TEXT PRIMARY KEY,
    plaid_item_id   TEXT UNIQUE NOT NULL,
    plaid_access_token TEXT NOT NULL,
    name       TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS accounts (
    id                 TEXT PRIMARY KEY,
    institution_id     TEXT NOT NULL REFERENCES institutions(id),
    plaid_account_id   TEXT UNIQUE NOT NULL,
    name               TEXT NOT NULL,
    official_name      TEXT,
    display_name       TEXT,
    type               TEXT NOT NULL,
    subtype            TEXT,
    currency           TEXT NOT NULL DEFAULT 'CAD',
    current_balance    REAL,
    available_balance  REAL,
    last_synced_at     DATETIME,
    created_at         DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS categories (
    id        TEXT PRIMARY KEY,
    name      TEXT NOT NULL,
    parent_id TEXT REFERENCES categories(id),
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
    category_confidence  TEXT,
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
    account_id              TEXT PRIMARY KEY REFERENCES accounts(id),
    last_statement_balance  REAL,
    last_statement_issue_date TEXT,
    last_payment_date       TEXT,
    last_payment_amount     REAL,
    minimum_payment_amount  REAL,
    next_payment_due_date   TEXT,
    is_overdue              INTEGER NOT NULL DEFAULT 0,
    updated_at              DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS card_config (
    account_id      TEXT PRIMARY KEY REFERENCES accounts(id),
    points_program  TEXT,
    reward_type     TEXT NOT NULL DEFAULT 'cashback',
    points_cpp      REAL NOT NULL DEFAULT 1.0,
    cpp_overridden  INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS cpp_cache (
    program    TEXT PRIMARY KEY,
    cpp        REAL NOT NULL,
    source     TEXT NOT NULL,
    fetched_at DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS card_reward_rates (
    id           TEXT PRIMARY KEY,
    account_id   TEXT NOT NULL REFERENCES accounts(id),
    category_id  TEXT REFERENCES categories(id),
    raw_category TEXT,
    reward_rate  REAL NOT NULL,
    cap_amount   REAL,
    cap_period   TEXT,
    notes        TEXT,
    created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS app_settings (
    key        TEXT PRIMARY KEY,
    value      TEXT NOT NULL,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS webauthn_credentials (
    id            TEXT PRIMARY KEY,
    credential_id BLOB UNIQUE NOT NULL,
    public_key    BLOB NOT NULL,
    sign_count    INTEGER NOT NULL DEFAULT 0,
    created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS tags (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL UNIQUE,
    color      TEXT NOT NULL DEFAULT '#64748b',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS transaction_tags (
    transaction_id TEXT NOT NULL REFERENCES transactions(id) ON DELETE CASCADE,
    tag_id         TEXT NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (transaction_id, tag_id)
);

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
    id                TEXT PRIMARY KEY,
    account_id        TEXT NOT NULL REFERENCES accounts(id),
    security_id       TEXT NOT NULL REFERENCES securities(id),
    quantity          REAL NOT NULL,
    cost_basis        REAL,
    institution_price REAL,
    institution_value REAL,
    currency          TEXT NOT NULL DEFAULT 'CAD',
    updated_at        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(account_id, security_id)
);
