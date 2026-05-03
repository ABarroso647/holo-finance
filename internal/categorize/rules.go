package categorize

import (
	"context"
	"regexp"
	"strings"

	"github.com/abarroso647/holo/internal/db/generated"
)

// ApplyRules runs all rules against uncategorized transactions.
// Returns the number of transactions categorized.
func ApplyRules(ctx context.Context, queries *db.Queries) (int, error) {
	rules, err := queries.ListRules(ctx)
	if err != nil {
		return 0, err
	}
	if len(rules) == 0 {
		return 0, nil
	}

	txns, err := queries.ListUncategorizedTransactions(ctx)
	if err != nil {
		return 0, err
	}

	count := 0
	for _, txn := range txns {
		categoryID := matchRules(rules, txn)
		if categoryID == "" {
			continue
		}
		if err := queries.UpdateTransactionCategory(ctx, db.UpdateTransactionCategoryParams{
			CategoryID:     &categoryID,
			CategorySource: "rule",
			ID:             txn.ID,
		}); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

// ApplyRulesToTransaction runs rules against a single transaction and updates it if matched.
func ApplyRulesToTransaction(ctx context.Context, queries *db.Queries, txn db.Transaction) (bool, error) {
	rules, err := queries.ListRules(ctx)
	if err != nil {
		return false, err
	}

	categoryID := matchRules(rules, txn)
	if categoryID == "" {
		return false, nil
	}

	return true, queries.UpdateTransactionCategory(ctx, db.UpdateTransactionCategoryParams{
		CategoryID:     &categoryID,
		CategorySource: "rule",
		ID:             txn.ID,
	})
}

func matchRules(rules []db.ListRulesRow, txn db.Transaction) string {
	for _, rule := range rules {
		var field string
		switch rule.MatchField {
		case "merchant_name":
			if txn.MerchantName != nil {
				field = *txn.MerchantName
			}
		default:
			field = txn.Name
		}

		if matches(rule.MatchType, rule.MatchValue, field) {
			return rule.CategoryID
		}
	}
	return ""
}

func matches(matchType, pattern, value string) bool {
	switch matchType {
	case "equals":
		return strings.EqualFold(value, pattern)
	case "contains":
		return strings.Contains(strings.ToLower(value), strings.ToLower(pattern))
	case "regex":
		re, err := regexp.Compile("(?i)" + pattern)
		if err != nil {
			return false
		}
		return re.MatchString(value)
	}
	return false
}
