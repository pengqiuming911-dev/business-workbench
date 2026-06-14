# Agent Tools 扩展实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 为 AI Agent 新增 10 个只读查询工具，覆盖客户、交易、渠道、喜报、文档、统计、运维全域数据

**Architecture:** 扩展现有 `agentTools.js` 的 `TOOL_DEFINITIONS` 和 `executeTool` 函数，在 `db.js` 中新增 3 个查询函数。`agentService.js` 无需修改（自动从 `TOOL_DEFINITIONS` 读取）。

**Tech Stack:** Node.js/Express + SQLite (sql.js) + DeepSeek tool calling

**Spec:** `docs/superpowers/specs/2026-06-13-agent-tools-expansion-design.md`

---

## File Structure

| File | Action | Responsibility |
|------|--------|---------------|
| `backend/db.js` | Modify | 新增 `queryTransactions`, `getAllChannels`, `getAllDirectCustomerSources` 函数 |
| `backend/services/agentTools.js` | Modify | 新增 10 个 tool definition + execution function |
| `backend/services/agentTools.test.js` | Modify | 新增全部 10 个工具的测试 |

---

### Task 1: CRM 工具 - 搜索客户

**Files:**
- Modify: `backend/services/agentTools.js`
- Modify: `backend/services/agentTools.test.js`

- [ ] **Step 1: 在 agentTools.js 的 TOOL_DEFINITIONS 末尾添加 search_customers 工具定义**

在 `TOOL_DEFINITIONS` 数组最后一个元素 `}` 后（第96行前），添加逗号和新工具：

```js
  {
    type: 'function',
    function: {
      name: 'search_customers',
      description: '搜索合投客户/潜在客户资料。支持按姓名、微信、行业、是否专户、是否竞品等条件过滤。当用户询问客户信息、查找某行业客户、或查询某客户资料时使用。',
      parameters: {
        type: 'object',
        properties: {
          keyword: { type: 'string', description: '搜索关键词，匹配客户姓名、实际购买人、微信昵称' },
          industry: { type: 'string', description: '行业过滤，如"互联网"、"金融"' },
          is_dedicated: { type: 'string', description: '是否专户客户，可选 "是"、"否"、"all"(不限)' },
          is_competitor: { type: 'string', description: '是否竞品群，可选 "是"、"否"、"all"(不限)' },
        },
      },
    },
  },
```

- [ ] **Step 2: 在 agentTools.js 的 executeTool switch 中添加 case**

在 `case 'get_observation_calendar':` 之后添加：

```js
      case 'search_customers':
        return executeSearchCustomers(args)
```

- [ ] **Step 3: 在 agentTools.js 底部添加 executeSearchCustomers 函数**

```js
function executeSearchCustomers(args = {}) {
  const db = dbModule.db
  let sql = 'SELECT user_name, actual_buyer, phone, wechat, total_assets, risk_tolerance, industry, is_actual_deal, lead_source, is_dedicated_account, is_competitor FROM co_invest_users WHERE 1=1'
  const params = []

  const keyword = (args.keyword || '').trim()
  if (keyword) {
    sql += ' AND (user_name LIKE ? OR actual_buyer LIKE ? OR wechat LIKE ?)'
    const like = `%${keyword}%`
    params.push(like, like, like)
  }

  const industry = (args.industry || '').trim()
  if (industry && industry !== 'all') {
    sql += ' AND industry = ?'
    params.push(industry)
  }

  const isDedicated = (args.is_dedicated || '').trim()
  if (isDedicated && isDedicated !== 'all') {
    sql += ' AND is_dedicated_account = ?'
    params.push(isDedicated)
  }

  const isCompetitor = (args.is_competitor || '').trim()
  if (isCompetitor && isCompetitor !== 'all') {
    sql += ' AND is_competitor = ?'
    params.push(isCompetitor)
  }

  sql += ' ORDER BY id LIMIT 20'

  const rows = db.exec(sql, params)
  if (!rows[0]) return { count: 0, customers: [] }

  const columns = rows[0].columns
  const customers = rows[0].values.map(vals => {
    const obj = {}
    columns.forEach((col, i) => { obj[col] = vals[i] })
    return obj
  })

  return { count: customers.length, customers }
}
```

- [ ] **Step 4: 运行现有测试确认不破坏已有功能**

