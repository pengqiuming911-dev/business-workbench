package db

import "testing"

func float64Ptr(v float64) *float64 { return &v }

func TestNormalizeFirstKnockoutRatioPtr_FromAbsoluteETFPrice(t *testing.T) {
	got := normalizeFirstKnockoutRatioPtr(float64Ptr(0.77353), float64Ptr(0.751))
	if got == nil {
		t.Fatalf("got nil")
	}
	want := 1.03
	if *got < want-0.00001 || *got > want+0.00001 {
		t.Fatalf("got %.6f want %.6f", *got, want)
	}
}

func TestNormalizeFirstKnockoutRatioPtr_LeavesRatioUntouched(t *testing.T) {
	got := normalizeFirstKnockoutRatioPtr(float64Ptr(1.03), float64Ptr(0.751))
	if got == nil || *got != 1.03 {
		t.Fatalf("got %v want 1.03", got)
	}
}
