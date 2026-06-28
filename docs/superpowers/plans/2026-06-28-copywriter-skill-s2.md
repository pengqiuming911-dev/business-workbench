# Copywriter Skill → Agent (S2 真实胜率) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 给 agent 加 `fetch_winrate` 工具，用 `chromedp` 驱动通毓终端(`terminal.tongyu-quant.com`)的"结构化产品回测"页，自动登录、填表、跑分析、读真实回测胜率；遇验证码/登录失败/站点不可达退回 `[胜率待补]` 占位；提供 `WINRATE_DRY_RUN` 用于脱离通毓测试 agent 流程。

**Architecture:** `chromedp` 起无头 Chrome，按 `references/tongyu-winrate.md` 的语义流程执行（登录→进回测页→按字段标签填表→点立即分析→读胜率）。字段定位用**按标签文本**的 XPath（参考文档明确：Vue SPA 的 ref 每次变，不能记死选择器）。纯函数 `paramToFormFields` 把产品参数映射到通毓表单字段标签（可单测）。`WINRATE_DRY_RUN=true` 时短路返回 canned 胜率。无凭证时返回 `[胜率待补]`。

**Tech Stack:** Go + `github.com/chromedp/chromedp`（新依赖，需服务器/开发机装 Chrome），`config.Config` env 凭证。

## Global Constraints

- **数字零例外锁死：** 胜率必须来自 `fetch_winrate` 的真实回测或用户手动提供，agent 绝不编造胜率数字。`fetch_winrate` 失败/无凭证/遇验证码 → 返回 `{"胜率":"[胜率待补]","reason":...}`，agent 据此请用户手动提供。`WINRATE_DRY_RUN=true` 仅用于测试（返回 canned 值，source 标 `"dry-run"`），**生产不得开 dry-run**。
- **凭证安全：** `TONGYU_USER`/`TONGYU_PASS` 只从 env 读（经 `config.Config`），绝不进仓库/skill 文件/prompt/日志。无凭证 → `[胜率待补]`，不报错外泄。
- **不绕过验证码：** 遇滑块验证码（"请完成下列验证后继续"）→ 立即退回 `[胜率待补]`，不尝试程序绕过。降低触发：持久化浏览器 profile、逐字输入、单次登录（不短时间反复重试，会触发风控）。
- **label 驱动选择器：** 通毓是 Vue SPA，ref 每次变。所有字段定位用按标签文本的 XPath（如 `//span[contains(text(),'期末障碍价')]/ancestor::*[contains(@class,'ant-form-item')]//input`），**不记死 ref/CSS**。选择器需在首次 live 运行时按真实 DOM 调（参考文档如此要求）。
- **新依赖：** S2 加 `github.com/chromedp/chromedp`（Go 模块）+ 需要 Chrome 二进制（开发机已有 `C:/Program Files/Google/Chrome/Application/chrome.exe`；生产 ECS 须装 Chrome，合 S2 前与运维 flag）。
- **测试边界（沿用 spec testing 表）：** 纯函数 `paramToFormFields` + dry-run 路径用单测（always run）；`chromedp` live 路径**人工验收**（需 `TONGYU_USER`/`TONGYU_PASS` + 能访问 `terminal.tongyu-quant.com` 的网络；本环境无法访问该站点，live e2e 由用户人工执行）。默认 `go test ./...` 须保持纯 Go/快/不依赖 Chrome——chromedp 代码只编译验证，不在默认测试里跑。
- **依赖前置：** Task 1（纯函数 + dry-run + config）完全可测；Task 2（chromedp live）编译验证 + 人工验收；Task 3（注册 + prompt）集成。

---

## File Structure

- `backend-go/internal/config/config.go` — 加 `TongyuUser`/`TongyuPass`/`WinrateDryRun`/`ChromePath` 4 个字段。
- `backend-go/internal/agent/winrate.go` — `formField` 结构、纯函数 `paramToFormFields`、`FetchWinrate`（dry-run 短路 + 调 live）、`(s *Service) fetchWinrate`。
- `backend-go/internal/agent/winrate_test.go` — `paramToFormFields` 表测试 + dry-run 测试（always run，不依赖 Chrome）。
- `backend-go/internal/agent/browser.go` — `tongyuCreds`、`newBrowserContext`、`runTongyuBacktest`（chromedp live 流程）、helpers（`loginTongyu`/`hasCaptcha`/`fillByLabel`/`clickByLabel`/`readWinrate`）。
- `backend-go/internal/agent/service.go` — Task 3：`executeTool` switch 加 `fetch_winrate` case、`toolDefinitions` 加项、`systemPrompt` 胜率句改为调 `fetch_winrate`。
- `backend-go/go.mod` / `go.sum` — Task 2：`go get github.com/chromedp/chromedp`。

