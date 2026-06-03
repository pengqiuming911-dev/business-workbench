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

async function fetchLatestPrice(code, retries = 3) {
  const secid = resolveSecId(code)
  if (!secid) throw new Error(`Invalid code: ${code}`)

  const url = `${EASTMONEY_API}?secid=${encodeURIComponent(secid)}&fields=f43,f44,f45,f46,f47,f170`

  for (let attempt = 1; attempt <= retries; attempt++) {
    try {
      const controller = new AbortController()
      const timeout = setTimeout(() => controller.abort(), 5000)

      const res = await fetch(url, {
        headers: {
          'User-Agent': 'Mozilla/5.0',
          'Referer': 'https://quote.eastmoney.com/'
        },
        signal: controller.signal
      })
      clearTimeout(timeout)

      const data = await res.json()

      if (!data || !data.data || data.data.f43 === undefined) {
        throw new Error(`No price data for ${code}: ${JSON.stringify(data)}`)
      }

      return data.data.f43 / 100
    } catch (err) {
      if (attempt === retries) throw err
      const delay = attempt * 1000
      console.log(`[priceService] Retry ${attempt}/${retries} for ${code} after ${delay}ms...`)
      await new Promise(r => setTimeout(r, delay))
    }
  }
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
    if (codes.length > 1) {
      await new Promise(r => setTimeout(r, 500))
    }
  }
  return { results, failed }
}

module.exports = { fetchLatestPrice, fetchAllPrices, resolveSecId, parseCode }
