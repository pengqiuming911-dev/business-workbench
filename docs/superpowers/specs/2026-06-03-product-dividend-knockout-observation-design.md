# 产品派息/敲出观察功能设计

**日期**: 2026-06-03  
**目标**: 将 `/product-completion` 页面从"完结与发行"报告替换为"产品派息/敲出观察"功能，展示存续产品的派息和敲出观察情况。

---

## 1. 概述

完全替换现有 `/product-completion` 路由对应的 `ProductCompletion.vue` 页面。新页面展示所有**存续产品**（由飞书产品表"持有状态"字段判断）的派息/敲出观察情况，包含：

- 每个存续产品的基础信息（从飞书产品表读取）
- 每个观察日的标的价格与敲出/派息判定结果
- 后台定时 + 手动刷新的标的价格获取机制
- 历史观察记录可追溯

## 2. 数据层

### 2.1 扩展 `products` 表

由于项目使用 sql.js（内存 SQLite），不支持 `ALTER TABLE ADD COLUMN`。实际实现是修改建表 SQL，重建数据库（首次同步时自动重建）。

新增字段：

| DB 列名 | 类型 | 说明 |
|---------|------|------|
| `manager` | TEXT | 私募管理人 |
| `holding_status` | TEXT | 持有状态 |
| `structure_type` | TEXT | 结构类型 |
| `code` | TEXT | 标的指数代码（如 sh000300） |
| `lock_days` | INTEGER | 锁定期天数 |
| `lock_months` | INTEGER | 锁定期月数（floor(天数/30)） |
| `first_knockout_ratio` | REAL | 首月敲出比例 |
| `entry_price` | REAL | 入场价 |
| `monthly_decrease` | REAL | 每月递减比例 |
| `term` | TEXT | 期限 |
| `parachute` | TEXT | 降落伞 |
| `dividend_barrier` | REAL | 派息障碍比例 |
| `monthly_coupon` | REAL | 月票息（税后） |
| `coupon_1st` | REAL | 第一段票息（税后） |
| `coupon_2nd` | REAL | 第二段票息（税后） |
| `coupon_3rd` | REAL | 第三段票息（税后） |
| `duration_months` | REAL | 存续时间（月） |
| `absolute_return` | REAL | 绝对收益率 |
| `holiday_adjust` | TEXT | 观察日节假日顺延/提前 |

建表 SQL：
```sql
CREATE TABLE IF NOT EXISTS products (
  id TEXT PRIMARY KEY,
  name TEXT,
  is_main INTEGER,
  issue_date TEXT,
  complete_date TEXT,
  subscribe_amount REAL,
  outstanding_amount REAL,
  manager TEXT,
  holding_status TEXT,
  structure_type TEXT,
  code TEXT,
  lock_days INTEGER,
  lock_months INTEGER,
  first_knockout_ratio REAL,
  entry_price REAL,
  monthly_decrease REAL,
  term TEXT,
  parachute TEXT,
  dividend_barrier REAL,
  monthly_coupon REAL,
  coupon_1st REAL,
  coupon_2nd REAL,
  coupon_3rd REAL,
  duration_months REAL,
  absolute_return REAL,
  holiday_adjust TEXT,
  raw TEXT
);
```

### 2.2 新建 `observations` 表

```sql
CREATE TABLE IF NOT EXISTS observations (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  product_id TEXT,
  observation_date TEXT,
  knockout_price REAL,
  dividend_line REAL,
  underlying_price REAL,
  is_knocked_out TEXT,
  is_dividend TEXT,
  months_since_entry INTEGER,
  updated_at TEXT,
  UNIQUE(product_id, observation_date),
  FOREIGN KEY (product_id) REFERENCES products(id)
);
```

`product_id` 关联的是 `products.id`（航班编号），航班编号是产品唯一标识。

**唯一约束**: `(product_id, observation_date)` 组合唯一，同一产品同一观察日只有一条记录（upsert 语义）。

### 2.3 新建 `price_cache` 表

