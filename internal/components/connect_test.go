package components_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"holo/internal/components"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConnectPage_UsesLocalStorage verifies the OAuth token is stored in localStorage,
// not sessionStorage. sessionStorage is window-scoped — Plaid's OAuth popup is a separate
// window and cannot read it. localStorage is shared across same-origin windows.
func TestConnectPage_UsesLocalStorage(t *testing.T) {
	var buf bytes.Buffer
	err := components.ConnectPage().Render(context.Background(), &buf)
	require.NoError(t, err)
	html := buf.String()

	assert.Contains(t, html, "localStorage.setItem('plaid_link_token'", "must store token in localStorage")
	assert.Contains(t, html, "localStorage.getItem('plaid_link_token'", "must read token from localStorage")
	assert.Contains(t, html, "localStorage.removeItem('plaid_link_token'", "must clean up token on success/exit")
	assert.NotContains(t, html, "sessionStorage", "sessionStorage breaks OAuth popup — must not appear anywhere")
}

// TestConnectPage_HandlesOAuthReturn verifies the connect page detects oauth_state_id
// in the URL and calls openPlaidLink with the current URL as receivedRedirectUri.
func TestConnectPage_HandlesOAuthReturn(t *testing.T) {
	var buf bytes.Buffer
	err := components.ConnectPage().Render(context.Background(), &buf)
	require.NoError(t, err)
	html := buf.String()

	assert.True(t,
		strings.Contains(html, "oauth_state_id") && strings.Contains(html, "openPlaidLink(window.location.href)"),
		"must detect oauth_state_id param and resume Plaid Link with receivedRedirectUri",
	)
}

// TestConnectPage_RelinkOAuthReturn verifies the connect page handles the case where
// the OAuth popup belongs to a relink flow (token stored under plaid_relink_token).
func TestConnectPage_RelinkOAuthReturn(t *testing.T) {
	var buf bytes.Buffer
	err := components.ConnectPage().Render(context.Background(), &buf)
	require.NoError(t, err)
	html := buf.String()

	assert.Contains(t, html, "plaid_relink_token", "must check for relink token on OAuth return")
	assert.Contains(t, html, "plaid_relink_institution_id", "must read relink institution ID on OAuth return")
	assert.Contains(t, html, "/api/plaid/relink-complete", "must call relink-complete (not exchange-token) for relink OAuth return")
}
