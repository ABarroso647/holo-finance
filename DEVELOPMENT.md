# Development Guide

## Local dev

    docker compose up --build

The Docker build runs `templ generate` automatically before compiling, so you don't need to run it yourself. Rebuild after any code change:

    docker compose up --build

## Modifying templates

`.templ` files compile to `_templ.go` files which are committed to the repo. If you want IDE autocomplete and type-checking to reflect your changes while editing outside Docker, run:

    go install github.com/a-h/templ/cmd/templ@v0.3.1001
    templ generate ./...

The Docker build regenerates them anyway, so this is only needed for local IDE support.

## Modifying SQL schema or queries

Schema lives in `internal/db/schema.sql`. Queries live in `internal/db/queries/`. After changing either, regenerate the Go bindings:

    go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
    sqlc generate

Commit the generated files in `internal/db/generated/`.

Database migrations live in `internal/db/migrations/` and run automatically on startup via goose.

## Environment

Copy `.env.example` to `.env` and fill in your keys — see the README for what each variable does. For local dev, `PLAID_ENV=sandbox` and `user_good` / `pass_good` work as test credentials in the Plaid Link flow.