---

### Task 1: config + 纯 `paramToFormFields` 映射 + dry-run `FetchWinrate` + 单测

把产品参数映射到通毓表单字段标签（纯函数，可单测）；`FetchWinrate` 在 dry-run 时短路、无凭证时占位、否则留给 Task 2 接 live。

**Files:**
- Modify: `backend-go/internal/config/config.go`
- Create: `backend-go/internal/agent/winrate.go`
- Test: `backend-go/internal/agent/winrate_test.go`

**Interfaces:**
- Consumes: `config.Config`（Task 1 新增字段）、`stringArg`（service.go:671）。
- Produces: `type formField struct { Label, Value string }`、`func paramToFormFields(params map[string]any) []formField`、`func FetchWinrate(params map[string]any, cfg config.Config) map[string]any`、`func (s *Service) fetchWinrate(args map[string]any) map[string]any`。Task 2 的 `runTongyuBacktest` 由 `FetchWinrate` 调用；Task 3 注册 `fetch_winrate` 工具。

- [ ] **Step 1: 加 config 字段**

`backend-go/internal/config/config.go`，在 `Config` struct 末尾（`SMTPFrom string` 后）加：
```go
	TongyuUser    string
	TongyuPass    string
	WinrateDryRun string
	ChromePath    string
```
在 `Load()` 的 return 字面量末尾（`SMTPFrom: ...` 行后）加：
```go
		TongyuUser:    os.Getenv("TONGYU_USER"),
		TongyuPass:    os.Getenv("TONGYU_PASS"),
		WinrateDryRun: getEnv("WINRATE_DRY_RUN", "false"),
		ChromePath:    os.Getenv("CHROME_PATH"),
```

- [ ] **Step 2: 写失败测试**

`backend-go/internal/agent/winrate_test.go`:
```go
package agent

import (
	"testing"

	"business-workbench/backend-go/internal/config"
)

func TestParamToFormFields_FullDCN(t *testing.T) {
	params := map[string]any{
		"期限":         "36",
		"锁定期":        "3",
		"期初敲出线":      "101",
		"降敲":         "0.5",
		"降落伞":        "60",
		"派息线":        "78",
		"费后派息":       "1.39",
		"保证金":        "50",
		"是否追保":       "不追保",
	}
	fields := paramToFormFields(params)
	byLabel := map[string]string{}
	for _, f := range fields {
		byLabel[f.Label] = f.Value
	}
	cases := map[string]string{
		"期限(月)":       "36",
		"锁定期(月)":      "3",
		"首次观察敲出价(%)":  "101",
		"敲出价递减步长(%)":  "0.5",
		"期末障碍价(%)":    "60",
		"派息障碍价(%)":    "78",
		"每月或有派息(%)":   "1.39",
		"保证金水平(%)":    "50",
		"是否追保":        "不追保",
	}
	for label, want := range cases {
		if got := byLabel[label]; got != want {
			t.Errorf("field %q = %q, want %q", label, got, want)
		}
	}
}

func TestParamToFormFields_OmitsEmpty(t *testing.T) {
	// 派息线/锁定期 不适用时不出现
	params := map[string]any{
		"期限":    "24",
		"降落伞":   "60",
		"费后派息":  "1.2",
		"保证金":   "100",
		"是否追保":  "不追保",
	}
	fields := paramToFormFields(params)
	for _, f := range fields {
		if f.Label == "派息障碍价(%)" {
			t.Error("派息线为空时不应出现 派息障碍价(%)")
		}
		if f.Label == "锁定期(月)" {
			t.Error("锁定期为空时不应出现 锁定期(月)")
		}
		if f.Value == "" {
			t.Errorf("field %q value 不应为空", f.Label)
		}
	}
}

func TestFetchWinrate_DryRun(t *testing.T) {
	s := &Service{cfg: config.Config{WinrateDryRun: "true"}}
	out := s.fetchWinrate(map[string]any{"标的": "中证1000", "structure_type": "DCN"})
	if got := out["胜率"]; got != "98.17%" {
		t.Errorf("dry-run 胜率 = %v, want 98.17%%", got)
	}
	if got := out["source"]; got != "dry-run" {
		t.Errorf("source = %v, want dry-run", got)
	}
}

func TestFetchWinrate_NoCreds(t *testing.T) {
	s := &Service{cfg: config.Config{WinrateDryRun: "false"}} // 无 TongyuUser/Pass
	out := s.fetchWinrate(map[string]any{"标的": "中证1000"})
	if got := out["胜率"]; got != "[胜率待补]" {
		t.Errorf("无凭证时胜率 = %v, want [胜率待补]", got)
	}
	if _, ok := out["reason"]; !ok {
		t.Error("无凭证时应带 reason")
	}
}
```

