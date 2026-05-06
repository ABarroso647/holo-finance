package handlers

import (
	"net/http"
	"strconv"

	"holo/internal/components"
	db "holo/internal/db/generated"
)

type TransactionHandler struct {
	queries *db.Queries
}

func NewTransactionHandler(queries *db.Queries) *TransactionHandler {
	return &TransactionHandler{queries: queries}
}

func (h *TransactionHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	page := 1
	if p := q.Get("page"); p != "" {
		if n, err := strconv.Atoi(p); err == nil && n > 0 {
			page = n
		}
	}
	const limit = 50
	offset := int64((page - 1) * limit)

	filters := components.TxnFilters{
		Search:     q.Get("q"),
		AccountID:  q.Get("account_id"),
		CategoryID: q.Get("category_id"),
		TagID:      q.Get("tag_id"),
		DateFrom:   q.Get("date_from"),
		DateTo:     q.Get("date_to"),
		Recurring:  q.Get("recurring"),
	}

	txns, total := h.runSearch(r, filters, limit, offset)
	summary := h.runSummary(r, filters)

	categories, _ := h.queries.ListCategories(r.Context())
	accounts, _ := h.queries.ListAccounts(r.Context())
	allTags, _ := h.queries.ListTags(r.Context())

	tagMap := make(map[string][]db.Tag)
	for _, txn := range txns {
		tags, _ := h.queries.ListTagsForTransaction(r.Context(), txn.ID)
		if len(tags) > 0 {
			tagMap[txn.ID] = tags
		}
	}

	if r.Header.Get("HX-Request") == "true" {
		components.TxnTableContent(txns, categories, tagMap, allTags, int(total), page, limit, filters, summary).Render(r.Context(), w)
		return
	}

	components.TransactionsPage(txns, categories, accounts, allTags, tagMap, int(total), page, limit, filters, summary).Render(r.Context(), w)
}

func (h *TransactionHandler) runSummary(r *http.Request, f components.TxnFilters) *components.TxnSummary {
	if !f.IsActive() {
		return nil
	}
	row, err := h.queries.SumFilteredTransactions(r.Context(), db.SumFilteredTransactionsParams{
		Search:     f.Search,
		AccountID:  f.AccountID,
		CategoryID: f.CategoryID,
		TagID:      f.TagID,
		DateFrom:   f.DateFrom,
		DateTo:     f.DateTo,
		Recurring:  f.Recurring,
	})
	if err != nil {
		return nil
	}
	return &components.TxnSummary{
		Spending: row.Spending,
		Income:   row.Income,
		Count:    row.Count,
	}
}

func (h *TransactionHandler) runSearch(r *http.Request, f components.TxnFilters, limit, offset int64) ([]db.ListTransactionsRow, int64) {
	rows, err := h.queries.SearchTransactions(r.Context(), db.SearchTransactionsParams{
		Search:     f.Search,
		AccountID:  f.AccountID,
		CategoryID: f.CategoryID,
		TagID:      f.TagID,
		DateFrom:   f.DateFrom,
		DateTo:     f.DateTo,
		Recurring:  f.Recurring,
		Limit:      limit,
		Offset:     offset,
	})
	if err != nil {
		return nil, 0
	}

	total, _ := h.queries.CountSearchTransactions(r.Context(), db.CountSearchTransactionsParams{
		Search:     f.Search,
		AccountID:  f.AccountID,
		CategoryID: f.CategoryID,
		TagID:      f.TagID,
		DateFrom:   f.DateFrom,
		DateTo:     f.DateTo,
		Recurring:  f.Recurring,
	})

	result := make([]db.ListTransactionsRow, len(rows))
	for i, row := range rows {
		result[i] = searchRowToList(row)
	}
	return result, total
}

func searchRowToList(r db.SearchTransactionsRow) db.ListTransactionsRow {
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
		CategoryConfidence: r.CategoryConfidence,
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
