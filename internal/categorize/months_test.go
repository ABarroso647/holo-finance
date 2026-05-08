package categorize

import (
	"testing"
	"time"
)

func TestIsMonthComplete_PastMonth(t *testing.T) {
	now := time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC)
	pastMonth := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	if !IsMonthComplete(pastMonth, 15, now) {
		t.Error("expected past month to always be complete")
	}
}

func TestIsMonthComplete_CurrentMonthNoSalary(t *testing.T) {
	now := time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC)
	currentMonth := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	if IsMonthComplete(currentMonth, 0, now) {
		t.Error("expected current month with salaryDay=0 to be incomplete")
	}
}

func TestIsMonthComplete_CurrentMonthBeforePayday(t *testing.T) {
	now := time.Date(2026, 5, 7, 0, 0, 0, 0, time.UTC)
	currentMonth := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	if IsMonthComplete(currentMonth, 15, now) {
		t.Error("expected current month to be incomplete when today (7) < salaryDay (15)")
	}
}

func TestIsMonthComplete_CurrentMonthOnPayday(t *testing.T) {
	now := time.Date(2026, 5, 15, 0, 0, 0, 0, time.UTC)
	currentMonth := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	if !IsMonthComplete(currentMonth, 15, now) {
		t.Error("expected current month to be complete when today (15) == salaryDay (15)")
	}
}

func TestIsMonthComplete_CurrentMonthAfterPayday(t *testing.T) {
	now := time.Date(2026, 5, 20, 0, 0, 0, 0, time.UTC)
	currentMonth := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	if !IsMonthComplete(currentMonth, 15, now) {
		t.Error("expected current month to be complete when today (20) > salaryDay (15)")
	}
}
