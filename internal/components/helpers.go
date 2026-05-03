package components

import (
	"fmt"
	"math"
	"net/url"
	"time"

	db "holo/internal/db/generated"
)

func parseYYYYMM(dateStr string) time.Time {
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return time.Now()
	}
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
}

func lastDayOf(t time.Time) time.Time {
	first := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
	return first.AddDate(0, 1, -1)
}

// TxnDateGroup groups transactions under a single calendar date.
type TxnDateGroup struct {
	Date  string
	Label string
	Txns  []db.ListTransactionsRow
	Net   float64 // sum of amounts for the day (positive = net spend)
}

func groupByDate(txns []db.ListTransactionsRow) []TxnDateGroup {
	var groups []TxnDateGroup
	for _, t := range txns {
		if len(groups) == 0 || groups[len(groups)-1].Date != t.Date {
			groups = append(groups, TxnDateGroup{
				Date:  t.Date,
				Label: formatTxnDate(t.Date),
			})
		}
		g := &groups[len(groups)-1]
		g.Txns = append(g.Txns, t)
		g.Net += t.Amount
	}
	return groups
}

func formatTxnDate(date string) string {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return date
	}
	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	d := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
	switch today.Sub(d) {
	case 0:
		return "Today"
	case 24 * time.Hour:
		return "Yesterday"
	}
	return t.Format("Monday, Jan 2, 2006")
}

func txnCategoryColor(txn db.ListTransactionsRow) string {
	if txn.CategoryColor != nil && *txn.CategoryColor != "" {
		return *txn.CategoryColor
	}
	return "#64748b"
}

func txnCategoryName(txn db.ListTransactionsRow) string {
	if txn.CategoryName != nil {
		return *txn.CategoryName
	}
	return ""
}

// filterUnappliedTags returns tags from allTags that are not in applied.
func filterUnappliedTags(allTags, applied []db.Tag) []db.Tag {
	appliedIDs := make(map[string]struct{}, len(applied))
	for _, t := range applied {
		appliedIDs[t.ID] = struct{}{}
	}
	var out []db.Tag
	for _, t := range allTags {
		if _, ok := appliedIDs[t.ID]; !ok {
			out = append(out, t)
		}
	}
	return out
}

func formatDayNet(net float64) string {
	if net >= 0 {
		return fmt.Sprintf("-$%.2f", net)
	}
	return fmt.Sprintf("+$%.2f", math.Abs(net))
}

// TxnFilters holds the active filter state for the transactions page.
type TxnFilters struct {
	Search     string
	AccountID  string
	CategoryID string
	TagID      string
	DateFrom   string
	DateTo     string
	Recurring  string // "1" = only recurring, "" = all
}

func (f TxnFilters) IsActive() bool {
	return f.Search != "" || f.AccountID != "" || f.CategoryID != "" || f.TagID != "" || f.DateFrom != "" || f.DateTo != "" || f.Recurring != ""
}

// QueryString returns filter params as a URL query string (no leading "?", no page param).
func (f TxnFilters) QueryString() string {
	v := url.Values{}
	if f.Search != "" {
		v.Set("q", f.Search)
	}
	if f.AccountID != "" {
		v.Set("account_id", f.AccountID)
	}
	if f.CategoryID != "" {
		v.Set("category_id", f.CategoryID)
	}
	if f.TagID != "" {
		v.Set("tag_id", f.TagID)
	}
	if f.DateFrom != "" {
		v.Set("date_from", f.DateFrom)
	}
	if f.DateTo != "" {
		v.Set("date_to", f.DateTo)
	}
	if f.Recurring != "" {
		v.Set("recurring", f.Recurring)
	}
	return v.Encode()
}

// SpendingRange is a quick-select time range for the spending page.
type SpendingRange struct {
	Label string
	URL   string
}

func spendingRanges() []SpendingRange {
	now := time.Now().UTC()
	today := now.Format("2006-01-02")
	thisMonthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC).Format("2006-01-02")
	lastMonthEnd := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, -1)
	lastMonthStart := time.Date(lastMonthEnd.Year(), lastMonthEnd.Month(), 1, 0, 0, 0, 0, time.UTC).Format("2006-01-02")
	return []SpendingRange{
		{"This Month", fmt.Sprintf("/spending?from=%s&to=%s", thisMonthStart, today)},
		{"Last Month", fmt.Sprintf("/spending?from=%s&to=%s", lastMonthStart, lastMonthEnd.Format("2006-01-02"))},
		{"3M", fmt.Sprintf("/spending?from=%s&to=%s", now.AddDate(0, -3, 0).Format("2006-01-02"), today)},
		{"6M", fmt.Sprintf("/spending?from=%s&to=%s", now.AddDate(0, -6, 0).Format("2006-01-02"), today)},
		{"12M", fmt.Sprintf("/spending?from=%s&to=%s", now.AddDate(0, -12, 0).Format("2006-01-02"), today)},
	}
}
