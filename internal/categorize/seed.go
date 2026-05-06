package categorize

import (
	"context"
	"database/sql"
	"fmt"
	"hash/fnv"
	"strings"

	"holo/internal/db/generated"
)

// parentCategories are the top-level categories seeded on every startup.
// Plaid sub-categories are created dynamically via EnsurePlaidCategory and
// linked to these parents.
var parentCategories = []db.UpsertCategoryParams{
	{ID: "cat_food", Name: "Food & Drink", Color: strPtr("#f97316")},
	{ID: "cat_shopping", Name: "Shopping", Color: strPtr("#ec4899")},
	{ID: "cat_transport", Name: "Transportation", Color: strPtr("#3b82f6")},
	{ID: "cat_health", Name: "Health & Medical", Color: strPtr("#22c55e")},
	{ID: "cat_entertainment", Name: "Entertainment", Color: strPtr("#8b5cf6")},
	{ID: "cat_housing", Name: "Housing & Utilities", Color: strPtr("#06b6d4")},
	{ID: "cat_personal", Name: "Personal Care", Color: strPtr("#f59e0b")},
	{ID: "cat_finance", Name: "Finance & Fees", Color: strPtr("#64748b")},
	{ID: "cat_education", Name: "Education", Color: strPtr("#14b8a6")},
	{ID: "cat_travel", Name: "Travel", Color: strPtr("#0ea5e9")},
	{ID: "cat_other", Name: "Other", Color: strPtr("#94a3b8")},
}

// plaidPrimaryToParent maps Plaid primary category strings to our parent IDs.
var plaidPrimaryToParent = map[string]string{
	"FOOD_AND_DRINK":      "cat_food",
	"GENERAL_MERCHANDISE": "cat_shopping",
	"TRANSPORTATION":      "cat_transport",
	"MEDICAL":             "cat_health",
	"ENTERTAINMENT":       "cat_entertainment",
	"HOME_IMPROVEMENT":    "cat_housing",
	"RENT_AND_UTILITIES":  "cat_housing",
	"PERSONAL_CARE":       "cat_personal",
	"BANK_FEES":           "cat_finance",
	"LOAN_PAYMENTS":       "cat_finance",
	"EDUCATION":           "cat_education",
	"TRAVEL":              "cat_travel",
}

// plaidParent returns the parent category ID for a Plaid primary category string.
func plaidParent(primary string) string {
	if id, ok := plaidPrimaryToParent[primary]; ok {
		return id
	}
	return "cat_other"
}

func SeedCategories(ctx context.Context, queries *db.Queries) error {
	for _, cat := range parentCategories {
		if _, err := queries.UpsertCategory(ctx, cat); err != nil {
			return err
		}
	}
	return nil
}

// EnsurePlaidCategory upserts a category row using Plaid's detailed category string
// verbatim as the ID. Name is derived by stripping the primary prefix and title-casing.
// Color is deterministically generated from the category ID so it's stable across runs.
func EnsurePlaidCategory(ctx context.Context, queries *db.Queries, detailed, primary string) (string, error) {
	id := detailed
	if id == "" {
		id = primary
	}
	if id == "" {
		return "", nil
	}

	name := prettifyPlaidCategory(detailed, primary)
	color := hashColor(id)
	parentID := plaidParent(primary)

	if _, err := queries.UpsertCategory(ctx, db.UpsertCategoryParams{
		ID:       id,
		Name:     name,
		Color:    &color,
		ParentID: &parentID,
	}); err != nil {
		return "", err
	}
	return id, nil
}

// prettifyPlaidCategory strips the primary prefix and title-cases the remainder.
// "FOOD_AND_DRINK_GROCERIES" + primary "FOOD_AND_DRINK" → "Groceries"
// "TRANSPORTATION_GAS_STATION" + primary "TRANSPORTATION" → "Gas Station"
func prettifyPlaidCategory(detailed, primary string) string {
	name := detailed
	if primary != "" && strings.HasPrefix(detailed, primary+"_") {
		name = detailed[len(primary)+1:]
	}
	parts := strings.Split(strings.ToLower(strings.ReplaceAll(name, "_", " ")), " ")
	for i, p := range parts {
		if len(p) > 0 {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(parts, " ")
}

// hashColor deterministically maps a string to a hex color from a curated palette.
// Same input always produces the same color; no hardcoded category names.
func hashColor(s string) string {
	palette := []string{
		"#f97316", "#ec4899", "#3b82f6", "#6366f1", "#8b5cf6",
		"#06b6d4", "#84cc16", "#f59e0b", "#14b8a6", "#0ea5e9",
		"#a78bfa", "#4ade80", "#94a3b8", "#fbbf24", "#f87171",
		"#fb923c", "#e879f9", "#22c55e", "#64748b", "#38bdf8",
	}
	h := fnv.New32a()
	fmt.Fprint(h, s)
	return palette[h.Sum32()%uint32(len(palette))]
}

func strPtr(s string) *string {
	return &s
}

func nullStr(s string) sql.NullString {
	return sql.NullString{String: s, Valid: s != ""}
}
