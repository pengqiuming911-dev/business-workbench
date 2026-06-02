# 产品派息/敲出观察 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the `/product-completion` page from "完结与发行" to "产品派息/敲出观察", showing ongoing product dividend/knock-out observation status with real-time A-share index prices.

**Architecture:** Extend the existing SQLite backend (db.js + index.js) with new tables (observations, price_cache) and APIs. Use Eastmoney free API for real-time index prices. Add node-cron for scheduled price updates. Rewrite the Vue frontend page.

**Tech Stack:** Vue 3 (Composition API), Express, sql.js (in-memory SQLite), node-cron, Eastmoney API

---

### Task 1: Install node-cron Dependency

**Files:**
- Modify: `backend/package.json`

- [ ] **Step 1: Install node-cron**

```bash
cd backend && npm install node-cron
```

- [ ] **Step 2: Verify installation**

Run: `node -e "require('node-cron'); console.log('ok')"`
Expected: `ok`

- [ ] **Step 3: Commit**

```bash
git add backend/package.json backend/package-lock.json
git commit -m "chore: add node-cron dependency"
```

---

### Task 2: Extend Products Table Schema in db.js

**Files:**
- Modify: `backend/db.js:22-31` (products CREATE TABLE)
- Modify: `backend/db.js:162-180` (importProducts function)
- Modify: `backend/db.js:420-429` (module.exports)

- [ ] **Step 1: Update products CREATE TABLE**

In `backend/db.js`, replace the existing `CREATE TABLE IF NOT EXISTS products` block (lines 22-31) with:

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

- [ ] **Step 2: Update importProducts function**

Replace the existing `importProducts` function (lines 162-180) with:

```javascript
function importProducts(rows) {
  const incomingIds = rows.map(r => r.id)
  const existing = queryAll('SELECT id FROM products')
  for (const e of existing) {
    if (!incomingIds.includes(e.id)) {
      runStatement('DELETE FROM products WHERE id = ?', [e.id])
    }
  }
  for (const r of rows) {
    runStatement(`
      INSERT OR REPLACE INTO products
        (id, name, is_main, issue_date, complete_date, subscribe_amount, outstanding_amount,
         manager, holding_status, structure_type, code, lock_days, lock_months,
         first_knockout_ratio, entry_price, monthly_decrease, term, parachute,
         dividend_barrier, monthly_coupon, coupon_1st, coupon_2nd, coupon_3rd,
         duration_months, absolute_return, holiday_adjust, raw)
      VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
    `, [
      r.id, r.name, r.is_main, r.issue_date, r.complete_date,
      r.subscribe_amount, r.outstanding_amount,
      r.manager, r.holding_status, r.structure_type, r.code,
      r.lock_days, r.lock_months, r.first_knockout_ratio,
      r.entry_price, r.monthly_decrease, r.term, r.parachute,
      r.dividend_barrier, r.monthly_coupon,
      r.coupon_1st, r.coupon_2nd, r.coupon_3rd,
      r.duration_months, r.absolute_return, r.holiday_adjust, r.raw
    ])
  }
  saveDatabase()
}
```

- [ ] **Step 3: Add new DB functions for observations and price_cache**

Add the following functions to `backend/db.js` (after `queryProducts`, around line 202):

```javascript
// ──── 存续产品查询 ────

function queryOngoingProducts() {
  return queryAll(`SELECT * FROM products WHERE holding_status LIKE '%持有%'`)
}

// ──── 观察记录表 ────

function upsertObserv(row) {
  const existing = queryOne(
    'SELECT id FROM observations WHERE product_id = ? AND observation_date = ?',
    [row.product_id, row.observation_date]
  )
  if (existing) {
    runStatement(`
      UPDATE observations SET
        knockout_price = ?, dividend_line = ?, underlying_price = ?,
        is_knocked_out = ?, is_dividend = ?, months_since_entry = ?, updated_at = ?
      WHERE id = ?
    `, [
      row.knockout_price, row.dividend_line, row.underlying_price,
      row.is_knocked_out, row.is_dividend, row.months_since_entry,
      new Date().toISOString(), existing.id
    ])
  } else {
    runStatement(`
      INSERT INTO observations
        (product_id, observation_date, knockout_price, dividend_line,
         underlying_price, is_knocked_out, is_dividend, months_since_entry, updated_at)
      VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
    `, [
      row.product_id, row.observation_date, row.knockout_price, row.dividend_line,
      row.underlying_price, row.is_knocked_out, row.is_dividend,
      row.months_since_entry, new Date().toISOString()
    ])
  }
}

function queryObservationsByProduct(productId) {
  return queryAll(
    'SELECT * FROM observations WHERE product_id = ? ORDER BY observation_date',
    [productId]
  )
}

function queryAllObservations() {
  return queryAll('SELECT * FROM observations ORDER BY product_id, observation_date')
}

// ──── 价格缓存表 ────

function upsertPrice(code, priceDate, price) {
  const existing = queryOne(
    'SELECT code FROM price_cache WHERE code = ? AND price_date = ?',
    [code, priceDate]
  )
  if (existing) {
    runStatement(
      'UPDATE price_cache SET price = ?, updated_at = ? WHERE code = ? AND price_date = ?',
      [price, new Date().toISOString(), code, priceDate]
    )
  } else {
    runStatement(
      'INSERT INTO price_cache (code, price_date, price, updated_at) VALUES (?, ?, ?, ?)',
      [code, priceDate, price, new Date().toISOString()]
    )
  }
  saveDatabase()
}

function queryLatestPrice(code) {
  return queryOne(
    'SELECT * FROM price_cache WHERE code = ? ORDER BY price_date DESC LIMIT 1',
    [code]
  )
}

function queryPriceByDate(code, priceDate) {
  return queryOne(
    'SELECT * FROM price_cache WHERE code = ? AND price_date = ?',
    [code, priceDate]
  )
}

function getLastObservationUpdate() {
  return queryOne('SELECT updated_at FROM observations ORDER BY updated_at DESC LIMIT 1')
}
```