```sql
CREATE TABLE IF NOT EXISTS price_cache (
  code TEXT,
  price_date TEXT,
  price REAL,
  updated_at TEXT,
  PRIMARY KEY (code, price_date)
);
```

## 3. 飞书同步映射

### 3.1 扩展 `POST /api/db/sync` 产品表映射

在 `backend/index.js` 的同步逻辑中，扩展现有 `productRows` 映射：

| 飞书列名 | DB 字段 | 转换逻辑 |
|----------|---------|---------|
| 航班编号 | `id` | `String().trim()` |
| 产品名称 | `name` | 直读 |
| 是否主产品 | `is_main` | "是"→1, 否则→0 |
| 认购日 | `issue_date` | Excel 序列号→`excelDateToString()` |
| 完结时间 | `complete_date` | Excel 序列号→`excelDateToString()` |
| 认购金额 | `subscribe_amount` | `Number() \|\| 0` |
| 存续金额 | `outstanding_amount` | `Number() \|\| 0` |
| 私募管理人 | `manager` | 直读 |
| 持有状态 | `holding_status` | 直读 |
| 结构类型 | `structure_type` | 直读 |
| 代码 | `code` | 直读（标的指数代码） |
| 锁定期 | `lock_days` | `Number() \|\| 0`；`lock_months = Math.floor(lock_days / 30)` |
| 敲出首月的敲出比例 | `first_knockout_ratio` | `Number() \|\| 0` |
| 入场价 | `entry_price` | `Number() \|\| 0` |
| 每月递减 | `monthly_decrease` | `Number() \|\| 0` |
| 期限 | `term` | 直读 |
| 降落伞 | `parachute` | 直读 |
| 派息障碍 | `dividend_barrier` | `Number() \|\| 0` |
| 月票息（税费后） | `monthly_coupon` | `Number() \|\| 0` |
| 第一段票息（税费后） | `coupon_1st` | `Number() \|\| 0` |
| 第二段票息（税费后） | `coupon_2nd` | `Number() \|\| 0` |
| 第三段票息（税费后） | `coupon_3rd` | `Number() \|\| 0` |
| 存续时间（月） | `duration_months` | `Number() \|\| 0` |
| 绝对收益率 | `absolute_return` | `Number() \|\| 0` |
| 观察日节假日顺延/提前 | `holiday_adjust` | 直读（"顺延"或"提前"） |

### 3.2 同步后自动触发

同步完成后：
1. 提取所有存续产品的 `code` 字段（去重）。存续产品判定规则：`holding_status` 字段值包含"持有"（如"持有中"、"持有"等），即 `holding_status.includes('持有')` 为 true
2. 批量获取标的价格
3. 生成/更新观察记录

## 4. 标的价格获取

### 4.1 数据源

使用**东方财富**免费 API（无需 key）：

```
GET https://push2.eastmoney.com/api/qt/stock/get?secid={market}.{code}&fields=f43,f44,f45,f46,f47,f170
```

- `market`: 1=沪市指数, 0=深市指数
- 代码前缀映射: `sh` → market=1, `sz` → market=0
- 返回字段:
  - `f43`: 最新价（整数，需 ÷100）
  - `f170`: 涨跌幅

**注意**: 东方财富价格可能因格式不同有差异，实现时需要根据实际返回值做兼容处理。如果 `f43` 返回值已经是浮点数则不除 100。

### 4.2 获取策略

- 收集所有存续产品的 `code`（去重）
- 逐个调用东方财富 API
- 结果写入 `price_cache` 表（`code`, `today`, `price`, `now()`）
- 失败时记录日志并跳过该 code，不影响其他产品

## 5. 观察日计算逻辑

### 5.1 观察日生成规则

每个存续产品从入场日（`issue_date`，即认购日）起，**每满一个月的对应日期**为观察日：

```
entryDate = product.issue_date (例如 2025-03-06)
观察日序列: 2025-04-06, 2025-05-06, 2025-06-06, ...
```

如果当月天数不足（如 1月31日 → 2月没有31号），则取该月最后一天。

