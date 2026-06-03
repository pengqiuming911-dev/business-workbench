const axios = require('axios')

const EASTMONEY_API = 'https://push2.eastmoney.com/api/qt/stock/get'

function parseCode(rawCode) {
  if (!rawCode) return null
  const match = rawCode.match(/(\d{6})\.\s*(SH|SZ|sh|sz)/i)
  if (match) {
    return { num: match[1], exchange: match[2].toUpperCase() }
  }
  const cleaned = rawCode.trim().toLowerCase()
  if (/^(sh|sz)\d{6}/.test(cleaned)) {
    return { num: cleaned.slice(2), exchange: cleaned.slice(0, 2).toUpperCase() }
  }
  return null
}

function resolveSecId(code) {
  const parsed = parseCode(code)
  if (!parsed) return null
  const market = parsed.exchange === 'SH' ? 1 : 0
  return `${market}.${parsed.num}`
}

async function fetchLatestPrice(code) {
  const secid = resolveSecId(code)
  if (!secid) throw new Error(`Invalid code: ${code}`)

  const res = await axios.get(EASTMONEY_API, {
    params: {
      secid,
      fields: 'f43,f44,f45,f46,f47,f170'
    },
    timeout: 5000,
    headers: {
      'User-Agent': 'Mozilla/5.0',
      'Referer': 'https://quote.eastmoney.com/'
    }
  })

  if (!res.data || !res.data.data || res.data.data.f43 === undefined) {
    throw new Error(`No price data for ${code}: ${JSON.stringify(res.data)}`)
  }

  const rawPrice = res.data.data.f43
  const price = rawPrice / 100
  return price
}

async function fetchAllPrices(codes) {
  const results = {}
  const failed = []
  for (const code of codes) {
    try {
      results[code] = await fetchLatestPrice(code)
    } catch (err) {
      console.error(`Failed to fetch price for ${code}:`, err.message)
      failed.push(code)
    }
  }
  return { results, failed }
}

module.exports = { fetchLatestPrice, fetchAllPrices, resolveSecId, parseCode }
