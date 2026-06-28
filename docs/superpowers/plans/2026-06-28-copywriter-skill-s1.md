# Copywriter Skill → Agent (S1 文案核心) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 把 `structured-product-copywriter` skill 集成进 DeepSeek agent 的第一阶段（S1/PR1）：skill 文件嵌入二进制、按需加载，agent 能在对话里按 skill 步骤收参、取实时点位、算绝对点位、产出长版+短版推介文案，客户可见数字只来自工具。胜率在 S1 走 `[胜率待补]` 占位（S2 另开计划接真实胜率）。

**Architecture:** skill 的 `SKILL.md` + `references/*.md` 纳入仓库、用 `//go:embed` 嵌进 `agent` 包；两个 meta-tool `load_skill`/`get_skill_reference` 按需返回原文。新增纯 Go 工具 `fetch_quote`（3 源兜底取价，port 自 `fetch_quote.py`）与 `calc_points`（机械点位换算，完整性闸门）。4 个工具按既有 `executeTool` switch + `toolDefinitions` 模式注册，`systemPrompt` 追加文案指令。S1 无新二进制依赖、无运维变更。S2（chromedp 真实胜率）、S3（Word 材料）各自后续 plan。

**Tech Stack:** Go（Gin agent `backend-go/internal/agent`）、`net/http`+`httptest`、`embed.FS`、DeepSeek tool-use。

## Global Constraints

- **数字零例外锁死：** 客户可见的点位/胜率只来自工具（`fetch_quote` 取价、`calc_points` 算点位），agent 绝不在对话里编造点位/胜率、绝不自己做小数换算。判断性内容（安全垫对比、卖点组织）留给 LLM。
- **skill 原文不改：** `SKILL.md` 与 `references/*.md` 原样嵌入，**不修改其内容**。S1 阶段 `fetch_winrate` 尚未上线，由 `systemPrompt` 指令覆盖胜率步骤为 `[胜率待补]`（S2 plan 再接真实工具并更新指令）。
- **纯 Go：** S1 不引新二进制依赖、不引新 Go 第三方包（仅 stdlib）。
- **参数来自用户对话：** 按 skill 步骤 1 对话式收齐 10 项参数，**不从 DB 拉取**。
- **依赖前置：** Task 1/2/3 是纯函数 + 单测（独立可测）；Task 4 是集成 glue（注册 + prompt + 人工验收）。
- **测试模式：** 后端 `cd backend-go && go test ./internal/agent/ -run <Test> -v`。纯函数表驱动；`fetch_quote` 用 `httptest.Server` mock 三源。

---

## File Structure

- `backend-go/internal/agent/skills/structured-product-copywriter/SKILL.md` — 从 `~/.claude/skills/structured-product-copywriter/SKILL.md` 原样复制（嵌入）。
- `backend-go/internal/agent/skills/structured-product-copywriter/references/*.md` — 同上原样复制 4 份参考文档（`amac-manager.md` / `docx-template.md` / `product-position-card.md` / `tongyu-winrate.md`）。S1 不用但前置嵌入，S2/S3 复用。
- `backend-go/internal/agent/calc.go` — 纯函数 `CalcPoints` + `parsePercent` + `approxPoint` + 方法 `(s *Service) calcPoints`。
- `backend-go/internal/agent/calc_test.go` — 测 `CalcPoints`。
- `backend-go/internal/agent/skill_loader.go` — `//go:embed` + 纯函数 `LoadSkillContent`/`GetSkillReference` + 方法 `(s *Service) loadSkill`/`getSkillReference`。
- `backend-go/internal/agent/skill_loader_test.go` — 测两个纯函数。
- `backend-go/internal/agent/quote.go` — 纯函数 `ResolveCode`/`FetchQuote` + 三源 + 方法 `(s *Service) fetchQuote`。
- `backend-go/internal/agent/quote_test.go` — 测 `ResolveCode` + `FetchQuote`（httptest mock 三源 + 兜底）。
- `backend-go/internal/agent/service.go` — Task 4：`executeTool` switch 加 4 case、`toolDefinitions` 加 4 项、`systemPrompt` 追加文案指令。

