# AI Agent 智能助手 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 新增一个独立聊天页面 `/agent`，作为基于 DeepSeek API 的智能业务助手，支持文档关键词检索（RAG）和 Function Calling（查询产品/观察日/价格/统计），流式输出响应。

**Architecture:** 后端新增 `agentService.js` 编排 DeepSeek chat completions API（OpenAI 兼容格式）。用户消息到达后：1) 从 SQLite 加载历史消息组装上下文；2) 通过关键词检索 `product_docs` 表找到相关文档注入 system prompt；3) 调用 DeepSeek API 并启用 stream + tools；4) 若模型返回 tool_calls 则在本地执行工具函数，将结果回填继续对话；5) 文本内容通过 SSE 实时推送给前端；6) 完成后保存消息到 DB。前端使用 Vue 3 实现 ChatGPT 风格对话界面，用 `marked` 渲染 Markdown。

**Tech Stack:** DeepSeek API (`https://api.deepseek.com`, model `deepseek-chat`), Node.js 原生 `fetch` + SSE, Express, sql.js (SQLite), Vue 3, marked, dompurify

---

## File Structure

| Action | File | Responsibility |
|--------|------|----------------|
| Modify | `backend/db.js` | 新增 `agent_conversations` + `agent_messages` 表 + CRUD 函数 |
| Create | `backend/services/documentRetriever.js` | 关键词检索 product_docs，返回相关文档片段 |
| Create | `backend/services/agentTools.js` | Function calling 工具定义（JSON schema） + 执行器 |
| Create | `backend/services/agentService.js` | Agent 编排：DeepSeek API 调用 + 工具循环 + SSE 流 |
| Create | `backend/services/agentTools.test.js` | 工具函数单元测试 |
| Modify | `backend/index.js` | 新增 Agent API 端点（chat SSE、会话 CRUD）+ 环境变量 |
| Modify | `backend/.env.example` | 新增 `DEEPSEEK_API_KEY` |
| Modify | `frontend/vite.config.js` | 新增 `/api/agent/chat` 独立代理防 SSE 缓冲 |
| Create | `frontend/views/AgentChat.vue` | 聊天主页面（会话列表 + 消息区 + 输入框） |
| Create | `frontend/components/ChatMessage.vue` | 单条消息渲染组件（Markdown + 代码块） |
| Modify | `frontend/router/index.js` | 新增 `/agent` 路由 |
| Modify | `frontend/components/SidebarNav.vue` | 新增"智能助手"导航项（Bot 图标） |
| Modify | `frontend/package.json` | 新增 `marked`、`dompurify` 依赖 |

---

### Task 1: DB 表与会话 CRUD

**Files:**
- Modify: `backend/db.js:197` (在 push_config 表后追加两个 CREATE TABLE)
- Modify: `backend/db.js:749-767` (在 module.exports 前新增函数并导出)

- [ ] **Step 1: 新增 `agent_conversations` 和 `agent_messages` 表**

在 `backend/db.js` 的 `db.exec(...)` 块中，`push_config` 表定义之后（第 197 行 `);` 后面），追加以下 SQL：

```sql
    CREATE TABLE IF NOT EXISTS agent_conversations (
      id INTEGER PRIMARY KEY AUTOINCREMENT,
      title TEXT NOT NULL DEFAULT '新对话',
      created_at TEXT NOT NULL DEFAULT (datetime('now')),
      updated_at TEXT NOT NULL DEFAULT (datetime('now'))
    );

    CREATE TABLE IF NOT EXISTS agent_messages (
      id INTEGER PRIMARY KEY AUTOINCREMENT,
      conversation_id INTEGER NOT NULL,
      role TEXT NOT NULL,
      content TEXT NOT NULL DEFAULT '',
      tool_calls TEXT DEFAULT NULL,
      tool_call_id TEXT DEFAULT NULL,
      created_at TEXT NOT NULL DEFAULT (datetime('now')),
      FOREIGN KEY (conversation_id) REFERENCES agent_conversations(id) ON DELETE CASCADE
    );
```

- [ ] **Step 2: 新增 CRUD 函数**

在 `backend/db.js` 的 `module.exports` 之前（约第 750 行），追加：

```js
// ──── Agent Conversations ────

function getAgentConversations() {
  return queryAll('SELECT * FROM agent_conversations ORDER BY updated_at DESC')
}

function getAgentConversation(id) {
  return queryOne('SELECT * FROM agent_conversations WHERE id = ?', [id])
}

function createAgentConversation(title) {
  const now = new Date().toISOString()
  db.run(
    'INSERT INTO agent_conversations (title, created_at, updated_at) VALUES (?, ?, ?)',
    [title || '新对话', now, now]
  )
  saveDatabase()
  const inserted = queryOne('SELECT last_insert_rowid() as id')
  return inserted.id
}

function updateAgentConversationTitle(id, title) {
  const now = new Date().toISOString()
  db.run(
    'UPDATE agent_conversations SET title = ?, updated_at = ? WHERE id = ?',
    [title, now, id]
  )
  saveDatabase()
}

function touchConversation(id) {
  const now = new Date().toISOString()
  db.run('UPDATE agent_conversations SET updated_at = ? WHERE id = ?', [now, id])
  saveDatabase()
}

function deleteAgentConversation(id) {
  db.run('DELETE FROM agent_messages WHERE conversation_id = ?', [id])
  db.run('DELETE FROM agent_conversations WHERE id = ?', [id])
  saveDatabase()
}

function getAgentMessages(conversationId) {
  return queryAll(
    'SELECT * FROM agent_messages WHERE conversation_id = ? ORDER BY created_at ASC',
    [conversationId]
  )
}

function addAgentMessage({ conversation_id, role, content, tool_calls, tool_call_id }) {
  const now = new Date().toISOString()
  db.run(
    'INSERT INTO agent_messages (conversation_id, role, content, tool_calls, tool_call_id, created_at) VALUES (?, ?, ?, ?, ?, ?)',
    [conversation_id, role, content || '', tool_calls ? JSON.stringify(tool_calls) : null, tool_call_id || null, now]
  )
  touchConversation(conversation_id)
}
```

