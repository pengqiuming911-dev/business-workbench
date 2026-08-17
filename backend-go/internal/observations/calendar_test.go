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

// TestCalendarSkipsDateWithNothingObservable 复现金澹观海2号场景：
// 欧式早利雪球、月票息为 0、处于锁定期内（monthsSinceEntry<lockMonths）→
// 该月既不观察派息也不观察敲出，不应出现在观察日历里。
func TestCalendarSkipsDateWithNothingObservable(t *testing.T) {
	product := model.Product{
		ID:                 "P1",
		Name:               "金澹观海2号-测试",
		IssueDate:          "2026-07-14",
		LockMonths:         intPtr(3),
		FirstKnockoutRatio: floatPtr(1.0),
		EntryPrice:         floatPtr(7631.76),
		MonthlyDecrease:    floatPtr(0.0085),
		MonthlyCoupon:      floatPtr(0),
	}
	// 发行 2026-07-14，+1 月 = 2026-08-14；monthsSinceEntry=1 < lockMonths=3 → 敲出 nil；
	// 月票息 0 → 派息不可观察；该月无可观察项。
	days := CalendarForMonth([]model.Product{product}, "2026-08", CalendarOpts{Status: "ongoing"})
	for _, day := range days {
		for _, cp := range day.Products {
			if cp.ID == "P1" {
				t.Fatalf("无可观察项的月份不应进日历，但 %s 出现在 %s", cp.Name, day.Date)
			}
		}
	}
}

// TestCalendarIncludesDateWhenKnockoutObservable 过锁定期后有敲出价 → 应进日历。
func TestCalendarIncludesDateWhenKnockoutObservable(t *testing.T) {
	product := model.Product{
		ID:                 "P2",
		Name:               "过锁定期-测试",
		IssueDate:          "2026-04-14",
		LockMonths:         intPtr(3),
		FirstKnockoutRatio: floatPtr(1.0),
		EntryPrice:         floatPtr(7631.76),
		MonthlyDecrease:    floatPtr(0.0085),
		MonthlyCoupon:      floatPtr(0),
	}
	// 发行 2026-04-14，+4 月 = 2026-08-14；monthsSinceEntry=4 >= lockMonths=3 → 敲出可观察。
	days := CalendarForMonth([]model.Product{product}, "2026-08", CalendarOpts{Status: "ongoing"})
	found := false
	for _, day := range days {
		for _, cp := range day.Products {
			if cp.ID == "P2" {
				found = true
			}
		}
	}
	if !found {
		t.Fatal("过锁定期且敲出可观察的产品应进日历")
	}
}

// TestCalendarIncludesDateWhenDividendObservable 有月票息 → 即使锁定期内也应进日历。
func TestCalendarIncludesDateWhenDividendObservable(t *testing.T) {
	product := model.Product{
		ID:                 "P3",
		Name:               "有月票息-测试",
		IssueDate:          "2026-07-14",
		LockMonths:         intPtr(3),
		FirstKnockoutRatio: floatPtr(1.0),
		EntryPrice:         floatPtr(7631.76),
		MonthlyDecrease:    floatPtr(0.0085),
		MonthlyCoupon:      floatPtr(0.05),
	}
	days := CalendarForMonth([]model.Product{product}, "2026-08", CalendarOpts{Status: "ongoing"})
	found := false
	for _, day := range days {
		for _, cp := range day.Products {
			if cp.ID == "P3" {
				found = true
			}
		}
	}
	if !found {
		t.Fatal("有月票息的产品应进日历（派息可观察）")
	}
}

// TestDatesUntilSkipsNothingObservable DatesUntil 同样不应返回无可观察项的日期。
func TestDatesUntilSkipsNothingObservable(t *testing.T) {
	product := model.Product{
		ID:                 "P4",
		Name:               "金澹观海2号-DatesUntil",
		IssueDate:          "2026-07-14",
		LockMonths:         intPtr(3),
		FirstKnockoutRatio: floatPtr(1.0),
		EntryPrice:         floatPtr(7631.76),
		MonthlyDecrease:    floatPtr(0.0085),
		MonthlyCoupon:      floatPtr(0),
	}
	// 截至 2026-08-31，仅有 month=1（2026-08-14）落在范围内，但它无可观察项。
	dates := DatesUntil(product, "2026-08-31")
	if len(dates) != 0 {
		t.Fatalf("锁定期内且无票息时不应有可观察日，got %d: %v", len(dates), dates)
	}
}

// TestNextObservationDateSkipsNothingObservable 下次观察日应跳过无可观察项的月份。
func TestNextObservationDateSkipsNothingObservable(t *testing.T) {
	product := model.Product{
		ID:                 "P5",
		Name:               "金澹观海2号-Next",
		IssueDate:          "2026-07-14",
		LockMonths:         intPtr(3),
		FirstKnockoutRatio: floatPtr(1.0),
		EntryPrice:         floatPtr(7631.76),
		MonthlyDecrease:    floatPtr(0.0085),
		MonthlyCoupon:      floatPtr(0),
	}
	// 今天 2026-08-20：month1(2026-08-14) 已过且无可观察项；下次可观察在 month3=2026-10-14（过锁定期）。
	got := NextObservationDate(product, "2026-08-20")
	if got != "" && got < "2026-10-01" {
		t.Fatalf("下次观察日应跳过无可观察的月份，got %s", got)
	}
	want := AdjustForHoliday(AddMonths(product.IssueDate, 3), product.HolidayAdjust)
	if got != want {
		t.Fatalf("下次观察日 = %s, want %s", got, want)
	}
}

func almostEqual(a, b, eps float64) bool {
	if a-b > eps || b-a > eps {
		return false
	}
	return true
}