- [ ] **Step 3: 跑测试确认失败**

Run: `cd backend-go && go test ./internal/agent/ -run TestParamToFormFields -run TestFetchWinrate -v`
Expected: FAIL with `undefined: paramToFormFields` / `undefined: (s *Service).fetchWinrate`。

- [ ] **Step 4: 写实现**

`backend-go/internal/agent/winrate.go`:
```go
package agent

import (
	"fmt"
	"strings"

	"business-workbench/backend-go/internal/config"
)

// formField 是通毓回测表单的一个填写项：按标签文本定位的输入框 + 要填的值。
type formField struct {
	Label string
	Value string
}

// paramToFormFields 把产品参数(中文键)映射成通毓"结构化产品回测"表单的字段标签 + 值。
// 映射来源：skills/structured-product-copywriter/references/tongyu-winrate.md 的字段映射表。
// 空值字段省略(如派息线/锁定期不适用)。标的与结构类型不在此映射——它们是搜索框/单选，由 chromedp 流程单独处理。
func paramToFormFields(params map[string]any) []formField {
	get := func(k string) string { return strings.TrimSpace(stringArg(params, k)) }
	var out []formField
	add := func(label, key string) {
		if v := get(key); v != "" {
			out = append(out, formField{Label: label, Value: v})
		}
	}
	add("期限(月)", "期限")
	add("锁定期(月)", "锁定期")
	add("首次观察敲出价(%)", "期初敲出线")
	add("敲出价递减步长(%)", "降敲")
	add("期末障碍价(%)", "降落伞")
	add("派息障碍价(%)", "派息线")
	add("每月或有派息(%)", "费后派息")
	add("保证金水平(%)", "保证金")
	add("是否追保", "是否追保")
	return out
}

// FetchWinrate 取真实回测胜率。
// - WINRATE_DRY_RUN=true：返回 canned 98.17%（source=dry-run），仅用于测试 agent 流程，生产不开。
// - 无 TONGYU_USER/PASS：返回 [胜率待补] + reason，不外泄凭证状态。
// - 否则：调 runTongyuBacktest（Task 2）。chromedp 失败/验证码/站点不可达 → [胜率待补] + reason。
func FetchWinrate(params map[string]any, cfg config.Config) map[string]any {
	if strings.EqualFold(cfg.WinrateDryRun, "true") {
		return map[string]any{"胜率": "98.17%", "source": "dry-run"}
	}
	if cfg.TongyuUser == "" || cfg.TongyuPass == "" {
		return map[string]any{"胜率": "[胜率待补]", "reason": "未配置 TONGYU_USER/TONGYU_PASS，请手动提供胜率"}
	}
	winrate, reason, err := runTongyuBacktest(params, tongyuCreds{User: cfg.TongyuUser, Pass: cfg.TongyuPass}, cfg.ChromePath)
	if err != nil || winrate == "" {
		r := reason
		if r == "" {
			r = err.Error()
		}
		return map[string]any{"胜率": "[胜率待补]", "reason": r}
	}
	return map[string]any{"胜率": winrate, "source": "tongyu"}
}

// fetchWinrate 是 agent 工具入口：取回测胜率或 [胜率待补] 占位。
func (s *Service) fetchWinrate(args map[string]any) map[string]any {
	out := FetchWinrate(args, s.cfg)
	out["标的"] = stringArg(args, "标的")
	return out
}

// tongyuCreds 与 runTongyuBacktest 的凭证契约（Task 2 定义 runTongyuBacktest）。
type tongyuCreds struct {
	User string
	Pass string
}

// runTongyuBacktest 由 Task 2 实现。此处先返回"未实现"占位以保 Task 1 编译通过——
// Task 1 的测试只走 dry-run/无凭证分支，不触达本函数。
func runTongyuBacktest(params map[string]any, creds tongyuCreds, chromePath string) (winrate, reason string, err error) {
	return "", "winrate live flow not implemented (Task 2)", fmt.Errorf("not implemented")
}
```

