require('dotenv').config()
const express = require('express')
const cors = require('cors')
const axios = require('axios')
const { initDatabase, importProducts, logSync, getLastSync, queryProducts, importCoInvestUsers, logCoInvestSync, getLastCoInvestSync, queryCoInvestUsers, getDistinctIndustries, getCustomerProductLinks, importCustomerProductLinks, importTransactions, importChannels, importDirectCustomerSources, importCustomers, computeUserPeakBalances, importProductDocs, logProductDocsSync, getLastProductDocsSync, getAllProductDocs, getProductDocsByMonth, queryOngoingProducts, upsertObserv, queryObservationsByProduct, upsertPrice, queryLatestPrice, queryPriceByDate, getLastObservationUpdate, upsertPoster, queryPostersByDate, queryPostersByProduct, queryAllPosters, deletePoster } = require('./db')
const { fetchAllPrices } = require('./services/priceService')
const { getObservationDates, getNextObservationDate, getObservationDatesForMonth, evaluateObservation, parseRatio, parseFirstKnockoutRatio } = require('./services/observationService')
const { generatePosterData, formatChineseDate, computeDividendCount, computeCumulativeDividendRate } = require('./services/posterService')
const {
  buildTodayObservationNotification,
  sendObservationEmail,
} = require('./services/observationNotificationService')

const app = express()
const PORT = process.env.PORT || 3001

const FEISHU_APP_ID = process.env.FEISHU_APP_ID
const FEISHU_APP_SECRET = process.env.FEISHU_APP_SECRET
const FEISHU_REDIRECT_URI = process.env.FEISHU_REDIRECT_URI
const FRONTEND_URL = process.env.FRONTEND_URL || 'http://localhost:5173'

app.use(cors({ origin: FRONTEND_URL, credentials: true }))
app.use(express.json())
app.use('/public', express.static(require('path').join(__dirname, 'public')))

// 内存存储 user_access_token（生产环境应换成 session/数据库）
let userToken = null

// ─────────────────────────────────────────
// 工具函数：获取 app_access_token
// ─────────────────────────────────────────
async function getAppAccessToken() {
  const res = await axios.post('https://open.feishu.cn/open-apis/auth/v3/app_access_token/internal', {
    app_id: FEISHU_APP_ID,
    app_secret: FEISHU_APP_SECRET,
  })
  if (res.data.code !== 0) {
    throw new Error(`获取 app_access_token 失败: ${res.data.msg}`)
  }
  return res.data.app_access_token
}

// ─────────────────────────────────────────
// GET /api/auth/login
// 生成飞书 OAuth 授权 URL，前端重定向到此 URL
// ─────────────────────────────────────────
app.get('/api/auth/login', (req, res) => {
  const url = `https://open.feishu.cn/open-apis/authen/v1/authorize` +
    `?app_id=${FEISHU_APP_ID}` +
    `&redirect_uri=${encodeURIComponent(FEISHU_REDIRECT_URI)}` +
    `&scope=drive:drive%20drive:file%20drive:export:readonly%20space:document:retrieve%20bitable:app:readonly%20bitable:app%20docx:document%20docx:document:readonly` +
    `&response_type=code`
  res.json({ url })
})

// ─────────────────────────────────────────
// GET /api/auth/callback
// 飞书回调，用 code 换 user_access_token
// ─────────────────────────────────────────
app.get('/api/auth/callback', async (req, res) => {
  const { code } = req.query
  if (!code) {
    return res.redirect(`${FRONTEND_URL}?auth=error&msg=missing_code`)
  }

  try {
    const appToken = await getAppAccessToken()
    const tokenRes = await axios.post(
      'https://open.feishu.cn/open-apis/authen/v1/oidc/access_token',
      {
        grant_type: 'authorization_code',
        code,
        redirect_uri: FEISHU_REDIRECT_URI,
      },
      { headers: { Authorization: `Bearer ${appToken}` } }
    )

    const data = tokenRes.data
    if (data.code !== 0) {
      console.error('换取 token 失败:', data)
      return res.redirect(`${FRONTEND_URL}?auth=error&msg=${encodeURIComponent(data.msg)}`)
    }

    userToken = data.data.access_token
    const grantedScope = data.data.scope || '（未返回 scope 字段）'
    console.log('飞书授权成功，已获取 user_access_token')
    console.log('实际授权的 scope：', grantedScope)
    res.redirect(`${FRONTEND_URL}?auth=success`)
  } catch (err) {
    console.error('OAuth 回调异常:', err.message)
    res.redirect(`${FRONTEND_URL}?auth=error&msg=${encodeURIComponent(err.message)}`)
  }
})

// ─────────────────────────────────────────
// GET /api/debug/token
// 调试：查看当前 token 对应的用户信息和权限
// ─────────────────────────────────────────
app.get('/api/debug/token', async (req, res) => {
  if (!userToken) return res.status(401).json({ error: '未授权' })
  try {
    const userInfo = await axios.get('https://open.feishu.cn/open-apis/authen/v1/user_info', {
      headers: { Authorization: `Bearer ${userToken}` }
    })
    const introspect = await axios.post(
      'https://open.feishu.cn/open-apis/authen/v1/token/introspect',
      { token: userToken, token_type_hint: 'access_token' },
      { headers: { Authorization: `Bearer ${await getAppAccessToken()}` } }
    ).catch(e => ({ data: { error: e.response?.data || e.message } }))
    res.json({ userInfo: userInfo.data, introspect: introspect.data })
  } catch (err) {
    res.status(500).json({ error: err.response?.data || err.message })
  }
})

// 前端查询当前授权状态
// ─────────────────────────────────────────
app.get('/api/auth/status', (req, res) => {
  res.json({ authorized: !!userToken })
})

// ─────────────────────────────────────────
// GET /api/drive/shared-with-me
// 列出「共享文件夹」（别人共享给我的）
// ─────────────────────────────────────────
app.get('/api/drive/shared-with-me', async (req, res) => {
  if (!userToken) return res.status(401).json({ error: '未授权，请先登录飞书' })
  const { folder_token, page_token } = req.query
  try {
    const params = { page_size: 50 }
    if (folder_token) params.folder_token = folder_token
    if (page_token) params.page_token = page_token
    // folder_type=share_with_me 列出共享给我的根目录
    if (!folder_token) params.folder_type = 'share_with_me'

    const response = await axios.get('https://open.feishu.cn/open-apis/drive/v1/files', {
      headers: { Authorization: `Bearer ${userToken}` },
      params,
    })
    const data = response.data
    console.log('共享文件夹响应:', JSON.stringify(data).slice(0, 500))
    if (data.code !== 0) return res.status(400).json({ error: data.msg, code: data.code })
    res.json(data.data)
  } catch (err) {
    const detail = err.response?.data || err.message
    console.error('获取共享文件夹失败:', JSON.stringify(detail))
    res.status(500).json({ error: err.message, detail })
  }
})

// ─────────────────────────────────────────
// GET /api/drive/shared-spaces
// 列出共享空间（团队文件夹）列表
// ─────────────────────────────────────────
app.get('/api/drive/shared-spaces', async (req, res) => {
  if (!userToken) return res.status(401).json({ error: '未授权，请先登录飞书' })
  try {
    const response = await axios.get('https://open.feishu.cn/open-apis/drive/v1/shared_spaces', {
      headers: { Authorization: `Bearer ${userToken}` },
      params: { page_size: 50 }
    })
    const data = response.data
    if (data.code !== 0) return res.status(400).json({ error: data.msg, code: data.code })
    res.json(data.data)
  } catch (err) {
    const detail = err.response?.data || err.message
    console.error('获取共享空间失败:', JSON.stringify(detail))
    res.status(500).json({ error: err.message, detail })
  }
})