- [ ] **Step 3: 导出新函数**

在 `module.exports` 对象中追加：

```js
  getAgentConversations, getAgentConversation, createAgentConversation,
  updateAgentConversationTitle, deleteAgentConversation,
  getAgentMessages, addAgentMessage,
```

- [ ] **Step 4: 验证服务器启动**

运行：
```bash
cd backend && npm run dev
```

预期：服务启动成功，日志输出"数据库初始化完成"。停止服务。

- [ ] **Step 5: Commit**

```bash
git add backend/db.js
git commit -m "feat: add agent_conversations and agent_messages tables with CRUD"
```

---

### Task 2: 文档关键词检索模块

**Files:**
- Create: `backend/services/documentRetriever.js`

- [ ] **Step 1: 创建检索模块**

创建 `backend/services/documentRetriever.js`：

```js
const dbModule = require('../db')

function searchDocs(query, limit = 5) {
  if (!query || !query.trim()) return []
  const keywords = query.trim().split(/\s+/).filter(k => k.length > 0)
  const docs = dbModule.getAllProductDocs()
  const scored = []

  for (const doc of docs) {
    const text = `${doc.doc_name} ${doc.parent_path} ${doc.raw_content || ''}`
    const lower = text.toLowerCase()
    let score = 0
    for (const kw of keywords) {
      const lowerKw = kw.toLowerCase()
      let idx = lower.indexOf(lowerKw)
      while (idx !== -1) {
        score += 1
        idx = lower.indexOf(lowerKw, idx + 1)
      }
    }
    if (score > 0) {
      const structure = doc.structure_json ? JSON.parse(doc.structure_json) : null
      scored.push({
        doc_name: doc.doc_name,
        parent_path: doc.parent_path,
        raw_content: doc.raw_content || '',
        structure,
        score,
      })
    }
  }

  scored.sort((a, b) => b.score - a.score)
  return scored.slice(0, limit)
}

function buildDocContext(docs) {
  if (!docs || docs.length === 0) return ''
  const parts = docs.map((d, i) => {
    const structStr = d.structure ? `\n结构信息：${JSON.stringify(d.structure)}` : ''
    return `[文档${i + 1}] ${d.doc_name} (${d.parent_path})\n${d.raw_content}${structStr}`
  })
  return `\n\n以下是与用户问题相关的文档资料，请参考这些文档回答问题：\n\n${parts.join('\n\n---\n\n')}`
}

module.exports = { searchDocs, buildDocContext }
```

- [ ] **Step 2: Commit**

```bash
git add backend/services/documentRetriever.js
git commit -m "feat: add keyword-based document retriever for agent RAG"
```

---

### Task 3: Function Calling 工具定义与执行器

**Files:**
- Create: `backend/services/agentTools.js`
- Create: `backend/services/agentTools.test.js`

- [ ] **Step 1: 编写工具函数测试**

创建 `backend/services/agentTools.test.js`：

```js
const assert = require('node:assert/strict')
const { TOOL_DEFINITIONS, executeTool } = require('./agentTools')

assert(Array.isArray(TOOL_DEFINITIONS))
assert(TOOL_DEFINITIONS.length >= 5, `Expected >=5 tools, got ${TOOL_DEFINITIONS.length}`)

const names = TOOL_DEFINITIONS.map(t => t.function.name)
assert(names.includes('search_products'), 'Missing search_products tool')
assert(names.includes('get_product_detail'), 'Missing get_product_detail tool')
assert(names.includes('get_observations'), 'Missing get_observations tool')
assert(names.includes('get_price'), 'Missing get_price tool')
assert(names.includes('get_dashboard_stats'), 'Missing get_dashboard_stats tool')

for (const tool of TOOL_DEFINITIONS) {
  assert(tool.type === 'function', `Tool ${tool.function.name} must have type "function"`)
  assert(tool.function.name, 'Tool must have a name')
  assert(tool.function.description, `Tool ${tool.function.name} missing description`)
  assert(tool.function.parameters, `Tool ${tool.function.name} missing parameters`)
}

console.log('agentTools tests passed')
```

- [ ] **Step 2: 运行测试验证失败**

运行：
```bash
node backend/services/agentTools.test.js
```

预期：FAIL，"Cannot find module './agentTools'"

- [ ] **Step 3: 创建工具模块**

创建 `backend/services/agentTools.js`：

