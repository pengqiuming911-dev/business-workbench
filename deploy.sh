#!/bin/bash
# 快速部署脚本（本地手动部署用）

set -e

echo "🚀 Starting deployment..."

# 1. 构建前端
echo "📦 Building frontend..."
cd frontend
npm run build
cd ..

# 2. 同步到服务器
echo "📤 Syncing files to server..."
SERVER_USER="your-user"
SERVER_HOST="your-server-ip"
SERVER_DIR="/var/www/business-workbench"

# 同步前端
rsync -avz --delete frontend/dist/ ${SERVER_USER}@${SERVER_HOST}:${SERVER_DIR}/frontend/dist/

# 同步后端（排除 node_modules 和数据库）
rsync -avz --delete \
  --exclude='node_modules' \
  --exclude='data.sqlite' \
  --exclude='.env' \
  backend/ ${SERVER_USER}@${SERVER_HOST}:${SERVER_DIR}/backend/

# 3. 远程安装依赖并重启
echo "🔄 Restarting backend..."
ssh ${SERVER_USER}@${SERVER_HOST} << 'ENDSSH'
  cd /var/www/business-workbench/backend
  npm ci --only=production
  pm2 restart business-workbench || pm2 start npm --name "business-workbench" -- start
  pm2 save
ENDSSH

echo "✅ Deployment completed!"
echo "🌐 Visit: http://your-server-ip"
