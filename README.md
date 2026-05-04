# Holo Personal Finance Tracker

Self-hosted personal finance dashboard. Connects to your Canadian bank accounts via Plaid, automatically categorizes transactions, and gives you a clean view of your spending, credit cards, and net worth.

## Features

- **Dashboard** — net worth snapshot, monthly income vs spending, top categories, recent transactions
- **Accounts** — balances across all linked institutions, expandable per-account transaction history
- **Transactions** — searchable, filterable transaction list with inline recategorization and tags
- **Spending** — spending by category, by card, and monthly trend charts
- **Cards** — credit card balances, utilization, payment due dates, and reward rates
- **Settings** — category management, card reward rates, friendly account names

Transactions are categorized automatically via a rules engine with an OpenRouter/DeepSeek LLM fallback for anything that doesn't match a rule.

## Environment variables

    cp .env.example .env

| Variable | Required | Description |
|---|---|---|
| `PLAID_CLIENT_ID` | Yes | From your [Plaid dashboard](https://dashboard.plaid.com) |
| `PLAID_SECRET` | Yes | From your Plaid dashboard — use the sandbox or production secret |
| `PLAID_ENV` | Yes | `sandbox` for testing, `production` for real accounts |
| `OPENROUTER_API_KEY` | Yes | From [openrouter.ai/keys](https://openrouter.ai/keys) — used for LLM transaction categorization |
| `OPENROUTER_MODEL` | No | Model to use. Defaults to `deepseek/deepseek-v4-flash` |
| `SESSION_SECRET` | Yes | Signs auth session cookies. Generate: `openssl rand -base64 32` |
| `ENCRYPTION_KEY` | No | 32-byte hex key encrypting Plaid tokens at rest. Derived from `SESSION_SECRET` if not set. Generate: `openssl rand -hex 32` |
| `DB_PATH` | No | Path to the SQLite database file. Defaults to `./holo.db` |
| `PORT` | Yes | Port to listen on |
| `WEBAUTHN_RPID` | Yes | Bare domain the passkey is registered against (no protocol, no port) |
| `WEBAUTHN_RPORIGINS` | Yes | Comma-separated list of full origins allowed to authenticate. Use `{port}` as a placeholder for `PORT` |
| `INVEST_BUFFER_CAD` | No | Monthly cash buffer to keep before recommending investment contributions. Defaults to `500` |

## Plaid setup

Holo requests three Plaid products when linking an account:

| Product | What it's used for |
|---|---|
| **Transactions** | Fetches and syncs transaction history |
| **Liabilities** | Fetches credit card balances, statements, minimum payments, and due dates |
| **Investments** | Fetches investment account holdings |

### Sandbox

1. Sign up at [dashboard.plaid.com](https://dashboard.plaid.com) — sandbox access is free and instant
2. Copy your **Sandbox** `client_id` and `secret` into `.env` and set `PLAID_ENV=sandbox`
3. When linking accounts in the app use `user_good` / `pass_good` as credentials in the Plaid Link flow

### Production

Production access requires Plaid to review and approve your application:

1. In the Plaid dashboard go to **Team Settings → Products** and request access to **Transactions**, **Liabilities**, and **Investments**
2. Submit the application — describe it as a personal self-hosted finance tracker for your own accounts
3. Once approved, copy your **Production** `client_id` and `secret` from **Team Settings → Keys**
4. Set `PLAID_ENV=production` in your `.env`
5. For OAuth-based institutions (some major banks), add your domain to **API → Allowed redirect URIs** — the URI should be `https://yourdomain.com/connect`
6. Re-link your accounts through the app — sandbox tokens do not carry over to production

## OpenRouter setup

OpenRouter is used as the LLM fallback for transactions that don't match any categorization rule.

1. Sign up at [openrouter.ai](https://openrouter.ai) and generate an API key under **Keys**
2. Add a small credit balance — categorization uses a cheap model by default and costs fractions of a cent per transaction
3. Set `OPENROUTER_API_KEY` in your `.env`
4. Optionally override the model with `OPENROUTER_MODEL`. The default (`deepseek/deepseek-v4-flash`) is fast and cheap

If `OPENROUTER_API_KEY` is not set, transactions that don't match a rule will remain uncategorized rather than erroring.

## First-time setup

1. Copy and fill in your `.env`:
   ```
   cp .env.example .env
   ```

2. Start the app:
   ```
   docker compose up --build
   ```

3. Navigate to `/auth/register` and register your passkey. This is the only account — Holo is single-user.

4. Go to `/connect` and link your first bank account through the Plaid Link flow.

5. After linking, navigate to any page — a sync runs automatically on first connect. Initial sync may take a moment depending on how much history Plaid returns.

6. Transactions will be auto-categorized on sync. Correct any misses inline from the Transactions page and create rules so future transactions are categorized automatically.

## Running locally

    docker compose up --build

Open http://localhost:PORT.

## Deployment

On every push to `master`, GitHub Actions builds a multi-platform image and pushes it to GHCR.

On your server, copy `compose.prod.yaml` and a filled-in `.env`, then:

    docker compose -f compose.prod.yaml pull
    docker compose -f compose.prod.yaml up -d

## Contributing

See [DEVELOPMENT.md](DEVELOPMENT.md).