- [ ] **Step 5: 跑测试确认通过**

Run: `cd backend-go && go test ./internal/agent/ -run TestParamToFormFields -run TestFetchWinrate -v`
Expected: PASS（4 个子测试全过）。

- [ ] **Step 6: 提交**

```bash
cd D:/projects/business-workbench
git add backend-go/internal/config/config.go backend-go/internal/agent/winrate.go backend-go/internal/agent/winrate_test.go
git commit -m "feat(agent): add fetch_winrate scaffolding (param mapping + dry-run + config)"
```

---

### Task 2: `browser.go` chromedp live 流程 + 接入 + 加 chromedp 依赖

用 chromedp 驱动通毓终端：登录→验证码检测→进回测页→按标签填表→点立即分析→读胜率。**label 驱动选择器**（参考文档要求），首次 live 运行需按真实 DOM 调。

**Files:**
- Create: `backend-go/internal/agent/browser.go`
- Modify: `backend-go/internal/agent/winrate.go`（替换 `runTongyuBacktest` 占位为真实现）
- Modify: `backend-go/go.mod` / `go.sum`（`go get github.com/chromedp/chromedp`）

**Interfaces:**
- Consumes: `paramToFormFields`（Task 1）、`github.com/chromedp/chromedp`、`context`、`errors`、`strings`、`time`。
- Produces: `func runTongyuBacktest(params map[string]any, creds tongyuCreds, chromePath string) (winrate, reason string, err error)` —— 真实现，替换 Task 1 占位。

- [ ] **Step 1: 加 chromedp 依赖**

Run:
```bash
cd D:/projects/business-workbench/backend-go
"D:/projects/business-workbench/.local-tools/go/bin/go" get github.com/chromedp/chromedp
```
确认 `backend-go/go.mod` 出现 `github.com/chromedp/chromedp`。

- [ ] **Step 2: 写 `browser.go`**

