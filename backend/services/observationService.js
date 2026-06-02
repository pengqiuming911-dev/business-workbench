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
