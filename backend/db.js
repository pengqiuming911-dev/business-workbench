const initSqlJs = require('sql.js')
const path = require('path')
const fs = require('fs')

const DB_PATH = path.join(__dirname, 'data.sqlite')

let db = null

// sql.js 需要异步初始化，导出 init 函数供 index.js 调用
async function initDatabase() {
  const SQL = await initSqlJs()
  // 如果已有数据库文件，加载它；否则新建
  if (fs.existsSync(DB_PATH)) {
    const buffer = fs.readFileSync(DB_PATH)
    db = new SQL.Database(buffer)
  } else {
    db = new SQL.Database()
  }

  // 建表
  db.exec(`
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

    CREATE TABLE IF NOT EXISTS sync_log (
      id INTEGER PRIMARY KEY AUTOINCREMENT,
      synced_at TEXT,
      row_count INTEGER
    );

    CREATE TABLE IF NOT EXISTS co_invest_users (
      id INTEGER PRIMARY KEY AUTOINCREMENT,
      user_name TEXT,
      actual_buyer TEXT,
      phone TEXT,
      wechat TEXT,
      total_assets REAL,
      risk_tolerance TEXT,
      industry TEXT,
      is_actual_deal TEXT,
      lead_source TEXT,
      asset_match TEXT,
      is_dedicated_account TEXT,
      is_competitor TEXT,
      raw TEXT
    );

    CREATE TABLE IF NOT EXISTS co_invest_sync_log (
      id INTEGER PRIMARY KEY AUTOINCREMENT,
      synced_at TEXT,
      row_count INTEGER
    );

    CREATE TABLE IF NOT EXISTS customer_product_link (
      id INTEGER PRIMARY KEY AUTOINCREMENT,
      product_id TEXT,
      user_name TEXT,
      actual_buyer TEXT
    );

    CREATE TABLE IF NOT EXISTS transactions (
      id INTEGER PRIMARY KEY AUTOINCREMENT,
      transaction_date TEXT,
      flight_id TEXT,
      counterparty TEXT,
      subscribe_amount REAL,
      raw TEXT
    );

    CREATE TABLE IF NOT EXISTS channels (
      id INTEGER PRIMARY KEY AUTOINCREMENT,
      channel_name TEXT,
      raw TEXT
    );

    CREATE TABLE IF NOT EXISTS direct_customer_sources (
      id INTEGER PRIMARY KEY AUTOINCREMENT,
      source_name TEXT,
      raw TEXT
    );

    CREATE TABLE IF NOT EXISTS customers (
      id INTEGER PRIMARY KEY AUTOINCREMENT,
      customer_name TEXT,
      raw TEXT
    );

    CREATE TABLE IF NOT EXISTS product_docs (
      id INTEGER PRIMARY KEY AUTOINCREMENT,
      doc_token TEXT UNIQUE,
      doc_name TEXT,
      parent_path TEXT,
      folder_token TEXT,
      raw_content TEXT,
      structure_json TEXT,
      synced_at TEXT
    );

    CREATE TABLE IF NOT EXISTS product_docs_sync_log (
      id INTEGER PRIMARY KEY AUTOINCREMENT,
      synced_at TEXT,
      doc_count INTEGER,
      folder_count INTEGER
    );
  `)

  saveDatabase()
  console.log('数据库初始化完成')
}

// 保存数据库到文件（sql.js 是内存数据库，需要手动持久化）
function saveDatabase() {
  const data = db.export()
  const buffer = Buffer.from(data)
  fs.writeFileSync(DB_PATH, buffer)
}

// 辅助：执行带参数的 INSERT/UPDATE/DELETE 语句
function runStatement(sql, params) {
  db.run(sql, params)
}

// 辅助：执行查询并返回所有行（对象数组）
function queryAll(sql, params) {
  const stmt = db.prepare(sql)
  if (params && params.length > 0) {
    stmt.bind(params)
  }
  const results = []
  while (stmt.step()) {
    results.push(stmt.getAsObject())
  }
  stmt.free()
  return results
}

// 辅助：执行查询返回一行
function queryOne(sql, params) {
  const stmt = db.prepare(sql)
  if (params && params.length > 0) {
    stmt.bind(params)
  }
  let result = null
  if (stmt.step()) {
    result = stmt.getAsObject()
  }
  stmt.free()
  return result
}

// ──── 产品表 ────

// 批量写入产品数据（先清空再插入）
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

