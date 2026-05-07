package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"holo/internal/testutil"
	plaidclient "holo/internal/plaid"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	// Load .env from project root so integration tests pick up credentials automatically.
	// Walks up from the handlers package directory.
	for _, path := range []string{".env", "../.env", "../../.env", "../../../.env"} {
		if godotenv.Load(path) == nil {
			break
		}
	}
}

// TestLinkToken_PlaidSandbox_ReturnsToken calls the real Plaid API to verify:
//  1. Our credentials are valid
//  2. The redirect URI (derived from hostname) is accepted by Plaid
//  3. A usable link_token is returned
//
// Loads credentials from .env automatically. Skipped if PLAID_CLIENT_ID is not set.
// Run: go test ./internal/handlers/ -run TestLinkToken_PlaidSandbox -v
func TestLinkToken_PlaidSandbox_ReturnsToken(t *testing.T) {
	if os.Getenv("PLAID_CLIENT_ID") == "" || os.Getenv("PLAID_SECRET") == "" {
		t.Skip("PLAID_CLIENT_ID / PLAID_SECRET not set — skipping Plaid integration test")
	}

	api, err := plaidclient.New()
	require.NoError(t, err, "failed to build Plaid client — check PLAID_CLIENT_ID and PLAID_SECRET")

	q := testutil.NewDB(t)
	h := &PlaidHandler{api: api, queries: q}

	// https://holo.abarroso.ca/connect must be registered in Plaid Dashboard → API → Allowed redirect URIs.
	redirectURI := "https://holo.abarroso.ca/connect"
	req := httptest.NewRequest(http.MethodPost, "/api/plaid/link-token", strings.NewReader(""))
	req.Host = "holo.abarroso.ca"
	req.Header.Set("X-Forwarded-Proto", "https")
	rr := httptest.NewRecorder()

	h.LinkToken(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("Plaid rejected link token (status %d).\nBody: %s\n\nMake sure %q is registered in Plaid Dashboard → API → Allowed redirect URIs", rr.Code, rr.Body.String(), redirectURI)
	}

	var resp map[string]string
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	token := resp["link_token"]
	assert.NotEmpty(t, token, "link_token must be present in response")
	assert.True(t, strings.HasPrefix(token, "link-"), "link_token must start with 'link-', got: %q", token)

	t.Logf("✓ Plaid accepted redirect URI %q and returned link token: %s…", redirectURI, token[:min(len(token), 24)])
}

// TestRedirectURI_RealHostDefaultsToHTTPS verifies non-localhost hosts default to https.
// This was the original Simplii bug — the old code defaulted to http when no header was set.
func TestRedirectURI_RealHostDefaultsToHTTPS(t *testing.T) {
	h := newTestPlaidHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Host = "holo.abarroso.ca"
	// No X-Forwarded-Proto header
	assert.Equal(t, "https://holo.abarroso.ca/connect", h.redirectURI(req))
}

// TestRedirectURI_LocalhostUsesHTTP verifies localhost uses http (Plaid sandbox allows it).
func TestRedirectURI_LocalhostUsesHTTP(t *testing.T) {
	h := newTestPlaidHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Host = "localhost:8080"
	assert.Equal(t, "http://localhost:8080/connect", h.redirectURI(req))
}

// TestRedirectURI_ProxyHeaderWins verifies X-Forwarded-Proto is respected.
func TestRedirectURI_ProxyHeaderWins(t *testing.T) {
	h := newTestPlaidHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Host = "holo.abarroso.ca"
	req.Header.Set("X-Forwarded-Proto", "https")
	assert.Equal(t, "https://holo.abarroso.ca/connect", h.redirectURI(req))
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
