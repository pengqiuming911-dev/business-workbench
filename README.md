# 业务工作台

Vue 3 + TypeScript 前端，Go + Gin 后端，SQLite 本地数据存储。

## 技术栈

- 前端：Vue 3, Vite, TypeScript
- 后端：Go, Gin, SQLite
- 集成：飞书 OAuth/Drive/Sheets/Bitable，东方财富行情，阿里云百炼兼容 OpenAI Chat Completions

## 本地开发

```powershell
npm install
npm run dev
```

默认会启动：

- Go 后端：`http://localhost:3001`
- Vite 前端：`http://localhost:5173`

## 环境变量

Go 后端读取 `backend-go/.env`。

```env
PORT=3001
FRONTEND_URL=http://localhost:5173
DATABASE_PATH=data.sqlite

FEISHU_APP_ID=cli_xxxxxxxxxx
FEISHU_APP_SECRET=xxxxxxxxxx
FEISHU_REDIRECT_URI=http://localhost:3001/api/auth/callback
FEISHU_PUSH_WEBHOOK=https://open.feishu.cn/open-apis/bot/v2/hook/xxx

DEEPSEEK_API_KEY=sk-xxx
DEEPSEEK_API_URL=https://dashscope.aliyuncs.com/compatible-mode/v1
DEEPSEEK_MODEL=deepseek-v3
CRON_TIMEZONE=Asia/Shanghai
```

## 常用命令

```powershell
npm run typecheck
npm run build

cd backend-go
go test ./...
go run ./cmd/server
```

## 项目结构

```text
business-workbench/
├── frontend/       # Vue 3 + TypeScript 前端
├── backend-go/     # Go + Gin 后端
├── docs/           # 部署和迁移文档
└── package.json    # 根开发脚本
```
