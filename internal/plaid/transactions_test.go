package plaidclient

import (
	"context"
	"testing"

	"holo/internal/testutil"
	plaid "github.com/plaid/plaid-go/v36/plaid"
)

func TestUpsertTransaction_SkipsRemovedAccount(t *testing.T) {
	ctx := context.Background()
	queries := testutil.NewDB(t)

	// Build a transaction whose AccountId has no matching row in the accounts table.
	txn := plaid.NewTransactionWithDefaults()
	txn.SetAccountId("nonexistent-plaid-account-id")
	txn.SetTransactionId("txn-orphan-001")

	err := upsertTransaction(ctx, queries, *txn)
	if err != nil {
		t.Fatalf("expected no error for removed account, got: %v", err)
	}

	count, err := queries.CountTransactions(ctx)
	if err != nil {
		t.Fatalf("CountTransactions: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 transactions inserted, got %d", count)
	}
}
