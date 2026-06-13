# Go 后端全面迁移实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 Node.js 后端的定时任务、交易日历、邮件通知和 RAG 文档检索全部迁移到 Go，使 `backend-go` 完全取代 `backend`（Node.js）。

**Architecture:** 新增三个独立包（`internal/trading`、`internal/email`、`internal/retriever`），修改 `router.go` 调度器从手写 ticker 切换到 `robfig/cron/v3`，修改 `agent/service.go` 在每轮对话前注入文档上下文。

**Tech Stack:** Go 1.22, Gin, robfig/cron/v3, net/smtp（标准库）

---

## 文件结构

```text
backend-go/
├── cmd/server/main.go                    # 修改: 添加 cron 优雅退出
├── go.mod                                # 修改: 新增 robfig/cron/v3
├── internal/
│   ├── app/router.go                     # 修改: 替换 startScheduler, 新增 3 个定时任务函数
│   ├── agent/service.go                  # 修改: buildMessages 接受 docContext, StreamChat 注入
│   ├── config/config.go                  # 修改: 新增 6 个 SMTP 字段
│   ├── db/repository.go                  # 无修改（ProductDocs("") 已支持全量加载）
│   ├── observations/calendar.go          # 修改: adjustForHoliday 调用 trading 包
│   ├── trading/calendar.go               # 新建: 交易日历
│   ├── trading/calendar_test.go          # 新建: 交易日历测试
│   ├── email/sender.go                   # 新建: 邮件通知服务
│   ├── email/sender_test.go              # 新建: 邮件渲染测试
│   ├── retriever/retriever.go           # 新建: RAG 文档检索
│   └── retriever/retriever_test.go      # 新建: RAG 检索测试
```

---

### Task 1: 添加 `robfig/cron/v3` 依赖

**Files:**
- Modify: `backend-go/go.mod`
- Modify: `backend-go/go.sum`

- [ ] **Step 1: 安装依赖**

Run in `backend-go/`:

```sh
cd backend-go && go get github.com/robfig/cron/v3
```

Expected: `go.mod` 中出现 `github.com/robfig/cron/v3 v3.0.1`，无报错。

- [ ] **Step 2: 验证安装**

```sh
cd backend-go && go mod tidy
```

Expected: 无报错。

- [ ] **Step 3: 提交**

```sh
git add backend-go/go.mod backend-go/go.sum
git commit -m "chore(go): add robfig/cron/v3 dependency"
```

---

### Task 2: 创建交易日历包 (`internal/trading`)

**Files:**
- Create: `backend-go/internal/trading/calendar.go`
- Create: `backend-go/internal/trading/calendar_test.go`

- [ ] **Step 1: 编写测试**

创建 `backend-go/internal/trading/calendar_test.go`：

```go
package trading_test

import (
	"testing"

	"business-workbench/backend-go/internal/trading"
)

func TestIsHoliday(t *testing.T) {
	if !trading.IsHoliday("2026-02-16") {
		t.Error("2026-02-16 should be a holiday (Spring Festival)")
	}
	if trading.IsHoliday("2026-02-25") {
		t.Error("2026-02-25 should not be a holiday")
	}
}

func TestIsWeekend(t *testing.T) {
	// 2026-06-14 is Sunday
	if !trading.IsWeekend("2026-06-14") {
		t.Error("2026-06-14 (Sunday) should be weekend")
	}
	// 2026-06-15 is Monday
	if trading.IsWeekend("2026-06-15") {
		t.Error("2026-06-15 (Monday) should not be weekend")
	}
}

func TestIsTradingDay(t *testing.T) {
	// Monday, not holiday
	if !trading.IsTradingDay("2026-06-15") {
		t.Error("2026-06-15 (Mon, no holiday) should be a trading day")
	}
	// Sunday
	if trading.IsTradingDay("2026-06-14") {
		t.Error("2026-06-14 (Sun) should not be a trading day")
	}
	// Holiday (Spring Festival 2026-02-16 is Monday? — actually a Mon, but it's a national holiday)
	if trading.IsTradingDay("2026-02-16") {
		t.Error("2026-02-16 (holiday) should not be a trading day")
	}
}

func TestAdjustToNearestTradingDay_Postpone(t *testing.T) {
	// 2026-02-16 is holiday (Spring Festival, 2026-02-16..2026-02-22)
	// Nearest trading day after Feb 16 is Feb 23 (Monday, not in holiday list)
	result := trading.AdjustToNearestTradingDay("2026-02-16", "postpone")
	if result == "2026-02-16" {
		t.Errorf("postpone from holiday 2026-02-16 should not return same date, got %s", result)
	}
	if !trading.IsTradingDay(result) {
		t.Errorf("postpone result %s should be a trading day", result)
	}
}

func TestAdjustToNearestTradingDay_Advance(t *testing.T) {
	// 2026-02-16 is holiday; advance should land on 2026-02-13 (Friday, a trading day)
	result := trading.AdjustToNearestTradingDay("2026-02-16", "advance")
	if result == "2026-02-16" {
		t.Errorf("advance from holiday 2026-02-16 should not return same date, got %s", result)
	}
	if !trading.IsTradingDay(result) {
		t.Errorf("advance result %s should be a trading day", result)
	}
}

func TestAdjustToNearestTradingDay_AlreadyTrading(t *testing.T) {
	// 2026-06-15 is a Monday, not a holiday => should return unchanged
	result := trading.AdjustToNearestTradingDay("2026-06-15", "postpone")
	if result != "2026-06-15" {
		t.Errorf("already a trading day should return unchanged, got %s", result)
	}
}

func TestAdjustToNearestTradingDay_Weekend(t *testing.T) {
	// 2026-06-14 (Sunday)
	// advance => 2026-06-12 (Friday)
	// postpone => 2026-06-15 (Monday)
	adv := trading.AdjustToNearestTradingDay("2026-06-14", "advance")
	pos := trading.AdjustToNearestTradingDay("2026-06-14", "postpone")
	if adv == "2026-06-14" || !trading.IsTradingDay(adv) {
		t.Errorf("advance from Sunday should be a prior trading day, got %s", adv)
	}
	if pos != "2026-06-15" {
		t.Errorf("postpone from Sunday 2026-06-14 should be Monday 2026-06-15, got %s", pos)
	}
}
```

- [ ] **Step 2: 运行测试，验证失败**

```sh
cd backend-go && go test ./internal/trading/... -v
```

Expected: FAIL（包不存在）。

- [ ] **Step 3: 创建 `internal/trading/calendar.go`**

