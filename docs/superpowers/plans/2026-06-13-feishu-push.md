# Feishu Push Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add configurable Feishu group bot (Webhook) push notifications for the "Today Observation" feature, with a frontend settings page and dynamic cron scheduling.

**Architecture:** Extend the existing Node.js Express backend with a new `push_config` SQLite table, a `feishuPushService` module, REST API endpoints, and a dynamic `node-cron` task. Add a Vue frontend page for configuring webhook URL and push time.

**Tech Stack:** Node.js, Express, sql.js (SQLite), node-cron, Vue 3, Vite

**Spec:** `docs/superpowers/specs/2026-06-13-feishu-push-design.md`

---

## File Structure

| Action | File | Responsibility |
|--------|------|----------------|
| Modify | `backend/db.js` | Add `push_config` table + CRUD functions |
| Create | `backend/services/feishuPushService.js` | Build push text + send to Feishu webhook |
| Create | `backend/services/feishuPushService.test.js` | Tests for push service |
| Modify | `backend/index.js` | Add API endpoints + dynamic cron task |
| Create | `frontend/views/PushSettings.vue` | Push settings page |
| Modify | `frontend/router/index.js` | Add `/push-settings` route |
| Modify | `frontend/components/SidebarNav.vue` | Add nav item |

---

### Task 1: DB Table and CRUD — `push_config`

**Files:**
- Modify: `backend/db.js:180-186` (append CREATE TABLE after `activity_logs`)
- Modify: `backend/db.js:689-704` (add new exports)

- [ ] **Step 1: Add the `push_config` CREATE TABLE**

In `backend/db.js`, inside the `db.exec(...)` block in `initDatabase()`, append the following SQL after the `activity_logs` table (line 186, before the closing backtick + `)`):

```sql
    CREATE TABLE IF NOT EXISTS push_config (
      id INTEGER PRIMARY KEY CHECK (id = 1),
      webhook_url TEXT NOT NULL DEFAULT '',
      cron_hour INTEGER NOT NULL DEFAULT 9,
      cron_minute INTEGER NOT NULL DEFAULT 0,
      enabled INTEGER NOT NULL DEFAULT 0,
      last_push_time TEXT DEFAULT NULL,
      last_push_result TEXT DEFAULT NULL,
      updated_at TEXT NOT NULL DEFAULT ''
    );
```

- [ ] **Step 2: Add the `getPushConfig` and `upsertPushConfig` functions**

Add the following code before the `module.exports` block (around line 688):

```js
// ──── Push Config ────

function getPushConfig() {
  const row = queryOne('SELECT * FROM push_config WHERE id = 1')
  if (!row) {
    return {
      webhook_url: process.env.FEISHU_PUSH_WEBHOOK || '',
      cron_hour: 9,
      cron_minute: 0,
      enabled: 0,
      last_push_time: null,
      last_push_result: null,
    }
  }
  return {
    webhook_url: row.webhook_url,
    cron_hour: row.cron_hour,
    cron_minute: row.cron_minute,
    enabled: row.enabled,
    last_push_time: row.last_push_time,
    last_push_result: row.last_push_result,
  }
}

function upsertPushConfig({ webhook_url, cron_hour, cron_minute, enabled }) {
  const existing = queryOne('SELECT id FROM push_config WHERE id = 1')
  const now = new Date().toISOString()
  if (existing) {
    runStatement(
      'UPDATE push_config SET webhook_url = ?, cron_hour = ?, cron_minute = ?, enabled = ?, updated_at = ? WHERE id = 1',
      [webhook_url, cron_hour, cron_minute, enabled ? 1 : 0, now]
    )
  } else {
    runStatement(
      'INSERT INTO push_config (id, webhook_url, cron_hour, cron_minute, enabled, updated_at) VALUES (1, ?, ?, ?, ?, ?)',
      [webhook_url, cron_hour, cron_minute, enabled ? 1 : 0, now]
    )
  }
  saveDatabase()
}

function updatePushResult(last_push_time, last_push_result) {
  const existing = queryOne('SELECT id FROM push_config WHERE id = 1')
  if (!existing) return
  runStatement(
    'UPDATE push_config SET last_push_time = ?, last_push_result = ? WHERE id = 1',
    [last_push_time, last_push_result]
  )
  saveDatabase()
}
```

