package plaidclient

import (
	"context"
	"fmt"
	"log"

	db "holo/internal/db/generated"
	plaid "github.com/plaid/plaid-go/v36/plaid"
)

// SyncRecurring fetches recurring transaction streams from Plaid and marks matching
// transactions as recurring. All transactions for the institution are reset first
// so removals from streams are reflected correctly.
func SyncRecurring(ctx context.Context, api *plaid.APIClient, queries *db.Queries, institutionID, accessToken string) error {
	req := plaid.NewTransactionsRecurringGetRequest(accessToken)
	resp, _, err := api.PlaidApi.TransactionsRecurringGet(ctx).TransactionsRecurringGetRequest(*req).Execute()
	if err != nil {
		if plaidErr, ok := err.(plaid.GenericOpenAPIError); ok {
			log.Printf("recurring sync failed: %s", string(plaidErr.Body()))
		}
		return fmt.Errorf("recurring sync: %w", err)
	}

	if err := queries.ResetRecurringForInstitution(ctx, institutionID); err != nil {
		return fmt.Errorf("reset recurring: %w", err)
	}

	var marked int
	for _, stream := range append(resp.GetOutflowStreams(), resp.GetInflowStreams()...) {
		for _, plaidTxnID := range stream.GetTransactionIds() {
			if err := queries.SetTransactionRecurring(ctx, plaidTxnID); err != nil {
				log.Printf("recurring: set failed for %s: %v", plaidTxnID, err)
			} else {
				marked++
			}
		}
	}
	log.Printf("recurring: marked %d transactions for institution %s", marked, institutionID)
	return nil
}
