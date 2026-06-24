# 喜报 Agent 生成功能 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 让现有 DeepSeek agent 在对话中根据自然语言描述,从 DB 拉取真实数据生成分红观察喜报,前端渲染并可下载 PNG,生成结果回传服务端留痕。

**Architecture:** agent 新增 `generate_poster` 工具:解析意图(产品 ID + 观察日)→ 复用 `posters.GenerateData` 从 DB 拉全部数字字段 → 返回结构化 `poster_artifact`。前端在 SSE 流里收到 `poster_artifact` 事件,用新组件 `DividendReportTemplate.vue`(`html2canvas-pro` 渲染,字段绑定、只读)渲染喜报并下载 PNG;下载后把 PNG + 字段 + 哈希 POST 回服务端,落 `poster_artifacts` 表归档。数字字段零例外来自 DB,LLM 不产生任何数字。

**Tech Stack:** Go(Gin,modernc.org/sqlite 纯 Go)、Vue 3 `<script setup>`、`html2canvas-pro`(已实测能还原模板的旋转背景纸/渐变/绶带)、SSE(`data: {json}\n\n`)。

## Global Constraints

- **数字零例外锁死:** 年化收益、本月分红、累计分红率、累计分红次数、派息界限、止盈界限、末月降至、挂钩标的、入场时间、派息观察日——全部来自 `posters.GenerateData`/`product` 模型/观察日入参,agent 与前端都不得手填或覆盖。文案类字段(标题/副标题/横幅/label/qr-caption/disclaimer)可用代码内默认值,disclaimer 措辞锁死不可由 agent 改。
- **渲染库:** 用 `html2canvas-pro`(不是普通 `html2canvas`)——fidelity 测试已证明普通版会把模板的两张旋转背景纸、纸张渐变、绶带伪元素画丢,pro 版能还原。模板 CSS 原样保留,不得改 fragile 的旋转/渐变/伪元素结构。
- **PDF 暂不做:** v1 只输出 PNG。`jspdf` 不引入。"PDF" 留待后续(若要做 = jspdf 包 PNG,非矢量)。
- **范围 v1:** 仅「分红观察喜报」。敲出喜报不在范围内(无 HTML 模板)。
- **留痕:** 每张生成并发出的喜报必须回传服务端归档(PNG + 字段 JSON + content_hash),可复现。无归档的喜报不许离开系统。
- **依赖:** 后端纯 Go 不引新二进制依赖;前端新增 `html2canvas-pro`,删除被孤立的 `PosterTemplate.vue`。
- **测试:** 后端用 `db.Open(":memory:")` + `InitSchema()` 起临时 store 跑单测( precedent: `internal/app/router_knockout_test.go`)。前端无测试运行器,用人工验收清单。

---

## File Structure

后端:
- `backend-go/internal/posters/posters.go` — 新增纯函数 `BuildArtifact`(Data→展示用字段 map),可单测。
- `backend-go/internal/posters/posters_test.go` — 新建,测 `BuildArtifact`。
- `backend-go/internal/agent/service.go` — 新增 `generate_poster` 工具(`generatePoster` 方法 + `toolDefinitions` 注册 + `executeTool` 分发);`StreamCallbacks` 加 `OnArtifact`;`StreamChat` 循环里提取 artifact 并回调;`extractArtifact` 纯函数(可测);`systemPrompt` 常量追加 `generate_poster` 使用说明与"不得编造数字"硬约束。
- `backend-go/internal/agent/artifact_test.go` — 新建,测 `extractArtifact`。
- `backend-go/internal/db/schema.go` — `schemaSQL` 追加 `poster_artifacts` 建表语句。
- `backend-go/internal/db/repository.go` — 新增 `SavePosterArtifact` / `QueryPosterArtifact`。
- `backend-go/internal/db/repository_test.go` — 新建(或复用现有测试文件),测归档读写。
- `backend-go/internal/app/router.go` — `agentChat` 的 `StreamCallbacks` 加 `OnArtifact` 回调发 SSE;新增 `POST /api/posters/artifact` 路由 + `savePosterArtifact` handler;`NewRouter` 注册路由。

前端:
- `frontend/components/DividendReportTemplate.vue` — 新建:把下载的 HTML 模板转成字段绑定的只读 SFC,`html2canvas-pro` 出 PNG,暴露 `downloadPng()`/`getPngDataUrl()`。
- `frontend/components/ChatMessage.vue` — 加 `artifact` prop + `v-if="artifact"` 渲染块。
- `frontend/views/AgentChat.vue` — SSE 加 `poster_artifact` 分支;传 `:artifact`;下载后 POST 归档。
- `frontend/components/AgentDrawer.vue` — 同 AgentChat 的 SSE 分支 + 渲染块 + 归档。
- `frontend/package.json` — 加 `html2canvas-pro` 依赖。
- 删除 `frontend/components/PosterTemplate.vue`(被孤立,新组件取代)。

---

### Task 1: `posters.BuildArtifact` 纯函数 + 单测

把 `posters.Data` + `model.Product` 转成前端模板直接消费的展示字段 map。数字在此处一次性格式化成字符串,前端不再做任何数字解析/四舍五入。

**Files:**
- Modify: `backend-go/internal/posters/posters.go`(在文件末尾追加)
- Test: `backend-go/internal/posters/posters_test.go`(新建)

**Interfaces:**
- Consumes: `model.Product`(已 import)、`Data`(同包)、`FormatChineseDate`(同包,posters.go:43)。
- Produces: `func BuildArtifact(product model.Product, data Data, observationDate string) map[string]any` —— 返回的 map 即后续 `generate_poster` 工具放进 `poster_artifact` 的载荷,也是前端 `DividendReportTemplate.vue` 的 `fields` prop 形状。

- [ ] **Step 1: 写失败测试**

