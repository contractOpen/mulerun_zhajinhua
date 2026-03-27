# 炸金花 Web 前端

Vue 3 + Vite 构建的炸金花（Zha Jinhua）Web 前端，支持登录、大厅、实时游戏桌、每日奖励和多链钱包连接。

## 技术栈

- **Vue 3.4** (Composition API, 单文件组件)
- **Vite 5** (开发服务器 + 构建工具)
- **WebSocket** 实时双向通信（代理到后端 `localhost:8080/ws`）

## 目录结构

```
frontend/
├── package.json
├── vite.config.js
├── src/
│   ├── App.vue                         # 根组件（登录 / 大厅 / 游戏桌 / 结算）
│   ├── main.js                         # 应用入口
│   └── composables/
│       ├── useGame.js                  # WebSocket 连接 & 游戏状态管理
│       ├── useI18n.js                  # 国际化（4 种语言）
│       ├── useWallet.js                # 多链钱包（MetaMask / TonConnect / Phantom）
│       └── useAudio.js                 # Web Audio API 程序化音效
```

## Composables 说明

| 模块 | 功能 |
|------|------|
| `useGame.js` | WebSocket 连接、自动重连、房间创建/加入/离开、下注/跟注/弃牌/比牌、匹配、每日奖励 |
| `useI18n.js` | 中文 (zh)、English (en)、Русский (ru)、Espanol (es) 四语切换，响应式 locale |
| `useWallet.js` | EVM (MetaMask)、TON (TonConnect)、Solana (Phantom) 钱包检测与连接 |
| `useAudio.js` | 基于 Web Audio API 的程序化音效引擎（背景音乐、操作音效） |

## 构建模式

| 模式 | 变量 `VITE_APP_MODE` | 钱包要求 | 输出目录 |
|------|-----------------------|----------|----------|
| **TE** (测试) | `te` | 无需钱包 | `../backend/static-te/` |
| **PE** (生产) | `pe` | 必须连接钱包 | `../backend/static-pe/` |

## 构建命令

```bash
npm install              # 安装依赖

# 测试环境
npm run dev              # 默认开发服务器 (TE 模式)
npm run dev:te           # 明确指定 TE 模式开发服务器
npm run build:te         # 构建 TE 版本 -> ../backend/static-te/

# 生产环境
npm run dev:pe           # PE 模式开发服务器
npm run build:pe         # 构建 PE 版本 -> ../backend/static-pe/
```

## Vercel 部署

网页前端可以直接单独部署到 Vercel。

当前线上地址：

```bash
Project: https://vercel.com/contractopens-projects/frontend
Production: https://frontend-j3a60rxac-contractopens-projects.vercel.app
Alias: https://frontend-gules-phi-47.vercel.app
```

1. 将 `frontend/` 目录导入为一个独立 Vercel Project
2. 保持 `Framework Preset = Vite`
3. `Build Command` 使用：

```bash
VITE_APP_MODE=pe npm run build
```

4. `Output Directory` 使用：

```bash
dist
```

5. 如需显式指定后端，配置环境变量：

```bash
VITE_API_BASE_URL=https://game.atdl.link
VITE_WS_URL=wss://game.atdl.link/ws
```

当前代码默认也会连接：

```bash
https://game.atdl.link
```

## 国际化 (i18n)

支持 4 种语言，登录页提供语言切换器：

- 中文 (`zh`)
- English (`en`)
- Русский (`ru`)
- Espanol (`es`)

## 功能列表

- **登录页** — 输入昵称、连接钱包、选择语言
- **大厅** — 查看/创建/加入房间、快速匹配、每日奖励领取、充值入口
- **游戏桌** — 实时发牌、看牌、下注、跟注、加注、全押、比牌、弃牌
- **结算** — 游戏结束后显示胜负与筹码变动
- **房间分享** — 通过邀请码分享房间
- **区块链头像** — 基于钱包地址生成唯一头像
- **每日奖励** — 每天可领取 3 次免费积分（每次 500，共 1500）

## 环境变量

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `VITE_APP_MODE` | 构建模式 | `te` |
| `VITE_API_BASE_URL` | 后端 HTTP 地址，例如 `https://1.2.3.4:8080` | 当前页面同源 |
| `VITE_WS_URL` | 后端 WebSocket 地址，例如 `wss://1.2.3.4:8080/ws` | 根据页面地址自动推导 |

当前仓库默认后端连接：

