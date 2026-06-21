package prices

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/text/encoding/simplifiedchinese"
)

const tencentQuoteAPI = "https://qt.gtimg.cn/q="
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

	// Build batch query symbols
	symbols := make([]string, 0, len(codes))
	symToCode := make(map[string]string, len(codes))
	for _, code := range codes {
		parsed, ok := parseCode(code)
		if !ok {
			result.Failed = append(result.Failed, code)
			continue
		}
		sym := strings.ToLower(parsed.Exchange) + parsed.Num
		symbols = append(symbols, sym)
		symToCode[sym] = code
	}
	if len(symbols) == 0 {
		return result
	}

	client := &http.Client{Timeout: 10 * time.Second}
	endpoint := tencentQuoteAPI + strings.Join(symbols, ",")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		for _, code := range codes {
			result.Failed = append(result.Failed, code)
		}
		return result
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := client.Do(req)
	if err != nil {
		for _, code := range codes {
			result.Failed = append(result.Failed, code)
		}
		return result
	}
	defer resp.Body.Close()

	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		for _, code := range codes {
			result.Failed = append(result.Failed, code)
		}
		return result
	}
	body, err := simplifiedchinese.GBK.NewDecoder().Bytes(rawBody)
	if err != nil {
		for _, code := range codes {
			result.Failed = append(result.Failed, code)
		}
		return result
	}

	found := make(map[string]bool)
	for _, line := range strings.Split(string(body), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || !strings.Contains(line, "=") {
			continue
		}
		eqIdx := strings.Index(line, "=")
		sym := strings.TrimPrefix(line[:eqIdx], "v_")
		code, ok := symToCode[sym]
		if !ok {
			continue
		}
		q1 := strings.Index(line, "\"")
		q2 := strings.LastIndex(line, "\"")
		if q1 < 0 || q2 <= q1 {
			continue
		}
		fields := strings.Split(line[q1+1:q2], "~")
		if len(fields) < 4 {
			continue
		}
		price, err := strconv.ParseFloat(fields[3], 64)
		if err != nil || price <= 0 {
			result.Failed = append(result.Failed, code)
			continue
		}
		result.Prices[code] = price
		found[code] = true
	}

	// Mark codes that weren't in the response as failed
	for _, code := range codes {
		if !found[code] {
			if _, alreadyFailed := result.Prices[code]; !alreadyFailed {
				result.Failed = append(result.Failed, code)
			}
		}
	}

	return result
}

type IndexResult struct {
	Code      string   `json:"code"`
	Name      string   `json:"name"`
	Value     *float64 `json:"value"`
	ChangePct *float64 `json:"change_pct"`
}

func FetchIndices(ctx context.Context, codes []string) []IndexResult {
	results := make([]IndexResult, len(codes))
	codeIndex := make(map[string]int, len(codes))

	// Build batch query symbols: sh000001,sz399006,...
	symbols := make([]string, len(codes))
	for i, code := range codes {
		results[i].Code = code
		parsed, ok := parseCode(code)
		if !ok {
			continue
		}
		sym := strings.ToLower(parsed.Exchange) + parsed.Num
		symbols[i] = sym
		codeIndex[sym] = i
	}

	// Single batch request instead of serial individual requests
	client := &http.Client{Timeout: 8 * time.Second}
	endpoint := tencentQuoteAPI + strings.Join(symbols, ",")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return results
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := client.Do(req)
	if err != nil {
		return results
	}
	defer resp.Body.Close()

	// Response is GBK-encoded; decode to UTF-8
	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return results
	}
	body, err := simplifiedchinese.GBK.NewDecoder().Bytes(rawBody)
	if err != nil {
		return results
	}

	// Parse each line: v_sh000001="1~上证指数~000001~4090.48~..."
	for _, line := range strings.Split(string(body), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || !strings.Contains(line, "=") {
			continue
		}
		// Extract symbol from v_sh000001="..."
		eqIdx := strings.Index(line, "=")
		varPart := line[:eqIdx]        // v_sh000001
		sym := strings.TrimPrefix(varPart, "v_") // sh000001
		idx, ok := codeIndex[sym]
		if !ok {
			continue
		}

		// Extract fields between quotes
		q1 := strings.Index(line, "\"")
		q2 := strings.LastIndex(line, "\"")
		if q1 < 0 || q2 <= q1 {
			continue
		}
		fields := strings.Split(line[q1+1:q2], "~")
		if len(fields) < 33 {
			continue
		}

		// fields[1]=name, fields[3]=current price, fields[32]=change_pct
		results[idx].Name = fields[1]

		if price, err := strconv.ParseFloat(fields[3], 64); err == nil && price > 0 {
			results[idx].Value = &price
		}
		if pct, err := strconv.ParseFloat(fields[32], 64); err == nil {
			results[idx].ChangePct = &pct
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
