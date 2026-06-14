# business-workbench Go backend

This directory is the Go + Gin backend for business-workbench.

## Run

```powershell
$env:PATH = "D:\projects\business-workbench\.local-tools\go\bin;$env:PATH"
$env:GOPROXY = "https://goproxy.cn,direct"
go run ./cmd/server
```

The service reads `.env` in its own directory and uses `data.sqlite` by default (both in `backend-go/`).

## Implemented API slice

- `GET /api/health`
- `GET /api/auth/status`
- `GET /api/auth/login`
- `GET /api/auth/callback`
- `DELETE /api/auth/logout`
- `GET /api/db/sync-status`
- `GET /api/db/sync-coinvest-status`
- `GET /api/db/products`
- `GET /api/db/industries`
- `GET /api/db/user-profiles`
- `POST /api/db/sync`
- `POST /api/db/sync-coinvest`
- `GET /api/drive/shared-with-me`
- `GET /api/drive/shared-spaces`
- `GET /api/drive/shared-files`
- `GET /api/drive/files`
- `GET /api/drive/download`
- `GET /api/drive/sheet-data`
- `GET /api/drive/doc-content`
- `GET /api/drive/export-sheet`
- `GET /api/drive/product-docs`
- `GET /api/drive/product-docs/sync-status`
- `POST /api/drive/sync-product-docs`
- `GET /api/dashboard/stats`
- `GET /api/dashboard/charts`
- `GET /api/observations/calendar`
- `GET /api/observations/products`
- `GET /api/observations`
- `GET /api/observations/today`
- `POST /api/observations/generate`
- `POST /api/observations/refresh-prices`
- `GET /api/posters/today`
- `GET /api/posters`
- `POST /api/posters/generate`
- `GET /api/push-config`
- `PUT /api/push-config`
- `POST /api/push/test`
- `GET /api/activity-logs`
- `GET /api/search`
- `GET /api/agent/conversations`
- `POST /api/agent/conversations`
- `DELETE /api/agent/conversations/:id`
- `GET /api/agent/conversations/:id/messages`
- `POST /api/agent/chat` (streaming chat with local tools for products, observations, customers, transactions, product analytics, posters, product docs, channels, sync status, and activity logs)

## Scheduled tasks (cron)

The service runs the following scheduled tasks via `robfig/cron/v3` (timezone: `CRON_TIMEZONE`, default `Asia/Shanghai`):

- Weekdays 11:30, 15:00, 15:30 — auto-update underlying prices and today's observation records
- Weekdays 15:05 — auto-generate knockout/dividend celebration posters
- Daily 10:00, 15:10 — send email notifications for today's observation products (SMTP optional)
- Every minute — dynamic Feishu webhook push (configured via `/api/push-config`)

## Trading calendar

Holiday adjustment for observation dates uses `internal/trading` with 2025–2026 Chinese national holidays. The `holiday_adjust` field on each product (`"提前"` / `"postpone"`) controls whether dates are advanced or postponed to the nearest trading day.

## RAG document retrieval

The Agent chat endpoint (`/api/agent/chat`) automatically searches `product_docs` using keyword scoring and injects the top 5 results into the LLM system prompt for context-augmented responses.

## Remaining work

- Agent write/side-effect tools, if needed after the read-only tool migration
