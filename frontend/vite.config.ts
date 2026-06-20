import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import https from 'node:https'

function proxyToEastMoney(baseUrl, pathRewrite) {
  return (req, res, next) => {
    const targetPath = pathRewrite(req.url || '')
    const url = new URL(targetPath, baseUrl)
    url.search = req.url.includes('?') ? req.url.slice(req.url.indexOf('?')) : ''
    const options = {
      hostname: url.hostname,
      path: url.pathname + url.search,
      method: 'GET',
      headers: { 'User-Agent': 'Mozilla/5.0', 'Referer': 'https://quote.eastmoney.com/' },
    }
    const proxyReq = https.request(options, (proxyRes) => {
      res.writeHead(proxyRes.statusCode || 200, { 'Content-Type': 'application/json' })
      proxyRes.pipe(res)
    })
    proxyReq.on('error', (err) => { res.writeHead(502, { 'Content-Type': 'application/json' }); res.end(JSON.stringify({ error: err.message })) })
    proxyReq.end()
  }
}

export default defineConfig({
  plugins: [
    vue(),
    {
      name: 'eastmoney-proxy',
      configureServer(server) {
        server.middlewares.use('/_em_kline',
          proxyToEastMoney('https://push2his.eastmoney.com', (u) => '/api/qt/stock/kline/get')
        )
        server.middlewares.use('/_em_quote',
          proxyToEastMoney('https://push2.eastmoney.com', (u) => '/api/qt/stock/get')
        )
      },
    },
  ],
  server: {
    proxy: {
      '/api/agent/chat': {
        target: 'http://localhost:3001',
        changeOrigin: true,
      },
      '/api': 'http://localhost:3001',
      '/public': 'http://localhost:3001',
    },
  },
})