Run: `cd backend && node services/agentTools.test.js`
Expected: `agentTools tests passed`

- [ ] **Step 5: Commit**

```bash
git add backend/services/agentTools.js
git commit -m "feat: add search_customers agent tool"
```

---

### Task 2: CRM 工具 - 客户持仓查询

**Files:**
- Modify: `backend/services/agentTools.js`
- Modify: `backend/services/agentTools.test.js`

- [ ] **Step 1: 在 TOOL_DEFINITIONS 数组中添加 get_customer_products**

```js
  {
    type: 'function',
    function: {
      name: 'get_customer_products',
      description: '查询指定客户（实际购买人）关联的所有产品持仓信息，包括产品名称、认购金额、存续金额、持有状态等。当用户询问某客户买了哪些产品或持仓情况时使用。',
      parameters: {
        type: 'object',
        properties: {
          customer_name: { type: 'string', description: '实际购买人姓名' },
        },
        required: ['customer_name'],
      },
    },
  },
```

- [ ] **Step 2: 在 executeTool switch 中添加 case**

```js
      case 'get_customer_products':
        return executeGetCustomerProducts(args)
```

- [ ] **Step 3: 添加 executeGetCustomerProducts 函数**

```js
function executeGetCustomerProducts({ customer_name }) {
  if (!customer_name) return { error: 'customer_name is required' }

  const db = dbModule.db
  const links = db.exec(
    "SELECT product_id FROM customer_product_link WHERE actual_buyer LIKE ?",
    [`%${customer_name}%`]
  )
  if (!links[0] || links[0].values.length === 0) {
    return { customer_name, count: 0, products: [] }
  }

  const productIds = links[0].values.map(r => r[0])
  const placeholders = productIds.map(() => '?').join(',')
  const rows = db.exec(
    `SELECT id, name, subscribe_amount, outstanding_amount, holding_status, issue_date, manager FROM products WHERE id IN (${placeholders})`,
    productIds
  )

  if (!rows[0]) return { customer_name, count: 0, products: [] }

  const columns = rows[0].columns
  const products = rows[0].values.map(vals => {
    const obj = {}
    columns.forEach((col, i) => { obj[col] = vals[i] })
    return { product_id: obj.id, ...obj }
  })

  return { customer_name, count: products.length, products }
}
```

- [ ] **Step 4: 运行测试**

Run: `cd backend && node services/agentTools.test.js`
Expected: `agentTools tests passed`

- [ ] **Step 5: Commit**

```bash
git add backend/services/agentTools.js
git commit -m "feat: add get_customer_products agent tool"
```

---

### Task 3: CRM 工具 - 客户峰值分析

**Files:**
- Modify: `backend/services/agentTools.js`

- [ ] **Step 1: 在 TOOL_DEFINITIONS 数组中添加 get_customer_peak_analysis**

```js
  {
    type: 'function',
    function: {
      name: 'get_customer_peak_analysis',
      description: '分析客户的存续余额峰值与当前差额。用于发现存量下降最多的客户、评估客户流失风险。不传 customer_name 则返回所有客户的分析。',
      parameters: {
        type: 'object',
        properties: {
          customer_name: { type: 'string', description: '实际购买人姓名，不传则返回全部客户' },
        },
      },
    },
  },
```

- [ ] **Step 2: 在 executeTool switch 中添加 case**

```js
      case 'get_customer_peak_analysis':
        return executeGetCustomerPeakAnalysis(args)
```

- [ ] **Step 3: 添加 executeGetCustomerPeakAnalysis 函数**

```js
function executeGetCustomerPeakAnalysis(args = {}) {
  const allPeaks = dbModule.computeUserPeakBalances()
  let entries = Object.entries(allPeaks).map(([name, data]) => ({
    customer_name: name,
    peak_balance: Math.round(data.peak_balance * 100) / 100,
    current_outstanding: Math.round(data.current_outstanding * 100) / 100,
    peak_diff: Math.round(data.peak_diff * 100) / 100,
  }))

  const keyword = (args.customer_name || '').trim()
  if (keyword) {
    entries = entries.filter(e => e.customer_name.includes(keyword))
  }

  entries.sort((a, b) => b.peak_diff - a.peak_diff)

  return { count: entries.length, analyses: entries }
}
```

- [ ] **Step 4: 运行测试**

Run: `cd backend && node services/agentTools.test.js`
Expected: `agentTools tests passed`

