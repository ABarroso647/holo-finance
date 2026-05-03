  # Holo Personal Finance Tracker

  Self-hosted personal finance dashboard. Connects to your Canadian bank accounts via Plaid, automatically categorizes transactions, and gives you a clean view of your spending, credit cards, and net worth.

  Self-hosted personal finance dashboard. Connects to your Canadian bank accounts via Plaid, automatically categorizes transactions, and gives you a clean view of your spending, credit cards, and net worth.

  ## Features

  - **Dashboard** — net worth snapshot, monthly income vs spending, top categories, recent transactions
  - **Accounts** — balances across all linked institutions, expandable per-account transaction history
  - **Transactions** — searchable, filterable transaction list with inline recategorization and tags
  - **Spending** — spending by category, by card, and monthly trend charts
  - **Cards** — credit card balances, utilization, payment due dates, and reward rates
  - **Settings** — category management, card reward rates, friendly account names

  Transactions are categorized automatically via a rules engine with an OpenRouter/DeepSeek LLM fallback for anything that doesn't match a rule.

  ## Requirements

  - Go 1.22+
  - Plaid account (sandbox is free)
  - OpenRouter API key (for LLM categorization)

  Install codegen tools once:

      go install github.com/a-h/templ/cmd/templ@latest
      go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
      go install github.com/pressly/goose/v3/cmd/goose@latest

  ## Setup

      cp .env.example .env

  Fill in .env:

      PLAID_CLIENT_ID=
      PLAID_SECRET=
      PLAID_ENV=sandbox
      OPENROUTER_API_KEY=
      SESSION_SECRET=
      DB_PATH=./holo.db
      PORT=8080
      WEBAUTHN_RPID=localhost
      WEBAUTHN_RPORIGIN=http://localhost:8080

  ## Run

      go run ./cmd/holo

  Open http://localhost:8080. On first run visit /auth/register to set up your passkey, then /connect to link your bank accounts.

  For sandbox testing use user_good / pass_good in the Plaid Link flow.

  ## Docker

      docker compose up

  ## Development

  After editing .templ files:

      templ generate ./...

  After editing SQL schema or queries:

      sqlc generate
