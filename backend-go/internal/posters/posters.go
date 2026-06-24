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

// BuildArtifact 把计算好的喜报数据转成前端模板直接消费的展示字段 map。
// 所有数字在此一次性格式化成字符串,前端与 agent 都不得再解析或改写这些数字。
func BuildArtifact(product model.Product, data Data, observationDate string) map[string]any {
	return map[string]any{
		"product_id":               product.ID,
		"product_name":             product.Name,
		"observation_date":         observationDate,
		"observation_date_display": FormatChineseDate(observationDate),
		"entry_date":               product.IssueDate,
		"entry_date_display":       FormatChineseDate(product.IssueDate),
		// 数字字段(锁死,前端原样展示)
		"annualized_return":        fmt.Sprintf("%.2f", data.AnnualizedReturn*100),
		"monthly_coupon":           fmt.Sprintf("%.2f", data.MonthlyCoupon*100),
		"cumulative_dividend_rate": fmt.Sprintf("%.2f", data.CumulativeDividendRate*100),
		"dividend_count":           data.DividendCount,
		"underlying_name":          data.UnderlyingName,
		"dividend_barrier_value":   data.DividendBarrierValue,
		"knockout_value":           data.KnockoutValue,
		"parachute_value":          data.ParachuteValue,
		// 文案字段(默认值,agent 不可改数字类;disclaimer 措辞锁死)
		"title":               "分红观察喜报",
		"subtitle":            "IMPORTANT MESSAGE",
		"congrats":            "Congratulations",
		"congrat_text_prefix": "热烈祝贺",
		"label_yield":         "年化收益:",
		"label_cumulative":    "累计分红",
		"label_monthly":       "本月分红:",
		"qr_caption":          "扫码了解更多详情",
		"disclaimer":          "* 本产品仅面向合格投资者,衍生品为高风险资产,投资需谨慎",
	}
}