---

### Task 1: `calc_points` 纯函数 + 单测

机械点位换算的完整性闸门：把降落伞/敲出/派息线百分比 + 当前点位 → 绝对点位 + 口语化「约点」。纯函数，TDD。

**Files:**
- Create: `backend-go/internal/agent/calc.go`
- Test: `backend-go/internal/agent/calc_test.go`

**Interfaces:**
- Consumes: `stringArg(args,key)`（service.go:671，同包）、`strconv`、`math`。
- Produces: `func CalcPoints(parachute, knockoutLine, dividendLine string, currentPrice float64) map[string]any` —— Task 4 的 `(s *Service) calcPoints` 调用它。返回 map 键：`parachute_point`(float64)/`parachute_point_approx`(string)/`knockout_point`/`knockout_point_approx`/`dividend_point`/`dividend_point_approx`（派息线为空时省略后两者）。

- [ ] **Step 1: 写失败测试**

`backend-go/internal/agent/calc_test.go`:
```go
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
```

- [ ] **Step 2: 跑测试确认失败**

Run: `cd backend-go && go test ./internal/agent/ -run TestCalcPoints -v`
Expected: FAIL with `undefined: CalcPoints`。

- [ ] **Step 3: 写最小实现**

`backend-go/internal/agent/calc.go`:
```go
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
```

- [ ] **Step 4: 跑测试确认通过**

Run: `cd backend-go && go test ./internal/agent/ -run TestCalcPoints -v`
Expected: PASS（5 个子测试全过）。

- [ ] **Step 5: 提交**

```bash
cd D:/projects/business-workbench
git add backend-go/internal/agent/calc.go backend-go/internal/agent/calc_test.go
git commit -m "feat(agent): add CalcPoints for structured-product pitch point calc"
```

---

### Task 2: skill 文件嵌入 + `load_skill`/`get_skill_reference` + 单测

把 skill 的 `SKILL.md` + 4 份 references 纳入仓库、`//go:embed` 嵌入，提供按需加载。纯函数 + TDD。

**Files:**
- Create: `backend-go/internal/agent/skills/structured-product-copywriter/SKILL.md`（从 `~/.claude/skills/structured-product-copywriter/SKILL.md` 原样复制）
- Create: `backend-go/internal/agent/skills/structured-product-copywriter/references/amac-manager.md`（原样复制）
- Create: `backend-go/internal/agent/skills/structured-product-copywriter/references/docx-template.md`（原样复制）
- Create: `backend-go/internal/agent/skills/structured-product-copywriter/references/product-position-card.md`（原样复制）
- Create: `backend-go/internal/agent/skills/structured-product-copywriter/references/tongyu-winrate.md`（原样复制）
- Create: `backend-go/internal/agent/skill_loader.go`
- Test: `backend-go/internal/agent/skill_loader_test.go`

**Interfaces:**
- Consumes: `embed`、`fmt`、`stringArg`（service.go:671）。
- Produces: `func LoadSkillContent(name string) (string, error)`、`func GetSkillReference(skillName, refName string) (string, error)`、方法 `(s *Service) loadSkill(args)` / `(s *Service) getSkillReference(args)`。Task 4 注册这两个工具。

- [ ] **Step 1: 复制 skill 文件进仓库**