- [ ] **Step 4: Add observations and price_cache table creation in initDatabase**

In the `db.exec()` call within `initDatabase`, add these two CREATE TABLE statements (after the existing tables, before the closing backtick on line 113):

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

CREATE TABLE IF NOT EXISTS price_cache (
  code TEXT,
  price_date TEXT,
  price REAL,
  updated_at TEXT,
  PRIMARY KEY (code, price_date)
);
```

- [ ] **Step 5: Update module.exports**

Replace the existing `module.exports` block (lines 420-429) with:

```javascript
module.exports = {
  initDatabase,
  importProducts, logSync, getLastSync, queryProducts,
  importCoInvestUsers, logCoInvestSync, getLastCoInvestSync,
  queryCoInvestUsers, getDistinctIndustries,
  getCustomerProductLinks, importCustomerProductLinks,
  importTransactions, importChannels, importDirectCustomerSources, importCustomers,
  computeUserPeakBalances,
  importProductDocs, logProductDocsSync, getLastProductDocsSync, getAllProductDocs, getProductDocsByMonth,
  queryOngoingProducts,
  upsertObserv, queryObservationsByProduct, queryAllObservations,
  upsertPrice, queryLatestPrice, queryPriceByDate,
  getLastObservationUpdate
}
```

- [ ] **Step 6: Verify backend still starts**

```bash
cd backend && node -e "const db = require('./db'); db.initDatabase().then(() => console.log('ok'))"
```
Expected: 数据库初始化完成 + ok

- [ ] **Step 7: Commit**

```bash
git add backend/db.js
git commit -m "feat(db): extend products table + add observations/price_cache tables and CRUD functions"
```

---

### Task 3: Extend Feishu Sync Mapping

**Files:**
- Modify: `backend/index.js:534-549` (productRows mapping in sync endpoint)

- [ ] **Step 1: Update productRows mapping in POST /api/db/sync**

In `backend/index.js`, replace the existing productRows loop (lines 534-549) with:

```javascript
    const productRows = []
    for (const r of prodRows) {
      const flightId = r['航班编号']
      if (!flightId) continue
      const isMain = String(r['是否主产品'] || r[' 是否主产品'] || '').trim() === '是' ? 1 : 0
      const lockDays = Number(r['锁定期']) || 0
      productRows.push({
        id: String(flightId).trim(),
        name: r['产品名称'] || null,
        is_main: isMain,
        issue_date: excelDateToString(r['认购日']),
        complete_date: excelDateToString(r['完结时间']),
        subscribe_amount: Number(r['认购金额']) || 0,
        outstanding_amount: Number(r['存续金额']) || 0,
        manager: r['私募管理人'] || null,
        holding_status: r['持有状态'] || null,
        structure_type: r['结构类型'] || null,
        code: r['代码'] || null,
        lock_days: lockDays,
        lock_months: Math.floor(lockDays / 30),
        first_knockout_ratio: Number(r['敲出']) || Number(r['首月的敲出比例']) || 0,
        entry_price: Number(r['入场价']) || 0,
        monthly_decrease: Number(r['每月递减']) || 0,
        term: r['期限'] || null,
        parachute: r['降落伞'] || null,
        dividend_barrier: Number(r['派息障碍']) || 0,
        monthly_coupon: Number(r['月票息']) || Number(r['月票息（税费后）']) || 0,
        coupon_1st: Number(r['第一段票息']) || Number(r['第一段票息（税费后）']) || 0,
        coupon_2nd: Number(r['第二段票息']) || Number(r['第二段票息（税费后）']) || 0,
        coupon_3rd: Number(r['第三段票息']) || Number(r['第三段票息（税费后）']) || 0,
        duration_months: Number(r['存续时间（月）']) || Number(r['存续时间']) || 0,
        absolute_return: Number(r['绝对收益率']) || 0,
        holiday_adjust: r['观察日节假日顺延/提前'] || r['观察日节假日顺延'] || null,
        raw: JSON.stringify(r),
      })
    }
```

- [ ] **Step 2: Verify server starts without error**

```bash
cd backend && node -e "require('./db').initDatabase().then(() => { require('./index'); console.log('ok') })"
```

Expected: Server starts successfully. Press Ctrl+C to stop.

- [ ] **Step 3: Commit**

```bash
git add backend/index.js
git commit -m "feat(sync): extend Feishu product sync mapping with 19 new fields"
```

---

### Task 4: Implement Price Fetching Service

**Files:**
- Create: `backend/services/priceService.js`

- [ ] **Step 1: Create the price service module**

Create `backend/services/priceService.js`:

```javascript
const axios = require('axios')

const EASTMONEY_API = 'https://push2.eastmoney.com/api/qt/stock/get'

function resolveSecId(code) {
  if (!code) return null
  const cleaned = code.trim().toLowerCase()
  if (cleaned.startsWith('sh') || cleaned.startsWith('1.') || cleaned.startsWith('5')) {
    return `1.${cleaned.replace(/^(sh|1\.)/, '')}`
  }
  if (cleaned.startsWith('sz') || cleaned.startsWith('0.') || cleaned.startsWith('3')) {
    return `0.${cleaned.replace(/^(sz|0\.)/, '')}`
  }
  return `1.${cleaned}`
}

