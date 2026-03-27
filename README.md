# 骡子炸金花 / Mule Zha Jin Hua   现在这个世界应该是骡子的了

> Web3 多链炸金花卡牌游戏 | A Web3 Multi-Chain Card Game by MuleRun

---

## 中文

### 项目简介

骡子炸金花 (Mule Zha Jin Hua) 是由 MuleRun 开发的 Web3 卡牌游戏。玩家可以通过网页或 Telegram Mini App 即时进入游戏，使用多链钱包购买筹码，在区块链透明机制下体验经典炸金花玩法。

**核心特性：**
- 经典炸金花规则，支持明牌/暗牌对战
- WebSocket 实时多人对局
- 多链钱包集成 (EVM / TON / Solana)
- 链上筹码购买，智能合约保障资金安全
- Telegram Mini App，即开即玩
- Bot AI 陪练，新手友好
- 每日签到奖励，持续活跃激励
- 四语言国际化 (中/英/俄/西)

### 营销计划

#### 目标用户

| 群体 | 描述 |
|------|------|
| Crypto / Web3 用户 | 已有链上钱包，熟悉链上交互 |
| Telegram 游戏社区 | 活跃于 Telegram Mini App 生态的玩家 |
| 休闲棋牌玩家 | 喜欢炸金花等经典卡牌游戏的用户 |

#### 用户获取

- **Telegram Mini App 裂变传播** -- 玩家一键分享游戏至群组和好友，低门槛拉新
- **房间邀请链接** -- 每个游戏房间生成专属链接，朋友点击即可加入对局
- **每日签到奖励** -- 每天 3 次免费领取，每次 500 积分，留存核心手段

#### 盈利模式

- **平台抽水** -- 每局游戏结束后按 `底注 x 参与人数` 收取平台费用
- **链上筹码购买** -- 玩家通过智能合约使用原生代币 (ETH/TON/SOL) 购买游戏积分，1 美元等值代币 = 1000 积分

#### 增长策略

- **多链支持** -- 同时覆盖 EVM、TON、Solana 三大生态，最大化用户触达
- **国际化** -- 支持中文、英文、俄文、西班牙文四种语言
- **Bot AI 引导** -- AI 机器人陪练降低新手门槛，提升首次体验

#### 社区建设

- **Telegram 群组运营** -- 建立官方玩家群，实时互动
- **房间分享机制** -- 鼓励玩家邀请好友组局
- **推荐奖励系统** -- (规划中) 邀请好友获得积分奖励

#### 竞争优势

| 优势 | 说明 |
|------|------|
| 即开即玩 | 无需下载，网页/Telegram 直接进入 |
| 区块链透明 | 筹码购买链上可查，公开透明 |
| 跨链支持 | 三大公链生态全覆盖 |

### MuleRun API 简介

