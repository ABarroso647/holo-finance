package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/abarroso647/holo/internal/components"
	db "github.com/abarroso647/holo/internal/db/generated"
	"github.com/go-chi/chi/v5"
)

type AccountsHandler struct {
	queries *db.Queries
}

func NewAccountsHandler(queries *db.Queries) *AccountsHandler {
	return &AccountsHandler{queries: queries}
}

func (h *AccountsHandler) Page(w http.ResponseWriter, r *http.Request) {
	accounts, err := h.queries.ListAccountsWithInstitution(r.Context())
	if err != nil {
		http.Error(w, fmt.Sprintf("accounts: %v", err), http.StatusInternalServerError)
		return
	}
	components.AccountsPage(accounts).Render(r.Context(), w)
}

// GET /api/accounts/{id}/detail — returns expanded account detail HTML
func (h *AccountsHandler) Detail(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	since := time.Now().AddDate(0, 0, -30).Format("2006-01-02")

	topCats, _ := h.queries.GetTopCategoriesForAccount(r.Context(), db.GetTopCategoriesForAccountParams{
		AccountID: id,
		Date:      since,
	})

	txns, _ := h.queries.GetRecentTransactionsByAccount(r.Context(), db.GetRecentTransactionsByAccountParams{
		AccountID: id,
		Date:      since,
	})

	components.AccountDetail(topCats, txns).Render(r.Context(), w)
}