async function fetchLatestPrice(code) {
  const secid = resolveSecId(code)
  if (!secid) throw new Error(`Invalid code: ${code}`)

  const res = await axios.get(EASTMONEY_API, {
    params: {
      secid,
      fields: 'f43,f44,f45,f46,f47,f170'
    },
    timeout: 5000,
    headers: {
      'User-Agent': 'Mozilla/5.0',
      'Referer': 'https://quote.eastmoney.com/'
    }
  })

  if (!res.data || !res.data.data || res.data.data.f43 === undefined) {
    throw new Error(`No price data for ${code}: ${JSON.stringify(res.data)}`)
  }

  const rawPrice = res.data.data.f43
  const price = rawPrice > 10000 ? rawPrice / 100 : rawPrice
  return price
}

async function fetchAllPrices(codes) {
  const results = {}
  const failed = []
  for (const code of codes) {
    try {
      results[code] = await fetchLatestPrice(code)
    } catch (err) {
      console.error(`Failed to fetch price for ${code}:`, err.message)
      failed.push(code)
    }
  }
  return { results, failed }
}

module.exports = { fetchLatestPrice, fetchAllPrices, resolveSecId }
```

- [ ] **Step 2: Verify price fetching works**

```bash
cd backend && node -e "const p = require('./services/priceService'); p.fetchLatestPrice('sh000300').then(price => console.log('沪深300:', price)).catch(e => console.error(e))"
```
Expected: `沪深300: <some number>` (e.g., `沪深300: 4914.56`)

- [ ] **Step 3: Commit**

```bash
git add backend/services/priceService.js
git commit -m "feat: add Eastmoney price fetching service for A-share indices"
```

---

### Task 5: Implement Observation Logic

**Files:**
- Create: `backend/services/observationService.js`

- [ ] **Step 1: Create the observation service module**

Create `backend/services/observationService.js`:

```javascript
function monthsBetween(entryDate, obsDate) {
  const entry = new Date(entryDate)
  const obs = new Date(obsDate)
  return (obs.getFullYear() - entry.getFullYear()) * 12 + (obs.getMonth() - entry.getMonth())
}

function addMonths(date, months) {
  const d = new Date(date)
  const targetMonth = d.getMonth() + months
  d.setMonth(targetMonth)
  if (d.getDate() !== new Date(date).getDate()) {
    d.setDate(0)
  }
  return d.toISOString().slice(0, 10)
}

function isWeekend(dateStr) {
  const day = new Date(dateStr).getDay()
  return day === 0 || day === 6
}

function adjustForHoliday(dateStr, holidayAdjust) {
  if (!isWeekend(dateStr)) return dateStr
  const d = new Date(dateStr)
  if (holidayAdjust === '提前') {
    while (isWeekend(d.toISOString().slice(0, 10))) {
      d.setDate(d.getDate() - 1)
    }
  } else {
    while (isWeekend(d.toISOString().slice(0, 10))) {
      d.setDate(d.getDate() + 1)
    }
  }
  return d.toISOString().slice(0, 10)
}

function getObservationDates(product) {
  const entryDate = product.issue_date
  if (!entryDate) return []

  const today = new Date().toISOString().slice(0, 10)
  const dates = []
  let months = 1

  while (true) {
    const nextDate = addMonths(entryDate, months)
    if (nextDate > today) break

    const adjusted = adjustForHoliday(nextDate, product.holiday_adjust)
    dates.push({ date: adjusted, monthsSinceEntry: months })
    months++
  }

  return dates
}

function computeKnockoutPrice(product, monthsSinceEntry) {
  if (monthsSinceEntry < (product.lock_months || 0)) return null
  const currentRatio = product.first_knockout_ratio
    - (monthsSinceEntry - product.lock_months) * product.monthly_decrease
  return product.entry_price * currentRatio
}

function computeDividendLine(product) {
  return product.entry_price * (product.dividend_barrier || 0)
}

function evaluateObservation(product, obsDate, underlyingPrice, monthsSinceEntry) {
  const dividendLine = computeDividendLine(product)
  const isDividend = underlyingPrice > dividendLine ? '是' : '否'

  const knockoutPrice = computeKnockoutPrice(product, monthsSinceEntry)
  let isKnockedOut = '--'
  if (knockoutPrice !== null) {
    isKnockedOut = underlyingPrice > knockoutPrice ? '是' : '否'
  }

  return {
    observation_date: obsDate,
    knockout_price: knockoutPrice,
    dividend_line: dividendLine,
    underlying_price: underlyingPrice,
    is_knocked_out: isKnockedOut,
    is_dividend: isDividend,
    months_since_entry: monthsSinceEntry,
  }
}