```js
const dbModule = require('../db')
const { fetchLatestPrice } = require('./priceService')

const TOOL_DEFINITIONS = [
  {
    type: 'function',
    function: {
      name: 'search_products',
      description: '根据关键词搜索产品名称。返回匹配的产品列表（id、名称、存续状态）。用于用户模糊查找某个产品时使用。',
      parameters: {
        type: 'object',
        properties: {
          keyword: { type: 'string', description: '搜索关键词，如产品名称的一部分' },
        },
        required: ['keyword'],
      },
    },
  },
  {
    type: 'function',
    function: {
      name: 'get_product_detail',
      description: '获取指定产品的详细信息，包括产品结构、标的、入场价、敲出线、派息线等全部字段。当用户询问某个具体产品的详情时使用。',
      parameters: {
        type: 'object',
        properties: {
          product_id: { type: 'string', description: '产品 ID（航班编号）' },
        },
        required: ['product_id'],
      },
    },
  },
  {
    type: 'function',
    function: {
      name: 'get_observations',
      description: '获取指定产品的观察日记录，包括每个月的观察日期、敲出价、派息线、标的价格、是否敲出/派息等。当用户询问产品的观察结果或历史表现时使用。',
      parameters: {
        type: 'object',
        properties: {
          product_id: { type: 'string', description: '产品 ID（航班编号）' },
        },
        required: ['product_id'],
      },
    },
  },
  {
    type: 'function',
    function: {
      name: 'get_price',
      description: '获取标的证券的最新实时价格（从东方财富 API 获取）。当用户询问某个标的的当前价格时使用。',
      parameters: {
        type: 'object',
        properties: {
          code: { type: 'string', description: '标的代码，如 sh000300、sz399006' },
        },
        required: ['code'],
      },
    },
  },
  {
    type: 'function',
    function: {
      name: 'get_dashboard_stats',
      description: '获取业务总览统计数据，包括产品总数、存续产品数、客户总数、渠道总数等。当用户询问整体业务情况或统计数据时使用。',
      parameters: {
        type: 'object',
        properties: {},
      },
    },
  },
]

async function executeTool(name, args) {
  try {
    switch (name) {
      case 'search_products':
        return executeSearchProducts(args)
      case 'get_product_detail':
        return executeGetProductDetail(args)
      case 'get_observations':
        return executeGetObservations(args)
      case 'get_price':
        return executeGetPrice(args)
      case 'get_dashboard_stats':
        return executeGetDashboardStats(args)
      default:
        return { error: `Unknown tool: ${name}` }
    }
  } catch (err) {
    return { error: err.message }
  }
}

function executeSearchProducts({ keyword }) {
  const db = dbModule.db
  const like = `%${keyword}%`
  const rows = db.exec(
    'SELECT id, name, holding_status, code FROM products WHERE name LIKE ? LIMIT 10',
    [like]
  )[0]?.values || []
  return {
    count: rows.length,
    products: rows.map(r => ({
      id: r[0], name: r[1], holding_status: r[2], code: r[3],
    })),
  }
}

function executeGetProductDetail({ product_id }) {
  const db = dbModule.db
  const rows = db.exec('SELECT * FROM products WHERE id = ?', [product_id])
  if (!rows[0] || rows[0].values.length === 0) {
    return { error: `Product ${product_id} not found` }
  }
  const columns = rows[0].columns
  const values = rows[0].values[0]
  const product = {}
  columns.forEach((col, i) => { product[col] = values[i] })
  return { product }
}

function executeGetObservations({ product_id }) {
  const observations = dbModule.queryObservationsByProduct(product_id)
  return {
    product_id,
    count: observations.length,
    observations: observations.map(o => ({
      observation_date: o.observation_date,
      knockout_price: o.knockout_price,
      dividend_line: o.dividend_line,
      underlying_price: o.underlying_price,
      is_knocked_out: o.is_knocked_out,
      is_dividend: o.is_dividend,
    })),
  }
}

async function executeGetPrice({ code }) {
  const price = await fetchLatestPrice(code)
  return { code, price, fetched_at: new Date().toISOString() }
}

function executeGetDashboardStats() {
  const db = dbModule.db
  const totalProducts = db.exec('SELECT COUNT(*) FROM products')[0]?.values[0][0] || 0
  const ongoingProducts = db.exec("SELECT COUNT(*) FROM products WHERE holding_status = '存续'")[0]?.values[0][0] || 0
  const completedProducts = db.exec("SELECT COUNT(*) FROM products WHERE holding_status = '已结束'")[0]?.values[0][0] || 0
  const totalCustomers = db.exec('SELECT COUNT(*) FROM customers')[0]?.values[0][0] || 0
  const totalChannels = db.exec('SELECT COUNT(*) FROM channels')[0]?.values[0][0] || 0
  return { totalProducts, ongoingProducts, completedProducts, totalCustomers, totalChannels }
}

module.exports = { TOOL_DEFINITIONS, executeTool }
```

- [ ] **Step 4: 运行测试验证通过**

运行：
```bash
node backend/services/agentTools.test.js
```

预期：输出"agentTools tests passed"

- [ ] **Step 5: Commit**

```bash
git add backend/services/agentTools.js backend/services/agentTools.test.js
git commit -m "feat: add agent tool definitions and executor for DeepSeek function calling"
```

---

### Task 4: Agent Service — DeepSeek API 编排

**Files:**
- Create: `backend/services/agentService.js`

- [ ] **Step 1: 创建 Agent Service**

创建 `backend/services/agentService.js`：

```js
const { TOOL_DEFINITIONS, executeTool } = require('./agentTools')
const { searchDocs, buildDocContext } = require('./documentRetriever')
const dbModule = require('../db')

const DEEPSEEK_API_URL = process.env.DEEPSEEK_API_URL || 'https://api.deepseek.com'
const DEEPSEEK_MODEL = process.env.DEEPSEEK_MODEL || 'deepseek-chat'
const DEEPSEEK_API_KEY = process.env.DEEPSEEK_API_KEY || ''
const MAX_TOOL_ROUNDS = 5

const SYSTEM_PROMPT = `你是一个专业的金融结构化产品业务助手，服务于"航班服务"业务工作台系统。

你的职责：
1. 回答关于结构化金融产品的专业知识问题（如敲出、派息、观察日、降落伞等概念）
2. 查询具体产品的状态、观察结果、标的价格等业务数据
3. 帮助用户理解如何使用业务工作台系统
4. 提供产品分析和对比的辅助决策建议

