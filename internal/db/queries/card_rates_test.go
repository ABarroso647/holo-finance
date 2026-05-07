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

func setupCardRates(t *testing.T) (*dbgen.Queries, string) {
	t.Helper()
	q := testutil.NewDB(t)
	ctx := context.Background()
	require.NoError(t, categorize.SeedCategories(ctx, q))
	inst := testutil.InsertInstitution(t, q)
	acctID := testutil.InsertAccount(t, q, inst, "credit")
	return q, acctID
}

func insertRate(t *testing.T, q *dbgen.Queries, id, accountID string, categoryID *string, rate float64) {
	t.Helper()
	_, err := q.UpsertCardRewardRate(context.Background(), dbgen.UpsertCardRewardRateParams{
		ID:         id,
		AccountID:  accountID,
		CategoryID: categoryID,
		RewardRate: rate,
	})
	require.NoError(t, err)
}

func strPtr(s string) *string { return &s }

func TestGetBestRateForCategory_ExactMatch(t *testing.T) {
	q, acctID := setupCardRates(t)
	ctx := context.Background()

	insertRate(t, q, "r1", acctID, strPtr("cat_food"), 3.0)

	catID := strPtr("cat_food")
	row, err := q.GetBestRateForCategory(ctx, catID)
	require.NoError(t, err)
	assert.InDelta(t, 3.0, row.RewardRate, 0.001, "exact match should return rate 3.0")
}

func TestGetBestRateForCategory_ParentMatch(t *testing.T) {
	q, acctID := setupCardRates(t)
	ctx := context.Background()

	// cat_food is already seeded as a parent; insert a child category
	testutil.InsertCategoryWithParent(t, q, "FOOD_AND_DRINK_GROCERIES", "cat_food")

	// Rate is set for parent cat_food
	insertRate(t, q, "r1", acctID, strPtr("cat_food"), 3.0)

	// Query with child category — should match via parent
	catID := strPtr("FOOD_AND_DRINK_GROCERIES")
	row, err := q.GetBestRateForCategory(ctx, catID)
	require.NoError(t, err)
	assert.InDelta(t, 3.0, row.RewardRate, 0.001, "should match parent category rate")
}

func TestGetBestRateForCategory_CatchAll(t *testing.T) {
	q, acctID := setupCardRates(t)
	ctx := context.Background()

	// Catch-all rate (NULL category)
	insertRate(t, q, "r1", acctID, nil, 1.0)

	// Query with a category that has no explicit match
	catID := strPtr("cat_food")
	row, err := q.GetBestRateForCategory(ctx, catID)
	require.NoError(t, err)
	assert.InDelta(t, 1.0, row.RewardRate, 0.001, "catch-all rate should be returned when no specific match")
}

func TestGetBestRateForCategory_ExactBeatsCatchAll(t *testing.T) {
	q, acctID := setupCardRates(t)
	ctx := context.Background()

	// Catch-all rate
	insertRate(t, q, "r1", acctID, nil, 1.0)
	// Exact match rate
	insertRate(t, q, "r2", acctID, strPtr("cat_food"), 3.0)

	catID := strPtr("cat_food")
	row, err := q.GetBestRateForCategory(ctx, catID)
	require.NoError(t, err)
	assert.InDelta(t, 3.0, row.RewardRate, 0.001, "exact match should beat catch-all")
}

func TestGetBestRateForCategory_ParentBeatsCatchAll(t *testing.T) {
	q, acctID := setupCardRates(t)
	ctx := context.Background()

	testutil.InsertCategoryWithParent(t, q, "FOOD_AND_DRINK_GROCERIES", "cat_food")

	// Catch-all rate
	insertRate(t, q, "r1", acctID, nil, 1.0)
	// Parent match rate
	insertRate(t, q, "r2", acctID, strPtr("cat_food"), 2.0)

	catID := strPtr("FOOD_AND_DRINK_GROCERIES")
	row, err := q.GetBestRateForCategory(ctx, catID)
	require.NoError(t, err)
	assert.InDelta(t, 2.0, row.RewardRate, 0.001, "parent match should beat catch-all")
}

func TestGetBestRateForCategory_ExactBeatsParent(t *testing.T) {
	q, acctID := setupCardRates(t)
	ctx := context.Background()

	testutil.InsertCategoryWithParent(t, q, "FOOD_AND_DRINK_GROCERIES", "cat_food")

	// Parent rate
	insertRate(t, q, "r1", acctID, strPtr("cat_food"), 2.0)
	// Exact match rate for the child category
	insertRate(t, q, "r2", acctID, strPtr("FOOD_AND_DRINK_GROCERIES"), 5.0)

	catID := strPtr("FOOD_AND_DRINK_GROCERIES")
	row, err := q.GetBestRateForCategory(ctx, catID)
	require.NoError(t, err)
	assert.InDelta(t, 5.0, row.RewardRate, 0.001, "exact match should beat parent match")
}
