package plaidclient

import (
	"context"
	"fmt"

	db "github.com/abarroso647/holo/internal/db/generated"
	"github.com/google/uuid"
	plaid "github.com/plaid/plaid-go/v36/plaid"
)

// SyncInvestments fetches holdings and securities for all accounts under a given access token
// and upserts them into the local DB.
func SyncInvestments(ctx context.Context, api *plaid.APIClient, queries *db.Queries, accessToken string) (int, error) {
	req := plaid.NewInvestmentsHoldingsGetRequest(accessToken)
	resp, _, err := api.PlaidApi.InvestmentsHoldingsGet(ctx).InvestmentsHoldingsGetRequest(*req).Execute()
	if err != nil {
		return 0, fmt.Errorf("investments holdings get: %w", err)
	}

	// Build plaid_security_id → internal security ID map
	securityIDMap := make(map[string]string, len(resp.GetSecurities()))

	for _, s := range resp.GetSecurities() {
		plaidSecID := s.GetSecurityId()
		if plaidSecID == "" {
			continue
		}

		// Reuse existing ID if present, otherwise create one
		existing, err := queries.GetSecurityByPlaidID(ctx, plaidSecID)
		var secID string
		if err == nil {
			secID = existing.ID
		} else {
			secID = uuid.New().String()
		}
		securityIDMap[plaidSecID] = secID

		ticker := nilIfEmpty(s.GetTickerSymbol())
		secType := nilIfEmpty(string(s.GetType()))
		closePrice := nilIfZero(s.GetClosePrice())
		closeDate := nilIfEmpty(s.GetClosePriceAsOf())

		currency := s.GetIsoCurrencyCode()
		if currency == "" {
			currency = "CAD"
		}

		if err := queries.UpsertSecurity(ctx, db.UpsertSecurityParams{
			ID:              secID,
			PlaidSecurityID: plaidSecID,
			TickerSymbol:    ticker,
			Name:            s.GetName(),
			Type:            secType,
			Currency:        currency,
			ClosePrice:      closePrice,
			ClosePriceAsOf:  closeDate,
		}); err != nil {
			return 0, fmt.Errorf("upsert security %s: %w", plaidSecID, err)
		}
	}

	count := 0
	for _, h := range resp.GetHoldings() {
		plaidAccID := h.GetAccountId()
		plaidSecID := h.GetSecurityId()

		acct, err := queries.GetAccountByPlaidID(ctx, plaidAccID)
		if err != nil {
			// Account not yet synced — skip this holding
			continue
		}

		secID, ok := securityIDMap[plaidSecID]
		if !ok {
			continue
		}

		costBasis := nilIfZero(h.GetCostBasis())
		price := nilIfZero(h.GetInstitutionPrice())
		value := nilIfZero(h.GetInstitutionValue())

		currency := h.GetIsoCurrencyCode()
		if currency == "" {
			currency = "CAD"
		}

		if err := queries.UpsertHolding(ctx, db.UpsertHoldingParams{
			ID:               uuid.New().String(),
			AccountID:        acct.ID,
			SecurityID:       secID,
			Quantity:         h.GetQuantity(),
			CostBasis:        costBasis,
			InstitutionPrice: price,
			InstitutionValue: value,
			Currency:         currency,
		}); err != nil {
			return count, fmt.Errorf("upsert holding: %w", err)
		}
		count++
	}

	return count, nil
}

func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func nilIfZero(f float64) *float64 {
	if f == 0 {
		return nil
	}
	return &f
}
