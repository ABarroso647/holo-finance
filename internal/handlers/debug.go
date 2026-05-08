package handlers

import (
	"fmt"
	"net/http"

	"holo/internal/components"
	"holo/internal/crypto"
	db "holo/internal/db/generated"
	plaidclient "holo/internal/plaid"
	"github.com/go-chi/chi/v5"
	plaid "github.com/plaid/plaid-go/v36/plaid"
)

type DebugHandler struct {
	queries *db.Queries
	api     *plaid.APIClient
	encKey  []byte
}

func NewDebugHandler(queries *db.Queries, api *plaid.APIClient) *DebugHandler {
	return &DebugHandler{queries: queries, api: api, encKey: crypto.KeyFromEnv()}
}

// SyncHistory renders a diagnostic page showing per-institution sync stats and cursor status.
func (h *DebugHandler) SyncHistory(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	stats, err := h.queries.GetInstitutionSyncStats(ctx)
	if err != nil {
		http.Error(w, fmt.Sprintf("query stats: %v", err), http.StatusInternalServerError)
		return
	}

	cursors, err := h.queries.ListAllCursors(ctx)
	if err != nil {
		http.Error(w, fmt.Sprintf("query cursors: %v", err), http.StatusInternalServerError)
		return
	}

	cursorSet := make(map[string]bool, len(cursors))
	for _, c := range cursors {
		if c.Cursor != nil && *c.Cursor != "" {
			cursorSet[c.ItemID] = true
		}
	}

	var rows []components.InstitutionSyncStat
	for _, s := range stats {
		earliest := ""
		if s.EarliestDate != nil {
			earliest = fmt.Sprintf("%v", s.EarliestDate)
		}
		latest := ""
		if s.LatestDate != nil {
			latest = fmt.Sprintf("%v", s.LatestDate)
		}
		rows = append(rows, components.InstitutionSyncStat{
			Name:         s.Name,
			ItemID:       s.PlaidItemID,
			EarliestDate: earliest,
			LatestDate:   latest,
			TotalTxns:    s.TotalTxns,
			AccountCount: s.AccountCount,
			HasCursor:    cursorSet[s.PlaidItemID],
		})
	}

	data := components.SyncHistoryData{Institutions: rows}
	components.DebugSyncHistory(data).Render(ctx, w)
}

