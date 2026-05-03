package plaidclient

import (
	"context"
	"fmt"
	"log"

	db "holo/internal/db/generated"
	plaid "github.com/plaid/plaid-go/v36/plaid"
)

// SyncLiabilities fetches credit card liability data for one institution.
// Returns an error if the Liabilities product is not enabled or not yet approved.
func SyncLiabilities(ctx context.Context, api *plaid.APIClient, queries *db.Queries, accessToken string) error {
	req := plaid.NewLiabilitiesGetRequest(accessToken)
	resp, _, err := api.PlaidApi.LiabilitiesGet(ctx).LiabilitiesGetRequest(*req).Execute()
	if err != nil {
		if plaidErr, ok := err.(plaid.GenericOpenAPIError); ok {
			log.Printf("liabilities sync failed: %s", string(plaidErr.Body()))
			return fmt.Errorf("Plaid Liabilities not available: %s", string(plaidErr.Body()))
		}
		return fmt.Errorf("liabilities sync: %w", err)
	}

	liabs := resp.GetLiabilities()
	credit := liabs.GetCredit()
	for _, cc := range credit {
		plaidAccountID := cc.GetAccountId()
		acct, err := queries.GetAccountByPlaidID(ctx, plaidAccountID)
		if err != nil {
			log.Printf("liabilities: account not found for plaid_id %s: %v", plaidAccountID, err)
			continue
		}

		isOverdue := int64(0)
		if ok, set := cc.GetIsOverdueOk(); set && ok != nil && *ok {
			isOverdue = 1
		}

		params := db.UpsertCreditCardLiabilityParams{
			AccountID: acct.ID,
			IsOverdue: isOverdue,
		}

		if v, ok := cc.GetLastStatementBalanceOk(); ok && v != nil {
			params.LastStatementBalance = v
		}
		if v, ok := cc.GetLastStatementIssueDateOk(); ok && v != nil {
			params.LastStatementIssueDate = v
		}
		if v, ok := cc.GetLastPaymentDateOk(); ok && v != nil {
			params.LastPaymentDate = v
		}
		if v, ok := cc.GetLastPaymentAmountOk(); ok && v != nil {
			params.LastPaymentAmount = v
		}
		if v, ok := cc.GetMinimumPaymentAmountOk(); ok && v != nil {
			params.MinimumPaymentAmount = v
		}
		if v, ok := cc.GetNextPaymentDueDateOk(); ok && v != nil {
			params.NextPaymentDueDate = v
		}

		if err := queries.UpsertCreditCardLiability(ctx, params); err != nil {
			log.Printf("liabilities: upsert failed for account %s: %v", acct.ID, err)
		}
	}
	return nil
}