本项目由 [MuleRun](https://mulerun.com) 平台开发工作流驱动构建。

- **MuleRun** 提供 AI 驱动的开发工具，大幅提升开发效率
- **API 能力**：代码生成、项目脚手架搭建、多语言支持
- 骡子炸金花从后端到前端到智能合约，均使用 MuleRun 开发流程完成
- 更多信息请访问 [mulerun.com](https://mulerun.com)

### 骡子 AI Bot 路线图

下一阶段目标：打造 **骡子AI Bot** -- 一个 AI 驱动的游戏助手。

| 功能 | 描述 |
|------|------|
| 智能教练 | 根据牌局实时分析，给出策略建议 |
| 手牌分析 | 评估当前手牌强度与胜率 |
| 策略建议 | 基于对手行为模式提供下注/弃牌建议 |
| Telegram Bot 集成 | 在 Telegram 中直接与 AI 助手对话 |
| 多语言交互 | 支持中英俄西四种语言自然语言对话 |
| 资金管理建议 | 智能推荐合理的下注额度与止损策略 |

### 项目结构

```
zhajinhua/
├── backend/         Go 后端服务
├── frontend/        Vue 3 网页前端 (TE/PE 模式)
├── frontend-tg/     Telegram Mini App
└── contracts/       智能合约 (EVM / TON / SOL)
```

详细文档：
- [`backend/README.md`](backend/README.md) -- 后端服务文档
- [`frontend/README.md`](frontend/README.md) -- 网页前端文档
- [`frontend-tg/README.md`](frontend-tg/README.md) -- Telegram Mini App 文档
- [`contracts/README.md`](contracts/README.md) -- 智能合约文档

### 快速开始

```bash
# 后端
cd backend
go build -o server .
APP_MODE=te ./server

# 网页前端 (测试环境)
cd frontend
npm install && npm run build:te

# Telegram Mini App
cd frontend-tg
npm install && npm run build
```

### 前端部署到 Vercel

当前推荐将两个前端作为独立项目部署到 Vercel：

- `frontend/`：网页前端
- `frontend-tg/`：Telegram Mini App

当前已部署的网页前端地址：

```bash
Project: https://vercel.com/contractopens-projects/frontend
Production: https://frontend-j3a60rxac-contractopens-projects.vercel.app
Alias: https://frontend-gules-phi-47.vercel.app
```

当前已部署的 TG 前端地址：

```bash
Project: https://vercel.com/contractopens-projects/mule-zjh-tg
Production: https://mule-zjh-2hrrjvxlf-contractopens-projects.vercel.app
Alias: https://mule-zjh-tg.vercel.app
```

后端统一连接：

```bash
https://game.atdl.link
```

推荐在两个 Vercel Project 中都配置：

```bash
VITE_API_BASE_URL=https://game.atdl.link
VITE_WS_URL=wss://game.atdl.link/ws
```

说明：

- `frontend/` 生产构建命令：`VITE_APP_MODE=pe npm run build`
- `frontend-tg/` 构建命令：`npm run build`
- 两者输出目录均为：`dist`

### 许可证

TBD

---

## English

### Project Introduction

Mule Zha Jin Hua is a Web3 card game developed by MuleRun. Players can instantly join games through a web browser or Telegram Mini App, purchase chips via multi-chain wallets, and enjoy the classic Zha Jin Hua experience with blockchain transparency.

**Key Features:**
- Classic Zha Jin Hua rules with blind/open betting
- Real-time multiplayer via WebSocket
- Multi-chain wallet integration (EVM / TON / Solana)
- On-chain chip purchases via smart contracts
- Telegram Mini App -- instant play, no download
- Bot AI for practice and onboarding
- Daily bonus system for player retention
- Internationalization in 4 languages (CN/EN/RU/ES)

### Marketing Plan

#### Target Audience

| Segment | Description |
|---------|-------------|
| Crypto / Web3 Users | Wallet holders familiar with on-chain interactions |
| Telegram Gaming Community | Players active in the Telegram Mini App ecosystem |
| Casual Card Game Players | Users who enjoy classic card games like Zha Jin Hua |

#### User Acquisition

- **Telegram Mini App Viral Sharing** -- One-tap sharing to groups and friends for frictionless onboarding
- **Room Invite Links** -- Each game room generates a unique link; friends click to join instantly
- **Daily Bonus Retention** -- 3 free claims per day (500 points each) to keep players coming back

#### Monetization

- **Platform Rake** -- A fee of `baseBet x number of participants` is collected per game round
- **On-Chain Chip Purchases** -- Players buy game points through smart contracts using native tokens (ETH/TON/SOL); 1 USD worth of native coin = 1,000 points

#### Growth Strategy

- **Multi-Chain Support** -- EVM, TON, and Solana ecosystems covered simultaneously for maximum reach
- **Internationalization** -- 4 languages (Chinese, English, Russian, Spanish)
- **Bot AI Onboarding** -- AI opponents lower the barrier for new players and improve first-time experience

#### Community Building

- **Telegram Group Operations** -- Official player groups for real-time interaction
- **Room Sharing Mechanics** -- Encourage players to invite friends for private games
- **Referral System** -- (Planned) Earn bonus points for inviting new players

#### Competitive Advantages

| Advantage | Details |
|-----------|---------|
| Instant Play | No download required -- play directly in browser or Telegram |
| Blockchain Transparency | On-chain chip purchases are publicly verifiable |
| Cross-Chain Support | Full coverage across three major blockchain ecosystems |

### MuleRun API Introduction

This project was built using the [MuleRun](https://mulerun.com) development workflow.

- **MuleRun** provides AI-powered development tools that accelerate the development process
- **API Capabilities**: code generation, project scaffolding, multi-language support
- Mule Zha Jin Hua -- from backend to frontend to smart contracts -- was built entirely with MuleRun's workflow
- Visit [mulerun.com](https://mulerun.com) for more information

### Mule AI Bot Roadmap

Next milestone: build **Mule AI Bot** -- an AI-powered game assistant.

| Feature | Description |
|---------|-------------|
| Intelligent Coaching | Real-time game analysis with strategy advice |
| Hand Analysis | Evaluate current hand strength and win probability |
| Strategy Suggestions | Betting/folding recommendations based on opponent patterns |
| Telegram Bot Integration | Chat with the AI assistant directly in Telegram |
| Multi-Language Interaction | Natural language support in CN/EN/RU/ES |
| Bankroll Management | Smart recommendations for bet sizing and stop-loss |

### Project Structure

```
zhajinhua/
├── backend/         Go backend server
├── frontend/        Vue 3 web frontend (TE/PE modes)
├── frontend-tg/     Telegram Mini App
└── contracts/       Smart contracts (EVM / TON / SOL)
```

Detailed documentation:
- [`backend/README.md`](backend/README.md) -- Backend server docs
- [`frontend/README.md`](frontend/README.md) -- Web frontend docs
- [`frontend-tg/README.md`](frontend-tg/README.md) -- Telegram Mini App docs
- [`contracts/README.md`](contracts/README.md) -- Smart contract docs

### Quick Start

```bash
# Backend
cd backend
go build -o server .
APP_MODE=te ./server

# Web frontend (test environment)
cd frontend
npm install && npm run build:te

# Telegram Mini App
cd frontend-tg
npm install && npm run build
```

### Deploy Frontends To Vercel

The recommended setup is to deploy the two frontends as separate Vercel projects:

- `frontend/`: web frontend
- `frontend-tg/`: Telegram Mini App frontend

Current deployed web frontend URLs:

```bash
Project: https://vercel.com/contractopens-projects/frontend
Production: https://frontend-j3a60rxac-contractopens-projects.vercel.app
Alias: https://frontend-gules-phi-47.vercel.app
```

Current deployed TG frontend URLs:

```bash
Project: https://vercel.com/contractopens-projects/mule-zjh-tg
Production: https://mule-zjh-2hrrjvxlf-contractopens-projects.vercel.app
Alias: https://mule-zjh-tg.vercel.app
```

Both frontends connect to:

```bash
https://game.atdl.link
```

Recommended environment variables for both Vercel projects:

```bash
VITE_API_BASE_URL=https://game.atdl.link
VITE_WS_URL=wss://game.atdl.link/ws
```

Notes:

- `frontend/` production build command: `VITE_APP_MODE=pe npm run build`
- `frontend-tg/` build command: `npm run build`
- output directory for both: `dist`

### License

TBD
