package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
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
	interest := toFloat64(summary.Interest)
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

	salaryEstimates, _ := h.queries.ListSalaryEstimates(ctx)

	catsJSON := buildDashCatsJSON(topCats, monthStart, monthEnd)

	budgets, _ := h.queries.ListBudgets(ctx)

	spendMap := map[string]float64{}
	for _, s := range topCats {
		spendMap[s.CategoryID] = s.Total
	}

	var budgetProgress []components.BudgetProgress
	for _, b := range budgets {
		spent := spendMap[b.CategoryID]
		pct := 0.0
		if b.MonthlyLimit > 0 {
			pct = spent / b.MonthlyLimit
		}
		color := ""
		if b.CategoryColor != nil {
			color = *b.CategoryColor
		}
		budgetProgress = append(budgetProgress, components.BudgetProgress{
			BudgetID:      b.ID,
			CategoryID:    b.CategoryID,
			CategoryName:  b.CategoryName,
			CategoryColor: color,
			MonthlyLimit:  b.MonthlyLimit,
			Spent:         spent,
			PctUsed:       pct,
		})
	}
	sort.Slice(budgetProgress, func(i, j int) bool {
		return budgetProgress[i].PctUsed > budgetProgress[j].PctUsed
	})

	tracker := buildBudgetTracker(ctx, h.queries, now)

	components.DashboardPage(netWorth, spending, income, salary, interest, cashback, recurring, catsJSON, monthlyJSON, accounts, recentTxns, monthLabel, isLastMonth, salaryEstimates, budgetProgress, tracker).Render(ctx, w)
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

func buildDashCatsJSON(cats []db.GetSpendingByCategoryRow, dateFrom, dateTo string) string {
	if len(cats) == 0 {
		return `{"labels":[],"values":[],"colors":[],"ids":[],"date_from":"","date_to":""}`
	}
	if len(cats) > 8 {
		cats = cats[:8]
	}
	labels := make([]string, len(cats))
	values := make([]float64, len(cats))
	colors := make([]string, len(cats))
	ids := make([]string, len(cats))
	for i, c := range cats {
		labels[i] = c.CategoryName
		values[i] = c.Total
		colors[i] = c.CategoryColor
		ids[i] = c.CategoryID
	}
	lb, _ := json.Marshal(labels)
	vb, _ := json.Marshal(values)
	cb, _ := json.Marshal(colors)
	ib, _ := json.Marshal(ids)
	dfb, _ := json.Marshal(dateFrom)
	dtb, _ := json.Marshal(dateTo)
	return fmt.Sprintf(`{"labels":%s,"values":%s,"colors":%s,"ids":%s,"date_from":%s,"date_to":%s}`, lb, vb, cb, ib, dfb, dtb)
}

// CategoryTxns returns dashboard transaction rows filtered by category and date range.
// Used by the chart click-through to filter the Recent Transactions widget inline.
func (h *DashboardHandler) CategoryTxns(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	q := r.URL.Query()
	categoryID := q.Get("category_id")
	dateFrom := q.Get("date_from")
	dateTo := q.Get("date_to")

	w.Header().Set("Content-Type", "text/html")

	if categoryID == "" {
		txns, _ := h.queries.ListTransactions(ctx, db.ListTransactionsParams{Limit: 10, Offset: 0})
		for _, txn := range txns {
			components.DashTxnRowFragment(txn).Render(ctx, w)
		}
		if len(txns) == 0 {
			fmt.Fprintf(w, `<tr><td colspan="4" style="padding:1rem;color:var(--muted);font-size:0.8rem">No transactions</td></tr>`)
		}
		return
	}

	rows, _ := h.queries.SearchTransactions(ctx, db.SearchTransactionsParams{
		Search:     "",
		AccountID:  "",
		CategoryID: categoryID,
		DateFrom:   dateFrom,
		DateTo:     dateTo,
		Recurring:  "",
		TagID:      "",
		Limit:      15,
		Offset:     0,
	})
	for _, row := range rows {
		components.DashSearchTxnRowFragment(row).Render(ctx, w)
	}
	if len(rows) == 0 {
		fmt.Fprintf(w, `<tr><td colspan="4" style="padding:1rem;color:var(--muted);font-size:0.8rem">No transactions</td></tr>`)
	}
}

