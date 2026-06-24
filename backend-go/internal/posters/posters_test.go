package posters

import (
	"testing"

	"business-workbench/backend-go/internal/model"
)

func TestBuildArtifact_FormatsNumbersAndDates(t *testing.T) {
	monthly := 0.0133
	coupon1st := 0.0
	product := model.Product{
		ID:        "P001",
		Name:      "鹿*8号(三期)",
		IssueDate: "2026-01-30",
		Code:      "恒科ETF指数(三期)",
		Parachute: "75%",
	}
	product.MonthlyCoupon = &monthly
	product.Coupon1st = &coupon1st

	data := Data{
		HasDividendObservation: true,
		UnderlyingName:         "恒科ETF指数",
		ParachuteValue:         "75%",
		KnockoutValue:          "102.00%",
		DividendBarrierValue:   "80%",
		MonthlyCoupon:          0.0133,
		AnnualizedReturn:       0.1596,
		DividendCount:          3,
		CumulativeDividendRate: 0.0399,
	}

	a := BuildArtifact(product, data, "2026-04-30")

	cases := map[string]string{
		"product_name":             "鹿*8号(三期)",
		"observation_date":         "2026-04-30",
		"observation_date_display": "2026年4月30日",
		"entry_date_display":       "2026年1月30日",
		"annualized_return":        "15.96",
		"monthly_coupon":           "1.33",
		"cumulative_dividend_rate": "3.99",
		"underlying_name":          "恒科ETF指数",
		"dividend_barrier_value":   "80%",
		"knockout_value":           "102.00%",
		"parachute_value":          "75%",
		"title":                    "分红观察喜报",
		"disclaimer":               "* 本产品仅面向合格投资者,衍生品为高风险资产,投资需谨慎",
	}
	for key, want := range cases {
		if got, _ := a[key].(string); got != want {
			t.Errorf("field %s = %q, want %q", key, got, want)
		}
	}
	if got, _ := a["dividend_count"].(int); got != 3 {
		t.Errorf("dividend_count = %d, want 3", got)
	}
	if got, _ := a["product_id"].(string); got != "P001" {
		t.Errorf("product_id = %q, want P001", got)
	}
}