```bash
cd D:/projects/business-workbench
SRC="$HOME/.claude/skills/structured-product-copywriter"
DST="backend-go/internal/agent/skills/structured-product-copywriter"
mkdir -p "$DST/references"
cp "$SRC/SKILL.md" "$DST/SKILL.md"
cp "$SRC/references/amac-manager.md" "$DST/references/amac-manager.md"
cp "$SRC/references/docx-template.md" "$DST/references/docx-template.md"
cp "$SRC/references/product-position-card.md" "$DST/references/product-position-card.md"
cp "$SRC/references/tongyu-winrate.md" "$DST/references/tongyu-winrate.md"
```
（Windows bash：`$HOME` 即 `C:/Users/WIKO`。若 `cp` 不可用，用 `cmd //c copy` 等价命令；关键是 5 个文件字节级原样到位。）

- [ ] **Step 2: 写失败测试**

`backend-go/internal/agent/skill_loader_test.go`:
```go
package agent

import (
	"strings"
	"testing"
)

func TestLoadSkillContent_Copywriter(t *testing.T) {
	content, err := LoadSkillContent("structured-product-copywriter")
	if err != nil {
		t.Fatalf("LoadSkillContent: %v", err)
	}
	if !strings.Contains(content, "结构化产品推介文案") {
		t.Error("缺 skill 标题「结构化产品推介文案」")
	}
	if !strings.Contains(content, "降落伞") {
		t.Error("缺 10 项参数表内容（降落伞）")
	}
}

func TestLoadSkillContent_NotFound(t *testing.T) {
	if _, err := LoadSkillContent("nonexistent-skill"); err == nil {
		t.Error("未知名应返回 error")
	}
}

func TestGetSkillReference_TongyuWinrate(t *testing.T) {
	content, err := GetSkillReference("structured-product-copywriter", "tongyu-winrate")
	if err != nil {
		t.Fatalf("GetSkillReference: %v", err)
	}
	if !strings.Contains(content, "胜率") {
		t.Error("缺「胜率」内容")
	}
	if !strings.Contains(content, "通毓") {
		t.Error("缺「通毓」内容")
	}
}

func TestGetSkillReference_NotFound(t *testing.T) {
	if _, err := GetSkillReference("structured-product-copywriter", "nope"); err == nil {
		t.Error("未知名应返回 error")
	}
}
```

- [ ] **Step 3: 跑测试确认失败**

Run: `cd backend-go && go test ./internal/agent/ -run TestLoadSkillContent -run TestGetSkillReference -v`
Expected: FAIL with `undefined: LoadSkillContent`。

- [ ] **Step 4: 写最小实现**

`backend-go/internal/agent/skill_loader.go`:
```go
package agent

import (
	"embed"
	"fmt"
)

//go:embed skills/structured-product-copywriter
var skillFS embed.FS

const copywriterSkillDir = "skills/structured-product-copywriter"

// LoadSkillContent 返回指定 skill 的 verbatim SKILL.md。
// 当前仅嵌入 "structured-product-copywriter"。
func LoadSkillContent(name string) (string, error) {
	path := fmt.Sprintf("%s/%s/SKILL.md", copywriterSkillDir, name)
	b, err := skillFS.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("skill %q not found: %w", name, err)
	}
	return string(b), nil
}

// GetSkillReference 返回某 skill 下 references/ 里的参考文档（refName 不带 .md）。
func GetSkillReference(skillName, refName string) (string, error) {
	path := fmt.Sprintf("%s/%s/references/%s.md", copywriterSkillDir, skillName, refName)
	b, err := skillFS.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("reference %q of skill %q not found: %w", refName, skillName, err)
	}
	return string(b), nil
}

// loadSkill 是 agent 工具入口：返回 skill 原文供 agent 按步执行。
func (s *Service) loadSkill(args map[string]any) map[string]any {
	name := stringArg(args, "name")
	if name == "" {
		name = "structured-product-copywriter"
	}
	content, err := LoadSkillContent(name)
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	return map[string]any{"name": name, "content": content}
}

// getSkillReference 是 agent 工具入口：按需返回重步骤参考文档。
func (s *Service) getSkillReference(args map[string]any) map[string]any {
	skillName := stringArg(args, "skill")
	if skillName == "" {
		skillName = "structured-product-copywriter"
	}
	refName := stringArg(args, "name")
	if refName == "" {
		return map[string]any{"error": "name is required"}
	}
	content, err := GetSkillReference(skillName, refName)
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	return map[string]any{"skill": skillName, "name": refName, "content": content}
}
```

