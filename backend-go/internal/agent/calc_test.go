package agent

import "testing"

func TestCalcPoints_PercentInputs(t *testing.T) {
	out := CalcPoints("60%", "101%", "78%", 8800)
	if got := out["parachute_point"]; got != 5280.0 {
		t.Errorf("parachute_point = %v, want 5280", got)
	}
	if got := out["parachute_point_approx"]; got != "5200点左右" {
		t.Errorf("parachute_point_approx = %v, want 5200点左右", got)
	}
	if got := out["knockout_point"]; got != 8888.0 {
		t.Errorf("knockout_point = %v, want 8888", got)
	}
	if got := out["knockout_point_approx"]; got != "8800点左右" {
		t.Errorf("knockout_point_approx = %v, want 8800点左右", got)
	}
	if got := out["dividend_point"]; got != 6864.0 {
		t.Errorf("dividend_point = %v, want 6864", got)
	}
	if got := out["dividend_point_approx"]; got != "6800点左右" {
		t.Errorf("dividend_point_approx = %v, want 6800点左右", got)
	}
}

func TestCalcPoints_RatioInputs(t *testing.T) {
	// 0.6 / 1.01 比率形式也应等价
	out := CalcPoints("0.6", "1.01", "0.78", 8800)
	if got := out["parachute_point"]; got != 5280.0 {
		t.Errorf("parachute_point = %v, want 5280", got)
	}
	if got := out["knockout_point"]; got != 8888.0 {
		t.Errorf("knockout_point = %v, want 8888", got)
	}
}

func TestCalcPoints_NoDividendLine(t *testing.T) {
	out := CalcPoints("60%", "101%", "", 8800)
	if _, ok := out["dividend_point"]; ok {
		t.Error("dividend_point 应在派息线为空时省略")
	}
	if _, ok := out["dividend_point_approx"]; ok {
		t.Error("dividend_point_approx 应在派息线为空时省略")
	}
}

func TestCalcPoints_ZeroPrice(t *testing.T) {
	out := CalcPoints("60%", "101%", "78%", 0)
	if len(out) != 0 {
		t.Errorf("currentPrice=0 时应返回空 map，got %v", out)
	}
}

func TestCalcPoints_BadPercent(t *testing.T) {
	out := CalcPoints("abc", "101%", "78%", 8800)
	if _, ok := out["parachute_point"]; ok {
		t.Error("降落伞非法时应省略 parachute_point")
	}
	if got := out["knockout_point"]; got != 8888.0 {
		t.Errorf("knockout_point 仍应算出 = %v, want 8888", got)
	}
}
