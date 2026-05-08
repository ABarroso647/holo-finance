package categorize

import (
	"context"
	"testing"

	"holo/internal/testutil"
)

func TestDetectSalary_RequiresThreeMonths(t *testing.T) {
	q := testutil.NewDB(t)
	ctx := context.Background()
	instID := testutil.InsertInstitution(t, q)
	acctID := testutil.InsertAccount(t, q, instID, "depository")

	// 2 months of wage income — should not produce an estimate
	testutil.InsertTransactionOnDate(t, q, "tx1", acctID, "plaid-tx1", "ACME Corp", -2000.0, "INCOME_WAGES", "plaid", "2026-01-15")
	testutil.InsertTransactionOnDate(t, q, "tx2", acctID, "plaid-tx2", "ACME Corp", -2000.0, "INCOME_WAGES", "plaid", "2026-02-15")

	if err := DetectSalary(ctx, q); err != nil {
		t.Fatalf("DetectSalary: %v", err)
	}

	estimates, err := q.ListSalaryEstimates(ctx)
	if err != nil {
		t.Fatalf("ListSalaryEstimates: %v", err)
	}
	if len(estimates) != 0 {
		t.Errorf("expected 0 estimates for 2 months, got %d", len(estimates))
	}
}

func TestDetectSalary_DetectsAfterThreeMonths(t *testing.T) {
	q := testutil.NewDB(t)
	ctx := context.Background()
	instID := testutil.InsertInstitution(t, q)
	acctID := testutil.InsertAccount(t, q, instID, "depository")

	// 3 months of wage income — should produce 1 estimate
	testutil.InsertTransactionOnDate(t, q, "tx1", acctID, "plaid-tx1", "ACME Corp", -2000.0, "INCOME_WAGES", "plaid", "2026-01-15")
	testutil.InsertTransactionOnDate(t, q, "tx2", acctID, "plaid-tx2", "ACME Corp", -2000.0, "INCOME_WAGES", "plaid", "2026-02-15")
	testutil.InsertTransactionOnDate(t, q, "tx3", acctID, "plaid-tx3", "ACME Corp", -2000.0, "INCOME_WAGES", "plaid", "2026-03-15")

	if err := DetectSalary(ctx, q); err != nil {
		t.Fatalf("DetectSalary: %v", err)
	}

	estimates, err := q.ListSalaryEstimates(ctx)
	if err != nil {
		t.Fatalf("ListSalaryEstimates: %v", err)
	}
	if len(estimates) != 1 {
		t.Fatalf("expected 1 estimate, got %d", len(estimates))
	}
	if estimates[0].MerchantName != "ACME Corp" {
		t.Errorf("expected merchant 'ACME Corp', got %q", estimates[0].MerchantName)
	}
}

func TestDetectSalary_ExcludesNonWageIncome(t *testing.T) {
	q := testutil.NewDB(t)
	ctx := context.Background()
	instID := testutil.InsertInstitution(t, q)
	acctID := testutil.InsertAccount(t, q, instID, "depository")

	// 3 months of interest income — should NOT produce an estimate (category is INCOME_INTEREST, not INCOME_WAGES)
	testutil.InsertTransactionOnDate(t, q, "tx1", acctID, "plaid-tx1", "Interest Payment", -50.0, "INCOME_INTEREST_EARNED", "plaid", "2026-01-15")
	testutil.InsertTransactionOnDate(t, q, "tx2", acctID, "plaid-tx2", "Interest Payment", -50.0, "INCOME_INTEREST_EARNED", "plaid", "2026-02-15")
	testutil.InsertTransactionOnDate(t, q, "tx3", acctID, "plaid-tx3", "Interest Payment", -50.0, "INCOME_INTEREST_EARNED", "plaid", "2026-03-15")

	if err := DetectSalary(ctx, q); err != nil {
		t.Fatalf("DetectSalary: %v", err)
	}

	estimates, err := q.ListSalaryEstimates(ctx)
	if err != nil {
		t.Fatalf("ListSalaryEstimates: %v", err)
	}
	if len(estimates) != 0 {
		t.Errorf("expected 0 estimates for non-wage income, got %d", len(estimates))
	}
}

func TestDetectSalary_AverageDayOfMonth(t *testing.T) {
	q := testutil.NewDB(t)
	ctx := context.Background()
	instID := testutil.InsertInstitution(t, q)
	acctID := testutil.InsertAccount(t, q, instID, "depository")

	// Payments on days 14, 15, 16 — avg should be ≈15
	testutil.InsertTransactionOnDate(t, q, "tx1", acctID, "plaid-tx1", "ACME Corp", -2000.0, "INCOME_WAGES", "plaid", "2026-01-14")
	testutil.InsertTransactionOnDate(t, q, "tx2", acctID, "plaid-tx2", "ACME Corp", -2000.0, "INCOME_WAGES", "plaid", "2026-02-15")
	testutil.InsertTransactionOnDate(t, q, "tx3", acctID, "plaid-tx3", "ACME Corp", -2000.0, "INCOME_WAGES", "plaid", "2026-03-16")

	if err := DetectSalary(ctx, q); err != nil {
		t.Fatalf("DetectSalary: %v", err)
	}

	estimates, err := q.ListSalaryEstimates(ctx)
	if err != nil {
		t.Fatalf("ListSalaryEstimates: %v", err)
	}
	if len(estimates) != 1 {
		t.Fatalf("expected 1 estimate, got %d", len(estimates))
	}
	avgDay := estimates[0].AvgDayOfMonth
	if avgDay < 14.9 || avgDay > 15.1 {
		t.Errorf("expected avg_day_of_month ≈ 15, got %f", avgDay)
	}
}