- [ ] **Step 5: 跑测试确认通过**

Run: `cd backend-go && go test ./internal/agent/ -run TestLoadSkillContent -run TestGetSkillReference -v`
Expected: PASS。

- [ ] **Step 6: 提交**

```bash
cd D:/projects/business-workbench
git add backend-go/internal/agent/skills backend-go/internal/agent/skill_loader.go backend-go/internal/agent/skill_loader_test.go
git commit -m "feat(agent): embed structured-product-copywriter skill + load_skill/get_skill_reference"
```

---

### Task 3: `fetch_quote` 纯 Go 取价 + 单测

port 自 skill 的 `fetch_quote.py`：3 源兜底（腾讯→新浪→东财），首个有效价胜出。纯函数 + `httptest` mock 测兜底链。

**Files:**
- Create: `backend-go/internal/agent/quote.go`
- Test: `backend-go/internal/agent/quote_test.go`

**Interfaces:**
- Consumes: `net/http`、`io`、`encoding/json`、`strconv`、`strings`、`time`、`stringArg`。
- Produces: `func ResolveCode(arg string) string`、`func FetchQuote(code string) (name string, price float64, source string, err error)`、三源包级变量 `tencentBase`/`sinaBase`/`eastmoneyBase`（测试覆盖）、`quoteHTTPClient`、方法 `(s *Service) fetchQuote(args)`。Task 4 注册 `fetch_quote` 工具。

- [ ] **Step 1: 写失败测试**

`backend-go/internal/agent/quote_test.go`:
```go
package agent

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestResolveCode_Name(t *testing.T) {
	if got := ResolveCode("中证1000"); got != "sh000852" {
		t.Errorf("got %q, want sh000852", got)
	}
	if got := ResolveCode("沪深300"); got != "sh000300" {
		t.Errorf("got %q, want sh000300", got)
	}
}

func TestResolveCode_RawCode(t *testing.T) {
	if got := ResolveCode("sh000852"); got != "sh000852" {
		t.Errorf("got %q, want sh000852", got)
	}
}

func TestResolveCode_Unknown(t *testing.T) {
	if got := ResolveCode("不存在的指数"); got != "" {
		t.Errorf("got %q, want empty", got)
	}
}

func TestFetchQuote_TencentOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `v_sh000852="1~000852~中证1000~8810.34~8790.50~..."`)
	}))
	t.Cleanup(srv.Close)
	orig := tencentBase
	tencentBase = srv.URL
	t.Cleanup(func() { tencentBase = orig })

	name, price, src, err := FetchQuote("sh000852")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if price != 8810.34 {
		t.Errorf("price = %v, want 8810.34", price)
	}
	if src != "tencent" {
		t.Errorf("source = %q, want tencent", src)
	}
	if name != "中证1000" {
		t.Errorf("name = %q, want 中证1000", name)
	}
}

func TestFetchQuote_FallbackToSina(t *testing.T) {
	// 腾讯返回空 body（无引号段）→ 失败；新浪返回有效 → 胜出
	tencent := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "")
	}))
	sina := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `hq_str_sh000852="中证1000,8800,8790,8810.34,..."`)
	}))
	t.Cleanup(tencent.Close)
	t.Cleanup(sina.Close)
	ot, osi := tencentBase, sinaBase
	tencentBase, sinaBase = tencent.URL, sina.URL
	t.Cleanup(func() { tencentBase, sinaBase = ot, osi })

	_, price, src, err := FetchQuote("sh000852")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if price != 8810.34 {
		t.Errorf("price = %v, want 8810.34", price)
	}
	if src != "sina" {
		t.Errorf("source = %q, want sina", src)
	}
}

func TestFetchQuote_AllFail(t *testing.T) {
	tencent := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(500) }))
	sina := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(500) }))
	em := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(500) }))
	t.Cleanup(tencent.Close)
	t.Cleanup(sina.Close)
	t.Cleanup(em.Close)
	ot, osi, oe := tencentBase, sinaBase, eastmoneyBase
	tencentBase, sinaBase, eastmoneyBase = tencent.URL, sina.URL, em.URL
	t.Cleanup(func() { tencentBase, sinaBase, eastmoneyBase = ot, osi, oe })

	_, _, _, err := FetchQuote("sh000852")
	if err == nil {
		t.Fatal("三源全败应返回 error")
	}
}
```

