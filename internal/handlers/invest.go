package handlers

import (
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

	data := components.InvestData{
		Months:      months,
		AvgIncome:   totalIncome / count,
		AvgSpending: totalSpending / count,
		AvgSurplus:  avgSurplus,
		Buffer:      buffer,
		Recommended: recommended,
		Holdings:    holdings,
	}

	components.InvestPage(data).Render(ctx, w)
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
