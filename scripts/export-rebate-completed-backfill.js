const fs = require('fs')
const path = require('path')
const XLSX = require('../frontend/node_modules/xlsx')

const sourcePath = process.argv[2]
const outputPath = process.argv[3]

if (!sourcePath || !outputPath) {
  console.error('Usage: node export-rebate-completed-backfill.js <source.xlsx> <output.json>')
  process.exit(1)
}

const workbook = XLSX.read(fs.readFileSync(sourcePath), { type: 'buffer' })
const sheet = workbook.Sheets[workbook.SheetNames[0]]
const rows = XLSX.utils.sheet_to_json(sheet, { defval: '', raw: false })

function cleanText(value) {
  return String(value ?? '').trim()
}

function cleanAmount(value) {
  const text = cleanText(value).replace(/,/g, '')
  if (!text) return { value: 0, ok: false }
  const parsed = Number(text)
  return Number.isFinite(parsed) ? { value: parsed, ok: true } : { value: 0, ok: false }
}

function cleanRatio(value) {
  const text = cleanText(value)
  if (!text) return { value: 0, ok: false }
  const normalized = text.endsWith('%') ? text.slice(0, -1) : text
  const parsed = Number(normalized.replace(/,/g, ''))
  if (!Number.isFinite(parsed)) return { value: 0, ok: false }
  return { value: text.endsWith('%') ? parsed / 100 : parsed, ok: true }
}

function cleanDatePart(value, suffix) {
  return cleanText(value).replace(new RegExp(`${suffix}$`), '')
}

const data = rows
  .slice(1)
  .map((row) => {
    const expenseAmount = cleanAmount(row.__EMPTY_2)
    const subscribeRatio = cleanRatio(row['渠道返还比例'])
    const managementRatio = cleanRatio(row.__EMPTY)
    const performanceRatio = cleanRatio(row.__EMPTY_1)
    const paymentTime = cleanText(row.__EMPTY_3).replace(/\//g, '-')
    const paymentYear = cleanDatePart(row.__EMPTY_4, '年')
    const paymentMonth = cleanDatePart(row.__EMPTY_5, '月')
    const paymentDay = cleanDatePart(row.__EMPTY_6, '日')

    return {
      order_id: cleanText(row['订单号']),
      expense_category: cleanText(row['支出流水明细']),
      expense_amount: expenseAmount.value,
      has_expense_amount: expenseAmount.ok,
      payment_time: paymentTime,
      payment_year: paymentYear,
      payment_month: paymentMonth,
      payment_day: paymentDay,
      channel_subscribe_ratio: subscribeRatio.value,
      has_channel_subscribe_ratio: subscribeRatio.ok,
      channel_management_ratio: managementRatio.value,
      has_channel_management_ratio: managementRatio.ok,
      channel_performance_ratio: performanceRatio.value,
      has_channel_performance_ratio: performanceRatio.ok,
    }
  })
  .filter((row) => row.order_id)

fs.mkdirSync(path.dirname(outputPath), { recursive: true })
fs.writeFileSync(outputPath, JSON.stringify(data, null, 2))
console.log(`wrote ${data.length} rows to ${outputPath}`)
