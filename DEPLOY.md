# 炸金花部署指南

## 前置要求

- **Docker** >= 20.10
- **Docker Compose** >= 2.0
- （手动部署时）Go 1.21+、Node.js 20+、GCC（CGO 编译 SQLite3）

## 快速开始（Docker）

```bash
# 克隆项目
git clone <repo-url> zhajinhua
cd zhajinhua

# 测试模式启动（无需钱包）
docker compose up -d --build

# 生产模式启动（需连接钱包）
APP_MODE=pe ADMIN_PASSWORD=your_secret docker compose up -d --build
```

启动后访问 `http://localhost:8080`。

## 环境变量参考

| 变量 | 说明 | 默认值 | 可选值 |
|------|------|--------|--------|
| `APP_MODE` | 运行模式 | `te` | `te`（测试，无需钱包）/ `pe`（生产，需钱包） |
| `PORT` | 监听端口 | `8080` | 任意可用端口 |
| `ADMIN_PASSWORD` | 管理后台密码 | `admin123` | 自定义强密码 |

## Docker Compose 用法

```bash
# 构建并启动
docker compose up -d --build

# 查看日志
docker compose logs -f app

# 停止
docker compose down

# 重建（代码更新后）
docker compose up -d --build --force-recreate

# 自定义参数
APP_MODE=pe ADMIN_PASSWORD=mysecret docker compose up -d
```

Docker Compose 会自动创建名为 `zhajinhua-data` 的持久化卷，SQLite 数据库存储在容器内 `/app/data/` 目录。

## 手动部署

### 1. 构建前端

```bash
# Web 前端（TE + PE 两个版本）
cd frontend
npm install
npm run build:te    # 输出到 ../backend/static-te/
npm run build:pe    # 输出到 ../backend/static-pe/

# Telegram Mini App 前端
cd ../frontend-tg
npm install
npm run build       # 输出到 ../backend/static-tg/
```

### 2. 构建后端

```bash
cd backend
CGO_ENABLED=1 go build -o zhajinhua .
```

> 注意：后端依赖 `mattn/go-sqlite3`，需要 CGO 和 GCC。

### 3. 运行

```bash
APP_MODE=te PORT=8080 ADMIN_PASSWORD=your_secret ./zhajinhua
```

静态文件目录（`static-te/`、`static-pe/`、`static-tg/`）必须与二进制文件位于同一目录。

## Telegram Bot 配置

