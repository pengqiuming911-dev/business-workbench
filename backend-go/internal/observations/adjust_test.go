package observations

import (
	"math"
	"testing"

	"business-workbench/backend-go/internal/model"
)

func floatPtr(value float64) *float64 {
	return &value
}

func intPtr(value int) *int {
	return &value
}

func TestAdjustForHoliday_Postpone(t *testing.T) {
	result := AdjustForHoliday("2026-02-16", "")
	if result == "2026-02-16" {
		t.Errorf("holiday date should not be returned unchanged, got %s", result)
	}
}

func TestAdjustForHoliday_Advance(t *testing.T) {
	result := AdjustForHoliday("2026-02-16", "提前")
	if result == "2026-02-16" {
		t.Errorf("holiday date should be adjusted backward, got %s", result)
	}
	if result > "2026-02-16" {
		t.Errorf("advance should produce earlier date, got %s", result)
	}
}

func TestAdjustForHoliday_NormalDay(t *testing.T) {
	result := AdjustForHoliday("2026-06-15", "")
	if result != "2026-06-15" {
		t.Errorf("trading day should be unchanged, got %s", result)
	}
}

func TestDatesUntil_UsesHolidayAdjust(t *testing.T) {
	product := model.Product{
		IssueDate:     "2026-01-01",
		HolidayAdjust: "提前",
	}
	dates := DatesUntil(product, "2026-03-01")
	if len(dates) == 0 {
		t.Fatal("DatesUntil returned no dates")
	}
	for _, d := range dates {
		if d.Date == "" {
			t.Error("observation date should not be empty")
		}
	}
}

func TestComputeKnockoutPrice_BeforeLockPeriodIsNotObserved(t *testing.T) {
	product := model.Product{
		EntryPrice:         floatPtr(8000),
		FirstKnockoutRatio: floatPtr(1.03),
		LockMonths:         intPtr(3),
		MonthlyDecrease:    floatPtr(0.005),
	}

	if price := ComputeKnockoutPrice(product, 2); price != nil {
		t.Fatalf("got knockout price %v before lock period, want nil", *price)
	}
}

func TestComputeKnockoutPrice_FirstObservationUsesInitialRatio(t *testing.T) {
	product := model.Product{
		EntryPrice:         floatPtr(8000),
		FirstKnockoutRatio: floatPtr(1.03),
		LockMonths:         intPtr(3),
		MonthlyDecrease:    floatPtr(0.005),
	}

	price := ComputeKnockoutPrice(product, 3)
	if price == nil || math.Abs(*price-8240) > 0.000001 {
		t.Fatalf("got knockout price %v, want 8240", price)
	}
}

func TestComputeKnockoutPrice_DecreasesForEachLaterObservation(t *testing.T) {
	product := model.Product{
		EntryPrice:         floatPtr(8000),
		FirstKnockoutRatio: floatPtr(1.03),
		LockMonths:         intPtr(3),
		MonthlyDecrease:    floatPtr(0.005),
	}

	price := ComputeKnockoutPrice(product, 5)
	want := 8000 * (1.03 - float64(5-3)*0.005)
	if price == nil || math.Abs(*price-want) > 0.000001 {
		t.Fatalf("got knockout price %v, want %.2f", price, want)
	}
}
