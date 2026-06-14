#!/bin/bash
# 服务器初始化脚本 - 阿里云 Ubuntu
# 用法: curl -sL <this-script-url> | bash

set -e

echo "🔧 Starting server setup..."

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 检查是否以 root 运行
if [ "$EUID" -eq 0 ]; then 
  echo -e "${RED}❌ Please do not run as root. Use a regular user with sudo privileges.${NC}"
  exit 1
fi

# 检测操作系统
if [ -f /etc/os-release ]; then
    . /etc/os-release
    OS=$NAME
fi

echo -e "${YELLOW}📋 Detected OS: $OS${NC}"

# Ubuntu/Debian
if [[ "$OS" == *"Ubuntu"* ]] || [[ "$OS" == *"Debian"* ]]; then
    
    echo -e "${YELLOW}📦 Updating system packages...${NC}"
    sudo apt update && sudo apt upgrade -y
    
    echo -e "${YELLOW}📦 Installing Node.js 18...${NC}"
    curl -fsSL https://deb.nodesource.com/setup_18.x | sudo -E bash -
    sudo apt install -y nodejs
    
    echo -e "${YELLOW}📦 Installing PM2...${NC}"
    sudo npm install -g pm2
    
    # 设置 PM2 开机启动
    if ! command -v pm2 &> /dev/null; then
        echo -e "${RED}❌ PM2 installation failed${NC}"
        exit 1
    fi
    
    pm2 startup
    echo -e "${GREEN}✅ To enable PM2 startup, run: sudo env PATH=\$PATH:/usr/bin pm2 startup systemd -u \$USER --hp \$HOME${NC}"
    
    echo -e "${YELLOW}📦 Installing Nginx...${NC}"
    sudo apt install -y nginx
    
    echo -e "${YELLOW}📦 Installing ufw firewall...${NC}"
    sudo apt install -y ufw
    
    # 配置防火墙
    echo -e "${YELLOW}🔒 Configuring firewall...${NC}"
    sudo ufw allow OpenSSH
    sudo ufw allow 'Nginx Full'
    sudo ufw --force enable
    
    echo -e "${GREEN}✅ Firewall enabled${NC}"
    
# CentOS/RHEL
elif [[ "$OS" == *"CentOS"* ]] || [[ "$OS" == *"Red Hat"* ]] || [[ "$OS" == *"Rocky"* ]]; then
    
    echo -e "${YELLOW}📦 Updating system packages...${NC}"
    sudo dnf update -y
    
    echo -e "${YELLOW}📦 Installing Node.js 18...${NC}"
    curl -fsSL https://rpm.nodesource.com/setup_18.x | sudo bash -
    sudo dnf install -y nodejs
    
    echo -e "${YELLOW}📦 Installing PM2...${NC}"
    sudo npm install -g pm2
    
    pm2 startup
    echo -e "${GREEN}✅ Run the startup command shown above to enable PM2 on boot${NC}"
    
    echo -e "${YELLOW}📦 Installing Nginx...${NC}"
    sudo dnf install -y nginx
    
    echo -e "${YELLOW}📦 Installing firewalld...${NC}"
    sudo dnf install -y firewalld
    sudo systemctl enable firewalld
    sudo systemctl start firewalld
    
    # 配置防火墙
    echo -e "${YELLOW}🔒 Configuring firewall...${NC}"
    sudo firewall-cmd --permanent --add-service=ssh
    sudo firewall-cmd --permanent --add-service=http
    sudo firewall-cmd --permanent --add-service=https
    sudo firewall-cmd --reload
    
    echo -e "${GREEN}✅ Firewall configured${NC}"
    
else
    echo -e "${RED}❌ Unsupported OS: $OS${NC}"
    echo "Please install Node.js, PM2, and Nginx manually."
    exit 1
fi

# 创建应用目录
echo -e "${YELLOW}📁 Creating application directories...${NC}"
sudo mkdir -p /var/www/business-workbench
sudo chown $USER:$USER /var/www/business-workbench

# 创建日志目录
sudo mkdir -p /var/log/business-workbench
sudo chown $USER:$USER /var/log/business-workbench

# 生成 SSH 密钥
echo -e "${YELLOW}🔑 Generating SSH key for GitHub Actions...${NC}"
SSH_KEY_PATH="$HOME/.ssh/github_deploy_key"

if [ ! -f "$SSH_KEY_PATH" ]; then
    ssh-keygen -t ed25519 -C "github-actions" -f "$SSH_KEY_PATH" -N ""
    cat "$SSH_KEY_PATH.pub" >> "$HOME/.ssh/authorized_keys"
    chmod 600 "$HOME/.ssh/authorized_keys"
    
    echo -e "${GREEN}✅ SSH key generated${NC}"
    echo -e "${YELLOW}⚠️  IMPORTANT: Copy this private key to GitHub Secrets:${NC}"
    echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    cat "$SSH_KEY_PATH"
    echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
