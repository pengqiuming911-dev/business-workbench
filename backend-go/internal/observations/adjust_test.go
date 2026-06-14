package observations

import (
	"testing"

	"business-workbench/backend-go/internal/model"
)

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