// ─────────────────────────────────────────
// GET /api/drive/shared-files?space_id=xxx&folder_token=xxx
// 列出共享空间内的文件，space_id 必填
// ─────────────────────────────────────────
app.get('/api/drive/shared-files', async (req, res) => {
  if (!userToken) return res.status(401).json({ error: '未授权，请先登录飞书' })
  const { space_id, folder_token, page_token } = req.query
  if (!space_id) return res.status(400).json({ error: '缺少 space_id 参数' })
  try {
    const params = { page_size: 50 }
    if (folder_token) params.folder_token = folder_token
    if (page_token) params.page_token = page_token
    const response = await axios.get(
      `https://open.feishu.cn/open-apis/drive/v1/shared_spaces/${space_id}/files`,
      { headers: { Authorization: `Bearer ${userToken}` }, params }
    )
    const data = response.data
    if (data.code !== 0) return res.status(400).json({ error: data.msg, code: data.code })
    res.json(data.data)
  } catch (err) {
    const detail = err.response?.data || err.message
    console.error('获取共享文件失败:', JSON.stringify(detail))
    res.status(500).json({ error: err.message, detail })
  }
})

// ─────────────────────────────────────────
// GET /api/drive/files
// 列出云盘根目录文件，可通过 ?folder_token=xxx 指定文件夹
// ─────────────────────────────────────────
app.get('/api/drive/files', async (req, res) => {
  if (!userToken) {
    return res.status(401).json({ error: '未授权，请先登录飞书' })
  }

  const { folder_token, page_token } = req.query
  try {
    const params = { page_size: 50 }
    if (folder_token) params.folder_token = folder_token
    if (page_token) params.page_token = page_token

    const response = await axios.get('https://open.feishu.cn/open-apis/drive/v1/files', {
      headers: { Authorization: `Bearer ${userToken}` },
      params,
    })

    const data = response.data
    if (data.code !== 0) {
      return res.status(400).json({ error: data.msg })
    }

    res.json(data.data)
  } catch (err) {
    const detail = err.response?.data || err.message
    console.error('获取文件列表失败:', JSON.stringify(detail))
    res.status(500).json({ error: err.message, detail })
  }
})