else
    echo -e "${YELLOW}⚠️  SSH key already exists at $SSH_KEY_PATH${NC}"
fi

# 配置 Nginx
echo -e "${YELLOW}⚙️  Configuring Nginx...${NC}"

NGINX_CONFIG="/etc/nginx/sites-available/business-workbench"
if [ ! -d "/etc/nginx/sites-available" ]; then
    NGINX_CONFIG="/etc/nginx/conf.d/business-workbench.conf"
fi

sudo tee "$NGINX_CONFIG" > /dev/null << 'NGINX_EOF'
server {
    listen 80;
    server_name _;  # 替换为你的域名或 IP

    # 前端静态文件
    location / {
        root /var/www/business-workbench/frontend/dist;
        try_files $uri $uri/ /index.html;
        index index.html;
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
        proxy_set_header X-Forwarded-Proto $scheme;
        
        # 超时设置
        proxy_connect_timeout 60s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;
    }

    # 静态资源缓存
    location ~* \.(js|css|png|jpg|jpeg|gif|ico|svg|woff|woff2|ttf|eot)$ {
        root /var/www/business-workbench/frontend/dist;
        expires 1y;
        add_header Cache-Control "public, immutable";
        try_files $uri =404;
    }

    # Gzip 压缩
    gzip on;
    gzip_vary on;
    gzip_min_length 1024;
    gzip_types text/plain text/css text/xml text/javascript application/javascript application/xml+rss application/json;
}
NGINX_EOF

# 启用配置
if [ -d "/etc/nginx/sites-available" ]; then
    sudo ln -sf "$NGINX_CONFIG" /etc/nginx/sites-enabled/
    sudo rm -f /etc/nginx/sites-enabled/default
fi

# 测试 Nginx 配置
echo -e "${YELLOW}🧪 Testing Nginx configuration...${NC}"
if sudo nginx -t; then
    sudo systemctl restart nginx
    sudo systemctl enable nginx
    echo -e "${GREEN}✅ Nginx configured and started${NC}"
else
    echo -e "${RED}❌ Nginx configuration test failed${NC}"
    exit 1
fi

# 创建 .env 文件模板
echo -e "${YELLOW}📝 Creating .env template...${NC}"
if [ ! -f "/var/www/business-workbench/backend/.env" ]; then
    cat > /var/www/business-workbench/backend/.env << 'ENV_EOF'
# Feishu API Configuration
FEISHU_APP_ID=cli_xxxxxxxxxx
FEISHU_APP_SECRET=xxxxxxxxxxxxxxxxxx
FEISHU_REDIRECT_URI=https://your-domain.com/api/auth/callback

# Server Configuration
PORT=3001
NODE_ENV=production

# Database
DATABASE_PATH=./data.sqlite

# Security
JWT_SECRET=your-secret-key-change-this
SESSION_SECRET=your-session-secret-change-this
ENV_EOF
    echo -e "${GREEN}✅ .env template created at /var/www/business-workbench/backend/.env${NC}"
    echo -e "${YELLOW}⚠️  Please edit this file with your actual configuration${NC}"
fi

# 安装 Certbot（可选 HTTPS）
echo -e "${YELLOW}🔐 Installing Certbot for HTTPS...${NC}"
if [[ "$OS" == *"Ubuntu"* ]] || [[ "$OS" == *"Debian"* ]]; then
    sudo apt install -y certbot python3-certbot-nginx
elif [[ "$OS" == *"CentOS"* ]] || [[ "$OS" == *"Red Hat"* ]]; then
    sudo dnf install -y certbot python3-certbot-nginx
fi

echo -e "${GREEN}✅ For HTTPS, run: sudo certbot --nginx -d your-domain.com${NC}"

# 完成
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}✅ Server setup completed!${NC}"
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
echo -e "${YELLOW}Next steps:${NC}"
echo "1. Edit /var/www/business-workbench/backend/.env with your actual config"
echo "2. Copy the SSH private key shown above to GitHub Secrets"
echo "3. Push your code to GitHub"
echo "4. GitHub Actions will automatically deploy!"
echo ""
echo -e "${YELLOW}Useful commands:${NC}"
echo "  pm2 monit              # Monitor processes"
echo "  pm2 logs               # View logs"
echo "  sudo nginx -t          # Test Nginx config"
echo "  sudo certbot --nginx   # Setup HTTPS"
echo ""