- [ ] **Step 3: Add new functions to module.exports**

In the `module.exports` object at the bottom of `db.js`, add these entries:

```js
  getPushConfig, upsertPushConfig, updatePushResult,
```

- [ ] **Step 4: Verify the server starts without errors**

Run:
```bash
cd backend && npm run dev
```

Expected: Server starts, log shows "数据库初始化完成" with no errors. Stop the server (Ctrl+C).

- [ ] **Step 5: Commit**

```bash
git add backend/db.js
git commit -m "feat: add push_config table and CRUD functions"
```

---

### Task 2: Feishu Push Service

**Files:**
- Create: `backend/services/feishuPushService.js`
- Create: `backend/services/feishuPushService.test.js`

- [ ] **Step 1: Write the failing test**

Create `backend/services/feishuPushService.test.js`:

```js
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
```

- [ ] **Step 2: Run test to verify it fails**

Run:
```bash
node backend/services/feishuPushService.test.js
```

Expected: FAIL with "Cannot find module './feishuPushService'"

- [ ] **Step 3: Create the service**

Create `backend/services/feishuPushService.js`:

```js
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
```

- [ ] **Step 4: Run test to verify it passes**

Run:
```bash
node backend/services/feishuPushService.test.js
```

Expected: "feishuPushService tests passed"

- [ ] **Step 5: Commit**

```bash
git add backend/services/feishuPushService.js backend/services/feishuPushService.test.js
git commit -m "feat: add feishuPushService with text builder and send function"
```

---

### Task 3: API Endpoints + Dynamic Cron

**Files:**
- Modify: `backend/index.js:1689-1695` (replace static cron with dynamic)
- Modify: `backend/index.js:1-14` (add imports)
- Modify: `backend/index.js:1610+` (add API routes before Dashboard stats)

- [ ] **Step 1: Add service imports**

At the top of `backend/index.js` (line 13), add:

```js
const { executeObservationPush } = require('./services/feishuPushService')
```

In the destructured import from `dbModule` on line 6, add `getPushConfig, upsertPushConfig, updatePushResult` to the destructuring list.

- [ ] **Step 2: Add API endpoints**

Add the following code right before the `// Dashboard stats` section (before line 1610):

```js
// ─────────────────────────────────────────
// Push Config API
// ─────────────────────────────────────────
app.get('/api/push-config', (req, res) => {
  try {
    const config = getPushConfig()
    res.json({
      webhook_url: config.webhook_url,
      cron_hour: config.cron_hour,
      cron_minute: config.cron_minute,
      enabled: !!config.enabled,
      last_push_time: config.last_push_time,
      last_push_result: config.last_push_result,
    })
  } catch (err) {
    res.status(500).json({ error: err.message })
  }
})

app.put('/api/push-config', (req, res) => {
  try {
    const { webhook_url, cron_hour, cron_minute, enabled } = req.body
    if (webhook_url !== undefined && typeof webhook_url !== 'string') {
      return res.status(400).json({ error: 'webhook_url must be a string' })
    }
    if (cron_hour !== undefined && (!Number.isInteger(cron_hour) || cron_hour < 0 || cron_hour > 23)) {
      return res.status(400).json({ error: 'cron_hour must be integer 0-23' })
    }
    if (cron_minute !== undefined && (!Number.isInteger(cron_minute) || cron_minute < 0 || cron_minute > 59)) {
      return res.status(400).json({ error: 'cron_minute must be integer 0-59' })
    }

    upsertPushConfig({
      webhook_url: webhook_url || '',
      cron_hour: cron_hour != null ? cron_hour : 9,
      cron_minute: cron_minute != null ? cron_minute : 0,
      enabled: !!enabled,
    })

    rescheduleFeishuPush()
    res.json({ ok: true })
  } catch (err) {
    res.status(500).json({ error: err.message })
  }
})

app.post('/api/push/test', async (req, res) => {
  try {
    const config = getPushConfig()
    if (!config.webhook_url) {
      return res.status(400).json({ error: 'webhook URL not configured' })
    }

    const result = await executeObservationPush({
      webhookUrl: config.webhook_url,
      refreshTodayObservations,
      buildTodayObservationNotification,
    })

    const now = new Date().toISOString()
    updatePushResult(now, result.sent ? `success (${result.count} products)` : result.reason)

    res.json(result)
  } catch (err) {
    const now = new Date().toISOString()
    updatePushResult(now, `error: ${err.message}`)
    res.status(500).json({ error: err.message })
  }
})
```

