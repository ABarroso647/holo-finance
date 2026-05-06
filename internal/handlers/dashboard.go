package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"holo/internal/components"
	db "holo/internal/db/generated"
)

type DashboardHandler struct {
	queries *db.Queries
}

func NewDashboardHandler(queries *db.Queries) *DashboardHandler {
	return &DashboardHandler{queries: queries}
}

func (h *DashboardHandler) Page(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	rawNW, _ := h.queries.GetNetWorth(ctx)
	netWorth := toFloat64(rawNW)

	now := time.Now().UTC()
	isLastMonth := r.URL.Query().Get("month") == "last"

	var monthStart, monthEnd, monthLabel string
	if isLastMonth {
		first := time.Date(now.Year(), now.Month()-1, 1, 0, 0, 0, 0, time.UTC)
		last := time.Date(now.Year(), now.Month(), 0, 0, 0, 0, 0, time.UTC)
		monthStart = first.Format("2006-01-02")
		monthEnd = last.Format("2006-01-02")
		monthLabel = first.Format("January 2006")
	} else {
		first := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		monthStart = first.Format("2006-01-02")
		monthEnd = now.Format("2006-01-02")
		monthLabel = first.Format("January 2006")
	}

	summary, _ := h.queries.GetThisMonthSummary(ctx, db.GetThisMonthSummaryParams{
		Date:   monthStart,
		Date_2: monthEnd,
	})
	spending := toFloat64(summary.Spending)
	income := toFloat64(summary.Income)
	salary := toFloat64(summary.Salary)
	cashback := toFloat64(summary.Cashback)

	topCats, _ := h.queries.GetSpendingByCategory(ctx, db.GetSpendingByCategoryParams{
		Date:   monthStart,
		Date_2: monthEnd,
	})

	recentTxns, _ := h.queries.ListTransactions(ctx, db.ListTransactionsParams{
		Limit:  10,
		Offset: 0,
	})

	rawRecurring, _ := h.queries.GetRecurringSpendForPeriod(ctx, db.GetRecurringSpendForPeriodParams{
		Date:   monthStart,
		Date_2: monthEnd,
	})
	recurring := toFloat64(rawRecurring)

	sixMonthsAgo := time.Date(now.Year(), now.Month()-5, 1, 0, 0, 0, 0, time.UTC)
	monthlyFlows, _ := h.queries.GetMonthlyFlows(ctx, sixMonthsAgo.Format("2006-01-02"))
	monthlyJSON := buildMonthlyFlowsJSON(monthlyFlows)

	accounts, _ := h.queries.ListAccountsWithInstitution(ctx)

	catsJSON := buildDashCatsJSON(topCats)

	components.DashboardPage(netWorth, spending, income, salary, cashback, recurring, catsJSON, monthlyJSON, accounts, recentTxns, monthLabel, isLastMonth).Render(ctx, w)
}

func buildMonthlyFlowsJSON(flows []db.GetMonthlyFlowsRow) string {
	type chartData struct {
		Labels   []string  `json:"labels"`
		Spending []float64 `json:"spending"`
		Income   []float64 `json:"income"`
	}
	d := chartData{
		Labels:   make([]string, len(flows)),
		Spending: make([]float64, len(flows)),
		Income:   make([]float64, len(flows)),
	}
	for i, m := range flows {
		if s, ok := m.Month.(string); ok {
			d.Labels[i] = s
		}
		d.Spending[i] = toFloat64(m.Spending)
		d.Income[i] = toFloat64(m.Income)
	}
	b, _ := json.Marshal(d)
	return string(b)
}

func buildDashCatsJSON(cats []db.GetSpendingByCategoryRow) string {
	if len(cats) == 0 {
		return `{"labels":[],"values":[],"colors":[]}`
	}
	if len(cats) > 8 {
		cats = cats[:8]
	}
	labels := make([]string, len(cats))
	values := make([]float64, len(cats))
	colors := make([]string, len(cats))
	for i, c := range cats {
		labels[i] = c.CategoryName
		values[i] = c.Total
		colors[i] = c.CategoryColor
	}
	lb, _ := json.Marshal(labels)
	vb, _ := json.Marshal(values)
	cb, _ := json.Marshal(colors)
	return fmt.Sprintf(`{"labels":%s,"values":%s,"colors":%s}`, lb, vb, cb)
}

func toFloat64(v interface{}) float64 {
	if v == nil {
		return 0
	}
	switch x := v.(type) {
	case float64:
		return x
	case float32:
		return float64(x)
	case int64:
		return float64(x)
	case int:
		return float64(x)
	case []byte:
		var f float64
		fmt.Sscanf(string(x), "%f", &f)
		return f
	}
	return 0
}