注意事项：
- 回答要准确，基于实际的系统数据和文档
- 对于金额、比率等数据，保留原始精度
- 如果不确定，明确告知用户并建议使用系统中的其他功能
- 使用中文回答`

async function streamChat({ conversationId, userMessage, abortSignal, onDelta, onToolCall, onDone, onError }) {
  const messages = await buildMessageHistory(conversationId, userMessage)

  let round = 0
  while (round < MAX_TOOL_ROUNDS) {
    round++
    if (abortSignal?.aborted) {
      onError(new Error('Client disconnected'))
      return
    }
    const result = await callDeepSeek(messages, abortSignal, onDelta, onToolCall)

    if (result.finishReason === 'tool_calls' && result.toolCalls.length > 0) {
      const assistantMsg = { role: 'assistant', content: result.content || '', tool_calls: result.toolCalls }
      messages.push(assistantMsg)
      dbModule.addAgentMessage({ conversation_id: conversationId, role: 'assistant', content: result.content || '', tool_calls: result.toolCalls })

      for (const tc of result.toolCalls) {
        const args = JSON.parse(tc.function.arguments || '{}')
        if (onToolCall) onToolCall(tc.function.name, args)
        const toolResult = await executeTool(tc.function.name, args)
        messages.push({
          role: 'tool',
          tool_call_id: tc.id,
          content: JSON.stringify(toolResult),
        })
        dbModule.addAgentMessage({ conversation_id: conversationId, role: 'tool', content: JSON.stringify(toolResult), tool_call_id: tc.id })
      }
      continue
    }

    dbModule.addAgentMessage({
      conversation_id: conversationId,
      role: 'assistant',
      content: result.content,
      tool_calls: result.toolCalls.length > 0 ? result.toolCalls : null,
    })

    if (onDone) onDone(result.usage)
    return
  }

  onError(new Error('Max tool call rounds exceeded'))
}

async function buildMessageHistory(conversationId, userMessage) {
  const dbMessages = dbModule.getAgentMessages(conversationId)

  const historyMessages = dbMessages.map(m => {
    const msg = { role: m.role, content: m.content }
    if (m.tool_calls) {
      try { msg.tool_calls = JSON.parse(m.tool_calls) } catch {}
    }
    if (m.tool_call_id) msg.tool_call_id = m.tool_call_id
    return msg
  })

  const relevantDocs = searchDocs(userMessage)
  const docContext = buildDocContext(relevantDocs)
  const systemContent = SYSTEM_PROMPT + docContext

  return [
    { role: 'system', content: systemContent },
    ...historyMessages,
    { role: 'user', content: userMessage },
  ]
}

async function callDeepSeek(messages, abortSignal, onDelta, onToolCall) {
  const body = {
    model: DEEPSEEK_MODEL,
    messages,
    tools: TOOL_DEFINITIONS,
    stream: true,
    stream_options: { include_usage: true },
  }

  const response = await fetch(`${DEEPSEEK_API_URL}/chat/completions`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${DEEPSEEK_API_KEY}`,
    },
    body: JSON.stringify(body),
    signal: abortSignal,
  })

  if (!response.ok) {
    const text = await response.text()
    throw new Error(`DeepSeek API error ${response.status}: ${text}`)
  }

  const reader = response.body.getReader()
  const decoder = new TextDecoder()

  let content = ''
  let toolCalls = []
  let finishReason = null
  let usage = null
  let sseBuffer = ''

  while (true) {
    const { done, value } = await reader.read()
    if (done) {
      if (sseBuffer.trim()) {
        const lastLine = sseBuffer.trim()
        if (lastLine.startsWith('data: ') && lastLine.slice(6) !== '[DONE]') {
          try {
            const parsed = JSON.parse(lastLine.slice(6))
            if (parsed.usage) usage = parsed.usage
          } catch {}
        }
      }
      break
    }

    sseBuffer += decoder.decode(value, { stream: true })
    const lines = sseBuffer.split('\n')
    sseBuffer = lines.pop()

    for (const line of lines) {
      const trimmed = line.trim()
      if (!trimmed || !trimmed.startsWith('data: ')) continue
      const data = trimmed.slice(6)
      if (data === '[DONE]') {
        sseBuffer = ''
        continue
      }

      let parsed
      try { parsed = JSON.parse(data) } catch { continue }

      if (parsed.usage) {
        usage = parsed.usage
      }

      const choice = parsed.choices?.[0]
      if (!choice) continue

      const delta = choice.delta || {}

      if (delta.content) {
        content += delta.content
        if (onDelta) onDelta(delta.content)
      }

      if (delta.tool_calls) {
        for (const tc of delta.tool_calls) {
          const idx = tc.index
          if (!toolCalls[idx]) {
            toolCalls[idx] = {
              id: tc.id || '',
              type: 'function',
              function: { name: '', arguments: '' },
            }
          }
          if (tc.id) toolCalls[idx].id = tc.id
          if (tc.function?.name) toolCalls[idx].function.name += tc.function.name
          if (tc.function?.arguments) toolCalls[idx].function.arguments += tc.function.arguments
        }
      }

      if (choice.finish_reason) {
        finishReason = choice.finish_reason
      }
    }
  }

  toolCalls = toolCalls.filter(Boolean)
  return { content, toolCalls, finishReason, usage }
}

module.exports = { streamChat }
```

- [ ] **Step 2: Commit**

```bash
git add backend/services/agentService.js
git commit -m "feat: add agent service with DeepSeek streaming and tool call loop"
```

---

### Task 5: Agent API 端点

**Files:**
- Modify: `backend/index.js:14-15` (添加 import)
- Modify: `backend/index.js:1772` (在 activity-logs 端点后、cron 前添加 Agent API)
- Modify: `backend/.env.example` (添加 DEEPSEEK_API_KEY)

- [ ] **Step 1: 添加 service import**

在 `backend/index.js` 第 14 行（`feishuPushService` import 之后），添加：

```js
const { streamChat } = require('./services/agentService')
```

在 dbModule 的解构导入（第 6 行）中追加：

```
getAgentConversations, getAgentConversation, createAgentConversation, updateAgentConversationTitle, deleteAgentConversation, getAgentMessages, addAgentMessage
```

- [ ] **Step 2: 添加 Agent API 端点**

在 `backend/index.js` 的 Activity logs 端点之后（第 1772 行 `})` 之后），cron 定时任务之前（第 1774 行之前），插入：

```js
// ─────────────────────────────────────────
// Agent Chat (SSE)
// ─────────────────────────────────────────
app.get('/api/agent/conversations', (req, res) => {
  try {
    const conversations = getAgentConversations()
    res.json(conversations)
  } catch (e) {
    res.status(500).json({ error: e.message })
  }
})

