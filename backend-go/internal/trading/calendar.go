package trading

import "time"

var holidays2025 = []string{
	"2025-01-01",
	"2025-01-28", "2025-01-29", "2025-01-30", "2025-01-31",
	"2025-02-01", "2025-02-02", "2025-02-03", "2025-02-04",
	"2025-04-04", "2025-04-05", "2025-04-06",
	"2025-05-01", "2025-05-02", "2025-05-03", "2025-05-04", "2025-05-05",
	"2025-05-31", "2025-06-01", "2025-06-02",
	"2025-10-01", "2025-10-02", "2025-10-03", "2025-10-04",
	"2025-10-05", "2025-10-06", "2025-10-07",
}

var holidays2026 = []string{
	"2026-01-01", "2026-01-02", "2026-01-03",
	"2026-02-16", "2026-02-17", "2026-02-18", "2026-02-19",
	"2026-02-20", "2026-02-21", "2026-02-22",
	"2026-04-04", "2026-04-05", "2026-04-06", "2026-04-07",
	"2026-05-01", "2026-05-02", "2026-05-03",
	"2026-06-19", "2026-06-20", "2026-06-21",
	"2026-09-25", "2026-09-26", "2026-09-27",
	"2026-10-01", "2026-10-02", "2026-10-03", "2026-10-04",
	"2026-10-05", "2026-10-06", "2026-10-07",
}

var holidaySet map[string]bool

func init() {
	holidaySet = make(map[string]bool, len(holidays2025)+len(holidays2026))
	for _, d := range holidays2025 {
		holidaySet[d] = true
	}
	for _, d := range holidays2026 {
		holidaySet[d] = true
	}
}

func IsWeekend(dateStr string) bool {
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return false
	}
	return t.Weekday() == time.Saturday || t.Weekday() == time.Sunday
}

func IsHoliday(dateStr string) bool {
	return holidaySet[dateStr]
}

func IsTradingDay(dateStr string) bool {
	return !IsWeekend(dateStr) && !IsHoliday(dateStr)
}

func AdjustToNearestTradingDay(dateStr, direction string) string {
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return dateStr
	}
	step := 1
	if direction == "advance" {
		step = -1
	}
	for !IsTradingDay(t.Format("2006-01-02")) {
		t = t.AddDate(0, 0, step)
	}
	return t.Format("2006-01-02")
}
