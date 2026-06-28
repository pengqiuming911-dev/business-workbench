# 把 structured-product-copywriter skill 集成进项目 agent — Design

> **状态:** design / 待 review。**范围:** S1(文案核心) + S2(真实胜率) + S3(Word 材料) 三个子项目，分阶段实现但同一份 spec。

## Goal

把本地 Claude Code skill `structured-product-copywriter`（位于 `~/.claude/skills/structured-product-copywriter/`）集成进 business-workbench 的 DeepSeek agent，让 agent 能在对话中**按 skill 里的内容和步骤执行**：对话式收齐 10 项产品参数 → 取实时点位 → 取真实回测胜率 → 做点位换算 → 产出长版+短版推介文案 → （可选）装配 Word 推介材料。客户可见的数字（点位、胜率）只来自工具，agent 绝不编造。

## 背景与约束

- **现有 agent:** `backend-go/internal/agent/service.go`，DeepSeek-via-Go，工具经 `toolDefinitions()` 注册 + `executeTool` switch 分发，SSE 流到 Vue 前端（`AgentChat.vue` / `AgentDrawer.vue`）。已有 RAG（`retriever` over `AllProductDocs`）、`get_price`（读 DB 缓存价）、`search_products`、`generate_poster`（喜报，数字锁死到 DB）。
- **skill 的运行时依赖:** skill 原生依赖 Claude Code 的 Bash / Playwright / 文件工具——`fetch_quote.py`（Python 取价）、`references/tongyu-winrate.md`（Playwright 登通毓终端回测）、`references/amac-manager.md`（浏览器截 AMAC）、`scripts/build_docx.py`（Python 出 Word）。Go agent **没有** Python 运行时、没有浏览器自动化能力。
- **既有约束冲突:** 喜报计划 `docs/superpowers/plans/2026-06-25-poster-from-agent.md:18` 定了「后端纯 Go 不引新二进制依赖」。本设计对**取价与点位换算**保持纯 Go；对**胜率/截图**引入 `chromedp`（需要服务器装 Chrome 二进制）——这是用户已确认接受的偏差，「纯 Go」对浏览器步骤放宽为「Go + Chrome」。
- **参数来源:** 按用户确认，**参数由用户在对话中提供**（按 skill 原文步骤 1），不从 DB 拉取。理由：skill 面向的是募集中/推介中的 prospective 产品，未必在持仓 ongoing 库里；与喜报（面向已有持仓 + 观察记录）是不同业务场景。
- **数字零例外（沿用喜报哲学）:** 客户可见数字（点位、胜率）只来自工具计算或工具获取，LLM 不做算术、不编行情/胜率。机械换算用 Go 工具，判断/卖点组织留给 LLM。

## Architecture

### skill 如何「进入」agent

skill 文件纳入仓库、用 `//go:embed` 嵌进 Go 二进制——服务器上不依赖 `~/.claude/skills/...` 磁盘路径（生产 ECS 上不存在）。两个 meta-tool 按需把内容暴露给 agent，base system prompt 保持精简：

- **`load_skill(name)`** → 返回 `SKILL.md` 原文（verbatim，不缩写）。`systemPrompt` 追加一句指令：*「当用户想要生成结构化产品推介文案/材料时，先调用 `load_skill('structured-product-copywriter')` 加载完整工作流，再严格按其步骤执行。」*
- **`get_skill_reference(name)`** → 返回某份重步骤参考文档（`tongyu-winrate.md` / `amac-manager.md` / `product-position-card.md` / `docx-template.md`），当 agent 走到该步骤时才加载。保持常驻 payload 小；详细表单字段映射只在需要时入上下文。

**为何这样:** verbatim（不缩写）忠实执行「按 skill 内容和步骤执行」；按需加载（不常驻 inline）避免污染非文案对话；确定性 tool call（非 RAG）规避方案 3 的检索赌博。若验收发现「tool 结果里的大段工作流指令」跟随性弱，fallback 是把 skill 提升进 systemPrompt——列为 contingency，不作默认。

### Tool inventory（全部 Go，按既有模式注册）