module.exports = {
  monthsBetween, addMonths, isWeekend, adjustForHoliday,
  getObservationDates, computeKnockoutPrice, computeDividendLine,
  evaluateObservation
}
```

- [ ] **Step 2: Verify observation date generation**

```bash
cd backend && node -e "
const obs = require('./services/observationService');
const product = { issue_date: '2025-03-06', holiday_adjust: '顺延' };
const dates = obs.getObservationDates(product);
console.log('Observation dates:', dates.length, dates.slice(0, 3));
"
```
Expected: Shows observation dates starting from 2025-04-06, with monthsSinceEntry 1, 2, 3, etc.

- [ ] **Step 3: Verify knockout price calculation**

```bash
cd backend && node -e "
const obs = require('./services/observationService');
const product = { entry_price: 3200, first_knockout_ratio: 1.03, lock_months: 6, monthly_decrease: 0.005 };
console.log('Month 5 (lock):', obs.computeKnockoutPrice(product, 5));
console.log('Month 6 (first):', obs.computeKnockoutPrice(product, 6));
console.log('Month 7:', obs.computeKnockoutPrice(product, 7));
"
```
Expected: Month 5 → `null`, Month 6 → `3296`, Month 7 → `3280`

- [ ] **Step 4: Commit**

```bash
git add backend/services/observationService.js
git commit -m "feat: add observation date calculation and knockout/dividend evaluation logic"
```

---

### Task 6: Add Backend API Endpoints

**Files:**
- Modify: `backend/index.js:5` (import new functions)
- Modify: `backend/index.js:994-1005` (add endpoints before server startup)

- [ ] **Step 1: Update imports at top of index.js**

In `backend/index.js`, line 5, add the new imports. Replace the existing require line:

```javascript
const { initDatabase, importProducts, logSync, getLastSync, queryProducts, importCoInvestUsers, logCoInvestSync, getLastCoInvestSync, queryCoInvestUsers, getDistinctIndustries, getCustomerProductLinks, importCustomerProductLinks, importTransactions, importChannels, importDirectCustomerSources, importCustomers, computeUserPeakBalances, importProductDocs, logProductDocsSync, getLastProductDocsSync, getAllProductDocs, getProductDocsByMonth, queryOngoingProducts, upsertObserv, queryObservationsByProduct, upsertPrice, queryLatestPrice, queryPriceByDate, getLastObservationUpdate } = require('./db')
```

Also add these requires at the top (after line 5):

```javascript
const { fetchAllPrices } = require('./services/priceService')
const { getObservationDates, evaluateObservation } = require('./services/observationService')
```

- [ ] **Step 2: Add the 4 observations API endpoints**

Add the following endpoints in `backend/index.js`, before the `initDatabase().then(...)` block (before line 994):

```javascript
// ─────────────────────────────────────────
// GET /api/observations
// 查询所有存续产品的观察记录
// ─────────────────────────────────────────
app.get('/api/observations', (req, res) => {
  try {
    const { search } = req.query
    let products = queryOngoingProducts()
    if (search) {
      const q = search.toLowerCase()
      products = products.filter(p =>
        (p.name && p.name.toLowerCase().includes(q)) ||
        p.id.toLowerCase().includes(q)
      )
    }

    const result = products.map(p => {
      const observations = queryObservationsByProduct(p.id)
      return {
        id: p.id,
        name: p.name,
        manager: p.manager,
        holding_status: p.holding_status,
        code: p.code,
        entry_price: p.entry_price,
        first_knockout_ratio: p.first_knockout_ratio,
        lock_months: p.lock_months,
        monthly_decrease: p.monthly_decrease,
        issue_date: p.issue_date,
        subscribe_amount: p.subscribe_amount,
        dividend_barrier: p.dividend_barrier,
        holiday_adjust: p.holiday_adjust,
        lock_days: p.lock_days,
        duration_months: p.duration_months,
        observations: observations.map(o => ({
          date: o.observation_date,
          knockout_price: o.knockout_price,
          dividend_line: o.dividend_line,
          underlying_price: o.underlying_price,
          is_knocked_out: o.is_knocked_out,
          is_dividend: o.is_dividend,
          months_since_entry: o.months_since_entry,
        })),
      }
    })

    const lastRecord = getLastObservationUpdate()
    res.json({
      products: result,
      lastUpdated: lastRecord ? lastRecord.updated_at : null,
    })
  } catch (err) {
    res.status(500).json({ error: err.message })
  }
})

// ─────────────────────────────────────────
// POST /api/observations/generate
// 生成/更新观察记录（获取最新价格 + 计算结果）
// ─────────────────────────────────────────
app.post('/api/observations/generate', async (req, res) => {
  try {
    const products = queryOngoingProducts()
    const codes = [...new Set(products.map(p => p.code).filter(Boolean))]

    const { results: prices, failed } = await fetchAllPrices(codes)

    const today = new Date().toISOString().slice(0, 10)
    for (const code of codes) {
      if (prices[code] !== undefined) {
        upsertPrice(code, today, prices[code])
      }
    }

    let generated = 0
    let updated = 0

    for (const product of products) {
      if (!product.code || !product.issue_date || !product.entry_price) continue

      const obsDates = getObservationDates(product)
      const latestPrice = prices[product.code] || null

      const existingObs = queryObservationsByProduct(product.id)
      const existingDates = new Set(existingObs.map(o => o.observation_date))

      for (const { date, monthsSinceEntry } of obsDates) {
        const priceForDate = latestPrice
        if (priceForDate === null) continue

        const evalResult = evaluateObservation(product, date, priceForDate, monthsSinceEntry)
        evalResult.product_id = product.id
        evalResult.underlying_price = priceForDate

        if (existingDates.has(date)) updated++
        else generated++

        upsertObserv(evalResult)
      }
    }

    res.json({ ok: true, generated, updated, priceRefreshed: codes.length, priceFailed: failed.length })
  } catch (err) {
    console.error('生成观察记录失败:', err)
    res.status(500).json({ error: err.message })
  }
})

// ─────────────────────────────────────────
// POST /api/observations/refresh-prices
// 仅刷新标的价格
// ─────────────────────────────────────────
app.post('/api/observations/refresh-prices', async (req, res) => {
  try {
    const products = queryOngoingProducts()
    const codes = [...new Set(products.map(p => p.code).filter(Boolean))]

    const { results: prices, failed } = await fetchAllPrices(codes)
    const today = new Date().toISOString().slice(0, 10)

    let refreshed = 0
    for (const code of codes) {
      if (prices[code] !== undefined) {
        upsertPrice(code, today, prices[code])
        refreshed++
      }
    }

    for (const product of products) {
      if (!product.code || prices[product.code] === undefined) continue
      const productObs = queryObservationsByProduct(product.id)
      for (const obs of productObs) {
        const latestPrice = prices[product.code]
        const evalResult = evaluateObservation(
          product, obs.observation_date, latestPrice, obs.months_since_entry
        )
        evalResult.product_id = product.id
        upsertObserv(evalResult)
      }
    }

    res.json({ ok: true, refreshed, failed: failed.length })
  } catch (err) {
    console.error('刷新价格失败:', err)
    res.status(500).json({ error: err.message })
  }
})

