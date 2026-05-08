package invest

import (
	"testing"
)

func TestRecommend_MissingIncome(t *testing.T) {
	rec := Recommend(0, 500, 10000, 6000, 2000, "ON")
	if len(rec.MissingInputs) == 0 {
		t.Error("expected MissingInputs to be populated when income is 0")
	}
	if rec.MissingInputs[0] != "annual income" {
		t.Errorf("expected 'annual income' in MissingInputs, got %v", rec.MissingInputs)
	}
	if rec.RRSPAmount != 0 || rec.TFSAAmount != 0 {
		t.Error("amounts should be 0 when income is missing")
	}
}

func TestRecommend_HighIncome_RRSP_First(t *testing.T) {
	// $150k Ontario → marginal rate ~26% + 11.16% = 37.16% > 30%, RRSP room available
	// Monthly surplus $3000, buffer $500 → investable $2500
	rec := Recommend(150000, 500, 50000, 12000, 3000, "ON")

	if len(rec.MissingInputs) > 0 {
		t.Fatalf("unexpected missing inputs: %v", rec.MissingInputs)
	}
	if rec.MarginalRate < 0.30 {
		t.Errorf("expected marginal rate >= 30%% for $150k ON, got %.4f", rec.MarginalRate)
	}
	if rec.Investable != 2500 {
		t.Errorf("expected investable 2500, got %.2f", rec.Investable)
	}
	if rec.RRSPAmount <= 0 {
		t.Errorf("expected RRSP allocation for high-income ON, got %.2f", rec.RRSPAmount)
	}
	if rec.RRSPTaxSaving <= 0 {
		t.Errorf("expected positive RRSPTaxSaving, got %.2f", rec.RRSPTaxSaving)
	}
	// RRSP tax saving = rrspAmt * 12 * rate
	expectedSaving := rec.RRSPAmount * 12 * rec.MarginalRate
	if absf(rec.RRSPTaxSaving-expectedSaving) > 0.01 {
		t.Errorf("RRSPTaxSaving mismatch: expected %.2f, got %.2f", expectedSaving, rec.RRSPTaxSaving)
	}
}

func TestRecommend_LowIncome_TFSA_First(t *testing.T) {
	// $40k Ontario → marginal rate 15% + 5.05% = 20.05% < 30% → TFSA first
	rec := Recommend(40000, 500, 10000, 20000, 2000, "ON")

	if len(rec.MissingInputs) > 0 {
		t.Fatalf("unexpected missing inputs: %v", rec.MissingInputs)
	}
	if rec.MarginalRate >= 0.30 {
		t.Errorf("expected marginal rate < 30%% for $40k ON, got %.4f", rec.MarginalRate)
	}
	if rec.TFSAAmount <= 0 {
		t.Errorf("expected TFSA allocation for low-income ON, got %.2f", rec.TFSAAmount)
	}
	if rec.RRSPAmount > rec.TFSAAmount {
		t.Errorf("RRSP should not exceed TFSA for low income: RRSP=%.2f TFSA=%.2f", rec.RRSPAmount, rec.TFSAAmount)
	}
}

func TestRecommend_HighIncome_NoRRSPRoom_UsesTFSA(t *testing.T) {
	// $200k Ontario, marginal rate > 30% but no RRSP room → route to TFSA
	rec := Recommend(200000, 500, 0, 20000, 3000, "ON")

	if len(rec.MissingInputs) > 0 {
		t.Fatalf("unexpected missing inputs: %v", rec.MissingInputs)
	}
	if rec.MarginalRate < 0.30 {
		t.Errorf("expected marginal rate >= 30%% for $200k ON, got %.4f", rec.MarginalRate)
	}
	if rec.RRSPAmount != 0 {
		t.Errorf("expected no RRSP allocation when rrspRoom=0, got %.2f", rec.RRSPAmount)
	}
	if rec.TFSAAmount <= 0 {
		t.Errorf("expected TFSA allocation when no RRSP room, got %.2f", rec.TFSAAmount)
	}
}

func TestRecommend_ZeroSurplus(t *testing.T) {
	// Surplus below buffer → investable = 0
	rec := Recommend(120000, 500, 20000, 10000, 400, "ON")

	if len(rec.MissingInputs) > 0 {
		t.Fatalf("unexpected missing inputs: %v", rec.MissingInputs)
	}
	if rec.Investable != 0 {
		t.Errorf("expected investable=0 when surplus < buffer, got %.2f", rec.Investable)
	}
	if rec.RRSPAmount != 0 {
		t.Errorf("expected RRSP=0 when investable=0, got %.2f", rec.RRSPAmount)
	}
	if rec.TFSAAmount != 0 {
		t.Errorf("expected TFSA=0 when investable=0, got %.2f", rec.TFSAAmount)
	}
	if rec.RRSPTaxSaving != 0 {
		t.Errorf("expected tax saving=0, got %.2f", rec.RRSPTaxSaving)
	}
}

func TestRecommend_ExactBuffer(t *testing.T) {
	// Surplus exactly equals buffer → investable = 0
	rec := Recommend(80000, 1000, 20000, 10000, 1000, "ON")
	if rec.Investable != 0 {
		t.Errorf("expected investable=0 when surplus==buffer, got %.2f", rec.Investable)
	}
}

func TestRecommend_RRSPCapByRoom(t *testing.T) {
	// Very small RRSP room relative to surplus
	// $120k ON, investable $2000/mo, RRSP room only $1200 (i.e. $100/mo cap)
	rec := Recommend(120000, 500, 1200, 50000, 2500, "ON")

	// marginal rate for $120k ON: federal 20.5% ($57,375–$114,750... wait, $120k is 26% bracket)
	// $114,750–$158,519: 26% federal + 11.16% ON = 37.16% → RRSP preferred
	if rec.RRSPAmount > 1200/12+0.01 {
		t.Errorf("RRSP monthly should be capped at %.2f, got %.2f", 1200.0/12, rec.RRSPAmount)
	}
}

func absf(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
