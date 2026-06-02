const axios = require('axios')

const EASTMONEY_API = 'https://push2.eastmoney.com/api/qt/stock/get'

function resolveSecId(code) {
  if (!code) return null
  const cleaned = code.trim().toLowerCase()
  if (cleaned.startsWith('sh') || cleaned.startsWith('1.') || cleaned.startsWith('5')) {
    return `1.${cleaned.replace(/^(sh|1\.)/, '')}`
  }
  if (cleaned.startsWith('sz') || cleaned.startsWith('0.') || cleaned.startsWith('3')) {
    return `0.${cleaned.replace(/^(sz|0\.)/, '')}`
  }
  return `1.${cleaned}`
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
  const price = rawPrice > 10000 ? rawPrice / 100 : rawPrice
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

module.exports = { fetchLatestPrice, fetchAllPrices, resolveSecId }
