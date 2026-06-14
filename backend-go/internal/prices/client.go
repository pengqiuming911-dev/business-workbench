package prices

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const eastmoneyAPI = "https://push2.eastmoney.com/api/qt/stock/get"

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
