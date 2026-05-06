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
