package handlers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"holo/internal/categorize"
	"holo/internal/crypto"
	db "holo/internal/db/generated"
	plaidclient "holo/internal/plaid"
	"github.com/google/uuid"
	plaid "github.com/plaid/plaid-go/v36/plaid"
)

type PlaidHandler struct {
	api     *plaid.APIClient
	queries *db.Queries
	encKey  []byte
}

func NewPlaidHandler(api *plaid.APIClient, queries *db.Queries) *PlaidHandler {
	return &PlaidHandler{api: api, queries: queries, encKey: crypto.KeyFromEnv()}
}

// redirectURI returns the OAuth redirect URI to use in link token requests.
// PLAID_REDIRECT_URI env var takes priority; falls back to deriving from the request.
func (h *PlaidHandler) redirectURI(r *http.Request) string {
	if v := os.Getenv("PLAID_REDIRECT_URI"); v != "" {
		return v
	}
	scheme := r.Header.Get("X-Forwarded-Proto")
	if scheme == "" {
		scheme = "https" // default to https — Plaid requires it for OAuth
	}
	return scheme + "://" + r.Host + "/connect"
}

func (h *PlaidHandler) LinkToken(w http.ResponseWriter, r *http.Request) {
	user := plaid.NewLinkTokenCreateRequestUser("holo-user")
	req := plaid.NewLinkTokenCreateRequest("Holo", "en", []plaid.CountryCode{plaid.COUNTRYCODE_CA}, *user)
	req.SetProducts([]plaid.Products{
		plaid.PRODUCTS_TRANSACTIONS,
	})
	req.SetOptionalProducts([]plaid.Products{
		plaid.PRODUCTS_LIABILITIES,
		plaid.PRODUCTS_INVESTMENTS,
	})

	redirectURI := h.redirectURI(r)
	req.SetRedirectUri(redirectURI)
	log.Printf("plaid: link token redirect_uri=%s", redirectURI)

	resp, _, err := h.api.PlaidApi.LinkTokenCreate(r.Context()).LinkTokenCreateRequest(*req).Execute()
	if err != nil {
		msg := fmt.Sprintf("create link token: %v", err)
		if plaidErr, ok := err.(plaid.GenericOpenAPIError); ok {
			msg = fmt.Sprintf("create link token: %s", string(plaidErr.Body()))
		}
		log.Printf("ERROR: %s", msg)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"link_token": resp.GetLinkToken()})
}