app.post('/api/agent/conversations', (req, res) => {
  try {
    const title = req.body.title || '新对话'
    const id = createAgentConversation(title)
    res.json({ id, title, created_at: new Date().toISOString() })
  } catch (e) {
    res.status(500).json({ error: e.message })
  }
})

app.delete('/api/agent/conversations/:id', (req, res) => {
  try {
    deleteAgentConversation(parseInt(req.params.id))
    res.json({ ok: true })
  } catch (e) {
    res.status(500).json({ error: e.message })
  }
})

app.get('/api/agent/conversations/:id/messages', (req, res) => {
  try {
    const messages = getAgentMessages(parseInt(req.params.id))
    res.json(messages)
  } catch (e) {
    res.status(500).json({ error: e.message })
  }
})

app.post('/api/agent/chat', async (req, res) => {
  const { conversation_id, message } = req.body
  if (!message || !message.trim()) {
    return res.status(400).json({ error: 'message is required' })
  }
  if (!process.env.DEEPSEEK_API_KEY) {
    return res.status(500).json({ error: 'DEEPSEEK_API_KEY not configured' })
  }

  let convId = conversation_id
  if (!convId) {
    convId = createAgentConversation(message.slice(0, 30))
  }

  addAgentMessage({ conversation_id: convId, role: 'user', content: message })

  if (getAgentMessages(convId).length <= 1) {
    updateAgentConversationTitle(convId, message.slice(0, 30))
  }

  res.setHeader('Content-Type', 'text/event-stream')
  res.setHeader('Cache-Control', 'no-cache')
  res.setHeader('Connection', 'keep-alive')
  res.setHeader('X-Accel-Buffering', 'no')

  res.write(`data: ${JSON.stringify({ type: 'conversation_id', conversation_id: convId })}\n\n`)

  const abortController = new AbortController()
  let clientDisconnected = false

  req.on('close', () => {
    clientDisconnected = true
    abortController.abort()
  })

  try {
    await streamChat({
      conversationId: convId,
      userMessage: message,
      abortSignal: abortController.signal,
      onDelta(text) {
        if (!clientDisconnected) res.write(`data: ${JSON.stringify({ type: 'delta', text })}\n\n`)
      },
      onToolCall(name) {
        if (!clientDisconnected) res.write(`data: ${JSON.stringify({ type: 'tool_call', name })}\n\n`)
      },
      onDone(usage) {
        if (!clientDisconnected) {
          res.write(`data: ${JSON.stringify({ type: 'done', usage })}\n\n`)
          res.end()
        }
      },
      onError(err) {
        if (!clientDisconnected) {
          res.write(`data: ${JSON.stringify({ type: 'error', error: err.message })}\n\n`)
          res.end()
        }
      },
    })
  } catch (err) {
    if (!clientDisconnected) {
      try {
        res.write(`data: ${JSON.stringify({ type: 'error', error: err.message })}\n\n`)
        res.end()
      } catch {}
    }
  }
})
```

- [ ] **Step 3: 更新 .env.example**

在 `backend/.env.example` 末尾追加：

```env

# DeepSeek API for Agent
DEEPSEEK_API_KEY=your_deepseek_api_key_here
DEEPSEEK_API_URL=https://api.deepseek.com
DEEPSEEK_MODEL=deepseek-chat
```

- [ ] **Step 4: 在 vite.config.js 中配置 SSE 代理防缓冲**

修改 `frontend/vite.config.js`，为 `/api/agent` 路径添加独立的代理配置（必须在通用 `/api` 之前）：

```js
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  server: {
    proxy: {
      '/api/agent/chat': {
        target: 'http://localhost:3001',
        changeOrigin: true,
        headers: {
          Connection: 'keep-alive',
          'Cache-Control': 'no-cache',
          'X-Accel-Buffering': 'no',
        },
      },
      '/api': 'http://localhost:3001',
      '/public': 'http://localhost:3001',
    },
  },
})
```

- [ ] **Step 5: 验证服务器启动**

运行：
```bash
cd backend && npm run dev
```

预期：服务启动成功。使用 curl 测试 `GET /api/agent/conversations` 返回空数组。停止服务。

- [ ] **Step 6: Commit**

```bash
git add backend/index.js backend/.env.example
git commit -m "feat: add Agent API endpoints with SSE streaming for DeepSeek chat"
```

---

### Task 6: 前端 Markdown 渲染组件

**Files:**
- Create: `frontend/components/ChatMessage.vue`
- Modify: `frontend/package.json`

- [ ] **Step 1: 安装依赖**

运行：
```bash
cd frontend && npm install marked dompurify
```

- [ ] **Step 2: 创建 ChatMessage 组件**

创建 `frontend/components/ChatMessage.vue`：

```vue
<template>
  <div class="chat-message" :class="[`role-${role}`, { streaming }]">
    <div class="message-avatar">
      <component :is="role === 'user' ? User : Bot" :size="18" :stroke-width="2" />
    </div>
    <div class="message-body">
      <div class="message-header">
        {{ role === 'user' ? '你' : '智能助手' }}
      </div>
      <div class="message-content" v-html="renderedContent"></div>
      <div v-if="toolCalls && toolCalls.length" class="tool-calls">
        <div v-for="(tc, i) in toolCalls" :key="i" class="tool-call-badge">
          调用工具：{{ tc.function?.name || tc }}
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { User, Bot } from '@lucide/vue'
import { marked } from 'marked'
import DOMPurify from 'dompurify'

