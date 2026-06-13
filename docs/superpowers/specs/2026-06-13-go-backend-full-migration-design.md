# Go 后端迁移完成设计

## 背景

当前 `backend-go/` 已完成约 85-90% 的 API 迁移，但有 4 项功能仍在 Node.js 中运行：

1. 6 个定时任务（价格更新、自动喜报生成、邮件提醒）
2. 交易日历 / 节假日调整（`adjustForHoliday` 是空操作）
3. 邮件通知服务（SMTP 发送观察提醒）
4. RAG 文档检索（Agent 请求时自动注入相关文档到 system prompt）

本次迁移目标是让 Go 后端完全取代 Node.js 后端。

---

## 新增包结构

```text
backend-go/internal/
├── trading/
│   └── calendar.go       # IsTradingDay, IsHoliday, AdjustToNearestTradingDay
├── email/
│   └── sender.go         # SendObservationEmail, renderText, renderHtml
└── retriever/
    └── retriever.go      # SearchDocs, BuildDocContext
```

---

## 1. 交易日历 (`internal/trading/calendar.go`)

### 来源

`backend/services/tradingCalendar.js`，静态节假日列表（2025–2026），共 57 行。

### 实现

```go
package trading

var holidays2025 = []string{"2025-01-01", "2025-01-28", /* ... */}
var holidays2026 = []string{"2026-01-01", /* ... */}

var holidaySet = map[string]bool{...} // 初始化时合并两个列表

func IsWeekend(t time.Time) bool {
    return t.Weekday() == time.Saturday || t.Weekday() == time.Sunday
}

func IsHoliday(dateStr string) bool {
    return holidaySet[dateStr]
}

func IsTradingDay(dateStr string) bool {
    t, _ := time.Parse("2006-01-02", dateStr)
    return !IsWeekend(t) && !IsHoliday(dateStr)
}

// direction: "advance" 提前，"postpone" 顺延
func AdjustToNearestTradingDay(dateStr, direction string) string {
    t, _ := time.Parse("2006-01-02", dateStr)
    for !IsTradingDay(t.Format("2006-01-02")) {
        if direction == "advance" {
            t = t.AddDate(0, 0, -1)
        } else {
            t = t.AddDate(0, 0, 1)
        }
    }
    return t.Format("2006-01-02")
}
```

### 修改 `internal/observations/calendar.go`

替换现有空操作：

```go
// 移除旧的占位实现，改为：
func adjustForHoliday(dateStr string, holidayAdjust string) string {
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

---

## 2. 邮件通知 (`internal/email/sender.go`)

### 来源

`backend/services/observationNotificationService.js`，使用 nodemailer，181 行。

### 实现要点

- 使用标准库 `net/smtp` + `encoding/base64` 构建 MIME 邮件（纯文本 + HTML）
- 不引入第三方邮件库，与 Go 标准库保持一致
- 当 SMTP 未配置时直接返回 `{sent: false, reason: "smtp-not-configured"}`

### 主要导出函数

```go
type EmailConfig struct {
    SMTPHost string
    SMTPPort string
    Secure   bool
    User     string
    Pass     string
    From     string
}

type Notification struct {
    Recipient string
    Subject   string
    Text      string
    HTML      string
    Products  []Product
}

func BuildTodayNotification(products []model.Product, prices map[string]float64, today string) *Notification

func SendObservationEmail(cfg EmailConfig, n *Notification) (sent bool, reason string, err error)
```

### 渲染逻辑

- `renderText`: 与 Node.js 完全对齐的纯文本格式
- `renderHtml`: 与 Node.js 完全对齐的 HTML 表格格式
- `escapeHtml`: 标准 `&`, `<`, `>`, `"`, `'` 转义

---

## 3. RAG 文档检索 (`internal/retriever/retriever.go`)

### 来源

`backend/services/documentRetriever.js`，49 行，关键词打分 + 全文匹配。

### 实现要点

```go
func SearchDocs(docs []model.ProductDoc, query string, limit int) []ScoredDoc
func BuildDocContext(scored []ScoredDoc) string
```

`SearchDocs`：将 query 拆分为空格分隔的关键词，在 `doc_name + parent_path + raw_content` 中统计出现次数，按分数排序，取 top N。

`BuildDocContext`：将得分文档格式化为系统提示后缀文本块，与 Node.js 完全对齐：

```text
以下是与用户问题相关的文档资料，请参考这些文档回答问题：

[文档1] 文件名 (路径)
内容
---
[文档2] ...
```

### 集成到 Agent (`internal/agent/service.go`)

修改 `buildMessages` 签名，接受 context 参数：

```go
func buildMessages(history []model.AgentMessage, userMessage string, docContext string) []chatMessage {
    prompt := systemPrompt
    if docContext != "" {
        prompt += "\n" + docContext
    }
    messages := []chatMessage{{Role: "system", Content: prompt}}
    // ... 其余不变
}
```

在 `StreamChat` 中调用检索：

```go
allDocs, _ := s.store.GetAllProductDocs()
scored := retriever.SearchDocs(allDocs, userMessage, 5)
docContext := retriever.BuildDocContext(scored)
messages := buildMessages(history, userMessage, docContext)
```

### 数据存储改动

需要在 `repository.go` 新增 `GetAllProductDocs() []model.ProductDoc`，一次性加载所有 `product_docs` 行供检索使用（Node.js 也是全量加载，当前数据量可控）。

---

