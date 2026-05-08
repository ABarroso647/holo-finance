package handlers

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os"
	"strconv"
	"time"

	"holo/internal/components"
	db "holo/internal/db/generated"
)

type InvestHandler struct {
	queries *db.Queries
}

func NewInvestHandler(queries *db.Queries) *InvestHandler {
	return &InvestHandler{queries: queries}
}

func (h *InvestHandler) Page(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	now := time.Now().UTC()

	// Use time.Date arithmetic that handles year boundaries correctly
	start := time.Date(now.Year(), now.Month()-3, 1, 0, 0, 0, 0, time.UTC)

	flows, err := h.queries.GetMonthlyFlows(ctx, start.Format("2006-01-02"))
	if err != nil {
		http.Error(w, fmt.Sprintf("monthly flows: %v", err), http.StatusInternalServerError)
		return
	}

	bufferStr, _ := h.queries.GetSetting(ctx, "invest_buffer")
	if bufferStr == "" {
		bufferStr = os.Getenv("INVEST_BUFFER_CAD")
	}
	buffer := 500.0
	if v, err := strconv.ParseFloat(bufferStr, 64); err == nil && v >= 0 {
		buffer = v
	}

	months := make([]components.MonthFlow, 0, len(flows))
	var totalIncome, totalSpending float64
	for _, f := range flows {
		inc := toFloat64(f.Income)
		sp := toFloat64(f.Spending)
		totalIncome += inc
		totalSpending += sp
		months = append(months, components.MonthFlow{
			Month:    fmt.Sprintf("%v", f.Month),
			Income:   inc,
			Spending: sp,
			Net:      inc - sp,
		})
	}

	count := float64(len(months))
	if count == 0 {
		count = 1
	}
	avgSurplus := (totalIncome - totalSpending) / count
	recommended := math.Max(avgSurplus-buffer, 0)

	holdings, _ := h.queries.ListAllHoldings(ctx)
	summary, _ := h.queries.GetPortfolioSummary(ctx)
	byType, _ := h.queries.GetAllocationByType(ctx)
	byAccount, _ := h.queries.GetAllocationByAccount(ctx)

	data := components.InvestData{
		Months:         months,
		AvgIncome:      totalIncome / count,
		AvgSpending:    totalSpending / count,
		AvgSurplus:     avgSurplus,
		Buffer:         buffer,
		Recommended:    recommended,
		Holdings:       holdings,
		PortfolioTotal: summary.TotalValue,
		PortfolioGain:  summary.TotalGain,
		AccountCount:   summary.AccountCount,
		ByType:         byType,
		ByAccount:      byAccount,
		ByTypeJSON:     buildInvestTypeJSON(byType),
	}

	components.InvestPage(data).Render(ctx, w)
}

func buildInvestTypeJSON(rows []db.GetAllocationByTypeRow) string {
	if len(rows) == 0 {
		return `{"labels":[],"values":[],"colors":[]}`
	}
	colorMap := map[string]string{
		"equity":       "#6366f1",
		"etf":          "#60a5fa",
		"mutual fund":  "#c084fc",
		"fixed income": "#34d399",
		"cash":         "#4ade80",
		"derivative":   "#fb923c",
		"other":        "#64748b",
	}
	labels := make([]string, len(rows))
	values := make([]float64, len(rows))
	colors := make([]string, len(rows))
	for i, r := range rows {
		label := "other"
		if s, ok := r.SecurityType.(string); ok && s != "" {
			label = s
		}
		labels[i] = label
		values[i] = r.Value
		if c, ok := colorMap[label]; ok {
			colors[i] = c
		} else {
			colors[i] = colorMap["other"]
		}
	}
	lb, _ := json.Marshal(labels)
	vb, _ := json.Marshal(values)
	cb, _ := json.Marshal(colors)
	return fmt.Sprintf(`{"labels":%s,"values":%s,"colors":%s}`, lb, vb, cb)
}

func (h *InvestHandler) UpdateBuffer(w http.ResponseWriter, r *http.Request) {
	val := r.FormValue("buffer")
	if _, err := strconv.ParseFloat(val, 64); err != nil {
		http.Error(w, "invalid buffer amount", http.StatusBadRequest)
		return
	}

	h.queries.UpsertSetting(r.Context(), db.UpsertSettingParams{ //nolint:errcheck
		Key:   "invest_buffer",
		Value: val,
	})

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, `<span style="color:var(--green);font-size:0.75rem">✓ Saved — <a href="/invest" style="color:var(--accent)">refresh page</a> to update recommendation</span>`)
}