- [ ] **Step 5: Commit**

```bash
git add backend/services/agentTools.js
git commit -m "feat: add get_customer_peak_analysis agent tool"
```

---

### Task 4: 交易工具 + db.js 扩展

**Files:**
- Modify: `backend/db.js`
- Modify: `backend/services/agentTools.js`

- [ ] **Step 1: 在 db.js 的 module.exports 前添加 queryTransactions 函数**

```js
function queryTransactions({ product_id, counterparty, start_date, end_date }) {
  let sql = 'SELECT transaction_date, flight_id, counterparty, subscribe_amount FROM transactions WHERE 1=1'
  const params = []

  if (product_id) {
    sql += ' AND flight_id = ?'
    params.push(product_id)
  }
  if (counterparty) {
    sql += ' AND counterparty LIKE ?'
    params.push(`%${counterparty}%`)
  }
  if (start_date) {
    sql += ' AND transaction_date >= ?'
    params.push(start_date)
  }
  if (end_date) {
    sql += ' AND transaction_date <= ?'
    params.push(end_date)
  }

  sql += ' ORDER BY transaction_date DESC LIMIT 100'
  return queryAll(sql, params)
}
```

- [ ] **Step 2: 在 db.js 的 module.exports 中添加 queryTransactions**

在 exports 对象中添加 `queryTransactions`。

- [ ] **Step 3: 在 TOOL_DEFINITIONS 添加 query_transactions**

```js
  {
    type: 'function',
    function: {
      name: 'query_transactions',
      description: '查询交易记录，支持按产品（航班编号）、交易对手、日期范围过滤。当用户问交易流水、认购记录或某产品的交易量时使用。',
      parameters: {
        type: 'object',
        properties: {
          product_id: { type: 'string', description: '航班编号（产品 ID）' },
          counterparty: { type: 'string', description: '交易对手关键词' },
          start_date: { type: 'string', description: '开始日期，格式 YYYY-MM-DD' },
          end_date: { type: 'string', description: '结束日期，格式 YYYY-MM-DD' },
        },
      },
    },
  },
```

- [ ] **Step 4: 在 executeTool switch 中添加 case**

```js
      case 'query_transactions':
        return executeQueryTransactions(args)
```

- [ ] **Step 5: 添加 executeQueryTransactions 函数**

```js
function executeQueryTransactions(args = {}) {
  const rows = dbModule.queryTransactions(args)
  const totalAmount = rows.reduce((sum, r) => sum + (r.subscribe_amount || 0), 0)
  return {
    count: rows.length,
    total_amount: Math.round(totalAmount * 100) / 100,
    transactions: rows,
  }
}
```

- [ ] **Step 6: 运行测试**

Run: `cd backend && node services/agentTools.test.js`
Expected: `agentTools tests passed`

- [ ] **Step 7: Commit**

```bash
git add backend/db.js backend/services/agentTools.js
git commit -m "feat: add query_transactions agent tool and db function"
```

---

### Task 5: 产品聚合统计工具

**Files:**
- Modify: `backend/services/agentTools.js`

- [ ] **Step 1: 在 TOOL_DEFINITIONS 添加 get_product_analytics**

```js
  {
    type: 'function',
    function: {
      name: 'get_product_analytics',
      description: '按指定维度聚合统计产品数据。支持按管理人(manager)、持有状态(holding_status)、结构类型(structure_type)、发行月份(issue_month)分组。返回每组的数量、认购总额、存续总额。',
      parameters: {
        type: 'object',
        properties: {
          group_by: { type: 'string', description: '分组维度', enum: ['manager', 'holding_status', 'structure_type', 'issue_month'] },
        },
        required: ['group_by'],
      },
    },
  },
```

- [ ] **Step 2: 在 executeTool switch 中添加 case**

```js
      case 'get_product_analytics':
        return executeGetProductAnalytics(args)
```

- [ ] **Step 3: 添加 executeGetProductAnalytics 函数**