const props = defineProps({
  role: { type: String, required: true },
  content: { type: String, default: '' },
  streaming: { type: Boolean, default: false },
  toolCalls: { type: Array, default: null },
})

marked.setOptions({
  breaks: true,
  gfm: true,
})

const renderedContent = computed(() => {
  if (!props.content) return ''
  const raw = marked.parse(props.content)
  return DOMPurify.sanitize(raw)
})
</script>

<style scoped>
.chat-message {
  display: flex;
  gap: 12px;
  padding: 16px 0;
}

.message-avatar {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  background: var(--bg-hover);
  color: var(--ink-soft);
}

.role-user .message-avatar {
  background: var(--brand-soft);
  color: var(--brand);
}

.role-assistant .message-avatar {
  background: var(--surface-muted);
  color: var(--ink);
}

.message-body {
  flex: 1;
  min-width: 0;
}

.message-header {
  font-size: 13px;
  font-weight: 600;
  color: var(--ink-strong);
  margin-bottom: 4px;
}

.message-content {
  font-size: 14.5px;
  line-height: 1.7;
  color: var(--ink);
  word-break: break-word;
}

.message-content :deep(p) {
  margin: 0 0 8px 0;
}

.message-content :deep(p:last-child) {
  margin-bottom: 0;
}

.message-content :deep(code) {
  background: var(--bg-hover);
  padding: 2px 6px;
  border-radius: 4px;
  font-family: var(--font-mono);
  font-size: 13px;
}

.message-content :deep(pre) {
  background: var(--bg-hover);
  padding: 12px 16px;
  border-radius: 8px;
  overflow-x: auto;
  margin: 8px 0;
}

.message-content :deep(pre code) {
  background: none;
  padding: 0;
}

.message-content :deep(table) {
  border-collapse: collapse;
  width: 100%;
  margin: 8px 0;
  font-size: 13px;
}

.message-content :deep(th),
.message-content :deep(td) {
  border: 1px solid var(--border);
  padding: 6px 12px;
  text-align: left;
}

.message-content :deep(th) {
  background: var(--bg-hover);
  font-weight: 600;
}

.tool-calls {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-top: 8px;
}

.tool-call-badge {
  font-size: 11px;
  padding: 2px 8px;
  border-radius: 4px;
  background: var(--brand-soft);
  color: var(--brand);
  font-weight: 500;
}

.streaming .message-content::after {
  content: '▍';
  animation: blink 1s step-end infinite;
  color: var(--brand);
}

@keyframes blink {
  50% { opacity: 0; }
}
</style>
```

- [ ] **Step 3: Commit**

```bash
git add frontend/components/ChatMessage.vue frontend/package.json frontend/package-lock.json
git commit -m "feat: add ChatMessage component with Markdown rendering"
```

---

### Task 7: Agent 聊天主页面

**Files:**
- Create: `frontend/views/AgentChat.vue`

- [ ] **Step 1: 创建 AgentChat.vue**

创建 `frontend/views/AgentChat.vue`：

```vue
<template>
  <WorkbenchLayout wide>
    <div class="agent-layout">
      <div class="conversation-sidebar" :class="{ collapsed: sidebarCollapsed }">
        <div class="sidebar-header">
          <button class="btn btn-primary btn-sm new-chat-btn" @click="newConversation">
            <Plus :size="16" /> 新对话
          </button>
          <button class="btn btn-outline btn-sm toggle-btn" @click="sidebarCollapsed = !sidebarCollapsed">
            <component :is="sidebarCollapsed ? PanelRightOpen : PanelRightClose" :size="16" />
          </button>
        </div>
        <div v-if="!sidebarCollapsed" class="conversation-list">
          <div
            v-for="conv in conversations"
            :key="conv.id"
            class="conversation-item"
            :class="{ active: currentConversationId === conv.id }"
            @click="selectConversation(conv.id)"
          >
            <MessageSquare :size="15" />
            <span class="conv-title">{{ conv.title }}</span>
            <button class="conv-delete" @click.stop="deleteConversation(conv.id)" title="删除">
              <Trash2 :size="13" />
            </button>
          </div>
          <div v-if="conversations.length === 0" class="empty-conversations">
            暂无对话记录
          </div>
        </div>
      </div>

      <div class="chat-main">
        <div ref="messagesContainer" class="messages-area">
          <div v-if="messages.length === 0" class="welcome-state">
            <Bot :size="48" :stroke-width="1.5" class="welcome-icon" />
            <h2 class="text-section">智能业务助手</h2>
            <p class="text-body" style="color:var(--ink-soft);max-width:400px;text-align:center">
              我可以帮你查询产品信息、分析观察结果、解读产品文档，或回答业务相关问题。试试问我：
            </p>
            <div class="suggestion-chips">
              <button v-for="s in suggestions" :key="s" class="suggestion-chip" @click="sendSuggestion(s)">
                {{ s }}
              </button>
            </div>
          </div>

          <template v-for="msg in messages" :key="msg.id || msg._tempId">
            <ChatMessage
              :role="msg.role"
              :content="msg.content"
              :streaming="msg.streaming"
              :tool-calls="msg.tool_calls_display"
            />
          </template>

          <div v-if="isLoading && !streaming" class="loading-indicator">
            <div class="dot-pulse"></div>
          </div>
        </div>

        <div class="input-area">
          <form class="input-form" @submit.prevent="sendMessage">
            <textarea
              ref="inputRef"
              v-model="inputText"
              class="input chat-input"
              rows="1"
              placeholder="输入你的问题..."
              @keydown="handleKeydown"
              @input="autoResize"
            ></textarea>
            <button type="submit" class="btn btn-primary send-btn" :disabled="!inputText.trim() || isLoading">
              <SendHorizontal :size="18" />
            </button>
          </form>
        </div>
      </div>
    </div>
  </WorkbenchLayout>
</template>

