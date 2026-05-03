package plaidclient

import (
	"context"
	"database/sql"
	"fmt"

	"holo/internal/categorize"
	"holo/internal/db/generated"
	"github.com/google/uuid"
	plaid "github.com/plaid/plaid-go/v36/plaid"
)

type SyncResult struct {
	Added    int
	Modified int
	Removed  int
}

func SyncTransactions(ctx context.Context, api *plaid.APIClient, queries *db.Queries, itemID, accessToken string) (SyncResult, error) {
	cursorRow, err := queries.GetCursor(ctx, itemID)
	var cursor string
	if err == nil && cursorRow != nil {
		cursor = *cursorRow
	}

	var result SyncResult
	hasMore := true

	for hasMore {
		req := plaid.NewTransactionsSyncRequest(accessToken)
		if cursor != "" {
			req.SetCursor(cursor)
		}
		req.SetCount(500)

		resp, _, err := api.PlaidApi.TransactionsSync(ctx).TransactionsSyncRequest(*req).Execute()
		if err != nil {
			return result, fmt.Errorf("transactions sync: %w", err)
		}

		for _, txn := range resp.GetAdded() {
			if err := upsertTransaction(ctx, queries, txn); err != nil {
				return result, err
			}
			result.Added++
		}
		for _, txn := range resp.GetModified() {
			if err := upsertTransaction(ctx, queries, txn); err != nil {
				return result, err
			}
			result.Modified++
		}
		for _, removed := range resp.GetRemoved() {
			if err := queries.DeleteTransaction(ctx, removed.GetTransactionId()); err != nil {
				return result, fmt.Errorf("delete transaction: %w", err)
			}
			result.Removed++
		}

		cursor = resp.GetNextCursor()
		hasMore = resp.GetHasMore()
	}

	if err := queries.UpsertCursor(ctx, db.UpsertCursorParams{
		ItemID: itemID,
		Cursor: &cursor,
	}); err != nil {
		return result, fmt.Errorf("upsert cursor: %w", err)
	}

	return result, nil
}

func upsertTransaction(ctx context.Context, queries *db.Queries, txn plaid.Transaction) error {
	accountID, err := getAccountIDByPlaidID(ctx, queries, txn.GetAccountId())
	if err != nil {
		return err
	}

	var authorizedDate *string
	if d, ok := txn.GetAuthorizedDateOk(); ok && d != nil && *d != "" {
		authorizedDate = d
	}

	var merchantName *string
	if m := txn.GetMerchantName(); m != "" {
		merchantName = &m
	}

	date := txn.GetDate()
	pending := txn.GetPending()
	isRecurring := false

	// Use Plaid's personal_finance_category verbatim — no translation layer.
	// EnsurePlaidCategory auto-upserts the category row on first encounter.
	var categoryID *string
	var categoryConfidence *string
	categorySource := "uncategorized"
	if pfc, ok := txn.GetPersonalFinanceCategoryOk(); ok && pfc != nil {
		id, err := categorize.EnsurePlaidCategory(ctx, queries, pfc.GetDetailed(), pfc.GetPrimary())
		if err != nil {
			return err
		}
		if id != "" {
			categoryID = &id
			categorySource = "plaid"
			if conf := pfc.GetConfidenceLevel(); conf != "" {
				categoryConfidence = &conf
			}
		}
	}

	_, err = queries.UpsertTransaction(ctx, db.UpsertTransactionParams{
		ID:                 uuid.New().String(),
		AccountID:          accountID,
		PlaidTransactionID: txn.GetTransactionId(),
		Date:               date,
		AuthorizedDate:     authorizedDate,
		Name:               txn.GetName(),
		MerchantName:       merchantName,
		Amount:             txn.GetAmount(),
		Currency:           txn.GetIsoCurrencyCode(),
		CategoryID:         categoryID,
		CategorySource:     categorySource,
		CategoryConfidence: categoryConfidence,
		Pending:            boolToInt(pending),
		IsRecurring:        boolToInt(isRecurring),
	})
	return err
}

func getAccountIDByPlaidID(ctx context.Context, queries *db.Queries, plaidAccountID string) (string, error) {
	acct, err := queries.GetAccountByPlaidID(ctx, plaidAccountID)
	if err != nil {
		return "", fmt.Errorf("account not found for plaid_account_id %s: %w", plaidAccountID, err)
	}
	return acct.ID, nil
}

func boolToInt(b bool) int64 {
	if b {
		return 1
	}
	return 0
}

func SyncAccounts(ctx context.Context, api *plaid.APIClient, queries *db.Queries, institutionID, accessToken string) error {
	req := plaid.NewAccountsGetRequest(accessToken)
	resp, _, err := api.PlaidApi.AccountsGet(ctx).AccountsGetRequest(*req).Execute()
	if err != nil {
		return fmt.Errorf("accounts get: %w", err)
	}

	for _, acct := range resp.GetAccounts() {
		var currentBalance, availableBalance *float64
		if b, ok := acct.Balances.GetCurrentOk(); ok {
			currentBalance = b
		}
		if b, ok := acct.Balances.GetAvailableOk(); ok {
			availableBalance = b
		}

		currency := acct.Balances.GetIsoCurrencyCode()
		if currency == "" {
			currency = "CAD"
		}

		subtype := string(acct.GetSubtype())
		officialName := acct.GetOfficialName()

		if _, err := queries.UpsertAccount(ctx, db.UpsertAccountParams{
			ID:               uuid.New().String(),
			InstitutionID:    institutionID,
			PlaidAccountID:   acct.GetAccountId(),
			Name:             acct.GetName(),
			OfficialName:     &officialName,
			Type:             string(acct.GetType()),
			Subtype:          &subtype,
			Currency:         currency,
			CurrentBalance:   currentBalance,
			AvailableBalance: availableBalance,
		}); err != nil {
			return fmt.Errorf("upsert account %s: %w", acct.GetAccountId(), err)
		}
	}
	return nil
}

func nullableString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func nullableSQL(s string) sql.NullString {
	return sql.NullString{String: s, Valid: s != ""}
}