## 4. 定时任务 (`internal/app/router.go`)

### 依赖引入

`go.mod` 新增：

```text
github.com/robfig/cron/v3 v3.0.1
```

### 现有飞书推送迁移

用 `robfig/cron` 替换现有 ticker：

```go
// 旧：在 startScheduler 里 1 分钟 ticker 手动判断时间
// 新：每分钟触发（cron 表达式 "* * * * *"），在 handle 内判断
pushCron := cron.New(cron.WithLocation(location))
pushCron.AddFunc("* * * * *", func() { s.handleFeishuPushCron() })
pushCron.Start()
// handleFeishuPushCron: 每分钟检查 push_config，匹配时间才执行
```

### 新增 6 个定时任务

```go
s.Cron.AddFunc("30 11 * * 1-5", s.scheduledPriceUpdate)
s.Cron.AddFunc("0 15 * * 1-5",  s.scheduledPriceUpdate)
s.Cron.AddFunc("30 15 * * 1-5", s.scheduledPriceUpdate)
s.Cron.AddFunc("5 15 * * 1-5",  s.generateAutoPosters)
s.Cron.AddFunc("0 10 * * *",    s.scheduledObservationEmail)
s.Cron.AddFunc("10 15 * * *",   s.scheduledObservationEmail)
```

### 新增 Handler 方法

```go
func (s *Server) scheduledPriceUpdate()
func (s *Server) generateAutoPosters()
func (s *Server) scheduledObservationEmail()
func (s *Server) refreshTodayObservations() (products, prices, updatedObs, failed)
func (s *Server) handleFeishuPushCron() // 每分钟触发，内部判断时间
```

#### `refreshTodayObservations` 逻辑

1. 获取所有进行中的产品
2. 提取所有标的代码（去重）
3. 批量从东方财富获取行情
4. 更新 `price_cache`
5. 对每个今日有观察日的产品调用 `EvaluateObservation`，写入 `observations` 表
6. 返回更新的记录数，供 price/email/poster 任务复用

#### `generateAutoPosters` 逻辑

与 Node.js `index.js` 的 `generateAutoPosters()` 完全对齐：

1. 遍历所有进行中产品
2. 判断今日是否在观察日列表中
3. 查询该产品今日是否有观察记录（`observations` 表，由 `refreshTodayObservations` 写入）
4. 调用 `generatePosterData` 计算喜报数据
5. 若敲出则插入 `knockout` 类型海报
6. 若派息则插入 `dividend` 类型海报
7. 日志输出今日生成数量

#### `scheduledObservationEmail` 逻辑

1. 调用 `refreshTodayObservations`
2. 构建 `email.BuildTodayNotification`
3. 调用 `email.SendObservationEmail`
4. 日志记录发送状态

---

## 5. Config 改动 (`internal/config/config.go`)

新增以下字段（均为可选，默认空字符串）：

```go
SMTPHost  string  // SMTP_HOST
SMTPPort  string  // SMTP_PORT (default "465")
SMTPUser  string  // SMTP_USER
SMTPPass  string  // SMTP_PASS
SMTPFrom  string  // SMTP_FROM (default SMTP_USER)
SMTPSecure string // SMTP_SECURE (default "true" if port=465)
```

---

## 6. `main.go` / `NewRouter` 改动

`Server` struct 新增字段：

```go
type Server struct {
    cfg      *config.Config
    store    *db.Store
    agent    *agent.Service
    feishu   *feishu.Client
    prices   *prices.Client
    Cron     *cron.Cron
}
```

`NewRouter` 中：

```go
s.Cron = cron.New(cron.WithLocation(location))
// 注册所有定时任务
// 启动 cron
```

`main.go` 中增加优雅退出：

```go
<-quit
s.Cron.Stop()
// 关闭 DB 等
```

---

## 7. `go.mod` 依赖变更

新增：

```text
github.com/robfig/cron/v3 v3.0.1
```

其余保持不变（SMTP 使用标准库，无需额外依赖）。

---

## 8. 边界情况与兼容性

- **节假日数据更新**：2026 年末需手动更新 `trading/calendar.go`，与 Node.js 一致
- **邮件配置缺失**：优雅降级，仅输出日志，不影响其他功能
- **RAG 文档为空**：`BuildDocContext` 返回空字符串，system prompt 不追加任何内容
- **定时任务并发安全**：所有定时任务均为短操作，无需互斥；`scheduledPriceUpdate` 可能在 cron 表达式重复时间点重复执行，用 `lastRunKey` 去重（与现有飞书推送同样）
- **时区**：所有 cron 任务使用 `robfig/cron.WithLocation(location)` 统一时区
- **旧 Node 后端**：迁移完成后 `backend/` 保留为兼容数据目录（`.env`、`data.sqlite`、静态资源），Node 代码不再需要运行

---

## 测试计划

1. **单元测试**：`trading/calendar.go` - IsTradingDay, AdjustToNearestTradingDay
2. **单元测试**：`email/sender.go` - renderText, renderHtml, escapeHtml
3. **单元测试**：`retriever/retriever.go` - SearchDocs 评分, BuildDocContext 格式
4. **集成验证**：手动触发 `scheduledPriceUpdate` 确认行情更新写入
5. **集成验证**：手动触发 `scheduledObservationEmail`，SMTP 未配置时应返回 `smtp-not-configured`
6. **集成验证**：启动服务后等 1 分钟确认所有 cron 正常注册（日志输出）
