package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"holo/internal/components"
	db "holo/internal/db/generated"
	"holo/internal/rewards"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type CardsHandler struct {
	queries *db.Queries
}

func NewCardsHandler(queries *db.Queries) *CardsHandler {
	return &CardsHandler{queries: queries}
}

func (h *CardsHandler) Page(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	accounts, err := h.queries.ListAccountsWithInstitution(ctx)
	if err != nil {
		http.Error(w, "failed to load accounts", http.StatusInternalServerError)
		return
	}

	var cards []components.CardSummary
	since30 := time.Now().AddDate(0, 0, -30).Format("2006-01-02")

	for _, acct := range accounts {
		if acct.Type != "credit" {
			continue
		}

		card := components.CardSummary{Account: acct}

		if liab, err := h.queries.GetCreditCardLiability(ctx, acct.ID); err == nil {
			card.Liability = &liab
		}

		spendSince := since30
		spendLabel := "last 30 days"
		if card.Liability != nil && card.Liability.LastStatementIssueDate != nil {
			spendSince = *card.Liability.LastStatementIssueDate
			spendLabel = "since " + formatShortDate(*card.Liability.LastStatementIssueDate)
		}
		card.SpendLabel = spendLabel
		card.SpendTotal, _ = h.queries.GetAccountSpendSince(ctx, db.GetAccountSpendSinceParams{
			AccountID: acct.ID,
			Date:      spendSince,
		})

		card.TopCats, _ = h.queries.GetTopCategoriesForAccount(ctx, db.GetTopCategoriesForAccountParams{
			AccountID: acct.ID,
			Date:      spendSince,
		})

		card.Rates, _ = h.queries.ListCardRewardRatesForAccount(ctx, acct.ID)

		cards = append(cards, card)
	}

	components.CardsPage(cards).Render(ctx, w)
}

func (h *CardsHandler) FetchRates(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	cardName := r.FormValue("card_name")
	view := r.FormValue("view")

	if cardName == "" {
		accounts, _ := h.queries.ListAccountsWithInstitution(r.Context())
		for _, a := range accounts {
			if a.ID == id {
				cardName = a.Name
				break
			}
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), 50*time.Second)
	defer cancel()

	w.Header().Set("Content-Type", "text/html")
	existingRates, _ := h.queries.ListCardRewardRatesForAccount(r.Context(), id)
	if err := rewards.FetchAndStoreRates(ctx, h.queries, id, cardName); err != nil {
		if view == "settings" {
			categories, _ := h.queries.ListCategories(r.Context())
			components.SettingsCardRatesContent(id, existingRates, categories, err.Error()).Render(r.Context(), w)
		} else {
			components.CardRatesTable(id, existingRates, err.Error(), "").Render(r.Context(), w)
		}
		return
	}

	rates, _ := h.queries.ListCardRewardRatesForAccount(r.Context(), id)
	if view == "settings" {
		categories, _ := h.queries.ListCategories(r.Context())
		components.SettingsCardRatesContent(id, rates, categories, "").Render(r.Context(), w)
	} else {
		components.CardRatesTable(id, rates, "", "").Render(r.Context(), w)
	}
}

func (h *CardsHandler) AddRate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	view := r.FormValue("view")

	rateStr := r.FormValue("rate")
	rate, err := strconv.ParseFloat(rateStr, 64)
	if err != nil || rate < 0 {
		http.Error(w, "invalid rate", http.StatusBadRequest)
		return
	}

	categoryID := r.FormValue("category_id")
	var catPtr *string
	if categoryID != "" {
		catPtr = &categoryID
	}

	capStr := r.FormValue("cap_amount")
	var capPtr *float64
	if capStr != "" {
		if cap, err := strconv.ParseFloat(capStr, 64); err == nil && cap > 0 {
			capPtr = &cap
		}
	}

	capPeriod := r.FormValue("cap_period")
	var capPeriodPtr *string
	if capPeriod != "" {
		capPeriodPtr = &capPeriod
	}

	rawCat := r.FormValue("raw_category")
	var rawCatPtr *string
	if rawCat != "" {
		rawCatPtr = &rawCat
	}

	w.Header().Set("Content-Type", "text/html")
	if _, err := h.queries.UpsertCardRewardRate(r.Context(), db.UpsertCardRewardRateParams{
		ID:         uuid.New().String(),
		AccountID:  id,
		CategoryID: catPtr,
		RawCategory: rawCatPtr,
		RewardRate: rate,
		CapAmount:  capPtr,
		CapPeriod:  capPeriodPtr,
		Notes:      nil,
	}); err != nil {
		fmt.Fprintf(w, `<span style="color:var(--red);font-size:0.8rem">Error: %s</span>`, err.Error())
		return
	}

	rates, _ := h.queries.ListCardRewardRatesForAccount(r.Context(), id)
	if view == "settings" {
		categories, _ := h.queries.ListCategories(r.Context())
		components.SettingsCardRatesContent(id, rates, categories, "").Render(r.Context(), w)
	} else {
		components.CardRatesTable(id, rates, "", "").Render(r.Context(), w)
	}
}

func (h *CardsHandler) DeleteRate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	rateID := chi.URLParam(r, "rate_id")
	view := r.URL.Query().Get("view")

	w.Header().Set("Content-Type", "text/html")
	if err := h.queries.DeleteCardRewardRate(r.Context(), rateID); err != nil {
		fmt.Fprintf(w, `<span style="color:var(--red);font-size:0.8rem">Error: %s</span>`, err.Error())
		return
	}

	rates, _ := h.queries.ListCardRewardRatesForAccount(r.Context(), id)
	if view == "settings" {
		categories, _ := h.queries.ListCategories(r.Context())
		components.SettingsCardRatesContent(id, rates, categories, "").Render(r.Context(), w)
	} else {
		components.CardRatesTable(id, rates, "", "").Render(r.Context(), w)
	}
}

// RematchRates re-runs tier 1/2 matching on all unmatched rates.
func (h *CardsHandler) RematchRates(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	n, err := rewards.RematchRates(r.Context(), h.queries)
	if err != nil {
		fmt.Fprintf(w, `<span style="color:var(--red);font-size:0.8rem">Error: %s</span>`, err.Error())
		return
	}
	if n == 0 {
		fmt.Fprintf(w, `<span style="color:var(--muted);font-size:0.8rem">No unmatched rates to re-match.</span>`)
		return
	}
	fmt.Fprintf(w, `<span style="color:var(--green);font-size:0.8rem">✓ Matched %d rates to categories. Refresh page to see updates.</span>`, n)
}

func formatShortDate(date string) string {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return date
	}
	return t.Format("Jan 2")
}
