# 持仓/返费/观察日历 多项改进 设计

- 日期：2026-06-20
- 分支：fix/router-rebate-bugs
- 涉及：前端 `ProductAnalysis.vue`、`CustomerHolding.vue`、`ProductCompletion.vue`、`RebatePending.vue`（及 `RebateCompleted.vue`、全局 CSS）；后端 `router.go`、`repository.go`、`models.go`、`observations/calendar.go`、`prices/client.go`

## 背景与目标

围绕「产品&持仓」「返费」「观察日历」三个模块，落地 8 项改进：数值格式化、完结产品日历与状态筛选、筛选区布局与横滑透出修复、驯鹿48号数据订正、观察概览降敲列、全局表格对齐规范、待返费校验列、以及无返费订单过滤。多数为前端调整，第 2/4/7/8 项涉及后端。

---

## 1. 首月敲出 保留2位小数（产品分析页 + 客户持有分析页）

- 纯前端。`FirstKnockoutRatio` 后端存的是小数比例（如 `1.0347`），不动后端。
- 两个页面「首月敲出」列改用格式化函数 `formatRatio2(val)`：`val==null → '--'`，否则 `Number(val).toFixed(2)`（四舍五入）。
  - `ProductAnalysis.vue` 首月敲出列。
  - `CustomerHolding.vue` 首月敲出列。
- 不改变「每月递减」「入场价」等其他列的展示。

## 2. 完结产品日历 + 状态筛选 + 今日实时点位（观察日历 tab）

观察日历 tab（`ProductCompletion.vue` 的 `activeTab === 'calendar'`）工具栏新增「状态」筛选：存续 / 已完结，默认存续。日历卡片随状态切换末行：

| 状态 | 卡片行 | 末行含义 |
|---|---|---|
| 存续 | 敲出 / 派息 / 今日 | 今日实时价（整月同一值） |
| 已完结 | 敲出 / 当日 | 该观察日当天收盘价 |

### 后端

- 扩展 `GET /api/observations/calendar?month=YYYY-MM&status=ongoing|completed`（`status` 缺省 `ongoing`）。
- `model.CalendarProduct` 增字段 `SpotPrice *float64 json:"spot_price"`。
- **存续**：`spot_price` = 今日实时价。优先读 DB 缓存 `PriceByDate(code, today)`；缺失的 code 才调 `prices.FetchAll` 实时拉取并 `UpsertPrice` 回写缓存——避免每次打开日历都串行打外部接口。
- **已完结**：
  - 新增 `Store.QueryCompletedProducts()`：`SELECT * FROM products WHERE holding_status LIKE '%完结%'`。
  - 观察日仍用 `observations.DatesForMonth` 计算；敲出价用 `ComputeKnockoutPrice`；`spot_price` = 该日已存观察记录的 `underlying_price`（`QueryObservationsByProduct` 按日期匹配）；无记录则 `nil`。
- `observations.CalendarForMonth` 按状态分别构建；已完结分支需把每产品的观察记录 map 传入以匹配当日收盘价。

### 前端

- 工具栏加状态单选（存续/已完结）；切换时带 `status` 重新 `loadCalendarData`。
- 卡片末行：存续显示 `今日`+`spot_price`，已完结显示 `当日`+`spot_price`；标签按 `status` 决定，复用 `spot_price` 字段。
- 存续卡片保留原 `敲出`、`派息` 两行，新增 `今日` 行。

## 3. 筛选区调整 + 高级筛选折叠 + 横滑透出修复

### 3a 产品分析页筛选字体

- 缩小 `ProductAnalysis.vue` 筛选项字体：`.filter-group label` 15→13px、`.input-sm` 15→13px、高度 36→32、`.filter-bar` gap/padding 收紧。
- 目标：申购日期/完结日期/持有状态/管理人 同处一行不换行。管理人位置不变。

### 3b 客户持有分析页高级筛选折叠

- `CustomerHolding.vue` 新增「高级筛选」折叠区（复用 `ProductAnalysis.vue` 的 `advanced-toggle` + `advanced-bar` 结构；`showAdvanced` ref 已存在但模板未接）。
- 移入折叠区：申购日期、完结日期、观察日（含 派息/敲出 复选框）。
- 主筛选条保留：客户 / 持有状态 / 返佣对象 / 产品名称。

### 3c 横滑透出修复（客户持有分析页，核对产品分析页）

- 用 systematic-debugging 复现定位根因。候选：sticky 列背景半透明 / `border-collapse` / z-index / sticky 列间间隙。
- 修复并核对产品分析页是否同问题（产品分析页已用 `border-collapse: separate`，重点查背景不透明与 z-index）。

## 4. 驯鹿48号 数据订正（debug & fix 根因）

`ComputeKnockoutPrice` 公式 `entry × (首月敲出比例 − (存续月−锁定期)×每月递减)` 与用户给出的公式一致，0.573≠0.766 是数据/入参问题。按 systematic-debugging：

