package agent

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// parsePercent 把百分比/比率字符串解析成分数。
// "60%" -> 0.6；"101%" -> 1.01；"0.6" -> 0.6；"60" -> 0.6（>2 视为已是百分数）。
// 第二返回值 false 表示非法或零。
func parsePercent(s string) (float64, bool) {
	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(s, "%")
	s = strings.TrimSpace(s)
	v, err := strconv.ParseFloat(s, 64)
	if err != nil || v == 0 {
		return 0, false
	}
	if v > 2 || v < -2 {
		return v / 100, true // 已是百分数（如 60）
	}
	return v, true // 比率（如 0.6）
}

// approxPoint 把绝对点位 floor 到百位并渲染 skill 的口语化「约点」
// （5280 -> "5200点左右"，对齐 skill 示例）。
func approxPoint(point float64) string {
	floored := math.Floor(point/100) * 100
	return fmt.Sprintf("%.0f点左右", floored)
}

// CalcPoints 算结构化产品推介的绝对点位。
// parachute/knockoutLine/dividendLine 是百分比串（"60%"/"101%"/"78%"）或比率（"0.6"）；
// currentPrice 来自 fetch_quote 的实时点位。dividendLine 可为 ""（不适用）。
// 返回绝对点位 + 「约点」；非法字段省略对应键。currentPrice<=0 时返回空 map。
func CalcPoints(parachute, knockoutLine, dividendLine string, currentPrice float64) map[string]any {
	out := map[string]any{}
	if currentPrice <= 0 {
		return out
	}
	if pp, ok := parsePercent(parachute); ok {
		pt := currentPrice * pp
		out["parachute_point"] = pt
		out["parachute_point_approx"] = approxPoint(pt)
	}
	if kp, ok := parsePercent(knockoutLine); ok {
		pt := currentPrice * kp
		out["knockout_point"] = pt
		out["knockout_point_approx"] = approxPoint(pt)
	}
	if dividendLine != "" {
		if dp, ok := parsePercent(dividendLine); ok {
			pt := currentPrice * dp
			out["dividend_point"] = pt
			out["dividend_point_approx"] = approxPoint(pt)
		}
	}
	return out
}

// calcPoints 是 agent 工具入口：从 args 取参数 + current_price，调 CalcPoints。
// current_price 应由 agent 先调 fetch_quote 拿到再传入。
func (s *Service) calcPoints(args map[string]any) map[string]any {
	parachute := stringArg(args, "降落伞")
	knockoutLine := stringArg(args, "期初敲出线")
	dividendLine := stringArg(args, "派息线")
	priceStr := stringArg(args, "current_price")
	currentPrice, err := strconv.ParseFloat(priceStr, 64)
	if err != nil || currentPrice <= 0 {
		return map[string]any{"error": "current_price 缺失或非法（应先调 fetch_quote 拿到点位）"}
	}
	out := CalcPoints(parachute, knockoutLine, dividendLine, currentPrice)
	out["current_price"] = currentPrice
	return out
}