`backend-go/internal/agent/browser.go`:
```go
package agent

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
)

// runTongyuBacktest 驱动通毓终端取真实回测胜率。
// 流程（按 references/tongyu-winrate.md）：登录 → 验证码检测(遇则退回) → 进回测页 →
// 关快捷查询 → 选结构类型 → 按标签填数值字段 → 点立即分析 → 读胜率。
// 选择器按字段标签文本定位（Vue SPA ref 每次变，不能记死）。首次 live 运行需按真实 DOM 调选择器。
func runTongyuBacktest(params map[string]any, creds tongyuCreds, chromePath string) (winrate, reason string, err error) {
	ctx, cancel := newBrowserContext(context.Background(), chromePath)
	defer cancel()

	if err := loginTongyu(ctx, creds); err != nil {
		return "", "登录失败：" + err.Error(), err
	}
	if hasCaptcha(ctx) {
		return "", "遇滑块验证码，请在浏览器手动跑一次取胜率", errors.New("captcha")
	}

	if err := chromedp.Run(ctx,
		chromedp.Navigate("https://terminal.tongyu-quant.com/#/investmentAnalysis/InvestmentAnalysis"),
		chromedp.Sleep(2*time.Second),
	); err != nil {
		return "", "进入回测页失败：" + err.Error(), err
	}
	// 关掉可能遮挡表单的"快捷查询"对话框（若存在）
	_ = chromedp.Run(ctx, chromedp.Click("//button[contains(@class,'close')]", chromedp.BySearch))

	// 选结构类型：按 structure_type 点对应单选/多选。structure_type 由 agent 从对话判断传入（如 "DCN+降敲+降落伞"）。
	structure := stringArg(params, "structure_type")
	if err := selectStructure(ctx, structure); err != nil {
		return "", "选结构类型失败：" + err.Error(), err
	}

	// 挂钩标的：搜索框输代码/名称（标的由 agent 传入）
	if u := stringArg(params, "标的"); u != "" {
		if err := fillUnderlying(ctx, u); err != nil {
			return "", "填挂钩标的失败：" + err.Error(), err
		}
	}

	// 填数值字段（paramToFormFields 给出 label+value）
	for _, f := range paramToFormFields(params) {
		if err := fillByLabel(ctx, f.Label, f.Value); err != nil {
			return "", fmt.Sprintf("填字段 %q 失败：%v", f.Label, err), err
		}
	}

	// 点"立即分析"
	if err := chromedp.Run(ctx, chromedp.Click("//button/span[contains(text(),'立即分析')]", chromedp.BySearch)); err != nil {
		return "", "点立即分析失败：" + err.Error(), err
	}

	// 等结果加载，读胜率
	wr, err := readWinrate(ctx)
	if err != nil {
		return "", "读胜率失败：" + err.Error(), err
	}
	return wr, "", nil
}

// newBrowserContext 起无头 Chrome。chromePath 空时让 chromedp 自动找 Chrome。
// 用持久化 userDataDir 降低验证码触发概率（参考文档建议）。
func newBrowserContext(parent context.Context, chromePath string) (context.Context, context.CancelFunc) {
	opts := append([]chromedp.ExecAllocatorOption{},
		chromedp.NoFirstRun,
		chromedp.NoDefaultBrowserCheck,
		chromedp.Headless,
		chromedp.DisableGPU,
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("lang", "zh-CN"),
	)
	if chromePath != "" {
		opts = append(opts, chromedp.ExecPath(chromePath))
	}
	allocCtx, cancelAlloc := chromedp.NewExecAllocator(parent, opts...)
	ctx, cancel := chromedp.NewContext(allocCtx)
	// 合并 cancel：取消时先 ctx 再 allocCtx
	return ctx, func() { cancel(); cancelAlloc() }
}

// loginTongyu 登录通毓。用页面中央账号密码框（不是顶部"管理员登录"）。
func loginTongyu(ctx context.Context, creds tongyuCreds) error {
	if err := chromedp.Run(ctx,
		chromedp.Navigate("https://terminal.tongyu-quant.com/#/login"),
		chromedp.Sleep(2*time.Second),
	); err != nil {
		return err
	}
	// 用户名/密码输入框：按 placeholder 或 type 定位（首次 live 运行按真实 DOM 调）
	if err := chromedp.Run(ctx,
		chromedp.WaitVisible(`//input[@type='text' or @name='username' or contains(@placeholder,'账号') or contains(@placeholder,'用户')][1]`, chromedp.BySearch),
		chromedp.SendKeys(`//input[@type='text' or @name='username' or contains(@placeholder,'账号') or contains(@placeholder,'用户')][1]`, creds.User, chromedp.BySearch),
		chromedp.SendKeys(`//input[@type='password'][1]`, creds.Pass, chromedp.BySearch),
		chromedp.Click(`//button[contains(.,'登录') or contains(.,'Login')][1]`, chromedp.BySearch),
		chromedp.Sleep(2*time.Second),
	); err != nil {
		return err
	}
	// 登录成功后 URL 跳 #/；若停在 #/login 且有"用户名或密码错误"则凭证错（不重试，防风控）
	var url string
	_ = chromedp.Run(ctx, chromedp.Location(&url))
	if strings.Contains(url, "/login") {
		var body string
		_ = chromedp.Run(ctx, chromedp.Text("body", &body, chromedp.ByQuery))
		if strings.Contains(body, "用户名或密码错误") {
			return errors.New("用户名或密码错误（凭证错，不重试）")
		}
		return errors.New("登录后未跳转主页")
	}
	return nil
}

// hasCaptcha 检测滑块验证码（"请完成下列验证后继续"）。遇则 true，调用方退回 [胜率待补]。
func hasCaptcha(ctx context.Context) bool {
	var body string
	if err := chromedp.Run(ctx, chromedp.Text("body", &body, chromedp.ByQuery)); err != nil {
		return false
	}
	return strings.Contains(body, "请完成下列验证后继续")
}

// selectStructure 按 structure_type 字符串点结构单选 + 叠加条款多选。
// structure_type 形如 "DCN+降敲+降落伞"（agent 从对话判断传入）。
// 实现按标签文本点击；首次 live 运行按真实 DOM 调选择器。
func selectStructure(ctx context.Context, structure string) error {
	for _, part := range strings.Split(structure, "+") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		// 点含该结构名的 label/span（antd 单选/多选用 label 包裹）
		_ = chromedp.Run(ctx, chromedp.Click(fmt.Sprintf(`//label[contains(.,'%s')]|//span[contains(.,'%s')]`, part, part), chromedp.BySearch))
		chromedp.Sleep(300 * time.Millisecond)
	}
	return nil
}

