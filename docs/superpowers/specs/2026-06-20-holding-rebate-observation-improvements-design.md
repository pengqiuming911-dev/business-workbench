# 持仓/返费/观察日历 多项改进 设计

- 日期：2026-06-20
- 分支：fix/router-rebate-bugs
- 涉及：前端 `ProductAnalysis.vue`、`CustomerHolding.vue`、`ProductCompletion.vue`、`RebatePending.vue`（及 `RebateCompleted.vue`、全局 CSS）；后端 `router.go`、`repository.go`、`models.go`、`observations/calendar.go`、`prices/client.go`

## 背景与目标

围绕「产品&持仓」「返费」「观察日历」三个模块，落地多项改进：数值格式化、完结产品日历与状态筛选、筛选区布局与横滑透出修复、驯鹿48号数据订正、观察概览降敲列、全局表格对齐规范、待返费校验列、无返费订单过滤、扣税比例空值标记、是否可返自动判定。多数为前端调整，第 2/4/7/8/9/10 项涉及后端。

### 返费数据源定义（统一口径）

- **最新返款明细** = 飞书返费文件夹（`rebateFolderToken`）里日期最新的返款明细工作簿（文件名含「返款明细」/「返费明细」+ 日期）。
- 其内 **「待返」sheet** 是扣税比例（申购费/管理费/业绩报酬扣税）、管理费实收、业绩报酬应收、以及订单是否可返的唯一数据源（即现行 `fetchDaifanRows` 读取的对象）。
- 第 7/9/10 项均基于该工作簿；具体列位（未返三费、订单号列等）实现时读已存 `rebate_detail_data.raw_json` 或拉取该 sheet 确认。


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

- 在 `rebatePending` 里加载最新返款明细工作簿的「待返」sheet（即现行 `fetchDaifanRows`/`loadDaifanTaxRatios` 已读取的同一份数据，复用缓存），按订单号解析其「未返 申购费/管理费/业绩报酬」，产出 `check_subscribe/management/performance`（"T"/"F"）挂到每行。
- ⚠️ 实现时先确认「待返」sheet 里「未返三费」与「订单号」的具体列位（现行解析只取了扣税比例/管理费实收/业绩报酬应收列，未返列需补解析）。

### 前端

- 渲染 T/F（T 绿、F 红），左对齐（符合第 6 项）。
- CSV 导出补「校验-申购费/管理费/业绩报酬」三列。

## 8. 无返费订单不展示在返费明细

- 规则：若订单的 申购费返还比例、管理费返还比例、业绩报酬返还比例（`subscribe_fee_ratio / management_fee_ratio / performance_fee_ratio`）**三者均为 0 或空**，则该订单不需要返费，不展示在待返费分析页（`/api/rebate/pending`）。
- 落地：在 `rebatePending`（或 `QueryPendingRebates`）按上述三比例过滤剔除。注意「暂不可返」（未在待返 sheet 区域出现）但比例非全 0 的订单仍保留（见第 10 项）。

## 9. 扣税比例 搜不到标「-」

- 扣税比例（申购费/管理费/业绩报酬扣税）来源不变：仍取自最新返款明细工作簿的「待返」sheet。
- 若订单的某项扣税比例在该 sheet 对应区域搜不到，则该扣税比例单元格显示「-」（空值），**不再显示「暂不可返」**。
- 内部仍保留「该费用是否可返」的判定（用于应返=0），仅展示文案由「暂不可返」改为「-」。
- `fmtPct` 等格式化函数相应调整：值为「暂不可返/不可返」占位时输出「-」。

## 10. 是否可返 自动判定 + 已返订单排除

- **是否可返** 改为按「待返」sheet 是否能搜到该订单自动判定（取代现行手动 是/否 点选）：
  - 订单在交易表有「返还对象」且有对应「返还比例」，且在最新返款明细「待返」sheet 能搜到 → **待返**。
  - 在交易表有返还对象+返还比例，但在「待返」sheet 搜不到 → **暂不可返**。
- **已返订单排除**：订单各项费用均已返（`outstanding_subscribe/management/performance` 均 ≤ 0，即进入已返费分析、无未返金额），则不在待返费分析页展示。
  - 排除条件要求订单「可返（在待返 sheet 出现）」且三费未返均 ≤ 0；「暂不可返」订单（不在 sheet）即便未返为 0 仍保留展示，以便用户看到其不可返状态。
  - 与第 8 项互补：第 8 项剔除「三比例全 0/空」的无需返费订单；本项剔除「可返且已全返」的订单。
- **是否可返替换手动是/否**（已确认）：是否可返由上述规则自动判定为「待返/暂不可返」，**替换**现行手动 是/否 点选与 `is_returnable` 存储值；筛选下拉选项由「全部/是/否」改为「全部/待返/暂不可返」；`是否可返` 列由可点按钮改为只读展示。`本次拟返` 勾选与「标记已返」流程保留。
- 落地：在 `rebatePending` 计算每订单的 `is_returnable`（待返/暂不可返）并应用排除；前端去掉 `toggleReturnable` 的写操作、调整筛选选项与列渲染。

---

## 影响面与风险

- 第 2 项新增 `status` 参数与 `SpotPrice` 字段，前端旧调用缺省 `ongoing`，向后兼容。
- 第 4 项结论依赖 DB 实际数据，可能改数据或改 `prices` client；需重新生成观察记录。
- 第 6 项全局靠左会影响所有数值列视觉，按用户确认执行。
- 第 7 项返款明细「未返」列映射需实现时确认；若 sheet 结构不符预期，回退为「校验列显示 `--`」不阻断主流程。
- 第 8/10 项过滤会减少返费明细行数（无需返费订单、已全返订单），属预期行为。
- 第 10 项确认替换手动 是/否，会移除现行 `is_returnable` 手动写操作与筛选选项；属行为变更。

## 验证

- 第 1 项：两页首月敲出显示 2 位小数。
- 第 2 项：存续卡片显示 敲出/派息/今日；已完结卡片显示 敲出/当日；状态切换正常。
- 第 3 项：产品分析筛选同行不换行；客户持有高级筛选折叠；横滑无透出。
- 第 4 项：驯鹿48号 2026-06-22 敲出价≈0.766、标的价格≈0.580。
- 第 5 项：全量表头无透出；新增降敲列显示每月递减值。
- 第 6 项：两行表头首行居中；所有表格字段值靠左。
- 第 7 项：校验列 T/F 与手工比对一致；CSV 含三列。
- 第 8 项：三比例全 0/空的订单不出现在返费明细。
- 第 9 项：扣税比例搜不到的单元格显示「-」。
- 第 10 项：是否可返按待返 sheet 命中显示「待返」/未命中「暂不可返」；可返且已全返的订单不展示。