- [ ] **Step 2: 跑测试确认失败**

Run: `cd backend-go && go test ./internal/agent/ -run TestResolveCode -run TestFetchQuote -v`
Expected: FAIL with `undefined: ResolveCode` / `undefined: FetchQuote` / `undefined: tencentBase`。

- [ ] **Step 3: 写最小实现**

`backend-go/internal/agent/quote.go`:
```go
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
	"中证1000":    "sh000852",
	"中证500":     "sh000905",
	"沪深300":     "sh000300",
	"沪深300指数":  "sh000300",
	"上证指数":     "sh000001",
	"上证50":      "sh000016",
	"创业板指":     "sz399006",
	"创业板指数":   "sz399006",
	"科创50":      "sh000688",
	"中证A500":    "sh932000",
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
	req, _ := http.NewRequest("GET", fmt.Sprintf("%s/q=%s", base, code), nil)
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
	name := parts[1]
	price, err := strconv.ParseFloat(parts[3], 64)
	if err != nil {
		return name, 0, fmt.Errorf("tencent: 价格非法 %q", parts[3])
	}
	return name, price, nil
}

// fetchFromSina: hq_str_sh000852="中证1000,开盘,昨收,最新,..."  parts[3]=最新价。需 Referer。
func fetchFromSina(base, code string) (string, float64, error) {
	req, _ := http.NewRequest("GET", fmt.Sprintf("%s/list=%s", base, code), nil)
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
	req, _ := http.NewRequest("GET", fmt.Sprintf("%s/api/qt/stock/get?secid=%s&fields=f43,f58", base, secid), nil)
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
	var lastErr error
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
```

- [ ] **Step 4: 跑测试确认通过**

Run: `cd backend-go && go test ./internal/agent/ -run TestResolveCode -run TestFetchQuote -v`
Expected: PASS（7 个子测试全过）。

- [ ] **Step 5: 提交**

```bash
cd D:/projects/business-workbench
git add backend-go/internal/agent/quote.go backend-go/internal/agent/quote_test.go
git commit -m "feat(agent): add fetch_quote (3-source live quote, port of fetch_quote.py)"
```

---

### Task 4: 注册 4 个工具 + `systemPrompt` 文案指令 + 编译 + 人工验收

把 Task 1/2/3 的方法接入 agent：`executeTool` switch 加 4 case、`toolDefinitions` 加 4 项、`systemPrompt` 追加文案指令。集成 glue，人工验收（需 DeepSeek）。

**Files:**
- Modify: `backend-go/internal/agent/service.go`（`executeTool` switch 约 line 242-275、`toolDefinitions` 约 line 900+、`systemPrompt` 约 line 25）

**Interfaces:**
- Consumes: `(s *Service) loadSkill` / `getSkillReference` / `fetchQuote` / `calcPoints`（Task 1/2/3 已定义）。
- Produces: agent 可在对话中调用 `load_skill` / `get_skill_reference` / `fetch_quote` / `calc_points` 四个工具，按 skill 步骤产出文案。

- [ ] **Step 1: `executeTool` switch 加 4 case**

