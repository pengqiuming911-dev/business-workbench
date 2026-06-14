# 阿里云部署完整方案

## ✅ 已完成的配置

### 1. GitHub Actions 自动部署
- ✅ `.github/workflows/deploy.yml` - 自动部署工作流
- ✅ Go 后端编译为单二进制，systemd 管理进程
- ✅ 后端 health check 端点 `/api/health`

### 2. 服务器初始化
- ✅ `server-setup.sh` - 一键服务器配置脚本
- ✅ 支持 Ubuntu/Debian 和 CentOS/RHEL
- ✅ 自动安装 Nginx
- ✅ 自动生成 SSH 密钥用于 GitHub Actions

### 3. Nginx 配置
- ✅ 前端静态文件服务
- ✅ API 反向代理（`/api/*` → `localhost:3001`）
- ✅ 静态资源缓存 1 年
- ✅ Gzip 压缩
- ✅ HTTPS 支持（Certbot）

### 4. 安全配置
- ✅ `.gitignore` 已更新，排除截图和配置文件
- ✅ 环境变量模板已创建
- ✅ 防火墙配置（UFW/Firewalld）

---

## 🚀 部署步骤

### 步骤 1：购买阿里云服务器

推荐配置：
- **操作系统**: Ubuntu 22.04 LTS
- **配置**: 2核4G（最低 1核2G 也可运行）
- **带宽**: 1-5Mbps（根据访问量）
- **系统盘**: 40GB SSD

### 步骤 2：初始化服务器

```bash
ssh root@your-server-ip

adduser deploy
usermod -aG sudo deploy

su - deploy

curl -o server-setup.sh https://raw.githubusercontent.com/pengqiuming911-dev/business-workbench/main/server-setup.sh
chmod +x server-setup.sh
./server-setup.sh
```

### 步骤 3：配置环境变量

环境变量由 GitHub Secrets 注入，通过 CI 写入 `/var/www/business-workbench/backend-go/.env`。

### 步骤 4：配置 GitHub Secrets

在 GitHub 仓库 → Settings → Secrets and variables → Actions，添加以下 Secrets：

| Secret 名称 | 值 | 获取方式 |
|------------|-----|---------|
| `SERVER_HOST` | `your-server-ip` | 服务器公网 IP |
| `SERVER_USER` | `deploy` | SSH 用户名 |
| `SSH_PRIVATE_KEY` | 私钥内容 | 服务器上运行 `cat ~/.ssh/github_deploy_key` |
| `SERVER_PORT` | `22` | SSH 端口（默认 22） |
| `FEISHU_APP_ID` | 飞书 App ID | 开发者后台 |
| `FEISHU_APP_SECRET` | 飞书 App Secret | 开发者后台 |
| `FEISHU_REDIRECT_URI` | OAuth 回调地址 | |
| `FRONTEND_URL` | 前端地址 | |
| `DEEPSEEK_API_KEY` | DeepSeek API Key | |
| `DEEPSEEK_API_URL` | DeepSeek API URL | |
| `DEEPSEEK_MODEL` | DeepSeek 模型名称 | |
| `SMTP_HOST` | SMTP 服务器 | |
| `SMTP_USER` | SMTP 用户 | |
| `SMTP_PASS` | SMTP 密码 | |

### 步骤 5：配置域名（可选但推荐）

1. 在阿里云购买域名
2. 添加 A 记录：`@` → `你的服务器IP`
3. 等待 DNS 生效（通常 10 分钟）

**配置 HTTPS：**
```bash
ssh deploy@your-server-ip

sudo certbot --nginx -d your-domain.com
```

### 步骤 6：更新 Nginx 配置（如果使用域名）

```bash
sudo nano /etc/nginx/sites-available/business-workbench
```

修改 `server_name`：
```nginx
server {
    listen 80;
    server_name your-domain.com;
    
    # ... 其他配置保持不变
}
```

重启 Nginx：
```bash
sudo nginx -t
sudo systemctl restart nginx
```

