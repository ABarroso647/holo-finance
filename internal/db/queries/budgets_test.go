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

func setupBudgets(t *testing.T) *dbgen.Queries {
	t.Helper()
	q := testutil.NewDB(t)
	require.NoError(t, categorize.SeedCategories(context.Background(), q))
	return q
}

func TestUpsertBudget_CreatesNew(t *testing.T) {
	q := setupBudgets(t)
	ctx := context.Background()

	budget, err := q.UpsertBudget(ctx, dbgen.UpsertBudgetParams{
		CategoryID:   "cat_food",
		MonthlyLimit: 500.0,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, budget.ID)
	assert.Equal(t, "cat_food", budget.CategoryID)
	assert.InDelta(t, 500.0, budget.MonthlyLimit, 0.01)
}

func TestUpsertBudget_UpdatesExisting(t *testing.T) {
	q := setupBudgets(t)
	ctx := context.Background()

	// Create initial budget
	first, err := q.UpsertBudget(ctx, dbgen.UpsertBudgetParams{
		CategoryID:   "cat_food",
		MonthlyLimit: 300.0,
	})
	require.NoError(t, err)

	// Upsert again with a new limit — should update, not create a second row
	second, err := q.UpsertBudget(ctx, dbgen.UpsertBudgetParams{
		CategoryID:   "cat_food",
		MonthlyLimit: 600.0,
	})
	require.NoError(t, err)

	// Same ID, updated limit
	assert.Equal(t, first.ID, second.ID)
	assert.InDelta(t, 600.0, second.MonthlyLimit, 0.01)

	// Only one row should exist
	rows, err := q.ListBudgets(ctx)
	require.NoError(t, err)
	assert.Len(t, rows, 1, "upsert should produce only one budget row per category")
}

func TestDeleteBudget_Removes(t *testing.T) {
	q := setupBudgets(t)
	ctx := context.Background()

	budget, err := q.UpsertBudget(ctx, dbgen.UpsertBudgetParams{
		CategoryID:   "cat_shopping",
		MonthlyLimit: 200.0,
	})
	require.NoError(t, err)

	err = q.DeleteBudget(ctx, budget.ID)
	require.NoError(t, err)

	rows, err := q.ListBudgets(ctx)
	require.NoError(t, err)
	assert.Len(t, rows, 0, "budget should be deleted")
}

func TestListBudgets_IncludesCategoryName(t *testing.T) {
	q := setupBudgets(t)
	ctx := context.Background()

	_, err := q.UpsertBudget(ctx, dbgen.UpsertBudgetParams{
		CategoryID:   "cat_food",
		MonthlyLimit: 400.0,
	})
	require.NoError(t, err)

	_, err = q.UpsertBudget(ctx, dbgen.UpsertBudgetParams{
		CategoryID:   "cat_transport",
		MonthlyLimit: 150.0,
	})
	require.NoError(t, err)

	rows, err := q.ListBudgets(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 2)

	// Results are ordered by category name — Food & Drink comes before Transport
	assert.Equal(t, "cat_food", rows[0].CategoryID)
	assert.NotEmpty(t, rows[0].CategoryName)
	assert.Equal(t, "cat_transport", rows[1].CategoryID)
	assert.NotEmpty(t, rows[1].CategoryName)
}
