package categorize

import "time"

// IsMonthComplete returns true if the given month is safe to include in spending averages.
// A past month is always complete. The current month is complete only if the paycheck
// has landed (today >= salaryDayOfMonth). If salaryDayOfMonth <= 0 (unknown), the
// current month is always considered incomplete.
func IsMonthComplete(month time.Time, salaryDayOfMonth int, now time.Time) bool {
	monthStart := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, month.Location())
	currentStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	if monthStart.Before(currentStart) {
		return true
	}
	if monthStart.After(currentStart) {
		return false
	}
	if salaryDayOfMonth <= 0 {
		return false
	}
	return now.Day() >= salaryDayOfMonth
}
