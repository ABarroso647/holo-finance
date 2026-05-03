package handlers

import (
	"fmt"
	"log"
	"net/http"

	"github.com/abarroso647/holo/internal/categorize"
	"github.com/abarroso647/holo/internal/components"
	db "github.com/abarroso647/holo/internal/db/generated"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type CategorizeHandler struct {
	queries *db.Queries
}

func NewCategorizeHandler(queries *db.Queries) *CategorizeHandler {
	return &CategorizeHandler{queries: queries}
}

// POST /api/categorize/run — runs rules first, then LLM on whatever is still uncategorized
func (h *CategorizeHandler) Run(w http.ResponseWriter, r *http.Request) {
	ruleCount, err := categorize.ApplyRules(r.Context(), h.queries)
	if err != nil {
		http.Error(w, fmt.Sprintf("rules: %v", err), http.StatusInternalServerError)
		return
	}

	llmCount, err := categorize.RunLLMCategorization(r.Context(), h.queries)
	if err != nil {
		log.Printf("LLM categorization error: %v", err)
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `<span style="color:var(--yellow)">Rules: %d categorized. LLM failed: %v</span>`, ruleCount, err)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	if ruleCount+llmCount > 0 {
		w.Header().Set("HX-Trigger", "txnTableRefresh")
	}
	if ruleCount+llmCount == 0 {
		fmt.Fprintf(w, `<span style="color:var(--muted)">Nothing to categorize</span>`)
	} else {
		fmt.Fprintf(w, `<span style="color:var(--green)">Rules: %d, LLM: %d categorized</span>`, ruleCount, llmCount)
	}
}

// POST /api/transactions/{id}/category — inline manual recategorize, returns updated row HTML
func (h *CategorizeHandler) UpdateCategory(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	categoryID := r.FormValue("category_id")

	var catPtr *string
	catSource := "uncategorized"
	if categoryID != "" {
		catPtr = &categoryID
		catSource = "manual"
	}

	if err := h.queries.UpdateTransactionCategory(r.Context(), db.UpdateTransactionCategoryParams{
		CategoryID:     catPtr,
		CategorySource: catSource,
		ID:             id,
	}); err != nil {
		http.Error(w, "update failed", http.StatusInternalServerError)
		return
	}

	txn, err := h.queries.GetTransaction(r.Context(), id)
	if err != nil {
		http.Error(w, "fetch failed", http.StatusInternalServerError)
		return
	}

	categories, _ := h.queries.ListCategories(r.Context())
	components.CategoryCellContent(toListRow(txn), categories).Render(r.Context(), w)
}

// POST /api/rules — creates a contains rule for a merchant name → category mapping
func (h *CategorizeHandler) CreateRule(w http.ResponseWriter, r *http.Request) {
	matchValue := r.FormValue("match_value")
	categoryID := r.FormValue("category_id")
	if matchValue == "" || categoryID == "" {
		http.Error(w, "match_value and category_id required", http.StatusBadRequest)
		return
	}

	if _, err := h.queries.InsertRule(r.Context(), db.InsertRuleParams{
		ID:         uuid.New().String(),
		MatchType:  "contains",
		MatchField: "name",
		MatchValue: matchValue,
		CategoryID: categoryID,
		Priority:   100,
	}); err != nil {
		http.Error(w, fmt.Sprintf("insert rule: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, `<span style="color:var(--green);font-size:0.75rem">✓ Rule saved</span>`)
}

func toListRow(r db.GetTransactionRow) db.ListTransactionsRow {
	return db.ListTransactionsRow{
		ID:                 r.ID,
		AccountID:          r.AccountID,
		PlaidTransactionID: r.PlaidTransactionID,
		Date:               r.Date,
		AuthorizedDate:     r.AuthorizedDate,
		Name:               r.Name,
		MerchantName:       r.MerchantName,
		Amount:             r.Amount,
		Currency:           r.Currency,
		CategoryID:         r.CategoryID,
		CategorySource:     r.CategorySource,
		Pending:            r.Pending,
		IsRecurring:        r.IsRecurring,
		Notes:              r.Notes,
		CreatedAt:          r.CreatedAt,
		UpdatedAt:          r.UpdatedAt,
		AccountName:        r.AccountName,
		InstitutionName:    r.InstitutionName,
		CategoryName:       r.CategoryName,
		CategoryColor:      r.CategoryColor,
	}
}
