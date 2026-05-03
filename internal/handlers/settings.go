package handlers

import (
	"fmt"
	"net/http"
	"os"

	"github.com/abarroso647/holo/internal/components"
	db "github.com/abarroso647/holo/internal/db/generated"
	"github.com/go-chi/chi/v5"
)

type SettingsHandler struct {
	queries *db.Queries
}

func NewSettingsHandler(queries *db.Queries) *SettingsHandler {
	return &SettingsHandler{queries: queries}
}

func (h *SettingsHandler) Page(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	accounts, _ := h.queries.ListAccountsWithInstitution(ctx)
	categories, _ := h.queries.ListCategories(ctx)
	rules, _ := h.queries.ListRules(ctx)

	model, err := h.queries.GetSetting(ctx, "openrouter_model")
	if err != nil || model == "" {
		model = os.Getenv("OPENROUTER_MODEL")
	}

	// Build card rates data for credit accounts
	var cardRates []components.CardRatesData
	for _, a := range accounts {
		if a.Type != "credit" {
			continue
		}
		rates, _ := h.queries.ListCardRewardRatesForAccount(ctx, a.ID)
		cardRates = append(cardRates, components.CardRatesData{
			Account: a,
			Rates:   rates,
		})
	}

	tags, _ := h.queries.ListTags(ctx)
	spendByTag, _ := h.queries.GetSpendByTag(ctx, db.GetSpendByTagParams{DateFrom: "", DateTo: ""})

	components.SettingsPage(accounts, categories, rules, model, cardRates, tags, spendByTag).Render(ctx, w)
}

func (h *SettingsHandler) UpdateDisplayName(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	name := r.FormValue("display_name")

	var namePtr *string
	if name != "" {
		namePtr = &name
	}

	h.queries.UpdateAccountDisplayName(r.Context(), db.UpdateAccountDisplayNameParams{
		DisplayName: namePtr,
		ID:          id,
	})

	accounts, _ := h.queries.ListAccountsWithInstitution(r.Context())
	for _, a := range accounts {
		if a.ID == id {
			components.AccountDisplayNameRow(a).Render(r.Context(), w)
			return
		}
	}
}

func (h *SettingsHandler) UpdateCategory(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	name := r.FormValue("name")
	color := r.FormValue("color")

	if name == "" {
		http.Error(w, "name required", http.StatusBadRequest)
		return
	}

	var colorPtr *string
	if color != "" {
		colorPtr = &color
	}

	cat, err := h.queries.UpdateCategory(r.Context(), db.UpdateCategoryParams{
		Name:  name,
		Color: colorPtr,
		ID:    id,
	})
	if err != nil {
		http.Error(w, "update failed", http.StatusInternalServerError)
		return
	}

	components.CategorySettingsRow(cat).Render(r.Context(), w)
}

func (h *SettingsHandler) UpdateModel(w http.ResponseWriter, r *http.Request) {
	model := r.FormValue("model")
	if model == "" {
		http.Error(w, "model required", http.StatusBadRequest)
		return
	}

	if err := h.queries.UpsertSetting(r.Context(), db.UpsertSettingParams{
		Key:   "openrouter_model",
		Value: model,
	}); err != nil {
		http.Error(w, "save failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, `<span style="color:var(--green);font-size:0.75rem">✓ Saved</span>`)
}

func (h *SettingsHandler) DeleteRule(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.queries.DeleteRule(r.Context(), id); err != nil {
		http.Error(w, "delete failed", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// ActiveModel returns the OpenRouter model to use, checking app_settings then env var.
func ActiveModel(queries *db.Queries, r *http.Request) string {
	if model, err := queries.GetSetting(r.Context(), "openrouter_model"); err == nil && model != "" {
		return model
	}
	if m := os.Getenv("OPENROUTER_MODEL"); m != "" {
		return m
	}
	return "deepseek/deepseek-r1-0528-qwen3-8b:free"
}
