package app

import (
	"testing"
)

func ptrF(v float64) *float64 { return &v }

func TestExcludeWhenAllRatiosZero(t *testing.T) {
	d := computeRebateDecision(rebateDecisionInput{
		SubReturnable: true, MgmtReturnable: true,
		OutstandingSub: 100, OutstandingMgmt: 50, OutstandingPerf: 20,
	})
	if !d.Exclude {
		t.Fatalf("expected exclude when all ratios zero")
	}
}

func TestExcludeReturnableAndFullyReturned(t *testing.T) {
	d := computeRebateDecision(rebateDecisionInput{
		SubRatio: 0.5, MgmtRatio: 0.5, PerfRatio: 0.5,
		SubReturnable: true, MgmtReturnable: true,
		OutstandingSub: 0, OutstandingMgmt: -1, OutstandingPerf: 0,
	})
	if !d.Exclude {
		t.Fatalf("expected exclude when returnable and fully returned")
	}
}

func TestKeepNotReturnableEvenIfNoOutstanding(t *testing.T) {
	// 暂不可返（不在待返 sheet）即便未返为 0 仍保留展示
	d := computeRebateDecision(rebateDecisionInput{
		SubRatio: 0.5, MgmtRatio: 0, PerfRatio: 0,
		SubReturnable: false, MgmtReturnable: false,
		OutstandingSub: 0, OutstandingMgmt: 0, OutstandingPerf: 0,
	})
	if d.Exclude {
		t.Fatalf("expected keep for 暂不可返 order")
	}
	if d.IsReturnable != "暂不可返" {
		t.Fatalf("is_returnable = %q, want 暂不可返", d.IsReturnable)
	}
}

func TestReturnableWithOutstandingMarked待返(t *testing.T) {
	d := computeRebateDecision(rebateDecisionInput{
		SubRatio: 0.5, MgmtRatio: 0.5, PerfRatio: 0.5,
		SubReturnable: true, MgmtReturnable: true,
		OutstandingSub: 100, OutstandingMgmt: 0, OutstandingPerf: 0,
	})
	if d.Exclude {
		t.Fatalf("expected keep")
	}
	if d.IsReturnable != "待返" {
		t.Fatalf("is_returnable = %q, want 待返", d.IsReturnable)
	}
}

func TestCheckTFMatches(t *testing.T) {
	d := computeRebateDecision(rebateDecisionInput{
		SubRatio: 0.5, MgmtRatio: 0.5, PerfRatio: 0.5,
		SubReturnable: true, MgmtReturnable: true,
		OutstandingSub: 100, OutstandingMgmt: 50, OutstandingPerf: 20,
		DetailSub: ptrF(100), DetailMgmt: ptrF(49), DetailPerf: nil,
	})
	if d.CheckSub != "T" {
		t.Fatalf("check_sub = %q, want T", d.CheckSub)
	}
	if d.CheckMgmt != "F" {
		t.Fatalf("check_mgmt = %q, want F (50 vs 49.99)", d.CheckMgmt)
	}
	// DetailPerf is nil and no RebateTarget → "--"
	if d.CheckPerf != "--" {
		t.Fatalf("check_perf = %q, want -- when detail nil and no rebate target", d.CheckPerf)
	}
}

func TestCheckDashWhenDetailNilWithRebateTarget(t *testing.T) {
	d := computeRebateDecision(rebateDecisionInput{
		SubRatio: 0.5, MgmtRatio: 0.5, PerfRatio: 0.5,
		SubReturnable: true, MgmtReturnable: true,
		OutstandingSub: 100, OutstandingMgmt: 50, OutstandingPerf: 20,
		DetailSub: ptrF(100), DetailMgmt: ptrF(49), DetailPerf: nil,
		RebateTarget: "某某",
	})
	if d.CheckSub != "T" {
		t.Fatalf("check_sub = %q, want T", d.CheckSub)
	}
	if d.CheckMgmt != "F" {
		t.Fatalf("check_mgmt = %q, want F", d.CheckMgmt)
	}
	// DetailPerf is nil but RebateTarget is set → "-"
	if d.CheckPerf != "-" {
		t.Fatalf("check_perf = %q, want - when detail nil and rebate target present", d.CheckPerf)
	}
}

func TestCheckSubMismatchReturnsF(t *testing.T) {
	d := computeRebateDecision(rebateDecisionInput{
		OrderID:         "snowball20260506001008",
		SubRatio:        0.5,
		MgmtRatio:       0.5,
		PerfRatio:       0.5,
		SubReturnable:   true,
		MgmtReturnable:  true,
		OutstandingSub:  0,
		OutstandingMgmt: 50,
		OutstandingPerf: 20,
		DetailSub:       ptrF(8457.5),
		DetailMgmt:      ptrF(50),
		DetailPerf:      ptrF(20),
	})
	if d.CheckSub != "F" {
		t.Fatalf("check_sub = %q, want F for mismatch", d.CheckSub)
	}
}