<script setup>
import { ref, onMounted, nextTick } from 'vue'
import WorkbenchLayout from '../components/WorkbenchLayout.vue'
import ChatMessage from '../components/ChatMessage.vue'
import {
  Bot, Plus, MessageSquare, Trash2, SendHorizontal,
  PanelRightOpen, PanelRightClose,
} from '@lucide/vue'

const conversations = ref([])
const currentConversationId = ref(null)
const messages = ref([])
const inputText = ref('')
const isLoading = ref(false)
const streaming = ref(false)
const sidebarCollapsed = ref(false)
const messagesContainer = ref(null)
const inputRef = ref(null)

const suggestions = [
  '目前有多少存续产品？',
  '今天有哪些产品需要观察？',
  '沪深300最新价格是多少？',
  '什么是敲出？什么是派息？',
]

async function loadConversations() {
  try {
    const res = await fetch('/api/agent/conversations')
    if (res.ok) conversations.value = await res.json()
  } catch {}
}

async function selectConversation(id) {
  currentConversationId.value = id
  try {
    const res = await fetch(`/api/agent/conversations/${id}/messages`)
    if (res.ok) {
      const msgs = await res.json()
      messages.value = msgs.map(m => ({
        ...m,
        tool_calls: null,
        tool_calls_display: m.tool_calls ? (typeof m.tool_calls === 'string' ? JSON.parse(m.tool_calls) : m.tool_calls) : null,
        streaming: false,
      }))
    }
  } catch {}
  scrollToBottom()
}

async function newConversation() {
  currentConversationId.value = null
  messages.value = []
  inputText.value = ''
  inputRef.value?.focus()
}

async function deleteConversation(id) {
  try {
    await fetch(`/api/agent/conversations/${id}`, { method: 'DELETE' })
    conversations.value = conversations.value.filter(c => c.id !== id)
    if (currentConversationId.value === id) {
      currentConversationId.value = null
      messages.value = []
    }
  } catch {}
}

function sendSuggestion(text) {
  inputText.value = text
  sendMessage()
}

async function sendMessage() {
  const text = inputText.value.trim()
  if (!text || isLoading) return

  inputText.value = ''
  isLoading.value = true
  streaming.value = true

  const tempId = `temp-${Date.now()}`
  messages.value.push({ _tempId: tempId, role: 'user', content: text, streaming: false })
  scrollToBottom()

  const assistantMsgId = `assistant-${Date.now()}`
  messages.value.push({ _tempId: assistantMsgId, role: 'assistant', content: '', streaming: true, tool_calls_display: [] })

  try {
    const res = await fetch('/api/agent/chat', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        conversation_id: currentConversationId.value,
        message: text,
      }),
    })

    if (!res.ok) {
      const err = await res.json()
      throw new Error(err.error || 'Request failed')
    }

    const reader = res.body.getReader()
    const decoder = new TextDecoder()
    let buffer = ''

    while (true) {
      const { done, value } = await reader.read()
      if (done) break

      buffer += decoder.decode(value, { stream: true })
      const lines = buffer.split('\n')
      buffer = lines.pop()

      for (const line of lines) {
        if (!line.startsWith('data: ')) continue
        const data = line.slice(6).trim()
        if (!data) continue

        let event
        try { event = JSON.parse(data) } catch { continue }

        if (event.type === 'conversation_id') {
          currentConversationId.value = event.conversation_id
        } else if (event.type === 'delta') {
          const msg = messages.value.find(m => m._tempId === assistantMsgId)
          if (msg) msg.content += event.text
          scrollToBottom()
        } else if (event.type === 'tool_call') {
          const msg = messages.value.find(m => m._tempId === assistantMsgId)
          if (msg) {
            if (!msg.tool_calls_display) msg.tool_calls_display = []
            msg.tool_calls_display.push({ function: { name: event.name } })
          }
        } else if (event.type === 'done') {
          streaming.value = false
          const msg = messages.value.find(m => m._tempId === assistantMsgId)
          if (msg) msg.streaming = false
        } else if (event.type === 'error') {
          const msg = messages.value.find(m => m._tempId === assistantMsgId)
          if (msg) {
            msg.content = `⚠ 错误：${event.error}`
            msg.streaming = false
          }
          streaming.value = false
        }
      }
    }
  } catch (err) {
    const msg = messages.value.find(m => m._tempId === assistantMsgId)
    if (msg) {
      msg.content = `⚠ 请求失败：${err.message}`
      msg.streaming = false
    }
  } finally {
    isLoading.value = false
    streaming.value = false
    const msg = messages.value.find(m => m._tempId === assistantMsgId)
    if (msg) msg.streaming = false
    await loadConversations()
  }
}

function handleKeydown(e) {
  if (e.key === 'Enter' && !e.shiftKey) {
    e.preventDefault()
    sendMessage()
  }
}

function autoResize() {
  const el = inputRef.value
  if (!el) return
  el.style.height = 'auto'
  el.style.height = Math.min(el.scrollHeight, 150) + 'px'
}

function scrollToBottom() {
  nextTick(() => {
    const container = messagesContainer.value
    if (container) container.scrollTop = container.scrollHeight
  })
}

onMounted(() => {
  loadConversations()
  inputRef.value?.focus()
})
</script>

<style scoped>
.agent-layout {
  display: flex;
  height: calc(100vh - 80px);
  gap: 0;
}

.conversation-sidebar {
  width: 260px;
  border-right: 1px solid var(--border);
  display: flex;
  flex-direction: column;
  flex-shrink: 0;
  transition: width 200ms ease;
}

.conversation-sidebar.collapsed {
  width: 48px;
}

.sidebar-header {
  display: flex;
  gap: 6px;
  padding: 12px;
  border-bottom: 1px solid var(--border-soft);
}

.new-chat-btn {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
}

.toggle-btn {
  padding: 6px;
}

