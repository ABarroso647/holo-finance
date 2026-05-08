package categorize

import (
	"context"
	"log"

	db "holo/internal/db/generated"
)

// DetectSalary detects recurring wage income patterns and refreshes salary_estimates.
// Non-fatal — logs errors and returns them but callers should not block on failure.
func DetectSalary(ctx context.Context, queries *db.Queries) error {
	patterns, err := queries.GetRecurringIncomePatterns(ctx)
	if err != nil {
		return err
	}
	if err := queries.DeleteAllSalaryEstimates(ctx); err != nil {
		return err
	}
	for _, p := range patterns {
		merchant := p.Merchant
		if err := queries.UpsertSalaryEstimate(ctx, db.UpsertSalaryEstimateParams{
			MerchantName:  merchant,
			AvgAmount:     p.AvgAmount,
			AvgDayOfMonth: p.AvgDay,
			MonthsSeen:    p.MonthsSeen,
		}); err != nil {
			log.Printf("salary: upsert %q: %v", merchant, err)
		}
	}
	return nil
}
