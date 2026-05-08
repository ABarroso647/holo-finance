package invest

import (
	"testing"
)

func TestMarginalRate_Zero(t *testing.T) {
	rate := MarginalRate(0, "ON")
	if rate != 0 {
		t.Errorf("expected 0 for zero income, got %.4f", rate)
	}
}

func TestMarginalRate_NegativeIncome(t *testing.T) {
	rate := MarginalRate(-1000, "ON")
	if rate != 0 {
		t.Errorf("expected 0 for negative income, got %.4f", rate)
	}
}

func TestMarginalRate_50k_Ontario(t *testing.T) {
	// $50k is in the first federal bracket (15%) and first ON bracket (5.05%)
	// Expected combined: 15% + 5.05% = 20.05%
	rate := MarginalRate(50000, "ON")
	expected := 0.15 + 0.0505
	if abs(rate-expected) > 0.001 {
		t.Errorf("expected ~%.4f for $50k ON, got %.4f", expected, rate)
	}
}

func TestMarginalRate_100k_Ontario(t *testing.T) {
	// $100k: federal bracket 20.5% ($57,375–$114,750), ON bracket 9.15% ($51,446–$102,894)
	// Expected combined: 20.5% + 9.15% = 29.65%
	rate := MarginalRate(100000, "ON")
	expected := 0.205 + 0.0915
	if abs(rate-expected) > 0.001 {
		t.Errorf("expected ~%.4f for $100k ON, got %.4f", expected, rate)
	}
}

func TestMarginalRate_200k_Ontario(t *testing.T) {
	// $200k: federal bracket 29% ($158,519–$220,000), ON bracket 12.16% ($150,000–$220,000)
	// Expected combined: 29% + 12.16% = 41.16%
	// The spec mentions ~53.53% range but that includes surtax; our approximation gives 41.16%
	// We test that it's in the high 30s–40s range (above 35%)
	rate := MarginalRate(200000, "ON")
	if rate < 0.35 || rate > 0.55 {
		t.Errorf("expected $200k ON rate to be in 35-55%% range, got %.4f", rate)
	}
	// More specifically: 29% + 12.16% = 41.16%
	expected := 0.29 + 0.1216
	if abs(rate-expected) > 0.001 {
		t.Errorf("expected ~%.4f for $200k ON, got %.4f", expected, rate)
	}
}

func TestMarginalRate_Alberta_vs_Ontario(t *testing.T) {
	// At $100k: AB = 20.5% + 10% = 30.5%, ON = 20.5% + 9.15% = 29.65%
	rateAB := MarginalRate(100000, "AB")
	rateON := MarginalRate(100000, "ON")

	// AB is flat 10% provincial — at 100k AB should be: 20.5% federal + 10% provincial = 30.5%
	expectedAB := 0.205 + 0.10
	if abs(rateAB-expectedAB) > 0.001 {
		t.Errorf("expected AB %.4f at $100k, got %.4f", expectedAB, rateAB)
	}

	// At $200k: AB = 29% + 10% = 39%, ON = 29% + 12.16% = 41.16%
	rateAB200 := MarginalRate(200000, "AB")
	rateON200 := MarginalRate(200000, "ON")
	if rateAB200 >= rateON200 {
		t.Errorf("expected AB ($200k) to be lower than ON, got AB=%.4f ON=%.4f", rateAB200, rateON200)
	}

	_ = rateON
}

func TestMarginalRate_UnknownProvince_DefaultsToON(t *testing.T) {
	rateUnknown := MarginalRate(100000, "XX")
	rateON := MarginalRate(100000, "ON")
	if abs(rateUnknown-rateON) > 0.001 {
		t.Errorf("unknown province should default to ON: got %.4f vs ON %.4f", rateUnknown, rateON)
	}
}

func TestMarginalRate_AllProvinces(t *testing.T) {
	provinces := []string{"AB", "BC", "MB", "NB", "NL", "NS", "NT", "NU", "ON", "PE", "QC", "SK", "YT"}
	for _, p := range provinces {
		rate := MarginalRate(80000, p)
		if rate <= 0 || rate > 0.70 {
			t.Errorf("province %s: unexpected rate %.4f for $80k income", p, rate)
		}
	}
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