.conversation-list {
  flex: 1;
  overflow-y: auto;
  padding: 8px;
}

.conversation-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 12px;
  border-radius: var(--radius-sm);
  cursor: pointer;
  font-size: 13px;
  color: var(--ink);
  transition: background 120ms;
}

.conversation-item:hover {
  background: var(--bg-hover);
}

.conversation-item.active {
  background: var(--bg-active);
}

.conv-title {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.conv-delete {
  opacity: 0;
  background: none;
  border: none;
  color: var(--ink-faint);
  cursor: pointer;
  padding: 2px;
  display: flex;
}

.conversation-item:hover .conv-delete {
  opacity: 1;
}

.empty-conversations {
  text-align: center;
  padding: 24px 12px;
  color: var(--ink-faint);
  font-size: 13px;
}

.chat-main {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.messages-area {
  flex: 1;
  overflow-y: auto;
  padding: 16px 32px;
}

.welcome-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 60px 20px;
  gap: 12px;
}

.welcome-icon {
  color: var(--brand);
  margin-bottom: 8px;
}

.suggestion-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  justify-content: center;
  margin-top: 16px;
}

.suggestion-chip {
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: 20px;
  padding: 8px 16px;
  font-size: 13px;
  color: var(--ink);
  cursor: pointer;
  transition: all 120ms;
}

.suggestion-chip:hover {
  background: var(--brand-soft);
  border-color: var(--brand);
  color: var(--brand);
}

.input-area {
  padding: 12px 32px 16px;
  border-top: 1px solid var(--border-soft);
}

.input-form {
  display: flex;
  gap: 8px;
  align-items: flex-end;
}

.chat-input {
  flex: 1;
  resize: none;
  min-height: 40px;
  max-height: 150px;
  line-height: 1.5;
  font-family: var(--font-sans);
}

.send-btn {
  height: 40px;
  width: 40px;
  padding: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.loading-indicator {
  padding: 16px 0;
  display: flex;
  justify-content: center;
}

.dot-pulse {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--brand);
  animation: pulse 1.2s ease-in-out infinite;
}

@keyframes pulse {
  0%, 100% { transform: scale(0.8); opacity: 0.4; }
  50% { transform: scale(1.2); opacity: 1; }
}

@media (max-width: 860px) {
  .conversation-sidebar {
    display: none;
  }

  .messages-area {
    padding: 12px 16px;
  }

  .input-area {
    padding: 8px 16px 12px;
  }
}
</style>
```

- [ ] **Step 2: Commit**

```bash
git add frontend/views/AgentChat.vue
git commit -m "feat: add AgentChat page with conversation sidebar and streaming chat UI"
```

---

### Task 8: 路由与侧边栏导航

**Files:**
- Modify: `frontend/router/index.js:9`
- Modify: `frontend/components/SidebarNav.vue:55-83`

- [ ] **Step 1: 添加路由**

在 `frontend/router/index.js` 中，现有路由列表最后一个（activity-log 之前或之后），追加：

```js
  { path: '/agent', component: () => import('../views/AgentChat.vue') },
```

- [ ] **Step 2: 添加侧边栏导航项**

在 `frontend/components/SidebarNav.vue` 的 `<script setup>` 中：

1) 在 `@lucide/vue` 的 import 中追加 `Bot` 图标（第 55-64 行区域）：

```js
  Bot,
```

2) 在 `navItems` 数组中追加（第 82 行后）：

```js
  { path: '/agent', title: '智能助手', icon: Bot },
```

- [ ] **Step 3: Commit**

```bash
git add frontend/router/index.js frontend/components/SidebarNav.vue
git commit -m "feat: add /agent route and sidebar nav entry for AI assistant"
```

---

### Task 9: 集成验证

- [ ] **Step 1: 安装全部依赖**

```bash
cd frontend && npm install
```

- [ ] **Step 2: 配置 DeepSeek API Key**

在 `backend/.env` 中添加你的 DeepSeek API Key：

```env
DEEPSEEK_API_KEY=sk-actual-key-here
```

- [ ] **Step 3: 启动完整服务**

Terminal 1:
```bash
cd backend && npm run dev
```

Terminal 2:
```bash
cd frontend && npm run dev
```

- [ ] **Step 4: 验证空状态页面渲染**

浏览器打开 `http://localhost:5173/agent`。

预期：
- 侧边栏显示"智能助手"导航项
- 聊天页面渲染，显示欢迎状态和建议 chip
- 会话列表为空

- [ ] **Step 5: 验证会话 CRUD**

```bash
curl -s -X POST http://localhost:3001/api/agent/conversations -H "Content-Type: application/json" -d "{\"title\":\"测试对话\"}"
curl -s http://localhost:3001/api/agent/conversations
```

预期：POST 返回 `{id: 1, title: "测试对话", ...}`；GET 返回包含该对话的数组。

- [ ] **Step 6: 发送聊天消息验证 SSE 流**

在聊天输入框输入"目前有多少存续产品？"并发送。

预期：
- 用户消息立即显示
- 助手回复流式渲染（逐字出现）
- 工具调用 chip 显示"get_dashboard_stats"
- 回复内容包含产品统计数据

- [ ] **Step 7: 验证多轮对话**

在同一个对话中继续输入"沪深300最新价格是多少？"

预期：
- 新消息追加到对话历史中
- 工具调用 chip 显示"get_price"
- 回复包含价格信息

- [ ] **Step 8: 验证新建对话**

点击"新对话"按钮，再发送一个问题。

预期：创建新对话，旧对话保留在列表中。

- [ ] **Step 9: 验证删除对话**

点击某个对话的删除按钮。

预期：对话从列表中移除，页面清空。

- [ ] **Step 10: 最终 Commit**

```bash
git add -A
git commit -m "feat: AI Agent chat with DeepSeek streaming, function calling, and document RAG"
```