| # | Tool | 子项目 | 运行时 | 职责 |
|---|------|--------|--------|------|
| 1 | `load_skill(name)` | S1 infra | pure Go(embed) | 返回 SKILL.md 原文 |
| 2 | `get_skill_reference(name)` | S1 infra | pure Go(embed) | 返回某参考文档 |
| 3 | `fetch_quote(标的)` | S1 | pure-Go HTTP | 实时点位，3 源兜底（腾讯/新浪/东财），重写 `fetch_quote.py` |
| 4 | `calc_points(params, current_price)` | S1 | pure Go | 降落伞/敲出/派息绝对点位机械换算 |
| 5 | `fetch_winrate(params)` | S2 | chromedp+Chrome | 登通毓终端、填回测表单、读胜率；验证码/站点不可达 → `[胜率待补]` |
| 6 | `screenshot_amac(manager_id, type)` | S3 | chromedp | AMAC 管理人/产品页截图 |
| 7 | `screenshot_product_card(params)` | S3 | chromedp | 通毓产品点位卡图 |
| 8 | `build_docx(manifest)` | S3 | Go docx lib | 按 `docx-template.md` 装配 推介材料.docx，经 `/public` 托管 |

### End-to-end agent flow

1. 用户要推介文案 / 甩参数 → agent 调 `load_skill(...)` 取工作流。
2. 步骤 1：核对 10 项参数，**一次性**问全缺失项（按 skill）。
3. 步骤 2：`fetch_quote(标的)` 取当前点位；问用户要历史参考底部（判断性，不自动取）；`fetch_winrate`（S2）取胜率或 `[胜率待补]`。
4. 步骤 3：`calc_points(...)` 取绝对点位；agent 组织卖点判断（LLM，仅用工具给的数字）。
5. 步骤 4：agent 在对话里写**长版+短版**文案（LLM；数字只来自工具，绝不自算）。
6. 步骤 5（仅当用户要 Word）：`screenshot_*` + `build_docx` → 返回下载 URL。

## File Structure

```
backend-go/internal/agent/
  skills/structured-product-copywriter/{SKILL.md, references/*.md}   # embedded
  skill_loader.go        # load_skill + get_skill_reference + //go:embed   [S1]
  quote.go               # fetch_quote (3 源 HTTP)                       [S1]
  calc.go                # calc_points (纯算术)                          [S1]
  winrate.go             # fetch_winrate (chromedp 通毓)                 [S2]
  browser.go             # chromedp 共享 helper (login/screenshot)       [S2/S3]
  docx.go                # build_docx (Go docx lib)                      [S3]
  service.go             # 注册 8 个工具 + systemPrompt 指令
  *_test.go              # 各工具测试
frontend/                # S1 无改动（文案=assistant 文本）；S3 加一个下载链接卡
```

## 子项目细节

### S1 — 文案核心（pure Go，无新二进制依赖，无运维变更）

**`load_skill` / `get_skill_reference`** (`skill_loader.go`)
- `//go:embed skills/structured-product-copywriter/*` 进 `embed.FS`。按名查文件，返回 `{"content": "..."}` 或 `{"error":"not found"}`。

**`fetch_quote(标的)`** (`quote.go`) — 重写 `fetch_quote.py`
- 纯 Go HTTP，依次打腾讯 `qt.gtimg.cn`、新浪 `hq.sinajs.cn`、东财 `push2.eastmoney.com`，第一个非空胜出。标的 name（中证1000/沪深300/中证500/创业板指/个股）→ 各源 symbol code 用一张静态表（直接 port `fetch_quote.py` 里的映射）。
- 返回 `{"标的":..., "最新点位": <float>, "source": "tencent|sina|eastmoney"}`。
- 三源全失败 → `{"error":"quote unavailable"}`，agent **回头问用户**要点位（按 skill：脚本失败才问，绝不编）。不与现有 `get_price`（读 DB 缓存）混用——`get_price` 是持仓缓存价，本工具是实时推介点位，语义不同。

**`calc_points(params, current_price)`** (`calc.go`) — 完整性闸门
- 输入：10 项参数 + `current_price`（来自 `fetch_quote`）。
- 纯算术，返回 skill 指定的三个绝对点位：
  - 降落伞绝对点位 = `current_price × 降落伞%`
  - 期初敲出绝对点位 = `current_price × 期初敲出线%`
  - 派息触发绝对点位 = `current_price × 派息线%`（仅当派息线适用）
- 同时返回**口语化「约点」**（按 skill 示例：5280 → "5200点左右"），agent 不再自行四舍五入。skill 原文示例把 5280 表为「5200点左右」，本工具按 skill 示例输出对齐，不另立舍入规则。
- **判断留给 LLM**（安全垫厚度对比、卖点组织）——那是判断不是算术，skill 明确要模型做。
- 为何用工具不用 LLM：客户可见点位、DeepSeek 小数算术不可靠、沿用喜报「数字零例外」闸门。

**文案渲染（前端）**
- 长版+短版作为**普通 assistant 文本**渲染（skill 输出文本块，自然聊天 UX）。S1 不加前端组件。
- *延后增强（非 v1）：* 一个带「复制长版 / 复制短版」按钮的轻卡（短版用于转发，复制到剪贴板有用）。v1 用文本选中即可。

