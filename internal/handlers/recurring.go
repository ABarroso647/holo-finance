package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"time"

	"holo/internal/components"
	db "holo/internal/db/generated"
)

type RecurringHandler struct {
	queries *db.Queries
}

func NewRecurringHandler(q *db.Queries) *RecurringHandler {
	return &RecurringHandler{queries: q}
}

func (h *RecurringHandler) Page(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	now := time.Now()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	monthEnd := monthStart.AddDate(0, 1, -1)

	monthlyTotal, _ := h.queries.GetRecurringSpendForPeriod(ctx, db.GetRecurringSpendForPeriodParams{
		Date:   monthStart.Format("2006-01-02"),
		Date_2: monthEnd.Format("2006-01-02"),
	})

	merchantRows, _ := h.queries.GetRecurringByMerchant(ctx)
	annualRows, _ := h.queries.GetAnnualRecurringByMerchant(ctx)
	exclusions, _ := h.queries.ListRecurringExclusions(ctx)

	// Build a set of merchants already covered by Plaid's recurring detection.
	plaidMerchants := make(map[string]bool, len(merchantRows))
	items := make([]components.RecurringItem, 0, len(merchantRows)+len(annualRows))

	for _, m := range merchantRows {
		plaidMerchants[m.Merchant] = true
		items = append(items, components.RecurringItem{
			Merchant:   m.Merchant,
			MonthsSeen: m.MonthsSeen,
			AvgAmount:  m.AvgAmount,
			MaxAmount:  m.MaxAmount,
			MinAmount:  m.MinAmount,
			LastDate:   fmt.Sprintf("%v", m.LastDate),
			FirstDate:  fmt.Sprintf("%v", m.FirstDate),
			YearlyEst:  m.AvgAmount * 12,
			IsAnnual:   false,
		})
	}

	// Append annual recurring charges not already in the Plaid-flagged list.
	for _, a := range annualRows {
		if plaidMerchants[a.Merchant] {
			continue
		}
		items = append(items, components.RecurringItem{
			Merchant:   a.Merchant,
			MonthsSeen: a.YearsSeen,
			AvgAmount:  a.AvgAmount,
			MaxAmount:  a.MaxAmount,
			MinAmount:  a.MinAmount,
			LastDate:   fmt.Sprintf("%v", a.LastDate),
			FirstDate:  fmt.Sprintf("%v", a.FirstDate),
			YearlyEst:  a.AvgAmount,
			IsAnnual:   true,
		})
	}

	sixMonthsAgo := now.AddDate(0, -6, 0).Format("2006-01-02")
	flows, _ := h.queries.GetMonthlyRecurringFlows(ctx, sixMonthsAgo)
	monthlyJSON := buildRecurringMonthlyJSON(flows)

	components.RecurringPage(toFloat64(monthlyTotal), items, exclusions, monthlyJSON).Render(ctx, w)
}

func (h *RecurringHandler) ExcludeMerchant(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Merchant string `json:"merchant"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Merchant == "" {
		http.Error(w, "merchant required", http.StatusBadRequest)
		return
	}
	_ = h.queries.ExcludeMerchantFromRecurring(r.Context(), body.Merchant)
	w.WriteHeader(http.StatusOK)
}

func (h *RecurringHandler) IncludeMerchant(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Merchant string `json:"merchant"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Merchant == "" {
		http.Error(w, "merchant required", http.StatusBadRequest)
		return
	}
	_ = h.queries.IncludeMerchantInRecurring(r.Context(), body.Merchant)
	w.WriteHeader(http.StatusOK)
}

func computeMedianAmount(rows []db.GetRecurringByMerchantRow) float64 {
	if len(rows) == 0 {
		return 0
	}
	amounts := make([]float64, len(rows))
	for i, r := range rows {
		amounts[i] = r.AvgAmount
	}
	sort.Float64s(amounts)
	mid := len(amounts) / 2
	if len(amounts)%2 == 0 {
		return (amounts[mid-1] + amounts[mid]) / 2
	}
	return amounts[mid]
}

func buildRecurringMonthlyJSON(flows []db.GetMonthlyRecurringFlowsRow) string {
	type chartData struct {
		Labels []string  `json:"labels"`
		Values []float64 `json:"values"`
	}
	d := chartData{
		Labels: make([]string, len(flows)),
		Values: make([]float64, len(flows)),
	}
	for i, f := range flows {
		d.Labels[i] = fmt.Sprintf("%v", f.Month)
		d.Values[i] = f.Total
	}
	b, _ := json.Marshal(d)
	return string(b)
}
