package queries_test

import (
	"context"
	"testing"

	"holo/internal/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetPortfolioSummary_SumsAllHoldings(t *testing.T) {
	q := testutil.NewDB(t)
	inst := testutil.InsertInstitution(t, q)
	acctID := testutil.InsertAccount(t, q, inst, "investment")

	testutil.InsertSecurity(t, q, "sec-1", "plaid-sec-1", "Apple Inc", "equity")
	testutil.InsertSecurity(t, q, "sec-2", "plaid-sec-2", "Vanguard ETF", "etf")

	cb1 := 900.0
	testutil.InsertHolding(t, q, "h-1", acctID, "sec-1", 10.0, 1000.0, &cb1)
	testutil.InsertHolding(t, q, "h-2", acctID, "sec-2", 5.0, 500.0, nil)

	row, err := q.GetPortfolioSummary(context.Background())
	require.NoError(t, err)
	assert.InDelta(t, 1500.0, row.TotalValue, 0.01, "total_value should sum both holdings")
}

func TestGetPortfolioSummary_TotalGain_NullCostBasis(t *testing.T) {
	q := testutil.NewDB(t)
	inst := testutil.InsertInstitution(t, q)
	acctID := testutil.InsertAccount(t, q, inst, "investment")

	testutil.InsertSecurity(t, q, "sec-1", "plaid-sec-1", "Some Fund", "etf")
	testutil.InsertHolding(t, q, "h-1", acctID, "sec-1", 10.0, 500.0, nil)

	row, err := q.GetPortfolioSummary(context.Background())
	require.NoError(t, err)
	assert.InDelta(t, 0.0, row.TotalGain, 0.01, "total_gain should be 0 when cost_basis is NULL")
}

func TestGetAllocationByType_GroupsCorrectly(t *testing.T) {
	q := testutil.NewDB(t)
	inst := testutil.InsertInstitution(t, q)
	acctID := testutil.InsertAccount(t, q, inst, "investment")

	testutil.InsertSecurity(t, q, "sec-eq", "plaid-eq", "Apple Inc", "equity")
	testutil.InsertSecurity(t, q, "sec-etf", "plaid-etf", "Vanguard ETF", "etf")

	cb := 700.0
	testutil.InsertHolding(t, q, "h-1", acctID, "sec-eq", 8.0, 800.0, &cb)
	testutil.InsertHolding(t, q, "h-2", acctID, "sec-etf", 2.0, 200.0, nil)

	rows, err := q.GetAllocationByType(context.Background())
	require.NoError(t, err)
	require.Len(t, rows, 2, "should have 2 security type rows")

	assert.Equal(t, "equity", rows[0].SecurityType, "first row should be equity (highest value)")
	assert.InDelta(t, 800.0, rows[0].Value, 0.01)
	assert.Equal(t, "etf", rows[1].SecurityType)
	assert.InDelta(t, 200.0, rows[1].Value, 0.01)
}

func TestGetAllocationByAccount_OrderedByValueDesc(t *testing.T) {
	q := testutil.NewDB(t)
	inst := testutil.InsertInstitution(t, q)

	acct1 := testutil.InsertAccount(t, q, inst, "investment")
	acct2 := testutil.InsertAccountWithID(t, q, inst, "acct-invest2", "investment")

	testutil.InsertSecurity(t, q, "sec-a", "plaid-sec-a", "Bond Fund", "fixed income")

	cb := 280.0
	testutil.InsertHolding(t, q, "h-1", acct1, "sec-a", 3.0, 300.0, &cb)
	testutil.InsertHolding(t, q, "h-2", acct2, "sec-a", 10.0, 1000.0, nil)

	rows, err := q.GetAllocationByAccount(context.Background())
	require.NoError(t, err)
	require.Len(t, rows, 2)
	assert.InDelta(t, 1000.0, rows[0].Value, 0.01, "first row should be highest-value account")
	assert.InDelta(t, 300.0, rows[1].Value, 0.01)
}
