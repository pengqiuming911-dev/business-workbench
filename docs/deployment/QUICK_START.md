# 🚀 快速部署指南

## 前置条件

1. ✅ 阿里云服务器（Ubuntu 20.04+）
2. ✅ 域名（可选，但推荐）
3. ✅ GitHub 仓库已创建
4. ✅ 服务器已配置好（参考 SERVER_SETUP.md）

---

## 📝 部署步骤

### 1️⃣ 服务器配置

```bash
# SSH 登录服务器
ssh your-user@your-server-ip

# 执行初始化脚本
bash server-setup.sh

# 配置 .env 文件
nano /var/www/business-workbench/.env
```

### 2️⃣ GitHub Secrets 配置

在 GitHub 仓库 → Settings → Secrets and variables → Actions 添加：

| Secret 名称 | 值 | 说明 |
|------------|-----|------|
| `SERVER_HOST` | `your-server-ip` | 服务器 IP |
| `SERVER_USER` | `your-user` | SSH 用户名 |
| `SSH_PRIVATE_KEY` | 私钥内容 | `~/.ssh/github_deploy_key` 文件内容 |
| `SERVER_PORT` | `22` | SSH 端口（默认 22） |

### 3️⃣ 首次部署

```bash
# 本地执行
cd D:\projects\business-workbench

# 推送到 GitHub
git add .
git commit -m "Add deployment configuration"
git push origin main
```

GitHub Actions 会自动触发部署！

### 4️⃣ 验证部署

```bash
# SSH 到服务器检查
ssh your-user@your-server-ip

# 检查 PM2 进程
pm2 status

# 检查 Nginx
sudo systemctl status nginx

# 查看日志
pm2 logs business-workbench
```

访问：`http://your-server-ip`

---

## 🔧 后续操作

### 自动部署
每次 push 到 `main` 分支，GitHub Actions 会自动部署。

### 手动重新部署
```bash
# 在 GitHub 仓库 → Actions → Deploy to Aliyun Server → Run workflow
```

### 回滚
```bash
# SSH 到服务器
ssh your-user@your-server-ip

# 查看备份
ls /var/www/business-workbench.backup.*

# 恢复备份
sudo cp -r /var/www/business-workbench.backup.20240115_120000 /var/www/business-workbench
pm2 restart business-workbench
```

---

## 🐛 常见问题

### Q: PM2 进程启动失败？
```bash
pm2 logs business-workbench --lines 100
pm2 restart business-workbench
```

### Q: Nginx 502 Bad Gateway？
```bash
# 检查后端是否在运行
pm2 status

# 检查端口
netstat -tlnp | grep 3001

# 检查 Nginx 配置
sudo nginx -t
```

### Q: 前端页面白屏？
```bash
# 检查前端构建文件
ls /var/www/business-workbench/frontend/dist/

# 检查 Nginx 配置
cat /etc/nginx/sites-enabled/business-workbench
```

### Q: API 请求 404？
```bash
# 检查后端日志
pm2 logs business-workbench

# 测试 API
curl http://localhost:3001/api/health
```

---

## 📊 监控

```bash
# 实时监控
pm2 monit

# 查看状态
pm2 status

# 查看日志
pm2 logs business-workbench --lines 50
```

---

## 🎯 下一步

- [ ] 配置 HTTPS（Certbot）
- [ ] 配置域名解析
- [ ] 设置数据库自动备份
- [ ] 配置监控告警