### 步骤 7：提交代码并部署

```bash
cd D:\projects\business-workbench

git status
git add .
git commit -m "feat: 优化 UI 和排版，添加自动部署配置"
git push origin main
```

推送后 GitHub Actions 会自动：
1. ✅ Checkout 代码
2. ✅ 构建前端
3. ✅ 编译 Go 后端（linux/amd64）
4. ✅ SSH 上传到服务器
5. ✅ 重启 systemd 服务
6. ✅ 部署完成

---

## 📊 验证部署

### 检查 GitHub Actions

访问：https://github.com/pengqiuming911-dev/business-workbench/actions

### 检查服务器

```bash
ssh deploy@your-server-ip

sudo systemctl status business-workbench

journalctl -u business-workbench -n 50

sudo systemctl status nginx

curl http://localhost:3001/api/health
```

### 访问网站

- **本地访问**: http://your-server-ip
- **域名访问**: http://your-domain.com（如果配置了域名）
- **HTTPS**: https://your-domain.com（如果配置了 SSL）

---

## 🔧 常用运维命令

### systemd 进程管理

```bash
sudo systemctl status business-workbench
sudo systemctl restart business-workbench
sudo systemctl stop business-workbench

journalctl -u business-workbench -f
journalctl -u business-workbench --since today
```

### Nginx 管理

```bash
sudo nginx -t
sudo systemctl restart nginx
sudo systemctl reload nginx
sudo systemctl status nginx
sudo tail -f /var/log/nginx/error.log
```

### 数据库管理

```bash
sqlite3 /var/www/business-workbench/backend-go/data.sqlite .dump > backup.sql
sqlite3 /var/www/business-workbench/backend-go/data.sqlite < backup.sql
sqlite3 /var/www/business-workbench/backend-go/data.sqlite "VACUUM;"
```

### 手动部署

```bash
ssh deploy@your-server-ip

sudo systemctl stop business-workbench

cd /var/www/business-workbench
# 替换 server 二进制（本地交叉编译后 scp）

sudo systemctl start business-workbench
```

---

## 🐛 故障排查

### 问题 1：部署失败

**检查 GitHub Actions 日志：**
- 访问 https://github.com/pengqiuming911-dev/business-workbench/actions
- 点击失败的 workflow
- 查看错误日志

### 问题 2：Go 后端启动失败

```bash
journalctl -u business-workbench -n 100

curl http://localhost:3001/api/health

cat /var/www/business-workbench/backend-go/.env
```

### 问题 3：Nginx 502 Bad Gateway

```bash
sudo systemctl status business-workbench

curl http://localhost:3001/api/health

sudo nginx -t
cat /var/log/nginx/error.log
```

### 问题 4：前端白屏

```bash
ls -la /var/www/business-workbench/frontend/dist/

cat /etc/nginx/sites-available/business-workbench

# 检查浏览器控制台错误
```

---

## 🔄 回滚

```bash
ssh deploy@your-server-ip

ls /var/www/business-workbench.backup.*

sudo cp -r /var/www/business-workbench.backup.20240115_120000 /var/www/business-workbench
sudo systemctl restart business-workbench
```

---

## ✅ 部署检查清单

- [ ] 阿里云服务器已购买
- [ ] 服务器已初始化（运行 `server-setup.sh`）
- [ ] GitHub Secrets 已添加
- [ ] 域名已配置（可选）
- [ ] HTTPS 已启用（可选）
- [ ] 代码已推送到 GitHub
- [ ] GitHub Actions 部署成功
- [ ] 访问 `http://your-server-ip/api/health` 正常
- [ ] 前端页面访问正常
- [ ] 飞书登录功能正常

---

## 📞 需要帮助？

如果遇到任何问题，请提供：
1. 服务器操作系统版本
2. GitHub Actions 错误日志
3. systemd 日志: `journalctl -u business-workbench -n 100`
4. Nginx 错误日志: `sudo tail -100 /var/log/nginx/error.log`