// fillUnderlying 在"挂钩标的"搜索框输标的，选第一条结果。
func fillUnderlying(ctx context.Context, underlying string) error {
	if err := chromedp.Run(ctx,
		chromedp.Click(`//input[@placeholder[contains(.,'标的') or contains(.,'代码') or contains(.,'名称')]][1]`, chromedp.BySearch),
		chromedp.SendKeys(`//input[@placeholder[contains(.,'标的') or contains(.,'代码') or contains(.,'名称')]][1]`, underlying, chromedp.BySearch),
		chromedp.Sleep(1*time.Second),
		chromedp.Click(`//li[contains(@class,'ant-select-item')][1]`, chromedp.BySearch),
	); err != nil {
		return err
	}
	return nil
}

// fillByLabel 按字段标签文本定位 antd 输入框并填值。
// 选择器策略：找含标签文本的 span/label，向上找 ant-form-item，取其下 input。
// 首次 live 运行需按真实 DOM 结构调（参考文档要求按字段名定位当前 ref）。
func fillByLabel(ctx context.Context, label, value string) error {
	xpath := fmt.Sprintf(`//span[contains(text(),'%s')]/ancestor::*[contains(@class,'ant-form-item')]//input | //label[contains(.,'%s')]/ancestor::*[contains(@class,'ant-form-item')]//input`, label, label)
	if err := chromedp.Run(ctx,
		chromedp.WaitVisible(xpath, chromedp.BySearch),
		chromedp.Click(xpath, chromedp.BySearch),
		chromedp.Input(value, chromedp.BySearch),
	); err != nil {
		// Input 失败时回退到 SendKeys 逐字输入（参考文档：fill 不生效改逐字）
		if e2 := chromedp.Run(ctx, chromedp.SendKeys(xpath, value, chromedp.BySearch)); e2 != nil {
			return err
		}
	}
	return nil
}

// readWinrate 从结果区读胜率百分比。结果区标题含"买入一份该SNOWBALL合约"。
// 胜率形如 "98.14%"。按结果区文本正则取首个百分比。
func readWinrate(ctx context.Context) (string, error) {
	// 等结果区出现
	if err := chromedp.Run(ctx,
		chromedp.WaitVisible(`//*[contains(.,'买入一份') or contains(.,'胜率')]`, chromedp.BySearch),
		chromedp.Sleep(3*time.Second),
	); err != nil {
		return "", err
	}
	var text string
	if err := chromedp.Run(ctx, chromedp.Text(`//*[contains(.,'胜率')]`, &text, chromedp.BySearch)); err != nil {
		return "", err
	}
	// 取首个百分比
	wr := firstPercent(text)
	if wr == "" {
		return "", errors.New("结果区未找到胜率百分比")
	}
	return wr, nil
}

// firstPercent 从文本里取第一个形如 12.34% 的百分比。
func firstPercent(s string) string {
	var b strings.Builder
	seen := false
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9' || r == '.':
			b.WriteRune(r)
			seen = true
		case r == '%' && seen:
			return b.String() + "%"
		default:
			if seen {
				b.Reset()
				seen = false
			}
		}
	}
	return ""
}
```

- [ ] **Step 3: 替换 winrate.go 的占位 runTongyuBacktest**

`backend-go/internal/agent/winrate.go`：删除 Task 1 的占位 `runTongyuBacktest`（含 `return "", "winrate live flow not implemented (Task 2)", fmt.Errorf("not implemented")` 整个函数），因为 Task 2 的 `browser.go` 提供了真实现（同包同名函数）。删除后 `winrate.go` 不再需要 `fmt` import——确认并清理：若 `winrate.go` 删占位后不再用 `fmt`，从 import 区移除 `"fmt"`；`strings` 仍用（`paramToFormFields`/`FetchWinrate` 用 `strings.TrimSpace`/`EqualFold`）保留。

- [ ] **Step 4: 编译验证（不跑 live）**

Run: `cd backend-go && go build ./... && go test ./internal/agent/ -run TestParamToFormFields -run TestFetchWinrate -v`
Expected: 编译通过（chromedp 依赖拉取成功）；4 个纯函数/dry-run 测试仍 PASS。chromedp live 路径不在测试范围。

- [ ] **Step 5: 提交**

```bash
cd D:/projects/business-workbench
git add backend-go/internal/agent/browser.go backend-go/internal/agent/winrate.go backend-go/go.mod backend-go/go.sum
git commit -m "feat(agent): implement fetch_winrate chromedp live flow (label-driven, manual-acceptance)"
```

- [ ] **Step 6: 人工验收（live，由用户执行——本环境无法访问 terminal.tongyu-quant.com）**

前置：`backend-go/.env` 配 `TONGYU_USER`/`TONGYU_PASS`，且运行环境能访问 `terminal.tongyu-quant.com`。启动后端，触发：
```bash
curl -N -X POST http://localhost:3001/api/agent/chat \
  -H 'Content-Type: application/json' \
  -d '{"conversation_id":0,"message":"给中证1000 2倍DCN取胜率，降落伞60%、期初敲出线101%、降敲0.5%、派息线78%、费后派息1.39%、期限36M锁3M、保证金50%不追保"}'
