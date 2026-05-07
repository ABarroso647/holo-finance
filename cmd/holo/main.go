package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"

	"holo/internal/auth"
	"holo/internal/categorize"
	"holo/internal/crypto"
	"holo/internal/db"
	dbgen "holo/internal/db/generated"
	"holo/internal/handlers"
	plaidclient "holo/internal/plaid"
	holoStatic "holo/static"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/gorilla/sessions"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./holo.db"
	}

	database, err := db.Open(dbPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer database.Close()

	queries := dbgen.New(database)

	// Re-encrypt any plaintext Plaid access tokens written before encryption was added.
	reencryptTokens(context.Background(), queries)

	if err := categorize.SeedCategories(context.Background(), queries); err != nil {
		log.Fatalf("seed categories: %v", err)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	secret := os.Getenv("SESSION_SECRET")
	if secret == "" {
		secret = "dev-secret-change-in-production"
		log.Printf("WARNING: SESSION_SECRET not set — using insecure default")
	}
	store := sessions.NewCookieStore([]byte(secret))

	authHandler, err := auth.New(queries, store, port)
	if err != nil {
		log.Fatalf("auth init: %v", err)
	}

	api, err := plaidclient.New()
	if err != nil {
		log.Printf("warning: plaid not configured: %v", err)
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Permissions-Policy", "accelerometer=*, encrypted-media=*")
			next.ServeHTTP(w, r)
		})
	})

	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.FS(holoStatic.FS))))

	r.Get("/auth/register", authHandler.RegisterPage)
	r.Post("/auth/register/begin", authHandler.BeginRegistration)
	r.Post("/auth/register/finish", authHandler.FinishRegistration)
	r.Get("/auth/login", authHandler.LoginPage)
	r.Post("/auth/login/begin", authHandler.BeginLogin)
	r.Post("/auth/login/finish", authHandler.FinishLogin)
	r.Post("/auth/logout", authHandler.Logout)

	r.Group(func(r chi.Router) {
		r.Use(auth.RequireAuth(store))

		dashHandler := handlers.NewDashboardHandler(queries)
		r.Get("/", dashHandler.Page)
		r.Get("/api/dashboard/category-txns", dashHandler.CategoryTxns)

		acctHandler := handlers.NewAccountsHandler(queries)
		r.Get("/accounts", acctHandler.Page)
		r.Get("/api/accounts/{id}/detail", acctHandler.Detail)

		spendHandler := handlers.NewSpendingHandler(queries)
		r.Get("/spending", spendHandler.Page)

		r.Get("/connect", handlers.ConnectPage)

		cardsHandler := handlers.NewCardsHandler(queries)
		r.Get("/cards", cardsHandler.Page)
		r.Post("/api/cards/{id}/fetch-rates", cardsHandler.FetchRates)
		r.Post("/api/cards/{id}/rates", cardsHandler.AddRate)
		r.Delete("/api/cards/{id}/rates/{rate_id}", cardsHandler.DeleteRate)
		r.Post("/api/cards/rematch-rates", cardsHandler.RematchRates)

		investHandler := handlers.NewInvestHandler(queries)
		r.Get("/invest", investHandler.Page)
		r.Post("/api/invest/buffer", investHandler.UpdateBuffer)

		exportHandler := handlers.NewExportHandler(queries)
		r.Get("/api/export", exportHandler.Export)

		txnHandler := handlers.NewTransactionHandler(queries)
		r.Get("/transactions", txnHandler.List)

		catHandler := handlers.NewCategorizeHandler(queries)
		r.Post("/api/categorize/run", catHandler.Run)
		r.Post("/api/transactions/{id}/category", catHandler.UpdateCategory)
		r.Post("/api/rules", catHandler.CreateRule)

		tagHandler := handlers.NewTagHandler(queries)
		r.Post("/api/transactions/{id}/tags", tagHandler.Add)
		r.Delete("/api/transactions/{id}/tags/{tag_id}", tagHandler.Remove)
		r.Post("/api/tags", tagHandler.Create)
		r.Delete("/api/tags/{id}", tagHandler.Delete)

		settingsHandler := handlers.NewSettingsHandler(queries)
		r.Get("/settings", settingsHandler.Page)
		r.Post("/api/accounts/{id}/display-name", settingsHandler.UpdateDisplayName)
		r.Post("/api/categories/{id}", settingsHandler.UpdateCategory)
		r.Post("/api/settings/openrouter-model", settingsHandler.UpdateModel)
		r.Delete("/api/rules/{id}", settingsHandler.DeleteRule)

		if api != nil {
			plaidHandler := handlers.NewPlaidHandler(api, queries)
			r.Post("/api/plaid/link-token", plaidHandler.LinkToken)
			r.Post("/api/plaid/exchange-token", plaidHandler.ExchangeToken)
			r.Post("/api/plaid/sync", plaidHandler.Sync)
			r.Post("/api/plaid/sync-liabilities", plaidHandler.SyncLiabilities)
			r.Post("/api/plaid/sync-investments", plaidHandler.SyncInvestments)
			r.Post("/api/plaid/relink-token", plaidHandler.RelinkToken)
			r.Post("/api/plaid/relink-complete", plaidHandler.RelinkComplete)
			r.Post("/api/plaid/disconnect", plaidHandler.DisconnectInstitution)
			r.Post("/api/accounts/{id}/remove", plaidHandler.RemoveAccount)
			r.Post("/api/plaid/webhook", plaidHandler.Webhook)
		}
	})

	log.Printf("starting holo on :%s", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatalf("server: %v", err)
	}
}

// reencryptTokens upgrades any plaintext Plaid access tokens stored before encryption was added.
func reencryptTokens(ctx context.Context, queries *dbgen.Queries) {
	encKey := crypto.KeyFromEnv()
	institutions, err := queries.ListInstitutions(ctx)
	if err != nil {
		log.Printf("reencrypt: list institutions: %v", err)
		return
	}
	upgraded := 0
	for _, inst := range institutions {
		if strings.HasPrefix(inst.PlaidAccessToken, "enc:v1:") {
			continue // already encrypted
		}
		enc, err := crypto.Encrypt(encKey, inst.PlaidAccessToken)
		if err != nil {
			log.Printf("reencrypt: encrypt for %s: %v", inst.Name, err)
			continue
		}
		if err := queries.UpdateInstitutionToken(ctx, dbgen.UpdateInstitutionTokenParams{
			ID:               inst.ID,
			PlaidAccessToken: enc,
		}); err != nil {
			log.Printf("reencrypt: update for %s: %v", inst.Name, err)
			continue
		}
		upgraded++
	}
	if upgraded > 0 {
		log.Printf("reencrypt: upgraded %d plaintext token(s) to AES-256-GCM", upgraded)
	}
}
