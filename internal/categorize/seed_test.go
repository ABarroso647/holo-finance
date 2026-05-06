package categorize

import (
	"context"
	"testing"

	"holo/internal/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlaidParentMap_AllKnownPrimariesHaveParent(t *testing.T) {
	known := []string{
		"FOOD_AND_DRINK", "GENERAL_MERCHANDISE", "TRANSPORTATION",
		"MEDICAL", "ENTERTAINMENT", "HOME_IMPROVEMENT",
		"RENT_AND_UTILITIES", "PERSONAL_CARE", "BANK_FEES",
		"LOAN_PAYMENTS", "EDUCATION", "TRAVEL",
	}
	validParents := map[string]bool{
		"cat_food": true, "cat_shopping": true, "cat_transport": true,
		"cat_health": true, "cat_entertainment": true, "cat_housing": true,
		"cat_personal": true, "cat_finance": true, "cat_education": true,
		"cat_travel": true, "cat_other": true,
	}
	for _, primary := range known {
		parentID := plaidParent(primary)
		assert.True(t, validParents[parentID], "primary %q maps to unknown parent %q", primary, parentID)
	}
}

func TestPlaidParentMap_UnknownPrimaryFallsBackToOther(t *testing.T) {
	assert.Equal(t, "cat_other", plaidParent("SOME_UNKNOWN_CATEGORY"))
}

func TestEnsurePlaidCategorySetsParent(t *testing.T) {
	q := testutil.NewDB(t)
	ctx := context.Background()

	require.NoError(t, SeedCategories(ctx, q))

	catID, err := EnsurePlaidCategory(ctx, q, "FOOD_AND_DRINK_GROCERIES", "FOOD_AND_DRINK")
	require.NoError(t, err)
	assert.Equal(t, "FOOD_AND_DRINK_GROCERIES", catID)

	cat, err := q.GetCategoryByID(ctx, catID)
	require.NoError(t, err)
	require.NotNil(t, cat.ParentID)
	assert.Equal(t, "cat_food", *cat.ParentID)
}

func TestEnsurePlaidCategorySetsParent_Travel(t *testing.T) {
	q := testutil.NewDB(t)
	ctx := context.Background()

	require.NoError(t, SeedCategories(ctx, q))

	catID, err := EnsurePlaidCategory(ctx, q, "TRAVEL_AIRLINES", "TRAVEL")
	require.NoError(t, err)

	cat, err := q.GetCategoryByID(ctx, catID)
	require.NoError(t, err)
	require.NotNil(t, cat.ParentID)
	assert.Equal(t, "cat_travel", *cat.ParentID)
}

func TestSeedCategories_SeedsAllParents(t *testing.T) {
	q := testutil.NewDB(t)
	ctx := context.Background()

	require.NoError(t, SeedCategories(ctx, q))

	categories, err := q.ListCategories(ctx)
	require.NoError(t, err)

	ids := make(map[string]bool)
	for _, c := range categories {
		ids[c.ID] = true
	}

	required := []string{
		"cat_food", "cat_shopping", "cat_transport", "cat_health",
		"cat_entertainment", "cat_housing", "cat_personal", "cat_finance",
		"cat_education", "cat_travel", "cat_other",
	}
	for _, id := range required {
		assert.True(t, ids[id], "missing parent category %q", id)
	}
}