// 记录同步日志
function logSync(rowCount) {
  runStatement('INSERT INTO sync_log (synced_at, row_count) VALUES (?, ?)', [new Date().toISOString(), rowCount])
  saveDatabase()
}

// 查询最近同步时间
function getLastSync() {
  return queryOne('SELECT synced_at, row_count FROM sync_log ORDER BY id DESC LIMIT 1')
}

// 查询产品数据（按完结日期或发行日期范围）
function queryProducts({ startDate, endDate }) {
  return queryAll(`
    SELECT * FROM products
    WHERE (
        (complete_date >= ? AND complete_date <= ?)
        OR (issue_date >= ? AND issue_date <= ?)
      )
  `, [startDate, endDate, startDate, endDate])
}

// ──── 合投用户表 ────

// 批量写入合投用户数据（先清空再插入）
function importCoInvestUsers(rows) {
  runStatement('DELETE FROM co_invest_users')
  for (const r of rows) {
    runStatement(`
      INSERT INTO co_invest_users (user_name, actual_buyer, phone, wechat, total_assets, risk_tolerance, industry, is_actual_deal, lead_source, asset_match, is_dedicated_account, is_competitor, raw)
      VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
    `, [r.user_name, r.actual_buyer, r.phone, r.wechat, r.total_assets, r.risk_tolerance, r.industry, r.is_actual_deal, r.lead_source, r.asset_match, r.is_dedicated_account, r.is_competitor, r.raw])
  }
  saveDatabase()
}

// 查询合投用户（支持多条件过滤）
function queryCoInvestUsers({ actualBuyer, nominalBuyer, isDedicated, isCompetitor, industry }) {
  let sql = 'SELECT * FROM co_invest_users WHERE 1=1'
  const params = []

  if (actualBuyer) {
    sql += ' AND actual_buyer LIKE ?'
    params.push(`%${actualBuyer}%`)
  }
  if (nominalBuyer) {
    sql += ' AND user_name LIKE ?'
    params.push(`%${nominalBuyer}%`)
  }
  if (isDedicated && isDedicated !== 'all') {
    sql += ' AND is_dedicated_account = ?'
    params.push(isDedicated)
  }
  if (isCompetitor && isCompetitor !== 'all') {
    sql += ' AND is_competitor = ?'
    params.push(isCompetitor)
  }
  if (industry && industry !== 'all') {
    sql += ' AND industry = ?'
    params.push(industry)
  }

  sql += ' ORDER BY id'
  return queryAll(sql, params)
}

// 获取所有行业的唯一值（用于下拉）
function getDistinctIndustries() {
  const rows = queryAll('SELECT DISTINCT industry FROM co_invest_users WHERE industry IS NOT NULL AND industry != "" ORDER BY industry')
  return rows.map(r => r.industry)
}

// 记录合投用户同步日志
function logCoInvestSync(rowCount) {
  runStatement('INSERT INTO co_invest_sync_log (synced_at, row_count) VALUES (?, ?)', [new Date().toISOString(), rowCount])
  saveDatabase()
}

// 查询最近合投用户同步时间
function getLastCoInvestSync() {
  return queryOne('SELECT synced_at, row_count FROM co_invest_sync_log ORDER BY id DESC LIMIT 1')
}

// ──── 客户-产品关联表 ────

// 获取客户-产品关联列表
function getCustomerProductLinks() {
  return queryAll('SELECT * FROM customer_product_link')
}

// 批量写入客户-产品关联
function importCustomerProductLinks(rows) {
  runStatement('DELETE FROM customer_product_link')
  for (const r of rows) {
    runStatement('INSERT INTO customer_product_link (product_id, user_name, actual_buyer) VALUES (?, ?, ?)', [r.product_id, r.user_name, r.actual_buyer])
  }
  saveDatabase()
}

// ──── 交易表 ────

// 批量写入交易表
function importTransactions(rows) {
  runStatement('DELETE FROM transactions')
  for (const r of rows) {
    runStatement('INSERT INTO transactions (transaction_date, flight_id, counterparty, subscribe_amount, raw) VALUES (?, ?, ?, ?, ?)', [r.transaction_date, r.flight_id, r.counterparty, r.subscribe_amount, r.raw])
  }
  saveDatabase()
}

// ──── 渠道表 ────

// 批量写入渠道表
function importChannels(rows) {
  runStatement('DELETE FROM channels')
  for (const r of rows) {
    runStatement('INSERT INTO channels (channel_name, raw) VALUES (?, ?)', [r.channel_name, r.raw])
  }
  saveDatabase()
}

