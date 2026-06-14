# 持有产品分析 — 设计规格

## 概述

将原"存续产品分析"（`OngoingProduct.vue`）改造为"持有产品分析"，包含"产品分析"和"客户持有分析"两个通过标签页切换的子页面。数据来源直接同步航班服务交易总表（产品表 + 交易表），移除原有的客户端 Excel 文件上传方案。

## 页面结构

- 路由：`/holding-analysis`（原 `/ongoing-product`）
- 侧边栏名称：`持有分析`
- 容器页面：`HoldingAnalysis.vue`，含两个标签页切换
  - **产品分析**（默认）→ `ProductAnalysis.vue`
  - **客户持有分析** → `CustomerHolding.vue`
- 删除旧的 `OngoingProduct.vue`

---

## 一、产品分析

### 1.1 数据源

航班服务交易总表 → **产品表**，通过后端 API `GET /api/holding/products` 获取。

### 1.2 字段映射（24列）

| 页面列名 | 数据库字段 | 取值逻辑 |
|---------|-----------|---------|
| 航班编号 | id | 直接读取 |
| 管理人 | manager | 直接读取（私募管理人） |
| 产品名称 | name | 直接读取 |
| 持有状态 | holding_status | 直接读取 |
| 申购日期 | issue_date | 直接读取（认购日） |
| 存续时间（月） | duration_months + duration_days | 完结：直接读取 duration_months；存续：由后端用 duration_days÷30.0 计算，保留1位小数 |
| 结构类型 | structure_type | 直接读取 |
| 标的 | code | 读取 code，省略括号内内容 |
| 锁定期 | lock_months | 直接读取 |
| 保证金比例 | margin_ratio | 直接读取（新增字段） |
| 敲入 | — | 直接读取，新增独立字段 |
| 首月敲出 | first_knockout_ratio | 读取（敲出字段） |
| 入场价 | entry_price | 直接读取 |
| 是否敲入 | knocked_in | 直接读取（新增字段） |
| 每月递减 | monthly_decrease | 直接读取 |
| 托管券商 | custodian | 直接读取（新增字段） |
| 交易对手 | counterparty | 直接读取（新增字段） |
| 期限 | term | 直接读取 |
| 完结时间 | complete_date | 直接读取 |
| 降落伞 | parachute | 直接读取 |
| 派息障碍 | dividend_barrier | 直接读取 |
| 月票息（税费后） | monthly_coupon | 直接读取 |
| 第一段票息（税费后） | coupon_1st | 直接读取 |
| 第二段票息（税费后） | coupon_2nd | 直接读取 |
| 第三段票息（税费后） | coupon_3rd | 直接读取 |
| 绝对收益率 | absolute_return | 直接读取 |

> **新增映射字段：** `duration_days`（存续天数）、`knocked_in`（是否敲入）、`margin_ratio`（保证金比例）、`custodian`（托管券商）、`counterparty`（交易对手）。这些字段在产品表 sheet 中有独立列，当前同步逻辑未涵盖，需补充。

### 1.3 筛选器

**基础筛选项（始终可见）：**
- **申购日期**：起止日期选择器，对应 `issue_date`
- **产品状态**：下拉单选，选项动态读取 `holding_status` 去重值
- **管理人**：下拉单选，选项动态读取 `manager` 去重值
- **完结时间**：起止日期选择器，对应 `complete_date`

**高级筛选项（折叠，点击展开）：**
- **标的**：下拉单选，选项动态读取 `code`（省略括号内容）去重值
- **结构类型**：下拉单选，取 `structure_type` 去重值
- **锁定期**：下拉单选，取 `lock_months` 去重值
- **保证金比例**：下拉单选，取 `margin_ratio` 去重值

### 1.4 表格交互

- 水平滚动，`航班编号` 列固定最左
- 持有状态用彩色标签展示（存续=绿色，完结=橙色）
- 支持分页

---

## 二、客户持有分析

### 2.1 数据源

航班服务交易总表 → **交易表**（主数据，含客户交易详情）+ **产品表**（补充字段如入场价、敲出等）。通过后端 API `GET /api/holding/transactions` 获取。

