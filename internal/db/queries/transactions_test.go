package queries_test

import (
	"context"
	"testing"

	"holo/internal/categorize"
	dbgen "holo/internal/db/generated"
	"holo/internal/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTxnSum(t *testing.T) (*dbgen.Queries, string) {
	t.Helper()
	q := testutil.NewDB(t)
	ctx := context.Background()
	require.NoError(t, categorize.SeedCategories(ctx, q))
	inst := testutil.InsertInstitution(t, q)
	acctID := testutil.InsertAccount(t, q, inst, "depository")
	return q, acctID
}

func TestSumFilteredTransactions_DateRange(t *testing.T) {
	q, acctID := setupTxnSum(t)
	ctx := context.Background()

	// Inside range
	testutil.InsertTransactionOnDate(t, q, "t1", acctID, "p1", "Coffee", 5.0, "", "plaid", "2026-03-15")
	testutil.InsertTransactionOnDate(t, q, "t2", acctID, "p2", "Lunch", 12.0, "", "plaid", "2026-03-20")
	// Outside range
	testutil.InsertTransactionOnDate(t, q, "t3", acctID, "p3", "Old charge", 99.0, "", "plaid", "2026-01-01")

	row, err := q.SumFilteredTransactions(ctx, dbgen.SumFilteredTransactionsParams{
		Search:     "",
		AccountID:  "",
		CategoryID: "",
		DateFrom:   "2026-03-01",
		DateTo:     "2026-03-31",
		Recurring:  "",
		TagID:      "",
	})
	require.NoError(t, err)
	assert.InDelta(t, 17.0, row.Spending, 0.01)
	assert.Equal(t, int64(2), row.Count)
}

func TestSumFilteredTransactions_CategoryFilter(t *testing.T) {
	q, acctID := setupTxnSum(t)
	ctx := context.Background()

	testutil.InsertCategoryWithParent(t, q, "FOOD_AND_DRINK_GROCERIES", "cat_food")

	testutil.InsertTransactionOnDate(t, q, "t1", acctID, "p1", "Groceries", 50.0, "FOOD_AND_DRINK_GROCERIES", "plaid", "2026-03-10")
	testutil.InsertTransactionOnDate(t, q, "t2", acctID, "p2", "Other spend", 30.0, "cat_other", "plaid", "2026-03-10")

	row, err := q.SumFilteredTransactions(ctx, dbgen.SumFilteredTransactionsParams{
		Search:     "",
		AccountID:  "",
		CategoryID: "FOOD_AND_DRINK_GROCERIES",
		DateFrom:   "2026-03-01",
		DateTo:     "2026-03-31",
		Recurring:  "",
		TagID:      "",
	})
	require.NoError(t, err)
	assert.InDelta(t, 50.0, row.Spending, 0.01)
	assert.Equal(t, int64(1), row.Count)
}

func TestSumFilteredTransactions_ExcludesTransfers(t *testing.T) {
	q, acctID := setupTxnSum(t)
	ctx := context.Background()

	testutil.InsertTransactionOnDate(t, q, "t1", acctID, "p1", "Coffee", 5.0, "FOOD_AND_DRINK_COFFEE", "plaid", "2026-03-10")
	testutil.InsertTransactionOnDate(t, q, "t2", acctID, "p2", "CC Payment", 500.0, "LOAN_PAYMENTS_CREDIT_CARD_PAYMENT", "plaid", "2026-03-10")
	testutil.InsertTransactionOnDate(t, q, "t3", acctID, "p3", "Transfer", 200.0, "TRANSFER_OUT", "plaid", "2026-03-10")

	row, err := q.SumFilteredTransactions(ctx, dbgen.SumFilteredTransactionsParams{
		Search:     "",
		AccountID:  "",
		CategoryID: "",
		DateFrom:   "2026-03-01",
		DateTo:     "2026-03-31",
		Recurring:  "",
		TagID:      "",
	})
	require.NoError(t, err)
	assert.InDelta(t, 5.0, row.Spending, 0.01, "transfers and CC payments should not be counted as spending")
}
