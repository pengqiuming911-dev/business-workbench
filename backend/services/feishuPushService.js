const axios = require('axios')

function formatPrice(value) {
  if (value === null || value === undefined || value === '') return '--'
  const num = Number(value)
  if (!Number.isFinite(num)) return String(value)
  return num.toFixed(2)
}

function formatValue(value) {
  return value === null || value === undefined || value === '' ? '--' : String(value)
}

function buildFeishuPushText(today, products) {
  const lines = [
    '今日产品派息/敲出观察提醒',
    `观察日期：${today}`,
    `今日需要观察产品数量：${products.length}`,
    '',
  ]

  for (const product of products) {
    const obs = product.observation || {}
    lines.push(
      `产品：${formatValue(product.name)}`,
      `航班编号：${formatValue(product.id)}`,
      `私募管理人：${formatValue(product.manager)}`,
      `标的代码：${formatValue(product.code)}`,
      `入场价：${formatPrice(product.entry_price)}`,
      `实时标的价格：${formatPrice(obs.underlying_price)}`,
      `敲出价：${formatPrice(obs.knockout_price)}`,
      `派息线：${formatPrice(obs.dividend_line)}`,
      `是否敲出：${formatValue(obs.is_knocked_out)}`,
      `是否派息：${formatValue(obs.is_dividend)}`,
      ''
    )
  }

  return lines.join('\n')
}

async function sendFeishuPush(webhookUrl, text) {
  const res = await axios.post(webhookUrl, {
    msg_type: 'text',
    content: { text },
  })
  if (res.data.code !== 0) {
    throw new Error(`飞书推送失败 (${res.data.code}): ${res.data.msg}`)
  }
  return res.data
}

async function executeObservationPush({ webhookUrl, refreshTodayObservations, buildTodayObservationNotification }) {
  const refreshed = await refreshTodayObservations()
  if (refreshed.codes.length === 0) {
    return { sent: false, reason: 'no-products', count: 0 }
  }

  const notification = buildTodayObservationNotification({
    products: refreshed.products,
    prices: refreshed.prices,
    today: refreshed.today,
  })

  if (notification.products.length === 0) {
    return { sent: false, reason: 'no-observation-today', count: 0 }
  }

  const text = notification.text
  await sendFeishuPush(webhookUrl, text)
  return { sent: true, count: notification.products.length }
}

module.exports = {
  buildFeishuPushText,
  sendFeishuPush,
  executeObservationPush,
}
