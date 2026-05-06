package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"holo/internal/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestPlaidHandler(t *testing.T) *PlaidHandler {
	t.Helper()
	q := testutil.NewDB(t)
	return &PlaidHandler{queries: q}
}

func TestRelinkToken_Returns400_WithoutInstitutionID(t *testing.T) {
	h := newTestPlaidHandler(t)
	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/api/plaid/relink-token", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.RelinkToken(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestRelinkToken_Returns404_WithUnknownInstitution(t *testing.T) {
	h := newTestPlaidHandler(t)
	body := `{"institution_id": "unknown-inst-id"}`
	req := httptest.NewRequest(http.MethodPost, "/api/plaid/relink-token", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.RelinkToken(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestRelinkComplete_Returns400_WithoutInstitutionID(t *testing.T) {
	h := newTestPlaidHandler(t)
	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/api/plaid/relink-complete", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.RelinkComplete(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestDisconnect_Returns400_WithoutInstitutionID(t *testing.T) {
	h := newTestPlaidHandler(t)
	req := httptest.NewRequest(http.MethodPost, "/api/plaid/disconnect", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.DisconnectInstitution(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestDisconnect_Returns404_WithUnknownInstitution(t *testing.T) {
	h := newTestPlaidHandler(t)
	req := httptest.NewRequest(http.MethodPost, "/api/plaid/disconnect", strings.NewReader(`{"institution_id":"nope"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.DisconnectInstitution(rr, req)
	assert.Equal(t, http.StatusNotFound, rr.Code)
}

// TestRedirectURI_UsesEnvVar verifies PLAID_REDIRECT_URI overrides header-based detection.
func TestRedirectURI_UsesEnvVar(t *testing.T) {
	h := newTestPlaidHandler(t)
	t.Setenv("PLAID_REDIRECT_URI", "https://example.com/connect")
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-Proto", "http")
	req.Host = "localhost:8080"
	assert.Equal(t, "https://example.com/connect", h.redirectURI(req))
}

// TestRedirectURI_DefaultsToHTTPS verifies we default to https (not http) when no header is set.
func TestRedirectURI_DefaultsToHTTPS(t *testing.T) {
	h := newTestPlaidHandler(t)
	t.Setenv("PLAID_REDIRECT_URI", "")
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Host = "finance.example.com"
	// No X-Forwarded-Proto header
	uri := h.redirectURI(req)
	assert.Equal(t, "https://finance.example.com/connect", uri)
}
