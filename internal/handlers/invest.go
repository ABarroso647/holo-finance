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
	"holo/internal/invest"
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

	// Read invest settings
	annualIncomeStr, _ := h.queries.GetSetting(ctx, "invest_annual_income")
	province, _ := h.queries.GetSetting(ctx, "invest_province")
	rrspRoomStr, _ := h.queries.GetSetting(ctx, "invest_rrsp_room")
	tfsaRoomStr, _ := h.queries.GetSetting(ctx, "invest_tfsa_room")
	tfsaRoomDateStr, _ := h.queries.GetSetting(ctx, "invest_tfsa_room_date")
	fhsaEnabledStr, _ := h.queries.GetSetting(ctx, "invest_fhsa_enabled")
	fhsaRoomStr, _ := h.queries.GetSetting(ctx, "invest_fhsa_room")
	fhsaRoomDateStr, _ := h.queries.GetSetting(ctx, "invest_fhsa_room_date")
	savingsPctStr, _ := h.queries.GetSetting(ctx, "invest_savings_pct")

	if province == "" {
		province = "ON"
	}

	annualIncome, _ := strconv.ParseFloat(annualIncomeStr, 64)
	rrspRoom, _ := strconv.ParseFloat(rrspRoomStr, 64)
	tfsaRoom, _ := strconv.ParseFloat(tfsaRoomStr, 64)
	fhsaRoom, _ := strconv.ParseFloat(fhsaRoomStr, 64)

	savingsPct := 20.0
	if v, err := strconv.ParseFloat(savingsPctStr, 64); err == nil && v > 0 {
		savingsPct = v
	}

	fhsaEnabled := fhsaEnabledStr == "1"

	// Parse room reference dates
	tfsaRoomDate := parseDate(tfsaRoomDateStr)
	fhsaRoomDate := parseDate(fhsaRoomDateStr)

	// Compute current room with auto-additions for elapsed Jan 1sts
	tfsaCurrentRoom := invest.CurrentTFSARoom(tfsaRoom, tfsaRoomDate, now)
	fhsaCurrentRoom := 0.0
	if fhsaEnabled {
		fhsaCurrentRoom = invest.CurrentFHSARoom(fhsaRoom, fhsaRoomDate, now)
	}

	rec := invest.Recommend(annualIncome, buffer, rrspRoom, tfsaCurrentRoom, avgSurplus, province)

	// Compute end-of-year contribution plan
	fhsaRoomForPlan := 0.0
	if fhsaEnabled {
		fhsaRoomForPlan = fhsaCurrentRoom
	}
	plan := invest.YearlyPlan(annualIncome, savingsPct, rrspRoom, tfsaCurrentRoom, fhsaRoomForPlan, rec.MarginalRate, now)

	data := components.InvestData{
		Months:          months,
		AvgIncome:       totalIncome / count,
		AvgSpending:     totalSpending / count,
		AvgSurplus:      avgSurplus,
		Buffer:          buffer,
		Recommended:     recommended,
		Holdings:        holdings,
		AnnualIncome:    annualIncome,
		Province:        province,
		RRSPRoom:        rrspRoom,
		TFSARoom:        tfsaRoom,
		TFSARoomDate:    tfsaRoomDateStr,
		TFSACurrentRoom: tfsaCurrentRoom,
		FHSAEnabled:     fhsaEnabled,
		FHSARoom:        fhsaRoom,
		FHSARoomDate:    fhsaRoomDateStr,
		FHSACurrentRoom: fhsaCurrentRoom,
		SavingsPct:      savingsPct,
		Recommendation:  rec,
		YearlyPlan:      plan,
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

func (h *InvestHandler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	fields := []struct {
		key string
		val string
	}{
		{"invest_annual_income", r.FormValue("invest_annual_income")},
		{"invest_province", r.FormValue("invest_province")},
		{"invest_rrsp_room", r.FormValue("invest_rrsp_room")},
		{"invest_tfsa_room", r.FormValue("invest_tfsa_room")},
		{"invest_tfsa_room_date", r.FormValue("invest_tfsa_room_date")},
		{"invest_fhsa_enabled", r.FormValue("invest_fhsa_enabled")},
		{"invest_fhsa_room", r.FormValue("invest_fhsa_room")},
		{"invest_fhsa_room_date", r.FormValue("invest_fhsa_room_date")},
		{"invest_savings_pct", r.FormValue("invest_savings_pct")},
		{"invest_buffer", r.FormValue("invest_buffer")},
	}

	for _, f := range fields {
		if f.val == "" {
			continue
		}
		h.queries.UpsertSetting(ctx, db.UpsertSettingParams{ //nolint:errcheck
			Key:   f.key,
			Value: f.val,
		})
	}

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, `<span style="color:var(--green);font-size:0.75rem">✓ Saved — <a href="/invest" style="color:var(--accent)">refresh to update recommendation</a></span>`)
}

// parseDate parses a YYYY-MM-DD string and returns zero time.Time on failure.
func parseDate(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return time.Time{}
	}
	return t
}
