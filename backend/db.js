const Database = require('better-sqlite3')
const path = require('path')

const db = new Database(path.join(__dirname, 'data.sqlite'))

// 建表（co_invest_users 每次重建以匹配新字段）
db.exec(`
  DROP TABLE IF EXISTS co_invest_users;
  DROP TABLE IF EXISTS co_invest_sync_log;
  DROP TABLE IF EXISTS customer_product_link;

  CREATE TABLE IF NOT EXISTS products (
    id TEXT PRIMARY KEY,
    name TEXT,
    is_main INTEGER,
    issue_date TEXT,
    complete_date TEXT,
    subscribe_amount REAL,
    outstanding_amount REAL,
    raw TEXT
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

  -- 客户-产品关联表（通过用户提供的数据同步）
  CREATE TABLE IF NOT EXISTS customer_product_link (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    product_id TEXT,
    user_name TEXT,
    actual_buyer TEXT
  );

  -- 交易表
  CREATE TABLE IF NOT EXISTS transactions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    transaction_date TEXT,
    flight_id TEXT,
    counterparty TEXT,
    subscribe_amount REAL,
    raw TEXT
  );

  -- 渠道表
  CREATE TABLE IF NOT EXISTS channels (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    channel_name TEXT,
    raw TEXT
  );

  -- 直客来源表
  CREATE TABLE IF NOT EXISTS direct_customer_sources (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    source_name TEXT,
    raw TEXT
  );

  -- 客户表
  CREATE TABLE IF NOT EXISTS customers (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    customer_name TEXT,
    raw TEXT
  );

  -- 产品库文档表
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

  -- 产品库文档同步日志
  CREATE TABLE IF NOT EXISTS product_docs_sync_log (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    synced_at TEXT,
    doc_count INTEGER,
    folder_count INTEGER
  );
`)

// 批量写入产品数据（先清空再插入）
function importProducts(rows) {
  const insert = db.prepare(`
    INSERT OR REPLACE INTO products
      (id, name, is_main, issue_date, complete_date, subscribe_amount, outstanding_amount, raw)
    VALUES
      (@id, @name, @is_main, @issue_date, @complete_date, @subscribe_amount, @outstanding_amount, @raw)
  `)

  const insertMany = db.transaction((rows) => {
    const incomingIds = new Set(rows.map(r => r.id))
    const existingIds = db.prepare('SELECT id FROM products').all().map(r => r.id)
    for (const id of existingIds) {
      if (!incomingIds.has(id)) {
        db.prepare('DELETE FROM products WHERE id = ?').run(id)
      }
    }
    for (const r of rows) {
      insert.run(r)
    }
  })

  insertMany(rows)
}

// 记录同步日志
function logSync(rowCount) {
  db.prepare('INSERT INTO sync_log (synced_at, row_count) VALUES (?, ?)').run(
    new Date().toISOString(), rowCount
  )
}

// 查询最近同步时间
function getLastSync() {
  return db.prepare('SELECT synced_at, row_count FROM sync_log ORDER BY id DESC LIMIT 1').get()
}

// 查询产品数据（按完结日期或发行日期范围）
function queryProducts({ startDate, endDate }) {
  return db.prepare(`
    SELECT * FROM products
    WHERE (
        (complete_date >= ? AND complete_date <= ?)
        OR (issue_date >= ? AND issue_date <= ?)
      )
  `).all(startDate, endDate, startDate, endDate)
}

// 批量写入合投用户数据（先清空再插入）
function importCoInvestUsers(rows) {
  const insert = db.prepare(`
    INSERT INTO co_invest_users (user_name, actual_buyer, phone, wechat, total_assets, risk_tolerance, industry, is_actual_deal, lead_source, asset_match, is_dedicated_account, is_competitor, raw)
    VALUES (@user_name, @actual_buyer, @phone, @wechat, @total_assets, @risk_tolerance, @industry, @is_actual_deal, @lead_source, @asset_match, @is_dedicated_account, @is_competitor, @raw)
  `)

  db.prepare('DELETE FROM co_invest_users').run()

  const insertMany = db.transaction((rows) => {
    for (const r of rows) {
      insert.run(r)
    }
  })

  insertMany(rows)
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
  return db.prepare(sql).all(...params)
}

// 获取所有行业的唯一值（用于下拉）
function getDistinctIndustries() {
  return db.prepare('SELECT DISTINCT industry FROM co_invest_users WHERE industry IS NOT NULL AND industry != "" ORDER BY industry').all().map(r => r.industry)
}

// 获取客户-产品关联列表
function getCustomerProductLinks() {
  return db.prepare('SELECT * FROM customer_product_link').all()
}

// 批量写入客户-产品关联
function importCustomerProductLinks(rows) {
  db.prepare('DELETE FROM customer_product_link').run()
  const insert = db.prepare('INSERT INTO customer_product_link (product_id, user_name, actual_buyer) VALUES (@product_id, @user_name, @actual_buyer)')
  const insertMany = db.transaction((rows) => {
    for (const r of rows) insert.run(r)
  })
  insertMany(rows)
}

// 批量写入交易表
function importTransactions(rows) {
  db.prepare('DELETE FROM transactions').run()
  const insert = db.prepare(`INSERT INTO transactions (transaction_date, flight_id, counterparty, subscribe_amount, raw) VALUES (@transaction_date, @flight_id, @counterparty, @subscribe_amount, @raw)`)
  const insertMany = db.transaction((rows) => {
    for (const r of rows) insert.run(r)
  })
  insertMany(rows)
}