```js
function executeGetProductAnalytics({ group_by }) {
  const validGroups = ['manager', 'holding_status', 'structure_type', 'issue_month']
  if (!validGroups.includes(group_by)) {
    return { error: `group_by must be one of: ${validGroups.join(', ')}` }
  }

  const db = dbModule.db
  let sql, groupExpr

  if (group_by === 'issue_month') {
    groupExpr = "substr(issue_date, 1, 7)"
    sql = `SELECT ${groupExpr} as grp, COUNT(*) as cnt, COALESCE(SUM(subscribe_amount), 0) as total_sub, COALESCE(SUM(outstanding_amount), 0) as total_out, SUM(CASE WHEN holding_status LIKE '%存续%' OR holding_status LIKE '%持有%' THEN 1 ELSE 0 END) as active_cnt FROM products WHERE issue_date IS NOT NULL AND issue_date != '' GROUP BY grp ORDER BY grp`
  } else {
    groupExpr = group_by
    sql = `SELECT ${groupExpr} as grp, COUNT(*) as cnt, COALESCE(SUM(subscribe_amount), 0) as total_sub, COALESCE(SUM(outstanding_amount), 0) as total_out, SUM(CASE WHEN holding_status LIKE '%存续%' OR holding_status LIKE '%持有%' THEN 1 ELSE 0 END) as active_cnt FROM products GROUP BY grp ORDER BY cnt DESC`
  }

  const rows = db.exec(sql)
  if (!rows[0]) return { group_by, groups: [] }

  const groups = rows[0].values.map(r => ({
    key: r[0] || '(空)',
    count: r[1],
    total_subscribe: Math.round(r[2] * 100) / 100,
    total_outstanding: Math.round(r[3] * 100) / 100,
    active_count: r[4],
  }))

  return { group_by, groups }
}
```

- [ ] **Step 4: 运行测试**

Run: `cd backend && node services/agentTools.test.js`
Expected: `agentTools tests passed`

- [ ] **Step 5: Commit**

```bash
git add backend/services/agentTools.js
git commit -m "feat: add get_product_analytics agent tool"
```

---

### Task 6: 喜报查询工具

**Files:**
- Modify: `backend/services/agentTools.js`

- [ ] **Step 1: 在 TOOL_DEFINITIONS 添加 get_posters**

```js
  {
    type: 'function',
    function: {
      name: 'get_posters',
      description: '查询产品喜报数据，包括绝对收益率、年化收益、派息次数等。支持按产品 ID 或观察日过滤。当用户问某产品的喜报、某天的喜报或收益统计时使用。',
      parameters: {
        type: 'object',
        properties: {
          product_id: { type: 'string', description: '产品 ID（航班编号）' },
          observation_date: { type: 'string', description: '观察日，格式 YYYY-MM-DD' },
        },
      },
    },
  },
```

- [ ] **Step 2: 在 executeTool switch 中添加 case**

```js
      case 'get_posters':
        return executeGetPosters(args)
```

- [ ] **Step 3: 添加 executeGetPosters 函数**

```js
function executeGetPosters(args = {}) {
  let posters
  if (args.product_id) {
    posters = dbModule.queryPostersByProduct(args.product_id)
  } else if (args.observation_date) {
    posters = dbModule.queryPostersByDate(args.observation_date)
  } else {
    posters = dbModule.queryAllPosters()
  }

  return {
    count: posters.length,
    posters: posters.map(p => ({
      product_id: p.product_id,
      product_name: p.product_name,
      poster_type: p.poster_type,
      observation_date: p.observation_date,
      absolute_return: p.absolute_return,
      annualized_return: p.annualized_return,
      duration_months: p.duration_months,
      dividend_count: p.dividend_count,
      cumulative_rate: p.cumulative_rate,
      monthly_coupon: p.monthly_coupon,
    })),
  }
}
```

- [ ] **Step 4: 运行测试**

Run: `cd backend && node services/agentTools.test.js`
Expected: `agentTools tests passed`

- [ ] **Step 5: Commit**

```bash
git add backend/services/agentTools.js
git commit -m "feat: add get_posters agent tool"
```

---

### Task 7: 产品文档搜索工具

**Files:**
- Modify: `backend/services/agentTools.js`

- [ ] **Step 1: 在 TOOL_DEFINITIONS 添加 search_product_docs**

```js
  {
    type: 'function',
    function: {
      name: 'search_product_docs',
      description: '搜索产品库文档，返回文档名称、路径和内容摘要。支持按关键词或月份过滤。当用户查找某月产品文档、搜索特定结构的文档时使用。',
      parameters: {
        type: 'object',
        properties: {
          keyword: { type: 'string', description: '文档名或内容关键词' },
          month: { type: 'string', description: '月份过滤，如 "6月"、"202606"、"2026-06"' },
        },
      },
    },
  },
```

