const { parseRatio } = require('./observationService')

function monthsBetween(entryDate, targetDate) {
  const entry = new Date(entryDate)
  const target = new Date(targetDate)
  return (target.getFullYear() - entry.getFullYear()) * 12 + (target.getMonth() - entry.getMonth())
}

function formatChineseDate(dateStr) {
  const d = new Date(dateStr)
  return `${d.getFullYear()}年${d.getMonth() + 1}月${d.getDate()}日`
}

function computeMonthlyCoupon(product) {
  const monthlyCoupon = parseRatio(product.monthly_coupon)
  if (monthlyCoupon > 0) return monthlyCoupon

  const durationMonths = Number(product.duration_months) || parseDurationMonths(product.term)

  if (durationMonths > 0 && durationMonths <= 12) {
    return parseRatio(product.coupon_1st)
  }
  if (durationMonths > 12) {
    return parseRatio(product.coupon_2nd)
  }
  return 0
}

function parseDurationMonths(termStr) {
  if (!termStr) return 0
  const match = String(termStr).match(/(\d+)\s*个?月/)
  if (match) return Number(match[1])
  const yearMatch = String(termStr).match(/(\d+)\s*年/)
  if (yearMatch) return Number(yearMatch[1]) * 12
  return 0
}

function computeAbsoluteReturn(product) {
  const monthlyCoupon = parseRatio(product.monthly_coupon)
  const durationMonths = Number(product.duration_months) || parseDurationMonths(product.term)
  const monthsSinceEntry = product.months_since_entry || 0

  if (monthlyCoupon > 0) {
    return monthlyCoupon * monthsSinceEntry
  }

  if (durationMonths > 0 && durationMonths <= 12) {
    const c1 = parseRatio(product.coupon_1st)
    return c1 > 0 ? (c1 / 12) * monthsSinceEntry : 0
  }
  if (durationMonths > 12) {
    const c2 = parseRatio(product.coupon_2nd)
    return c2 > 0 ? (c2 / 12) * monthsSinceEntry : 0
  }
  return 0
}

function computeAnnualizedReturn(product) {
  const monthlyCoupon = parseRatio(product.monthly_coupon)
  const durationMonths = Number(product.duration_months) || parseDurationMonths(product.term)

  if (monthlyCoupon > 0) return monthlyCoupon * 12

  if (durationMonths > 0 && durationMonths <= 12) {
    return parseRatio(product.coupon_1st)
  }
  if (durationMonths > 12) {
    return parseRatio(product.coupon_2nd)
  }
  return 0
}

function getUnderlyingName(code) {
  if (!code) return ''
  const match = code.match(/^([^(（]+)/)
  return match ? match[1].trim() : code
}

function getKnockoutPercent(product, monthsAtKnockout) {
  const firstKORatio = parseRatio(product.first_knockout_ratio)
  const lockMonths = Number(product.lock_months) || 0
  const monthlyDecrease = parseRatio(product.monthly_decrease)
  if (!firstKORatio) return null
  const ratio = firstKORatio - Math.max(0, monthsAtKnockout - lockMonths) * monthlyDecrease
  return Math.round(ratio * 10000) / 100
}

function computeDividendCount(product, date) {
  if (!product.issue_date) return 0
  return monthsBetween(product.issue_date, date)
}

function computeCumulativeDividendRate(product, count) {
  const monthlyCoupon = parseRatio(product.monthly_coupon)
  const durationMonths = Number(product.duration_months) || parseDurationMonths(product.term)

  if (monthlyCoupon > 0) return monthlyCoupon * count

  if (durationMonths > 0 && durationMonths <= 12) {
    const c1 = parseRatio(product.coupon_1st)
    return c1 > 0 ? (c1 / 12) * count : 0
  }
  if (durationMonths > 12) {
    const c2 = parseRatio(product.coupon_2nd)
    return c2 > 0 ? (c2 / 12) * count : 0
  }
  return 0
}

function getParachuteValue(product) {
  const raw = product.parachute
  if (raw === null || raw === undefined || raw === '') return null
  const str = String(raw).trim()
  const match = str.match(/(\d+\.?\d*)/)
  if (match) return `${match[1]}%`
  return null
}

function generatePosterData(product, observationDate, monthsSinceEntry) {
  const hasMonthlyCoupon = parseRatio(product.monthly_coupon) > 0

  return {
    product_id: product.id,
    product_name: product.name || '',
    poster_type: 'placeholder',
    date: observationDate,
    months_since_entry: monthsSinceEntry,
    has_dividend_observation: hasMonthlyCoupon,
    underlying_name: getUnderlyingName(product.code),
    parachute_value: getParachuteValue(product),
    knockout_value: getKnockoutPercent(product, monthsSinceEntry),
    dividend_barrier_value: product.dividend_barrier ? `${Math.round(parseRatio(product.dividend_barrier) * 100)}%` : null,
    entry_date: product.issue_date,
    monthly_coupon: parseRatio(product.monthly_coupon),
    absolute_return: computeAbsoluteReturn({ ...product, months_since_entry: monthsSinceEntry }),
    annualized_return: computeAnnualizedReturn(product),
    dividend_count: computeDividendCount(product, observationDate),
    cumulative_dividend_rate: computeCumulativeDividendRate(product, computeDividendCount(product, observationDate)),
  }
}

module.exports = {
  generatePosterData,
  computeMonthlyCoupon,
  computeAbsoluteReturn,
  computeAnnualizedReturn,
  getUnderlyingName,
  getKnockoutPercent,
  computeDividendCount,
  computeCumulativeDividendRate,
  getParachuteValue,
  parseDurationMonths,
  formatChineseDate,
}
