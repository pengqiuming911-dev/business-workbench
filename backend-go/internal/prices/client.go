package prices

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const eastmoneyAPI = "https://push2.eastmoney.com/api/qt/stock/get"
const tencentKlineAPI = "https://web.ifzq.gtimg.cn/appstock/app/fqkline/get"

type Result struct {
	Prices map[string]float64
	Failed []string
}

type parsedCode struct {
	Num      string
	Exchange string
}

func FetchAll(ctx context.Context, codes []string) Result {
	result := Result{Prices: map[string]float64{}, Failed: []string{}}
	client := &http.Client{Timeout: 5 * time.Second}
	for _, code := range codes {
		price, err := fetchLatest(ctx, client, code)
		if err != nil {
			result.Failed = append(result.Failed, code)
			continue
		}
		result.Prices[code] = price
	}
	return result
}

func fetchLatest(ctx context.Context, client *http.Client, code string) (float64, error) {
	secid, parsed, err := resolveSecID(code)
	if err != nil {
		return 0, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, eastmoneyAPI+"?secid="+secid+"&fields=f43,f44,f45,f46,f47,f170", nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Referer", "https://quote.eastmoney.com/")

	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return 0, fmt.Errorf("price API status %d", resp.StatusCode)
	}

	var payload struct {
		Data struct {
			F43 float64 `json:"f43"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return 0, err
	}
	if payload.Data.F43 == 0 {
		return 0, fmt.Errorf("no price data for %s", code)
	}
	if strings.HasPrefix(parsed.Num, "513") {
		return payload.Data.F43 / 1000, nil
	}
	return payload.Data.F43 / 100, nil
}

func resolveSecID(code string) (string, parsedCode, error) {
	parsed, ok := parseCode(code)
	if !ok {
		return "", parsedCode{}, fmt.Errorf("invalid code: %s", code)
	}
	market := "0"
	if parsed.Exchange == "SH" && (strings.HasPrefix(parsed.Num, "0") || strings.HasPrefix(parsed.Num, "5")) {
		market = "1"
	}
	return market + "." + parsed.Num, parsed, nil
}

type IndexResult struct {
	Code      string   `json:"code"`
	Name      string   `json:"name"`
	Value     *float64 `json:"value"`
	ChangePct *float64 `json:"change_pct"`
}

func FetchIndices(ctx context.Context, codes []string) []IndexResult {
	results := make([]IndexResult, len(codes))
	for i, code := range codes {
		results[i].Code = code
		parsed, ok := parseCode(code)
		if !ok {
			continue
		}
		market := "1"
		if parsed.Exchange == "SZ" {
			market = "0"
		}
		secid := market + "." + parsed.Num
		client := &http.Client{Timeout: 5 * time.Second}
		endpoint := eastmoneyAPI + "?secid=" + secid + "&fields=f43,f58,f169,f170"
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			continue
		}
		req.Header.Set("User-Agent", "Mozilla/5.0")
		req.Header.Set("Referer", "https://quote.eastmoney.com/")
		resp, err := client.Do(req)
		if err != nil {
			continue
		}
		var payload struct {
			Data struct {
				F43  *float64 `json:"f43"`
				F58  string   `json:"f58"`
				F169 *float64 `json:"f169"`
				F170 *float64 `json:"f170"`
			} `json:"data"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
			resp.Body.Close()
			continue
		}
		resp.Body.Close()

		results[i].Name = payload.Data.F58
		if payload.Data.F43 != nil {
			divisor := 100.0
			if strings.HasPrefix(parsed.Num, "51") {
				divisor = 1000.0
			}
			v := *payload.Data.F43 / divisor
			results[i].Value = &v
		}
		if payload.Data.F170 != nil {
			pct := *payload.Data.F170 / 100
			results[i].ChangePct = &pct
		}
	}
	return results
}

type KlinePoint struct {
	Date  string  `json:"date"`
	Close float64 `json:"close"`
}

type KlineResult struct {
	Code   string       `json:"code"`
	Name   string       `json:"name"`
	Klines []KlinePoint `json:"klines"`
}

func FetchKlines(ctx context.Context, codes []string, days int) []KlineResult {
	results := make([]KlineResult, len(codes))
	client := &http.Client{Timeout: 8 * time.Second}
	for i, code := range codes {
		results[i].Code = code
		parsed, ok := parseCode(code)
		if !ok {
			continue
		}
		symbol := strings.ToLower(parsed.Exchange) + parsed.Num
		param := fmt.Sprintf("%s,day,,,%d,qfq", symbol, days)
		endpoint := tencentKlineAPI + "?param=" + url.QueryEscape(param)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			continue
		}
		req.Header.Set("User-Agent", "Mozilla/5.0")
		req.Header.Set("Referer", "https://gu.qq.com/")
		resp, err := client.Do(req)
		if err != nil {
			continue
		}
		points, err := decodeTencentKlines(resp.Body, symbol)
		resp.Body.Close()
		if err != nil {
			continue
		}
		results[i].Klines = points
	}
	return results
}

func decodeTencentKlines(body interface{ Read([]byte) (int, error) }, symbol string) ([]KlinePoint, error) {
	var payload struct {
		Code int `json:"code"`
		Data map[string]struct {
			Day    [][]string `json:"day"`
			QfqDay [][]string `json:"qfqday"`
		} `json:"data"`
	}
	if err := json.NewDecoder(body).Decode(&payload); err != nil {
		return nil, err
	}
	if payload.Code != 0 {
		return nil, fmt.Errorf("Tencent kline API code %d", payload.Code)
	}
	series, ok := payload.Data[symbol]
	if !ok {
		return nil, fmt.Errorf("no kline data for %s", symbol)
	}
	rows := series.Day
	if len(rows) == 0 {
		rows = series.QfqDay
	}
	points := make([]KlinePoint, 0, len(rows))
	for _, row := range rows {
		if len(row) < 3 {
			continue
		}
		closePrice, err := strconv.ParseFloat(row[2], 64)
		if err != nil {
			continue
		}
		points = append(points, KlinePoint{Date: row[0], Close: closePrice})
	}
	if len(points) == 0 {
		return nil, fmt.Errorf("empty kline data for %s", symbol)
	}
	return points, nil
}

func parseCode(rawCode string) (parsedCode, bool) {
	rawCode = strings.TrimSpace(rawCode)
	if rawCode == "" {
		return parsedCode{}, false
	}
	re := regexp.MustCompile(`(?i)(\d{6})\s*[.\s]*\s*(SH|SZ)`)
	if match := re.FindStringSubmatch(rawCode); len(match) == 3 {
		return parsedCode{Num: match[1], Exchange: strings.ToUpper(match[2])}, true
	}
	cleaned := strings.ToLower(rawCode)
	if len(cleaned) >= 8 && (strings.HasPrefix(cleaned, "sh") || strings.HasPrefix(cleaned, "sz")) {
		return parsedCode{Num: cleaned[2:8], Exchange: strings.ToUpper(cleaned[:2])}, true
	}
	return parsedCode{}, false
}