`backend-go/internal/posters/posters_test.go`:
```go
package posters

import (
	"testing"

	"business-workbench/backend-go/internal/model"
)

func TestBuildArtifact_FormatsNumbersAndDates(t *testing.T) {
	monthly := 0.0133
	coupon1st := 0.0
	product := model.Product{
		ID:        "P001",
		Name:      "鹿*8号(三期)",
		IssueDate: "2026-01-30",
		Code:      "恒科ETF指数(三期)",
		Parachute: "75%",
	}
	product.MonthlyCoupon = &monthly
	product.Coupon1st = &coupon1st

	data := Data{
		HasDividendObservation: true,
		UnderlyingName:         "恒科ETF指数",
		ParachuteValue:         "75%",
		KnockoutValue:          "102.00%",
		DividendBarrierValue:   "80%",
		MonthlyCoupon:          0.0133,
		AnnualizedReturn:       0.1596,
		DividendCount:          3,
		CumulativeDividendRate: 0.0399,
	}

	a := BuildArtifact(product, data, "2026-04-30")

	cases := map[string]string{
		"product_name":             "鹿*8号(三期)",
		"observation_date":         "2026-04-30",
		"observation_date_display": "2026年4月30日",
		"entry_date_display":       "2026年1月30日",
		"annualized_return":        "15.96",
		"monthly_coupon":           "1.33",
		"cumulative_dividend_rate": "3.99",
		"underlying_name":          "恒科ETF指数",
		"dividend_barrier_value":   "80%",
		"knockout_value":           "102.00%",
		"parachute_value":          "75%",
		"title":                    "分红观察喜报",
		"disclaimer":               "* 本产品仅面向合格投资者,衍生品为高风险资产,投资需谨慎",
	}
	for key, want := range cases {
		if got, _ := a[key].(string); got != want {
			t.Errorf("field %s = %q, want %q", key, got, want)
		}
	}
	if got, _ := a["dividend_count"].(int); got != 3 {
		t.Errorf("dividend_count = %d, want 3", got)
	}
	if got, _ := a["product_id"].(string); got != "P001" {
		t.Errorf("product_id = %q, want P001", got)
	}
}
```

- [ ] **Step 2: 跑测试确认失败**

Run: `cd backend-go && go test ./internal/posters/ -run TestBuildArtifact -v`
Expected: FAIL with `undefined: BuildArtifact`。

- [ ] **Step 3: 写最小实现**

追加到 `backend-go/internal/posters/posters.go` 末尾(`fmt` 已 import 于 posters.go:4):
```go
// BuildArtifact 把计算好的喜报数据转成前端模板直接消费的展示字段 map。
// 所有数字在此一次性格式化成字符串,前端与 agent 都不得再解析或改写这些数字。
func BuildArtifact(product model.Product, data Data, observationDate string) map[string]any {
	return map[string]any{
		"product_id":               product.ID,
		"product_name":             product.Name,
		"observation_date":         observationDate,
		"observation_date_display": FormatChineseDate(observationDate),
		"entry_date":               product.IssueDate,
		"entry_date_display":       FormatChineseDate(product.IssueDate),
		// 数字字段(锁死,前端原样展示)
		"annualized_return":        fmt.Sprintf("%.2f", data.AnnualizedReturn*100),
		"monthly_coupon":           fmt.Sprintf("%.2f", data.MonthlyCoupon*100),
		"cumulative_dividend_rate": fmt.Sprintf("%.2f", data.CumulativeDividendRate*100),
		"dividend_count":           data.DividendCount,
		"underlying_name":          data.UnderlyingName,
		"dividend_barrier_value":   data.DividendBarrierValue,
		"knockout_value":           data.KnockoutValue,
		"parachute_value":          data.ParachuteValue,
		// 文案字段(默认值,agent 不可改数字类;disclaimer 措辞锁死)
		"title":               "分红观察喜报",
		"subtitle":            "IMPORTANT MESSAGE",
		"congrats":            "Congratulations",
		"congrat_text_prefix": "热烈祝贺",
		"label_yield":         "年化收益:",
		"label_cumulative":    "累计分红",
		"label_monthly":       "本月分红:",
		"qr_caption":          "扫码了解更多详情",
		"disclaimer":          "* 本产品仅面向合格投资者,衍生品为高风险资产,投资需谨慎",
	}
}
```

- [ ] **Step 4: 跑测试确认通过**

Run: `cd backend-go && go test ./internal/posters/ -run TestBuildArtifact -v`
Expected: PASS。

- [ ] **Step 5: 提交**

```bash
cd D:/projects/business-workbench
git add backend-go/internal/posters/posters.go backend-go/internal/posters/posters_test.go
git commit -m "feat(posters): add BuildArtifact to format 喜报 fields for frontend"
```

---

### Task 2: agent `extractArtifact` 纯函数 + 单测

从工具返回的 `map[string]any` 里安全取出 `poster_artifact`,把"检测 artifact"逻辑独立出来便于测试,避免直接测 SSE。

**Files:**
- Modify: `backend-go/internal/agent/service.go`
- Test: `backend-go/internal/agent/artifact_test.go`(新建)

**Interfaces:**
- Produces: `func extractArtifact(toolResult map[string]any) (map[string]any, bool)` —— 供 Task 3 的 `StreamChat` 循环调用。

- [ ] **Step 1: 写失败测试**

`backend-go/internal/agent/artifact_test.go`:
```go
package agent

import "testing"

func TestExtractArtifact(t *testing.T) {
	artifact := map[string]any{"product_name": "鹿*8号(三期)", "annualized_return": "15.96"}

	t.Run("present", func(t *testing.T) {
		result := map[string]any{
			"poster_artifact": artifact,
			"message":         "已生成",
		}
		got, ok := extractArtifact(result)
		if !ok {
			t.Fatal("expected ok=true")
		}
		if got["product_name"] != "鹿*8号(三期)" {
			t.Errorf("got %v", got["product_name"])
		}
	})

	t.Run("absent", func(t *testing.T) {
		if _, ok := extractArtifact(map[string]any{"count": 0}); ok {
			t.Fatal("expected ok=false when no artifact")
		}
	})

	t.Run("wrong_type", func(t *testing.T) {
		if _, ok := extractArtifact(map[string]any{"poster_artifact": "not a map"}); ok {
			t.Fatal("expected ok=false when artifact is not a map")
		}
	})
}
```

- [ ] **Step 2: 跑测试确认失败**

Run: `cd backend-go && go test ./internal/agent/ -run TestExtractArtifact -v`
Expected: FAIL with `undefined: extractArtifact`。

- [ ] **Step 3: 写最小实现**

在 `service.go` 的 `executeTool` 函数上方(约 line 220 之前)追加:
```go
// extractArtifact 从工具返回结果里安全取出 poster_artifact 载荷。
// 返回 false 表示该工具结果不含喜报 artifact(普通工具调用)。
func extractArtifact(toolResult map[string]any) (map[string]any, bool) {
	raw, ok := toolResult["poster_artifact"]
	if !ok {
		return nil, false
	}
	m, ok := raw.(map[string]any)
	return m, ok
}
```

- [ ] **Step 4: 跑测试确认通过**

Run: `cd backend-go && go test ./internal/agent/ -run TestExtractArtifact -v`
Expected: PASS。

- [ ] **Step 5: 提交**