// ─────────────────────────────────────────
// GET /api/drive/download?file_token=xxx&file_name=xxx.xlsx
// 下载云盘中的 Excel 文件并返回给前端（或保存到本地）
// ─────────────────────────────────────────
app.get('/api/drive/download', async (req, res) => {
  if (!userToken) {
    return res.status(401).json({ error: '未授权，请先登录飞书' })
  }

  const { file_token, file_name } = req.query
  if (!file_token) {
    return res.status(400).json({ error: '缺少 file_token 参数' })
  }

  try {
    const response = await axios.get(
      `https://open.feishu.cn/open-apis/drive/v1/files/${file_token}/download`,
      {
        headers: { Authorization: `Bearer ${userToken}` },
        responseType: 'stream',
      }
    )

    const name = file_name || 'download.xlsx'
    res.setHeader('Content-Disposition', `attachment; filename="${encodeURIComponent(name)}"`)
    res.setHeader('Content-Type', 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet')
    response.data.pipe(res)
  } catch (err) {
    console.error('下载文件失败:', err.message)
    res.status(500).json({ error: err.message })
  }
})

// ─────────────────────────────────────────
// GET /api/drive/sheet-data?sheet_token=xxx&sheet_id=xxx
// 读取飞书电子表格指定 sheet 的所有数据，返回行数组
// sheet_id 直接来自飞书 URL 中的 ?sheet=xxx 参数
// ─────────────────────────────────────────
app.get('/api/drive/sheet-data', async (req, res) => {
  if (!userToken) return res.status(401).json({ error: '未授权，请先登录飞书' })
  const { sheet_token, sheet_id } = req.query
  if (!sheet_token || !sheet_id) return res.status(400).json({ error: '缺少 sheet_token 或 sheet_id 参数' })

  try {
    // 分批读取，每批 500 行，避免超过 10MB 限制
    const BATCH = 500
    let allValues = null

    for (let startRow = 1; ; startRow += BATCH) {
      const endRow = startRow + BATCH - 1
      const range = `${sheet_id}!A${startRow}:ZZ${endRow}`
      const valRes = await axios.get(
        `https://open.feishu.cn/open-apis/sheets/v2/spreadsheets/${sheet_token}/values/${encodeURIComponent(range)}`,
        { headers: { Authorization: `Bearer ${userToken}` } }
      )
      if (valRes.data.code !== 0) {
        return res.status(400).json({ error: valRes.data.msg, code: valRes.data.code })
      }

      const batch = valRes.data.data?.valueRange?.values || []
      if (allValues === null) {
        allValues = batch  // 第一批包含表头
      } else {
        allValues = allValues.concat(batch)
      }

      // 如果返回行数小于 BATCH，说明已到末尾
      if (batch.length < BATCH) break
    }

    if (!allValues || allValues.length === 0) return res.json({ rows: [] })

    // 第一行作为表头，后续行转为对象数组
    const headers = allValues[0].map(h => (h == null ? '' : String(h)))
    const rows = []
    for (let i = 1; i < allValues.length; i++) {
      const row = allValues[i]
      if (!row || row.every(c => c == null || c === '')) continue
      const obj = {}
      headers.forEach((h, j) => { obj[h] = row[j] ?? null })
      rows.push(obj)
    }

    console.log(`读取 sheet[${sheet_id}] 成功，共 ${rows.length} 行`)
    res.json({ rows })
  } catch (err) {
    const detail = err.response?.data || err.message
    console.error('读取 sheet 数据失败:', JSON.stringify(detail))
    res.status(500).json({ error: err.message, detail })
  }
})

// ─────────────────────────────────────────
// GET /api/drive/doc-content?doc_token=xxx
// 读取飞书文档内容，返回纯文本
// ─────────────────────────────────────────
app.get('/api/drive/doc-content', async (req, res) => {
  if (!userToken) return res.status(401).json({ error: '未授权，请先登录飞书' })
  const { doc_token } = req.query
  if (!doc_token) return res.status(400).json({ error: '缺少 doc_token 参数' })

  try {
    // 使用正确的blocks API获取文档内容
    const response = await axios.get(
      `https://open.feishu.cn/open-apis/docx/v1/documents/${doc_token}/blocks`,
      { headers: { Authorization: `Bearer ${userToken}` } }
    )
    const data = response.data
    if (data.code !== 0) {
      return res.status(400).json({ error: data.msg })
    }

    // 解析文档块，提取纯文本
    // 飞书文档blocks API返回结构: data.data.items[i].block.content
    const items = data.data?.items || []
    const textLines = []

    // 递归提取文本的辅助函数
    function extractText(obj) {
      if (!obj) return ''
      if (typeof obj === 'string') return obj
      if (typeof obj === 'number') return String(obj)

      let result = ''
      if (Array.isArray(obj)) {
        for (const item of obj) {
          result += extractText(item)
        }
      } else if (typeof obj === 'object') {
        // 飞书文档常见结构: text_run.content, text.content
        if (obj.text_run) {
          result += extractText(obj.text_run)
        }
        if (obj.elements) {
          result += extractText(obj.elements)
        }
        if (obj.text) {
          result += typeof obj.text === 'string' ? obj.text : extractText(obj.text)
        }
        if (obj.content) {
          result += extractText(obj.content)
        }
      }
      return result
    }

    for (const item of items) {
      const block = item.block
      if (!block || !block.content) continue

      const lineText = extractText(block.content).trim()
      if (lineText) {
        textLines.push(lineText)
      }
    }

    const fullText = textLines.join('\n')
    console.log(`读取文档[${doc_token}], blocks数量: ${items.length}, 文本长度: ${fullText.length}`)
    res.json({ text: fullText })
  } catch (err) {
    const detail = err.response?.data || err.message
    console.error('读取文档内容失败:', JSON.stringify(detail))
    res.status(500).json({ error: err.message, detail })
  }
})

// ─────────────────────────────────────────
// GET /api/drive/export-sheet?sheet_token=xxx
// 导出飞书电子表格为 xlsx 并返回给前端
// 飞书 sheets 链接中的 token 即 spreadsheetToken
// ─────────────────────────────────────────
app.get('/api/drive/export-sheet', async (req, res) => {
  if (!userToken) return res.status(401).json({ error: '未授权，请先登录飞书' })
  const { sheet_token } = req.query
  if (!sheet_token) return res.status(400).json({ error: '缺少 sheet_token 参数' })

  try {
    // 第一步：创建导出任务
    const createRes = await axios.post(
      'https://open.feishu.cn/open-apis/drive/v1/export_tasks',
      { file_extension: 'xlsx', token: sheet_token, type: 'sheet' },
      { headers: { Authorization: `Bearer ${userToken}` } }
    )
    if (createRes.data.code !== 0) {
      console.error('创建导出任务失败:', JSON.stringify(createRes.data))
      return res.status(400).json({ error: createRes.data.msg, code: createRes.data.code, raw: createRes.data })
    }
    const ticket = createRes.data.data.ticket
    console.log('导出任务已创建，ticket:', ticket)

    // 第二步：轮询任务状态（最多等 15 秒）
    let fileToken = null
    for (let i = 0; i < 15; i++) {
      await new Promise(r => setTimeout(r, 1000))
      const pollRes = await axios.get(
        `https://open.feishu.cn/open-apis/drive/v1/export_tasks/${ticket}`,
        { headers: { Authorization: `Bearer ${userToken}` }, params: { token: sheet_token } }
      )
      const job = pollRes.data.data?.result
      console.log(`导出状态[${i + 1}]:`, job?.job_status, job?.job_error_msg)
      if (job?.job_status === 0) { // 0 = 成功
        fileToken = job.file_token
        break
      }
      if (job?.job_status === 2) { // 2 = 失败
        return res.status(500).json({ error: '导出失败：' + job.job_error_msg })
      }
    }
    if (!fileToken) return res.status(504).json({ error: '导出超时，请稍后重试' })

    // 第三步：下载导出的文件
    const dlRes = await axios.get(
      `https://open.feishu.cn/open-apis/drive/v1/export_tasks/file/${fileToken}/download`,
      { headers: { Authorization: `Bearer ${userToken}` }, responseType: 'stream' }
    )
    res.setHeader('Content-Type', 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet')
    res.setHeader('Content-Disposition', 'attachment; filename="export.xlsx"')
    dlRes.data.pipe(res)
  } catch (err) {
    const detail = err.response?.data || err.message
    console.error('导出电子表格失败:', JSON.stringify(detail))
    res.status(500).json({ error: err.message, detail })
  }
})

// ─────────────────────────────────────────
// DELETE /api/auth/logout
// 清除本地 token
// ─────────────────────────────────────────
app.delete('/api/auth/logout', (req, res) => {
  userToken = null
  res.json({ ok: true })
})

// GET /api/auth/logout (方便浏览器访问)
app.get('/api/auth/logout', (req, res) => {
  userToken = null
  res.json({ ok: true })
})

// ─────────────────────────────────────────
// POST /api/db/sync
// 从飞书直接读取 sheet 数据写入 SQLite
// ─────────────────────────────────────────
const SHEET_TOKEN = 'HdxnsNXSQhKoSItKiLwcnEmjn8b'

// 已知的 sheet 信息（来自 /api/debug/sheet-meta）
const SHEETS_META = {
  产品表: { id: '3JiyjX', rows: 134, cols: 58 },
  交易表: { id: '0PZFXO', rows: 2871, cols: 34 },
}

async function readSheet(sheetId, rowCount, colCount) {
  function colLetter(n) {
    let s = ''
    while (n > 0) { n--; s = String.fromCharCode(65 + n % 26) + s; n = Math.floor(n / 26) }
    return s || 'A'
  }

  function cellToString(v) {
    if (v == null) return ''
    if (typeof v === 'string') return v.trim()
    if (typeof v === 'number' || typeof v === 'boolean') return String(v).trim()
    if (Array.isArray(v)) return v.map(cellToString).join('').trim()
    if (typeof v === 'object') {
      if (v.text) return String(v.text).trim()
      if (v.elements) return v.elements.map(cellToString).join('').trim()
    }
    return String(v).trim()
  }

  const BATCH = 500
  let allValues = null

  for (let startRow = 1; startRow <= rowCount + 1; startRow += BATCH) {
    const endRow = Math.min(startRow + BATCH - 1, rowCount + 1)
    const range = `${sheetId}!A${startRow}:${colLetter(colCount)}${endRow}`
    const valRes = await axios.get(
      `https://open.feishu.cn/open-apis/sheets/v2/spreadsheets/${SHEET_TOKEN}/values/${encodeURIComponent(range)}?valueRenderOption=UnformattedValue`,
      { headers: { Authorization: `Bearer ${userToken}` } }
    )
    if (valRes.data.code !== 0) {
      throw new Error(`读取失败(${valRes.data.code}): ${valRes.data.msg}`)
    }
    const batch = valRes.data.data?.valueRange?.values || []
    allValues = allValues ? allValues.concat(batch) : batch
    if (batch.length < BATCH) break
  }

  if (!allValues || allValues.length === 0) return []
  const headers = allValues[0].map(cellToString)
  const rows = []
  for (let i = 1; i < allValues.length; i++) {
    const row = allValues[i]
    if (!row || row.every(c => c == null || c === '')) continue
    const obj = {}
    headers.forEach((h, j) => { obj[h] = row[j] ?? null })
    rows.push(obj)
  }
  return rows
}

// Excel 序列数转日期字符串 (YYYY-MM-DD)
function excelDateToString(val) {
  if (!val) return null
  const num = Number(val)
  if (isNaN(num) || num < 1) return String(val)  // 不是数字，原样返回
  // Excel 日期序列：1900-01-01 = 1，但 Excel 有1900年2月29日bug，需+1修正
  const date = new Date((num - 25569) * 86400 * 1000)
  if (isNaN(date.getTime())) return String(val)
  return date.toISOString().slice(0, 10)
}

app.post('/api/db/sync', async (req, res) => {
  if (!userToken) return res.status(401).json({ error: '未授权，请先登录飞书' })

  try {
    // 读产品表
    const { id: prodId, rows: prodRowCount, cols: prodColCount } = SHEETS_META['产品表']
    console.log(`开始读取产品表，共 ${prodRowCount} 行 ${prodColCount} 列...`)
    const prodRows = await readSheet(prodId, prodRowCount, prodColCount)
    console.log(`产品表读取完成：${prodRows.length} 行`)

    function findField(r, patterns) {
      for (const pattern of patterns) {
        const normalized = pattern.replace(/\s+/g, '')
        for (const key of Object.keys(r)) {
          if (key.replace(/\s+/g, '') === normalized || key.replace(/\s+/g, '').includes(normalized)) {
            return r[key]
          }
        }
      }
      return undefined
    }

    const productRows = []
    for (const r of prodRows) {
      const flightId = r['航班编号']
      if (!flightId) continue
      const isMain = String(r['是否主产品'] || r[' 是否主产品'] || '').trim() === '是' ? 1 : 0
      const lockDays = Number(r['锁定期']) || 0
      const rawKO = r['敲出']
      const entryPrice = Number(r['入场价']) || 0
      const firstKO = rawKO != null && !String(rawKO).includes('*') ? parseFirstKnockoutRatio(rawKO, entryPrice) : 0
      productRows.push({
        id: String(flightId).trim(),
        name: r['产品名称'] || null,
        is_main: isMain,
        issue_date: excelDateToString(r['认购日']),
        complete_date: excelDateToString(r['完结时间']),
        subscribe_amount: Number(r['认购金额']) || 0,
        outstanding_amount: Number(r['存续金额']) || 0,
        manager: r['私募管理人'] || null,
        holding_status: r['持有状态'] || null,
        structure_type: r['结构类型'] || null,
        code: r['代码'] || null,
        lock_days: lockDays,
        lock_months: Math.floor(lockDays / 30),
        first_knockout_ratio: firstKO,
        entry_price: entryPrice,
        monthly_decrease: parseRatio(findField(r, ['每月递减'])),
        term: findField(r, ['期限']) || null,
        parachute: r['降落伞'] || null,
        dividend_barrier: parseRatio(findField(r, ['派息障碍'])),
        monthly_coupon: parseRatio(findField(r, ['月票息'])),
        coupon_1st: parseRatio(findField(r, ['第一段票息'])),
        coupon_2nd: parseRatio(findField(r, ['第二段票息'])),
        coupon_3rd: parseRatio(findField(r, ['第三段票息'])),
        duration_months: Number(findField(r, ['存续时间'])) || 0,
        absolute_return: Number(findField(r, ['绝对收益率'])) || 0,
        holiday_adjust: findField(r, ['观察日节假日']) || null,
        raw: JSON.stringify(r),
      })
    }

    importProducts(productRows)
    console.log(`产品表同步完成，共写入 ${productRows.length} 条`)

    // ── 同步后自动更新价格和观察记录 ──
    try {
      const ongoingProducts = queryOngoingProducts()
      const ongoingCodes = [...new Set(ongoingProducts.map(p => p.code).filter(Boolean))]
      if (ongoingCodes.length > 0) {
        const { results: autoPrices, failed: autoFailed } = await fetchAllPrices(ongoingCodes)
        const today = new Date().toISOString().slice(0, 10)
        for (const code of ongoingCodes) {
          if (autoPrices[code] !== undefined) upsertPrice(code, today, autoPrices[code])
        }
        let autoGen = 0
        for (const product of ongoingProducts) {
          if (!product.code || !product.issue_date || !product.entry_price || autoPrices[product.code] === undefined) continue
          const obsDates = getObservationDates(product)
          const existingObs = queryObservationsByProduct(product.id)
          const existingMap = new Map(existingObs.map(o => [o.observation_date, o]))
          for (const { date, monthsSinceEntry } of obsDates) {
            if (existingMap.has(date)) continue
            const cachedPrice = queryPriceByDate(product.code, date)
            const priceForDate = cachedPrice ? cachedPrice.price : autoPrices[product.code]
            const evalResult = evaluateObservation(product, date, priceForDate, monthsSinceEntry)
            evalResult.product_id = product.id
            upsertObserv(evalResult)
            autoGen++
          }
        }
        console.log(`同步后自动更新：价格 ${ongoingCodes.length - (autoFailed?.length || 0)}/${ongoingCodes.length}，新增观察记录 ${autoGen} 条`)
      }
    } catch (autoErr) {
      console.warn('[同步后自动更新观察记录失败，不影响主同步]', autoErr.message)
    }

    // 读交易表
    const { id: txId, rows: txRowCount, cols: txColCount } = SHEETS_META['交易表']
    console.log(`开始读取交易表，共 ${txRowCount} 行 ${txColCount} 列...`)
    const txRows = await readSheet(txId, txRowCount, txColCount)
    console.log(`交易表读取完成：${txRows.length} 行`)

    const transactionRows = []
    for (const r of txRows) {
      const flightId = r['航班编号']
      if (!flightId) continue
      transactionRows.push({
        transaction_date: excelDateToString(r['交易日期']),
        flight_id: String(flightId).trim(),
        counterparty: r['交易对手'] || r['对手方'] || null,
        subscribe_amount: Number(r['认购金额']) || 0,
        raw: JSON.stringify(r),
      })
    }

    importTransactions(transactionRows)
    console.log(`交易表同步完成，共写入 ${transactionRows.length} 条`)

    const totalCount = productRows.length + transactionRows.length
    logSync(totalCount)
    res.json({ ok: true, rowCount: totalCount, productCount: productRows.length, transactionCount: transactionRows.length })
  } catch (err) {
    const detail = err.response?.data || err.message
    console.error('同步失败:', JSON.stringify(detail))
    res.status(500).json({ error: err.message, detail })
  }
})

// ─────────────────────────────────────────
// GET /api/db/sync-status
// 查询最近一次同步时间和行数
// ─────────────────────────────────────────
app.get('/api/db/sync-status', (req, res) => {
  const last = getLastSync()
  res.json(last || { synced_at: null, row_count: 0 })
})

// ─────────────────────────────────────────
// GET /api/db/products?start=YYYY-MM-DD&end=YYYY-MM-DD
// 按日期范围查询产品数据
// ─────────────────────────────────────────
app.get('/api/db/products', (req, res) => {
  const { start, end } = req.query
  if (!start || !end) return res.status(400).json({ error: '缺少 start 或 end 参数' })
  try {
    const rows = queryProducts({ startDate: start, endDate: end + ' 23:59:59' })
    res.json({ rows })
  } catch (err) {
    res.status(500).json({ error: err.message })
  }
})

// 合投用户表（多维表格）配置
const COINVEST_APP_TOKEN = 'G1sbbVhL2awTltsOoi8cqci4nJh'
const COINVEST_TABLE_ID = 'tbl5mm7sQ001Z7p1'

// ─────────────────────────────────────────
// POST /api/db/sync-coinvest
// 从飞书多维表格同步合投用户数据
// ─────────────────────────────────────────
app.post('/api/db/sync-coinvest', async (req, res) => {
  if (!userToken) return res.status(401).json({ error: '未授权，请先登录飞书' })

  try {
    let allRecords = []
    let pageToken = null

    // 分页读取所有记录
    do {
      const params = { page_size: 500 }
      if (pageToken) params.page_token = pageToken

      const url = `https://open.feishu.cn/open-apis/bitable/v1/apps/${COINVEST_APP_TOKEN}/tables/${COINVEST_TABLE_ID}/records`
      const valRes = await axios.get(url, {
        headers: { Authorization: `Bearer ${userToken}` },
        params,
      })

      if (valRes.data.code !== 0) {
        throw new Error(`读取合投用户表失败(${valRes.data.code}): ${valRes.data.msg}`)
      }

      const data = valRes.data.data
      if (data.items && data.items.length > 0) {
        allRecords = allRecords.concat(data.items)
      }
      console.log(`合投用户表分页读取：本次返回 ${data.items?.length || 0} 条，fields 示例：`, JSON.stringify(data.items?.[0]?.fields))

      pageToken = data.page_token
      // 如果没有下一页，page_token 为空
    } while (pageToken)

    console.log(`合投用户表读取完成：共 ${allRecords.length} 条记录`)

    const rows = []
    for (const record of allRecords) {
      const f = record.fields || {}
      // 复杂字段（数组/对象）转文本
      const toText = (v) => {
        if (v == null) return null
        if (typeof v === 'string') return v
        if (Array.isArray(v)) return v.map(i => typeof i === 'object' ? (i.text || i.name || JSON.stringify(i)) : i).join('、')
        if (typeof v === 'object') return v.text || v.name || JSON.stringify(v)
        return String(v)
      }
      rows.push({
        user_name: f['名义购买人'] || null,
        actual_buyer: toText(f['实际购买人']),
        phone: toText(f['手机号']),
        wechat: toText(f['微信昵称']),
        total_assets: Number(f['资产总和/万']) || 0,
        risk_tolerance: toText(f['风险承受']),
        industry: toText(f['行业']) || toText(f['客户行业']),
        is_actual_deal: toText(f['是否成交客户']),
        lead_source: toText(f['进线来源分类']),
        asset_match: toText(f['资产匹配度']),
        is_dedicated_account: toText(f['是否专户客户']),
        is_competitor: toText(f['客户是否竞品群']),
        raw: JSON.stringify(f),
      })
    }

    importCoInvestUsers(rows)
    logCoInvestSync(rows.length)
    console.log(`合投用户同步完成，共写入 ${rows.length} 条`)
    res.json({ ok: true, rowCount: rows.length })
  } catch (err) {
    const detail = err.response?.data || err.message
    console.error('合投用户同步失败:', JSON.stringify(detail))
    res.status(500).json({ error: err.message, detail })
  }
})

// ─────────────────────────────────────────
// GET /api/db/sync-coinvest-status
// 查询合投用户表同步状态
// ─────────────────────────────────────────
app.get('/api/db/sync-coinvest-status', (req, res) => {
  const last = getLastCoInvestSync()
  res.json(last || { synced_at: null, row_count: 0 })
})

// ─────────────────────────────────────────
// GET /api/db/user-profiles
// 用户画像查询：支持多条件搜索合投用户
// ─────────────────────────────────────────
app.get('/api/db/user-profiles', (req, res) => {
  const { actual_buyer, nominal_buyer, is_dedicated, is_competitor, industry } = req.query

  try {
    const users = queryCoInvestUsers({
      actualBuyer: actual_buyer || '',
      nominalBuyer: nominal_buyer || '',
      isDedicated: is_dedicated || '',
      isCompetitor: is_competitor || '',
      industry: industry || ''
    })

    // 计算所有客户的历史存量峰值和峰值差额
    const peakData = computeUserPeakBalances()

    // 从 raw JSON 中解析客户是否竞品群
    const result = users.map(u => {
      let raw = {}
      try { raw = JSON.parse(u.raw || '{}') } catch {}

      // 从 raw JSON 解析额外字段
      const boughtBefore = raw['衍选成交前购买过结构化产品'] != null ? String(raw['衍选成交前购买过结构化产品']) : ''
      const assetRange = raw['境内资产规模区间/万RMB'] != null ? String(raw['境内资产规模区间/万RMB']) : ''

      // 获取该客户的峰值数据
      const buyer = u.actual_buyer || ''
      const userPeak = peakData[buyer] || {}

      return {
        id: u.id,
        actual_buyer: u.actual_buyer,
        nominal_buyer: u.user_name,
        is_competitor: u.is_competitor || '',
        is_dedicated_account: u.is_dedicated_account || '',
        bought_before_yanxuan: boughtBefore,
        asset_range: assetRange,
        total_assets: u.total_assets,
        risk_tolerance: u.risk_tolerance,
        industry: u.industry,
        wechat: u.wechat,
        phone: u.phone,
        lead_source: u.lead_source,
        // 历史存量峰值和峰值差额
        peak_balance: userPeak.peak_balance ?? null,
        peak_diff: userPeak.peak_diff ?? null,
      }
    })

    res.json({ rows: result, total: result.length })
  } catch (err) {
    console.error('查询用户画像失败:', err.message)
    res.status(500).json({ error: err.message })
  }
})

// ─────────────────────────────────────────
// GET /api/db/industries
// 获取所有客户行业（下拉用）
// ─────────────────────────────────────────
app.get('/api/db/industries', (req, res) => {
  try {
    const industries = getDistinctIndustries()
    res.json({ rows: industries })
  } catch (err) {
    res.status(500).json({ error: err.message })
  }
})

// 临时调试：查看合投用户表字段名
app.get('/api/debug/coinvest-fields', async (req, res) => {
  if (!userToken) return res.status(401).json({ error: '未授权' })
  try {
    const url = `https://open.feishu.cn/open-apis/bitable/v1/apps/${COINVEST_APP_TOKEN}/tables/${COINVEST_TABLE_ID}/records?page_size=1`
    const valRes = await axios.get(url, { headers: { Authorization: `Bearer ${userToken}` } })
    if (valRes.data.code !== 0) return res.status(400).json(valRes.data)
    const fields = valRes.data.data?.items?.[0]?.fields || {}
    res.json({ fieldNames: Object.keys(fields), sample: fields })
  } catch (err) {
    res.status(500).json({ error: err.message, detail: err.response?.data })
  }
})

// 临时调试：查看 sheet meta
app.get('/api/debug/sheet-meta', async (req, res) => {
  if (!userToken) return res.status(401).json({ error: '未授权' })
  const token = req.query.token || SHEET_TOKEN
  try {
    const metaRes = await axios.get(
      `https://open.feishu.cn/open-apis/sheets/v3/spreadsheets/${token}/sheets/query`,
      { headers: { Authorization: `Bearer ${userToken}` } }
    )
    res.json(metaRes.data)
  } catch (err) {
    res.status(500).json({ error: err.message, detail: err.response?.data })
  }
})

// ─────────────────────────────────────────
// POST /api/drive/sync-product-docs
// 同步飞书产品库文件夹中的所有文档到本地SQLite
// 产品库 token: W9OGfnjzQl8dOOdqPFwcL6gEnkf
// ─────────────────────────────────────────
const PRODUCT_LIBRARY_TOKEN = 'W9OGfnjzQl8dOOdqPFwcL6gEnkf'

// 递归获取文件夹下所有文件和子文件夹
async function getAllItemsRecursively(folderToken, parentPath = '', items = [], folderCount = { count: 0 }) {
  try {
    const response = await axios.get('https://open.feishu.cn/open-apis/drive/v1/files', {
      headers: { Authorization: `Bearer ${userToken}` },
      params: { folder_token: folderToken, page_size: 200 }
    })

    if (response.data.code !== 0) {
      console.error('获取文件夹内容失败:', response.data.msg)
      return { items, folderCount }
    }

    const files = response.data.data?.files || []

    for (const file of files) {
      const currentPath = parentPath ? `${parentPath} / ${file.name}` : file.name

      if (file.type === 'folder') {
        folderCount.count++
        // 递归获取子文件夹
        await getAllItemsRecursively(file.token, currentPath, items, folderCount)
      } else {
        // 是文件，记录下来
        items.push({
          ...file,
          parent_path: currentPath
        })
      }
    }
  } catch (err) {
    console.error('递归获取文件夹失败:', err.message)
  }

  return { items, folderCount }
}

// 读取文档内容（使用 raw content API 获取纯文本）
async function readDocContent(docToken) {
  try {
    // 使用 raw content API，直接获取文档的纯文本内容
    const response = await axios.get(
      `https://open.feishu.cn/open-apis/docx/v1/documents/${docToken}/raw_content`,
      { headers: { Authorization: `Bearer ${userToken}` } }
    )

    if (response.data.code !== 0) {
      console.log('读取文档失败, code:', response.data.code, 'msg:', response.data.msg)
      return null
    }

    const content = response.data.data?.content || ''
    console.log(`文档 ${docToken} raw content 长度: ${content.length}`)
    return content || null
  } catch (err) {
    const detail = err.response?.data || err.message
    console.error('读取文档失败:', JSON.stringify(detail))
    return null
  }
}

// 解析产品结构
function parseProductStructure(text) {
  if (!text) return null

  const fieldPatterns = {
    '结构': /结构[：:]\s*(.+)/,
    '标的': /标的[：:]\s*(.+)/,
    '期限': /期限[：:]\s*(.+)/,
    '保证金比例': /保证金比例[：:]\s*(.+)/,
    '期初敲出线': /(?:期初)?敲出线[：:]\s*(.+)/,
    '降敲': /降敲[：:]\s*(.+)/,
    '降落伞': /降落伞[：:]\s*(.+)/,
    '派息线': /派息线[：:]\s*(.+)/,
    '票息（税费后）': /票息[（(]税费后[）)][：:]\s*(.+)/,
    '打款时间': /打款时间[：:]\s*(.+)/,
    '入场时间': /入场时间[：:]\s*(.+)/,
  }

  const result = {}
  for (const [field, pattern] of Object.entries(fieldPatterns)) {
    const match = text.match(pattern)
    if (match) result[field] = match[1].trim()
  }

  return Object.keys(result).length > 0 ? result : null
}

app.post('/api/drive/sync-product-docs', async (req, res) => {
  if (!userToken) return res.status(401).json({ error: '未授权，请先登录飞书' })

  try {
    console.log('开始同步产品库文档...')

    // 递归获取所有文档
    const { items: allFiles, folderCount } = await getAllItemsRecursively(PRODUCT_LIBRARY_TOKEN)

    console.log(`找到 ${allFiles.length} 个文件，${folderCount.count} 个子文件夹`)

    // 列出所有找到的文件用于调试
    console.log('所有文件列表:', allFiles.map(f => ({ name: f.name, type: f.type, parent_path: f.parent_path })))

    // 不过滤，尝试读取所有文件
    const docs = allFiles.filter(f => f.type !== 'folder')

    console.log(`文件类型分布:`, docs.map(f => f.type).reduce((acc, t) => { acc[t] = (acc[t] || 0) + 1; return acc; }, {}))

    console.log(`开始读取 ${docs.length} 个文档内容...`)

    const syncedDocs = []
    for (let i = 0; i < docs.length; i++) {
      const doc = docs[i]
      console.log(`[${i + 1}/${docs.length}] 读取文档: ${doc.name}, 类型: ${doc.type}`)

      const rawContent = await readDocContent(doc.token)
      console.log(`文档 ${doc.name} 内容长度: ${(rawContent || '').length}`)
      const structured = parseProductStructure(rawContent)
      console.log(`文档 ${doc.name} 解析结果:`, structured)

      // 避免过于频繁的 API 调用，读取完一个文档后暂停一下
      if (i < docs.length - 1) await new Promise(r => setTimeout(r, 500))

      syncedDocs.push({
        doc_token: doc.token,
        doc_name: doc.name,
        parent_path: doc.parent_path,
        folder_token: doc.parent_token,
        raw_content: rawContent || '',
        structure_json: structured ? JSON.stringify(structured) : '',
        synced_at: new Date().toISOString()
      })
    }

    // 写入数据库
    importProductDocs(syncedDocs)
    logProductDocsSync(syncedDocs.length, folderCount.count)

    console.log(`同步完成！共同步 ${syncedDocs.length} 个文档到数据库`)

    res.json({
      ok: true,
      message: `同步成功，共 ${syncedDocs.length} 个文档`,
      doc_count: syncedDocs.length,
      folder_count: folderCount.count,
      last_sync: new Date().toISOString()
    })
  } catch (err) {
    console.error('同步产品库文档失败:', err)
    res.status(500).json({ error: err.message })
  }
})

// GET /api/drive/product-docs - 获取已同步的产品文档
app.get('/api/drive/product-docs', (req, res) => {
  try {
    const { month } = req.query
    let docs

    if (month) {
      docs = getProductDocsByMonth(month)
    } else {
      docs = getAllProductDocs()
    }

    // 解析 structure_json
    const result = docs.map(doc => ({
      ...doc,
      structured: doc.structure_json ? JSON.parse(doc.structure_json) : null
    }))

    res.json(result)
  } catch (err) {
    res.status(500).json({ error: err.message })
  }
})

// GET /api/drive/product-docs/sync-status - 获取同步状态
app.get('/api/drive/product-docs/sync-status', (req, res) => {
  try {
    const lastSync = getLastProductDocsSync()
    res.json(lastSync || { message: '从未同步' })
  } catch (err) {
    res.status(500).json({ error: err.message })
  }
})

// ─────────────────────────────────────────
// GET /api/observations
// 查询所有存续产品的观察记录
// ─────────────────────────────────────────
app.get('/api/observations', (req, res) => {
  try {
    const { search, code } = req.query
    let products = queryOngoingProducts()
    if (search) {
      const q = search.toLowerCase()
      products = products.filter(p =>
        (p.name && p.name.toLowerCase().includes(q)) ||
        p.id.toLowerCase().includes(q)
      )
    }
    if (code) {
      const codeLower = code.toLowerCase()
      products = products.filter(p => p.code && p.code.toLowerCase().includes(codeLower))
    }

    const result = products.map(p => {
      const observations = queryObservationsByProduct(p.id)
      return {
        id: p.id,
        name: p.name,
        manager: p.manager,
        holding_status: p.holding_status,
        code: p.code,
        entry_price: p.entry_price,
        first_knockout_ratio: p.first_knockout_ratio,
        lock_months: p.lock_months,
        monthly_decrease: p.monthly_decrease,
        issue_date: p.issue_date,
        subscribe_amount: p.subscribe_amount,
        dividend_barrier: p.dividend_barrier,
        holiday_adjust: p.holiday_adjust,
        lock_days: p.lock_days,
        duration_months: p.duration_months,
        next_observation_date: getNextObservationDate(p),
        observations: observations.map(o => ({
          date: o.observation_date,
          knockout_price: o.knockout_price,
          dividend_line: o.dividend_line,
          underlying_price: o.underlying_price,
          is_knocked_out: o.is_knocked_out,
          is_dividend: o.is_dividend,
          months_since_entry: o.months_since_entry,
        })),
      }
    })

    const lastRecord = getLastObservationUpdate()
    res.json({
      products: result,
      lastUpdated: lastRecord ? lastRecord.updated_at : null,
    })
  } catch (err) {
    res.status(500).json({ error: err.message })
  }
})

app.get('/api/observations/calendar', (req, res) => {
  try {
    const month = req.query.month || new Date().toISOString().slice(0, 7)
    if (!/^\d{4}-\d{2}$/.test(month)) {
      return res.status(400).json({ error: '月份格式应为 YYYY-MM' })
    }

    const products = queryOngoingProducts()
    const dates = new Map()

    for (const product of products) {
      const monthDates = getObservationDatesForMonth(product, month)
      for (const obs of monthDates) {
        if (!dates.has(obs.date)) dates.set(obs.date, [])
        dates.get(obs.date).push({
          id: product.id,
          name: product.name,
          manager: product.manager,
          code: product.code,
          months_since_entry: obs.monthsSinceEntry,
        })
      }
    }

    const calendar = [...dates.entries()]
      .sort(([a], [b]) => a.localeCompare(b))
      .map(([date, productsForDate]) => ({
        date,
        products: productsForDate.sort((a, b) => String(a.name || '').localeCompare(String(b.name || ''), 'zh-CN')),
      }))

    res.json({ month, calendar })
  } catch (err) {
    res.status(500).json({ error: err.message })
  }
})

app.get('/api/observations/today', (req, res) => {
  try {
    const today = new Date().toISOString().slice(0, 10)
    const products = queryOngoingProducts()

    const result = products.map(p => {
      const obsDates = getObservationDates(p)
      const hasObservationToday = obsDates.some(d => d.date === today)

      if (!hasObservationToday) return null

      const observations = queryObservationsByProduct(p.id)
      return {
        id: p.id,
        name: p.name,
        manager: p.manager,
        holding_status: p.holding_status,
        code: p.code,
        entry_price: p.entry_price,
        first_knockout_ratio: p.first_knockout_ratio,
        lock_months: p.lock_months,
        monthly_decrease: p.monthly_decrease,
        issue_date: p.issue_date,
        subscribe_amount: p.subscribe_amount,
        dividend_barrier: p.dividend_barrier,
        holiday_adjust: p.holiday_adjust,
        lock_days: p.lock_days,
        duration_months: p.duration_months,
        next_observation_date: getNextObservationDate(p, today),
        observations: observations.map(o => ({
          date: o.observation_date,
          knockout_price: o.knockout_price,
          dividend_line: o.dividend_line,
          underlying_price: o.underlying_price,
          is_knocked_out: o.is_knocked_out,
          is_dividend: o.is_dividend,
          months_since_entry: o.months_since_entry,
        })),
      }
    }).filter(Boolean)

    const lastRecord = getLastObservationUpdate()
    res.json({
      products: result,
      today: today,
      lastUpdated: lastRecord ? lastRecord.updated_at : null,
    })
  } catch (err) {
    res.status(500).json({ error: err.message })
  }
})

// ─────────────────────────────────────────
// POST /api/observations/generate
// 生成观察记录（获取最新价格 + 计算结果）
// 已有观察记录且包含价格时跳过，新观察日优先使用缓存的历史价格
// ─────────────────────────────────────────
app.post('/api/observations/generate', async (req, res) => {
  try {
    const products = queryOngoingProducts()
    console.log(`[generate] 存续产品数量: ${products.length}`)
    const codes = [...new Set(products.map(p => p.code).filter(Boolean))]
    console.log(`[generate] 标的代码(去重): ${codes.join(', ')}`)

    console.log(`[generate] 开始获取 ${codes.length} 个标的价格...`)
    const { results: prices, failed } = await fetchAllPrices(codes)
    console.log(`[generate] 价格获取完成: 成功 ${Object.keys(prices).length}, 失败 ${failed.length}`)
    if (failed.length > 0) console.log(`[generate] 失败的代码:`, failed)

    const today = new Date().toISOString().slice(0, 10)
    for (const code of codes) {
      if (prices[code] !== undefined) {
        upsertPrice(code, today, prices[code])
      }
    }

    let generated = 0
    let skippedExisting = 0
    let skippedNoCode = 0
    let skippedNoPrice = 0
    let skippedNoDates = 0

    for (const product of products) {
      if (!product.code || !product.issue_date || !product.entry_price) {
        skippedNoCode++
        continue
      }

      const obsDates = getObservationDates(product)
      if (obsDates.length === 0) { skippedNoDates++; continue }

      const latestPrice = prices[product.code] || null
      const existingObs = queryObservationsByProduct(product.id)
      const existingMap = new Map(existingObs.map(o => [o.observation_date, o]))

      for (const { date, monthsSinceEntry } of obsDates) {
        const cachedPrice = queryPriceByDate(product.code, date)
        const existingRecord = existingMap.get(date)
        const priceForDate = cachedPrice ? cachedPrice.price : (existingRecord?.underlying_price ?? latestPrice)
        if (priceForDate === null || priceForDate === undefined) {
          skippedNoPrice++
          continue
        }

        const evalResult = evaluateObservation(product, date, priceForDate, monthsSinceEntry)
        evalResult.product_id = product.id
        upsertObserv(evalResult)
        if (existingRecord) {
          skippedExisting++
          continue
        }
        generated++
      }
    }

    console.log(`[generate] 结果: 新增=${generated}, 重算(已有)=${skippedExisting}, 跳过(无code)=${skippedNoCode}, 跳过(无价格)=${skippedNoPrice}, 跳过(无观察日)=${skippedNoDates}`)
    res.json({ ok: true, generated, recalculatedExisting: skippedExisting, priceRefreshed: codes.length, priceFailed: failed.length })
  } catch (err) {
    console.error('生成观察记录失败:', err)
    res.status(500).json({ error: err.message })
  }
})

// ─────────────────────────────────────────
// POST /api/observations/refresh-prices
// 仅刷新标的价格：更新今日观察日使用最新价格，历史观察日使用对应日期的缓存价格
// ─────────────────────────────────────────
app.post('/api/observations/refresh-prices', async (req, res) => {
  try {
    const products = queryOngoingProducts()
    const codes = [...new Set(products.map(p => p.code).filter(Boolean))]

    const { results: prices, failed } = await fetchAllPrices(codes)
    const today = new Date().toISOString().slice(0, 10)

    let refreshed = 0
    for (const code of codes) {
      if (prices[code] !== undefined) {
        upsertPrice(code, today, prices[code])
        refreshed++
      }
    }

    let updated = 0
    for (const product of products) {
      if (!product.code || prices[product.code] === undefined) continue
      const latestPrice = prices[product.code]
      const productObs = queryObservationsByProduct(product.id)
      for (const obs of productObs) {
        const cachedPrice = queryPriceByDate(product.code, obs.observation_date)
        const priceForObs = cachedPrice ? cachedPrice.price : latestPrice
        const evalResult = evaluateObservation(
          product, obs.observation_date, priceForObs, obs.months_since_entry
        )
        evalResult.product_id = product.id
        upsertObserv(evalResult)
        updated++
      }
    }

    res.json({ ok: true, refreshed, updated, failed: failed.length })
  } catch (err) {
    console.error('刷新价格失败:', err)
    res.status(500).json({ error: err.message })
  }
})

// ─────────────────────────────────────────
// GET /api/observations/products
// 获取所有存续产品列表（概要）
// ─────────────────────────────────────────
app.get('/api/observations/products', (req, res) => {
  try {
    const products = queryOngoingProducts()
    res.json({ products })
  } catch (err) {
    res.status(500).json({ error: err.message })
  }
})

// ─────────────────────────────────────────
// 喜报 API
// ─────────────────────────────────────────
app.get('/api/posters/today', (req, res) => {
  try {
    const date = req.query.date || new Date().toISOString().slice(0, 10)
    const today = new Date().toISOString().slice(0, 10)
    const posters = queryPostersByDate(date)
    res.json({ posters, today, queryDate: date })
  } catch (err) {
    res.status(500).json({ error: err.message })
  }
})

app.get('/api/posters', (req, res) => {
  try {
    const { product_id } = req.query
    const posters = product_id ? queryPostersByProduct(product_id) : queryAllPosters()
    res.json({ posters })
  } catch (err) {
    res.status(500).json({ error: err.message })
  }
})

app.post('/api/posters/generate', async (req, res) => {
  try {
    const targetDate = req.body?.date || new Date().toISOString().slice(0, 10)
    const products = queryOngoingProducts()
    const todayProducts = []

    for (const p of products) {
      const obsDates = getObservationDates(p)
      const hasDate = obsDates.some(d => d.date === targetDate)
      if (hasDate) todayProducts.push(p)
    }

    if (todayProducts.length === 0) {
      return res.json({ ok: true, generated: 0, message: `${targetDate} 无产品需要观察` })
    }

    let generated = 0
    let knockoutCount = 0
    let dividendCount = 0

    for (const product of todayProducts) {
      if (!product.code) continue

      const obsDates = getObservationDates(product)
      const targetObs = obsDates.find(d => d.date === targetDate)
      if (!targetObs) continue

      const posterData = generatePosterData(product, targetDate, targetObs.monthsSinceEntry)
      if (!posterData) continue

      const isKnockout = posterData.knockout_value !== null
      const isDividend = posterData.has_dividend_observation && posterData.dividend_barrier_value !== null

      if (isKnockout) {
        knockoutCount++
        const row = {
          product_id: product.id,
          poster_type: 'knockout',
          observation_date: targetDate,
          product_name: product.name || '',
          date_display: formatChineseDate(targetDate),
          months_since_entry: targetObs.monthsSinceEntry,
          underlying_name: posterData.underlying_name,
          absolute_return: posterData.absolute_return,
          annualized_return: posterData.annualized_return,
          duration_months: targetObs.monthsSinceEntry,
          parachute_value: posterData.parachute_value,
          knockout_value: posterData.knockout_value,
          dividend_barrier_value: null,
          dividend_count: 0,
          cumulative_rate: 0,
          monthly_coupon: posterData.monthly_coupon,
          entry_date: product.issue_date,
        }
        upsertPoster(row)
        if (!isDividend) continue
      }

      if (isDividend && !isKnockout) {
        dividendCount++
        const row = {
          product_id: product.id,
          poster_type: 'dividend',
          observation_date: targetDate,
          product_name: product.name || '',
          date_display: formatChineseDate(targetDate),
          months_since_entry: targetObs.monthsSinceEntry,
          underlying_name: posterData.underlying_name,
          absolute_return: 0,
          annualized_return: posterData.annualized_return,
          duration_months: 0,
          parachute_value: posterData.parachute_value,
          knockout_value: posterData.knockout_value,
          dividend_barrier_value: posterData.dividend_barrier_value,
          dividend_count: posterData.dividend_count,
          cumulative_rate: posterData.cumulative_dividend_rate,
          monthly_coupon: posterData.monthly_coupon,
          entry_date: product.issue_date,
        }
        upsertPoster(row)
      }
    }

    generated = knockoutCount + dividendCount
    res.json({
      ok: true,
      generated,
      knockout: knockoutCount,
      dividend: dividendCount,
      today: targetDate,
    })
  } catch (err) {
    console.error('生成喜报失败:', err)
    res.status(500).json({ error: err.message })
  }
})

// ─────────────────────────────────────────
// 定时任务：自动更新价格和观察记录，并在今日有观察产品时发送邮件提醒
// ─────────────────────────────────────────
const cron = require('node-cron')
const CRON_TIMEZONE = process.env.CRON_TIMEZONE || 'Asia/Shanghai'

async function refreshTodayObservations() {
  const products = queryOngoingProducts()
  const codes = [...new Set(products.map(p => p.code).filter(Boolean))]
  if (codes.length === 0) {
    return { products, codes, prices: {}, failed: [], today: new Date().toISOString().slice(0, 10), updatedObs: 0 }
  }

  console.log(`[定时任务] 开始更新 ${codes.length} 个标的价格...`)
  const { results: prices, failed } = await fetchAllPrices(codes)
  const today = new Date().toISOString().slice(0, 10)

  for (const code of codes) {
    if (prices[code] !== undefined) {
      upsertPrice(code, today, prices[code])
    }
  }

  let updatedObs = 0
  for (const product of products) {
    if (!product.code || prices[product.code] === undefined) continue
    const obsDates = getObservationDates(product, today)
    for (const { date, monthsSinceEntry } of obsDates) {
      if (date === today) {
        const evalResult = evaluateObservation(product, date, prices[product.code], monthsSinceEntry)
        evalResult.product_id = product.id
        upsertObserv(evalResult)
        updatedObs++
      }
    }
  }

  return { products, codes, prices, failed, today, updatedObs }
}

async function scheduledPriceUpdate() {
  try {
    const { codes, failed, updatedObs } = await refreshTodayObservations()
    if (codes.length === 0) return
    console.log(`[定时任务] 完成: 价格更新 ${codes.length - failed.length}/${codes.length}, 观察记录更新 ${updatedObs} 条`)
  } catch (err) {
    console.error('[定时任务] 失败:', err.message)
  }
}

async function scheduledObservationEmail() {
  try {
    const refreshed = await refreshTodayObservations()
    if (refreshed.codes.length === 0) return

    const notification = buildTodayObservationNotification({
      products: refreshed.products,
      prices: refreshed.prices,
      today: refreshed.today,
    })

    const result = await sendObservationEmail({ notification })
    if (result.sent) {
      console.log(`[邮件提醒] 已发送今日观察提醒: ${result.count} 个产品`)
    } else {
      console.log(`[邮件提醒] 未发送: ${result.reason}`)
    }
  } catch (err) {
    console.error('[邮件提醒] 失败:', err.message)
  }
}

async function generateAutoPosters() {
  try {
    const today = new Date().toISOString().slice(0, 10)
    const products = queryOngoingProducts()
    let knocked = 0
    let dividends = 0

    for (const product of products) {
      if (!product.code) continue
      const obsDates = getObservationDates(product, today)
      const todayObsInfo = obsDates.find(d => d.date === today)
      if (!todayObsInfo) continue

      const observations = queryObservationsByProduct(product.id)
      const todayObsRecord = observations.find(o => o.observation_date === today)
      if (!todayObsRecord) continue

      const posterData = generatePosterData(product, today, todayObsInfo.monthsSinceEntry)
      if (!posterData) continue

      const isKnockout = posterData.knockout_value !== null && todayObsRecord.is_knocked_out === '是'
      const isDividend = posterData.has_dividend_observation && posterData.dividend_barrier_value !== null && todayObsRecord.is_dividend === '是'

      if (isKnockout) {
        knocked++
        const row = {
          product_id: product.id,
          poster_type: 'knockout',
          observation_date: today,
          product_name: product.name || '',
          date_display: formatChineseDate(today),
          months_since_entry: todayObsInfo.monthsSinceEntry,
          underlying_name: posterData.underlying_name,
          absolute_return: posterData.absolute_return,
          annualized_return: posterData.annualized_return,
          duration_months: todayObsInfo.monthsSinceEntry,
          parachute_value: posterData.parachute_value,
          knockout_value: posterData.knockout_value,
          dividend_barrier_value: null,
          dividend_count: 0,
          cumulative_rate: 0,
          monthly_coupon: posterData.monthly_coupon,
          entry_date: product.issue_date,
        }
        upsertPoster(row)
        if (!isDividend) continue
      }

      if (isDividend && !isKnockout) {
        dividends++
        const row = {
          product_id: product.id,
          poster_type: 'dividend',
          observation_date: today,
          product_name: product.name || '',
          date_display: formatChineseDate(today),
          months_since_entry: todayObsInfo.monthsSinceEntry,
          underlying_name: posterData.underlying_name,
          absolute_return: 0,
          annualized_return: posterData.annualized_return,
          duration_months: 0,
          parachute_value: posterData.parachute_value,
          knockout_value: posterData.knockout_value,
          dividend_barrier_value: posterData.dividend_barrier_value,
          dividend_count: posterData.dividend_count,
          cumulative_rate: posterData.cumulative_dividend_rate,
          monthly_coupon: posterData.monthly_coupon,
          entry_date: product.issue_date,
        }
        upsertPoster(row)
      }
    }

    console.log(`[喜报生成] 今日自动生成：敲出喜报 ${knocked} 张，派息喜报 ${dividends} 张`)
  } catch (err) {
    console.error('[喜报生成] 失败:', err.message)
  }
}

cron.schedule('30 11 * * 1-5', scheduledPriceUpdate, { timezone: CRON_TIMEZONE })
cron.schedule('0 15 * * 1-5', scheduledPriceUpdate, { timezone: CRON_TIMEZONE })
cron.schedule('30 15 * * 1-5', scheduledPriceUpdate, { timezone: CRON_TIMEZONE })
cron.schedule('5 15 * * 1-5', generateAutoPosters, { timezone: CRON_TIMEZONE })
cron.schedule('0 10 * * *', scheduledObservationEmail, { timezone: CRON_TIMEZONE })
cron.schedule('10 15 * * *', scheduledObservationEmail, { timezone: CRON_TIMEZONE })
console.log(`定时任务已注册: 工作日 11:30, 15:00, 15:30 更新价格；15:05 自动生成喜报；每日 10:00, 15:10 邮件提醒 (${CRON_TIMEZONE})`)

// 先初始化数据库，再启动服务器
initDatabase().then(() => {
  app.listen(PORT, () => {
    console.log(`飞书中转服务已启动: http://localhost:${PORT}`)
    if (!FEISHU_APP_ID || FEISHU_APP_ID === 'your_app_id_here') {
      console.warn('警告: 请先在 server/.env 中填写 FEISHU_APP_ID 和 FEISHU_APP_SECRET')
    }
  })
}).catch(err => {
  console.error('数据库初始化失败:', err)
  process.exit(1)
})