```go
package trading

import "time"

var holidays2025 = []string{
	"2025-01-01",
	"2025-01-28", "2025-01-29", "2025-01-30", "2025-01-31",
	"2025-02-01", "2025-02-02", "2025-02-03", "2025-02-04",
	"2025-04-04", "2025-04-05", "2025-04-06",
	"2025-05-01", "2025-05-02", "2025-05-03", "2025-05-04", "2025-05-05",
	"2025-05-31", "2025-06-01", "2025-06-02",
	"2025-10-01", "2025-10-02", "2025-10-03", "2025-10-04",
	"2025-10-05", "2025-10-06", "2025-10-07",
}

var holidays2026 = []string{
	"2026-01-01", "2026-01-02", "2026-01-03",
	"2026-02-16", "2026-02-17", "2026-02-18", "2026-02-19",
	"2026-02-20", "2026-02-21", "2026-02-22",
	"2026-04-04", "2026-04-05", "2026-04-06", "2026-04-07",
	"2026-05-01", "2026-05-02", "2026-05-03",
	"2026-06-19", "2026-06-20", "2026-06-21",
	"2026-09-25", "2026-09-26", "2026-09-27",
	"2026-10-01", "2026-10-02", "2026-10-03", "2026-10-04",
	"2026-10-05", "2026-10-06", "2026-10-07",
}

var holidaySet map[string]bool

func init() {
	holidaySet = make(map[string]bool, len(holidays2025)+len(holidays2026))
	for _, d := range holidays2025 {
		holidaySet[d] = true
	}
	for _, d := range holidays2026 {
		holidaySet[d] = true
	}
}

func IsWeekend(dateStr string) bool {
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return false
	}
	return t.Weekday() == time.Saturday || t.Weekday() == time.Sunday
}

func IsHoliday(dateStr string) bool {
	return holidaySet[dateStr]
}

func IsTradingDay(dateStr string) bool {
	return !IsWeekend(dateStr) && !IsHoliday(dateStr)
}

// direction: "advance" 提前至最近交易日，"postpone" 顺延至最近交易日
// 如果 dateStr 本身是交易日，直接返回。
func AdjustToNearestTradingDay(dateStr, direction string) string {
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return dateStr
	}
	step := 1
	if direction == "advance" {
		step = -1
	}
	for !IsTradingDay(t.Format("2006-01-02")) {
		t = t.AddDate(0, 0, step)
	}
	return t.Format("2006-01-02")
}
```

- [ ] **Step 4: 运行测试，验证全部通过**

```sh
cd backend-go && go test ./internal/trading/... -v
```

Expected: 所有测试 PASS。

- [ ] **Step 5: 提交**

```sh
git add backend-go/internal/trading/
git commit -m "feat(go): add trading calendar with 2025-2026 holidays"
```

---

### Task 3: 将 `adjustForHoliday` 连接交易日历

**Files:**
- Modify: `backend-go/internal/observations/calendar.go` (lines 182-185)
- Create: `backend-go/internal/observations/adjust_test.go`

- [ ] **Step 1: 编写测试**

创建 `backend-go/internal/observations/adjust_test.go`：

```go
package observations

import (
	"testing"

	"business-workbench/backend-go/internal/model"
)

func TestAdjustForHoliday_Postpone(t *testing.T) {
	// 2026-02-16 is Spring Festival holiday (Monday)
	result := adjustForHoliday("2026-02-16", "")
	if result == "2026-02-16" {
		t.Errorf("holiday date should not be returned unchanged, got %s", result)
	}
}

func TestAdjustForHoliday_Advance(t *testing.T) {
	// 2026-02-16 is Spring Festival; with "提前" should advance to prior trading day
	result := adjustForHoliday("2026-02-16", "提前")
	if result == "2026-02-16" {
		t.Errorf("holiday date should be adjusted backward, got %s", result)
	}
	if result > "2026-02-16" {
		t.Errorf("advance should produce earlier date, got %s", result)
	}
}

func TestAdjustForHoliday_NormalDay(t *testing.T) {
	// 2026-06-15 is Monday, not a holiday
	result := adjustForHoliday("2026-06-15", "")
	if result != "2026-06-15" {
		t.Errorf("trading day should be unchanged, got %s", result)
	}
}

func TestDatesUntil_UsesHolidayAdjust(t *testing.T) {
	product := model.Product{
		IssueDate:     "2026-01-01",
		HolidayAdjust: "提前",
	}
	dates := DatesUntil(product, "2026-03-01")
	if len(dates) == 0 {
		t.Fatal("DatesUntil returned no dates")
	}
	for _, d := range dates {
		if d.Date == "" {
			t.Error("observation date should not be empty")
		}
	}
}
```

- [ ] **Step 2: 运行测试，验证测试中 `TestAdjustForHoliday_Postpone` 等应失败（`adjustForHoliday` 当前为空操作）**

```sh
cd backend-go && go test ./internal/observations/... -run 'TestAdjustForHoliday|TestDatesUntil' -v
```

Expected: FAIL（空操作返回原始日期，断言 `!= "2026-02-16"` 失败）。

- [ ] **Step 3: 修改 `internal/observations/calendar.go`**

将旧空操作替换为调用 `trading` 包：

找到这段代码（行 182-185）：

```go
func adjustForHoliday(dateStr, holidayAdjust string) string {
    // The Go migration keeps holiday adjustment neutral until the full trading calendar is ported.
    return dateStr
}
```

替换为：

```go
func adjustForHoliday(dateStr, holidayAdjust string) string {
    if trading.IsTradingDay(dateStr) {
        return dateStr
    }
    direction := "postpone"
    if holidayAdjust == "提前" {
        direction = "advance"
    }
    return trading.AdjustToNearestTradingDay(dateStr, direction)
}
```

并在文件顶部 `import` 块中添加：

```go
import (
    "math"
    "sort"
    "time"

    "business-workbench/backend-go/internal/model"
    "business-workbench/backend-go/internal/trading"
)
```

- [ ] **Step 4: 运行测试，验证全部通过**

```sh
cd backend-go && go test ./internal/observations/... -v
```

Expected: 所有测试 PASS，包括新增的 4 个。

- [ ] **Step 5: 提交**

```sh
git add backend-go/internal/observations/
git commit -m "feat(go): wire adjustForHoliday to trading calendar"
```

---

### Task 4: 更新 Config 添加 SMTP 字段

**Files:**
- Modify: `backend-go/internal/config/config.go`

- [ ] **Step 1: 修改 `config.go`**

在 `Config` struct 末尾追加 6 个字段：

```go
type Config struct {
    Port              string
    FrontendURL       string
    DatabasePath      string
    FeishuAppID       string
    FeishuAppSecret   string
    FeishuRedirectURI string
    DeepSeekAPIKey    string
    DeepSeekAPIURL    string
    DeepSeekModel     string
    CronTimezone      string
    FeishuPushWebhook string
    SMTPHost          string
    SMTPPort          string
    SMTPSecure        string
    SMTPUser          string
    SMTPPass          string
    SMTPFrom          string
}
```