- [ ] **Step 3: Replace static observation email cron with dynamic Feishu push cron**

Replace the static cron block at lines 1689-1695 with this dynamic version. Keep the existing price/poster/email cron lines and add the Feishu push scheduling:

```js
cron.schedule('30 11 * * 1-5', scheduledPriceUpdate, { timezone: CRON_TIMEZONE })
cron.schedule('0 15 * * 1-5', scheduledPriceUpdate, { timezone: CRON_TIMEZONE })
cron.schedule('30 15 * * 1-5', scheduledPriceUpdate, { timezone: CRON_TIMEZONE })
cron.schedule('5 15 * * 1-5', generateAutoPosters, { timezone: CRON_TIMEZONE })
cron.schedule('0 10 * * *', scheduledObservationEmail, { timezone: CRON_TIMEZONE })
cron.schedule('10 15 * * *', scheduledObservationEmail, { timezone: CRON_TIMEZONE })

// Dynamic Feishu push task
let feishuPushTask = null

async function feishuPushCronHandler() {
  try {
    const config = getPushConfig()
    if (!config.webhook_url) {
      console.log('[飞书推送] 未配置 webhook URL，跳过')
      return
    }

    const result = await executeObservationPush({
      webhookUrl: config.webhook_url,
      refreshTodayObservations,
      buildTodayObservationNotification,
    })

    const now = new Date().toISOString()
    if (result.sent) {
      console.log(`[飞书推送] 已发送今日观察提醒: ${result.count} 个产品`)
      updatePushResult(now, `success (${result.count} products)`)
    } else {
      console.log(`[飞书推送] 未发送: ${result.reason}`)
      updatePushResult(now, result.reason)
    }
  } catch (err) {
    const now = new Date().toISOString()
    console.error('[飞书推送] 失败:', err.message)
    updatePushResult(now, `error: ${err.message}`)
  }
}

function rescheduleFeishuPush() {
  if (feishuPushTask) {
    feishuPushTask.stop()
    feishuPushTask = null
  }

  const config = getPushConfig()
  if (!config.enabled || !config.webhook_url) {
    console.log(`[飞书推送] 定时任务未启用 (${CRON_TIMEZONE})`)
    return
  }

  const cronExpr = `${config.cron_minute} ${config.cron_hour} * * *`
  feishuPushTask = cron.schedule(cronExpr, feishuPushCronHandler, { timezone: CRON_TIMEZONE })
  console.log(`[飞书推送] 定时任务已注册: 每日 ${String(config.cron_hour).padStart(2, '0')}:${String(config.cron_minute).padStart(2, '0')} (${CRON_TIMEZONE})`)
}

rescheduleFeishuPush()
console.log(`定时任务已注册: 工作日 11:30, 15:00, 15:30 更新价格；15:05 自动生成喜报；每日 10:00, 15:10 邮件提醒 (${CRON_TIMEZONE})`)
```

- [ ] **Step 4: Verify the server starts without errors**

Run:
```bash
cd backend && npm run dev
```

Expected: Server starts, logs show cron tasks registered including `[飞书推送] 定时任务未启用`. Stop the server.

- [ ] **Step 5: Commit**

```bash
git add backend/index.js
git commit -m "feat: add push-config API endpoints and dynamic Feishu push cron task"
```

---

### Task 4: Frontend Settings Page

**Files:**
- Create: `frontend/views/PushSettings.vue`

- [ ] **Step 1: CreatePushSettings.vue**

Create `frontend/views/PushSettings.vue`:

