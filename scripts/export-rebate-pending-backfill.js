const fs = require('fs')
const path = require('path')
const XLSX = require('../frontend/node_modules/xlsx')

const sourcePath = process.argv[2]
const outputPath = process.argv[3]

if (!sourcePath || !outputPath) {
  console.error('Usage: node export-rebate-pending-backfill.js <source.xlsx> <output.json>')
  process.exit(1)
}

const workbook = XLSX.read(fs.readFileSync(sourcePath), { type: 'buffer' })
const sheet = workbook.Sheets[workbook.SheetNames[0]]
const rows = XLSX.utils.sheet_to_json(sheet, { header: 1, defval: '', raw: false })

function text(value) {
  return String(value ?? '').trim()
}

function num(value) {
  const normalized = text(value).replace(/,/g, '').replace(/\s+/g, '')
  if (!normalized || normalized === '-' || normalized === '--') return { ok: false, value: 0 }
  const parsed = Number(normalized)
  return Number.isFinite(parsed) ? { ok: true, value: parsed } : { ok: false, value: 0 }
}

function ratio(value) {
  const raw = text(value)
  if (!raw || raw === '-' || raw === '--') return { ok: false, value: 0 }
  const normalized = raw.endsWith('%') ? raw.slice(0, -1) : raw
  const parsed = Number(normalized.replace(/,/g, '').replace(/\s+/g, ''))
  if (!Number.isFinite(parsed)) return { ok: false, value: 0 }
  return { ok: true, value: raw.endsWith('%') ? parsed / 100 : parsed }
}

const data = rows
  .slice(2)
  .map((row) => {
    const principal = num(row[5])
    const subReceivable = num(row[6])
    const managementReceivable = num(row[7])
    const performanceReceivable = num(row[8])
    const subRatio = ratio(row[9])
    const managementRatio = ratio(row[10])
    const performanceRatio = ratio(row[11])
    const taxSub = ratio(row[12])
    const taxManagement = ratio(row[13])
    const taxPerformance = ratio(row[14])

    return {
      order_id: text(row[0]),
      flight_id: text(row[1]),
      product_name: text(row[2]),
      customer_name: text(row[3]),
      rebate_target: text(row[4]),
      principal: principal.value,
      has_principal: principal.ok,
      subscribe_receivable: subReceivable.value,
      has_subscribe_receivable: subReceivable.ok,
      management_fee_received: managementReceivable.value,
      has_management_fee_received: managementReceivable.ok,
      performance_fee_receivable: performanceReceivable.value,
      has_performance_fee_receivable: performanceReceivable.ok,
      subscribe_fee_ratio: subRatio.value,
      has_subscribe_fee_ratio: subRatio.ok,
      management_fee_ratio: managementRatio.value,
      has_management_fee_ratio: managementRatio.ok,
      performance_fee_ratio: performanceRatio.value,
      has_performance_fee_ratio: performanceRatio.ok,
      tax_subscribe_ratio: taxSub.value,
      has_tax_subscribe_ratio: taxSub.ok,
      tax_management_ratio: taxManagement.value,
      has_tax_management_ratio: taxManagement.ok,
      tax_performance_ratio: taxPerformance.value,
      has_tax_performance_ratio: taxPerformance.ok,
      expected_subscribe: num(row[15]).value,
      has_expected_subscribe: num(row[15]).ok,
      expected_management: num(row[16]).value,
      has_expected_management: num(row[16]).ok,
      expected_performance: num(row[17]).value,
      has_expected_performance: num(row[17]).ok,
      returned_subscribe: num(row[18]).value,
      has_returned_subscribe: num(row[18]).ok,
      returned_management: num(row[19]).value,
      has_returned_management: num(row[19]).ok,
      returned_performance: num(row[20]).value,
      has_returned_performance: num(row[20]).ok,
      outstanding_subscribe: num(row[21]).value,
      has_outstanding_subscribe: num(row[21]).ok,
      outstanding_management: num(row[22]).value,
      has_outstanding_management: num(row[22]).ok,
      outstanding_performance: num(row[23]).value,
      has_outstanding_performance: num(row[23]).ok,
      is_returnable: text(row[24]),
    }
  })
  .filter((row) => row.order_id && row.order_id !== '订单号')

fs.mkdirSync(path.dirname(outputPath), { recursive: true })
fs.writeFileSync(outputPath, JSON.stringify(data, null, 2))
console.log(`wrote ${data.length} rows to ${outputPath}`)
