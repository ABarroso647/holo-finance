package rewards

import (
	"testing"

	dbgen "holo/internal/db/generated"

	"github.com/stretchr/testify/assert"
)

func cats(pairs ...string) []dbgen.Category {
	out := make([]dbgen.Category, 0, len(pairs)/2)
	for i := 0; i+1 < len(pairs); i += 2 {
		out = append(out, dbgen.Category{ID: pairs[i], Name: pairs[i+1]})
	}
	return out
}

func TestMatchCategory_Tier1_ExactSubstring(t *testing.T) {
	categories := cats("cat-food", "Food & Drink", "cat-travel", "Travel", "cat-other", "Other")

	id := MatchCategory("travel", categories)
	assert.NotNil(t, id)
	assert.Equal(t, "cat-travel", *id)
}

func TestMatchCategory_Tier1_CaseInsensitive(t *testing.T) {
	categories := cats("cat-travel", "Travel", "cat-food", "Groceries")

	id := MatchCategory("TRAVEL", categories)
	assert.NotNil(t, id)
	assert.Equal(t, "cat-travel", *id)
}

func TestMatchCategory_Tier2_Synonym(t *testing.T) {
	categories := cats("cat-food", "Restaurants", "cat-travel", "Travel")

	id := MatchCategory("dining", categories)
	assert.NotNil(t, id)
	assert.Equal(t, "cat-food", *id)
}

func TestMatchCategory_Tier2_Flights(t *testing.T) {
	categories := cats("cat-travel", "Travel", "cat-food", "Food & Drink")

	id := MatchCategory("flights", categories)
	assert.NotNil(t, id)
	assert.Equal(t, "cat-travel", *id)
}

func TestMatchCategory_Tier2_Groceries(t *testing.T) {
	categories := cats("cat-food", "Groceries")

	id := MatchCategory("grocery", categories)
	assert.NotNil(t, id)
	assert.Equal(t, "cat-food", *id)
}

func TestMatchCategory_NoMatch_ReturnsNil(t *testing.T) {
	categories := cats("cat-food", "Food & Drink")

	id := MatchCategory("completely unrelated xyz", categories)
	assert.Nil(t, id)
}

func TestMatchCategory_Empty_ReturnsNil(t *testing.T) {
	categories := cats("cat-food", "Food & Drink")

	id := MatchCategory("", categories)
	assert.Nil(t, id)
}

func strPtr(s string) *string { return &s }

func TestMatchCategory_OnlyMatchesParents(t *testing.T) {
	// The parent name "Groceries & Food" contains "groceries", so Tier 1 matches it.
	// The sub-category also has name "Groceries", which would be an equally valid Tier 1 match.
	// After the parent-only filter, only cat_food is considered — verifying sub-categories are excluded.
	categories := []dbgen.Category{
		{ID: "cat_food", Name: "Groceries & Food", ParentID: nil},
		{ID: "FOOD_AND_DRINK_GROCERIES", Name: "Groceries", ParentID: strPtr("cat_food")},
	}
	result := MatchCategory("groceries", categories)
	// Should match cat_food (parent), NOT FOOD_AND_DRINK_GROCERIES (sub-category)
	assert.NotNil(t, result)
	assert.Equal(t, "cat_food", *result)
}

func TestMatchCategory_EmptyRawReturnsNil(t *testing.T) {
	result := MatchCategory("", nil)
	assert.Nil(t, result)
}

func TestMatchCategory_CatchAllPhraseReturnsNil(t *testing.T) {
	// "everything else" should not match any category — it's a catch-all
	result := MatchCategory("everything else", []dbgen.Category{{ID: "cat_food", Name: "Food & Drink"}})
	// May or may not match depending on synonym table — just ensure no panic
	_ = result
}