// ─────────────────────────────────────────
// GET /api/observations/products
// 获取所有存续产品列表（概要）
// ─────────────────────────────────────────
app.get('/api/observations/products', (req, res) => {
  try {
    const products = queryOngoingProducts()
    res.json({ products })
  } catch (err) {
    res.status(500).json({ error: err.message })
  }
})
```

- [ ] **Step 3: Add post-sync auto-trigger to POST /api/db/sync**

In `backend/index.js`, in the `POST /api/db/sync` handler, find the section after `importProducts(productRows)` (around line 551) where `logSync(totalCount)` is called. Add the following code block right after `importProducts(productRows)`:

```javascript
    // ── 同步后自动更新价格和观察记录 ──
    try {
      const ongoingProducts = queryOngoingProducts()
      const ongoingCodes = [...new Set(ongoingProducts.map(p => p.code).filter(Boolean))]
      if (ongoingCodes.length > 0) {
        const { results: autoPrices, failed: autoFailed } = await fetchAllPrices(ongoingCodes)
        const today = new Date().toISOString().slice(0, 10)
        for (const code of ongoingCodes) {
          if (autoPrices[code] !== undefined) upsertPrice(code, today, autoPrices[code])
        }
        let autoGen = 0
        for (const product of ongoingProducts) {
          if (!product.code || !product.issue_date || !product.entry_price || autoPrices[product.code] === undefined) continue
          const obsDates = getObservationDates(product)
          for (const { date, monthsSinceEntry } of obsDates) {
            const evalResult = evaluateObservation(product, date, autoPrices[product.code], monthsSinceEntry)
            evalResult.product_id = product.id
            upsertObserv(evalResult)
            autoGen++
          }
        }
        console.log(`同步后自动更新：价格 ${ongoingCodes.length - (autoFailed?.length || 0)}/${ongoingCodes.length}，观察记录 ${autoGen} 条`)
      }
    } catch (autoErr) {
      console.warn('[同步后自动更新观察记录失败，不影响主同步]', autoErr.message)
    }
```

- [ ] **Step 4: Verify server starts and endpoints are accessible**

```bash
cd backend && node index.js &
curl http://localhost:3001/api/observations
curl http://localhost:3001/api/observations/products
```
Expected: Both return valid JSON (empty products array if no data synced).

- [ ] **Step 5: Commit**

```bash
git add backend/index.js backend/services/
git commit -m "feat(api): add observations CRUD endpoints, price refresh, and post-sync auto-trigger"
```

---

### Task 7: Add Cron Scheduling

**Files:**
- Modify: `backend/index.js` (after API endpoints, before server startup)

- [ ] **Step 1: Add cron jobs in index.js**

Add this code in `backend/index.js` just before the `initDatabase().then(...)` block:

```javascript
// ─────────────────────────────────────────
// 定时任务：每个工作日 11:30、15:00、15:30 自动更新价格和观察记录
// ─────────────────────────────────────────
const cron = require('node-cron')

async function scheduledPriceUpdate() {
  try {
    const products = queryOngoingProducts()
    const codes = [...new Set(products.map(p => p.code).filter(Boolean))]
    if (codes.length === 0) return

    console.log(`[定时任务] 开始更新 ${codes.length} 个标的价格...`)
    const { results: prices, failed } = await fetchAllPrices(codes)
    const today = new Date().toISOString().slice(0, 10)

    for (const code of codes) {
      if (prices[code] !== undefined) {
        upsertPrice(code, today, prices[code])
      }
    }

    let updatedObs = 0
    for (const product of products) {
      if (!product.code || prices[product.code] === undefined) continue
      const obsDates = getObservationDates(product)
      for (const { date, monthsSinceEntry } of obsDates) {
        if (date === today) {
          const evalResult = evaluateObservation(product, date, prices[product.code], monthsSinceEntry)
          evalResult.product_id = product.id
          upsertObserv(evalResult)
          updatedObs++
        }
      }
    }

    console.log(`[定时任务] 完成: 价格更新 ${codes.length - failed.length}/${codes.length}, 观察记录更新 ${updatedObs} 条`)
  } catch (err) {
    console.error('[定时任务] 失败:', err.message)
  }
}

