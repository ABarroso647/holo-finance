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

func setupAnalytics(t *testing.T) (*dbgen.Queries, string) {
	t.Helper()
	q := testutil.NewDB(t)
	ctx := context.Background()
	require.NoError(t, categorize.SeedCategories(ctx, q))
	inst := testutil.InsertInstitution(t, q)
	acctID := testutil.InsertAccount(t, q, inst, "depository")
	return q, acctID
}

func TestGetRecurringSpendForPeriod_OnlyRecurring(t *testing.T) {
	q, acctID := setupAnalytics(t)
	ctx := context.Background()

	testutil.InsertRecurringTransaction(t, q, "t1", acctID, "p1", "Netflix", 15.99, "cat_entertainment", "plaid", "2026-05-01", true)
	testutil.InsertRecurringTransaction(t, q, "t2", acctID, "p2", "One-off", 50.0, "cat_shopping", "plaid", "2026-05-05", false)

	row, err := q.GetRecurringSpendForPeriod(ctx, dbgen.GetRecurringSpendForPeriodParams{
		Date:   "2026-05-01",
		Date_2: "2026-05-31",
	})
	require.NoError(t, err)
	assert.InDelta(t, 15.99, row, 0.01, "only recurring transaction should be counted")
}

func TestGetSpendingByCategory_RollsUpToParent(t *testing.T) {
	q, acctID := setupAnalytics(t)
	ctx := context.Background()

	// Two different food sub-categories both under cat_food
	testutil.InsertCategory(t, q, "FOOD_AND_DRINK_GROCERIES", "Groceries", "#aaa")
	testutil.InsertCategoryWithParent(t, q, "FOOD_AND_DRINK_GROCERIES", "cat_food")
	testutil.InsertCategory(t, q, "FOOD_AND_DRINK_RESTAURANTS", "Restaurants", "#bbb")
	testutil.InsertCategoryWithParent(t, q, "FOOD_AND_DRINK_RESTAURANTS", "cat_food")

	testutil.InsertTransaction(t, q, "t1", acctID, "p1", "Superstore", 50.0, "FOOD_AND_DRINK_GROCERIES", "plaid")
	testutil.InsertTransaction(t, q, "t2", acctID, "p2", "McDonalds", 20.0, "FOOD_AND_DRINK_RESTAURANTS", "plaid")

	rows, err := q.GetSpendingByCategory(ctx, dbgen.GetSpendingByCategoryParams{
		Date:   "2026-01-01",
		Date_2: "2026-12-31",
	})
	require.NoError(t, err)

	assert.Len(t, rows, 1, "two food sub-categories should roll up to one row")
	assert.Equal(t, "cat_food", rows[0].CategoryID)
	assert.Equal(t, "Food & Drink", rows[0].CategoryName)
	assert.InDelta(t, 70.0, rows[0].Total, 0.01)
}

func TestGetSpendingByCategory_CreditSubtractedFromTotal(t *testing.T) {
	q, acctID := setupAnalytics(t)
	ctx := context.Background()

	testutil.InsertCategoryWithParent(t, q, "FOOD_AND_DRINK_RESTAURANTS", "cat_food")

	testutil.InsertTransaction(t, q, "t1", acctID, "p1", "McDonalds", 50.0, "FOOD_AND_DRINK_RESTAURANTS", "plaid")
	testutil.InsertTransaction(t, q, "t2", acctID, "p2", "McDonalds Refund", -10.0, "FOOD_AND_DRINK_RESTAURANTS", "plaid")

	rows, err := q.GetSpendingByCategory(ctx, dbgen.GetSpendingByCategoryParams{
		Date:   "2026-01-01",
		Date_2: "2026-12-31",
	})
	require.NoError(t, err)

	require.Len(t, rows, 1)
	assert.InDelta(t, 40.0, rows[0].Total, 0.01, "refund should be subtracted from spending")
}

func TestGetSpendingByCategory_ExcludesNetCredit(t *testing.T) {
	q, acctID := setupAnalytics(t)
	ctx := context.Background()

	testutil.InsertCategoryWithParent(t, q, "FOOD_AND_DRINK_RESTAURANTS", "cat_food")

	// Net credit (refund > spend)
	testutil.InsertTransaction(t, q, "t1", acctID, "p1", "Purchase", 5.0, "FOOD_AND_DRINK_RESTAURANTS", "plaid")
	testutil.InsertTransaction(t, q, "t2", acctID, "p2", "Refund", -20.0, "FOOD_AND_DRINK_RESTAURANTS", "plaid")

	rows, err := q.GetSpendingByCategory(ctx, dbgen.GetSpendingByCategoryParams{
		Date:   "2026-01-01",
		Date_2: "2026-12-31",
	})
	require.NoError(t, err)
	assert.Len(t, rows, 0, "net credit category should not appear")
}

func TestGetSpendingByCategory_DedupsOther(t *testing.T) {
	q, acctID := setupAnalytics(t)
	ctx := context.Background()

	// Two sub-categories with different primaries both falling to cat_other
	testutil.InsertCategoryWithParent(t, q, "UNKNOWN_CAT_A", "cat_other")
	testutil.InsertCategoryWithParent(t, q, "UNKNOWN_CAT_B", "cat_other")

	testutil.InsertTransaction(t, q, "t1", acctID, "p1", "Misc A", 30.0, "UNKNOWN_CAT_A", "plaid")
	testutil.InsertTransaction(t, q, "t2", acctID, "p2", "Misc B", 20.0, "UNKNOWN_CAT_B", "plaid")

	rows, err := q.GetSpendingByCategory(ctx, dbgen.GetSpendingByCategoryParams{
		Date:   "2026-01-01",
		Date_2: "2026-12-31",
	})
	require.NoError(t, err)
	assert.Len(t, rows, 1, "two cat_other sub-categories should merge into one row")
	assert.Equal(t, "cat_other", rows[0].CategoryID)
	assert.InDelta(t, 50.0, rows[0].Total, 0.01)
}

