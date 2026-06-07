const assert = require('node:assert/strict')
const {
  buildTodayObservationNotification,
} = require('./observationNotificationService')

const fixedToday = '2026-06-08'

const products = [
  {
    id: 'FL001',
    name: '测试敲出产品',
    manager: '测试管理人',
    holding_status: '存续',
    code: 'sh000300',
    entry_price: 3200,
    first_knockout_ratio: '103%',
    lock_months: 3,
    monthly_decrease: '0.5%',
    issue_date: '2026-01-06',
    subscribe_amount: 1000,
    dividend_barrier: '97%',
    holiday_adjust: '顺延',
    duration_months: 12,
  },
  {
    id: 'FL002',
    name: '非今日观察产品',
    holding_status: '存续',
    code: 'sh000905',
    entry_price: 5000,
    first_knockout_ratio: '100%',
    lock_months: 3,
    monthly_decrease: '0%',
    issue_date: '2026-01-09',
    dividend_barrier: '80%',
  },
]

const notification = buildTodayObservationNotification({
  products,
  prices: { sh000300: 3300, sh000905: 6000 },
  today: fixedToday,
})

assert.equal(notification.products.length, 1)
assert.equal(notification.products[0].id, 'FL001')
assert.equal(notification.products[0].observation.underlying_price, 3300)
assert.equal(notification.products[0].observation.knockout_price, 3264)
assert.equal(notification.products[0].observation.is_knocked_out, '是')
assert.equal(notification.products[0].observation.is_dividend, '是')

assert.match(notification.subject, /今日产品派息\/敲出观察提醒/)
assert.match(notification.text, /测试敲出产品/)
assert.match(notification.text, /sh000300/)
assert.match(notification.text, /3300\.00/)
assert.match(notification.html, /测试敲出产品/)
assert.match(notification.html, /3300\.00/)

console.log('observationNotificationService tests passed')