// 批量写入渠道表
function importChannels(rows) {
  db.prepare('DELETE FROM channels').run()
  const insert = db.prepare(`INSERT INTO channels (channel_name, raw) VALUES (@channel_name, @raw)`)
  const insertMany = db.transaction((rows) => {
    for (const r of rows) insert.run(r)
  })
  insertMany(rows)
}

// 批量写入直客来源表
function importDirectCustomerSources(rows) {
  db.prepare('DELETE FROM direct_customer_sources').run()
  const insert = db.prepare(`INSERT INTO direct_customer_sources (source_name, raw) VALUES (@source_name, @raw)`)
  const insertMany = db.transaction((rows) => {
    for (const r of rows) insert.run(r)
  })
  insertMany(rows)
}

// 批量写入客户表
function importCustomers(rows) {
  db.prepare('DELETE FROM customers').run()
  const insert = db.prepare(`INSERT INTO customers (customer_name, raw) VALUES (@customer_name, @raw)`)
  const insertMany = db.transaction((rows) => {
    for (const r of rows) insert.run(r)
  })
  insertMany(rows)
}

// 计算客户历史存量峰值和峰值差额
// 历史存量峰值：对每个航班日期（issue_date），计算在该日期之前的产品中，
//             完结时间在该日期之后或完结时间为0的产品金额合计，取最大值
// 峰值差额：历史存量峰值 - 当前存续产品金额合计
function computeUserPeakBalances() {
  const links = db.prepare('SELECT * FROM customer_product_link').all()
  const products = db.prepare('SELECT * FROM products').all()

  const result = {} // { actual_buyer: { peak_balance, peak_diff, current_outstanding } }

  for (const link of links) {
    const buyer = link.actual_buyer
    if (!buyer) continue

    // 找出该客户关联的所有产品
    const customerProducts = products.filter(p => {
      return link.product_id === p.id
    })

    // 当前存续产品（未完结）：完结时间在今天之后 或 完结时间为0
    const now = new Date().toISOString().slice(0, 10)
    const activeProducts = customerProducts.filter(p => {
      const notCompleted = !p.complete_date || p.complete_date === '' || p.complete_date === '0' || p.complete_date > now
      return notCompleted
    })
    const currentOutstanding = activeProducts.reduce((sum, p) => sum + (p.outstanding_amount || 0), 0)

    // 计算峰值：遍历每个航班日期，找到历史最大存量
    let peakBalance = currentOutstanding
    if (customerProducts.length > 0) {
      // 收集该客户所有的航班日期（issue_date）
      const flightDates = new Set()
      for (const p of customerProducts) {
        if (p.issue_date) flightDates.add(p.issue_date)
      }
      const sortedDates = Array.from(flightDates).sort()

      for (const flightDate of sortedDates) {
        // 在该航班日期之前的所有产品中，存续状态的产品金额合计
        // 存续状态：完结时间在该航班日期之后 或 完结时间为0
        const activeOnFlightDate = customerProducts.filter(p => {
          // 产品在该航班日期之前已进场
          const issuedBefore = !p.issue_date || p.issue_date <= flightDate
          // 产品尚未完结：完结时间 > 航班日期 或 完结时间为0/空
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

// 记录合投用户同步日志
function logCoInvestSync(rowCount) {
  db.prepare('INSERT INTO co_invest_sync_log (synced_at, row_count) VALUES (?, ?)').run(
    new Date().toISOString(), rowCount
  )
}

// 查询最近合投用户同步时间
function getLastCoInvestSync() {
  return db.prepare('SELECT synced_at, row_count FROM co_invest_sync_log ORDER BY id DESC LIMIT 1').get()
}

// 批量写入产品文档数据
function importProductDocs(docs) {
  const insert = db.prepare(`
    INSERT OR REPLACE INTO product_docs
      (doc_token, doc_name, parent_path, folder_token, raw_content, structure_json, synced_at)
    VALUES
      (@doc_token, @doc_name, @parent_path, @folder_token, @raw_content, @structure_json, @synced_at)
  `)

  db.prepare('DELETE FROM product_docs').run()

  const insertMany = db.transaction((docs) => {
    for (const doc of docs) {
      insert.run(doc)
    }
  })

  insertMany(docs)
}

// 记录产品文档同步日志
function logProductDocsSync(docCount, folderCount) {
  db.prepare('INSERT INTO product_docs_sync_log (synced_at, doc_count, folder_count) VALUES (?, ?, ?)').run(
    new Date().toISOString(), docCount, folderCount
  )
}

// 查询最近产品文档同步时间
function getLastProductDocsSync() {
  return db.prepare('SELECT synced_at, doc_count, folder_count FROM product_docs_sync_log ORDER BY id DESC LIMIT 1').get()
}

// 查询所有产品文档
function getAllProductDocs() {
  return db.prepare('SELECT * FROM product_docs ORDER BY parent_path, doc_name').all()
}

// 按月份查询产品文档（忽略空格差异）
function getProductDocsByMonth(month) {
  // 移除月份中的所有空格来匹配
  const normalizedMonth = month.replace(/\s+/g, '')
  return db.prepare('SELECT * FROM product_docs ORDER BY parent_path, doc_name').all().filter(doc => {
    const normalizedPath = doc.parent_path.replace(/\s+/g, '')
    return normalizedPath.includes(normalizedMonth)
  })
}

module.exports = {
  importProducts, logSync, getLastSync, queryProducts,
  importCoInvestUsers, logCoInvestSync, getLastCoInvestSync,
  queryCoInvestUsers, getDistinctIndustries,
  getCustomerProductLinks, importCustomerProductLinks,
  importTransactions, importChannels, importDirectCustomerSources, importCustomers,
  computeUserPeakBalances,
  importProductDocs, logProductDocsSync, getLastProductDocsSync, getAllProductDocs, getProductDocsByMonth
}
