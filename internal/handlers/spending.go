package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/abarroso647/holo/internal/components"
	db "github.com/abarroso647/holo/internal/db/generated"
)

type SpendingHandler struct {
	queries *db.Queries
}

func NewSpendingHandler(queries *db.Queries) *SpendingHandler {
	return &SpendingHandler{queries: queries}
}

func (h *SpendingHandler) Page(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	now := time.Now().UTC()

	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")

	if from == "" {
		from = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC).Format("2006-01-02")
	}
	if to == "" {
		to = now.Format("2006-01-02")
	}

	cats, err := h.queries.GetSpendingByCategory(ctx, db.GetSpendingByCategoryParams{
		Date:   from,
		Date_2: to,
	})
	if err != nil {
		http.Error(w, fmt.Sprintf("spending: %v", err), http.StatusInternalServerError)
		return
	}

	twelveMonthsAgo := now.AddDate(0, -11, 0)
	startOfRange := time.Date(twelveMonthsAgo.Year(), twelveMonthsAgo.Month(), 1, 0, 0, 0, 0, time.UTC)
	rawMonthly, err := h.queries.GetMonthlyTotals(ctx, startOfRange.Format("2006-01-02"))
	if err != nil {
		http.Error(w, fmt.Sprintf("monthly: %v", err), http.StatusInternalServerError)
		return
	}

	monthly := make([]components.MonthlyTotal, 0, len(rawMonthly))
	for _, row := range rawMonthly {
		m := components.MonthlyTotal{
			Month:    fmt.Sprintf("%v", row.Month),
			Spending: toFloat64(row.Spending),
			Income:   toFloat64(row.Income),
		}
		monthly = append(monthly, m)
	}

	trendRows, err := h.queries.GetMonthlySpendingByCategory(ctx, startOfRange.Format("2006-01-02"))
	if err != nil {
		http.Error(w, fmt.Sprintf("trend: %v", err), http.StatusInternalServerError)
		return
	}

	categoriesJSON := buildCategoriesJSON(cats)
	monthlyJSON := buildMonthlyJSON(monthly)
	trendJSON := buildCategoryTrendJSON(trendRows)

	components.SpendingPage(from, to, cats, monthly, categoriesJSON, monthlyJSON, trendJSON).Render(ctx, w)
}

func buildCategoriesJSON(cats []db.GetSpendingByCategoryRow) string {
	type chartData struct {
		Labels []string  `json:"labels"`
		Values []float64 `json:"values"`
		Colors []string  `json:"colors"`
	}
	d := chartData{
		Labels: make([]string, len(cats)),
		Values: make([]float64, len(cats)),
		Colors: make([]string, len(cats)),
	}
	for i, c := range cats {
		d.Labels[i] = c.CategoryName
		d.Values[i] = c.Total
		d.Colors[i] = c.CategoryColor
	}
	b, _ := json.Marshal(d)
	return string(b)
}

func buildCategoryTrendJSON(rows []db.GetMonthlySpendingByCategoryRow) string {
	// Collect ordered months and per-category totals
	monthSet := map[string]struct{}{}
	var months []string
	type catData struct {
		Name   string
		Color  string
		Totals map[string]float64
	}
	cats := map[string]*catData{}
	var catOrder []string

	for _, r := range rows {
		m := fmt.Sprintf("%v", r.Month)
		if _, ok := monthSet[m]; !ok {
			monthSet[m] = struct{}{}
			months = append(months, m)
		}
		id := fmt.Sprintf("%v", r.CategoryID)
		if _, ok := cats[id]; !ok {
			cats[id] = &catData{
				Name:   r.CategoryName,
				Color:  r.CategoryColor,
				Totals: map[string]float64{},
			}
			catOrder = append(catOrder, id)
		}
		cats[id].Totals[m] = toFloat64(r.Total)
	}

	type dataset struct {
		Label           string    `json:"label"`
		Data            []float64 `json:"data"`
		BackgroundColor string    `json:"backgroundColor"`
	}
	var datasets []dataset
	for _, id := range catOrder {
		c := cats[id]
		vals := make([]float64, len(months))
		for i, m := range months {
			vals[i] = c.Totals[m]
		}
		datasets = append(datasets, dataset{
			Label:           c.Name,
			Data:            vals,
			BackgroundColor: c.Color,
		})
	}

	type chartData struct {
		Labels   []string  `json:"labels"`
		Datasets []dataset `json:"datasets"`
	}
	b, _ := json.Marshal(chartData{Labels: months, Datasets: datasets})
	return string(b)
}

func buildMonthlyJSON(monthly []components.MonthlyTotal) string {
	type chartData struct {
		Labels   []string  `json:"labels"`
		Spending []float64 `json:"spending"`
		Income   []float64 `json:"income"`
	}
	d := chartData{
		Labels:   make([]string, len(monthly)),
		Spending: make([]float64, len(monthly)),
		Income:   make([]float64, len(monthly)),
	}
	for i, m := range monthly {
		d.Labels[i] = m.Month
		d.Spending[i] = m.Spending
		d.Income[i] = m.Income
	}
	b, _ := json.Marshal(d)
	return string(b)
}