```bash
VITE_API_BASE_URL=https://game.atdl.link
VITE_WS_URL=wss://game.atdl.link/ws
```

运行时通过 `__APP_MODE__` 全局常量注入到前端代码中。

---

# Zha Jinhua Web Frontend

Vue 3 + Vite web frontend for the Zha Jinhua (Chinese poker) game. Supports login, lobby, real-time game table, daily bonus, and multi-chain wallet connection.

## Tech Stack

- **Vue 3.4** (Composition API, Single File Components)
- **Vite 5** (dev server + build tool)
- **WebSocket** real-time communication (proxied to backend `localhost:8080/ws`)

## Directory Structure

```
frontend/
├── package.json
├── vite.config.js
├── src/
│   ├── App.vue                         # Root component (login / lobby / game / settlement)
│   ├── main.js                         # App entry
│   └── composables/
│       ├── useGame.js                  # WebSocket connection & game state management
│       ├── useI18n.js                  # Internationalization (4 languages)
│       ├── useWallet.js                # Multi-chain wallet (MetaMask / TonConnect / Phantom)
│       └── useAudio.js                 # Web Audio API programmatic sound effects
```

## Composables

| Module | Responsibility |
|--------|----------------|
| `useGame.js` | WebSocket connection, auto-reconnect, room CRUD, betting/calling/folding/comparing, matchmaking, daily bonus |
| `useI18n.js` | Chinese (zh), English (en), Russian (ru), Spanish (es) with reactive locale switching |
| `useWallet.js` | EVM (MetaMask), TON (TonConnect), and Solana (Phantom) wallet detection and connection |
| `useAudio.js` | Programmatic sound engine built on Web Audio API (BGM, action SFX) |

## Build Modes

| Mode | `VITE_APP_MODE` | Wallet Required | Output Directory |
|------|-----------------|-----------------|------------------|
| **TE** (test) | `te` | No | `../backend/static-te/` |
| **PE** (production) | `pe` | Yes | `../backend/static-pe/` |

## Build Commands

```bash
npm install              # Install dependencies

# Test environment
npm run dev              # Default dev server (TE mode)
npm run dev:te           # Explicit TE mode dev server
npm run build:te         # Build TE -> ../backend/static-te/

# Production environment
npm run dev:pe           # PE mode dev server
npm run build:pe         # Build PE -> ../backend/static-pe/
```

## Vercel Deployment

The web frontend can be deployed to Vercel as a standalone project.

Current deployed URLs:

```bash
Project: https://vercel.com/contractopens-projects/frontend
Production: https://frontend-j3a60rxac-contractopens-projects.vercel.app
Alias: https://frontend-gules-phi-47.vercel.app
```

1. Import the `frontend/` directory as a separate Vercel project
2. Keep `Framework Preset = Vite`
3. Use this build command:

```bash
VITE_APP_MODE=pe npm run build
```

4. Use this output directory:

```bash
dist
```

5. Optional environment variables:

```bash
VITE_API_BASE_URL=https://game.atdl.link
VITE_WS_URL=wss://game.atdl.link/ws
```

The current codebase also defaults to:

```bash
https://game.atdl.link
```

## Internationalization (i18n)

Four languages supported, with a language picker on the login page:

- Chinese (`zh`)
- English (`en`)
- Russian (`ru`)
- Spanish (`es`)

## Features

- **Login Page** -- Enter nickname, connect wallet, select language
- **Lobby** -- View/create/join rooms, quick match, daily bonus, recharge
- **Game Table** -- Real-time dealing, look at cards, bet, call, raise, all-in, compare, fold
- **Settlement** -- Post-game results with chip changes
- **Room Sharing** -- Share rooms via invite code
- **Blockchain Avatars** -- Unique avatars generated from wallet addresses
- **Daily Bonus** -- Claim free chips 3 times per day (500 each, 1500 total)

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `VITE_APP_MODE` | Build mode | `te` |
| `VITE_API_BASE_URL` | Backend HTTP base URL, e.g. `https://1.2.3.4:8080` | Same-origin with current page |
| `VITE_WS_URL` | Backend WebSocket URL, e.g. `wss://1.2.3.4:8080/ws` | Derived from current page |

Current default backend connection in this repo:

```bash
VITE_API_BASE_URL=https://game.atdl.link
VITE_WS_URL=wss://game.atdl.link/ws
```

Injected at build time as the `__APP_MODE__` global constant.