```vue
<template>
  <WorkbenchLayout>
    <h1 class="text-page-title">飞书推送设置</h1>
    <p class="text-body" style="margin-bottom:24px">配置今日观察的飞书群机器人推送。设置推送时间后，系统将每天定时把今日观察内容推送到飞书群。</p>

    <PanelCard title="推送配置">
      <div class="form-row">
        <label>启用推送</label>
        <label class="toggle-label">
          <input type="checkbox" v-model="form.enabled" />
          <span>{{ form.enabled ? '已启用' : '已关闭' }}</span>
        </label>
      </div>

      <div class="form-row">
        <label>Webhook URL</label>
        <input
          v-model="form.webhook_url"
          type="text"
          class="input"
          placeholder="https://open.feishu.cn/open-apis/bot/v2/hook/xxx"
        />
        <span class="text-label" style="margin-top:4px;display:block">
          在飞书群 → 设置 → 群机器人 → 添加自定义机器人，复制 Webhook 地址
        </span>
      </div>

      <div class="form-row">
        <label>推送时间</label>
        <div class="time-picker">
          <select v-model.number="form.cron_hour" class="input time-select">
            <option v-for="h in 24" :key="h - 1" :value="h - 1">{{ String(h - 1).padStart(2, '0') }} 时</option>
          </select>
          <select v-model.number="form.cron_minute" class="input time-select">
            <option v-for="m in minuteOptions" :key="m" :value="m">{{ String(m).padStart(2, '0') }} 分</option>
          </select>
        </div>
      </div>

      <div class="form-row" style="display:flex;gap:12px;flex-wrap:wrap">
        <button class="btn btn-primary" :disabled="saving" @click="saveConfig">
          {{ saving ? '保存中...' : '保存设置' }}
        </button>
        <button class="btn btn-secondary" :disabled="testing" @click="testPush">
          {{ testing ? '发送中...' : '发送测试消息' }}
        </button>
      </div>

      <p v-if="message" class="text-label" :class="{ 'error-msg': isError, 'success-msg': !isError }" style="margin-top:12px">
        {{ message }}
      </p>
    </PanelCard>

    <PanelCard title="推送状态" style="margin-top:20px">
      <div class="form-row">
        <label>上次推送时间</label>
        <span class="text-body">{{ lastPushTime || '从未推送' }}</span>
      </div>
      <div class="form-row">
        <label>上次推送结果</label>
        <span class="text-body">{{ lastPushResult || '--' }}</span>
      </div>
    </PanelCard>
  </WorkbenchLayout>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import WorkbenchLayout from '../components/WorkbenchLayout.vue'
import PanelCard from '../components/PanelCard.vue'

const form = ref({
  webhook_url: '',
  cron_hour: 9,
  cron_minute: 0,
  enabled: false,
})

const saving = ref(false)
const testing = ref(false)
const message = ref('')
const isError = ref(false)
const lastPushTime = ref('')
const lastPushResult = ref('')

const minuteOptions = [0, 5, 10, 15, 20, 25, 30, 35, 40, 45, 50, 55]

async function loadConfig() {
  try {
    const res = await fetch('/api/push-config')
    const data = await res.json()
    form.value = {
      webhook_url: data.webhook_url || '',
      cron_hour: data.cron_hour ?? 9,
      cron_minute: data.cron_minute ?? 0,
      enabled: !!data.enabled,
    }
    lastPushTime.value = data.last_push_time || ''
    lastPushResult.value = data.last_push_result || ''
  } catch (err) {
    message.value = '加载配置失败: ' + err.message
    isError.value = true
  }
}

async function saveConfig() {
  saving.value = true
  message.value = ''
  try {
    const res = await fetch('/api/push-config', {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(form.value),
    })
    const data = await res.json()
    if (!res.ok) throw new Error(data.error || '保存失败')
    message.value = '保存成功'
    isError.value = false
  } catch (err) {
    message.value = '保存失败: ' + err.message
    isError.value = true
  } finally {
    saving.value = false
  }
}

async function testPush() {
  testing.value = true
  message.value = ''
  try {
    const res = await fetch('/api/push/test', { method: 'POST' })
    const data = await res.json()
    if (!res.ok) throw new Error(data.error || '推送失败')
    if (data.sent) {
      message.value = `测试推送成功，已发送 ${data.count} 个产品`
      isError.value = false
    } else {
      message.value = `未发送: ${data.reason}`
      isError.value = false
    }
    await loadConfig()
  } catch (err) {
    message.value = '推送失败: ' + err.message
    isError.value = true
  } finally {
    testing.value = false
  }
}

onMounted(loadConfig)
</script>

<style scoped>
.toggle-label {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
  font-size: 14px;
}

.time-picker {
  display: flex;
  gap: 8px;
}

.time-select {
  width: 120px;
}

.error-msg {
  color: #e54d2e;
}

.success-msg {
  color: #2e8b57;
}
</style>
```