func (h *PlaidHandler) ExchangeToken(w http.ResponseWriter, r *http.Request) {
	log.Printf("plaid: exchange-token called")
	var body struct {
		PublicToken     string `json:"public_token"`
		InstitutionName string `json:"institution_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	resp, _, err := h.api.PlaidApi.ItemPublicTokenExchange(r.Context()).
		ItemPublicTokenExchangeRequest(*plaid.NewItemPublicTokenExchangeRequest(body.PublicToken)).
		Execute()
	if err != nil {
		msg := fmt.Sprintf("exchange token: %v", err)
		if plaidErr, ok := err.(plaid.GenericOpenAPIError); ok {
			msg = fmt.Sprintf("exchange token: %s", string(plaidErr.Body()))
		}
		log.Printf("ERROR: %s", msg)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}
	log.Printf("plaid: exchange-token success, item_id=%s", resp.GetItemId())

	accessToken := resp.GetAccessToken()
	itemID := resp.GetItemId()

	encToken, err := crypto.Encrypt(h.encKey, accessToken)
	if err != nil {
		http.Error(w, fmt.Sprintf("encrypt token: %v", err), http.StatusInternalServerError)
		return
	}

	institution, err := h.queries.UpsertInstitution(r.Context(), db.UpsertInstitutionParams{
		ID:               uuid.New().String(),
		PlaidItemID:      itemID,
		PlaidAccessToken: encToken,
		Name:             body.InstitutionName,
	})
	if err != nil {
		http.Error(w, fmt.Sprintf("upsert institution: %v", err), http.StatusInternalServerError)
		return
	}

	if err := plaidclient.SyncAccounts(r.Context(), h.api, h.queries, institution.ID, accessToken); err != nil {
		http.Error(w, fmt.Sprintf("sync accounts: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok", "item_id": itemID})
}

func (h *PlaidHandler) Sync(w http.ResponseWriter, r *http.Request) {
	institutions, err := h.queries.ListInstitutions(r.Context())
	if err != nil {
		http.Error(w, fmt.Sprintf("list institutions: %v", err), http.StatusInternalServerError)
		return
	}

	type syncSummary struct {
		Name     string `json:"name"`
		Added    int    `json:"added"`
		Modified int    `json:"modified"`
		Removed  int    `json:"removed"`
		Error    string `json:"error,omitempty"`
	}
	results := make([]syncSummary, 0, len(institutions))

	for _, inst := range institutions {
		token, err := crypto.Decrypt(h.encKey, inst.PlaidAccessToken)
		if err != nil {
			results = append(results, syncSummary{Name: inst.Name, Error: "decrypt token: " + err.Error()})
			continue
		}
		result, err := plaidclient.SyncTransactions(r.Context(), h.api, h.queries, inst.PlaidItemID, token)
		s := syncSummary{Name: inst.Name, Added: result.Added, Modified: result.Modified, Removed: result.Removed}
		if err != nil {
			s.Error = err.Error()
		}
		results = append(results, s)

		if err := plaidclient.SyncRecurring(r.Context(), h.api, h.queries, inst.ID, token); err != nil {
			log.Printf("recurring sync for %s: %v", inst.Name, err)
		}
	}

	ruleCount, err := categorize.ApplyRules(r.Context(), h.queries)
	if err != nil {
		log.Printf("apply rules after sync: %v", err)
	}

	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("Content-Type", "text/html")
		w.Header().Set("HX-Trigger", "txnTableRefresh")
		for _, s := range results {
			if s.Error != "" {
				fmt.Fprintf(w, `<span style="color:var(--red)">%s: %s — <a href="/accounts" style="color:var(--red)">try Re-linking</a></span><br>`, s.Name, s.Error)
			} else {
				fmt.Fprintf(w, `<span style="color:var(--green)">%s: +%d added, %d modified, %d removed</span><br>`, s.Name, s.Added, s.Modified, s.Removed)
			}
		}
		if ruleCount > 0 {
			fmt.Fprintf(w, `<span style="color:var(--muted)">Rules applied: %d categorized</span><br>`, ruleCount)
		}
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func (h *PlaidHandler) SyncLiabilities(w http.ResponseWriter, r *http.Request) {
	institutions, err := h.queries.ListInstitutions(r.Context())
	if err != nil {
		http.Error(w, fmt.Sprintf("list institutions: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")

	if len(institutions) == 0 {
		fmt.Fprintf(w, `<span style="color:var(--muted);font-size:0.8rem">No institutions connected</span>`)
		return
	}

	var synced, failed int
	var firstErr string
	for _, inst := range institutions {
		token, err := crypto.Decrypt(h.encKey, inst.PlaidAccessToken)
		if err != nil {
			failed++
			if firstErr == "" {
				firstErr = "decrypt: " + err.Error()
			}
			continue
		}
		if err := plaidclient.SyncLiabilities(r.Context(), h.api, h.queries, token); err != nil {
			failed++
			if firstErr == "" {
				firstErr = err.Error()
			}
			log.Printf("liabilities sync failed for %s: %v", inst.Name, err)
		} else {
			synced++
		}
	}

	if failed > 0 && synced == 0 {
		fmt.Fprintf(w, `<span style="color:var(--yellow);font-size:0.8rem" title="%s">⚠ Some accounts don't support liabilities — try re-linking</span>`, firstErr)
	} else if failed > 0 {
		fmt.Fprintf(w, `<span style="color:var(--yellow);font-size:0.8rem">✓ Synced %d, %d accounts don't support liabilities</span>`, synced, failed)
	} else {
		fmt.Fprintf(w, `<span style="color:var(--green);font-size:0.8rem">✓ Payment data synced (%d institutions)</span>`, synced)
	}
}

