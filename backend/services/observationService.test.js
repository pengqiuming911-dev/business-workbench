const assert = require('node:assert/strict')
const {
  parseFirstKnockoutRatio,
  computeKnockoutPrice,
  evaluateObservation,
  getNextObservationDate,
  getObservationDatesForMonth,
} = require('./observationService')

function approx(actual, expected) {
  assert.ok(Math.abs(actual - expected) < 1e-8, `expected ${actual} to equal ${expected}`)
}

const percentProduct = {
  entry_price: 3200,
  first_knockout_ratio: '103%',
  monthly_decrease: '0.5%',
  lock_months: 3,
  dividend_barrier: '97%',
}

approx(parseFirstKnockoutRatio(0.76704, 0.752), 1.02)
approx(parseFirstKnockoutRatio(7609.5622, 7534.22), 1.01)

assert.equal(computeKnockoutPrice(percentProduct, 2), null)
approx(computeKnockoutPrice(percentProduct, 3), 3296)
approx(computeKnockoutPrice(percentProduct, 5), 3264)

const atBarrier = evaluateObservation(percentProduct, '2026-06-04', 3264, 5)
assert.equal(atBarrier.is_knocked_out, '是')
assert.equal(atBarrier.knockout_price, 3264)

const belowLock = evaluateObservation(percentProduct, '2026-04-04', 3400, 2)
assert.equal(belowLock.is_knocked_out, '不观察')
assert.equal(belowLock.knockout_price, null)

const scheduleProduct = {
  issue_date: '2026-01-15',
  holiday_adjust: '顺延',
}

assert.equal(getNextObservationDate(scheduleProduct, '2026-02-15'), '2026-02-23')
assert.equal(getNextObservationDate(scheduleProduct, '2026-02-23'), '2026-03-16')
assert.equal(getNextObservationDate(scheduleProduct, '2026-03-16'), '2026-04-15')
assert.equal(getNextObservationDate({ holiday_adjust: '顺延' }, '2026-03-16'), null)

assert.deepEqual(getObservationDatesForMonth(scheduleProduct, '2026-03'), [
  { date: '2026-03-16', monthsSinceEntry: 2 },
])
assert.deepEqual(getObservationDatesForMonth(scheduleProduct, 'bad-month'), [])

console.log('observationService tests passed')