`service.go` `executeTool` 的 switch（约 line 242-275），在 `case "get_activity_logs":` 之前（或任意现有 case 旁）追加：
```go
	case "load_skill":
		return s.loadSkill(args)
	case "get_skill_reference":
		return s.getSkillReference(args)
	case "fetch_quote":
		return s.fetchQuote(args)
	case "calc_points":
		return s.calcPoints(args)
```

- [ ] **Step 2: `toolDefinitions` 加 4 项**

`service.go` `toolDefinitions()` 的 slice 里（约 line 900+，挨着 `get_posters`/`generate_poster` 旁）追加：
```go
		{
			Type: "function",
			Function: map[string]any{
				"name":        "load_skill",
				"description": "加载结构化产品推介文案生成 skill 的完整工作流（SKILL.md 原文）。当用户想要生成结构化产品推介文案/材料（雪球/降敲雪球/DCN/FCN/限亏雪球等）时，先调用本工具，再严格按返回的工作流步骤执行。",
				"parameters": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"name": map[string]any{"type": "string", "description": "skill 名，默认 structured-product-copywriter"},
					},
				},
			},
		},
		{
			Type: "function",
			Function: map[string]any{
				"name":        "get_skill_reference",
				"description": "按需获取 skill 的某份重步骤参考文档（如 tongyu-winrate 通毓胜率流程、amac-manager AMAC 公示、product-position-card 产品点位卡、docx-template Word 模板）。走到该步骤时再调。",
				"parameters": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"skill": map[string]any{"type": "string", "description": "skill 名，默认 structured-product-copywriter"},
						"name":  map[string]any{"type": "string", "description": "参考文档名（不带 .md），如 tongyu-winrate"},
					},
					"required": []string{"name"},
				},
			},
		},
		{
			Type: "function",
			Function: map[string]any{
				"name":        "fetch_quote",
				"description": "获取指数/个股当前实时点位（腾讯→新浪→东财三源兜底）。文案里所有「当前点位」必须来自本工具，绝不编造。失败时如实告知并让用户手动提供，不要补数。",
				"parameters": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"标的":  map[string]any{"type": "string", "description": "标的名（如 中证1000/沪深300/创业板指）或代码（如 sh000852）"},
						"code": map[string]any{"type": "string", "description": "可选，直接给代码 sh000852/sz399006"},
					},
					"required": []string{"标的"},
				},
			},
		},
		{
			Type: "function",
			Function: map[string]any{
				"name":        "calc_points",
				"description": "按当前点位 + 降落伞/期初敲出线/派息线百分比，机械换算绝对点位（降落伞点位、期初敲出点位、派息触发点位）+ 口语化约点。文案里的绝对点位必须来自本工具，绝不自己算小数。current_price 先用 fetch_quote 拿到。",
				"parameters": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"降落伞":         map[string]any{"type": "string", "description": "如 60%"},
						"期初敲出线":      map[string]any{"type": "string", "description": "如 101%"},
						"派息线":         map[string]any{"type": "string", "description": "如 78%；不适用时留空"},
						"current_price": map[string]any{"type": "number", "description": "fetch_quote 返回的当前点位"},
					},
					"required": []string{"降落伞", "期初敲出线", "current_price"},
				},
			},
		},
```

- [ ] **Step 3: `systemPrompt` 追加文案指令**

`service.go` line 25 的 `systemPrompt` 常量，在末尾 `不要自行补数。"` 之前（即字符串末尾的 `"` 之前）追加一段（保持单个 const 字符串，用 `\n` 拼接）：