cron.schedule('30 11 * * 1-5', scheduledPriceUpdate)
cron.schedule('0 15 * * 1-5', scheduledPriceUpdate)
cron.schedule('30 15 * * 1-5', scheduledPriceUpdate)
console.log('定时任务已注册: 工作日 11:30, 15:00, 15:30')
```

- [ ] **Step 2: Verify cron registration**

```bash
cd backend && node -e "
const cron = require('node-cron');
console.log('Cron tasks registered:', cron.getTasks().size);
"
```
Expected: Shows tasks registered.

- [ ] **Step 3: Commit**

```bash
git add backend/index.js
git commit -m "feat: add cron jobs for scheduled price updates on weekdays"
```

---

### Task 8: Rewrite Frontend ProductCompletion.vue

**Files:**
- Rewrite: `frontend/views/ProductCompletion.vue`

- [ ] **Step 1: Replace ProductCompletion.vue with new page**

Replace the entire file `frontend/views/ProductCompletion.vue` with:

```vue
<template>
  <SubPageLayout title="产品派息/敲出观察">
    <div class="section">
      <p class="desc">展示存续产品（持有中）的派息与敲出观察情况。数据来源为航班服务交易总表 · 产品表。</p>

      <div class="panel">
        <h3 class="panel-title">操作</h3>
        <div class="form-row">
          <label>数据来源</label>
          <div class="file-source">
            <span class="file-badge">📊 航班服务交易总表 · 产品表</span>
            <span class="file-from">本地数据库</span>
          </div>
        </div>
        <div class="form-row">
          <label>搜索</label>
          <input v-model="searchText" type="text" class="input" placeholder="按产品名称或航班编号搜索..." />
        </div>
        <button class="btn btn-primary" :disabled="refreshing" @click="refreshPrices">
          {{ refreshing ? '刷新中...' : '刷新标的价格' }}
        </button>
        <button class="btn btn-secondary" :disabled="generating" @click="generateObservations">
          {{ generating ? '生成中...' : '生成观察记录' }}
        </button>
        <span v-if="lastUpdated" class="update-time">最后更新: {{ lastUpdated }}</span>
        <span v-if="errorMsg" class="error">{{ errorMsg }}</span>
        <span v-if="successMsg" class="success">{{ successMsg }}</span>
      </div>

      <div v-if="filteredProducts.length" class="report-panel">
        <h3 class="section-title">存续产品观察概览</h3>
        <div class="table-wrap">
          <table class="overview-table">
            <thead>
              <tr>
                <th class="col-left sticky-col">航班编号</th>
                <th class="col-left">产品名称</th>
                <th class="col-left">私募管理人</th>
                <th class="col-left">持有状态</th>
                <th class="col-left">代码</th>
                <th class="col-right">入场价</th>
                <th class="col-left">入场日</th>
                <th class="col-right">存续月</th>
                <th class="col-right">锁定期(月)</th>
                <th class="col-left">最近观察日</th>
                <th class="col-right">标的价格</th>
                <th class="col-right">敲出价</th>
                <th class="col-right">派息线</th>
                <th class="col-center">是否敲出</th>
                <th class="col-center">是否派息</th>
              </tr>
            </thead>
            <tbody>
              <template v-for="p in filteredProducts" :key="p.id">
                <tr class="data-row" @click="toggleExpand(p.id)">
                  <td class="col-left sticky-col">
                    <span class="chevron" :class="{ open: expandedId === p.id }">›</span>
                    {{ p.id }}
                  </td>
                  <td class="col-left">{{ p.name }}</td>
                  <td class="col-left">{{ p.manager }}</td>
                  <td class="col-left">
                    <span class="status-badge">{{ p.holding_status }}</span>
                  </td>
                  <td class="col-left code-cell">{{ p.code }}</td>
                  <td class="col-right">{{ formatPrice(p.entry_price) }}</td>
                  <td class="col-left">{{ p.issue_date || '--' }}</td>
                  <td class="col-right">{{ computeMonthsSince(p) }}</td>
                  <td class="col-right">{{ p.lock_months || '--' }}</td>
                  <td class="col-left">{{ latestObs(p)?.date || '--' }}</td>
                  <td class="col-right">{{ formatPrice(latestObs(p)?.underlying_price) }}</td>
                  <td class="col-right">{{ formatPrice(latestObs(p)?.knockout_price) }}</td>
                  <td class="col-right">{{ formatPrice(latestObs(p)?.dividend_line) }}</td>
                  <td class="col-center" :class="knockoutClass(latestObs(p)?.is_knocked_out)">
                    {{ latestObs(p)?.is_knocked_out || '--' }}
                  </td>
                  <td class="col-center" :class="dividendClass(latestObs(p)?.is_dividend)">
                    {{ latestObs(p)?.is_dividend || '--' }}
                  </td>
                </tr>
                <tr v-if="expandedId === p.id && p.observations.length" class="detail-row">
                  <td colspan="15" class="detail-cell">
                    <div class="detail-label">历史观察日明细</div>
                    <table class="detail-table">
                      <thead>
                        <tr>
                          <th>观察日</th>
                          <th>标的价格</th>
                          <th>敲出价</th>
                          <th>派息线</th>
                          <th>是否敲出</th>
                          <th>是否派息</th>
                        </tr>
                      </thead>
                      <tbody>
                        <tr v-for="obs in p.observations" :key="obs.date">
                          <td>{{ obs.date }}</td>
                          <td>{{ formatPrice(obs.underlying_price) }}</td>
                          <td>{{ formatPrice(obs.knockout_price) }}</td>
                          <td>{{ formatPrice(obs.dividend_line) }}</td>
                          <td :class="knockoutClass(obs.is_knocked_out)">{{ obs.is_knocked_out }}</td>
                          <td :class="dividendClass(obs.is_dividend)">{{ obs.is_dividend }}</td>
                        </tr>
                      </tbody>
                    </table>
                  </td>
                </tr>
                <tr v-if="expandedId === p.id && !p.observations.length" class="detail-row">
                  <td colspan="15" class="detail-cell">
                    <div class="detail-empty">暂无观察日记录</div>
                  </td>
                </tr>
              </template>
            </tbody>
          </table>
        </div>
        <p class="table-summary">共 {{ filteredProducts.length }} 个存续产品</p>
      </div>

      <div v-else-if="loaded && !filteredProducts.length" class="empty-state">
        <p>暂无存续产品数据，请先在「数据准备」页面同步飞书数据。</p>
      </div>
    </div>
  </SubPageLayout>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import SubPageLayout from '../components/SubPageLayout.vue'