```bash
cd D:/projects/business-workbench
git add backend-go/internal/agent/service.go backend-go/internal/agent/artifact_test.go
git commit -m "feat(agent): add extractArtifact helper for poster artifacts"
```

---

### Task 3: agent `generate_poster` 工具 + `OnArtifact` 回调 + SSE 事件

把工具注册进 agent,数据流跑通:工具调 `posters.BuildArtifact` → 返回 `poster_artifact` → `StreamChat` 用 `extractArtifact` 检出 → 通过新回调 `OnArtifact` 通知 HTTP 层 → `agentChat` 发 `poster_artifact` SSE 事件。

**Files:**
- Modify: `backend-go/internal/agent/service.go`(多处:`StreamCallbacks`、`StreamChat` 循环、`executeTool` switch、`toolDefinitions`、`systemPrompt`、import)
- Modify: `backend-go/internal/app/router.go`(`agentChat` 的 `StreamCallbacks`)
- Test: 复用 Task 2 的 `extractArtifact` 测试;本任务无新单测(SSE/模型调用需真实 DeepSeek,属人工验收)。

**Interfaces:**
- Consumes: `posters.BuildArtifact`(Task 1)、`extractArtifact`(Task 2)、`s.store.QueryOngoingProducts`、`s.store.QueryObservationsByProduct`、`stringArg`(service.go:598)、`posters.GenerateData`(posters.go:26)。
- Produces: SSE 事件 `{"type":"poster_artifact","artifact":{...}}`,由 Task 5/6 的前端消费。

- [ ] **Step 1: 加 `OnArtifact` 到 `StreamCallbacks`**

`service.go` 约 line 32-37,把 `StreamCallbacks` 改为:
```go
type StreamCallbacks struct {
	OnReasoning func(string)
	OnDelta     func(string)
	OnToolCall  func(string)
	OnToolDone  func(string)
	OnArtifact  func(map[string]any)
}
```

- [ ] **Step 2: `StreamChat` 循环里提取并回调 artifact**

`service.go` 约 line 57-91 的工具调用块,在 `toolResult := s.executeTool(...)` 之后、`callbacks.OnToolDone` 之前/之后均可,插入:
```go
		toolResult := s.executeTool(toolCall.Function.Name, toolCall.Function.Arguments)
		if art, ok := extractArtifact(toolResult); ok && callbacks.OnArtifact != nil {
			callbacks.OnArtifact(art)
		}
		if callbacks.OnToolDone != nil {
			callbacks.OnToolDone(toolCall.Function.Name)
		}
```
(即把原 `toolResult := ...` 行下面紧接 `if callbacks.OnToolDone` 的两行,中间插入 extractArtifact 块。)

- [ ] **Step 3: 加 `generate_poster` 到 `executeTool` switch**

`service.go` `executeTool` 的 switch(约 line 220-262),在 `case "get_posters":` 旁边加一行:
```go
	case "generate_poster":
		return s.generatePoster(args)
```

- [ ] **Step 4: 实现 `generatePoster` 方法**

在 `service.go` 的 `getPosters` 方法(约 line 405-424)之后追加。确保文件 import 区有 `"time"` 与 `"business-workbench/backend-go/internal/posters"`(若缺则补)。
```go
// generatePoster 是 agent 的喜报生成工具:按产品 ID + 观察日从 DB 拉真实数据,
// 经 posters.GenerateData / BuildArtifact 组装成展示字段,以 poster_artifact 返回。
// 数字全部来自 DB,本函数不产生、不接受任何数字入参。
func (s *Service) generatePoster(args map[string]any) map[string]any {
	productID := stringArg(args, "product_id")
	observationDate := stringArg(args, "observation_date")
	if observationDate == "" {
		observationDate = time.Now().Format("2006-01-02")
	}
	if productID == "" {
		return map[string]any{"error": "product_id is required"}
	}

	products, err := s.store.QueryOngoingProducts()
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	var product *model.Product
	for i := range products {
		if products[i].ID == productID {
			product = &products[i]
			break
		}
	}
	if product == nil {
		return map[string]any{"error": "product not found or not ongoing: " + productID}
	}

	records, err := s.store.QueryObservationsByProduct(productID)
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	monthsSinceEntry := 0
	found := false
	for i := range records {
		if records[i].ObservationDate == observationDate && records[i].MonthsSinceEntry != nil {
			monthsSinceEntry = *records[i].MonthsSinceEntry
			found = true
			break
		}
	}
	if !found {
		return map[string]any{"error": "no observation record for " + observationDate + " on product " + productID}
	}

	data := posters.GenerateData(*product, observationDate, monthsSinceEntry)
	artifact := posters.BuildArtifact(*product, data, observationDate)
	return map[string]any{
		"poster_artifact":   artifact,
		"product_id":        productID,
		"observation_date":  observationDate,
		"message":           "已生成「" + product.Name + "」(" + observationDate + ")的分红观察喜报,请在下方查看并下载。",
	}
}
```

- [ ] **Step 5: 在 `toolDefinitions()` 注册工具**

`service.go` `toolDefinitions()`(约 line 664-889)的 slice 里,挨着 `get_posters` 定义(约 line 829-842)追加:
```go
	{
		Type: "function",
		Function: map[string]any{
			"name":        "generate_poster",
			"description": "为指定产品在指定观察日生成分红观察喜报(可下载的 PNG)。所有数字(年化、本月分红、累计分红率、界限、标的等)均由系统从该产品的真实观察数据计算,不要在参数里提供任何数字。先调用 search_products 拿到 product_id,再调用本工具。",
			"parameters": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"product_id":       map[string]any{"type": "string", "description": "产品 ID(由 search_products 返回)"},
					"observation_date": map[string]any{"type": "string", "description": "观察日 YYYY-MM-DD,默认今天"},
				},
				"required": []string{"product_id"},
			},
		},
	},
```

- [ ] **Step 6: 更新 `systemPrompt` 加硬约束**

