const test = require('node:test')
const assert = require('node:assert')

const {
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
  generatePosterData,
} = require('./posterService')

test('posterService tests passed', async (t) => {
  await t.test('computeMonthlyCoupon: 月票息有值时直接返回', () => {
    const p = { monthly_coupon: 0.012, duration_months: 12, coupon_1st: 0.144, coupon_2nd: 0 }
    const result = computeMonthlyCoupon(p)
    assert.strictEqual(result, 0.012)
  })

  await t.test('computeMonthlyCoupon: 月票息无值，存续期≤12月用第一段票息', () => {
    const p = { monthly_coupon: 0, duration_months: 10, coupon_1st: 0.12, coupon_2nd: 0 }
    const result = computeMonthlyCoupon(p)
    assert.ok(Math.abs(result - 0.12) < 0.0001)
  })

  await t.test('computeMonthlyCoupon: 月票息无值，存续期>12月用第二段票息', () => {
    const p = { monthly_coupon: 0, duration_months: 24, coupon_1st: 0.12, coupon_2nd: 0.144 }
    const result = computeMonthlyCoupon(p)
    assert.ok(Math.abs(result - 0.144) < 0.0001)
  })

  await t.test('computeAbsoluteReturn: 月票息有值', () => {
    const p = { monthly_coupon: 0.012, duration_months: 12, months_since_entry: 6, coupon_1st: 0, coupon_2nd: 0 }
    const result = computeAbsoluteReturn(p)
    assert.ok(Math.abs(result - 0.072) < 0.0001)
  })

  await t.test('computeAbsoluteReturn: 第一段票息，存续期≤12月', () => {
    const p = { monthly_coupon: 0, duration_months: 10, coupon_1st: 0.12, coupon_2nd: 0, months_since_entry: 6 }
    const result = computeAbsoluteReturn(p)
    assert.ok(Math.abs(result - 0.06) < 0.0001)
  })

  await t.test('computeAnnualizedReturn: 月票息有值', () => {
    const p = { monthly_coupon: 0.0133, duration_months: 0 }
    const result = computeAnnualizedReturn(p)
    assert.ok(Math.abs(result - 0.1596) < 0.001)
  })

  await t.test('computeAnnualizedReturn: 存续期≤12月用第一段票息', () => {
    const p = { monthly_coupon: 0, duration_months: 10, coupon_1st: 0.12, coupon_2nd: 0 }
    const result = computeAnnualizedReturn(p)
    assert.ok(Math.abs(result - 0.12) < 0.0001)
  })

  await t.test('computeAnnualizedReturn: 存续期>12月用第二段票息', () => {
    const p = { monthly_coupon: 0, duration_months: 24, coupon_1st: 0.12, coupon_2nd: 0.144 }
    const result = computeAnnualizedReturn(p)
    assert.ok(Math.abs(result - 0.144) < 0.0001)
  })

  await t.test('getUnderlyingName: 只取括号前的文字', () => {
    assert.strictEqual(getUnderlyingName('中证1000指数 (H3399.CSI)'), '中证1000指数')
    assert.strictEqual(getUnderlyingName('恒科ETF'), '恒科ETF')
    assert.strictEqual(getUnderlyingName('沪深300（SH000300）'), '沪深300')
  })

  await t.test('getKnockoutPercent: 计算敲出百分比', () => {
    const p = { first_knockout_ratio: 1.04, lock_months: 3, monthly_decrease: 0.005 }
    const result = getKnockoutPercent(p, 6)
    assert.ok(Math.abs(result - 102.5) < 0.01)
  })

  await t.test('computeDividendCount: 计算派息次数', () => {
    const count = computeDividendCount({ issue_date: '2025-01-01' }, '2025-04-01')
    assert.strictEqual(count, 3)
  })

  await t.test('computeCumulativeDividendRate: 月票息有值', () => {
    const p = { monthly_coupon: 0.0133, duration_months: 0 }
    const result = computeCumulativeDividendRate(p, 3)
    assert.ok(Math.abs(result - 0.0399) < 0.001)
  })

  await t.test('getParachuteValue: 提取降落伞百分比', () => {
    assert.strictEqual(getParachuteValue({ parachute: '80%' }), '80%')
    assert.strictEqual(getParachuteValue({ parachute: '70.5%' }), '70.5%')
    assert.strictEqual(getParachuteValue({ parachute: '' }), null)
  })

  await t.test('parseDurationMonths: 解析期限月数', () => {
    assert.strictEqual(parseDurationMonths('12个月'), 12)
    assert.strictEqual(parseDurationMonths('2年'), 24)
    assert.strictEqual(parseDurationMonths('6月'), 6)
  })

  await t.test('formatChineseDate: 中文日期格式', () => {
    assert.strictEqual(formatChineseDate('2026-04-23'), '2026年4月23日')
    assert.strictEqual(formatChineseDate('2025-12-23'), '2025年12月23日')
  })

  await t.test('generatePosterData: 完整数据生成', () => {
    const product = {
      id: 'H001',
      name: '鹿8号（三期）',
      code: '中证1000指数 (H3399.CSI)',
      monthly_coupon: 0.0133,
      duration_months: 12,
      parachute: '70%',
      first_knockout_ratio: 1.01,
      lock_months: 3,
      monthly_decrease: 0.005,
      dividend_barrier: 0.8,
      issue_date: '2025-12-23',
    }
    const data = generatePosterData(product, '2026-06-23', 6)
    assert.strictEqual(data.product_id, 'H001')
    assert.strictEqual(data.underlying_name, '中证1000指数')
    assert.strictEqual(data.parachute_value, '70%')
    assert.ok(data.has_dividend_observation)
    assert.strictEqual(data.dividend_barrier_value, '80%')
    assert.ok(Math.abs(data.annualized_return - 0.1596) < 0.001)
  })

  await t.test('generatePosterData: 月票息无值时不观察派息但仍计算敲出收益', () => {
    const product = {
      id: 'H002',
      name: '鹿99号（三期）',
      code: '中证1000指数（000852.SH）',
      monthly_coupon: 0,
      coupon_1st: 0.12,
      coupon_2nd: 0.18,
      duration_months: 10,
      parachute: '70%',
      first_knockout_ratio: 1.01,
      lock_months: 3,
      monthly_decrease: 0,
      dividend_barrier: 0.8,
      issue_date: '2025-12-23',
    }
    const data = generatePosterData(product, '2026-04-23', 4)
    assert.strictEqual(data.has_dividend_observation, false)
    assert.strictEqual(data.parachute_value, '70%')
    assert.ok(Math.abs(data.absolute_return - 0.04) < 0.0001)
    assert.ok(Math.abs(data.annualized_return - 0.12) < 0.0001)
  })
})
