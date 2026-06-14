package trading_test

import (
	"testing"

	"business-workbench/backend-go/internal/trading"
)

func TestIsHoliday(t *testing.T) {
	if !trading.IsHoliday("2026-02-16") {
		t.Error("2026-02-16 should be a holiday (Spring Festival)")
	}
	if trading.IsHoliday("2026-02-25") {
		t.Error("2026-02-25 should not be a holiday")
	}
}

func TestIsWeekend(t *testing.T) {
	if !trading.IsWeekend("2026-06-14") {
		t.Error("2026-06-14 (Sunday) should be weekend")
	}
	if trading.IsWeekend("2026-06-15") {
		t.Error("2026-06-15 (Monday) should not be weekend")
	}
}

func TestIsTradingDay(t *testing.T) {
	if !trading.IsTradingDay("2026-06-15") {
		t.Error("2026-06-15 (Mon, no holiday) should be a trading day")
	}
	if trading.IsTradingDay("2026-06-14") {
		t.Error("2026-06-14 (Sun) should not be a trading day")
	}
	if trading.IsTradingDay("2026-02-16") {
		t.Error("2026-02-16 (holiday) should not be a trading day")
	}
}

func TestAdjustToNearestTradingDay_Postpone(t *testing.T) {
	result := trading.AdjustToNearestTradingDay("2026-02-16", "postpone")
	if result == "2026-02-16" {
		t.Errorf("postpone from holiday 2026-02-16 should not return same date, got %s", result)
	}
	if !trading.IsTradingDay(result) {
		t.Errorf("postpone result %s should be a trading day", result)
	}
}

func TestAdjustToNearestTradingDay_Advance(t *testing.T) {
	result := trading.AdjustToNearestTradingDay("2026-02-16", "advance")
	if result == "2026-02-16" {
		t.Errorf("advance from holiday 2026-02-16 should not return same date, got %s", result)
	}
	if !trading.IsTradingDay(result) {
		t.Errorf("advance result %s should be a trading day", result)
	}
}

func TestAdjustToNearestTradingDay_AlreadyTrading(t *testing.T) {
	result := trading.AdjustToNearestTradingDay("2026-06-15", "postpone")
	if result != "2026-06-15" {
		t.Errorf("already a trading day should return unchanged, got %s", result)
	}
}

func TestAdjustToNearestTradingDay_Weekend(t *testing.T) {
	adv := trading.AdjustToNearestTradingDay("2026-06-14", "advance")
	pos := trading.AdjustToNearestTradingDay("2026-06-14", "postpone")
	if adv == "2026-06-14" || !trading.IsTradingDay(adv) {
		t.Errorf("advance from Sunday should be a prior trading day, got %s", adv)
	}
	if pos != "2026-06-15" {
		t.Errorf("postpone from Sunday 2026-06-14 should be Monday 2026-06-15, got %s", pos)
	}
}