- [ ] **Step 2: Verify the page renders**

Run the frontend dev server and navigate to `http://localhost:5173/push-settings`. The page should render (but API calls will 404 if backend is not running).

Expected: Page renders with form fields for webhook URL, hour/minute selects, and buttons.

- [ ] **Step 3: Commit**

```bash
git add frontend/views/PushSettings.vue
git commit -m "feat: add PushSettings.vue page for configuring Feishu push"
```

---

### Task 5: Router + Sidebar Nav

**Files:**
- Modify: `frontend/router/index.js:9-11`
- Modify: `frontend/components/SidebarNav.vue:61-89`

- [ ] **Step 1: Add the route**

In `frontend/router/index.js`, add a new route after the existing routes (before the closing `]`):

```js
  { path: '/push-settings', component: () => import('../views/PushSettings.vue') },
```

- [ ] **Step 2: Add nav item to sidebar**

In `frontend/components/SidebarNav.vue`, import `Send` from `@lucide/vue` (add it to the existing import on line 61-71):

```js
  Send,
```

Then add a new entry to the `navItems` array (line 83-89):

```js
  { path: '/push-settings', title: '推送设置', icon: Send },
```

- [ ] **Step 3: Verify navigation works**

Run the frontend dev server. Navigate to `http://localhost:5173`. The sidebar should show a "推送设置" nav item. Clicking it should navigate to the PushSettings page.

- [ ] **Step 4: Commit**

```bash
git add frontend/router/index.js frontend/components/SidebarNav.vue
git commit -m "feat: add push-settings route and sidebar nav entry"
```

---

### Task 6: Integration Verify

- [ ] **Step 1: Start both backend and frontend**

Terminal 1:
```bash
cd backend && npm run dev
```

Terminal 2:
```bash
cd frontend && npm run dev
```

- [ ] **Step 2: Verify GET /api/push-config**

Run:
```bash
curl -s http://localhost:3001/api/push-config
```

Expected: Returns JSON with default values (empty webhook_url, cron_hour 9, cron_minute 0, enabled false).

- [ ] **Step 3: Verify PUT /api/push-config via browser**

1. Open `http://localhost:5173/push-settings`
2. Enter a test webhook URL: `https://open.feishu.cn/open-apis/bot/v2/hook/test123`
3. Set hour to 9, minute to 0
4. Check "启用推送"
5. Click "保存设置"

Expected: "保存成功" message appears. Server log shows `[飞书推送] 定时任务已注册: 每日 09:00 (Asia/Shanghai)`.

- [ ] **Step 4: Verify config persists after backend restart**

Stop the backend (Ctrl+C), restart it:
```bash
cd backend && npm run dev
```

Run:
```bash
curl -s http://localhost:3001/api/push-config
```

Expected: Returns the saved config (webhook URL, hour 9, minute 0, enabled true).

- [ ] **Step 5: Verify test push returns expected error (webhook is fake)**

In the browser, click "发送测试消息" with the fake webhook URL.

Expected: The push fails with an error message (because the webhook URL is fake), but the API call completes and the error is displayed. `last_push_result` in the config shows the error.

- [ ] **Step 6: Final commit**

```bash
git add -A
git commit -m "feat: Feishu push notification for today observation - complete"
```