const searchText = ref('')
const products = ref([])
const lastUpdated = ref(null)
const loaded = ref(false)
const refreshing = ref(false)
const generating = ref(false)
const errorMsg = ref('')
const successMsg = ref('')
const expandedId = ref(null)

onMounted(() => loadData())

async function loadData() {
  errorMsg.value = ''
  try {
    const res = await fetch('/api/observations')
    const data = await res.json()
    if (!res.ok) throw new Error(data.error || '加载失败')
    products.value = data.products || []
    lastUpdated.value = data.lastUpdated
  } catch (err) {
    errorMsg.value = err.message
  } finally {
    loaded.value = true
  }
}

async function refreshPrices() {
  refreshing.value = true
  errorMsg.value = ''
  successMsg.value = ''
  try {
    const res = await fetch('/api/observations/refresh-prices', { method: 'POST' })
    const data = await res.json()
    if (!res.ok) throw new Error(data.error || '刷新失败')
    successMsg.value = `价格刷新完成：${data.refreshed} 个成功${data.failed ? '，' + data.failed + ' 个失败' : ''}`
    await loadData()
  } catch (err) {
    errorMsg.value = err.message
  } finally {
    refreshing.value = false
  }
}

async function generateObservations() {
  generating.value = true
  errorMsg.value = ''
  successMsg.value = ''
  try {
    const res = await fetch('/api/observations/generate', { method: 'POST' })
    const data = await res.json()
    if (!res.ok) throw new Error(data.error || '生成失败')
    successMsg.value = `生成完成：新增 ${data.generated} 条，更新 ${data.updated} 条`
    await loadData()
  } catch (err) {
    errorMsg.value = err.message
  } finally {
    generating.value = false
  }
}

function toggleExpand(id) {
  expandedId.value = expandedId.value === id ? null : id
}

const filteredProducts = computed(() => {
  if (!searchText.value) return products.value
  const q = searchText.value.toLowerCase()
  return products.value.filter(p =>
    (p.name && p.name.toLowerCase().includes(q)) || p.id.toLowerCase().includes(q)
  )
})

function latestObs(product) {
  if (!product.observations || !product.observations.length) return null
  return product.observations[product.observations.length - 1]
}

function computeMonthsSince(product) {
  if (!product.issue_date) return '--'
  const entry = new Date(product.issue_date)
  const now = new Date()
  return (now.getFullYear() - entry.getFullYear()) * 12 + (now.getMonth() - entry.getMonth())
}

