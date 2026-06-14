package observations

import (
	"math"
	"sort"
	"time"

	"business-workbench/backend-go/internal/model"
	"business-workbench/backend-go/internal/trading"
)

func CalendarForMonth(products []model.Product, month string) []model.CalendarDay {
	dates := map[string][]model.CalendarProduct{}
	for _, product := range products {
		for _, obs := range DatesForMonth(product, month) {
			dates[obs.Date] = append(dates[obs.Date], buildCalendarProduct(product, obs.MonthsSinceEntry))
		}
	}

	result := make([]model.CalendarDay, 0, len(dates))
	for date, rows := range dates {
		sort.Slice(rows, func(i, j int) bool {
			return rows[i].Name < rows[j].Name
		})
		result = append(result, model.CalendarDay{Date: date, Products: rows})
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Date < result[j].Date
	})
	return result
}

type ObservationDate struct {
	Date             string
	MonthsSinceEntry int
}

type Evaluation struct {
	ObservationDate  string
	KnockoutPrice    *float64
	DividendLine     *float64
	UnderlyingPrice  float64
	IsKnockedOut     string
	IsDividend       string
	MonthsSinceEntry int
}

func DatesForMonth(product model.Product, month string) []ObservationDate {
	if len(month) != 7 || product.IssueDate == "" {
		return nil
	}
	monthStart := month + "-01"
	start, err := time.Parse("2006-01-02", monthStart)
	if err != nil {
		return nil
	}
	monthEnd := start.AddDate(0, 1, -1).Format("2006-01-02")

	result := []ObservationDate{}
	for months := 1; months < 600; months++ {
		rawDate := AddMonths(product.IssueDate, months)
		if rawDate > monthEnd {
			break
		}
		adjusted := AdjustForHoliday(rawDate, product.HolidayAdjust)
		if adjusted >= monthStart && adjusted <= monthEnd {
			result = append(result, ObservationDate{Date: adjusted, MonthsSinceEntry: months})
		}
	}
	return result
}

func DatesUntil(product model.Product, today string) []ObservationDate {
	if product.IssueDate == "" {
		return nil
	}
	result := []ObservationDate{}
	for months := 1; months < 600; months++ {
		rawDate := AddMonths(product.IssueDate, months)
		if rawDate > today {
			break
		}
		result = append(result, ObservationDate{
			Date:             AdjustForHoliday(rawDate, product.HolidayAdjust),
			MonthsSinceEntry: months,
		})
	}
	return result
}

func NextObservationDate(product model.Product, today string) string {
	if product.IssueDate == "" {
		return ""
	}
	for months := 1; months < 600; months++ {
		rawDate := AddMonths(product.IssueDate, months)
		adjusted := AdjustForHoliday(rawDate, product.HolidayAdjust)
		if adjusted > today {
			return adjusted
		}
	}
	return ""
}

func buildCalendarProduct(product model.Product, monthsSinceEntry int) model.CalendarProduct {
	knockoutPrice := ComputeKnockoutPrice(product, monthsSinceEntry)
	dividendLine := ComputeDividendLine(product)
	return model.CalendarProduct{
		ID:                     product.ID,
		Name:                   product.Name,
		Manager:                product.Manager,
		Code:                   product.Code,
		MonthsSinceEntry:       monthsSinceEntry,
		EntryPrice:             product.EntryPrice,
		KnockoutPrice:          knockoutPrice,
		DividendLine:           dividendLine,
		IsKnockoutObservable:   knockoutPrice != nil,
		HasDividendObservation: parseRatio(product.MonthlyCoupon) > 0,
	}
}

func ComputeKnockoutPrice(product model.Product, monthsSinceEntry int) *float64 {
	firstKnockoutRatio := parseRatio(product.FirstKnockoutRatio)
	entryPrice := derefFloat(product.EntryPrice)
	lockMonths := derefInt(product.LockMonths)
	monthlyDecrease := parseRatio(product.MonthlyDecrease)

	if firstKnockoutRatio == 0 || entryPrice == 0 {
		return nil
	}
	if monthsSinceEntry < lockMonths {
		return nil
	}
	value := entryPrice * (firstKnockoutRatio - float64(monthsSinceEntry-lockMonths)*monthlyDecrease)
	return &value
}

func ComputeDividendLine(product model.Product) *float64 {
	value := derefFloat(product.EntryPrice) * parseRatio(product.DividendBarrier)
	return &value
}

func EvaluateObservation(product model.Product, obsDate string, underlyingPrice float64, monthsSinceEntry int) Evaluation {
	dividendLine := ComputeDividendLine(product)
	isDividend := "否"
	if dividendLine != nil && underlyingPrice >= *dividendLine {
		isDividend = "是"
	}

	knockoutPrice := ComputeKnockoutPrice(product, monthsSinceEntry)
	isKnockedOut := "不观察"
	if knockoutPrice != nil {
		isKnockedOut = "否"
		if underlyingPrice >= *knockoutPrice {
			isKnockedOut = "是"
		}
	}

	return Evaluation{
		ObservationDate:  obsDate,
		KnockoutPrice:    knockoutPrice,
		DividendLine:     dividendLine,
		UnderlyingPrice:  underlyingPrice,
		IsKnockedOut:     isKnockedOut,
		IsDividend:       isDividend,
		MonthsSinceEntry: monthsSinceEntry,
	}
}

func AddMonths(dateStr string, months int) string {
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return ""
	}
	targetDay := date.Day()
	next := date.AddDate(0, months, 0)
	if next.Day() != targetDay {
		next = time.Date(next.Year(), next.Month(), 0, 0, 0, 0, 0, time.UTC)
	}
	return next.Format("2006-01-02")
}

func AdjustForHoliday(dateStr, holidayAdjust string) string {
	if trading.IsTradingDay(dateStr) {
		return dateStr
	}
	direction := "postpone"
	if holidayAdjust == "提前" {
		direction = "advance"
	}
	return trading.AdjustToNearestTradingDay(dateStr, direction)
}

func parseRatio(value *float64) float64 {
	if value == nil || math.IsNaN(*value) || math.IsInf(*value, 0) {
		return 0
	}
	if math.Abs(*value) > 2 {
		return *value / 100
	}
	return *value
}

func derefFloat(value *float64) float64 {
	if value == nil {
		return 0
	}
	return *value
}

func derefInt(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}