// ---------------------------------------------------------------------------
// GetRecurringByMerchant tests
// ---------------------------------------------------------------------------

func TestGetRecurringByMerchant_GroupsCorrectly(t *testing.T) {
	q, acctID := setupAnalytics(t)
	ctx := context.Background()

	// 3 Netflix transactions across 3 different months — should produce 1 row with months_seen=3
	testutil.InsertRecurringTransaction(t, q, "t1", acctID, "p1", "Netflix", 15.99, "cat_entertainment", "plaid", "2026-03-01", true)
	testutil.InsertRecurringTransaction(t, q, "t2", acctID, "p2", "Netflix", 15.99, "cat_entertainment", "plaid", "2026-04-01", true)
	testutil.InsertRecurringTransaction(t, q, "t3", acctID, "p3", "Netflix", 15.99, "cat_entertainment", "plaid", "2026-05-01", true)

	rows, err := q.GetRecurringByMerchant(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1, "three Netflix transactions should produce one grouped row")
	assert.Equal(t, "Netflix", rows[0].Merchant)
	assert.Equal(t, int64(3), rows[0].MonthsSeen)
	assert.InDelta(t, 15.99, rows[0].AvgAmount, 0.01)
}

func TestGetRecurringByMerchant_ExcludesNonRecurring(t *testing.T) {
	q, acctID := setupAnalytics(t)
	ctx := context.Background()

	// One recurring, one non-recurring
	testutil.InsertRecurringTransaction(t, q, "t1", acctID, "p1", "Spotify", 9.99, "cat_entertainment", "plaid", "2026-05-01", true)
	testutil.InsertRecurringTransaction(t, q, "t2", acctID, "p2", "One-off", 50.0, "cat_shopping", "plaid", "2026-05-05", false)

	rows, err := q.GetRecurringByMerchant(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1, "only recurring transactions should appear")
	assert.Equal(t, "Spotify", rows[0].Merchant)
}

func TestGetRecurringByMerchant_ExcludesTransferCategory(t *testing.T) {
	q, acctID := setupAnalytics(t)
	ctx := context.Background()

	// Recurring but with a TRANSFER category
	testutil.InsertRecurringTransaction(t, q, "t1", acctID, "p1", "Credit Card Payment", 500.0, "LOAN_PAYMENTS_CREDIT_CARD_PAYMENT", "plaid", "2026-05-01", true)
	// A normal recurring
	testutil.InsertRecurringTransaction(t, q, "t2", acctID, "p2", "Netflix", 15.99, "cat_entertainment", "plaid", "2026-05-01", true)

	rows, err := q.GetRecurringByMerchant(ctx)
	require.NoError(t, err)
	for _, row := range rows {
		assert.NotEqual(t, "Credit Card Payment", row.Merchant, "TRANSFER category recurring should be excluded")
	}
	assert.Len(t, rows, 1, "only Netflix should appear")
}

func TestGetRecurringByMerchant_UsesMerchantNameFallback(t *testing.T) {
	q, acctID := setupAnalytics(t)
	ctx := context.Background()

	// Insert a transaction with merchant_name set — should group by merchant_name, not name
	merchantName := "Netflix Inc."
	_, err := q.UpsertTransaction(ctx, dbgen.UpsertTransactionParams{
		ID:                 "t1",
		AccountID:          acctID,
		PlaidTransactionID: "p1",
		Date:               "2026-05-01",
		Name:               "NETFLIX.COM",
		MerchantName:       &merchantName,
		Amount:             15.99,
		Currency:           "CAD",
		IsRecurring:        1,
	})
	require.NoError(t, err)

	// Another transaction with same merchant_name but different name
	_, err = q.UpsertTransaction(ctx, dbgen.UpsertTransactionParams{
		ID:                 "t2",
		AccountID:          acctID,
		PlaidTransactionID: "p2",
		Date:               "2026-04-01",
		Name:               "Netflix Monthly",
		MerchantName:       &merchantName,
		Amount:             15.99,
		Currency:           "CAD",
		IsRecurring:        1,
	})
	require.NoError(t, err)

	rows, err := q.GetRecurringByMerchant(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1, "two transactions with same merchant_name should group into one row")
	assert.Equal(t, "Netflix Inc.", rows[0].Merchant, "merchant_name should be used when available")
	assert.Equal(t, int64(2), rows[0].MonthsSeen)
}

func TestGetRecurringByMerchant_RespectsExclusions(t *testing.T) {
	q, acctID := setupAnalytics(t)
	ctx := context.Background()

	testutil.InsertRecurringTransaction(t, q, "t1", acctID, "p1", "Netflix", 15.99, "cat_entertainment", "plaid", "2026-05-01", true)
	testutil.InsertRecurringTransaction(t, q, "t2", acctID, "p2", "Spotify", 9.99, "cat_entertainment", "plaid", "2026-05-02", true)

	// Exclude Netflix
	err := q.ExcludeMerchantFromRecurring(ctx, "Netflix")
	require.NoError(t, err)

	rows, err := q.GetRecurringByMerchant(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1, "excluded merchant should not appear")
	assert.Equal(t, "Spotify", rows[0].Merchant)
}