// needsCategories maps parent category IDs to the Needs bucket.
var needsCategories = map[string]bool{
	"cat_housing":   true,
	"cat_health":    true,
	"cat_transport": true,
}

// wantsCategories maps parent category IDs to the Wants bucket.
var wantsCategories = map[string]bool{
	"cat_food":          true,
	"cat_entertainment": true,
	"cat_shopping":      true,
	"cat_personal":      true,
	"cat_travel":        true,
	"cat_other":         true,
	"cat_finance":       true,
	"cat_education":     true,
}

func buildBudgetTracker(ctx context.Context, queries *db.Queries, now time.Time) components.BudgetTracker {
	// Read annual income setting.
	incomeStr, _ := queries.GetSetting(ctx, "invest_annual_income")
	if incomeStr == "" {
		return components.BudgetTracker{Available: false}
	}
	annualIncome, err := strconv.ParseFloat(incomeStr, 64)
	if err != nil || annualIncome <= 0 {
		return components.BudgetTracker{Available: false}
	}
	monthlyIncome := annualIncome / 12.0

	// Read configurable percentages (with defaults).
	needsPct := readFloatSetting(ctx, queries, "budget_needs_pct", 50.0)
	wantsPct := readFloatSetting(ctx, queries, "budget_wants_pct", 30.0)
	savingsPct := readFloatSetting(ctx, queries, "budget_savings_pct", 20.0)

	// Past 3 months date range.
	threeMonthsAgo := now.AddDate(0, -3, 0).Format("2006-01-02")
	today := now.Format("2006-01-02")

	cats, _ := queries.GetSpendingByCategory(ctx, db.GetSpendingByCategoryParams{
		Date:   threeMonthsAgo,
		Date_2: today,
	})

	var needsTotal, wantsTotal float64
	for _, c := range cats {
		if needsCategories[c.CategoryID] {
			needsTotal += c.Total
		} else if wantsCategories[c.CategoryID] {
			wantsTotal += c.Total
		}
	}

	// Monthly averages over 3 months.
	avgNeeds := needsTotal / 3.0
	avgWants := wantsTotal / 3.0
	avgSavings := monthlyIncome - avgNeeds - avgWants

	// Targets.
	needsTarget := monthlyIncome * needsPct / 100.0
	wantsTarget := monthlyIncome * wantsPct / 100.0
	savingsTarget := monthlyIncome * savingsPct / 100.0

	// Actual percentages.
	var needsActualPct, wantsActualPct, savingsActualPct float64
	if monthlyIncome > 0 {
		needsActualPct = avgNeeds / monthlyIncome * 100.0
		wantsActualPct = avgWants / monthlyIncome * 100.0
		savingsActualPct = avgSavings / monthlyIncome * 100.0
	}

	return components.BudgetTracker{
		NeedsPct:         needsPct,
		WantsPct:         wantsPct,
		SavingsPct:       savingsPct,
		AvgMonthlyIncome: monthlyIncome,
		AvgNeedsSpend:    avgNeeds,
		AvgWantsSpend:    avgWants,
		AvgSavingsActual: avgSavings,
		NeedsTarget:      needsTarget,
		WantsTarget:      wantsTarget,
		SavingsTarget:    savingsTarget,
		NeedsActualPct:   needsActualPct,
		WantsActualPct:   wantsActualPct,
		SavingsActualPct: savingsActualPct,
		Available:        true,
	}
}

func readFloatSetting(ctx context.Context, queries *db.Queries, key string, defaultVal float64) float64 {
	s, err := queries.GetSetting(ctx, key)
	if err != nil || s == "" {
		return defaultVal
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return defaultVal
	}
	return v
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
