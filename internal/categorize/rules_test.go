package categorize

import (
	"context"
	"testing"

	"holo/internal/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMatches(t *testing.T) {
	tests := []struct {
		matchType string
		pattern   string
		value     string
		want      bool
	}{
		{"equals", "Netflix", "Netflix", true},
		{"equals", "netflix", "Netflix", true},
		{"equals", "Netflix", "Netflix Inc", false},
		{"contains", "netflix", "NETFLIX SUBSCRIPTION", true},
		{"contains", "uber", "Uber Eats", true},
		{"contains", "netflix", "Spotify", false},
		{"regex", `^NETFLIX`, "NETFLIX SUBSCRIPTION", true},
		{"regex", `^NETFLIX`, "PAY NETFLIX", false},
		{"regex", `(?i)tim hortons`, "Tim Hortons #42", true},
		{"unknown", "x", "x", false},
	}
	for _, tc := range tests {
		got := matches(tc.matchType, tc.pattern, tc.value)
		assert.Equal(t, tc.want, got, "%s(%q, %q)", tc.matchType, tc.pattern, tc.value)
	}
}

func TestMatchRules_FirstMatchWins(t *testing.T) {
	q := testutil.NewDB(t)
	ctx := context.Background()

	testutil.InsertCategory(t, q, "cat-streaming", "Streaming", "#6366f1")
	testutil.InsertCategory(t, q, "cat-food", "Food", "#22c55e")

	testutil.InsertRule(t, q, "r1", "contains", "name", "netflix", "cat-streaming")
	testutil.InsertRule(t, q, "r2", "contains", "name", "net", "cat-food")

	rules, err := q.ListRules(ctx)
	require.NoError(t, err)

	inst := testutil.InsertInstitution(t, q)
	acctID := testutil.InsertAccount(t, q, inst, "depository")
	testutil.InsertTransaction(t, q, "txn1", acctID, "plaid-txn1", "NETFLIX SUBSCRIPTION", 15.99, "", "uncategorized")

	txns, err := q.ListUncategorizedTransactions(ctx)
	require.NoError(t, err)
	require.Len(t, txns, 1)

	got := matchRules(rules, txns[0])
	assert.Equal(t, "cat-streaming", got)
}

func TestApplyRules_CategorizesUncategorized(t *testing.T) {
	q := testutil.NewDB(t)
	ctx := context.Background()

	testutil.InsertCategory(t, q, "cat-streaming", "Streaming", "#6366f1")
	testutil.InsertRule(t, q, "r1", "contains", "name", "netflix", "cat-streaming")

	inst := testutil.InsertInstitution(t, q)
	acctID := testutil.InsertAccount(t, q, inst, "depository")
	testutil.InsertTransaction(t, q, "txn1", acctID, "p1", "NETFLIX SUBSCRIPTION", 15.99, "", "uncategorized")
	testutil.InsertTransaction(t, q, "txn2", acctID, "p2", "Tim Hortons", 4.50, "", "uncategorized")

	n, err := ApplyRules(ctx, q)
	require.NoError(t, err)
	assert.Equal(t, 1, n)

	uncategorized, err := q.ListUncategorizedTransactions(ctx)
	require.NoError(t, err)
	assert.Len(t, uncategorized, 1)
	assert.Equal(t, "Tim Hortons", uncategorized[0].Name)
}

func TestApplyRules_DoesNotOverwriteManual(t *testing.T) {
	q := testutil.NewDB(t)
	ctx := context.Background()

	testutil.InsertCategory(t, q, "cat-streaming", "Streaming", "#6366f1")
	testutil.InsertCategory(t, q, "cat-manual", "Manual Cat", "#aabbcc")
	testutil.InsertRule(t, q, "r1", "contains", "name", "netflix", "cat-streaming")

	inst := testutil.InsertInstitution(t, q)
	acctID := testutil.InsertAccount(t, q, inst, "depository")
	testutil.InsertTransaction(t, q, "txn1", acctID, "p1", "NETFLIX SUBSCRIPTION", 15.99, "cat-manual", "manual")

	n, err := ApplyRules(ctx, q)
	require.NoError(t, err)
	assert.Equal(t, 0, n)
}

func TestApplyRuleToNonManual_BackfillsExisting(t *testing.T) {
	q := testutil.NewDB(t)
	ctx := context.Background()

	testutil.InsertCategory(t, q, "cat-streaming", "Streaming", "#6366f1")
	testutil.InsertCategory(t, q, "cat-food", "Food", "#22c55e")
	testutil.InsertCategory(t, q, "cat-manual-cat", "Manual Cat", "#aabbcc")

	inst := testutil.InsertInstitution(t, q)
	acctID := testutil.InsertAccount(t, q, inst, "depository")
	testutil.InsertTransaction(t, q, "txn1", acctID, "p1", "NETFLIX SUBSCRIPTION", 15.99, "cat-food", "plaid")
	testutil.InsertTransaction(t, q, "txn2", acctID, "p2", "Netflix Monthly", 15.99, "cat-food", "rule")
	testutil.InsertTransaction(t, q, "txn3", acctID, "p3", "NETFLIX SUBSCRIPTION", 15.99, "cat-manual-cat", "manual")

	testutil.InsertRule(t, q, "r1", "contains", "name", "netflix", "cat-streaming")
	rules, err := q.ListRules(ctx)
	require.NoError(t, err)

	n, err := ApplyRuleToNonManual(ctx, q, rules[0])
	require.NoError(t, err)
	assert.Equal(t, 2, n)
}

func TestApplyRuleToNonManual_DoesNotTouchManual(t *testing.T) {
	q := testutil.NewDB(t)
	ctx := context.Background()

	testutil.InsertCategory(t, q, "cat-streaming", "Streaming", "#6366f1")
	testutil.InsertCategory(t, q, "cat-manual-cat", "Manual Cat", "#aabbcc")

	inst := testutil.InsertInstitution(t, q)
	acctID := testutil.InsertAccount(t, q, inst, "depository")
	testutil.InsertTransaction(t, q, "txn1", acctID, "p1", "NETFLIX SUBSCRIPTION", 15.99, "cat-manual-cat", "manual")

	testutil.InsertRule(t, q, "r1", "contains", "name", "netflix", "cat-streaming")
	rules, err := q.ListRules(ctx)
	require.NoError(t, err)

	n, err := ApplyRuleToNonManual(ctx, q, rules[0])
	require.NoError(t, err)
	assert.Equal(t, 0, n)
}