- 查 DB：`鹿秀驯鹿48号` 的 `issue_date / lock_months / first_knockout_ratio / monthly_decrease / code`，及 2026-06-22 的 stored observation（`months_since_entry / knockout_price / underlying_price`）。
- 候选根因：
  - (a) `issue_date` 存错致 `monthsSinceEntry` 偏大（0.573 对应≈56 月，而非用户预期的 5）。
  - (b) 恒科ETF 价格源返回 0.601 而非 0.580：`prices` client 对该 code 的 `secid`/除数解析；恒科ETF 应为 `513180.SH` → `HasPrefix("513")` → `/1000`。
  - (c) 该日 `underlying_price` 为空（`priceForObservation` 缓存与实时均未命中）。
- 确认根因后改代码或改数据，再 `POST /api/observations/generate` 重新生成观察记录。未确认前不声称已修复。

## 5. 存续产品观察概览（全量 tab）：透出修复 + 降敲列

### 5a 表头透出修复

- `ProductCompletion.vue` `.overview-table` 现用 `border-collapse: collapse`（横滑表头透出主因）→ 改 `border-collapse: separate; border-spacing: 0`，并确保 sticky 列/表头背景不透明、z-index 正确（对齐已修好的 holding 表写法）。

### 5b 降敲列

- 在「标的价格」列后新增「降敲」列，直接读 `p.monthly_decrease`（产品表「每月递减」字段，`productObservationPayload` 已包含该字段）。
- 展示格式与产品分析页「每月递减」列一致。

## 6. 全局表格对齐规范

- **两行表头**：第一行（分组表头，如 `header-group-row`）字段居中；第二行子表头保持原样。
- **所有字段值靠左**：产品&持仓、返费、观察日历-全量 等全局表格的 `td`（含原 `num-col` 右对齐的数值列）改为左对齐。
- 落地：以共享/全局 CSS 为主（`.data-table td { text-align:left }`、`.header-group-row th { text-align:center }`），逐表核对覆盖：`RebatePending.vue`、`RebateCompleted.vue`、`ProductAnalysis.vue`、`CustomerHolding.vue`、`ProductCompletion.vue` 概览表。单行表头不受「第一行居中」影响，字段值仍靠左。

## 7. 待返费分析页：本次拟返缩窄 + 校验列

### 本次拟返缩窄

- `RebatePending.vue` 本次拟返组下 申购费/管理费/业绩报酬 三列 `min-width` 由 100px 收窄至 ~64–72px；合计列保留。

### 新增「校验」列组

- 表头：在「是否可返」(`rowspan=2`) 之后、「本次拟返」(`colspan=4`) 之前，插入「校验」`colspan=3`，子列 申购费/管理费/业绩报酬。
- 每格 T/F：比对当前页计算的 **未返**（`outstanding_subscribe/management/performance`）与 **最新返款明细 sheet** 里同订单同费用的未返金额；一致→T，不一致→F（浮点容差 0.01）。

### 后端

- 在 `rebatePending` 里加载最新返款明细 sheet：取飞书返费文件夹（`rebateFolderToken`）里日期最新的、名称含「返款明细」/「返费明细」的电子表格，选其「返款明细」sheet 读原始行（仿 `fetchDaifanRows` 选「待返」sheet 的写法；亦可优先读已存 `rebate_detail_data.raw_json`，缺失再拉取）。
- 按订单号解析未返 申购费/管理费/业绩报酬，产出 `check_subscribe/management/performance`（"T"/"F"）挂到每行。
- ⚠️ 实现时先读已存 `raw_json`（或拉取该 sheet）确认「未返三费」与「订单号」的具体列位再解析。

### 前端

- 渲染 T/F（T 绿、F 红），左对齐（符合第 6 项）。
- CSV 导出补「校验-申购费/管理费/业绩报酬」三列。

## 8. 无返费订单不展示在返费明细

- 规则：若订单的 申购费返还比例、管理费返还比例、业绩报酬返还比例（`subscribe_fee_ratio / management_fee_ratio / performance_fee_ratio`）**三者均为 0 或空**，则该订单不需要返费，不展示在待返费分析页（`/api/rebate/pending`）。
- 落地：在 `rebatePending`（或 `QueryPendingRebates`）按上述三比例过滤剔除。注意「暂不可返」（未在待返 sheet 区域出现）但比例非全 0 的订单仍保留。

---

## 影响面与风险

- 第 2 项新增 `status` 参数与 `SpotPrice` 字段，前端旧调用缺省 `ongoing`，向后兼容。
- 第 4 项结论依赖 DB 实际数据，可能改数据或改 `prices` client；需重新生成观察记录。
- 第 6 项全局靠左会影响所有数值列视觉，按用户确认执行。
- 第 7 项返款明细列映射需实现时确认；若 sheet 结构不符预期，回退为「校验列显示 `--`」不阻断主流程。
- 第 8 项过滤会减少返费明细行数，属预期行为。

## 验证

- 第 1 项：两页首月敲出显示 2 位小数。
- 第 2 项：存续卡片显示 敲出/派息/今日；已完结卡片显示 敲出/当日；状态切换正常。
- 第 3 项：产品分析筛选同行不换行；客户持有高级筛选折叠；横滑无透出。
- 第 4 项：驯鹿48号 2026-06-22 敲出价≈0.766、标的价格≈0.580。
- 第 5 项：全量表头无透出；新增降敲列显示每月递减值。
- 第 6 项：两行表头首行居中；所有表格字段值靠左。
- 第 7 项：校验列 T/F 与手工比对一致；CSV 含三列。
- 第 8 项：三比例全 0/空的订单不出现在返费明细。