```
Expected（SSE）：`tool_call: fetch_winrate` → 成功时 `{"胜率":"<N>.%","source":"tongyu"}`；遇验证码/站点不可达 → `{"胜率":"[胜率待补]","reason":...}`，agent 据此请用户手动提供。**若选择器填不进/读不到胜率**：按 `references/tongyu-winrate.md` 的"按字段名定位当前 ref"指引，在 `browser.go` 的 `fillByLabel`/`readWinrate`/`loginTongyu` 里调 XPath（这是参考文档预见到的 live 调参，不是 bug）。dry-run 验证（不需通毓）：`WINRATE_DRY_RUN=true` 启动，同 curl，应返回 `{"胜率":"98.17%","source":"dry-run"}`。

---

### Task 3: 注册 `fetch_winrate` 工具 + 更新 systemPrompt + 编译

把 `fetchWinrate` 接进 agent，并把胜率步骤从 S1 的"无工具用 [胜率待补]"改为"调 fetch_winrate"。

**Files:**
- Modify: `backend-go/internal/agent/service.go`（`executeTool` switch、`toolDefinitions`、`systemPrompt`）

**Interfaces:**
- Consumes: `(s *Service) fetchWinrate`（Task 1）。
- Produces: agent 可调 `fetch_winrate` 工具；systemPrompt 胜率步骤指向它。

- [ ] **Step 1: `executeTool` switch 加 case**

`service.go` `executeTool` switch 里（Task 4 of S1 加的 `case "calc_points":` 旁）加：
```go
	case "fetch_winrate":
		return s.fetchWinrate(args)
```

- [ ] **Step 2: `toolDefinitions` 加项**

`service.go` `toolDefinitions()` 里（`calc_points` 定义旁）加：
```go
		{
			Type: "function",
			Function: map[string]any{
				"name":        "fetch_winrate",
				"description": "通过通毓终端结构化产品回测算真实胜率。需要 TONGYU 凭证（服务端配置）；遇验证码/登录失败/站点不可达会返回 [胜率待补] 占位，届时请用户手动提供，绝不编造胜率。structure_type 由你从对话判断（如 DCN/雪球+降敲+降落伞）。标的用中文名或代码。",
				"parameters": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"structure_type": map[string]any{"type": "string", "description": "结构类型，如 DCN、雪球；含叠加条款用 + 连，如 DCN+降敲+降落伞"},
						"标的":           map[string]any{"type": "string", "description": "标的名（如 中证1000）或代码"},
						"期限":           map[string]any{"type": "string", "description": "如 36"},
						"锁定期":          map[string]any{"type": "string", "description": "如 3；无则不传"},
						"期初敲出线":        map[string]any{"type": "string", "description": "如 101"},
						"降敲":           map[string]any{"type": "string", "description": "如 0.5"},
						"降落伞":          map[string]any{"type": "string", "description": "如 60"},
						"派息线":          map[string]any{"type": "string", "description": "如 78；不适用不传"},
						"费后派息":         map[string]any{"type": "string", "description": "如 1.39"},
						"保证金":          map[string]any{"type": "string", "description": "如 50"},
						"是否追保":         map[string]any{"type": "string", "description": "不追保/追保"},
					},
					"required": []string{"structure_type", "标的", "期限", "期初敲出线", "降落伞", "费后派息", "保证金"},
				},
			},
		},