### 5.2 节假日调整

根据产品的 `holiday_adjust` 字段：
- **顺延**: 观察日遇非交易日，顺延至下一个交易日
- **提前**: 观察日遇非交易日，提前至前一个交易日
- 交易日判断: 使用 `exchange-trading-day` npm 包（内置 A 股交易日数据）。若该包不可用，则降级为简单规则——排除周末（周六、周日），不处理节假日，在代码注释中标注后续可增强

### 5.3 敲出价计算

```
knockoutPrice(product, monthsSinceEntry):
  if monthsSinceEntry < product.lock_months:
    return null  // 锁定期内不观察敲出
  currentRatio = product.first_knockout_ratio - (monthsSinceEntry - product.lock_months) * product.monthly_decrease
  return product.entry_price * currentRatio
```

### 5.4 派息线计算

```
dividendLine(product):
  return product.entry_price * product.dividend_barrier
```

### 5.5 monthsBetween 计算定义

```
monthsBetween(entryDate, obsDate):
  // obsDate 一定是 entryDate 的第 N 个满月对应日（由 5.1 生成），所以：
  // 直接计算年份差*12 + 月份差
  years = obsDate.year - entryDate.year
  months = obsDate.month - entryDate.month
  return years * 12 + months
```

例如：入场日 2025-03-06，观察日 2025-04-06 → 返回 1；观察日 2026-03-06 → 返回 12。

### 5.6 观察结果判定

```
evaluate(product, obsDate, underlyingPrice):
  monthsSinceEntry = monthsBetween(product.issue_date, obsDate)
  dividendLine = computeDividendLine(product)
  isDividend = underlyingPrice > dividendLine ? '是' : '否'

  knockoutPrice = computeKnockoutPrice(product, monthsSinceEntry)
  if knockoutPrice == null:
    isKnockedOut = '--'
  else:
    isKnockedOut = underlyingPrice > knockoutPrice ? '是' : '否'

  return { isDividend, isKnockedOut, dividendLine, knockoutPrice, monthsSinceEntry }
```

### 5.7 存续期与锁定期关系

- **存续期 < 锁定期**: 只观察派息，不观察敲出（敲出价=null，是否敲出="--"）
- **存续期 >= 锁定期**: 既观察派息，也观察敲出

## 6. 后端 API 端点

### 6.1 `GET /api/observations`

查询所有存续产品的观察记录。

**Query 参数**:
- `search`（可选）: 按产品名称/航班编号模糊搜索
- `code`（可选）: 按标的代码筛选

**响应**:
```json
{
  "products": [
    {
      "id": "HB2025001",
      "name": "某航班产品A",
      "manager": "XX私募",
      "holding_status": "持有中",
      "code": "sh000300",
      "entry_price": 3200.5,
      "first_knockout_ratio": 1.03,
      "lock_months": 6,
      "monthly_decrease": 0.005,
      "issue_date": "2025-03-06",
      "subscribe_amount": 5000000,
      "observations": [
        {
          "date": "2025-04-06",
          "knockout_price": null,
          "dividend_line": 3104.5,
          "underlying_price": 3250.0,
          "is_knocked_out": "--",
          "is_dividend": "是",
          "months_since_entry": 1
        }
      ]
    }
  ],
  "lastUpdated": "2026-06-03 15:30:00"
}
```

### 6.2 `POST /api/observations/generate`

生成/更新观察记录。触发流程：
1. 从 products 表取所有存续产品
2. 获取最新标的价格
3. 计算每个产品的观察日列表
4. 对每个观察日生成/更新 observations 记录

**响应**: `{ ok: true, generated: 15, updated: 3 }`

### 6.3 `POST /api/observations/refresh-prices`

仅刷新标的价格（不重新生成观察记录，只更新已有记录的 `underlying_price` 和结果）。

**响应**: `{ ok: true, refreshed: 5, failed: 0 }`

### 6.4 `GET /api/observations/products`

获取所有存续产品列表（概要信息，不含观察记录）。

