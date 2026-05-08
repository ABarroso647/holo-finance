package invest

import (
	"fmt"
	"time"
)

// Recommendation holds the result of an RRSP vs TFSA recommendation calculation.
type Recommendation struct {
	MarginalRate      float64
	AvgMonthlySurplus float64
	Buffer            float64
	Investable        float64 // max(surplus - buffer, 0)
	RRSPAmount        float64
	TFSAAmount        float64
	RRSPTaxSaving     float64 // RRSPAmount * 12 * marginalRate (annual estimate)
	Rationale         string
	MissingInputs     []string
}

// YearlyContributionPlan describes end-of-year contribution targets.
type YearlyContributionPlan struct {
	// How much to save annually to hit the savings percentage target
	TargetAnnualSavings float64
	// Months remaining in the calendar year (including current month)
	MonthsRemaining int
	// Consistent monthly contribution target (annualSavings / 12)
	MonthlyTarget float64
	// Per-account monthly breakdowns
	RRSPMonthly float64
	TFSAMonthly float64
	FHSAMonthly float64
	// Room remaining for each account
	RRSPRoom float64
	TFSARoom float64
	FHSARoom float64
}

// YearlyPlan computes end-of-year contribution targets given income, savings %, account room,
// marginal tax rate, and whether FHSA is enabled.
//
// Priority order:
//  1. FHSA first (if fhsaRoom > 0) — best of both worlds (deduction + tax-free growth)
//  2. RRSP if marginalRate >= 0.30 and rrspRoom > 0
//  3. TFSA for any remainder
//
// Each account is capped at room/12 per month.
func YearlyPlan(annualIncome, savingsPct, rrspRoom, tfsaRoom, fhsaRoom float64, marginalRate float64, now time.Time) YearlyContributionPlan {
	targetAnnualSavings := annualIncome * savingsPct / 100
	monthsRemaining := 12 - int(now.Month()) + 1
	monthlyTarget := targetAnnualSavings / 12

	remaining := monthlyTarget

	var fhsaMonthly, rrspMonthly, tfsaMonthly float64

	// 1. FHSA first
	if fhsaRoom > 0 {
		fhsaMonthly = fhsaRoom / 12
		if fhsaMonthly > remaining {
			fhsaMonthly = remaining
		}
		remaining -= fhsaMonthly
	}

	// 2. RRSP if high marginal rate
	if remaining > 0 && marginalRate >= 0.30 && rrspRoom > 0 {
		rrspMonthly = rrspRoom / 12
		if rrspMonthly > remaining {
			rrspMonthly = remaining
		}
		remaining -= rrspMonthly
	}

	// 3. TFSA for the rest
	if remaining > 0 && tfsaRoom > 0 {
		tfsaMonthly = tfsaRoom / 12
		if tfsaMonthly > remaining {
			tfsaMonthly = remaining
		}
	}

	return YearlyContributionPlan{
		TargetAnnualSavings: targetAnnualSavings,
		MonthsRemaining:     monthsRemaining,
		MonthlyTarget:       monthlyTarget,
		RRSPMonthly:         rrspMonthly,
		TFSAMonthly:         tfsaMonthly,
		FHSAMonthly:         fhsaMonthly,
		RRSPRoom:            rrspRoom,
		TFSARoom:            tfsaRoom,
		FHSARoom:            fhsaRoom,
	}
}

// Recommend computes an RRSP vs TFSA allocation recommendation.
//
// Logic:
//  1. If annualIncome == 0, return MissingInputs = ["annual income"]
//  2. marginalRate = MarginalRate(annualIncome, province)
//  3. investable = max(avgMonthlySurplus - buffer, 0)
//  4. If marginalRate >= 0.30 and rrspRoom > 0: route 100% to RRSP (cap at rrspRoom if needed)
//  5. If marginalRate < 0.30 or rrspRoom == 0: route to TFSA (cap at tfsaRoom if needed)
//  6. Any overflow goes to the other bucket
//  7. RRSPTaxSaving = RRSPAmount * 12 * marginalRate
func Recommend(annualIncome, buffer, rrspRoom, tfsaRoom, avgMonthlySurplus float64, province string) Recommendation {
	if annualIncome == 0 {
		return Recommendation{
			MissingInputs: []string{"annual income"},
		}
	}

	rate := MarginalRate(annualIncome, province)

	investable := avgMonthlySurplus - buffer
	if investable < 0 {
		investable = 0
	}

	var rrspAmt, tfsaAmt float64
	var rationale string

	if investable == 0 {
		rationale = "Your average monthly surplus is at or below your buffer. Focus on building cash reserves before investing."
	} else if rate >= 0.30 && rrspRoom > 0 {
		// High marginal rate → prioritize RRSP for the immediate tax deduction
		rrspAmt = investable
		if rrspRoom > 0 && rrspAmt > rrspRoom/12 {
			// Cap monthly RRSP at 1/12 of annual room (rough monthly allocation)
			// But if rrspRoom < investable total we cap to room / 12
			// Use total room as the cap — if investable fits entirely within room, use it all
			if investable*12 > rrspRoom {
				rrspAmt = rrspRoom / 12
				tfsaAmt = investable - rrspAmt
				if tfsaRoom > 0 && tfsaAmt > tfsaRoom/12 {
					tfsaAmt = tfsaRoom / 12
				}
			}
		}
		rationale = fmt.Sprintf(
			"Your marginal rate of %.1f%% is above 30%%, so RRSP contributions give you a meaningful tax deduction. "+
				"Contribute $%.0f/mo to RRSP first%s.",
			rate*100,
			rrspAmt,
			func() string {
				if tfsaAmt > 0 {
					return fmt.Sprintf(", then $%.0f/mo to TFSA once RRSP room runs out", tfsaAmt)
				}
				return ""
			}(),
		)
	} else {
		// Low marginal rate or no RRSP room → use TFSA
		tfsaAmt = investable
		if tfsaRoom > 0 && tfsaAmt*12 > tfsaRoom {
			tfsaAmt = tfsaRoom / 12
			overflow := investable - tfsaAmt
			if rrspRoom > 0 && overflow > 0 {
				rrspAmt = overflow
				if rrspAmt*12 > rrspRoom {
					rrspAmt = rrspRoom / 12
				}
			}
		}

		if rrspRoom == 0 && rate >= 0.30 {
			rationale = fmt.Sprintf(
				"You have no RRSP room available, so the full $%.0f/mo goes to TFSA for tax-free growth.",
				tfsaAmt,
			)
		} else {
			rationale = fmt.Sprintf(
				"Your marginal rate of %.1f%% is below 30%%, so TFSA is preferred — contributions grow tax-free "+
					"and withdrawals don't affect income-tested benefits. Contribute $%.0f/mo to TFSA%s.",
				rate*100,
				tfsaAmt,
				func() string {
					if rrspAmt > 0 {
						return fmt.Sprintf(", then $%.0f/mo to RRSP for any overflow", rrspAmt)
					}
					return ""
				}(),
			)
		}
	}

	taxSaving := rrspAmt * 12 * rate

	return Recommendation{
		MarginalRate:      rate,
		AvgMonthlySurplus: avgMonthlySurplus,
		Buffer:            buffer,
		Investable:        investable,
		RRSPAmount:        rrspAmt,
		TFSAAmount:        tfsaAmt,
		RRSPTaxSaving:     taxSaving,
		Rationale:         rationale,
	}
}
