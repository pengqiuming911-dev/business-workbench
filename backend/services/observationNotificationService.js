const { getObservationDates, evaluateObservation } = require('./observationService')

const DEFAULT_RECIPIENT = 'pengqiuming@iyanxuna.cn'

function formatPrice(value) {
  if (value === null || value === undefined || value === '') return '--'
  const num = Number(value)
  if (!Number.isFinite(num)) return String(value)
  return num.toFixed(2)
}

function formatValue(value) {
  return value === null || value === undefined || value === '' ? '--' : String(value)
}

function buildTodayObservationNotification({
  products,
  prices,
  today = new Date().toISOString().slice(0, 10),
  recipient = DEFAULT_RECIPIENT,
}) {
  const notificationProducts = []

  for (const product of products || []) {
    const code = product.code
    if (!code || prices[code] === undefined || prices[code] === null) continue

    const todayObservationDate = getObservationDates(product, today)
      .find(item => item.date === today)
    if (!todayObservationDate) continue

    const observation = evaluateObservation(
      product,
      today,
      prices[code],
      todayObservationDate.monthsSinceEntry
    )

    notificationProducts.push({
      ...product,
      observation,
    })
  }

  const subject = `今日产品派息/敲出观察提醒（${today}，${notificationProducts.length}个）`
  const text = renderText(today, notificationProducts)
  const html = renderHtml(today, notificationProducts)

  return {
    recipient,
    subject,
    text,
    html,
    products: notificationProducts,
  }
}

function renderText(today, products) {
  const lines = [
    `今日产品派息/敲出观察提醒`,
    `观察日期：${today}`,
    `今日需要观察产品数量：${products.length}`,
    '',
  ]

  for (const product of products) {
    const obs = product.observation
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

function renderHtml(today, products) {
  const rows = products.map(product => {
    const obs = product.observation
    return `
      <tr>
        <td>${escapeHtml(product.id)}</td>
        <td>${escapeHtml(product.name)}</td>
        <td>${escapeHtml(product.manager)}</td>
        <td>${escapeHtml(product.code)}</td>
        <td style="text-align:right">${formatPrice(product.entry_price)}</td>
        <td style="text-align:right">${formatPrice(obs.underlying_price)}</td>
        <td style="text-align:right">${formatPrice(obs.knockout_price)}</td>
        <td style="text-align:right">${formatPrice(obs.dividend_line)}</td>
        <td style="text-align:center">${escapeHtml(obs.is_knocked_out)}</td>
        <td style="text-align:center">${escapeHtml(obs.is_dividend)}</td>
      </tr>
    `
  }).join('')

  return `
    <div>
      <h2>今日产品派息/敲出观察提醒</h2>
      <p>观察日期：${escapeHtml(today)}；今日需要观察产品数量：${products.length}</p>
      <table border="1" cellpadding="8" cellspacing="0" style="border-collapse:collapse;font-size:13px;">
        <thead>
          <tr>
            <th>航班编号</th>
            <th>产品名称</th>
            <th>私募管理人</th>
            <th>标的代码</th>
            <th>入场价</th>
            <th>实时标的价格</th>
            <th>敲出价</th>
            <th>派息线</th>
            <th>是否敲出</th>
            <th>是否派息</th>
          </tr>
        </thead>
        <tbody>${rows}</tbody>
      </table>
    </div>
  `
}

function escapeHtml(value) {
  return formatValue(value)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;')
}

function createTransportFromEnv(env = process.env) {
  if (!env.SMTP_HOST || !env.SMTP_USER || !env.SMTP_PASS) return null

  const nodemailer = require('nodemailer')
  const port = Number(env.SMTP_PORT || 465)
  return nodemailer.createTransport({
    host: env.SMTP_HOST,
    port,
    secure: env.SMTP_SECURE ? env.SMTP_SECURE === 'true' : port === 465,
    auth: {
      user: env.SMTP_USER,
      pass: env.SMTP_PASS,
    },
  })
}

async function sendObservationEmail({ notification, env = process.env, transport } = {}) {
  if (!notification || notification.products.length === 0) {
    return { sent: false, reason: 'no-products' }
  }

  const mailTransport = transport || createTransportFromEnv(env)
  if (!mailTransport) {
    return { sent: false, reason: 'smtp-not-configured' }
  }

  await mailTransport.sendMail({
    from: env.SMTP_FROM || env.SMTP_USER,
    to: notification.recipient,
    subject: notification.subject,
    text: notification.text,
    html: notification.html,
  })

  return { sent: true, count: notification.products.length }
}

module.exports = {
  DEFAULT_RECIPIENT,
  buildTodayObservationNotification,
  sendObservationEmail,
  createTransportFromEnv,
}
