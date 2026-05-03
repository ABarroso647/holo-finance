package plaidclient

import (
	"fmt"
	"log"
	"os"

	plaid "github.com/plaid/plaid-go/v36/plaid"
)

func maskSecret(s string) string {
	if len(s) <= 6 {
		return "***"
	}
	return s[:4] + "…" + s[len(s)-2:]
}

func New() (*plaid.APIClient, error) {
	env := os.Getenv("PLAID_ENV")
	clientID := os.Getenv("PLAID_CLIENT_ID")
	secret := os.Getenv("PLAID_SECRET")

	if clientID == "" || secret == "" {
		return nil, fmt.Errorf("PLAID_CLIENT_ID and PLAID_SECRET must be set")
	}

	cfg := plaid.NewConfiguration()
	cfg.AddDefaultHeader("PLAID-CLIENT-ID", clientID)
	cfg.AddDefaultHeader("PLAID-SECRET", secret)

	if env == "production" {
		cfg.UseEnvironment(plaid.Production)
		log.Printf("plaid: using Production environment, client_id=%s", maskSecret(clientID))
	} else {
		cfg.UseEnvironment(plaid.Sandbox)
		log.Printf("plaid: using Sandbox environment, client_id=%s", maskSecret(clientID))
	}

	return plaid.NewAPIClient(cfg), nil
}