### S2 — 真实胜率（chromedp + Chrome）

**`fetch_winrate(params)`** (`winrate.go` + `browser.go`)
- `chromedp` 起无头 Chrome，按 `references/tongyu-winrate.md` 逐步执行（agent 已通过 `get_skill_reference` 拿到映射文档；本工具负责执行）：登 `terminal.tongyu-quant.com`、按参数填回测表单、点「立即分析」、从结果区读胜率。
- 凭证：`TONGYU_USER` / `TONGYU_PASS` 从 env 读（与 `DEEPSEEK_API_KEY` 一致），**绝不**进仓库/skill 文件/prompt（skill 已强制）。
- 失败兜底 → **`[胜率待补]`** 占位（非 error）：滑块验证码、登录失败、站点不可达、选择器未命中。agent 告知「胜率待补，请手动提供」并接受用户口述胜率（按 skill）。
- 返回 `{"胜率": "98.17%"}` 或 `{"胜率": "[胜率待补]", "reason": "captcha"}`。
- 不做单测（需真实浏览器+线上站点+凭证）→ 人工验收，与 `generate_poster` 同豁免类。提供 `WINRATE_DRY_RUN=true` env 返回 canned 胜率，使流程可脱离通毓测试。

### S3 — Word 推介材料（Go docx + chromedp）

**`screenshot_amac` / `screenshot_product_card`** (`browser.go`)
- chromedp 渲染页面（AMAC 字段 JS 异步加载，纯 HTTP 抓不到，见 `amac-manager.md`），等目标区出现后整页截图 → PNG 落 `public/` 下静态目录（复用喜报归档的 `router.Static("/public","public")` 模式）。返回 `{"url": "/public/..."}`。
- 产品卡：通毓 `smallTool/index.html#/product-position`，按 `product-position-card.md` 填表，「复制为图片」→ 读剪贴板 PNG，截图兜底。

**`build_docx(manifest)`** (`docx.go`)
- Go docx 库——**自写最小纯 Go 写入器**（`archive/zip` + `encoding/xml`，仅 stdlib）。调研结论：`unioffice` 是**商业付费**（非 MIT）、`fumiama/go-docx` 是 **AGPL v3**、MIT 替代（nguyenthenguyen/docx 等）均无完整 image+table+hyperlink 能力。用户已选自写以避 license 问题、保纯 Go。覆盖 `docx-template.md` 的 7 种 section（heading/subheading/body/params/image/separator/link_list）+ 图片 20cm 高度上限缩放 + 缺图红字占位 + BMP 外 emoji 剥离 + 雅黑/Consolas 字体。
- 输入 `manifest` = 章节结构（文案长版/短版 → 公告群通知 → 派息敲出观察表图 → 胜率数据+图 → 一页通 → 管理人公示图 → 产品公示图 → 托管募集账户 → 销售常见问题），含文本内容 + 图片 URL（来自截图工具，或 `[图片待补:xxx]` 占位——镜像 `build_docx.py` 的缺图行为）。
- 输出：写 `public/推介材料/<id>.docx`，返回 `{"url": "/public/推介材料/<id>.docx"}`。前端加一个下载链接卡（仅 S3 的小组件）。
- 测试：固定 manifest → 非空 `.docx`，用 `archive/zip` 读回校验 zip 结构（[Content_Types].xml/word/document.xml/media）+ XML 文本 + 图片嵌入 + 缺图占位。

## Cross-Cutting

### Error handling（沿用 `{"error":...}` 工具返回 + skill 诚实规则）

- **工具失败 → 如实告知、绝不补数**（skill 硬规则，也呼应喜报 Q1 风险）：
  - `fetch_quote` 三源全败 → `{"error":...}`，agent 问用户要点位（不编）。
  - `fetch_winrate` 验证码/站点异常 → `{"胜率":"[胜率待补]","reason":...}`（占位非 error，agent 邀用户补）。登录失败同。
  - `calc_points` 缺 `current_price` 或参数非法 → `{"error":...}`，agent 重新收参。纯算术不会错；这是闸门工具。
  - `screenshot_*` / `build_docx` 失败 → `{"error":...}`；docx 步骤只在用户要 Word 时才跑，故障局限在 S3。
