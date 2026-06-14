package posters

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"business-workbench/backend-go/internal/model"
)

type Data struct {
	HasDividendObservation bool
	UnderlyingName         string
	ParachuteValue         string
	KnockoutValue          string
	DividendBarrierValue   string
	MonthlyCoupon          float64
	AbsoluteReturn         float64
	AnnualizedReturn       float64
	DividendCount          int
	CumulativeDividendRate float64
}

func GenerateData(product model.Product, observationDate string, monthsSinceEntry int) Data {
	monthlyCoupon := parseRatio(product.MonthlyCoupon)
	dividendCount := computeDividendCount(product.IssueDate, observationDate)
	return Data{
		HasDividendObservation: monthlyCoupon > 0,
		UnderlyingName:         getUnderlyingName(product.Code),
		ParachuteValue:         getParachuteValue(product.Parachute),
		KnockoutValue:          knockoutPercent(product, monthsSinceEntry),
		DividendBarrierValue:   dividendBarrierValue(product.DividendBarrier),
		MonthlyCoupon:          monthlyCoupon,
		AbsoluteReturn:         computeAbsoluteReturn(product, monthsSinceEntry),
		AnnualizedReturn:       computeAnnualizedReturn(product),
		DividendCount:          dividendCount,
		CumulativeDividendRate: computeCumulativeDividendRate(product, dividendCount),
	}
}

func FormatChineseDate(dateStr string) string {
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return dateStr
	}
	return fmt.Sprintf("%d年%d月%d日", date.Year(), date.Month(), date.Day())
}

func getUnderlyingName(code string) string {
	code = strings.TrimSpace(code)
	if code == "" {
		return ""
	}
	if idx := strings.IndexAny(code, "(（"); idx >= 0 {
		return strings.TrimSpace(code[:idx])
	}
	return code
}

func getParachuteValue(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	re := regexp.MustCompile(`(\d+\.?\d*)`)
	match := re.FindStringSubmatch(raw)
	if len(match) < 2 {
		return ""
	}
	return match[1] + "%"
}

func knockoutPercent(product model.Product, monthsSinceEntry int) string {
	firstKORatio := parseRatio(product.FirstKnockoutRatio)
	lockMonths := derefInt(product.LockMonths)
	monthlyDecrease := parseRatio(product.MonthlyDecrease)
	if firstKORatio == 0 {
		return ""
	}
	ratio := firstKORatio - float64(max(0, monthsSinceEntry-lockMonths))*monthlyDecrease
	return fmt.Sprintf("%.2f%%", ratio*100)
}

func dividendBarrierValue(value *float64) string {
	ratio := parseRatio(value)
	if ratio == 0 {
		return ""
	}
	return fmt.Sprintf("%.0f%%", ratio*100)
}

func computeAbsoluteReturn(product model.Product, monthsSinceEntry int) float64 {
	monthlyCoupon := parseRatio(product.MonthlyCoupon)
	durationMonths := productDurationMonths(product)
	if monthlyCoupon > 0 {
		return monthlyCoupon * float64(monthsSinceEntry)
	}
	if durationMonths > 0 && durationMonths <= 12 {
		return parseRatio(product.Coupon1st) / 12 * float64(monthsSinceEntry)
	}
	if durationMonths > 12 {
		return parseRatio(product.Coupon2nd) / 12 * float64(monthsSinceEntry)
	}
	return 0
}

func computeAnnualizedReturn(product model.Product) float64 {
	monthlyCoupon := parseRatio(product.MonthlyCoupon)
	durationMonths := productDurationMonths(product)
	if monthlyCoupon > 0 {
		return monthlyCoupon * 12
	}
	if durationMonths > 0 && durationMonths <= 12 {
		return parseRatio(product.Coupon1st)
	}
	if durationMonths > 12 {
		return parseRatio(product.Coupon2nd)
	}
	return 0
}

func computeCumulativeDividendRate(product model.Product, count int) float64 {
	monthlyCoupon := parseRatio(product.MonthlyCoupon)
	durationMonths := productDurationMonths(product)
	if monthlyCoupon > 0 {
		return monthlyCoupon * float64(count)
	}
	if durationMonths > 0 && durationMonths <= 12 {
		return parseRatio(product.Coupon1st) / 12 * float64(count)
	}
	if durationMonths > 12 {
		return parseRatio(product.Coupon2nd) / 12 * float64(count)
	}
	return 0
}

func computeDividendCount(entryDate string, targetDate string) int {
	entry, err := time.Parse("2006-01-02", entryDate)
	if err != nil {
		return 0
	}
	target, err := time.Parse("2006-01-02", targetDate)
	if err != nil {
		return 0
	}
	return (target.Year()-entry.Year())*12 + int(target.Month()-entry.Month())
}

func productDurationMonths(product model.Product) int {
	if product.DurationMonths != nil {
		return int(*product.DurationMonths)
	}
	re := regexp.MustCompile(`(\d+)`)
	match := re.FindStringSubmatch(product.Term)
	if len(match) < 2 {
		return 0
	}
	value, _ := strconv.Atoi(match[1])
	if strings.Contains(product.Term, "年") {
		return value * 12
	}
	return value
}

func parseRatio(value *float64) float64 {
	if value == nil {
		return 0
	}
	if *value > 2 || *value < -2 {
		return *value / 100
	}
	return *value
}

func derefInt(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}