把：
```go
const systemPrompt = "你是一个专业的金融结构化产品业务助手，服务于业务工作台系统。请使用中文回答，优先基于系统内已有业务数据和用户问题给出简洁、准确的回复。需要查询产品、客户、交易、观察日历、投顾材料或业务统计时，主动调用可用工具。\n\n搜索产品时请注意：产品名称（name）通常是「航班服务XX号」这样的格式，标的指数或挂钩标的可能在标的代码（code）字段中。如果按产品名称搜索未果，请尝试用标的关键词搜索，例如用「中证1000」「沪深300」「恒科」「中证500」等关键词。也可以先调用 get_product_analytics 查看有哪些不同的标的和结构类型，再针对性搜索。\n\n当用户想要生成、制作、下载「喜报」「分红喜报」「分红观察喜报」时：先调用 search_products 找到目标产品的 product_id，再调用 generate_poster(product_id, observation_date) 生成。喜报里的所有数字（年化收益、本月分红、累计分红率、累计分红次数、派息界限、止盈界限、末月降至、挂钩标的、入场时间）都由系统从真实数据计算，你绝不可在对话中编造、估算或改写这些数字，也不可在 generate_poster 参数里传任何数字。若系统返回错误（如无该观察日记录），如实告知用户，不要自行补数。"
```
改为（仅末尾追加一段）：
```go
const systemPrompt = "你是一个专业的金融结构化产品业务助手，服务于业务工作台系统。请使用中文回答，优先基于系统内已有业务数据和用户问题给出简洁、准确的回复。需要查询产品、客户、交易、观察日历、投顾材料或业务统计时，主动调用可用工具。\n\n搜索产品时请注意：产品名称（name）通常是「航班服务XX号」这样的格式，标的指数或挂钩标的可能在标的代码（code）字段中。如果按产品名称搜索未果，请尝试用标的关键词搜索，例如用「中证1000」「沪深300」「恒科」「中证500」等关键词。也可以先调用 get_product_analytics 查看有哪些不同的标的和结构类型，再针对性搜索。\n\n当用户想要生成、制作、下载「喜报」「分红喜报」「分红观察喜报」时：先调用 search_products 找到目标产品的 product_id，再调用 generate_poster(product_id, observation_date) 生成。喜报里的所有数字（年化收益、本月分红、累计分红率、累计分红次数、派息界限、止盈界限、末月降至、挂钩标的、入场时间）都由系统从真实数据计算，你绝不可在对话中编造、估算或改写这些数字，也不可在 generate_poster 参数里传任何数字。若系统返回错误（如无该观察日记录），如实告知用户，不要自行补数。\n\n当用户想要生成结构化产品推介文案/材料（雪球/降敲雪球/DCN/FCN/限亏雪球等）时：先调用 load_skill('structured-product-copywriter') 加载完整工作流，再严格按其步骤执行——核对 10 项参数（缺了一次性问全）、取当前点位、做点位换算、产出长版+短版文案。文案里的「当前点位」必须调 fetch_quote 取真实值，绝对点位（降落伞/敲出/派息触发）必须调 calc_points 计算，你绝不可在对话中编造点位、胜率或自行做小数换算。胜率步骤：当前阶段无自动获取工具，用 [胜率待补] 占位并请用户手动提供，不要编一个胜率数字。历史参考底部属判断性数据，问用户要，不要自己填。若 fetch_quote 失败，如实告知并让用户手动提供点位。走到通毓胜率/AMAC/Word 等重步骤时，可调 get_skill_reference 取对应参考文档。"
```

- [ ] **Step 4: 编译 + 跑全部 agent 测试**

Run: `cd backend-go && go build ./... && go test ./internal/agent/ -v`
Expected: 编译通过；全部测试 PASS（含 Task 1/2/3 的新测试）。

- [ ] **Step 5: 人工验收（端到端）**