- **工具不接受合成数字当真相。** `calc_points` 接参数+价格后**计算**；`fetch_*` 取真实数据。任何工具的参数 schema 里都没有 `胜率=` 或 `点位=` 作为「真值」输入——agent 喂给 `fetch_winrate` 的唯一胜率输入是它从工具本身或用户处拿到的。与喜报 `generate_poster`（不接受数字参数）同形态。
- **喜报教训的前瞻应用**（来自喜报 Q1–Q11 grill）：v1 文案是临时聊天文本，**不归档、不存 content_hash**，故不重演喜报的归档矛盾。若日后要归档文案 artifact，**从第一天起服务端重算数字**，不信客户端 JSON。设计中标为 future risk，v1 out of scope。

### Testing

| Tool | 测试 | 方式 |
|----|------|-----|
| `load_skill` / `get_skill_reference` | unit | 断言 embed FS 对已知名返回内容、未知名返回 error |
| `fetch_quote` | unit | `httptest.Server` mock 腾讯/新浪/东财；断言首非空源胜出 + 兜底链 + 全败→error |
| `calc_points` | unit | 表驱动；锁 8800×60%=5280、8800×101%=8888、口语约点「5200点左右」 |
| `fetch_winrate` | 人工验收 + dry-run | `WINRATE_DRY_RUN=true` 返回 canned 98.17% 以脱离通毓测 agent 流程；线上通毓做 live 验收 |
| `screenshot_*` | 人工验收 | 需浏览器+线上 AMAC/通毓，无法单测 |
| `build_docx` | unit(轻) | 固定 manifest → 非空 `.docx` 且标题数符合；用 archive/zip 读回 |
| agent 流程(端到端) | 人工验收 | 需 DeepSeek + 线上服务，与 `generate_poster` 同豁免类 |

`calc_points` 与 `fetch_quote` 是高价值纯函数测试——它们是护数字的闸门工具，给与喜报 `BuildArtifact` 同等待遇。

### Config / creds / ops

- **config**（`config.Config`，env，与 `DEEPSEEK_API_KEY` 一致）：S2/S3 加 `TONGYU_USER` / `TONGYU_PASS`；`WINRATE_DRY_RUN`（默认 false）；`CHROME_PATH`（可选，默认无头）。**S1 无新 config。**
- **生产运维：** S2/S3 需 ECS（阿里云 47.103.54.197）装 Chrome。这是真运维步骤——装无头 Chrome + 依赖、验证 server 上 `chromedp` 能起。**S1 无运维变更即可发布。** 合 S2 前与运维 flag。
- **凭证安全：** tongyu 凭证只从 env 读，绝不进仓库/skill 文件/prompt。skill 已强制；工具从 config 读，不记日志。
- **`chromedp` 依赖：** `go.mod` 加 `github.com/chromedp/chromedp`。浏览器步骤唯一非平凡新 Go 依赖；S1 不加任何依赖。自写 docx 写入器于 S3 加（无新第三方依赖）。

### Phasing / merge plan

- **PR1 (S1):** `skill_loader` + `fetch_quote` + `calc_points` + 工具注册 + `systemPrompt` 指令 + 单测。胜率走 `[胜率待补]`。无 Chrome、无新二进制依赖、无运维变更。独立可合。
- **PR2 (S2):** `fetch_winrate` + `browser.go` chromedp helper + config + dry-run。前置：server 装 Chrome。合后 agent 可取真实胜率。
- **PR3 (S3):** `screenshot_amac` + `screenshot_product_card` + `build_docx`（自写 docx 写入器）+ 前端下载链接卡。复用 S2 的 `browser.go`。
- 每阶段独立验收清单；本 spec 一份文档覆盖三阶段，但实现分阶段，S1 不被 Chrome 运维阻塞。

## Key Decisions（设计时确认）

1. **`calc_points` 用 Go 工具**（与 skill 原文唯一偏差：机械换算→Go，判断→LLM）。理由：客户可见点位、LLM 小数算术不可靠、沿用喜报数字零例外。
2. **文案 = S1 普通文本**，「复制按钮」延后。
3. **自写最小纯 Go docx 写入器**（unioffice 商业/fumiama AGPL/MIT 库不全，用户已选自写避 license）。
4. **skill 内容 `load_skill` 按需加载**（非 systemPrompt 常驻），fallback 为提升进 systemPrompt。
5. **参数来自用户对话**（非 DB），按 skill 原文；与喜报（DB 锁死）是不同业务场景。
6. **v1 不归档文案**，规避喜报归档矛盾；日后归档则服务端重算。

## Out of Scope (v1)

- 文案 artifact 归档 / content_hash / 可复现留痕（日后另开，且须服务端重算）。
- 「复制长版/短版」按钮卡（延后增强）。
- DB 拉参数（用户已确认走对话式收参）。
- 喜报与文案的联动（同一产品既能出喜报又能出文案）——未来可加。