`service.go` line 24 的 `systemPrompt` 常量,在末尾的字符串里追加一段(保持一个 const 字符串,用 `\n` 拼接):
```go
const systemPrompt = "你是一个专业的金融结构化产品业务助手,服务于业务工作台系统。请使用中文回答,优先基于系统内已有业务数据和用户问题给出简洁、准确的回复。需要查询产品、客户、交易、观察日历、投顾材料或业务统计时,主动调用可用工具。\n\n搜索产品时请注意:产品名称(name)通常是「航班服务XX号」这样的格式,标的指数或挂钩标的可能在标的代码(code)字段中。如果按产品名称搜索未果,请尝试用标的关键词搜索,例如用「中证1000」「沪深300」「恒科」「中证500」等关键词。也可以先调用 get_product_analytics 查看有哪些不同的标的和结构类型,再针对性搜索。\n\n当用户想要生成、制作、下载「喜报」「分红喜报」「分红观察喜报」时:先调用 search_products 找到目标产品的 product_id,再调用 generate_poster(product_id, observation_date) 生成。喜报里的所有数字(年化收益、本月分红、累计分红率、累计分红次数、派息界限、止盈界限、末月降至、挂钩标的、入场时间)都由系统从真实数据计算,你绝不可在对话中编造、估算或改写这些数字,也不可在 generate_poster 参数里传任何数字。若系统返回错误(如无该观察日记录),如实告知用户,不要自行补数。"
```

- [ ] **Step 7: HTTP 层发 SSE 事件**

`router.go` `agentChat`(约 line 1923-1936)的 `StreamCallbacks` 字面量,追加 `OnArtifact`:
```go
	content, err := s.agentSvc.StreamChat(c.Request.Context(), history, req.Message, agent.StreamCallbacks{
		OnReasoning: func(text string) {
			writeSSE(c, gin.H{"type": "reasoning_delta", "text": text})
		},
		OnDelta: func(text string) {
			writeSSE(c, gin.H{"type": "delta", "text": text})
		},
		OnToolCall: func(name string) {
			writeSSE(c, gin.H{"type": "tool_call", "name": name})
		},
		OnToolDone: func(name string) {
			writeSSE(c, gin.H{"type": "tool_done", "name": name})
		},
		OnArtifact: func(a map[string]any) {
			writeSSE(c, gin.H{"type": "poster_artifact", "artifact": a})
		},
	})
```

- [ ] **Step 8: 编译 + 跑全部后端测试**

Run: `cd backend-go && go build ./... && go test ./...`
Expected: 编译通过,全部测试 PASS(含 Task 1/2 的新测试)。

- [ ] **Step 9: 人工验收(后端)**

启动后端 `go run ./cmd/server`,用 curl 触发一次 agent 对话(需 `.env` 里有 `DEEPSEEK_API_KEY`):
```bash
curl -N -X POST http://localhost:3001/api/agent/chat \
  -H 'Content-Type: application/json' \
  -d '{"conversation_id":0,"message":"给鹿*8号三期生成分红喜报"}'
```
Expected: SSE 流里在 `tool_call`/`tool_done` 之后出现一行 `data: {"type":"poster_artifact","artifact":{"product_name":"鹿*8号(三期)",...}}`。(若该产品不在你库中,换一个存在的 ongoing 产品名。)

- [ ] **Step 10: 提交**

```bash
cd D:/projects/business-workbench
git add backend-go/internal/agent/service.go backend-go/internal/app/router.go
git commit -m "feat(agent): add generate_poster tool + poster_artifact SSE event"
```

---

### Task 4: 归档表 + repo + `POST /api/posters/artifact` 端点

前端渲染并下载 PNG 后,把 PNG + 字段 JSON + content_hash 回传服务端归档,可复现。

**Files:**
- Modify: `backend-go/internal/db/schema.go`(把 `poster_artifacts` 加进 `schemaSQL`)
- Modify: `backend-go/internal/db/repository.go`(新增 `SavePosterArtifact` / `QueryPosterArtifact`)
- Test: `backend-go/internal/db/repository_test.go`(新建)
- Modify: `backend-go/internal/app/router.go`(`savePosterArtifact` handler + `NewRouter` 注册)

**Interfaces:**
- Consumes: `db.Store`、`isoNow`(repository.go 已有)、`nullableDBString`(已有)。
- Produces: `POST /api/posters/artifact` → `{id, url}`,PNG 落 `backend-go/public/poster-artifacts/<id>.png`,由现有 `router.Static("/public","public")`(router.go:107)托管。

- [ ] **Step 1: 写失败测试(repo)**

`backend-go/internal/db/repository_test.go`:
```go
package db

import (
	"path/filepath"
	"testing"
)

func TestSaveAndQueryPosterArtifact(t *testing.T) {
	store, err := Open(":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	if err := store.InitSchema(); err != nil {
		t.Fatalf("schema: %v", err)
	}

	fieldsJSON := `{"product_name":"鹿*8号(三期)","annualized_return":"15.96"}`
	id, err := store.SavePosterArtifact("P001", "2026-04-30", fieldsJSON, "poster-artifacts/1.png", "abc123hash")
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	if id == 0 {
		t.Fatal("expected non-zero id")
	}

	got, err := store.QueryPosterArtifact(id)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if got.ProductID != "P001" || got.ObservationDate != "2026-04-30" {
		t.Errorf("got %+v", got)
	}
	if got.ContentHash != "abc123hash" || got.PngPath != "poster-artifacts/1.png" {
		t.Errorf("hash/path = %q / %q", got.ContentHash, got.PngPath)
	}
	if got.FieldsJSON != fieldsJSON {
		t.Errorf("fields_json = %q", got.FieldsJSON)
	}
	_ = filepath.Separator // silence unused import on non-cross-platform builds
}
```

- [ ] **Step 2: 跑测试确认失败**

Run: `cd backend-go && go test ./internal/db/ -run TestSaveAndQueryPosterArtifact -v`
Expected: FAIL with `undefined: SavePosterArtifact`。

- [ ] **Step 3: 加表结构**

`schema.go` 的 `schemaSQL` 常量里(在 `posters` 表 CREATE 之后,`agent_conversations` 之前均可)追加:
```sql
CREATE TABLE IF NOT EXISTS poster_artifacts (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  product_id TEXT NOT NULL,
  observation_date TEXT NOT NULL,
  fields_json TEXT NOT NULL,
  png_path TEXT,
  content_hash TEXT NOT NULL,
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
);
```

- [ ] **Step 4: 加 repo 方法 + 模型**

先在 `backend-go/internal/model/models.go` 末尾加结构:
```go
type PosterArtifact struct {
	ID              int64  `json:"id"`
	ProductID       string `json:"product_id"`
	ObservationDate string `json:"observation_date"`
	FieldsJSON      string `json:"fields_json"`
	PngPath         string `json:"png_path"`
	ContentHash     string `json:"content_hash"`
	CreatedAt       string `json:"created_at"`
}
```

