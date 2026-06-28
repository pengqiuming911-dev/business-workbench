package agent

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// name2Code 标的名 -> 行情代码。Port 自 skill 的 fetch_quote.py NAME2CODE。
var name2Code = map[string]string{
	"中证1000":  "sh000852",
	"中证500":   "sh000905",
	"沪深300":   "sh000300",
	"沪深300指数": "sh000300",
	"上证指数":    "sh000001",
	"上证50":    "sh000016",
	"创业板指":    "sz399006",
	"创业板指数":   "sz399006",
	"科创50":    "sh000688",
	"中证A500":  "sh932000",
}

// 三源 base URL 为包级变量，便于测试用 httptest 覆盖。
var (
	tencentBase   = "https://qt.gtimg.cn"
	sinaBase      = "https://hq.sinajs.cn"
	eastmoneyBase = "https://push2.eastmoney.com"
)

var quoteHTTPClient = &http.Client{Timeout: 8 * time.Second}

// ResolveCode 把标的名或原始代码解析为行情代码。
func ResolveCode(arg string) string {
	arg = strings.TrimSpace(arg)
	if arg == "" {
		return ""
	}
	if code, ok := name2Code[arg]; ok {
		return code
	}
	low := strings.ToLower(arg)
	for k, v := range name2Code {
		if strings.ToLower(k) == low {
			return v
		}
	}
	if (strings.HasPrefix(low, "sh") || strings.HasPrefix(low, "sz")) && len(low) >= 8 {
		return low
	}
	return ""
}

// splitQuoted 取行情行里第一对双引号之间的内容。
func splitQuoted(text string) string {
	i := strings.Index(text, "\"")
	if i < 0 {
		return ""
	}
	rest := text[i+1:]
	j := strings.Index(rest, "\"")
	if j < 0 {
		return rest
	}
	return rest[:j]
}

// fetchFromTencent: v_sh000852="1~000852~中证1000~8810.34~..."  parts[3]=最新价。
func fetchFromTencent(base, code string) (string, float64, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/q=%s", base, code), nil)
	if err != nil {
		return "", 0, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")
	resp, err := quoteHTTPClient.Do(req)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", 0, err
	}
	seg := splitQuoted(string(body))
	if seg == "" {
		return "", 0, fmt.Errorf("tencent: 无引号段")
	}
	parts := strings.Split(seg, "~")
	if len(parts) < 4 {
		return "", 0, fmt.Errorf("tencent: 段数不足")
	}
	name := parts[2]
	price, err := strconv.ParseFloat(parts[3], 64)
	if err != nil {
		return name, 0, fmt.Errorf("tencent: 价格非法 %q", parts[3])
	}
	return name, price, nil
}

// fetchFromSina: hq_str_sh000852="中证1000,开盘,昨收,最新,..."  parts[3]=最新价。需 Referer。
func fetchFromSina(base, code string) (string, float64, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/list=%s", base, code), nil)
	if err != nil {
		return "", 0, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Referer", "https://finance.sina.com.cn")
	resp, err := quoteHTTPClient.Do(req)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", 0, err
	}
	seg := splitQuoted(string(body))
	if seg == "" {
		return "", 0, fmt.Errorf("sina: 无引号段")
	}
	parts := strings.Split(seg, ",")
	if len(parts) < 4 {
		return "", 0, fmt.Errorf("sina: 段数不足")
	}
	name := parts[0]
	price, err := strconv.ParseFloat(parts[3], 64)
	if err != nil {
		return name, 0, fmt.Errorf("sina: 价格非法 %q", parts[3])
	}
	return name, price, nil
}

// fetchFromEastmoney: push2 JSON，data.f43=最新价（单位:分，/100）。
func fetchFromEastmoney(base, code string) (string, float64, error) {
	prefix := "1"
	if strings.HasPrefix(code, "sz") {
		prefix = "0"
	}
	secid := fmt.Sprintf("%s.%s", prefix, code[2:])
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/api/qt/stock/get?secid=%s&fields=f43,f58", base, secid), nil)
	if err != nil {
		return "", 0, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")
	resp, err := quoteHTTPClient.Do(req)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()
	var payload struct {
		Data struct {
			F43 float64 `json:"f43"`
			F58 string  `json:"f58"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", 0, err
	}
	if payload.Data.F43 == 0 {
		return payload.Data.F58, 0, fmt.Errorf("eastmoney: 无 f43")
	}
	return payload.Data.F58, payload.Data.F43 / 100.0, nil
}

// FetchQuote 依次试 腾讯→新浪→东财，首个有效价（>0）胜出。
// 返回 name、price、source（"tencent"|"sina"|"eastmoney"）；全败返回 error。
func FetchQuote(code string) (name string, price float64, source string, err error) {
	type src struct {
		fn   func(string, string) (string, float64, error)
		base string
		name string
	}
	sources := []src{
		{fetchFromTencent, tencentBase, "tencent"},
		{fetchFromSina, sinaBase, "sina"},
		{fetchFromEastmoney, eastmoneyBase, "eastmoney"},
	}
	var lastErr error = fmt.Errorf("no source returned a valid price")
	for _, s := range sources {
		n, p, e := s.fn(s.base, code)
		if e == nil && p > 0 {
			return n, p, s.name, nil
		}
		if e != nil {
			lastErr = e
		}
	}
	return "", 0, "", fmt.Errorf("三源全部失败: %v", lastErr)
}

// fetchQuote 是 agent 工具入口：按标的取实时点位。
func (s *Service) fetchQuote(args map[string]any) map[string]any {
	arg := stringArg(args, "标的")
	if arg == "" {
		arg = stringArg(args, "code")
	}
	if arg == "" {
		return map[string]any{"error": "标的 is required"}
	}
	code := ResolveCode(arg)
	if code == "" {
		return map[string]any{"error": "无法识别标的「" + arg + "」，请用中文名(如 中证1000)或代码(如 sh000852)"}
	}
	name, price, source, err := FetchQuote(code)
	if err != nil {
		return map[string]any{"error": "自动获取失败：" + err.Error() + "。请手动提供 " + arg + " 当前点位"}
	}
	return map[string]any{"标的": name, "code": code, "最新点位": price, "source": source}
}
