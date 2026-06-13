const assert = require('node:assert/strict')
const { buildFeishuPushText } = require('./feishuPushService')

const today = '2026-06-08'

const products = [
  {
    id: 'FL001',
    name: '测试敲出产品',
    manager: '测试管理人',
    code: 'sh000300',
    entry_price: 3200,
    observation: {
      underlying_price: 3300,
      knockout_price: 3264,
      dividend_line: 2976,
      is_knocked_out: '是',
      is_dividend: '是',
    },
  },
]

const text = buildFeishuPushText(today, products)

assert.match(text, /今日产品派息\/敲出观察提醒/)
assert.match(text, /2026-06-08/)
assert.match(text, /测试敲出产品/)
assert.match(text, /sh000300/)
assert.match(text, /3300\.00/)
assert.match(text, /3264\.00/)
assert.match(text, /敲出/)
assert.match(text, /派息/)

console.log('feishuPushService tests passed')
