package handlers

import (
	"fmt"
	"net/http"

	"holo/internal/components"
	db "holo/internal/db/generated"
	plaid "github.com/plaid/plaid-go/v36/plaid"
	"github.com/go-chi/chi/v5"
)

// DebugHandler provides diagnostic endpoints for investigating sync issues.
type DebugHandler struct {
	queries *db.Queries
	api     *plaid.APIClient
}

func NewDebugHandler(queries *db.Queries, api *plaid.APIClient) *DebugHandler {
	return &DebugHandler{queries: queries, api: api}
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

	// Build a set of item_ids that have a non-null cursor.
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

// SyncAccounts is a stub for P2-P; lists accounts diagnostic info.
func (h *DebugHandler) SyncAccounts(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "not yet implemented (P2-P)", http.StatusNotImplemented)
}

// ReaddAccount is a stub for P2-P.
func (h *DebugHandler) ReaddAccount(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "not yet implemented (P2-P)", http.StatusNotImplemented)
}

// RemoveStaleAccount is a stub for P2-P.
func (h *DebugHandler) RemoveStaleAccount(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "not yet implemented (P2-P)", http.StatusNotImplemented)
}