function formatPrice(val) {
  if (val === null || val === undefined) return '--'
  return Number(val).toLocaleString('zh-CN', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
}

function knockoutClass(status) {
  if (status === '是') return 'result-yes-knockout'
  if (status === '否') return 'result-no'
  return ''
}

function dividendClass(status) {
  if (status === '是') return 'result-yes-dividend'
  if (status === '否') return 'result-no'
  return ''
}
</script>

<style scoped>
.desc { color: #6B5C4E; font-size: 14px; line-height: 1.8; margin-bottom: 24px; }

.panel {
  background: #fff;
  border-radius: 12px;
  padding: 24px;
  margin-bottom: 20px;
  border: 1px solid #E8DDD0;
}

.panel-title { font-size: 15px; font-weight: 600; color: #1A1109; margin-bottom: 16px; }

.form-row {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 16px;
}

.form-row > label:first-child {
  font-size: 13px;
  color: #6B5C4E;
  white-space: nowrap;
  width: 70px;
  flex-shrink: 0;
}

.input {
  flex: 1;
  border: 1px solid #E8DDD0;
  border-radius: 6px;
  padding: 8px 12px;
  font-size: 13px;
  outline: none;
  background: #fff;
  color: #1A1109;
}

.input:focus { border-color: #8B7355; }

.file-source { flex: 1; display: flex; align-items: center; gap: 10px; }
.file-badge { font-size: 13px; color: #1A1109; font-weight: 500; }
.file-from { font-size: 12px; color: #A8967E; background: #F5F0E8; padding: 2px 8px; border-radius: 10px; }

.btn { padding: 8px 20px; border-radius: 6px; font-size: 13px; cursor: pointer; border: none; font-weight: 500; margin-right: 8px; }
.btn-primary { background: #C62828; color: #fff; }
.btn-primary:hover:not(:disabled) { background: #B71C1C; }
.btn-primary:disabled { background: #EF9A9A; cursor: not-allowed; }
.btn-secondary { background: #8B7355; color: #fff; }
.btn-secondary:hover:not(:disabled) { background: #7A6348; }
.btn-secondary:disabled { background: #C4B5A5; cursor: not-allowed; }

.update-time { margin-left: 16px; color: #8B7355; font-size: 12px; }
.error { margin-left: 12px; color: #C62828; font-size: 13px; }
.success { margin-left: 12px; color: #2E7D45; font-size: 13px; }

.report-panel {
  background: #fff;
  border-radius: 12px;
  padding: 28px;
  border: 1px solid #E8DDD0;
}

.section-title {
  font-size: 15px;
  font-weight: 700;
  color: #1A1109;
  margin-bottom: 20px;
  padding-bottom: 12px;
  border-bottom: 2px solid #F0EAE0;
}

.table-wrap {
  overflow-x: auto;
  margin-bottom: 12px;
}

.overview-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 12px;
  min-width: 1400px;
}

.overview-table th {
  padding: 10px 12px;
  border-bottom: 1px solid #E8DDD0;
  color: #8B7355;
  font-weight: 600;
  background: #FAF7F4;
  font-size: 11px;
  letter-spacing: 0.02em;
  white-space: nowrap;
  position: sticky;
  top: 0;
  z-index: 1;
}

.data-row {
  cursor: pointer;
  transition: background 0.15s;
}
.data-row:hover { background: #FAF7F4; }

.overview-table td {
  padding: 11px 12px;
  border-bottom: 1px solid #F0EAE0;
  color: #1A1109;
  white-space: nowrap;
}

.col-left { text-align: left; }
.col-right { text-align: right; }
.col-center { text-align: center; }

.sticky-col {
  position: sticky;
  left: 0;
  background: #fff;
  z-index: 2;
}
.data-row:hover .sticky-col { background: #FAF7F4; }
.overview-table th.sticky-col { z-index: 3; background: #FAF7F4; }

.chevron {
  font-size: 14px;
  color: #A8967E;
  transition: transform 0.2s;
  display: inline-block;
  line-height: 1;
  margin-right: 4px;
}
.chevron.open { transform: rotate(90deg); }

.code-cell { font-family: monospace; font-size: 11px; color: #6B5C4E; }

.status-badge {
  font-size: 11px;
  padding: 2px 8px;
  border-radius: 10px;
  background: #E8F4EC;
  color: #2E7D45;
}

.result-yes-knockout {
  color: #C62828;
  font-weight: 600;
  background: #FEF3E2;
  border-radius: 4px;
  padding: 2px 6px;
}

.result-yes-dividend {
  color: #2E7D45;
  font-weight: 600;
  background: #E8F4EC;
  border-radius: 4px;
  padding: 2px 6px;
}

.result-no {
  color: #8B7355;
}

.detail-row td {
  padding: 0;
  border-bottom: 1px solid #F0EAE0;
}

.detail-cell {
  background: #FAFAF8;
}

.detail-label {
  font-size: 11px;
  font-weight: 600;
  color: #8B7355;
  letter-spacing: 0.04em;
  padding: 12px 16px 8px;
}

.detail-empty {
  font-size: 12px;
  color: #A8967E;
  padding: 12px 16px;
}

.detail-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 11px;
  margin: 0 16px 12px;
}

.detail-table th {
  padding: 6px 12px;
  border-bottom: 1px solid #E8DDD0;
  color: #8B7355;
  font-weight: 600;
  background: transparent;
  text-align: left;
}

.detail-table td {
  padding: 6px 12px;
  border-bottom: 1px solid #F0EAE0;
  color: #3D3028;
}

.table-summary {
  font-size: 12px;
  color: #8B7355;
  text-align: right;
  padding-top: 8px;
}

.empty-state {
  text-align: center;
  padding: 48px 24px;
  color: #A8967E;
  font-size: 14px;
  background: #fff;
  border-radius: 12px;
  border: 1px solid #E8DDD0;
}
</style>
```

- [ ] **Step 2: Verify frontend compiles**

```bash
cd frontend && npx vite build --mode development
```
Expected: Build succeeds without TypeScript or compilation errors.

- [ ] **Step 3: Commit**

```bash
git add frontend/views/ProductCompletion.vue
git commit -m "feat: rewrite ProductCompletion as dividend/knockout observation page"
```

---

### Task 9: Update Navigation Labels

**Files:**
- Modify: `frontend/views/Home.vue:15` (nav link label)
- Modify: `frontend/views/Home.vue:58` (flow step label)
- Modify: `frontend/components/SubPageLayout.vue:14` (nav link label)

- [ ] **Step 1: Update Home.vue nav link**

In `frontend/views/Home.vue` line 15, change:
```
<router-link to="/product-completion" class="nav-link">完结与发行</router-link>
```
to:
```
<router-link to="/product-completion" class="nav-link">派息/敲出观察</router-link>
```

- [ ] **Step 2: Update Home.vue flow step**

In `frontend/views/Home.vue` line 58, change:
```javascript
{ name: '完结与发行', path: '/product-completion' },
```
to:
```javascript
{ name: '派息/敲出', path: '/product-completion' },
```

- [ ] **Step 3: Update SubPageLayout.vue nav link**

In `frontend/components/SubPageLayout.vue` line 14, change:
```
<router-link to="/product-completion" class="nav-link">完结与发行</router-link>
```
to:
```
<router-link to="/product-completion" class="nav-link">派息/敲出观察</router-link>
```

- [ ] **Step 4: Commit**

```bash
git add frontend/views/Home.vue frontend/components/SubPageLayout.vue
git commit -m "feat: update nav labels from 完结与发行 to 派息/敲出观察"
```

---

### Task 10: Delete Old Database and Full Integration Test

**Files:**
- Delete: `backend/data.sqlite` (to rebuild with new schema)

- [ ] **Step 1: Delete old database file**

```bash
Remove-Item backend/data.sqlite -ErrorAction SilentlyContinue
```

If the file doesn't exist, that's fine — the command will silently continue.

- [ ] **Step 2: Start the full application**

```bash
npm run dev
```
(This runs both backend and frontend via concurrently)

- [ ] **Step 3: Verify in browser**

1. Open `http://localhost:5174/product-completion` (or whichever port Vite uses)
2. Verify the page shows "产品派息/敲出观察" title
3. The navigation should show "派息/敲出观察" instead of "完结与发行"
4. If no data synced yet, you should see the empty state message
5. If data has been synced (via 数据准备 page), products should appear

- [ ] **Step 4: Test the refresh and generate flow**

1. Sync Feishu data first (via 数据准备 page, or by calling `POST /api/db/sync` if already authorized)
2. Click "生成观察记录" button
3. Verify observations appear in the table
4. Click "刷新标的价格" button
5. Verify prices update

- [ ] **Step 5: Final commit**

```bash
git add -A
git commit -m "feat: product dividend/knockout observation - complete feature"
```