- [ ] **Step 2: 在 executeTool switch 中添加 case**

```js
      case 'search_product_docs':
        return executeSearchProductDocs(args)
```

- [ ] **Step 3: 添加 executeSearchProductDocs 函数**

```js
function executeSearchProductDocs(args = {}) {
  const month = (args.month || '').trim()
  const keyword = (args.keyword || '').trim().toLowerCase()

  let docs
  if (month) {
    docs = dbModule.getProductDocsByMonth(month)
  } else {
    docs = dbModule.getAllProductDocs()
  }

  if (keyword) {
    docs = docs.filter(doc => {
      const text = `${doc.doc_name} ${doc.parent_path} ${doc.raw_content || ''}`.toLowerCase()
      return text.includes(keyword)
    })
  }

  return {
    count: docs.length,
    docs: docs.slice(0, 10).map(doc => {
      let structure = null
      if (doc.structure_json) {
        try { structure = JSON.parse(doc.structure_json) } catch {}
      }
      return {
        doc_name: doc.doc_name,
        parent_path: doc.parent_path,
        raw_content: (doc.raw_content || '').slice(0, 500),
        structure,
      }
    }),
  }
}
```

- [ ] **Step 4: 运行测试**

Run: `cd backend && node services/agentTools.test.js`
Expected: `agentTools tests passed`

- [ ] **Step 5: Commit**

```bash
git add backend/services/agentTools.js
git commit -m "feat: add search_product_docs agent tool"
```

---

### Task 8: 渠道汇总 + db.js 扩展

**Files:**
- Modify: `backend/db.js`
- Modify: `backend/services/agentTools.js`

- [ ] **Step 1: 在 db.js 的 module.exports 前添加查询函数**

```js
function getAllChannels() {
  return queryAll('SELECT id, channel_name FROM channels ORDER BY id')
}

function getAllDirectCustomerSources() {
  return queryAll('SELECT id, source_name FROM direct_customer_sources ORDER BY id')
}
```

- [ ] **Step 2: 在 db.js 的 module.exports 中添加**

添加 `getAllChannels, getAllDirectCustomerSources`。

- [ ] **Step 3: 在 TOOL_DEFINITIONS 添加 get_channels_summary**

```js
  {
    type: 'function',
    function: {
      name: 'get_channels_summary',
      description: '获取客户来源渠道的汇总信息，包括渠道列表和直客来源列表。当用户问客户渠道分布或来源情况时使用。',
      parameters: {
        type: 'object',
        properties: {},
      },
    },
  },
```

- [ ] **Step 4: 在 executeTool switch 中添加 case**

```js
      case 'get_channels_summary':
        return executeGetChannelsSummary(args)
```

- [ ] **Step 5: 添加 executeGetChannelsSummary 函数**

```js
function executeGetChannelsSummary() {
  const channels = dbModule.getAllChannels()
  const sources = dbModule.getAllDirectCustomerSources()
  return {
    channels: { count: channels.length, items: channels },
    direct_customer_sources: { count: sources.length, items: sources },
  }
}
```

- [ ] **Step 6: 运行测试**

Run: `cd backend && node services/agentTools.test.js`
Expected: `agentTools tests passed`

- [ ] **Step 7: Commit**

```bash
git add backend/db.js backend/services/agentTools.js
git commit -m "feat: add get_channels_summary agent tool and db functions"
```

---

### Task 9: 同步状态工具

**Files:**
- Modify: `backend/services/agentTools.js`

- [ ] **Step 1: 在 TOOL_DEFINITIONS 添加 get_sync_status**

```js
  {
    type: 'function',
    function: {
      name: 'get_sync_status',
      description: '查看各数据源的最近同步状态，包括产品+交易表、合投用户表、产品文档的最后同步时间和数据量。当用户问数据什么时候更新的、同步状态时使用。',
      parameters: {
        type: 'object',
        properties: {},
      },
    },
  },
```

- [ ] **Step 2: 在 executeTool switch 中添加 case**

```js
      case 'get_sync_status':
        return executeGetSyncStatus(args)
```

- [ ] **Step 3: 添加 executeGetSyncStatus 函数**