启动后端 `cd backend-go && go run ./cmd/server`（需 `.env` 里有 `DEEPSEEK_API_KEY`），用 curl 触发 agent 对话：
```bash
curl -N -X POST http://localhost:3001/api/agent/chat \
  -H 'Content-Type: application/json' \
  -d '{"conversation_id":0,"message":"给中证1000 2倍DCN写一份推介文案，降落伞60%、期初敲出线101%、降敲每月0.5%、派息线78%、费后月票息1.39%、期限36M锁3M、保证金50%不追保、打款日6月30日截止、入场7月3号"}'
```
Expected（SSE 流里）：
- 先出现 `tool_call: load_skill`。
- 出现 `tool_call: fetch_quote`，`tool_done` 后可见取到的中证1000 实时点位。
- 出现 `tool_call: calc_points`，可见算出的降落伞/敲出/派息绝对点位 + 约点。
- assistant 文本输出长版 + 短版文案，数字（点位、约点）与工具返回一致，**未出现编造的胜率**（胜率处为 `[胜率待补]` 或追问用户）。
- 文案标题为正向卖点、非条件句；保证金 50% → 标题含「2 倍」。

若点位换算方向不对（如安全垫判断与历史底部方向相反），核对用户给的历史底部是否被如实采纳——属内容校验，不阻塞合入。

- [ ] **Step 6: 提交**

```bash
cd D:/projects/business-workbench
git add backend-go/internal/agent/service.go
git commit -m "feat(agent): register load_skill/get_skill_reference/fetch_quote/calc_points tools + copywriter prompt"
```

---

## Self-Review

**1. Spec coverage:**
- skill 嵌入 + `load_skill`/`get_skill_reference` 按需加载 → Task 2。✅
- `fetch_quote` 纯 Go 3 源取价（重写 `fetch_quote.py`）→ Task 3。✅
- `calc_points` 机械换算（完整性闸门，客户可见点位不靠 LLM）→ Task 1。✅
- 工具注册 + systemPrompt 指令 → Task 4。✅
- 数字零例外（点位/胜率只来自工具）→ Task 4 systemPrompt 硬约束 + 工具不接受合成数字当真值。✅
- 参数来自用户对话（非 DB）→ Task 4 验收用例即用户口述参数。✅
- S1 胜率走 `[胜率待补]`（S2 另开）→ Task 4 systemPrompt 明示 + 验收期望未编造胜率。✅
- S1 无新二进制依赖、无运维变更 → 全部纯 stdlib + embed。✅
- 文案 = 普通文本（无前端改动）→ 无前端任务，验收看 assistant 文本。✅
- S2/S3（chromedp 胜率、Word）→ 显式 out of scope，后续 plan。✅

**2. Placeholder scan:** 无 TBD/TODO。`fetch_quote` 三源 URL/解析全具体；`calc_points` 舍入规则（floor 百位，对齐 skill 示例 5280→5200）有测试锁定。`systemPrompt` 文案指令为完整字符串。Task 2 复制 skill 文件给出具体 `cp` 命令与 Windows 兜底说明。无「add error handling」式空话。

**3. Type consistency:** `CalcPoints(parachute, knockoutLine, dividendLine string, currentPrice float64) map[string]any` 在 Task 1 定义、Task 4 `calcPoints` 方法调用一致；返回键名 `parachute_point`/`parachute_point_approx`/`knockout_point`/`knockout_point_approx`/`dividend_point`/`dividend_point_approx` 在 Task 1 测试与实现一致。`LoadSkillContent(name string) (string, error)` / `GetSkillReference(skillName, refName string) (string, error)` 在 Task 2 定义、Task 4 `loadSkill`/`getSkillReference` 方法调用一致。`FetchQuote(code string) (name, price, source string, err error)` + `ResolveCode(arg string) string` 在 Task 3 定义、`fetchQuote` 方法调用一致；包级变量 `tencentBase`/`sinaBase`/`eastmoneyBase`/`quoteHTTPClient` 在 Task 3 实现与测试一致。四个 `(s *Service)` 方法名 `loadSkill`/`getSkillReference`/`fetchQuote`/`calcPoints` 在 Task 1/2/3 定义、Task 4 `executeTool` switch 调用一致。工具名 `load_skill`/`get_skill_reference`/`fetch_quote`/`calc_points` 在 Task 4 switch 与 toolDefinitions 一致。