**响应**: `{ products: [{ id, name, code, holding_status, ... }] }`

## 7. 前端页面

### 7.1 页面替换

- 路由 `/product-completion` 不变
- 完全替换 `ProductCompletion.vue` 文件内容
- 页面标题改为 "产品派息/敲出观察"
- 沿用 `SubPageLayout` 组件

### 7.2 页面布局

**顶部操作区**:
- 数据来源标识: "航班服务交易总表 · 产品表"
- 搜索框: 按产品名称/航班编号搜索
- "刷新价格"按钮: 调用 `POST /api/observations/refresh-prices`
- 最后更新时间显示

**主体表格**:

| 列名 | 说明 |
|------|------|
| 航班编号 | 产品唯一标识 |
| 产品名称 | 产品名 |
| 私募管理人 | 管理人名称 |
| 持有状态 | 持有中/已退出等 |
| 代码 | 标的指数代码 |
| 入场价 | 入场价格 |
| 入场日 | 认购日 |
| 存续月 | 从入场至今的月数 |
| 锁定期(月) | 锁定期月数 |
| 最近观察日 | 最近一次观察日日期 |
| 标的价格 | 最新获取的标的价格 |
| 敲出价 | 当前存续月对应的敲出价（锁定期内显示"--"） |
| 派息线 | 派息线价格 |
| 是否敲出 | 最近观察日的敲出结果（是/否/--） |
| 是否派息 | 最近观察日的派息结果（是/否） |

**可展开行**: 点击行展开该产品的所有历史观察日明细：
- 每行: 观察日日期 | 标的价格 | 敲出价 | 派息线 | 是否敲出 | 是否派息

**颜色规则**:
- 敲出=是 → 红色背景高亮 (`#FEF3E2` / `#C62828`)
- 派息=是 → 绿色 (`#E8F4EC` / `#2E7D45`)
- 敲出=否 → 浅灰
- 敲出=-- → 不显示（锁定期内）

### 7.3 交互
- 页面加载时自动调用 `GET /api/observations` 获取数据
- 搜索框实时过滤（前端过滤，无需请求后端）
- 表格支持横向滚动（字段多）
- 沿用现有 earth-tone 棕色系设计风格

## 8. 定时任务

### 8.1 依赖

新增 `node-cron` npm 包。

### 8.2 定时规则

每个工作日（周一至周五）的 **11:30、15:00、15:30** 各执行一次：

```
cron expressions:
  '30 11 * * 1-5'   // 11:30
  '0 15 * * 1-5'    // 15:00
  '30 15 * * 1-5'   // 15:30
```

### 8.3 执行流程

1. 从 products 表查询所有存续产品（`holding_status` 包含"持有"），提取 `code`（去重）
2. 对每个 code 调用东方财富 API 获取最新价格
3. 写入 `price_cache` 表
4. 判断今天是否为任何存续产品的观察日
5. 如果是，为对应产品生成/更新 `observations` 记录

### 8.4 错误处理

- 单个 code 价格获取失败不阻塞其他 code
- 整体失败记录日志并打印错误信息
- 定时任务异常不影响 Web 服务运行

## 9. 不变动的部分

- `GET /api/db/products` API 保留不变（可能有其他页面引用）
- 现有路由 `/product-completion` 路径不变
- `SubPageLayout` 组件不变
- 现有 CSS 设计系统（earth-tone 棕色系）不变

## 10. 影响范围

| 文件 | 变更类型 | 说明 |
|------|---------|------|
| `backend/db.js` | 修改 | 重建 products 表 + 新增 observations 和 price_cache 表；新增查询/写入函数 |
| `backend/index.js` | 修改 | 扩展 sync 映射；新增 4 个 observations API；新增定时任务 |
| `backend/package.json` | 修改 | 新增 `node-cron` 依赖 |
| `frontend/views/ProductCompletion.vue` | 重写 | 完全替换为派息/敲出观察页面 |
| `backend/data.sqlite` | 删除 | 首次需删除旧数据库，触发重建 |
