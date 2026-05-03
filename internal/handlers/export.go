package handlers

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"

	db "holo/internal/db/generated"
)

type ExportHandler struct {
	queries *db.Queries
}

func NewExportHandler(queries *db.Queries) *ExportHandler {
	return &ExportHandler{queries: queries}
}

func (h *ExportHandler) Export(w http.ResponseWriter, r *http.Request) {
	format := r.URL.Query().Get("format")
	dateFrom := r.URL.Query().Get("from")
	dateTo := r.URL.Query().Get("to")

	rows, err := h.queries.ExportTransactions(r.Context(), db.ExportTransactionsParams{
		DateFrom: dateFrom,
		DateTo:   dateTo,
	})
	if err != nil {
		http.Error(w, fmt.Sprintf("export: %v", err), http.StatusInternalServerError)
		return
	}

	if format == "csv" {
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", "attachment; filename=holo-transactions.csv")

		cw := csv.NewWriter(w)
		cw.Write([]string{"date", "account", "institution", "name", "merchant", "amount", "currency", "category", "category_source", "is_recurring", "notes"}) //nolint:errcheck
		for _, row := range rows {
			merchant := ""
			if row.MerchantName != nil {
				merchant = *row.MerchantName
			}
			catName := ""
			if row.CategoryName != nil {
				catName = fmt.Sprintf("%v", *row.CategoryName)
			}
			notes := ""
			if row.Notes != nil {
				notes = *row.Notes
			}
			recurringStr := "no"
			if row.IsRecurring == 1 {
				recurringStr = "yes"
			}
			cw.Write([]string{ //nolint:errcheck
				row.Date,
				fmt.Sprintf("%v", row.AccountName),
				fmt.Sprintf("%v", row.InstitutionName),
				row.Name,
				merchant,
				fmt.Sprintf("%.2f", row.Amount),
				row.Currency,
				catName,
				row.CategorySource,
				recurringStr,
				notes,
			})
		}
		cw.Flush()
		return
	}

	// Default: JSON
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", "attachment; filename=holo-transactions.json")

	type exportRow struct {
		Date           string  `json:"date"`
		Account        string  `json:"account"`
		Institution    string  `json:"institution"`
		Name           string  `json:"name"`
		MerchantName   *string `json:"merchant_name"`
		Amount         float64 `json:"amount"`
		Currency       string  `json:"currency"`
		Category       *string `json:"category"`
		CategorySource string  `json:"category_source"`
		IsRecurring    bool    `json:"is_recurring"`
		Notes          *string `json:"notes"`
	}

	out := make([]exportRow, 0, len(rows))
	for _, row := range rows {
		var catPtr *string
		if row.CategoryName != nil {
			s := fmt.Sprintf("%v", *row.CategoryName)
			catPtr = &s
		}
		out = append(out, exportRow{
			Date:           row.Date,
			Account:        fmt.Sprintf("%v", row.AccountName),
			Institution:    fmt.Sprintf("%v", row.InstitutionName),
			Name:           row.Name,
			MerchantName:   row.MerchantName,
			Amount:         row.Amount,
			Currency:       row.Currency,
			Category:       catPtr,
			CategorySource: row.CategorySource,
			IsRecurring:    row.IsRecurring == 1,
			Notes:          row.Notes,
		})
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.Encode(out) //nolint:errcheck
}
