const fs = require('fs')
const path = require('path')
const initSqlJs = require('sql.js')
const { evaluateObservation, parseRatio, parseFirstKnockoutRatio } = require('../services/observationService')

const dbPath = path.join(__dirname, '..', 'data.sqlite')

function findField(row, patterns) {
  for (const pattern of patterns) {
    const normalized = pattern.replace(/\s+/g, '')
    for (const key of Object.keys(row)) {
      const normalizedKey = key.replace(/\s+/g, '')
      if (normalizedKey === normalized || normalizedKey.includes(normalized)) {
        return row[key]
      }
    }
  }
  return undefined
}

function queryAll(db, sql, params = []) {
  const stmt = db.prepare(sql)
  stmt.bind(params)
  const rows = []
  while (stmt.step()) rows.push(stmt.getAsObject())
  stmt.free()
  return rows
}

function updateProductRatios(db) {
  const products = queryAll(db, 'SELECT id, raw FROM products WHERE raw IS NOT NULL')
  const stmt = db.prepare(`
    UPDATE products
    SET first_knockout_ratio = ?, monthly_decrease = ?, dividend_barrier = ?,
        monthly_coupon = ?, coupon_1st = ?, coupon_2nd = ?, coupon_3rd = ?
    WHERE id = ?
  `)

  let updated = 0
  for (const product of products) {
    let row
    try {
      row = JSON.parse(product.raw)
    } catch {
      continue
    }

    const rawKO = row['敲出']
    const entryPrice = row['入场价']
    const firstKO = rawKO != null && !String(rawKO).includes('*') ? parseFirstKnockoutRatio(rawKO, entryPrice) : 0
    stmt.run([
      firstKO,
      parseRatio(findField(row, ['每月递减'])),
      parseRatio(findField(row, ['派息障碍'])),
      parseRatio(findField(row, ['月票息'])),
      parseRatio(findField(row, ['第一段票息'])),
      parseRatio(findField(row, ['第二段票息'])),
      parseRatio(findField(row, ['第三段票息'])),
      product.id,
    ])
    updated++
  }
  stmt.free()
  return updated
}

function recalculateObservations(db) {
  const rows = queryAll(db, `
    SELECT
      o.product_id, o.observation_date, o.underlying_price, o.months_since_entry,
      p.entry_price, p.first_knockout_ratio, p.monthly_decrease, p.lock_months,
      p.dividend_barrier
    FROM observations o
    JOIN products p ON p.id = o.product_id
    WHERE o.underlying_price IS NOT NULL
  `)
  const stmt = db.prepare(`
    UPDATE observations
    SET knockout_price = ?, dividend_line = ?, is_knocked_out = ?, is_dividend = ?,
        updated_at = ?
    WHERE product_id = ? AND observation_date = ?
  `)

  let updated = 0
  for (const row of rows) {
    const result = evaluateObservation(row, row.observation_date, row.underlying_price, row.months_since_entry)
    stmt.run([
      result.knockout_price,
      result.dividend_line,
      result.is_knocked_out,
      result.is_dividend,
      new Date().toISOString(),
      row.product_id,
      row.observation_date,
    ])
    updated++
  }
  stmt.free()
  return updated
}

async function main() {
  if (!fs.existsSync(dbPath)) {
    throw new Error(`Database not found: ${dbPath}`)
  }

  const SQL = await initSqlJs()
  const db = new SQL.Database(fs.readFileSync(dbPath))
  const productsUpdated = updateProductRatios(db)
  const observationsUpdated = recalculateObservations(db)
  fs.writeFileSync(dbPath, Buffer.from(db.export()))
  db.close()

  console.log(`products updated: ${productsUpdated}`)
  console.log(`observations recalculated: ${observationsUpdated}`)
}

main().catch(err => {
  console.error(err)
  process.exit(1)
})