1. 通过 [@BotFather](https://t.me/BotFather) 创建 Bot，记录 Bot Token
2. 发送 `/newapp` 创建 Mini App
3. 设置 Web App URL：`https://your-domain.com/tg/`
4. 确保后端 `/tg/` 路径提供 `static-tg/` 中的文件
5. 可选：`/setmenubutton` 添加快速入口按钮
6. 分享链接格式：`https://t.me/<bot_username>/app?startapp=ROOM_CODE`

## 智能合约部署

游戏积分购买合约支持 EVM、TON、Solana 三条链。详细部署说明请参阅：

- [contracts/README.md](contracts/README.md) — 合约概览和各链说明
- [contracts/ton/README.md](contracts/ton/README.md) — TON 合约编译和部署详细指南

## 生产上线检查清单

- [ ] 将 `APP_MODE` 设为 `pe`
- [ ] 修改 `ADMIN_PASSWORD` 为强密码（默认 `admin123` 不安全）
- [ ] 配置 HTTPS（推荐使用 Nginx/Caddy 反向代理）
- [ ] TG Mini App 必须使用 HTTPS URL
- [ ] 部署并配置智能合约，确保钱包验证正常
- [ ] 根据 TON/ETH/SOL 当前汇率设置合约价格参数
- [ ] 检查 WebSocket 代理配置（Nginx 需 `proxy_set_header Upgrade`）
- [ ] 确认数据卷挂载正确，数据不会随容器销毁丢失

## 监控与日志

```bash
# Docker 日志
docker compose logs -f app

# 健康检查（Docker Compose 自带）
# 每 30 秒检查 http://localhost:8080/health

# 管理后台
# 访问 /admin，使用 ADMIN_PASSWORD 登录
```

## 数据库备份

SQLite 数据库文件位于：

- **Docker 部署**：`zhajinhua-data` 卷内的 `/app/data/` 目录
- **手动部署**：二进制文件同目录下的 `data/` 目录

```bash
# Docker 环境备份
docker compose exec app sqlite3 /app/data/zhajinhua.db ".backup '/app/data/backup.db'"

# 或复制卷内文件
docker cp $(docker compose ps -q app):/app/data/zhajinhua.db ./backup.db

# 手动环境备份
sqlite3 data/zhajinhua.db ".backup 'backup.db'"
```

建议定期备份数据库文件，SQLite WAL 模式下可安全在线备份。

---

# Zhajinhua Deployment Guide

## Prerequisites

- **Docker** >= 20.10
- **Docker Compose** >= 2.0
- (For manual deployment) Go 1.21+, Node.js 20+, GCC (CGO required for SQLite3)

## Quick Start (Docker)

```bash
# Clone the project
git clone <repo-url> zhajinhua
cd zhajinhua

# Start in test mode (no wallet required)
docker compose up -d --build

# Start in production mode (wallet required)
APP_MODE=pe ADMIN_PASSWORD=your_secret docker compose up -d --build
```

Visit `http://localhost:8080` after startup.

## Environment Variables Reference

| Variable | Description | Default | Options |
|----------|-------------|---------|---------|
| `APP_MODE` | Runtime mode | `te` | `te` (test, no wallet) / `pe` (production, wallet required) |
| `PORT` | Listening port | `8080` | Any available port |
| `ADMIN_PASSWORD` | Admin panel password | `admin123` | Custom strong password |

## Docker Compose Usage

```bash
# Build and start
docker compose up -d --build

# View logs
docker compose logs -f app

# Stop
docker compose down

# Rebuild (after code changes)
docker compose up -d --build --force-recreate

# Custom parameters
APP_MODE=pe ADMIN_PASSWORD=mysecret docker compose up -d
```

Docker Compose automatically creates a persistent volume named `zhajinhua-data`. The SQLite database is stored at `/app/data/` inside the container.

## Manual Deployment

### 1. Build Frontends

```bash
# Web frontend (TE + PE variants)
cd frontend
npm install
npm run build:te    # outputs to ../backend/static-te/
npm run build:pe    # outputs to ../backend/static-pe/

# Telegram Mini App frontend
cd ../frontend-tg
npm install
npm run build       # outputs to ../backend/static-tg/
```

### 2. Build Backend

```bash
cd backend
CGO_ENABLED=1 go build -o zhajinhua .
```

> Note: The backend depends on `mattn/go-sqlite3`, which requires CGO and GCC.

### 3. Run

```bash
APP_MODE=te PORT=8080 ADMIN_PASSWORD=your_secret ./zhajinhua
```

Static file directories (`static-te/`, `static-pe/`, `static-tg/`) must be in the same directory as the binary.

## Telegram Bot Setup

1. Create a bot via [@BotFather](https://t.me/BotFather) and save the Bot Token
2. Send `/newapp` to create a Mini App
3. Set the Web App URL to `https://your-domain.com/tg/`
4. Ensure the backend serves `static-tg/` files at the `/tg/` path
5. Optional: use `/setmenubutton` to add a quick-access button
6. Share link format: `https://t.me/<bot_username>/app?startapp=ROOM_CODE`

## Smart Contract Deployment

Game chip purchase contracts are available for EVM, TON, and Solana. See detailed instructions:

- [contracts/README.md](contracts/README.md) -- Contract overview and per-chain details
- [contracts/ton/README.md](contracts/ton/README.md) -- TON contract build and deploy guide

## Production Checklist

- [ ] Set `APP_MODE` to `pe`
- [ ] Change `ADMIN_PASSWORD` to a strong password (default `admin123` is insecure)
- [ ] Configure HTTPS (recommended: Nginx or Caddy as reverse proxy)
- [ ] TG Mini App requires an HTTPS URL
- [ ] Deploy and configure smart contracts; verify wallet authentication works
- [ ] Set contract price parameters based on current TON/ETH/SOL exchange rates
- [ ] Check WebSocket proxy config (Nginx needs `proxy_set_header Upgrade`)
- [ ] Confirm data volume is mounted correctly so data survives container restarts

## Monitoring and Logs

```bash
# Docker logs
docker compose logs -f app

# Health check (built into Docker Compose)
# Checks http://localhost:8080/health every 30 seconds

# Admin panel
# Visit /admin, authenticate with ADMIN_PASSWORD
```

## Database Backup

SQLite database file locations:

- **Docker deployment**: `/app/data/` inside the `zhajinhua-data` volume
- **Manual deployment**: `data/` directory next to the binary

```bash
# Docker backup
docker compose exec app sqlite3 /app/data/zhajinhua.db ".backup '/app/data/backup.db'"

# Or copy from container
docker cp $(docker compose ps -q app):/app/data/zhajinhua.db ./backup.db

# Manual backup
sqlite3 data/zhajinhua.db ".backup 'backup.db'"
```

Regular backups are recommended. SQLite WAL mode supports safe online backups.