```

- [ ] **Step 3: 更新 systemPrompt 胜率句**

`service.go` `systemPrompt` 常量里，把 S1 写的胜率句：
`胜率步骤：当前阶段无自动获取工具，用 [胜率待补] 占位并请用户手动提供，不要编一个胜率数字。`
改为：
`胜率步骤：调用 fetch_winrate 取真实回测胜率（structure_type 由你判断传入）；若 fetch_winrate 返回 [胜率待补]（无凭证/遇验证码/站点不可达），如实告知并请用户手动提供，绝不编一个胜率数字。`

（保持单 const 字符串，仅替换这一句。）

- [ ] **Step 4: 编译 + 全 agent 测试**

Run: `cd backend-go && go build ./... && go test ./internal/agent/ -v`
Expected: 编译通过；全部测试 PASS（含 Task 1 的 winrate 测试 + S1 的测试）。

- [ ] **Step 5: 人工验收（dry-run，本环境可跑）**

`WINRATE_DRY_RUN=true` 启动后端，curl 触发文案对话（同 S1 Task 4 的文案请求）。Expected：SSE 出现 `tool_call: fetch_winrate`，结果 `{"胜率":"98.17%","source":"dry-run"}`，agent 文案里胜率用 98.17%（非编造）。live 通毓验收见 Task 2 Step 6（用户执行）。

- [ ] **Step 6: 提交**

```bash
cd D:/projects/business-workbench
git add backend-go/internal/agent/service.go
git commit -m "feat(agent): register fetch_winrate tool + point prompt win-rate step at it"
```

---

## Self-Review

**1. Spec coverage:**
- `fetch_winrate` chromedp 通毓流程 → Task 2。✅
- 凭证 env（TONGYU_USER/PASS）经 config → Task 1 config。✅
- 验证码/登录失败/站点不可达 → [胜率待补] 占位 → Task 2（hasCaptcha/loginTongyu 错误）+ Task 1 FetchWinrate 映射。✅
- WINRATE_DRY_RUN → Task 1 FetchWinrate dry-run 短路 + Task 3 dry-run 验收。✅
- label 驱动选择器（参考文档要求）→ Task 2 fillByLabel/selectStructure 用按标签文本 XPath。✅
- 纯函数 paramToFormFields 单测 → Task 1。✅
- 默认 go test 不依赖 Chrome → chromedp live 路径不在测试（只编译），Task 1 测试只走 dry-run/无凭证/纯映射。✅
- 注册工具 + prompt 更新 → Task 3。✅
- 生产 ECS 装 Chrome（运维）→ Global Constraints 标注，合 S2 前 flag（非代码任务）。✅

**2. Placeholder scan:** 无 TBD/TODO。Task 2 的 `runTongyuBacktest` 是真 chromedp 实现（非占位），选择器 label 驱动 + 明示"首次 live 运行按真实 DOM 调"（这是参考文档预见的要求，非 plan 占位）。Task 1 的占位 `runTongyuBacktest` 在 Task 2 Step 3 明确删除替换。config/toolDefinitions/prompt 全具体。

**3. Type consistency:** `formField{Label,Value string}` 在 Task 1 定义、Task 2 `paramToFormFields` 调用一致。`paramToFormFields(params map[string]any) []formField` 在 Task 1 定义、Task 2 `runTongyuBacktest` 调用一致。`FetchWinrate(params map[string]any, cfg config.Config) map[string]any` 在 Task 1 定义、`(s *Service) fetchWinrate` 调用一致。`tongyuCreds{User,Pass string}` 在 Task 1 定义、Task 2 `runTongyuBacktest`/`loginTongyu` 调用一致。`runTongyuBacktest(params, creds, chromePath) (winrate, reason string, err error)` 在 Task 1 占位与 Task 2 真实现签名一致。`(s *Service) fetchWinrate(args)` 在 Task 1 定义、Task 3 executeTool switch 调用一致。工具名 `fetch_winrate` 在 Task 3 switch 与 toolDefinitions 一致。config 字段 `TongyuUser`/`TongyuPass`/`WinrateDryRun`/`ChromePath` 在 Task 1 config 定义、`FetchWinrate` 调用一致。