在 `Load()` 函数的 `return Config{...}` 块末尾追加：

```go
SMTPHost:   os.Getenv("SMTP_HOST"),
SMTPPort:   getEnv("SMTP_PORT", "465"),
SMTPSecure: getEnv("SMTP_SECURE", "true"),
SMTPUser:   os.Getenv("SMTP_USER"),
SMTPPass:   os.Getenv("SMTP_PASS"),
SMTPFrom:   getEnv("SMTP_FROM", os.Getenv("SMTP_USER")),
```

- [ ] **Step 2: 编译验证**

```sh
cd backend-go && go build ./cmd/server
```

Expected: 无报错。

- [ ] **Step 3: 提交**

```sh
git add backend-go/internal/config/config.go
git commit -m "feat(go): add SMTP config fields for email notifications"
```

---

### Task 5: 创建邮件通知包 (`internal/email`)

**Files:**
- Create: `backend-go/internal/email/sender.go`
- Create: `backend-go/internal/email/sender_test.go`

- [ ] **Step 1: 编写测试**

创建 `backend-go/internal/email/sender_test.go`：

```go
package email

import (
	"strings"
	"testing"

	"business-workbench/backend-go/internal/model"
)

func TestRenderText(t *testing.T) {
	products := []NotificationProduct{
		{
			Product: model.Product{ID: "F001", Name: "测试产品", Manager: "管理人A", Code: "000001"},
			EntryPriceStr: "100.00",
			Obs:           ObservationSnapshot{UnderlyingPriceStr: "105.50", KnockoutPriceStr: "110.00", DividendLineStr: "95.00", IsKnockedOut: "否", IsDividend: "是"},
		},
	}
	text := renderText("2026-06-13", products)
	for _, want := range []string{"2026-06-13", "测试产品", "管理人A", "否", "是"} {
		if !strings.Contains(text, want) {
			t.Errorf("renderText missing %q\n%s", want, text)
		}
	}
}

func TestRenderHtml(t *testing.T) {
	products := []NotificationProduct{
		{
			Product: model.Product{ID: "F001", Name: "测试产品", Manager: "管理人A", Code: "000001"},
			EntryPriceStr: "100.00",
			Obs:           ObservationSnapshot{UnderlyingPriceStr: "105.50", KnockoutPriceStr: "110.00", DividendLineStr: "95.00", IsKnockedOut: "否", IsDividend: "是"},
		},
	}
	html := renderHtml("2026-06-13", products)
	if !strings.Contains(html, "<table") || !strings.Contains(html, "</table>") {
		t.Error("renderHtml should produce a table")
	}
	if !strings.Contains(html, "测试产品") {
		t.Error("renderHtml should include product name")
	}
}

func TestEscapeHtml(t *testing.T) {
	input := `<script>alert("xss")</script>`
	got := escapeHtml(input)
	if strings.Contains(got, "<") || strings.Contains(got, ">") || strings.Contains(got, `"`) {
		t.Errorf("escapeHtml should sanitize, got %q", got)
	}
	if !strings.Contains(got, "&lt;") {
		t.Error("escapeHtml should replace < with &lt;")
	}
}

func TestBuildTodayNotification_NoProducts(t *testing.T) {
	cfg := Config{SMTPHost: "", SMTPUser: "", SMTPPass: ""}
	n := BuildTodayNotification(nil, nil, "2026-06-13", cfg)
	if n.Subject == "" {
		t.Error("subject should not be empty")
	}
	if len(n.Products) != 0 {
		t.Errorf("expected 0 products, got %d", len(n.Products))
	}
}

func TestSendObservationEmail_NoSMTP(t *testing.T) {
	cfg := Config{}
	n := &Notification{Products: []NotificationProduct{{}}}
	sent, reason := SendObservationEmail(cfg, n)
	if sent {
		t.Error("should not send when SMTP not configured")
	}
	if reason != "smtp-not-configured" {
		t.Errorf("expected smtp-not-configured, got %s", reason)
	}
}

func TestSendObservationEmail_NoProducts(t *testing.T) {
	cfg := Config{SMTPHost: "smtp.example.com", SMTPUser: "u", SMTPPass: "p"}
	n := &Notification{Products: nil}
	sent, reason := SendObservationEmail(cfg, n)
	if sent {
		t.Error("should not send when no products")
	}
	if reason != "no-product" {
		t.Errorf("expected no-product, got %s", reason)
	}
}
```

- [ ] **Step 2: 运行测试，验证失败**

```sh
cd backend-go && go test ./internal/email/... -v
```

Expected: FAIL（包不存在）。

- [ ] **Step 3: 创建 `internal/email/sender.go`**

```go
package email

import (
	"fmt"
	"net/smtp"
	"strings"

	"business-workbench/backend-go/internal/model"
	"business-workbench/backend-go/internal/observations"
)

type Config struct {
	SMTPHost   string
	SMTPPort   string
	SMTPSecure string
	SMTPUser   string
	SMTPPass   string
	SMTPFrom   string
}

type ObservationSnapshot struct {
	UnderlyingPriceStr string
	KnockoutPriceStr   string
	DividendLineStr    string
	IsKnockedOut       string
	IsDividend         string
}

type NotificationProduct struct {
	Product       model.Product
	EntryPriceStr string
	Obs           ObservationSnapshot
}

type Notification struct {
	Recipient string
	Subject   string
	Text      string
	HTML      string
	Products  []NotificationProduct
}

const defaultRecipient = "pengqiuming@iyanxuna.cn"

func formatPrice(value *float64) string {
	if value == nil {
		return "--"
	}
	return fmt.Sprintf("%.2f", *value)
}

func formatValue(v interface{}) string {
	if v == nil {
		return "--"
	}
	return strings.TrimSpace(fmt.Sprint(v))
}

func BuildTodayNotification(products []model.Product, prices map[string]float64, today string, cfg Config) *Notification {
	result := []NotificationProduct{}
	for _, product := range products {
		if product.Code == "" {
			continue
		}
		price, ok := prices[product.Code]
		if !ok {
			continue
		}
		obsInfo := findTodayObsInfo(product, today)
		if obsInfo.MonthsSinceEntry == 0 && obsInfo.Date == "" {
			continue
		}
		eval := observations.EvaluateObservation(product, today, price, obsInfo.MonthsSinceEntry)
		result = append(result, NotificationProduct{
			Product:       product,
			EntryPriceStr: formatPrice(product.EntryPrice),
			Obs:           ObservationSnapshot{
				UnderlyingPriceStr: fmt.Sprintf("%.2f", eval.UnderlyingPrice),
				KnockoutPriceStr:   formatPrice(eval.KnockoutPrice),
				DividendLineStr:    formatPrice(eval.DividendLine),
				IsKnockedOut:       eval.IsKnockedOut,
				IsDividend:         eval.IsDividend,
			},
		})
	}
	recipient := defaultRecipient
	subject := fmt.Sprintf("今日产品派息/敲出观察提醒（%s，%d个）", today, len(result))
	return &Notification{
		Recipient: recipient,
		Subject:   subject,
		Text:      renderText(today, result),
		HTML:      renderHtml(today, result),
		Products:  result,
	}
}

