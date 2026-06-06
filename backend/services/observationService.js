const { isTradingDay, adjustToNearestTradingDay } = require('./tradingCalendar')

function monthsBetween(entryDate, obsDate) {
  const entry = new Date(entryDate)
  const obs = new Date(obsDate)
  return (obs.getFullYear() - entry.getFullYear()) * 12 + (obs.getMonth() - entry.getMonth())
}

function addMonths(date, months) {
  const d = new Date(date)
  const targetDay = new Date(date).getDate()
  d.setMonth(d.getMonth() + months)
  if (d.getDate() !== targetDay) {
    d.setDate(0)
  }
  return d.toISOString().slice(0, 10)
}

function adjustForHoliday(dateStr, holidayAdjust) {
  if (isTradingDay(dateStr)) return dateStr
  const direction = holidayAdjust === '提前' ? 'advance' : 'postpone'
  return adjustToNearestTradingDay(dateStr, direction)
}

function getObservationDates(product, todayOverride) {
  const entryDate = product.issue_date
  if (!entryDate) return []

  const today = todayOverride || new Date().toISOString().slice(0, 10)
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

function getNextObservationDate(product, todayOverride) {
  const entryDate = product.issue_date
  if (!entryDate) return null

  const today = todayOverride || new Date().toISOString().slice(0, 10)
  let months = 1

  while (months < 600) {
    const rawDate = addMonths(entryDate, months)
    const adjusted = adjustForHoliday(rawDate, product.holiday_adjust)
    if (adjusted > today) return adjusted
    months++
  }

  return null
}

function getObservationDatesForMonth(product, month) {
  if (!/^\d{4}-\d{2}$/.test(month || '')) return []
  const entryDate = product.issue_date
  if (!entryDate) return []

  const monthStart = `${month}-01`
  const monthEnd = new Date(`${monthStart}T00:00:00Z`)
  monthEnd.setUTCMonth(monthEnd.getUTCMonth() + 1)
  monthEnd.setUTCDate(0)
  const monthEndStr = monthEnd.toISOString().slice(0, 10)
  const dates = []
  let months = 1

  while (months < 600) {
    const rawDate = addMonths(entryDate, months)
    if (rawDate > monthEndStr) break
    const adjusted = adjustForHoliday(rawDate, product.holiday_adjust)
    if (adjusted >= monthStart && adjusted <= monthEndStr) {
      dates.push({ date: adjusted, monthsSinceEntry: months })
    }
    months++
  }

  return dates
}

function parseRatio(value) {
  if (value === null || value === undefined || value === '') return 0
  if (typeof value === 'number') {
    if (!Number.isFinite(value)) return 0
    return Math.abs(value) > 2 ? value / 100 : value
  }

  const raw = String(value).trim()
  if (!raw) return 0
  const hasPercent = raw.includes('%')
  const normalized = raw.replace(/,/g, '').replace(/%/g, '').trim()
  const number = Number(normalized)
  if (!Number.isFinite(number)) return 0
  if (hasPercent || Math.abs(number) > 2) return number / 100
  return number
}

function parseFirstKnockoutRatio(value, entryPrice) {
  const numericValue = Number(value)
  const numericEntryPrice = Number(entryPrice)
  if (
    Number.isFinite(numericValue) &&
    Number.isFinite(numericEntryPrice) &&
    numericEntryPrice > 0 &&
    numericValue > 0
  ) {
    const impliedRatio = numericValue / numericEntryPrice
    if (impliedRatio >= 0.5 && impliedRatio <= 2) return impliedRatio
  }
  return parseRatio(value)
}

function computeKnockoutPrice(product, monthsSinceEntry) {
  const firstKnockoutRatio = parseRatio(product.first_knockout_ratio)
  const entryPrice = Number(product.entry_price) || 0
  const lockMonths = Number(product.lock_months) || 0
  const monthlyDecrease = parseRatio(product.monthly_decrease)

  if (!firstKnockoutRatio || !entryPrice) return null
  if (monthsSinceEntry < lockMonths) return null

  const currentRatio = firstKnockoutRatio - (monthsSinceEntry - lockMonths) * monthlyDecrease
  return entryPrice * currentRatio
}

function computeDividendLine(product) {
  return (Number(product.entry_price) || 0) * parseRatio(product.dividend_barrier)
}

function evaluateObservation(product, obsDate, underlyingPrice, monthsSinceEntry) {
  const dividendLine = computeDividendLine(product)
  const isDividend = underlyingPrice >= dividendLine ? '是' : '否'

  const knockoutPrice = computeKnockoutPrice(product, monthsSinceEntry)
  let isKnockedOut = '不观察'
  if (knockoutPrice !== null) {
    isKnockedOut = underlyingPrice >= knockoutPrice ? '是' : '否'
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
  monthsBetween,
  addMonths,
  adjustForHoliday,
  getObservationDates,
  getNextObservationDate,
  getObservationDatesForMonth,
  parseRatio,
  parseFirstKnockoutRatio,
  computeKnockoutPrice,
  computeDividendLine,
  evaluateObservation,
}
