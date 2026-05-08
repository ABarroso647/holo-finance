package invest

import (
	"testing"
	"time"
)

func date(year, month, day int) time.Time {
	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
}

func TestCurrentTFSARoom_SameYear(t *testing.T) {
	refDate := date(2025, 3, 15)
	now := date(2025, 9, 1)
	room := CurrentTFSARoom(10000, refDate, now)
	if room != 10000 {
		t.Errorf("expected 10000, got %.2f", room)
	}
}

func TestCurrentTFSARoom_OneYearElapsed(t *testing.T) {
	refDate := date(2024, 6, 1)
	now := date(2025, 6, 1)
	room := CurrentTFSARoom(10000, refDate, now)
	expected := 10000 + TFSAAnnualLimit
	if room != expected {
		t.Errorf("expected %.2f, got %.2f", expected, room)
	}
}

func TestCurrentTFSARoom_TwoYearsElapsed(t *testing.T) {
	refDate := date(2023, 6, 1)
	now := date(2025, 6, 1)
	room := CurrentTFSARoom(5000, refDate, now)
	expected := 5000 + 2*TFSAAnnualLimit
	if room != expected {
		t.Errorf("expected %.2f, got %.2f", expected, room)
	}
}

func TestCurrentTFSARoom_ZeroReferenceDate(t *testing.T) {
	var zeroDate time.Time
	now := date(2025, 6, 1)
	room := CurrentTFSARoom(10000, zeroDate, now)
	// zero referenceDate is treated as now, so no Jan 1sts have passed
	if room != 10000 {
		t.Errorf("expected 10000, got %.2f", room)
	}
}

func TestCurrentFHSARoom_SameYear(t *testing.T) {
	refDate := date(2025, 1, 15)
	now := date(2025, 8, 1)
	room := CurrentFHSARoom(8000, refDate, now)
	if room != 8000 {
		t.Errorf("expected 8000, got %.2f", room)
	}
}

func TestCurrentFHSARoom_OneYearElapsed(t *testing.T) {
	refDate := date(2024, 3, 1)
	now := date(2025, 3, 1)
	room := CurrentFHSARoom(8000, refDate, now)
	expected := 8000 + FHSAAnnualLimit
	if room != expected {
		t.Errorf("expected %.2f, got %.2f", expected, room)
	}
}

func TestCurrentFHSARoom_TwoYearsElapsed(t *testing.T) {
	refDate := date(2023, 4, 1)
	now := date(2025, 4, 1)
	room := CurrentFHSARoom(8000, refDate, now)
	expected := 8000 + 2*FHSAAnnualLimit
	if room != expected {
		t.Errorf("expected %.2f, got %.2f", expected, room)
	}
}

func TestCurrentFHSARoom_CappedAtLifetimeMax(t *testing.T) {
	refDate := date(2023, 4, 1)
	now := date(2030, 4, 1)
	room := CurrentFHSARoom(38000, refDate, now)
	if room != FHSALifetimeMax {
		t.Errorf("expected %.2f (lifetime max), got %.2f", FHSALifetimeMax, room)
	}
}

func TestCurrentFHSARoom_ZeroReferenceDate(t *testing.T) {
	var zeroDate time.Time
	now := date(2025, 6, 1)
	room := CurrentFHSARoom(8000, zeroDate, now)
	if room != 8000 {
		t.Errorf("expected 8000 (no years elapsed), got %.2f", room)
	}
}

func TestCurrentTFSARoom_ExactlyJan1(t *testing.T) {
	// Jan 1 itself should count as a new year addition
	refDate := date(2024, 12, 31)
	now := date(2025, 1, 1)
	room := CurrentTFSARoom(10000, refDate, now)
	expected := 10000 + TFSAAnnualLimit
	if room != expected {
		t.Errorf("expected %.2f on Jan 1 boundary, got %.2f", expected, room)
	}
}