### 2.2 字段映射

| 页面列名 | 数据源 → 字段 | 取值逻辑 |
|---------|-------------|---------|
| 产品名字 | 交易表 → product_name | 直接读取 |
| 姓名 | 交易表 → name | 直接读取 |
| 实际申购人 | 交易表 → actual_buyer | 直接读取 |
| 金额/万 | 交易表 → amount | 直接读取 |
| 申购费返还比例 | 交易表 → subscribe_fee_ratio | 直接读取 |
| 管理费返还比例 | 交易表 → management_fee_ratio | 直接读取 |
| 业绩报酬返还比例 | 交易表 → performance_fee_ratio | 直接读取 |
| 返还对象 | 交易表 → rebate_target | 直接读取 |
| 申购日期 | 交易表 → flight_date | 直接读取（航班日期） |
| 存续状态 | 交易表 → holding_status | 直接读取 |
| 完结日期 | 交易表 → complete_date | 直接读取 |
| 挂钩标的 | 交易表 → underlying | 直接读取 |
| 结构类型 | 交易表 → structure_type | 直接读取 |
| 锁定期 | 交易表 → lock_period | 直接读取 |
| 观察日 | 后端计算 | 见 2.3 观察日计算逻辑 |
| 入场价 | 产品表 → entry_price | 通过航班编号关联 |
| 首月敲出 | 产品表 → first_knockout_ratio | 通过航班编号关联（敲出字段） |
| 每月降敲 | 产品表 → monthly_decrease | 通过航班编号关联（每月递减） |
| 敲出价 | 后端计算 | 见 2.4 敲出价计算逻辑 |
| 今日点位 | prices.price_cache | 实时获取东方财富点位 / 每日15:05缓存 |
| 敲出线以上/下 | 后端计算 | 敲出价 vs 今日点位 |
| 降落伞 | 产品表 → parachute | 通过航班编号关联 |
| 派息障碍（如有） | 交易表 → dividend_barrier | 直接读取 |
| 月票息（税费后） | 交易表 → monthly_coupon | 直接读取 |
| 第一段票息（税费后） | 交易表 → coupon_1st | 直接读取 |
| 第二段票息（税费后） | 产品表 → coupon_2nd | 通过航班编号关联 |
| 第三段票息（税费后） | 产品表 → coupon_3rd | 通过航班编号关联 |

### 2.3 观察日计算逻辑

```
输入：issue_date（认购日/申购日期）
输出：下一个观察日（含当日）+ 观察类型

规则：
1. 若存续状态 = "完结"，显示"已完结"
2. 产品入场起，每满一个月的同一天为观察日
   例：3月6日入场 → 4月6日、5月6日、6月6日…
3. 遇节假日顺延（复用现有 trading/calendar.go 逻辑）
4. 存续月份 < 锁定期月份 → 不观察敲出，只观察派息
5. 存续月份 ≥ 锁定期月份 → 同时观察派息和敲出
6. 若月票息（税费后）为空 → 不观察派息
7. 展示包含当日及之后最近的一个观察日
   标注：派息 / 敲出 / 派息/敲出
```

### 2.4 敲出价计算逻辑

```
完结产品：显示最后一个观察日的敲出价格
存续产品：入场价 × 当月敲出比例

敲出比例计算：
  当月敲出比例 = 首月敲出比例 - (存续月份 - 锁定期月份) × 每月递减比例
  当存续月份 < 锁定期月份时，不观察敲出（显示"—"）

存续月份取值：下一个敲出观察日时的累计存续月份
  例：已过第1个观察日，未到第2个 → 存续月份 = 2
```

### 2.5 过滤器

**基础筛选项：**

- **客户**：文本输入框 + "姓名"/"实际申购人" 复选框
  - 两个都选中 = 满足其一
  - 只选一个 = 仅满足该选项
- **观察日**：起止日期选择 + "派息"/"敲出" 复选框
  - 在选定时间段内观察派息/敲出的记录
  - 逻辑同上
- **返还对象**：下拉单选，取 `rebate_target` 去重值
- **存续状态**：下拉单选，取 `holding_status` 去重值

