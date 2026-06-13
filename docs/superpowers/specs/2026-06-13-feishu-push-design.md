# Feishu Push for Today's Observation

## Overview

Add configurable Feishu group bot (Webhook) push notifications for the "Today Observation" (today observation) feature. Users can set the push time and Webhook URL through a frontend settings page. The backend uses the existing Node.js Express service with `node-cron` for scheduling.

## Context

The project already has:

- Node.js Express backend (`backend/index.js`) on port 3001
- `node-cron` installed and running scheduled tasks (price updates, email notifications, poster generation)
- `observationNotificationService.js` that builds text/HTML email notifications for today's observation products
- Frontend Vue 3 (no TypeScript) with sidebar navigation, `WorkbenchLayout` wrapper, and `PanelCard` component
- SQLite database via `sql.js`

## Architecture

```
Vue Frontend (PushSettings.vue)
       |  REST API (GET/PUT /api/push-config, POST /api/push/test)
       v
Node.js Backend (index.js)
  -> push_config table (SQLite)
  -> node-cron dynamic task
  -> feishuPushService.js
       |  HTTPS POST
       v
Feishu Group Bot Webhook
```

## Backend Changes

### 1. New DB Table: `push_config`

```sql
CREATE TABLE IF NOT EXISTS push_config (
  id INTEGER PRIMARY KEY CHECK (id = 1),
  webhook_url TEXT NOT NULL DEFAULT '',
  cron_hour INTEGER NOT NULL DEFAULT 9,
  cron_minute INTEGER NOT NULL DEFAULT 0,
  enabled INTEGER NOT NULL DEFAULT 0,
  last_push_time TEXT DEFAULT NULL,
  last_push_result TEXT DEFAULT NULL,
  updated_at TEXT NOT NULL DEFAULT (datetime('now'))
)
```

Single-row table (enforced by `CHECK (id = 1)`). Stores one configuration.

Added to `db.js` `initDatabase()` function. New exported functions:

- `getPushConfig()` — returns the config row or default values
- `upsertPushConfig({ webhook_url, cron_hour, cron_minute, enabled })` — inserts or updates the single row

### 2. New Service: `services/feishuPushService.js`

- `buildFeishuPushText(today, products)` — builds plain text from observation products, same format as `observationNotificationService.renderText()`
- `sendFeishuPush(webhookUrl, text)` — POST to Feishu webhook:
  ```json
  { "msg_type": "text", "content": { "text": "<content>" } }
  ```
- `executeObservationPush(webhookUrl)` — calls `refreshTodayObservations()` then `buildFeishuPushText()` then `sendFeishuPush()`

### 3. New API Endpoints (in `index.js`)

**GET /api/push-config**

Response:
```json
{
  "webhook_url": "https://open.feishu.cn/...",
  "cron_hour": 9,
  "cron_minute": 0,
  "enabled": true,
  "last_push_time": "2026-06-13T09:00:00",
  "last_push_result": "success"
}
```

**PUT /api/push-config**

Request body:
```json
{
  "webhook_url": "https://open.feishu.cn/...",
  "cron_hour": 9,
  "cron_minute": 0,
  "enabled": true
}
```

On update: stop existing Feishu cron task, re-schedule if `enabled === true`.

**POST /api/push/test**

No body required. Triggers a one-time push using the stored config. Returns `{ sent: true/false, count, error? }`.

### 4. Dynamic Cron Task

On server startup, after `initDatabase()`:

1. Read `push_config` from DB
2. If `enabled === true` and `webhook_url` is non-empty, schedule: `cron.schedule(\`${cron_minute} ${cron_hour} * * *\`, feishuPushTask, { timezone: CRON_TIMEZONE })`
3. Store the task reference so it can be stopped and re-scheduled on config update

The task function `feishuPushTask` calls `executeObservationPush(webhookUrl)`, updates `last_push_time` and `last_push_result` in DB.

## Frontend Changes

### 1. New Route

Add to `frontend/router/index.js`:

```js
{ path: '/push-settings', component: () => import('../views/PushSettings.vue') },
```

### 2. New View: `views/PushSettings.vue`

Wrapped in `<WorkbenchLayout>`. Uses `PanelCard` for sections.

**Form fields:**

- **Enable push**: checkbox toggle
- **Webhook URL**: text input with placeholder `https://open.feishu.cn/open-apis/bot/v2/hook/xxx`
- **Push hour**: select 0-23
- **Push minute**: select 0-59 (step 5: 0, 5, 10, ... 55)
- **Save button**: PUT to `/api/push-config`
- **Test push button**: POST to `/api/push/test`
- **Status display**: last push time and result

API base URL: `http://localhost:3001` (same as existing backend).

### 3. Sidebar Navigation

Add to `navItems` in `SidebarNav.vue`:

```js
{ path: '/push-settings', title: 'Push Settings', icon: Send },
```

Import `Send` from `@lucide/vue`.

## Push Content Format

Plain text, matching existing email format:

```
Today Product Dividend/Knockout Observation Alert
Observation Date: 2026-06-13
Products requiring observation today: N

Product: XXX
Flight ID: YYY
Private Fund Manager: ZZZ
Ticker Code: 001234
Entry Price: 100.00
Current Underlying Price: 98.50
Knockout Price: 95.00
Dividend Line: 80.00
Knocked Out: No
Dividend: Yes
```

## Error Handling

- If `webhook_url` is empty, push is skipped with `last_push_result = "no-webhook"`
- If push returns non-0 `code` from Feishu, store error message in `last_push_result`
- Frontend displays last push result prominently

## Environment Variables

`FEISHU_PUSH_WEBHOOK` (optional) — if set, used as default webhook URL when no DB config exists. Allows setting webhook via env for deployments.

## Security

- Webhook URL is treated as a secret. Not logged in console (only the result is logged).
- No authentication required for push-config API (matches existing backend pattern of no auth for internal APIs).
