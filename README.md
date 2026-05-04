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

## Setup

    cp .env.example .env

| Variable | Required | Description |
|---|---|---|
| `PLAID_CLIENT_ID` | Yes | From your [Plaid dashboard](https://dashboard.plaid.com) |
| `PLAID_SECRET` | Yes | From your Plaid dashboard — use the sandbox or production secret |
| `PLAID_ENV` | Yes | `sandbox` for testing, `production` for real accounts |
| `OPENROUTER_API_KEY` | Yes | From [openrouter.ai](https://openrouter.ai) — used for LLM transaction categorization |
| `SESSION_SECRET` | Yes | Any random 32+ character string — signs auth session cookies |
| `DB_PATH` | No | Path to the SQLite database file. Defaults to `./holo.db` |
| `PORT` | No | Port to listen on. Defaults to `8080` |
| `WEBAUTHN_RPID` | No | Domain without protocol, e.g. `holo.example.com`. Defaults to `localhost` |
| `WEBAUTHN_RPORIGIN` | No | Full origin, e.g. `https://holo.example.com`. Defaults to `http://localhost:8080` |

## Running locally

    docker compose up --build

Open http://localhost:8080. On first run visit `/auth/register` to set up your passkey, then `/connect` to link your bank accounts.

For sandbox testing use `user_good` / `pass_good` in the Plaid Link flow.

## Deployment

On every push to `main`, GitHub Actions builds a multi-platform image and pushes it to GHCR.

On your server, copy `compose.prod.yaml` and a filled-in `.env`, then:

    docker compose -f compose.prod.yaml pull
    docker compose -f compose.prod.yaml up -d

## Contributing

See [DEVELOPMENT.md](DEVELOPMENT.md).