```js
function executeGetSyncStatus() {
  const productSync = dbModule.getLastSync()
  const coInvestSync = dbModule.getLastCoInvestSync()
  const docsSync = dbModule.getLastProductDocsSync()

  const sources = []

  sources.push({
    name: '产品+交易',
    last_sync: productSync ? productSync.synced_at : null,
    row_count: productSync ? productSync.row_count : 0,
  })

  sources.push({
    name: '合投用户',
    last_sync: coInvestSync ? coInvestSync.synced_at : null,
    row_count: coInvestSync ? coInvestSync.row_count : 0,
  })

  sources.push({
    name: '产品文档',
    last_sync: docsSync ? docsSync.synced_at : null,
    doc_count: docsSync ? docsSync.doc_count : 0,
    folder_count: docsSync ? docsSync.folder_count : 0,
  })

  return { sources }
}
```

- [ ] **Step 4: 运行测试**

Run: `cd backend && node services/agentTools.test.js`
Expected: `agentTools tests passed`

- [ ] **Step 5: Commit**

```bash
git add backend/services/agentTools.js
git commit -m "feat: add get_sync_status agent tool"
```

---

### Task 10: 操作日志工具

**Files:**
- Modify: `backend/services/agentTools.js`

- [ ] **Step 1: 在 TOOL_DEFINITIONS 添加 get_activity_logs**

```js
  {
    type: 'function',
    function: {
      name: 'get_activity_logs',
      description: '查询系统操作日志，记录数据同步、推送等操作历史。支持按类型过滤。当用户问最近做了什么操作或查看同步记录时使用。',
      parameters: {
        type: 'object',
        properties: {
          type: { type: 'string', description: '日志类型，如 "sync"' },
          limit: { type: 'integer', description: '返回条数，默认 20' },
        },
      },
    },
  },
```

- [ ] **Step 2: 在 executeTool switch 中添加 case**

```js
      case 'get_activity_logs':
        return executeGetActivityLogs(args)
```

- [ ] **Step 3: 添加 executeGetActivityLogs 函数**

```js
function executeGetActivityLogs(args = {}) {
  const type = (args.type || '').trim() || undefined
  const limit = args.limit || 20
  const logs = dbModule.queryActivityLogs(type, limit)
  return { count: logs.length, logs }
}
```

- [ ] **Step 4: 运行测试**

Run: `cd backend && node services/agentTools.test.js`
Expected: `agentTools tests passed`

- [ ] **Step 5: Commit**

```bash
git add backend/services/agentTools.js
git commit -m "feat: add get_activity_logs agent tool"
```

---

### Task 11: 更新测试 - 验证全部 16 个工具

**Files:**
- Modify: `backend/services/agentTools.test.js`

- [x] **Step 1: 更新测试文件**

更新 `backend/services/agentTools.test.js`，在现有测试基础上新增所有新工具的验证：