func (h *PlaidHandler) SyncInvestments(w http.ResponseWriter, r *http.Request) {
	institutions, err := h.queries.ListInstitutions(r.Context())
	if err != nil {
		http.Error(w, fmt.Sprintf("list institutions: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")

	if len(institutions) == 0 {
		fmt.Fprintf(w, `<span style="color:var(--muted);font-size:0.8rem">No institutions connected</span>`)
		return
	}

	var total, failed int
	for _, inst := range institutions {
		token, err := crypto.Decrypt(h.encKey, inst.PlaidAccessToken)
		if err != nil {
			failed++
			log.Printf("sync investments: decrypt for %s: %v", inst.Name, err)
			continue
		}
		n, err := plaidclient.SyncInvestments(r.Context(), h.api, h.queries, token)
		if err != nil {
			failed++
			log.Printf("sync investments: %s: %v", inst.Name, err)
		} else {
			total += n
		}
	}

	if failed > 0 && total == 0 {
		fmt.Fprintf(w, `<span style="color:var(--yellow);font-size:0.8rem">⚠ Investments not available — enable Investments product in Plaid dashboard</span>`)
	} else if failed > 0 {
		fmt.Fprintf(w, `<span style="color:var(--yellow);font-size:0.8rem">⚠ %d holdings synced, %d institutions failed</span>`, total, failed)
	} else {
		fmt.Fprintf(w, `<span style="color:var(--green);font-size:0.8rem">✓ %d holdings synced</span>`, total)
	}
}

func (h *PlaidHandler) Webhook(w http.ResponseWriter, r *http.Request) {
	var payload map[string]any
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	raw, _ := json.Marshal(payload)
	itemID, _ := payload["item_id"].(string)
	webhookType, _ := payload["webhook_type"].(string)
	webhookCode, _ := payload["webhook_code"].(string)

	_ = logWebhook(r.Context(), h.queries, itemID, webhookType, webhookCode, string(raw))

	if webhookType == "TRANSACTIONS" {
		inst, err := h.queries.GetInstitutionByItemID(r.Context(), itemID)
		if err == nil {
			go func() {
				token, err := crypto.Decrypt(h.encKey, inst.PlaidAccessToken)
				if err != nil {
					log.Printf("webhook: decrypt token for %s: %v", inst.PlaidItemID, err)
					return
				}
				plaidclient.SyncTransactions(context.Background(), h.api, h.queries, inst.PlaidItemID, token) //nolint:errcheck
			}()
		}
	}

	w.WriteHeader(http.StatusOK)
}

// RelinkToken creates a Plaid update-mode link token for an existing institution.
// The frontend reuses this token to re-authenticate without creating a new institution.
func (h *PlaidHandler) RelinkToken(w http.ResponseWriter, r *http.Request) {
	var body struct {
		InstitutionID string `json:"institution_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.InstitutionID == "" {
		http.Error(w, "institution_id required", http.StatusBadRequest)
		return
	}

	inst, err := h.queries.GetInstitutionByID(r.Context(), body.InstitutionID)
	if err != nil {
		http.Error(w, "institution not found", http.StatusNotFound)
		return
	}

	accessToken, err := crypto.Decrypt(h.encKey, inst.PlaidAccessToken)
	if err != nil {
		http.Error(w, fmt.Sprintf("decrypt token: %v", err), http.StatusInternalServerError)
		return
	}

	user := plaid.NewLinkTokenCreateRequestUser("holo-user")
	req := plaid.NewLinkTokenCreateRequest("Holo", "en", []plaid.CountryCode{plaid.COUNTRYCODE_CA}, *user)
	req.SetAccessToken(accessToken)
	req.SetOptionalProducts([]plaid.Products{
		plaid.PRODUCTS_LIABILITIES,
		plaid.PRODUCTS_INVESTMENTS,
	})

	redirectURI := h.redirectURI(r)
	req.SetRedirectUri(redirectURI)
	log.Printf("plaid: relink token redirect_uri=%s", redirectURI)

	resp, _, err := h.api.PlaidApi.LinkTokenCreate(r.Context()).LinkTokenCreateRequest(*req).Execute()
	if err != nil {
		msg := fmt.Sprintf("create relink token: %v", err)
		if plaidErr, ok := err.(plaid.GenericOpenAPIError); ok {
			msg = fmt.Sprintf("create relink token: %s", string(plaidErr.Body()))
		}
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]string{"link_token": resp.GetLinkToken()})
}

// RelinkComplete re-syncs an institution after Plaid update-mode re-authentication.
// No token exchange needed — the access token is unchanged.
func (h *PlaidHandler) RelinkComplete(w http.ResponseWriter, r *http.Request) {
	var body struct {
		InstitutionID string `json:"institution_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.InstitutionID == "" {
		http.Error(w, "institution_id required", http.StatusBadRequest)
		return
	}

	inst, err := h.queries.GetInstitutionByID(r.Context(), body.InstitutionID)
	if err != nil {
		http.Error(w, "institution not found", http.StatusNotFound)
		return
	}

	token, err := crypto.Decrypt(h.encKey, inst.PlaidAccessToken)
	if err != nil {
		http.Error(w, fmt.Sprintf("decrypt token: %v", err), http.StatusInternalServerError)
		return
	}

	if _, err := plaidclient.SyncTransactions(r.Context(), h.api, h.queries, inst.PlaidItemID, token); err != nil {
		log.Printf("relink sync transactions for %s: %v", inst.Name, err)
	}
	if err := plaidclient.SyncLiabilities(r.Context(), h.api, h.queries, token); err != nil {
		log.Printf("relink sync liabilities for %s: %v", inst.Name, err)
	}
	if err := plaidclient.SyncRecurring(r.Context(), h.api, h.queries, inst.ID, token); err != nil {
		log.Printf("relink sync recurring for %s: %v", inst.Name, err)
	}

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, `<span style="color:var(--green);font-size:0.8rem">✓ Re-linked and synced</span>`)
}

// DisconnectInstitution removes an institution, its accounts, and all associated transactions from the DB,
// and calls Plaid ItemRemove to revoke the access token.
func (h *PlaidHandler) DisconnectInstitution(w http.ResponseWriter, r *http.Request) {
	var body struct {
		InstitutionID string `json:"institution_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.InstitutionID == "" {
		http.Error(w, "institution_id required", http.StatusBadRequest)
		return
	}

	inst, err := h.queries.GetInstitutionByID(r.Context(), body.InstitutionID)
	if err != nil {
		http.Error(w, "institution not found", http.StatusNotFound)
		return
	}

	token, err := crypto.Decrypt(h.encKey, inst.PlaidAccessToken)
	if err != nil {
		http.Error(w, "decrypt token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Tell Plaid to revoke access — best-effort, don't block on error.
	req := plaid.NewItemRemoveRequest(token)
	if _, _, err := h.api.PlaidApi.ItemRemove(r.Context()).ItemRemoveRequest(*req).Execute(); err != nil {
		log.Printf("plaid ItemRemove for %s: %v (continuing with local delete)", inst.Name, err)
	}

	// Delete in dependency order: transactions → accounts → institution.
	if err := h.queries.DeleteTransactionsByInstitution(r.Context(), body.InstitutionID); err != nil {
		http.Error(w, "delete transactions: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if err := h.queries.DeleteAccountsByInstitution(r.Context(), body.InstitutionID); err != nil {
		http.Error(w, "delete accounts: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if err := h.queries.DeleteInstitution(r.Context(), body.InstitutionID); err != nil {
		http.Error(w, "delete institution: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("disconnected institution %s (%s)", inst.Name, inst.ID)
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, `<span style="color:var(--green);font-size:0.8rem">✓ Disconnected</span>`)
}

// deduplicates retried webhooks via SHA-256 hash as ID (INSERT OR IGNORE).
func logWebhook(ctx context.Context, queries *db.Queries, itemID, webhookType, webhookCode, payload string) error {
	h := sha256.Sum256([]byte(payload))
	id := hex.EncodeToString(h[:])
	return queries.CreateWebhookEvent(ctx, db.CreateWebhookEventParams{
		ID:          id,
		ItemID:      &itemID,
		WebhookType: &webhookType,
		WebhookCode: &webhookCode,
		Payload:     &payload,
	})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}