在 `repository.go` 追加(挨着其他 poster 方法,约 line 594 之后):
```go
func (s *Store) SavePosterArtifact(productID, observationDate, fieldsJSON, pngPath, contentHash string) (int64, error) {
	res, err := s.DB.Exec(`INSERT INTO poster_artifacts (product_id, observation_date, fields_json, png_path, content_hash, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		productID, observationDate, fieldsJSON, nullableDBString(pngPath), contentHash, isoNow())
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) QueryPosterArtifact(id int64) (model.PosterArtifact, error) {
	var row model.PosterArtifact
	var pngPath sql.NullString
	err := s.DB.QueryRow(`SELECT id, product_id, observation_date, fields_json, png_path, content_hash, created_at FROM poster_artifacts WHERE id = ?`, id).
		Scan(&row.ID, &row.ProductID, &row.ObservationDate, &row.FieldsJSON, &pngPath, &row.ContentHash, &row.CreatedAt)
	if pngPath.Valid {
		row.PngPath = pngPath.String
	}
	return row, err
}
```
(repository.go 已 import `database/sql` 与 `model`,确认即可。)

- [ ] **Step 5: 跑测试确认通过**

Run: `cd backend-go && go test ./internal/db/ -run TestSaveAndQueryPosterArtifact -v`
Expected: PASS。

- [ ] **Step 6: 加 HTTP handler**

`router.go` 在 `generatePosters`(约 line 1493)之后追加。`os` 与 `encoding/base64` 需在 import 区补(确认/补):
```go
func (s *Server) savePosterArtifact(c *gin.Context) {
	var req struct {
		ProductID       string `json:"product_id"`
		ObservationDate string `json:"observation_date"`
		Fields          map[string]any `json:"fields"`
		PNGBase64       string `json:"png_base64"`
		ContentHash     string `json:"content_hash"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.ProductID == "" || req.ContentHash == "" || req.PNGBase64 == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "product_id, content_hash, png_base64 are required"})
		return
	}
	fieldsJSON, _ := json.Marshal(req.Fields)

	id, err := s.store.SavePosterArtifact(req.ProductID, req.ObservationDate, string(fieldsJSON), "", req.ContentHash)
	if err != nil {
		writeError(c, err)
		return
	}
	dir := "public/poster-artifacts"
	if err := os.MkdirAll(dir, 0o755); err != nil {
		writeError(c, err)
		return
	}
	pngPath := fmt.Sprintf("%s/%d.png", dir, id)
	data, err := base64.StdEncoding.DecodeString(req.PNGBase64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid png_base64: " + err.Error()})
		return
	}
	if err := os.WriteFile(pngPath, data, 0o644); err != nil {
		writeError(c, err)
		return
	}
	// 回填 png_path(省一个 repo 方法:直接 Exec)
	if _, err := s.store.DB.Exec(`UPDATE poster_artifacts SET png_path = ? WHERE id = ?`, pngPath, id); err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id, "url": "/public/poster-artifacts/" + fmt.Sprint(id) + ".png"})
}
```
(确认 `fmt`、`os`、`encoding/base64`、`encoding/json` 已在 router.go import。`s.store.DB` 是 `*sql.DB`,可直接 Exec。)

- [ ] **Step 7: 注册路由**

`router.go` `NewRouter` 约 line 153-165,在 `router.POST("/api/posters/generate", server.generatePosters)` 下一行加:
```go
	router.POST("/api/posters/artifact", server.savePosterArtifact)
```

- [ ] **Step 8: 编译 + 测试**

Run: `cd backend-go && go build ./... && go test ./...`
Expected: PASS。

- [ ] **Step 9: 人工验收(端点)**

启动后端,用一张最小 PNG(base64)调用:
```bash
curl -X POST http://localhost:3001/api/posters/artifact \
  -H 'Content-Type: application/json' \
  -d '{"product_id":"P001","observation_date":"2026-04-30","fields":{"product_name":"测试"},"png_base64":"iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+M8AAAMBAQDJ/pLvAAAAAElFTkSuQmCC","content_hash":"testhash"}'
```
Expected: `{"id":1,"url":"/public/poster-artifacts/1.png"}`,且 `backend-go/public/poster-artifacts/1.png` 文件存在、`GET http://localhost:3001/public/poster-artifacts/1.png` 可下载。

- [ ] **Step 10: 提交**

```bash
cd D:/projects/business-workbench
git add backend-go/internal/db/schema.go backend-go/internal/db/repository.go backend-go/internal/db/repository_test.go backend-go/internal/model/models.go backend-go/internal/app/router.go
git commit -m "feat(posters): archive generated 喜报 PNG + fields to poster_artifacts"
```

---

### Task 5: 前端 `DividendReportTemplate.vue` 组件

把下载的 `dividend_report_editable.html` 转成字段绑定的只读 Vue SFC,用 `html2canvas-pro` 出 PNG,暴露 `downloadPng()` / `getPngDataUrl()`。CSS 原样保留(fidelity 已验证)。

**Files:**
- Create: `frontend/components/DividendReportTemplate.vue`
- 依赖:Task 前置——`html2canvas-pro` 加入 `frontend/package.json`。

**Interfaces:**
- Consumes: `html2canvas-pro`(npm 包)。
- Produces: Vue 组件,props `{ fields: Object }`;`defineExpose({ downloadPng, getPngDataUrl })`。`fields` 形状 = Task 1 `BuildArtifact` 返回的 map。

- [ ] **Step 1: 加依赖**

Run:
```bash
cd D:/projects/business-workbench/frontend
npm install html2canvas-pro
```
确认 `frontend/package.json` 的 `dependencies` 出现 `"html2canvas-pro": "^1.x"`。普通 `html2canvas` 依赖可保留(其他组件未必用;本组件只 import pro)。

- [ ] **Step 2: 创建组件**

`frontend/components/DividendReportTemplate.vue`:
```vue
<template>
  <div class="poster-wrapper">
    <div ref="stageRef" class="stage" aria-label="分红观察喜报">
      <div class="logo">衍选</div>
      <div class="bg-paper one"></div>
      <div class="bg-paper two"></div>

      <section class="paper">
        <div class="top-line"></div>
        <div class="title">{{ fields.title }}</div>
        <div class="subtitle">{{ fields.subtitle }}</div>
        <div class="outer-border"></div>
        <div class="congrats">{{ fields.congrats }}</div>
        <div class="message-box">
          <div class="congrat-text">{{ fields.congrat_text_prefix }}&nbsp; {{ fields.product_name }}</div>
          <div class="money">💴</div>
          <div class="date-row">派息观察日:{{ fields.observation_date_display }}</div>
        </div>
        <div class="thick-line"></div>
        <div class="label-yield">{{ fields.label_yield }}</div>
        <div class="yield">{{ fields.annualized_return }}<span class="pct">%</span></div>
        <div class="stat-label left">{{ fields.label_cumulative }} {{ fields.dividend_count }}次:</div>
        <div class="stat-label right">{{ fields.label_monthly }}</div>
        <div class="stat-num left">{{ fields.cumulative_dividend_rate }}<span class="pct">%</span></div>
        <div class="stat-num right">{{ fields.monthly_coupon }}<span class="pct">%</span></div>
        <div class="dash"></div>
        <div class="info">
          挂钩标的: {{ fields.underlying_name }}<br>
          派息界限: {{ fields.dividend_barrier_value }}<br>
          止盈界限: {{ fields.knockout_value }}<br>
          末月降至: {{ fields.parachute_value }}<br>
          入场时间: {{ fields.entry_date_display }}<br>
          <div class="note">{{ fields.disclaimer }}</div>
        </div>
        <div class="qr" aria-label="二维码装饰">
          <b class="finder f1"></b><b class="finder f2"></b><b class="finder f3"></b>
          <i style="grid-column:6;grid-row:1"></i><i style="grid-column:8;grid-row:1"></i><i style="grid-column:6;grid-row:2"></i><i style="grid-column:8;grid-row:3"></i><i style="grid-column:6;grid-row:5"></i><i style="grid-column:7;grid-row:5"></i><i style="grid-column:9;grid-row:5"></i><i style="grid-column:11;grid-row:5"></i><i style="grid-column:13;grid-row:5"></i>
          <i style="grid-column:1;grid-row:6"></i><i style="grid-column:3;grid-row:6"></i><i style="grid-column:5;grid-row:6"></i><i style="grid-column:7;grid-row:6"></i><i style="grid-column:10;grid-row:6"></i><i style="grid-column:12;grid-row:6"></i>
          <i style="grid-column:2;grid-row:7"></i><i style="grid-column:4;grid-row:7"></i><i style="grid-column:6;grid-row:7"></i><i style="grid-column:8;grid-row:7"></i><i style="grid-column:9;grid-row:7"></i><i style="grid-column:13;grid-row:7"></i>
          <i style="grid-column:1;grid-row:8"></i><i style="grid-column:5;grid-row:8"></i><i style="grid-column:7;grid-row:8"></i><i style="grid-column:11;grid-row:8"></i><i style="grid-column:12;grid-row:8"></i>
          <i style="grid-column:3;grid-row:9"></i><i style="grid-column:6;grid-row:9"></i><i style="grid-column:8;grid-row:9"></i><i style="grid-column:10;grid-row:9"></i><i style="grid-column:13;grid-row:9"></i>
          <i style="grid-column:6;grid-row:10"></i><i style="grid-column:8;grid-row:10"></i><i style="grid-column:10;grid-row:10"></i><i style="grid-column:12;grid-row:10"></i>
          <i style="grid-column:5;grid-row:11"></i><i style="grid-column:7;grid-row:11"></i><i style="grid-column:9;grid-row:11"></i><i style="grid-column:11;grid-row:11"></i><i style="grid-column:13;grid-row:11"></i>
          <i style="grid-column:6;grid-row:12"></i><i style="grid-column:8;grid-row:12"></i><i style="grid-column:10;grid-row:12"></i><i style="grid-column:12;grid-row:12"></i>
          <i style="grid-column:5;grid-row:13"></i><i style="grid-column:7;grid-row:13"></i><i style="grid-column:9;grid-row:13"></i><i style="grid-column:11;grid-row:13"></i>
        </div>
        <div class="qr-caption">{{ fields.qr_caption }}</div>
      </section>
    </div>

    <div class="poster-actions">
      <button class="btn-download" :disabled="isGenerating" @click="downloadPng">
        {{ isGenerating ? '生成中...' : '下载图片' }}
      </button>
    </div>
  </div>
</template>

<script setup>
import { ref, nextTick } from 'vue'
import html2canvas from 'html2canvas-pro'

const props = defineProps({
  fields: { type: Object, required: true },
})

const stageRef = ref(null)
const isGenerating = ref(false)

async function getPngDataUrl() {
  if (!stageRef.value) return null
  await nextTick()
  await document.fonts.ready
  const canvas = await html2canvas(stageRef.value, {
    scale: 2,
    backgroundColor: null,
    useCORS: true,
    logging: false,
  })
  return canvas.toDataURL('image/png')
}

async function downloadPng() {
  if (!stageRef.value || isGenerating.value) return
  isGenerating.value = true
  try {
    const dataUrl = await getPngDataUrl()
    if (!dataUrl) return
    const name = (props.fields.product_name || '产品').replace(/[\\/:*?"<>|]/g, '_')
    const link = document.createElement('a')
    link.download = `分红观察喜报_${name}_${props.fields.observation_date || ''}.png`
    link.href = dataUrl
    link.click()
  } catch (e) {
    console.error('生成喜报图片失败:', e)
  } finally {
    isGenerating.value = false
  }
}

defineExpose({ downloadPng, getPngDataUrl })
</script>

<style scoped src="@/../downloads-dividend-style.css"></style>
```

**注意 `<style>` 来源:** 上面的 `scoped src=` 是占位说明——实际把下载模板 `C:\Users\WIKO\Downloads\dividend_report_editable.html` 的 `<style>...</style>` 全部内联到本 SFC 的 `<style scoped>` 里(从 `:root{...}` 到 `[contenteditable="true"]:focus{...}` 全部照搬,**不要改任何选择器与值**,fidelity 已验证)。即最终组件用:
```vue
<style scoped>
:root{ --red:#cf141d; --deep-red:#c71018; --paper:#f5f0e7; --ink:#111; }
*{box-sizing:border-box}
/* …把下载模板 <style> 内全部 CSS 原样粘贴到此… */
.poster-wrapper{ display:flex; flex-direction:column; align-items:center; }
.poster-actions{ margin-top:12px; }
.btn-download{ /* 复用 PosterTemplate.vue 的 .btn-download 样式,或自拟 */ padding:8px 16px; background:#cf141d; color:#fff; border:none; border-radius:6px; cursor:pointer; }
.btn-download:disabled{ opacity:.6; cursor:default; }
</style>
```
(把 `body{...}` 那条整页居中黑底的规则删掉——它会污染组件宿主;保留 `.stage` 及其内部全部规则。)

- [ ] **Step 3: 删除被孤立的旧组件**

Run:
```bash
cd D:/projects/business-workbench/frontend
git rm components/PosterTemplate.vue
```
(确认无 import 后删除;新组件取代。)

- [ ] **Step 4: 人工验收(组件单独可渲染)**

临时在 `AgentChat.vue` 或一个临时路由里挂载 `<DividendReportTemplate :fields="sampleFields" />`,`sampleFields` 用 Task 1 测试里的同款值,`npm run dev`,打开页面:
- Expected: 喜报渲染正常(红底、两张旋转背景纸、米色卡片、绶带、数字、二维码网格都在)。
- 点"下载图片":得到一张 PNG,打开比对 `.local-tools/poster-fidelity/html2canvas-pro.png`,视觉一致。
- 验收后移除临时挂载。

- [ ] **Step 5: 提交**

```bash
cd D:/projects/business-workbench
git add frontend/components/DividendReportTemplate.vue frontend/package.json frontend/package-lock.json
git rm frontend/components/PosterTemplate.vue
git commit -m "feat(frontend): add DividendReportTemplate, drop orphaned PosterTemplate"
```

---

### Task 6: 前端接入 SSE `poster_artifact` + 渲染 + 归档回传

在 AgentChat 与 AgentDrawer 的 SSE 消费循环里加 `poster_artifact` 分支,把 artifact 挂到 assistant 消息上;ChatMessage(及 AgentDrawer 内联模板)渲染 `DividendReportTemplate`;下载后调用 Task 4 的 `POST /api/posters/artifact` 回传归档。

**Files:**
- Modify: `frontend/components/ChatMessage.vue`(加 `artifact` prop + 渲染块)
- Modify: `frontend/views/AgentChat.vue`(SSE 分支 + `:artifact` + 归档)
- Modify: `frontend/components/AgentDrawer.vue`(SSE 分支 + 内联渲染块 + 归档)

**Interfaces:**
- Consumes: `DividendReportTemplate.vue`(Task 5)、`POST /api/posters/artifact`(Task 4)、SSE 事件 `{"type":"poster_artifact","artifact":{...}}`(Task 3)。
- Produces: 用户在对话里看到喜报卡片 + 下载按钮;下载即归档。

- [ ] **Step 1: `ChatMessage.vue` 加 artifact 渲染**

`ChatMessage.vue` props(约 line 33-38)加一个:
```js
const props = defineProps({
  role: { type: String, required: true },
  content: { type: String, default: '' },
  streaming: { type: Boolean, default: false },
  toolCalls: { type: Array, default: null },
  artifact: { type: Object, default: null },
})
```
template(约 line 6-23 的 `.message-column` 内,在 `.message-card` 之后)加:
```vue
      <div v-if="artifact" class="artifact-card">
        <DividendReportTemplate :fields="artifact" />
      </div>
```
`<script setup>` 顶部 import:
```js
import DividendReportTemplate from './DividendReportTemplate.vue'
```

- [ ] **Step 2: `AgentChat.vue` SSE 加分支 + 传 prop + 归档**

SSE 解析(约 line 250-298 的 `else if` 链),在 `tool_done` 分支后加:
```js
} else if (event.type === 'poster_artifact') {
  const msg = messages.value.find((m) => m._tempId === assistantMsgId)
  if (msg) msg.artifact = event.artifact
  scrollToBottom()
}
```
渲染处(约 line 61-68)加 `:artifact`:
```vue
<ChatMessage
  :role="msg.role"
  :content="msg.content"
  :streaming="msg.streaming"
  :tool-calls="msg.tool_calls_display"
  :artifact="msg.artifact"
/>
```
归档:在 `ChatMessage` 上监听下载完成事件以回传。最简做法——给 `DividendReportTemplate` 的下载按钮加一个"下载并归档"包装:在 `ChatMessage.vue` 不直接用组件自带按钮,改为监听。**为减少改动**,采用:在 `ChatMessage.vue` 的 artifact 块里不用组件自带下载按钮,而由 ChatMessage 提供按钮,调用组件 `getPngDataUrl()` 后自行下载 + 归档。把 Step 1 的 artifact 块改为:
```vue
      <div v-if="artifact" class="artifact-card">
        <DividendReportTemplate ref="tplRef" :fields="artifact" :hide-actions="true" />
        <button class="btn-archive" :disabled="archiving" @click="downloadAndArchive">{{ archiving ? '归档中...' : '下载并归档' }}</button>
      </div>
```
`DividendReportTemplate.vue` 加 `hideActions` prop,当 true 时隐藏自带 `.poster-actions`(模板里 `<div v-if="!hideActions" class="poster-actions">`)。
`ChatMessage.vue` script 加:
```js
import { ref } from 'vue'
const tplRef = ref(null)
const archiving = ref(false)

async function sha256Hex(b64) {
  const bin = atob(b64)
  const buf = new Uint8Array(bin.length)
  for (let i = 0; i < bin.length; i++) buf[i] = bin.charCodeAt(i)
  const digest = await crypto.subtle.digest('SHA-256', buf)
  return Array.from(new Uint8Array(digest)).map(b => b.toString(16).padStart(2, '0')).join('')
}

async function downloadAndArchive() {
  const tpl = tplRef.value
  if (!tpl || archiving.value) return
  archiving.value = true
  try {
    const dataUrl = await tpl.getPngDataUrl()
    if (!dataUrl) return
    const b64 = dataUrl.split(',')[1]
    // 下载
    const name = (props.artifact.product_name || '产品').replace(/[\\/:*?"<>|]/g, '_')
    const link = document.createElement('a')
    link.download = `分红观察喜报_${name}_${props.artifact.observation_date || ''}.png`
    link.href = dataUrl
    link.click()
    // 归档
    const hash = await sha256Hex(b64)
    await fetch('/api/posters/artifact', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        product_id: props.artifact.product_id,
        observation_date: props.artifact.observation_date,
        fields: props.artifact,
        png_base64: b64,
        content_hash: hash,
      }),
    })
  } catch (e) {
    console.error('归档失败:', e)
  } finally {
    archiving.value = false
  }
}
```

- [ ] **Step 3: `AgentDrawer.vue` SSE 加分支 + 内联渲染 + 归档**

SSE switch(约 line 147-172)加 case:
```js
case 'poster_artifact': {
  const last = messages.value[messages.value.length - 1]
  if (last && last.role === 'assistant') last.artifact = event.artifact
  else messages.value.push({ role: 'assistant', content: '', artifact: event.artifact })
  scrollToBottom()
  break
}
```
assistant 消息对象初始化(约 line 130)加 `artifact: null`:
```js
let assistantMsg = { role: 'assistant', content: '', artifact: null }
```
template(约 line 22-32 的 `.chat-msg` 内,在 `msg.content` bubble 之后)加:
```vue
              <div v-if="msg.artifact" class="artifact-card">
                <DividendReportTemplate :fields="msg.artifact" :hide-actions="true" />
                <button class="btn-archive" :disabled="msg.archiving" @click="downloadAndArchive(msg)">{{ msg.archiving ? '归档中...' : '下载并归档' }}</button>
              </div>
```
`<script setup>` import `DividendReportTemplate` 并加与 Step 2 同款的 `sha256Hex` + `downloadAndArchive(msg)`(用 `msg` 替代 `props.artifact`/`tplRef`:在该函数内用 `document.querySelector` 取最近一个渲染好的组件实例较脆弱,故 AgentDrawer 改为:不在组件 ref 上调用,而是直接在按钮处理里新建一个临时挂载的 `DividendReportTemplate` 太复杂。**简化方案**:AgentDrawer 复用 `ChatMessage.vue` 来渲染 assistant 消息——但 AgentDrawer 当前是内联渲染,不引用 ChatMessage。**取舍**:为控制改动量,AgentDrawer 的 artifact 块直接用 `DividendReportTemplate` 自带下载按钮(不自建归档按钮),并在 `DividendReportTemplate` 的 `downloadPng` 之后 emit 一个 `downloaded` 事件(payload=dataUrl),AgentDrawer 监听该事件做归档。)

**统一更优解(回改 Task 5):** 给 `DividendReportTemplate.vue` 加 `emit(['downloaded'])`,在 `downloadPng` 内得到 dataUrl 后 `emit('downloaded', dataUrl)`。于是两个宿主都只需:
```vue
<DividendReportTemplate :fields="msg.artifact" @downloaded="onDownloaded($event, msg)" />
```
`onDownloaded(dataUrl, msg)` 内:自行触发 `<a download>`(若组件已触发则跳过)、算 hash、POST 归档。组件自带按钮照常显示(`hide-actions` 不再需要,删掉该 prop 以简化)。

→ **修订 Task 5 Step 2**:组件去掉 `hideActions`、`getPngDataUrl` 保留 expose、`downloadPng` 内 emit `downloaded`(payload为 dataUrl)。ChatMessage/AgentDrawer 都用 `@downloaded` 归档,不再各自实现下载按钮。

按此修订落实 Task 5 组件:
```js
const emit = defineEmits(['downloaded'])
async function downloadPng() {
  // …同前得到 dataUrl…
  const link = document.createElement('a')
  link.download = `分红观察喜报_${name}_${props.fields.observation_date || ''}.png`
  link.href = dataUrl
  link.click()
  emit('downloaded', dataUrl)
}
```
两个宿主:
```js
async function onDownloaded(dataUrl, msg) {
  if (!dataUrl) return
  msg.archiving = true
  try {
    const b64 = dataUrl.split(',')[1]
    const hash = await sha256Hex(b64)
    await fetch('/api/posters/artifact', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        product_id: msg.artifact.product_id,
        observation_date: msg.artifact.observation_date,
        fields: msg.artifact,
        png_base64: b64,
        content_hash: hash,
      }),
    })
  } catch (e) { console.error('归档失败:', e) }
  finally { msg.archiving = false }
}
```
(ChatMessage 里 `msg` 即当前消息;可用 `<DividendReportTemplate :fields="artifact" @downloaded="d => onDownloaded(d, props)" />`,props 不可变故把 archiving 放 ref。)

- [ ] **Step 4: 人工验收(端到端)**

启动后端 + `npm run dev`,在 AgentChat 页输入"给鹿*8号三期生成分红喜报"(换成库里存在的 ongoing 产品):
- Expected: assistant 回复文字 + 出现喜报卡片(旋转背景纸/绶带/数字齐全)。
- 点"下载图片":浏览器下载 PNG;同时 Network 出现 `POST /api/posters/artifact` 200,`backend-go/public/poster-artifacts/<id>.png` 生成。
- 在 AgentDrawer(若产品里有挂入口)重复一次,行为一致。

- [ ] **Step 5: 提交**

```bash
cd D:/projects/business-workbench
git add frontend/components/ChatMessage.vue frontend/components/DividendReportTemplate.vue frontend/views/AgentChat.vue frontend/components/AgentDrawer.vue
git commit -m "feat(frontend): render poster_artifact in chat + archive on download"
```

---

## Self-Review

**1. Spec coverage:**
- 客户端 html2canvas-pro 渲染 PNG → Task 5。✅
- agent 工具解析意图→DB→结构化返回 → Task 3(`generate_poster`,数字来自 `posters.GenerateData`)。✅
- 数字零例外锁死 → Task 1(`BuildArtifact` 唯一出口)+ Task 3(`systemPrompt` 硬约束 + 工具不接受数字参数)。✅
- 文案可注入、disclaimer 锁死 → Task 1 默认值。✅
- 工具返回→前端渲染 → Task 3 SSE + Task 6 渲染。✅
- 留痕(a) → Task 4 归档表/端点 + Task 6 下载即回传。✅
- 范围 v1 仅分红、删 PosterTemplate → Task 5 Step 3。✅
- fidelity 闸门 → 已在 Task 前置实测(Task 5 复用验证)。✅
- 待 sanity check 的「止盈界限=当期敲出递减」语义 → 不阻塞,Task 1 测试已锁定 `KnockoutValue` 原样透传,工程师拿真实产品核对 `posters.knockoutPercent` 递减逻辑即可(不属本计划代码改动)。

**2. Placeholder scan:** Task 5 的 `<style scoped src=...>` 已显式说明为占位说明并给出最终内联写法,非真占位——已要求把下载模板 CSS 原样粘贴。无 TBD/TODO。归档 `UPDATE png_path` 用 `s.store.DB.Exec` 直连,有具体 SQL。

**3. Type consistency:** `BuildArtifact` 返回 `map[string]any`,字段名(`product_id`/`annualized_return`/`observation_date`/...)在 Task 3 工具、Task 5 组件 `fields.*`、Task 6 归档 `msg.artifact.*` 三处一致。`extractArtifact` 签名 `(map[string]any) (map[string]any, bool)` 在 Task 2 定义、Task 3 Step 2 调用一致。`OnArtifact func(map[string]any)` 在 Task 3 Step 1 定义、Step 7 调用一致。`SavePosterArtifact(productID, observationDate, fieldsJSON, pngPath, contentHash) (int64, error)` 在 Task 4 Step 1 测试、Step 4 实现、Step 6 handler 调用一致。`DividendReportTemplate` props `{fields}` + expose `{downloadPng, getPngDataUrl}` + emit `['downloaded']` 在 Task 5 与 Task 6 一致(已在 Task 6 Step 3 统一修订)。