```js
const assert = require('node:assert/strict')
const { TOOL_DEFINITIONS, executeTool } = require('./agentTools')
const dbModule = require('../db')

assert(Array.isArray(TOOL_DEFINITIONS))
assert(TOOL_DEFINITIONS.length === 16, `Expected 16 tools, got ${TOOL_DEFINITIONS.length}`)

const expectedNames = [
  'search_products', 'get_product_detail', 'get_observations', 'get_price',
  'get_dashboard_stats', 'get_observation_calendar',
  'search_customers', 'get_customer_products', 'get_customer_peak_analysis',
  'query_transactions', 'get_product_analytics', 'get_posters',
  'search_product_docs', 'get_channels_summary', 'get_sync_status', 'get_activity_logs',
]
const names = TOOL_DEFINITIONS.map(t => t.function.name)
for (const expected of expectedNames) {
  assert(names.includes(expected), `Missing tool: ${expected}`)
}

for (const tool of TOOL_DEFINITIONS) {
  assert(tool.type === 'function', `Tool ${tool.function.name} must have type "function"`)
  assert(tool.function.name, 'Tool must have a name')
  assert(tool.function.description, `Tool ${tool.function.name} missing description`)
  assert(tool.function.parameters, `Tool ${tool.function.name} missing parameters`)
}

async function main() {
  await dbModule.initDatabase()

  // === 原有测试：get_observation_calendar ===
  const byMonth = await executeTool('get_observation_calendar', { month: '2026-06' })
  assert(!byMonth.error, `Monthly calendar query failed: ${byMonth.error}`)
  assert.equal(byMonth.query.mode, 'month')
  assert.equal(byMonth.query.month, '2026-06')
  assert(Array.isArray(byMonth.calendar), 'calendar should be an array')
  assert(byMonth.calendar.length > 0, 'expected monthly calendar data for 2026-06')
  assert(byMonth.calendar.every(item => item.date.startsWith('2026-06-')), 'monthly results should stay inside requested month')
  assert.equal(byMonth.summary.date_count, byMonth.calendar.length)

  const firstDate = byMonth.calendar[0].date
  const firstProduct = byMonth.calendar[0].products[0]
  assert(firstProduct, 'expected at least one product on first calendar date')

  const byDate = await executeTool('get_observation_calendar', { date: firstDate })
  assert(!byDate.error, `Single-date calendar query failed: ${byDate.error}`)
  assert.equal(byDate.query.mode, 'date')
  assert.deepEqual(byDate.calendar.map(item => item.date), [firstDate])
  assert(byDate.calendar[0].products.length > 0, 'expected products on selected date')

  const byRange = await executeTool('get_observation_calendar', {
    start_date: firstDate,
    end_date: firstDate,
  })
  assert(!byRange.error, `Range calendar query failed: ${byRange.error}`)
  assert.equal(byRange.query.mode, 'range')
  assert.equal(byRange.query.start_date, firstDate)
  assert.equal(byRange.query.end_date, firstDate)
  assert.deepEqual(byRange.calendar.map(item => item.date), [firstDate])

  const byProduct = await executeTool('get_observation_calendar', {
    month: '2026-06',
    product_keyword: firstProduct.name.slice(0, 2),
  })
  assert(!byProduct.error, `Product-filter calendar query failed: ${byProduct.error}`)
  assert(byProduct.calendar.length > 0, 'expected filtered calendar data by product')
  assert(byProduct.calendar.every(item => item.products.every(product => product.name.includes(firstProduct.name.slice(0, 2)))))

  const byManager = await executeTool('get_observation_calendar', {
    month: '2026-06',
    manager: firstProduct.manager,
  })
  assert(!byManager.error, `Manager-filter calendar query failed: ${byManager.error}`)
  assert(byManager.calendar.length > 0, 'expected filtered calendar data by manager')
  assert(byManager.calendar.every(item => item.products.every(product => product.manager === firstProduct.manager)))

  const invalidRange = await executeTool('get_observation_calendar', {
    start_date: '2026-06-30',
    end_date: '2026-06-01',
  })
  assert.equal(invalidRange.error, 'end_date must be greater than or equal to start_date')

  const missingStart = await executeTool('get_observation_calendar', {
    end_date: '2026-06-01',
  })
  assert.equal(missingStart.error, 'start_date is required when end_date is provided')

  const emptyResult = await executeTool('get_observation_calendar', {
    month: '2026-06',
    product_keyword: '__definitely_not_found__',
  })
  assert(!emptyResult.error, `Empty calendar query should not error: ${emptyResult.error}`)
  assert.equal(emptyResult.summary.date_count, 0)
  assert.equal(emptyResult.summary.product_count, 0)
  assert.deepEqual(emptyResult.calendar, [])

  // === 新工具测试：search_customers ===
  const allCustomers = await executeTool('search_customers', {})
  assert(!allCustomers.error, `search_customers failed: ${allCustomers.error}`)
  assert(typeof allCustomers.count === 'number', 'count should be a number')
  assert(Array.isArray(allCustomers.customers), 'customers should be an array')

  const noCustomers = await executeTool('search_customers', { keyword: '__nonexistent__' })
  assert(!noCustomers.error, 'search_customers should not error on empty result')
  assert.equal(noCustomers.count, 0)

  // === 新工具测试：get_customer_products ===
  const missingCustomer = await executeTool('get_customer_products', {})
  assert(missingCustomer.error, 'should error without customer_name')

  const unknownCustomer = await executeTool('get_customer_products', { customer_name: '__nonexistent__' })
  assert(!unknownCustomer.error, 'should not error for unknown customer')
  assert.equal(unknownCustomer.count, 0)

  // === 新工具测试：get_customer_peak_analysis ===
  const allPeaks = await executeTool('get_customer_peak_analysis', {})
  assert(!allPeaks.error, `get_customer_peak_analysis failed: ${allPeaks.error}`)
  assert(typeof allPeaks.count === 'number')
  assert(Array.isArray(allPeaks.analyses))

  const filteredPeaks = await executeTool('get_customer_peak_analysis', { customer_name: '__nonexistent__' })
  assert(!filteredPeaks.error)
  assert.equal(filteredPeaks.count, 0)

  // === 新工具测试：query_transactions ===
  const allTx = await executeTool('query_transactions', {})
  assert(!allTx.error, `query_transactions failed: ${allTx.error}`)
  assert(typeof allTx.count === 'number')
  assert(typeof allTx.total_amount === 'number')
  assert(Array.isArray(allTx.transactions))

  // === 新工具测试：get_product_analytics ===
  const byMgr = await executeTool('get_product_analytics', { group_by: 'manager' })
  assert(!byMgr.error, `get_product_analytics(manager) failed: ${byMgr.error}`)
  assert.equal(byMgr.group_by, 'manager')
  assert(Array.isArray(byMgr.groups))
  if (byMgr.groups.length > 0) {
    const g = byMgr.groups[0]
    assert(typeof g.key === 'string')
    assert(typeof g.count === 'number')
    assert(typeof g.total_subscribe === 'number')
  }

  const byStatus = await executeTool('get_product_analytics', { group_by: 'holding_status' })
  assert(!byStatus.error)
  assert(byStatus.groups.length > 0, 'expected holding_status groups')

  const invalidGroup = await executeTool('get_product_analytics', { group_by: 'invalid_field' })
  assert(invalidGroup.error, 'should error on invalid group_by')

  // === 新工具测试：get_posters ===
  const allPosters = await executeTool('get_posters', {})
  assert(!allPosters.error, `get_posters failed: ${allPosters.error}`)
  assert(typeof allPosters.count === 'number')
  assert(Array.isArray(allPosters.posters))

  // === 新工具测试：search_product_docs ===
  const allDocs = await executeTool('search_product_docs', {})
  assert(!allDocs.error, `search_product_docs failed: ${allDocs.error}`)
  assert(typeof allDocs.count === 'number')
  assert(Array.isArray(allDocs.docs))

  const noDocs = await executeTool('search_product_docs', { keyword: '__zzz_no_match__' })
  assert(!noDocs.error)
  assert.equal(noDocs.count, 0)

  // === 新工具测试：get_channels_summary ===
  const channels = await executeTool('get_channels_summary', {})
  assert(!channels.error, `get_channels_summary failed: ${channels.error}`)
  assert(typeof channels.channels.count === 'number')
  assert(Array.isArray(channels.channels.items))
  assert(typeof channels.direct_customer_sources.count === 'number')

  // === 新工具测试：get_sync_status ===
  const syncStatus = await executeTool('get_sync_status', {})
  assert(!syncStatus.error, `get_sync_status failed: ${syncStatus.error}`)
  assert(Array.isArray(syncStatus.sources))
  assert(syncStatus.sources.length === 3, 'expected 3 data sources')
  const sourceNames = syncStatus.sources.map(s => s.name)
  assert(sourceNames.includes('产品+交易'))
  assert(sourceNames.includes('合投用户'))
  assert(sourceNames.includes('产品文档'))

  // === 新工具测试：get_activity_logs ===
  const logs = await executeTool('get_activity_logs', {})
  assert(!logs.error, `get_activity_logs failed: ${logs.error}`)
  assert(typeof logs.count === 'number')
  assert(Array.isArray(logs.logs))
  if (logs.logs.length > 0) {
    const log = logs.logs[0]
    assert(typeof log.id === 'number')
    assert(typeof log.type === 'string')
    assert(typeof log.action === 'string')
  }

  const filteredLogs = await executeTool('get_activity_logs', { type: 'sync', limit: 5 })
  assert(!filteredLogs.error)
  assert(filteredLogs.logs.every(l => l.type === 'sync'), 'filtered logs should all be type=sync')
}

main()
  .then(() => {
    console.log('agentTools tests passed')
  })
  .catch(err => {
    console.error(err)
    process.exit(1)
  })
```

- [x] **Step 2: 运行全部测试**

Run: `cd backend && node services/agentTools.test.js`
Expected: `agentTools tests passed`

- [ ] **Step 3: Commit**

```bash
git add backend/services/agentTools.test.js
git commit -m "test: add comprehensive tests for all 16 agent tools"
```