type obsDateInfo struct {
	Date             string
	MonthsSinceEntry int
}

func findTodayObsInfo(product model.Product, today string) obsDateInfo {
	for _, d := range observationDatesUntil(product, today) {
		if d.Date == today {
			return obsDateInfo{Date: d.Date, MonthsSinceEntry: d.MonthsSinceEntry}
		}
	}
	return obsDateInfo{}
}

func observationDatesUntil(product model.Product, today string) []struct {
	Date             string
	MonthsSinceEntry int
} {
	if product.IssueDate == "" {
		return nil
	}
	var result []struct {
		Date             string
		MonthsSinceEntry int
	}
	for months := 1; months < 600; months++ {
		raw := addMonthsLocal(product.IssueDate, months)
		if raw == "" || raw > today {
			break
		}
		adjusted := adjustLocal(raw, product.HolidayAdjust)
		result = append(result, struct {
			Date             string
			MonthsSinceEntry int
		}{Date: adjusted, MonthsSinceEntry: months})
	}
	return result
}

func addMonthsLocal(dateStr string, months int) string {
	t, err := parseDate(dateStr)
	if err != nil {
		return ""
	}
	targetDay := t.Day()
	next := t.AddDate(0, months, 0)
	if next.Day() != targetDay {
		from := t.AddDate(0, months, -targetDay+1)
		_ = from
		next = next.AddDate(0, 0, -next.Day())
	}
	return next.Format("2006-01-02")
}

func adjustLocal(dateStr, holidayAdjust string) string {
	// Inline: import trading package
	if isTradingDayLocal(dateStr) {
		return dateStr
	}
	direction := "postpone"
	if holidayAdjust == "提前" {
		direction = "advance"
	}
	return adjustTradingDayLocal(dateStr, direction)
}
```

**注意**：为避免 `internal/email` 对 `internal/observations` 的依赖产生循环（两者相互依赖 `observations` 调用 `trading`，`email` 调用 `observations`），将 `observationDatesUntil` 中的节假日调整直接调用 `trading` 包，并复制 `addMonths` 逻辑。

完整文件（合并后覆盖上面片段）：

```go
package email

import (
	"fmt"
	"net/smtp"
	"strings"

	"business-workbench/backend-go/internal/model"
	"business-workbench/backend-go/internal/observations"
)

type Config struct {
	SMTPHost   string
	SMTPPort   string
	SMTPSecure string
	SMTPUser   string
	SMTPPass   string
	SMTPFrom   string
}

type ObservationSnapshot struct {
	UnderlyingPriceStr string
	KnockoutPriceStr   string
	DividendLineStr    string
	IsKnockedOut       string
	IsDividend         string
}

type NotificationProduct struct {
	Product       model.Product
	EntryPriceStr string
	Obs           ObservationSnapshot
}

type Notification struct {
	Recipient string
	Subject   string
	Text      string
	HTML      string
	Products  []NotificationProduct
}

const defaultRecipient = "pengqiuming@iyanxuna.cn"

func formatPrice(value *float64) string {
	if value == nil {
		return "--"
	}
	return fmt.Sprintf("%.2f", *value)
}

func BuildTodayNotification(products []model.Product, prices map[string]float64, today string, cfg Config) *Notification {
	result := []NotificationProduct{}
	for _, product := range products {
		if product.Code == "" {
			continue
		}
		price, ok := prices[product.Code]
		if !ok {
			continue
		}
		info := todayObsInfo(product, today)
		if info.monthsSinceEntry == 0 && info.date == "" {
			continue
		}
		eval := observations.EvaluateObservation(product, today, price, info.monthsSinceEntry)
		result = append(result, NotificationProduct{
			Product:       product,
			EntryPriceStr: formatPrice(product.EntryPrice),
			Obs: ObservationSnapshot{
				UnderlyingPriceStr: fmt.Sprintf("%.2f", eval.UnderlyingPrice),
				KnockoutPriceStr:   formatPrice(eval.KnockoutPrice),
				DividendLineStr:    formatPrice(eval.DividendLine),
				IsKnockedOut:       eval.IsKnockedOut,
				IsDividend:         eval.IsDividend,
			},
		})
	}
	subject := fmt.Sprintf("今日产品派息/敲出观察提醒（%s，%d个）", today, len(result))
	return &Notification{
		Recipient: defaultRecipient,
		Subject:   subject,
		Text:      renderText(today, result),
		HTML:      renderHtml(today, result),
		Products:  result,
	}
}

type todayObs struct {
	date             string
	monthsSinceEntry int
}

func todayObsInfo(product model.Product, today string) todayObs {
	for _, d := range observations.DatesUntil(product, today) {
		if d.Date == today {
			return todayObs{date: d.Date, monthsSinceEntry: d.MonthsSinceEntry}
		}
	}
	return todayObs{}
}

func SendObservationEmail(cfg Config, n *Notification) (sent bool, reason string) {
	if n == nil || len(n.Products) == 0 {
		return false, "no-product"
	}
	if cfg.SMTPHost == "" || cfg.SMTPUser == "" || cfg.SMTPPass == "" {
		return false, "smtp-not-configured"
	}
	from := cfg.SMTPFrom
	if from == "" {
		from = cfg.SMTPUser
	}
	port := cfg.SMTPPort
	if port == "" {
		port = "465"
	}
	addr := cfg.SMTPHost + ":" + port
	auth := smtp.PlainAuth("", cfg.SMTPUser, cfg.SMTPPass, cfg.SMTPHost)
	body := buildMIMEBody(from, n.Recipient, n.Subject, n.Text, n.HTML)
	err := smtp.SendMail(addr, auth, from, []string{n.Recipient}, []byte(body))
	if err != nil {
		return false, "send-error: " + err.Error()
	}
	return true, ""
}

