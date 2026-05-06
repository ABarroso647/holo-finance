package testutil

import (
	"context"
	"testing"

	dbgen "holo/internal/db/generated"
)

func InsertCategory(t *testing.T, q *dbgen.Queries, id, name, color string) {
	t.Helper()
	if _, err := q.UpsertCategory(context.Background(), dbgen.UpsertCategoryParams{
		ID:    id,
		Name:  name,
		Color: &color,
	}); err != nil {
		t.Fatalf("InsertCategory %q: %v", id, err)
	}
}

func InsertInstitution(t *testing.T, q *dbgen.Queries) string {
	t.Helper()
	inst, err := q.UpsertInstitution(context.Background(), dbgen.UpsertInstitutionParams{
		ID:               "inst-test",
		PlaidItemID:      "item-test",
		PlaidAccessToken: "tok-test",
		Name:             "Test Bank",
	})
	if err != nil {
		t.Fatalf("InsertInstitution: %v", err)
	}
	return inst.ID
}

func InsertAccount(t *testing.T, q *dbgen.Queries, institutionID, accountType string) string {
	t.Helper()
	acctID := "acct-" + accountType
	if _, err := q.UpsertAccount(context.Background(), dbgen.UpsertAccountParams{
		ID:             acctID,
		InstitutionID:  institutionID,
		PlaidAccountID: "plaid-" + acctID,
		Name:           "Test " + accountType,
		Type:           accountType,
		Currency:       "CAD",
	}); err != nil {
		t.Fatalf("InsertAccount %q: %v", accountType, err)
	}
	return acctID
}

func InsertTransaction(t *testing.T, q *dbgen.Queries, id, accountID, plaidID, name string, amount float64, categoryID, categorySource string) {
	t.Helper()
	InsertTransactionOnDate(t, q, id, accountID, plaidID, name, amount, categoryID, categorySource, "2026-05-01")
}

func InsertTransactionOnDate(t *testing.T, q *dbgen.Queries, id, accountID, plaidID, name string, amount float64, categoryID, categorySource, date string) {
	t.Helper()
	var catPtr *string
	if categoryID != "" {
		catPtr = &categoryID
	}
	if _, err := q.UpsertTransaction(context.Background(), dbgen.UpsertTransactionParams{
		ID:                 id,
		AccountID:          accountID,
		PlaidTransactionID: plaidID,
		Date:               date,
		Name:               name,
		Amount:             amount,
		Currency:           "CAD",
		CategoryID:         catPtr,
		CategorySource:     categorySource,
	}); err != nil {
		t.Fatalf("InsertTransactionOnDate %q: %v", id, err)
	}
}

func InsertRecurringTransaction(t *testing.T, q *dbgen.Queries, id, accountID, plaidID, name string, amount float64, categoryID, categorySource, date string, isRecurring bool) {
	t.Helper()
	var catPtr *string
	if categoryID != "" {
		catPtr = &categoryID
	}
	recurring := int64(0)
	if isRecurring {
		recurring = 1
	}
	if _, err := q.UpsertTransaction(context.Background(), dbgen.UpsertTransactionParams{
		ID:                 id,
		AccountID:          accountID,
		PlaidTransactionID: plaidID,
		Date:               date,
		Name:               name,
		Amount:             amount,
		Currency:           "CAD",
		CategoryID:         catPtr,
		CategorySource:     categorySource,
		IsRecurring:        recurring,
	}); err != nil {
		t.Fatalf("InsertRecurringTransaction %q: %v", id, err)
	}
}

func InsertCategoryWithParent(t *testing.T, q *dbgen.Queries, id, parentID string) {
	t.Helper()
	color := "#aaaaaa"
	if _, err := q.UpsertCategory(context.Background(), dbgen.UpsertCategoryParams{
		ID:       id,
		Name:     id,
		Color:    &color,
		ParentID: &parentID,
	}); err != nil {
		t.Fatalf("InsertCategoryWithParent %q: %v", id, err)
	}
}

func InsertRule(t *testing.T, q *dbgen.Queries, id, matchType, matchField, matchValue, categoryID string) {
	t.Helper()
	if _, err := q.InsertRule(context.Background(), dbgen.InsertRuleParams{
		ID:         id,
		MatchType:  matchType,
		MatchField: matchField,
		MatchValue: matchValue,
		CategoryID: categoryID,
		Priority:   100,
	}); err != nil {
		t.Fatalf("InsertRule %q: %v", id, err)
	}
}