**高级筛选项（折叠）：**
- **产品名字**：文本输入框，模糊匹配
- **申购日期**：起止日期选择器，对应 `flight_date`
- **完结日期**：起止日期选择器，对应 `complete_date`

---

## 三、后端变更

### 3.1 数据库 Schema 新增字段

**products 表新增列：**
```sql
ALTER TABLE products ADD COLUMN duration_days INTEGER;
ALTER TABLE products ADD COLUMN knocked_in TEXT;
ALTER TABLE products ADD COLUMN margin_ratio REAL;
ALTER TABLE products ADD COLUMN custodian TEXT;
ALTER TABLE products ADD COLUMN counterparty TEXT;
```

**transactions 表扩展列：**
```sql
ALTER TABLE transactions ADD COLUMN product_name TEXT;
ALTER TABLE transactions ADD COLUMN name TEXT;
ALTER TABLE transactions ADD COLUMN actual_buyer TEXT;
ALTER TABLE transactions ADD COLUMN amount REAL;
ALTER TABLE transactions ADD COLUMN subscribe_fee_ratio REAL;
ALTER TABLE transactions ADD COLUMN management_fee_ratio REAL;
ALTER TABLE transactions ADD COLUMN performance_fee_ratio REAL;
ALTER TABLE transactions ADD COLUMN rebate_target TEXT;
ALTER TABLE transactions ADD COLUMN flight_date TEXT;
ALTER TABLE transactions ADD COLUMN holding_status TEXT;
ALTER TABLE transactions ADD COLUMN complete_date TEXT;
ALTER TABLE transactions ADD COLUMN underlying TEXT;
ALTER TABLE transactions ADD COLUMN structure_type TEXT;
ALTER TABLE transactions ADD COLUMN lock_period TEXT;
ALTER TABLE transactions ADD COLUMN dividend_barrier REAL;
ALTER TABLE transactions ADD COLUMN monthly_coupon REAL;
ALTER TABLE transactions ADD COLUMN coupon_1st REAL;
```

### 3.2 数据同步增强

- `mapProductSheetRows`：增加 `duration_days`、`knocked_in`、`margin_ratio`、`custodian`、`counterparty` 的列映射
- `mapTransactionSheetRows`：扩展完整字段映射，读取交易表所有指定列
- 飞书读取交易表的行参数从 34 提高到足够大（如 500 行）

### 3.3 新增 API

| 端点 | 方法 | 描述 |
|------|------|------|
| `/api/holding/products` | GET | 产品分析数据，支持所有筛选参数 |
| `/api/holding/transactions` | GET | 客户持有分析数据，含观察日/敲出价计算 |
| `/api/holding/price/:code` | POST | 手动刷新某标的今日点位 |

### 3.4 计算逻辑（后端实现）

- 观察日计算：复用 `trading/calendar.go` 的交易日逻辑
- 敲出价计算、存续月份计算：在查询时动态计算并附加到交易记录响应中
- 今日点位：复用 `prices/client.go` 的东方财富 API

---

## 四、前端变更

### 4.1 文件结构

| 操作 | 文件路径 | 说明 |
|------|---------|------|
| 删除 | `views/OngoingProduct.vue` | 旧版客户端 Excel 分析 |
| 新建 | `views/HoldingAnalysis.vue` | 容器页面 + 标签页切换 |
| 新建 | `views/ProductAnalysis.vue` | 产品分析子页面 |
| 新建 | `views/CustomerHolding.vue` | 客户持有分析子页面 |
| 修改 | `router/index.ts` | 更新路由 `/holding-analysis` |
| 修改 | `components/SidebarNav.vue` | 更新导航名称和路径 |

### 4.2 交互

- 标签切换：使用 Vue 组件状态，不改变 URL（或可选使用 query 参数）
- 筛选器：自定义组件（项目无 UI 库），复用现有 CSS 变量
- 日期选择：使用原生 `<input type="date">`
- 下拉筛选：自定义 select 组件，选项通过 API 获取
- 高级选项折叠：CSS transition + v-show 控制
- 数据表格：水平滚动，首列 sticky，分页