func buildMIMEBody(from, to, subject, text, html string) string {
	boundary := "GoMailBoundary"
	var b strings.Builder
	b.WriteString("From: " + from + "\r\n")
	b.WriteString("To: " + to + "\r\n")
	b.WriteString("Subject: " + subject + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: multipart/alternative; boundary=" + boundary + "\r\n")
	b.WriteString("\r\n")
	b.WriteString("--" + boundary + "\r\n")
	b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	b.WriteString("\r\n" + text + "\r\n")
	b.WriteString("--" + boundary + "\r\n")
	b.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	b.WriteString("\r\n" + html + "\r\n")
	b.WriteString("--" + boundary + "--\r\n")
	return b.String()
}

func renderText(today string, products []NotificationProduct) string {
	lines := []string{
		"今日产品派息/敲出观察提醒",
		"观察日期：" + today,
		fmt.Sprintf("今日需要观察产品数量：%d", len(products)),
		"",
	}
	for _, p := range products {
		lines = append(lines,
			"产品："+formatValue(p.Product.Name),
			"航班编号："+formatValue(p.Product.ID),
			"私募管理人："+formatValue(p.Product.Manager),
			"标的代码："+formatValue(p.Product.Code),
			"入场价："+p.EntryPriceStr,
			"实时标的价格："+p.Obs.UnderlyingPriceStr,
			"敲出价："+p.Obs.KnockoutPriceStr,
			"派息线："+p.Obs.DividendLineStr,
			"是否敲出："+p.Obs.IsKnockedOut,
			"是否派息："+p.Obs.IsDividend,
			"",
		)
	}
	return strings.Join(lines, "\n")
}

func renderHtml(today string, products []NotificationProduct) string {
	var rows strings.Builder
	for _, p := range products {
		rows.WriteString(fmt.Sprintf(`<tr>
			<td>%s</td><td>%s</td><td>%s</td><td>%s</td>
			<td style="text-align:right">%s</td><td style="text-align:right">%s</td>
			<td style="text-align:right">%s</td><td style="text-align:right">%s</td>
			<td style="text-align:center">%s</td><td style="text-align:center">%s</td>
		</tr>`,
			escapeHtml(p.Product.ID), escapeHtml(p.Product.Name),
			escapeHtml(p.Product.Manager), escapeHtml(p.Product.Code),
			escapeHtml(p.EntryPriceStr), escapeHtml(p.Obs.UnderlyingPriceStr),
			escapeHtml(p.Obs.KnockoutPriceStr), escapeHtml(p.Obs.DividendLineStr),
			escapeHtml(p.Obs.IsKnockedOut), escapeHtml(p.Obs.IsDividend),
		))
	}
	return fmt.Sprintf(`<div>
		<h2>今日产品派息/敲出观察提醒</h2>
		<p>观察日期：%s；今日需要观察产品数量：%d</p>
		<table border="1" cellpadding="8" cellspacing="0" style="border-collapse:collapse;font-size:13px;">
			<thead><tr>
				<th>航班编号</th><th>产品名称</th><th>私募管理人</th><th>标的代码</th>
				<th>入场价</th><th>实时标的价格</th><th>敲出价</th><th>派息线</th>
				<th>是否敲出</th><th>是否派息</th>
			</tr></thead>
			<tbody>%s</tbody>
		</table>
	</div>`, escapeHtml(today), len(products), rows.String())
}

func escapeHtml(value any) string {
	s := fmt.Sprint(value)
	if value == nil {
		s = "--"
	}
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	s = strings.ReplaceAll(s, "'", "&#39;")
	return s
}

func formatValue(v interface{}) string {
	if v == nil {
		return "--"
	}
	s := strings.TrimSpace(fmt.Sprint(v))
	if s == "" || s == "<nil>" {
		return "--"
	}
	return s
}
```

- [ ] **Step 4: 运行测试，验证全部通过**

```sh
cd backend-go && go test ./internal/email/... -v
```

Expected: 所有测试 PASS。

- [ ] **Step 5: 提交**

```sh
git add backend-go/internal/email/
git commit -m "feat(go): add email notification service for observations"
```

---

### Task 6: 创建 RAG 文档检索包 (`internal/retriever`)

**Files:**
- Create: `backend-go/internal/retriever/retriever.go`
- Create: `backend-go/internal/retriever/retriever_test.go`

- [ ] **Step 1: 编写测试**

创建 `backend-go/internal/retriever/retriever_test.go`：

```go
package retriever

import (
	"strings"
	"testing"
)

func TestSearchDocs_Empty(t *testing.T) {
	docs := []map[string]any{}
	result := SearchDocs(docs, "anything", 5)
	if len(result) != 0 {
		t.Errorf("expected 0 results, got %d", len(result))
	}
}

func TestSearchDocs_Score(t *testing.T) {
	docs := []map[string]any{
		{"doc_name": "产品月报", "parent_path": "/2024/01", "raw_content": "这是关于敲出产品的分析", "structure_json": ""},
		{"doc_name": "派息说明", "parent_path": "/2024/02", "raw_content": "派息线计算方式和派息条件", "structure_json": ""},
		{"doc_name": "无关文档", "parent_path": "/other", "raw_content": "其他内容", "structure_json": ""},
	}
	result := SearchDocs(docs, "派息", 5)
	if len(result) == 0 {
		t.Fatal("expected at least 1 result for keyword 派息")
	}
	if result[0].DocName != "派息说明" {
		t.Errorf("expected top result to be '派息说明', got %q", result[0].DocName)
	}
}

func TestSearchDocs_Limit(t *testing.T) {
	docs := []map[string]any{
		{"doc_name": "A", "parent_path": "/", "raw_content": "关键词 关键词 关键词", "structure_json": ""},
		{"doc_name": "B", "parent_path": "/", "raw_content": "关键词", "structure_json": ""},
		{"doc_name": "C", "parent_path": "/", "raw_content": "关键词 关键词", "structure_json": ""},
	}
	result := SearchDocs(docs, "关键词", 2)
	if len(result) != 2 {
		t.Errorf("expected limit 2 results, got %d", len(result))
	}
}

func TestSearchDocs_MultiKeyword(t *testing.T) {
	docs := []map[string]any{
		{"doc_name": "敲出分析", "parent_path": "/report", "raw_content": "这个产品已经敲出了", "structure_json": ""},
		{"doc_name": "派息报告", "parent_path": "/report", "raw_content": "这个产品派息了", "structure_json": ""},
	}
	result := SearchDocs(docs, "敲出 派息", 5)
	if len(result) != 2 {
		t.Errorf("expected 2 results matching either keyword, got %d", len(result))
	}
}

func TestBuildDocContext_Empty(t *testing.T) {
	ctx := BuildDocContext(nil)
	if ctx != "" {
		t.Errorf("expected empty string for nil docs, got %q", ctx)
	}
}

func TestBuildDocContext_Format(t *testing.T) {
	scored := []ScoredDoc{
		{DocName: "报告A", ParentPath: "/2024", RawContent: "内容A", Score: 3},
		{DocName: "报告B", ParentPath: "/2025", RawContent: "内容B", Score: 2},
	}
	ctx := BuildDocContext(scored)
	if !strings.Contains(ctx, "[文档1] 报告A") {
		t.Error("context should contain [文档1] 报告A")
	}
	if !strings.Contains(ctx, "[文档2] 报告B") {
		t.Error("context should contain [文档2] 报告B")
	}
	if !strings.Contains(ctx, "请参考这些文档回答问题") {
		t.Error("context should contain the intro sentence")
	}
}
```

- [ ] **Step 2: 运行测试，验证失败**

```sh
cd backend-go && go test ./internal/retriever/... -v
```

Expected: FAIL。

- [ ] **Step 3: 创建 `internal/retriever/retriever.go`**

```go
package retriever

import (
	"encoding/json"
	"fmt"
	"strings"
)

type ScoredDoc struct {
	DocName    string
	ParentPath string
	RawContent string
	Structure  any
	Score      int
}

func SearchDocs(docs []map[string]any, query string, limit int) []ScoredDoc {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil
	}
	keywords := strings.Fields(query)

	var scored []ScoredDoc
	for _, doc := range docs {
		docName, _ := doc["doc_name"].(string)
		parentPath, _ := doc["parent_path"].(string)
		rawContent, _ := doc["raw_content"].(string)
		structureJSON, _ := doc["structure_json"].(string)

		text := strings.ToLower(docName + " " + parentPath + " " + rawContent)
		score := 0
		for _, kw := range keywords {
			kwLower := strings.ToLower(kw)
			idx := strings.Index(text, kwLower)
			for idx != -1 {
				score++
				idx = strings.Index(text[idx+len(kwLower):], kwLower)
			}
		}
		if score == 0 {
			continue
		}
		var structure any
		if structureJSON != "" {
			_ = json.Unmarshal([]byte(structureJSON), &structure)
		}
		scored = append(scored, ScoredDoc{
			DocName:    docName,
			ParentPath: parentPath,
			RawContent: rawContent,
			Structure:  structure,
			Score:      score,
		})
	}

	sortScored(scored)
	if limit > 0 && len(scored) > limit {
		scored = scored[:limit]
	}
	return scored
}