// ──── 直客来源表 ────

// 批量写入直客来源表
function importDirectCustomerSources(rows) {
  runStatement('DELETE FROM direct_customer_sources')
  for (const r of rows) {
    runStatement('INSERT INTO direct_customer_sources (source_name, raw) VALUES (?, ?)', [r.source_name, r.raw])
  }
  saveDatabase()
}

// ──── 客户表 ────

// 批量写入客户表
function importCustomers(rows) {
  runStatement('DELETE FROM customers')
  for (const r of rows) {
    runStatement('INSERT INTO customers (customer_name, raw) VALUES (?, ?)', [r.customer_name, r.raw])
  }
  saveDatabase()
}

// ──── 峰值计算 ────

// 计算客户历史存量峰值和峰值差额
function computeUserPeakBalances() {
  const links = queryAll('SELECT * FROM customer_product_link')
  const products = queryAll('SELECT * FROM products')

  const result = {}

  for (const link of links) {
    const buyer = link.actual_buyer
    if (!buyer) continue

    // 找出该客户关联的所有产品
    const customerProducts = products.filter(p => link.product_id === p.id)

    // 当前存续产品（未完结）
    const now = new Date().toISOString().slice(0, 10)
    const activeProducts = customerProducts.filter(p => {
      const notCompleted = !p.complete_date || p.complete_date === '' || p.complete_date === '0' || p.complete_date > now
      return notCompleted
    })
    const currentOutstanding = activeProducts.reduce((sum, p) => sum + (p.outstanding_amount || 0), 0)

    // 计算峰值
    let peakBalance = currentOutstanding
    if (customerProducts.length > 0) {
      const flightDates = new Set()
      for (const p of customerProducts) {
        if (p.issue_date) flightDates.add(p.issue_date)
      }
      const sortedDates = Array.from(flightDates).sort()

      for (const flightDate of sortedDates) {
        const activeOnFlightDate = customerProducts.filter(p => {
          const issuedBefore = !p.issue_date || p.issue_date <= flightDate
          const notCompleted = !p.complete_date || p.complete_date === '' || p.complete_date === '0' || p.complete_date > flightDate
          return issuedBefore && notCompleted
        })
        const balanceOnDate = activeOnFlightDate.reduce((sum, p) => sum + (p.outstanding_amount || 0), 0)
        if (balanceOnDate > peakBalance) peakBalance = balanceOnDate
      }
    }

    result[buyer] = {
      peak_balance: peakBalance,
      peak_diff: peakBalance - currentOutstanding,
      current_outstanding: currentOutstanding
    }
  }

  return result
}

// ──── 产品库文档表 ────

// 批量写入产品文档数据
function importProductDocs(docs) {
  runStatement('DELETE FROM product_docs')
  for (const doc of docs) {
    runStatement(`
      INSERT OR REPLACE INTO product_docs
        (doc_token, doc_name, parent_path, folder_token, raw_content, structure_json, synced_at)
      VALUES (?, ?, ?, ?, ?, ?, ?)
    `, [doc.doc_token, doc.doc_name, doc.parent_path, doc.folder_token, doc.raw_content, doc.structure_json, doc.synced_at])
  }
  saveDatabase()
}

// 记录产品文档同步日志
function logProductDocsSync(docCount, folderCount) {
  runStatement('INSERT INTO product_docs_sync_log (synced_at, doc_count, folder_count) VALUES (?, ?, ?)', [new Date().toISOString(), docCount, folderCount])
  saveDatabase()
}

// 查询最近产品文档同步时间
function getLastProductDocsSync() {
  return queryOne('SELECT synced_at, doc_count, folder_count FROM product_docs_sync_log ORDER BY id DESC LIMIT 1')
}

// 查询所有产品文档
function getAllProductDocs() {
  return queryAll('SELECT * FROM product_docs ORDER BY parent_path, doc_name')
}

// 按月份查询产品文档
function getProductDocsByMonth(month) {
  const normalizedMonth = month.replace(/\s+/g, '')
  const docs = queryAll('SELECT * FROM product_docs ORDER BY parent_path, doc_name')
  return docs.filter(doc => {
    const normalizedPath = doc.parent_path.replace(/\s+/g, '')
    return normalizedPath.includes(normalizedMonth)
  })
}

// ──── 存续产品查询 ────

function queryOngoingProducts() {
  return queryAll(`SELECT * FROM products WHERE holding_status LIKE '%存续%' OR holding_status LIKE '%持有%'`)
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