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

func insertCardRewardRate(t *testing.T, q *dbgen.Queries, id, accountID string, categoryID *string, rate float64) {
	t.Helper()
	if _, err := q.UpsertCardRewardRate(context.Background(), dbgen.UpsertCardRewardRateParams{
		ID:         id,
		AccountID:  accountID,
		CategoryID: categoryID,
		RewardRate: rate,
	}); err != nil {
		t.Fatalf("insertCardRewardRate %q: %v", id, err)
	}
}

func searchAll(t *testing.T, q *dbgen.Queries) []dbgen.SearchTransactionsRow {
	t.Helper()
	rows, err := q.SearchTransactions(context.Background(), dbgen.SearchTransactionsParams{
		Search:     "",
		AccountID:  "",
		CategoryID: "",
		DateFrom:   "",
		DateTo:     "",
		Recurring:  "",
		TagID:      "",
		Limit:      100,
		Offset:     0,
	})
	require.NoError(t, err)
	return rows
}

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

func TestSearchTransactions_BestCardRate_ExactMatch(t *testing.T) {
	q := testutil.NewDB(t)
	ctx := context.Background()
	require.NoError(t, categorize.SeedCategories(ctx, q))

	inst := testutil.InsertInstitution(t, q)
	debitID := testutil.InsertAccount(t, q, inst, "depository")
	creditID := testutil.InsertAccount(t, q, inst, "credit")

	catID := "cat_food"
	insertCardRewardRate(t, q, "rate1", creditID, &catID, 3.0)

	testutil.InsertTransaction(t, q, "t1", debitID, "p1", "Groceries", 20.0, catID, "plaid")

	rows := searchAll(t, q)
	require.Len(t, rows, 1)
	require.NotNil(t, rows[0].BestCardRate)
	assert.InDelta(t, 3.0, *rows[0].BestCardRate, 0.001)
	require.NotNil(t, rows[0].BestCardName)
	assert.Equal(t, "Test credit", *rows[0].BestCardName)
}

func TestSearchTransactions_BestCardRate_ParentMatch(t *testing.T) {
	q := testutil.NewDB(t)
	ctx := context.Background()
	require.NoError(t, categorize.SeedCategories(ctx, q))

	inst := testutil.InsertInstitution(t, q)
	debitID := testutil.InsertAccount(t, q, inst, "depository")
	creditID := testutil.InsertAccount(t, q, inst, "credit")

	// Insert sub-category with parent cat_food
	testutil.InsertCategoryWithParent(t, q, "FOOD_AND_DRINK_GROCERIES", "cat_food")

	// Rate for the parent category only (no explicit rate for the sub-category)
	parentID := "cat_food"
	insertCardRewardRate(t, q, "rate1", creditID, &parentID, 3.0)

	// Transaction categorized as the sub-category
	testutil.InsertTransaction(t, q, "t1", debitID, "p1", "Groceries", 20.0, "FOOD_AND_DRINK_GROCERIES", "plaid")

	rows := searchAll(t, q)
	require.Len(t, rows, 1)
	require.NotNil(t, rows[0].BestCardRate)
	assert.InDelta(t, 3.0, *rows[0].BestCardRate, 0.001)
}

func TestSearchTransactions_BestCardRate_CatchAll(t *testing.T) {
	q := testutil.NewDB(t)
	ctx := context.Background()
	require.NoError(t, categorize.SeedCategories(ctx, q))

	inst := testutil.InsertInstitution(t, q)
	debitID := testutil.InsertAccount(t, q, inst, "depository")
	creditID := testutil.InsertAccount(t, q, inst, "credit")

	// NULL category_id = catch-all rate
	insertCardRewardRate(t, q, "rate1", creditID, nil, 1.0)

	// Transaction categorized as something with no explicit or parent rate
	testutil.InsertTransaction(t, q, "t1", debitID, "p1", "Random spend", 15.0, "cat_other", "plaid")

	rows := searchAll(t, q)
	require.Len(t, rows, 1)
	require.NotNil(t, rows[0].BestCardRate)
	assert.InDelta(t, 1.0, *rows[0].BestCardRate, 0.001)
}

func TestSearchTransactions_BestCardRate_NilWhenNoRates(t *testing.T) {
	q := testutil.NewDB(t)
	ctx := context.Background()
	require.NoError(t, categorize.SeedCategories(ctx, q))

	inst := testutil.InsertInstitution(t, q)
	debitID := testutil.InsertAccount(t, q, inst, "depository")

	// No card_reward_rates rows at all
	testutil.InsertTransaction(t, q, "t1", debitID, "p1", "Coffee", 5.0, "cat_food", "plaid")

	rows := searchAll(t, q)
	require.Len(t, rows, 1)
	assert.Nil(t, rows[0].BestCardName)
	assert.Nil(t, rows[0].BestCardRate)
}