func sortScored(s []ScoredDoc) {
	for i := 0; i < len(s); i++ {
		for j := i + 1; j < len(s); j++ {
			if s[j].Score > s[i].Score {
				s[i], s[j] = s[j], s[i]
			}
		}
	}
}

func BuildDocContext(scored []ScoredDoc) string {
	if len(scored) == 0 {
		return ""
	}
	var parts []string
	for i, d := range scored {
		structStr := ""
		if d.Structure != nil {
			j, _ := json.Marshal(d.Structure)
			structStr = fmt.Sprintf("\n结构信息：%s", string(j))
		}
		parts = append(parts, fmt.Sprintf("[文档%d] %s (%s)\n%s%s", i+1, d.DocName, d.ParentPath, d.RawContent, structStr))
	}
	return "\n\n以下是与用户问题相关的文档资料，请参考这些文档回答问题：\n\n" + strings.Join(parts, "\n\n---\n\n")
}
```

- [ ] **Step 4: 运行测试，验证全部通过**

```sh
cd backend-go && go test ./internal/retriever/... -v
```

Expected: 所有测试 PASS。

- [ ] **Step 5: 提交**

```sh
git add backend-go/internal/retriever/
git commit -m "feat(go): add RAG document retriever for agent context"
```

---

### Task 7: 将 RAG 检索集成到 Agent Service

**Files:**
- Modify: `backend-go/internal/db/repository.go` (新增 `AllProductDocs` 方法)
- Modify: `backend-go/internal/agent/service.go` (修改 `buildMessages`、`StreamChat`)

- [ ] **Step 1: 在 `repository.go` 末尾添加 `AllProductDocs` 方法**

在 `internal/db/repository.go` 文件中，`SearchProductDocs` 函数之前（约行 798），插入：

```go
func (s *Store) AllProductDocs() ([]map[string]any, error) {
	query := `SELECT doc_token, doc_name, parent_path, folder_token, raw_content, structure_json, synced_at
		FROM product_docs ORDER BY parent_path, doc_name`
	return s.queryMaps(query)
}
```

- [ ] **Step 2: 修改 `internal/agent/service.go`**

在文件顶部 `import` 块中添加：

```go
import (
    "bufio"
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "sort"
    "strings"
    "time"

    "business-workbench/backend-go/internal/config"
    "business-workbench/backend-go/internal/db"
    "business-workbench/backend-go/internal/model"
    "business-workbench/backend-go/internal/observations"
    "business-workbench/backend-go/internal/retriever"
)
```

**修改 `buildMessages` 函数签名**（原来是包级函数，现在改为接收 docContext 参数）：

找到：

```go
func buildMessages(history []model.AgentMessage, userMessage string) []chatMessage {
    messages := []chatMessage{{Role: "system", Content: systemPrompt}}
```

替换为：

```go
func buildMessages(history []model.AgentMessage, userMessage string, docContext string) []chatMessage {
    prompt := systemPrompt
    if docContext != "" {
        prompt += "\n" + docContext
    }
    messages := []chatMessage{{Role: "system", Content: prompt}}
```

**修改 `StreamChat` 方法**，在调用 `buildMessages` 前注入文档检索：

找到：

```go
func (s *Service) StreamChat(ctx context.Context, history []model.AgentMessage, userMessage string, callbacks StreamCallbacks) (string, error) {
    if strings.TrimSpace(s.cfg.DeepSeekAPIKey) == "" {
        return "", fmt.Errorf("DEEPSEEK_API_KEY not configured")
    }

    messages := buildMessages(history, userMessage)
```

替换为：

```go
func (s *Service) StreamChat(ctx context.Context, history []model.AgentMessage, userMessage string, callbacks StreamCallbacks) (string, error) {
    if strings.TrimSpace(s.cfg.DeepSeekAPIKey) == "" {
        return "", fmt.Errorf("DEEPSEEK_API_KEY not configured")
    }

    docContext := ""
    allDocs, err := s.store.AllProductDocs()
    if err == nil && len(allDocs) > 0 {
        scored := retriever.SearchDocs(allDocs, userMessage, 5)
        docContext = retriever.BuildDocContext(scored)
    }
    messages := buildMessages(history, userMessage, docContext)
```

- [ ] **Step 3: 编译验证**

```sh
cd backend-go && go build ./cmd/server
```

Expected: 无错误。

- [ ] **Step 4: 运行现有测试（如有）**

```sh
cd backend-go && go test ./internal/agent/... -v
```

Expected: 无错误（agent 包无现有测试，跳过不影响）。

- [ ] **Step 5: 提交**

```sh
git add backend-go/internal/db/repository.go backend-go/internal/agent/service.go
git commit -m "feat(go): inject RAG doc context into agent system prompt"
```

---

### Task 8: 替换调度器（`startScheduler`）并新增 3 个定时任务

**Files:**
- Modify: `backend-go/internal/app/router.go` (Server struct、startScheduler、新增方法)
- Modify: `backend-go/cmd/server/main.go` (返回 cron 实例，优雅退出)

- [ ] **Step 1: 修改 `router.go` — 引入 cron，更新 Server struct**

在 `router.go` 顶部 `import` 块添加（`posters`、`observations`、`prices` 已存在）：

```go
"business-workbench/backend-go/internal/email"

"github.com/robfig/cron/v3"
```

将 `Server` struct 改为：

```go
type Server struct {
    cfg      config.Config
    store    *db.Store
    agentSvc *agent.Service
    feishu   *feishu.Client
    Cron     *cron.Cron
}
```

- [ ] **Step 2: 修改 `NewRouter` — 初始化 cron**

在 `NewRouter` 函数，`server := &Server{...}` 改为：

```go
location, _ := time.LoadLocation(cfg.CronTimezone)
if location == nil {
    location = time.FixedZone("Asia/Shanghai", 8*60*60)
}
server := &Server{
    cfg:      cfg,
    store:    store,
    agentSvc: agent.NewService(cfg, store),
    feishu:   feishu.New(cfg.FeishuAppID, cfg.FeishuAppSecret, cfg.FeishuRedirectURI),
    Cron:     cron.New(cron.WithLocation(location)),
}
```

在 `server.startScheduler()` 之前，替换为：

```go
server.startScheduler()
server.Cron.Start()
return router
```

（原来 `startScheduler` 内部启动自己的 goroutine，改为内部注册 cron 任务但不启动，由 `NewRouter` 统一 Start 和管理生命周期。）

- [ ] **Step 3: 完全替换 `startScheduler` 方法**

删除原有 `startScheduler`（行 935-963），替换为：

```go
func (s *Server) startScheduler() {
    s.Cron.AddFunc("30 11 * * 1-5", s.scheduledPriceUpdate)
    s.Cron.AddFunc("0 15 * * 1-5", s.scheduledPriceUpdate)
    s.Cron.AddFunc("30 15 * * 1-5", s.scheduledPriceUpdate)
    s.Cron.AddFunc("5 15 * * 1-5", s.generateAutoPosters)
    s.Cron.AddFunc("0 10 * * *", s.scheduledObservationEmail)
    s.Cron.AddFunc("10 15 * * *", s.scheduledObservationEmail)
    s.Cron.AddFunc("* * * * *", s.handleFeishuPushMinute)
}
```

- [ ] **Step 4: 新增 `handleFeishuPushMinute`（替代旧 ticker）**

在 `startScheduler` 之后添加：

```go
var feishuLastRunKey string

func (s *Server) handleFeishuPushMinute() {
    now := time.Now()
    cfg, err := s.store.GetPushConfig(s.cfg.FeishuPushWebhook)
    if err != nil || !cfg.Enabled || strings.TrimSpace(cfg.WebhookURL) == "" {
        return
    }
    if now.Hour() != cfg.CronHour || now.Minute() != cfg.CronMinute {
        return
    }
    runKey := now.Format("2006-01-02 15:04")
    if runKey == feishuLastRunKey {
        return
    }
    feishuLastRunKey = runKey
    count, pushErr := s.executeScheduledObservationPush(context.Background(), cfg.WebhookURL)
    result := fmt.Sprintf("success (%d products)", count)
    if pushErr != nil {
        result = "error: " + pushErr.Error()
    }
    _ = s.store.UpdatePushResult(time.Now().UTC().Format(time.RFC3339Nano), result)
}
```

- [ ] **Step 5: 新增 `scheduledPriceUpdate` 方法**

```go
func (s *Server) scheduledPriceUpdate() {
    ctx := context.Background()
    products, err := s.store.QueryOngoingProducts()
    if err != nil {
        fmt.Printf("[定时任务] 获取进行中产品失败: %v\n", err)
        return
    }
    codes := uniqueProductCodes(products)
    if len(codes) == 0 {
        return
    }
    fmt.Printf("[定时任务] 开始更新 %d 个标的价格...\n", len(codes))
    priceResult := prices.FetchAll(ctx, codes)
    today := time.Now().Format("2006-01-02")
    for code, price := range priceResult.Prices {
        if err := s.store.UpsertPrice(code, today, price); err != nil {
            fmt.Printf("[定时任务] 写入价格 %s 失败: %v\n", code, err)
            return
        }
    }
    updatedObs, err := s.updateTodayObservations(products, today, priceResult.Prices)
    if err != nil {
        fmt.Printf("[定时任务] 更新今日观察记录失败: %v\n", err)
        return
    }
    failed := len(priceResult.Failed)
    fmt.Printf("[定时任务] 完成: 价格更新 %d/%d, 观察记录更新 %d 条\n", len(codes)-failed, len(codes), updatedObs)
}
```

`updateTodayObservations` 是对现有 `updateTodayObservationRecords` 的复用包装，返回更新数：

```go
func (s *Server) updateTodayObservations(products []model.Product, today string, latest map[string]float64) (int, error) {
    count := 0
    for _, product := range products {
        obs, ok := observationInfoForDate(product, today)
        if !ok {
            continue
        }
        price, ok := priceForObservation(s.store, product, today, latest)
        if !ok {
            continue
        }
        eval := observations.EvaluateObservation(product, today, price, obs.MonthsSinceEntry)
        if err := s.store.UpsertObservation(product.ID, observationEvalMap(eval)); err != nil {
            return count, err
        }
        count++
    }
    return count, nil
}
```

- [ ] **Step 6: 新增 `generateAutoPosters` 方法（复用现有 `generatePosters` HTTP handler 的逻辑）**

```go
func (s *Server) generateAutoPosters() {
	today := time.Now().Format("2006-01-02")
	products, err := s.store.QueryOngoingProducts()
	if err != nil {
		fmt.Printf("[喜报生成] 获取产品失败: %v\n", err)
		return
	}
	knockoutCount, dividendCount := 0, 0
	for _, product := range products {
		if product.Code == "" {
			continue
		}
		targetObsInfo, ok := observationInfoForDate(product, today)
		if !ok {
			continue
		}
		records, err := s.store.QueryObservationsByProduct(product.ID)
		if err != nil {
			continue
		}
		var targetRecord *model.Observation
		for i := range records {
			if records[i].ObservationDate == today {
				targetRecord = &records[i]
				break
			}
		}
		if targetRecord == nil {
			continue
		}
		data := posters.GenerateData(product, today, targetObsInfo.MonthsSinceEntry)
		isKnockout := data.KnockoutValue != "" && targetRecord.IsKnockedOut == "是"
		isDividend := data.HasDividendObservation && data.DividendBarrierValue != "" && targetRecord.IsDividend == "是"

		if isKnockout {
			knockoutCount++
			row := posterRow(product, today, targetObsInfo.MonthsSinceEntry, data, "knockout")
			row.AbsoluteReturn = floatPtr(data.AbsoluteReturn)
			row.DurationMonths = intPtr(targetObsInfo.MonthsSinceEntry)
			row.DividendBarrierValue = ""
			row.DividendCount = intPtr(0)
			row.CumulativeRate = floatPtr(0)
			_ = s.store.UpsertPoster(row)
			if !isDividend {
				continue
			}
		}
		if isDividend && !isKnockout {
			dividendCount++
			row := posterRow(product, today, targetObsInfo.MonthsSinceEntry, data, "dividend")
			row.AbsoluteReturn = floatPtr(0)
			row.DurationMonths = intPtr(0)
			row.DividendCount = intPtr(data.DividendCount)
			row.CumulativeRate = floatPtr(data.CumulativeDividendRate)
			_ = s.store.UpsertPoster(row)
		}
	}
	fmt.Printf("[喜报生成] 今日自动生成：敲出喜报 %d 张，派息喜报 %d 张\n", knockoutCount, dividendCount)
}
```

`observationInfoForDate`（router.go:1432）和 `posterRow`（router.go:1444）均已存在，无需新增辅助函数。节假日调整通过 `observations.DatesForMonth` → `adjustForHoliday`（已连接 `trading` 包）自动处理。

- [ ] **Step 7: 新增 `scheduledObservationEmail` 方法**

```go
func (s *Server) scheduledObservationEmail() {
    ctx := context.Background()
    products, err := s.store.QueryOngoingProducts()
    if err != nil {
        fmt.Printf("[邮件提醒] 获取产品失败: %v\n", err)
        return
    }
    codes := uniqueProductCodes(products)
    if len(codes) == 0 {
        return
    }
    priceResult := prices.FetchAll(ctx, codes)
    today := time.Now().Format("2006-01-02")
    for code, price := range priceResult.Prices {
        _ = s.store.UpsertPrice(code, today, price)
    }

    emailCfg := email.Config{
        SMTPHost:   s.cfg.SMTPHost,
        SMTPPort:   s.cfg.SMTPPort,
        SMTPSecure: s.cfg.SMTPSecure,
        SMTPUser:   s.cfg.SMTPUser,
        SMTPPass:   s.cfg.SMTPPass,
        SMTPFrom:   s.cfg.SMTPFrom,
    }
    notification := email.BuildTodayNotification(products, priceResult.Prices, today, emailCfg)
    sent, reason := email.SendObservationEmail(emailCfg, notification)
    if sent {
        fmt.Printf("[邮件提醒] 已发送今日观察提醒: %d 个产品\n", len(notification.Products))
    } else {
        fmt.Printf("[邮件提醒] 未发送: %s\n", reason)
    }
}
```

- [ ] **Step 8: 编译验证**

```sh
cd backend-go && go build ./cmd/server
```

Expected: 编译通过，无错误。若有多余未使用 import 报错，删除多余行即可。

- [ ] **Step 9: 提交**

```sh
git add backend-go/internal/app/router.go backend-go/internal/observations/calendar.go
git commit -m "feat(go): add 6 cron jobs (price update, auto posters, email) via robfig/cron"
```

---

### Task 9: 更新 `main.go` — 优雅退出 cron

**Files:**
- Modify: `backend-go/cmd/server/main.go`

- [ ] **Step 1: 重写 `main.go`**

```go
package main

import (
    "context"
    "log"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"

    "business-workbench/backend-go/internal/app"
    "business-workbench/backend-go/internal/config"
    "business-workbench/backend-go/internal/db"
)

func main() {
    cfg := config.Load()

    store, err := db.Open(cfg.DatabasePath)
    if err != nil {
        log.Fatalf("open database: %v", err)
    }
    defer store.Close()

    if err := store.InitSchema(); err != nil {
        log.Fatalf("init database schema: %v", err)
    }

    router := app.NewRouter(cfg, store)
    srv := &http.Server{Addr: ":" + cfg.Port, Handler: router}

    go func() {
        log.Printf("business-workbench-go starting on :%s", cfg.Port)
        if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Fatalf("server error: %v", err)
        }
    }()

    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit
    log.Println("shutting down...")
    if cron := app.GetCron(router); cron != nil {
        cron.Stop()
    }
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    if err := srv.Shutdown(ctx); err != nil {
        log.Fatalf("server forced shutdown: %v", err)
    }
    log.Println("server exited")
}
```

在 `router.go` 末尾添加辅助函数供 main 使用：

```go
func GetCron(router *gin.Engine) *cron.Cron {
    // Cron is managed via Server; expose through gin.Engine context is not ideal,
    // so we use a package-level variable.
    return schedulerInstance
}

var schedulerInstance *cron.Cron
```

并在 `NewRouter` 中赋值：

```go
schedulerInstance = server.Cron
```

- [ ] **Step 2: 编译验证**

```sh
cd backend-go && go build ./cmd/server
```

- [ ] **Step 3: 提交**

```sh
git add backend-go/cmd/server/main.go backend-go/internal/app/router.go
git commit -m "feat(go): graceful shutdown with cron stop"
```

---

### Task 10: 更新 `.env.example` 并执行完整测试

- [ ] **Step 1: 更新 `backend/.env.example`**

在 SMTP 注释后添加说明行（保持注释状态，仅供文档参考）：

在 `FEISHU_PUSH_WEBHOOK=...` 行后（若不存在则追加）添加：

```ini
# SMTP configuration for observation email reminders
# SMTP_HOST=smtp.example.com
# SMTP_PORT=465
# SMTP_SECURE=true
# SMTP_USER=your_email@example.com
# SMTP_PASS=your_smtp_auth_code
# SMTP_FROM=your_email@example.com
```

- [ ] **Step 2: 运行所有测试**

```sh
cd backend-go && go test ./... -v
```

Expected: 所有包测试 PASS（trading、email、retriever 新增测试 + 现有测试不受影响）。

- [ ] **Step 3: 全量编译验证**

```sh
cd backend-go && go build ./cmd/server
```

Expected: 编译通过。

- [ ] **Step 4: 启动服务验证 cron 注册**

```sh
cd backend-go && ./server 2>&1 | head -20
```

Expected: 日志中出现 `[定时任务]` 和 `[飞书推送]` 注册信息（约 1-2 秒内可见）。

- [ ] **Step 5: 最终提交**

```sh
git add -A
git commit -m "chore: update .env.example with SMTP placeholders"
```

- [ ] **Step 6: 验证 README 迁移状态更新**

更新 `backend-go/README.md` 末尾的迁移说明，注明：
- 定时任务：已完全迁移（robfig/cron）
- 交易日历：已完全迁移（2025-2026 节假日）
- 邮件通知：已完全迁移（SMTP，可选配置）
- RAG 文档检索：已完全迁移（关键词评分，自动注入 Agent system prompt）

```sh
git add backend-go/README.md
git commit -m "docs(go): update migration status as fully complete"
```