// ResetCursor nullifies the Plaid cursor for the given item_id, forcing a full re-sync.
func (h *DebugHandler) ResetCursor(w http.ResponseWriter, r *http.Request) {
	itemID := chi.URLParam(r, "item_id")
	if itemID == "" {
		http.Error(w, "missing item_id", http.StatusBadRequest)
		return
	}

	if err := h.queries.ResetCursor(r.Context(), itemID); err != nil {
		http.Error(w, fmt.Sprintf("reset cursor: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, `<span style="color:var(--green)">Cursor reset — next sync will pull full history.</span>`)
}

// SyncAccounts cross-references the local DB against live Plaid data, surfacing
// ghost accounts (in Plaid but not in DB) and stale accounts (in DB but not in Plaid).
func (h *DebugHandler) SyncAccounts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	institutions, err := h.queries.ListInstitutions(ctx)
	if err != nil {
		http.Error(w, "list institutions: "+err.Error(), http.StatusInternalServerError)
		return
	}

	localRows, err := h.queries.ListAccountsForDebug(ctx)
	if err != nil {
		http.Error(w, "list accounts: "+err.Error(), http.StatusInternalServerError)
		return
	}

	type localAcct struct {
		localID        string
		plaidAccountID string
		name           string
		acctType       string
	}
	byInst := make(map[string][]localAcct)
	for _, row := range localRows {
		byInst[row.InstitutionID] = append(byInst[row.InstitutionID], localAcct{
			localID:        row.ID,
			plaidAccountID: row.PlaidAccountID,
			name:           row.Name,
			acctType:       row.Type,
		})
	}

	var data components.SyncAccountsData

	for _, inst := range institutions {
		diff := components.InstitutionAccountDiff{Name: inst.Name}

		token, err := crypto.Decrypt(h.encKey, inst.PlaidAccessToken)
		if err != nil {
			diff.Error = "decrypt token: " + err.Error()
			data.Institutions = append(data.Institutions, diff)
			continue
		}

		req := plaid.NewAccountsGetRequest(token)
		resp, _, err := h.api.PlaidApi.AccountsGet(ctx).AccountsGetRequest(*req).Execute()
		if err != nil {
			diff.Error = "plaid accounts get: " + err.Error()
			data.Institutions = append(data.Institutions, diff)
			continue
		}

		plaidAccounts := resp.GetAccounts()
		plaidIDs := make(map[string]plaid.AccountBase, len(plaidAccounts))
		for _, pa := range plaidAccounts {
			plaidIDs[pa.GetAccountId()] = pa
		}

		localAccts := byInst[inst.ID]
		localIDSet := make(map[string]localAcct, len(localAccts))
		for _, la := range localAccts {
			localIDSet[la.plaidAccountID] = la
		}

		for _, pa := range plaidAccounts {
			pid := pa.GetAccountId()
			mask := pa.GetMask()
			acctType := string(pa.GetType())
			name := pa.GetName()
			if la, ok := localIDSet[pid]; ok {
				diff.Accounts = append(diff.Accounts, components.AccountDiff{
					PlaidAccountID: pid,
					Name:           name,
					Mask:           mask,
					Type:           acctType,
					Status:         "healthy",
					LocalID:        la.localID,
				})
			} else {
				diff.Accounts = append(diff.Accounts, components.AccountDiff{
					PlaidAccountID: pid,
					Name:           name,
					Mask:           mask,
					Type:           acctType,
					Status:         "ghost",
					LocalID:        "",
				})
			}
		}

		for _, la := range localAccts {
			if _, ok := plaidIDs[la.plaidAccountID]; !ok {
				diff.Accounts = append(diff.Accounts, components.AccountDiff{
					PlaidAccountID: la.plaidAccountID,
					Name:           la.name,
					Mask:           "",
					Type:           la.acctType,
					Status:         "stale",
					LocalID:        la.localID,
				})
			}
		}

		data.Institutions = append(data.Institutions, diff)
	}

	components.DebugSyncAccounts(data).Render(ctx, w)
}

// ReaddAccount re-inserts a ghost account (in Plaid but missing from DB) using SyncAccounts.
func (h *DebugHandler) ReaddAccount(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	plaidAccountID := chi.URLParam(r, "plaid_account_id")
	if plaidAccountID == "" {
		http.Error(w, "plaid_account_id required", http.StatusBadRequest)
		return
	}

	institutions, err := h.queries.ListInstitutions(ctx)
	if err != nil {
		http.Error(w, "list institutions: "+err.Error(), http.StatusInternalServerError)
		return
	}

	for _, inst := range institutions {
		token, err := crypto.Decrypt(h.encKey, inst.PlaidAccessToken)
		if err != nil {
			continue
		}

		req := plaid.NewAccountsGetRequest(token)
		resp, _, err := h.api.PlaidApi.AccountsGet(ctx).AccountsGetRequest(*req).Execute()
		if err != nil {
			continue
		}

		for _, pa := range resp.GetAccounts() {
			if pa.GetAccountId() != plaidAccountID {
				continue
			}

			if err := plaidclient.SyncAccounts(ctx, h.api, h.queries, inst.ID, token); err != nil {
				http.Error(w, "re-sync accounts: "+err.Error(), http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "text/html")
			fmt.Fprintf(w, `<span style="color:var(--green);font-size:0.85rem">✓ Re-added %s to DB</span>`, pa.GetName())
			return
		}
	}

	http.Error(w, "plaid account not found across any institution", http.StatusNotFound)
}

// RemoveStaleAccount deletes a stale local account (not in Plaid) from the DB.
func (h *DebugHandler) RemoveStaleAccount(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	accountID := chi.URLParam(r, "account_id")
	if accountID == "" {
		http.Error(w, "account_id required", http.StatusBadRequest)
		return
	}

	if err := h.queries.DeleteAccountByID(ctx, accountID); err != nil {
		http.Error(w, "delete account: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, `<span style="color:var(--green);font-size:0.85rem">✓ Account removed from DB</span>`)
}
