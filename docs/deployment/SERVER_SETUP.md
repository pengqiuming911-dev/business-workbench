# 阿里云服务器环境配置

## 1. 系统要求
- 操作系统: Ubuntu 20.04+ 或 CentOS 8+
- 最低配置: 2核4G（推荐）
- Node.js: 18+ LTS
- Nginx
- PM2 (进程管理)

## 2. 服务器初始化脚本

```bash
#!/bin/bash
# 保存为 server-setup.sh 并执行

# 更新系统
sudo apt update && sudo apt upgrade -y

# 安装 Node.js 18
curl -fsSL https://deb.nodesource.com/setup_18.x | sudo -E bash -
sudo apt install -y nodejs

# 安装 PM2
sudo npm install -g pm2

# 安装 Nginx
sudo apt install -y nginx

# 创建应用目录
sudo mkdir -p /var/www/business-workbench
sudo chown $USER:$USER /var/www/business-workbench

# 创建日志目录
sudo mkdir -p /var/log/business-workbench
sudo chown $USER:$USER /var/log/business-workbench
```

## 3. 配置 SSH 密钥（GitHub Actions 用）

```bash
# 生成 SSH 密钥对
ssh-keygen -t ed25519 -C "github-actions" -f ~/.ssh/github_deploy_key
cat ~/.ssh/github_deploy_key.pub >> ~/.ssh/authorized_keys

# 记录私钥内容（稍后添加到 GitHub Secrets）
cat ~/.ssh/github_deploy_key
```

## 4. 环境变量配置

创建 `/var/www/business-workbench/.env`：

```bash
# 飞书 API 配置
FEISHU_APP_ID=your_feishu_app_id
FEISHU_APP_SECRET=your_feishu_app_secret
FEISHU_REDIRECT_URI=https://your-domain.com/api/auth/callback

# 服务器配置
PORT=3001
NODE_ENV=production

# 数据库配置
DATABASE_PATH=./data.sqlite
```

## 5. Nginx 配置

创建 `/etc/nginx/sites-available/business-workbench`：

```nginx
server {
    listen 80;
    server_name your-domain.com;  # 替换为你的域名或 IP

    # 前端静态文件
    location / {
        root /var/www/business-workbench/frontend/dist;
        try_files $uri $uri/ /index.html;
    }

    # API 反向代理
    location /api/ {
        proxy_pass http://localhost:3001;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_cache_bypass $http_upgrade;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }

    # 静态资源缓存
    location ~* \.(js|css|png|jpg|jpeg|gif|ico|svg)$ {
        root /var/www/business-workbench/frontend/dist;
        expires 1y;
        add_header Cache-Control "public, immutable";
    }
}
```

启用配置：

```bash
sudo ln -s /etc/nginx/sites-available/business-workbench /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl restart nginx
```

## 6. HTTPS 配置（可选）

```bash
# 安装 Certbot
sudo apt install -y certbot python3-certbot-nginx

# 申请证书
sudo certbot --nginx -d your-domain.com
```

## 7. 防火墙配置

```bash
sudo ufw allow 22    # SSH
sudo ufw allow 80    # HTTP
sudo ufw allow 443   # HTTPS
sudo ufw enable
```
