package observations

import (
	"testing"

	"business-workbench/backend-go/internal/model"
)

// TestComputeKnockoutPriceXunlu48 复核驯鹿48号敲出价公式：
// 入场价 0.751 × (首月敲出 103% − (存续月5 − 锁定期3) × 每月递减0.5%) = 0.751 × 1.02 = 0.76602
func TestComputeKnockoutPriceXunlu48(t *testing.T) {
	product := model.Product{
		EntryPrice:         floatPtr(0.751),
		FirstKnockoutRatio: floatPtr(1.03),
		LockMonths:         intPtr(3),
		MonthlyDecrease:    floatPtr(0.005),
	}
	kp := ComputeKnockoutPrice(product, 5)
	if kp == nil {
		t.Fatalf("expected knockout price, got nil")
	}
	if !almostEqual(*kp, 0.76602, 1e-6) {
		t.Fatalf("knockout price = %v, want 0.76602", *kp)
	}
}

// TestComputeKnockoutPriceLockedBeforeLockPeriod 锁定期内不观察敲出。
func TestComputeKnockoutPriceNilBeforeLockPeriod(t *testing.T) {
	product := model.Product{
		EntryPrice:         floatPtr(0.751),
		FirstKnockoutRatio: floatPtr(1.03),
		LockMonths:         intPtr(3),
		MonthlyDecrease:    floatPtr(0.005),
	}
	if kp := ComputeKnockoutPrice(product, 2); kp != nil {
		t.Fatalf("expected nil before lock period, got %v", *kp)
	}
}

func TestSpotPriceForOngoingUsesTodayPrice(t *testing.T) {
	product := model.Product{ID: "P1", Code: "513180.SH"}
	obs := ObservationDate{Date: "2026-06-22", MonthsSinceEntry: 5}
	opts := CalendarOpts{Status: "ongoing", TodayPrice: map[string]float64{"513180.SH": 0.590}}
	sp := spotPriceFor(product, obs, opts)
	if sp == nil || !almostEqual(*sp, 0.590, 1e-9) {
		t.Fatalf("ongoing spot = %v, want 0.590", sp)
	}
}

func TestSpotPriceForCompletedUsesCloseByDate(t *testing.T) {
	product := model.Product{ID: "P1", Code: "513180.SH"}
	obs := ObservationDate{Date: "2026-06-22", MonthsSinceEntry: 5}
	opts := CalendarOpts{
		Status:      "completed",
		CloseByDate: map[string]map[string]float64{"P1": {"2026-06-22": 0.580}},
	}
	sp := spotPriceFor(product, obs, opts)
	if sp == nil || !almostEqual(*sp, 0.580, 1e-9) {
		t.Fatalf("completed spot = %v, want 0.580", sp)
	}
}

func TestSpotPriceForCompletedMissingCloseIsNil(t *testing.T) {
	product := model.Product{ID: "P2", Code: "513180.SH"}
	obs := ObservationDate{Date: "2026-06-22", MonthsSinceEntry: 5}
	opts := CalendarOpts{Status: "completed", CloseByDate: map[string]map[string]float64{}}
	if sp := spotPriceFor(product, obs, opts); sp != nil {
		t.Fatalf("expected nil spot when no close record, got %v", *sp)
	}
}

func almostEqual(a, b, eps float64) bool {
	if a-b > eps || b-a > eps {
		return false
	}
	return true
}
