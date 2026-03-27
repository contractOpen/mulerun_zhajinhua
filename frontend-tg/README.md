# 炸金花 Telegram Mini App

Telegram Mini App 版本的炸金花游戏客户端，专为 TG 内嵌浏览器优化。跳过登录页，自动获取 TG 用户信息，支持 TON 钱包身份和触觉反馈。

## 技术栈

- **Vue 3.4** + **Vite 5** 单页应用
- **Telegram Web App SDK** (`window.Telegram.WebApp`)
- **TonConnect** — TON 钱包连接
- 通过 `@shared` 别名复用主前端 composables

## `@shared` 别名

`vite.config.js` 中配置了路径别名：

```js
'@shared': path.resolve(__dirname, '../frontend/src/composables')
```

这意味着 `import { useGame } from '@shared/useGame'` 实际导入的是 `../frontend/src/composables/useGame.js`。TG 版本和 Web 版本共享同一套游戏逻辑和国际化模块。

## 目录结构

```
frontend-tg/
├── package.json
├── vite.config.js
├── src/
│   ├── App.vue                         # 入口组件（大厅 + 游戏桌，无登录页）
│   ├── main.js                         # 应用挂载
│   └── composables/
│       └── useTelegram.js              # TG SDK 封装
│
│   (通过 @shared 别名引用主前端)
│       useGame.js                      # WebSocket 游戏逻辑
│       useI18n.js                      # 国际化
│       useWallet.js                    # 钱包逻辑
│       useAudio.js                     # 音效
```

## 构建命令

```bash
npm install              # 安装依赖
npm run dev              # 开发模式
npm run build            # 构建 -> ../backend/static-tg/
```

可选环境变量：

```bash
VITE_API_BASE_URL=https://game.atdl.link
VITE_WS_URL=wss://game.atdl.link/ws
```

## Vercel 部署

Telegram Mini App 前端可以直接单独部署到 Vercel。

当前线上地址：

```bash
Project: https://vercel.com/contractopens-projects/mule-zjh-tg
Production: https://mule-zjh-2hrrjvxlf-contractopens-projects.vercel.app
Alias: https://mule-zjh-tg.vercel.app
```

1. 将 `frontend-tg/` 目录导入为一个独立 Vercel Project
2. `Framework Preset = Vite`
3. `Build Command` 使用：

```bash
npm run build
```

4. `Output Directory` 使用：

```bash
dist
```

5. 推荐配置环境变量：

```bash
VITE_API_BASE_URL=https://game.atdl.link
VITE_WS_URL=wss://game.atdl.link/ws
```

## TG 专属特性

### 自动登录
无需手动输入昵称，直接从 `Telegram.WebApp.initDataUnsafe.user` 获取用户名和 ID。

### 触觉反馈
下注、看牌、弃牌等操作调用 `HapticFeedback` API，提供原生触感体验：
- `impact` — 按钮操作
- `success` / `error` — 结果反馈
- `selection` — 选择操作

### 主题适配
使用 `--tg-theme-*` CSS 变量，自动跟随 TG 深色/浅色模式。

### TON 钱包身份
优先通过 TonConnect 获取真实 TON 地址，不可用时回退到 `tg_<userId>` 格式。所有房间操作使用 `tonIdentity` + `'ton'` 链标识。

## URL 参数 & 深度链接

| 参数 | 用途 | 示例 |
|------|------|------|
| `?room=CODE` | 直接加入指定房间 | `https://t.me/bot/app?room=ABC123` |
| `?startapp=CODE` | TG 启动参数，等同于 room | `https://t.me/bot/app?startapp=ABC123` |

用户点击分享链接后自动进入对应房间，无需手动输入房间码。

## Composables

### `useTelegram.js`

| 导出 | 说明 |
|------|------|
| `init()` | 调用 `tg.ready()` / `tg.expand()`，解析用户信息 |
| `getTonAddress()` | 尝试 TonConnect，回退 `initDataUnsafe.wallet` |
| `hapticFeedback(type)` | 触觉反馈：impact / success / error / selection |
| `showAlert()` / `showConfirm()` | TG 原生弹窗，不可用时降级为浏览器弹窗 |
| `setMainButton()` / `hideMainButton()` | TG 底部主按钮控制 |
| `userId` / `userName` / `userLanguage` | 响应式属性，来自 TG 用户数据 |

### 共享模块（来自主前端）

- **`useGame.js`** — WebSocket 游戏逻辑（连接、房间、操作、奖励）
- **`useI18n.js`** — 国际化，自动检测 `userLanguage`

## 功能

- **快速开始** — 创建 2 人公开房间，立即匹配
- **创建房间** — 创建私人房间，分享房间码
- **加入房间** — 输入房间码或通过 URL 参数直接加入
- **每日奖励** — 领取每日免费积分
- **简化游戏桌** — 看牌、跟注、加注、全押、弃牌，带结果弹窗

## 配置 TG Bot & Mini App

1. 通过 [@BotFather](https://t.me/BotFather) 创建 Bot
2. 发送 `/newapp` 创建 Mini App
3. 设置 Web App URL 为你的部署地址（如 `https://your-domain.com/tg/`）
4. 后端需将 `/tg/` 路径指向 `static-tg/` 目录
5. 可选：配置 `/setmenubutton` 添加快速入口

---

# Zhajinhua Telegram Mini App

Telegram Mini App client for Zha Jinhua (Chinese poker), optimized for the TG in-app browser. Skips the login page, reads TG user info automatically, and supports TON wallet identity and haptic feedback.

## Tech Stack

- **Vue 3.4** + **Vite 5** SPA
- **Telegram Web App SDK** (`window.Telegram.WebApp`)
- **TonConnect** -- TON wallet connection
- Reuses main frontend composables via the `@shared` alias

## `@shared` Alias

Configured in `vite.config.js`:

```js
'@shared': path.resolve(__dirname, '../frontend/src/composables')
```

`import { useGame } from '@shared/useGame'` resolves to `../frontend/src/composables/useGame.js`. The TG and web versions share the same game logic and i18n modules.

## Directory Structure

```
frontend-tg/
├── package.json
├── vite.config.js
├── src/
│   ├── App.vue                         # Entry component (lobby + game, no login page)
│   ├── main.js                         # App mount
│   └── composables/
│       └── useTelegram.js              # TG SDK wrapper
│
│   (via @shared alias from main frontend)
│       useGame.js                      # WebSocket game logic
│       useI18n.js                      # i18n
│       useWallet.js                    # Wallet logic
│       useAudio.js                     # Sound effects
```

## Build Commands

```bash
npm install              # Install dependencies
npm run dev              # Dev server
npm run build            # Build -> ../backend/static-tg/
```

Optional environment variables:

```bash
VITE_API_BASE_URL=https://game.atdl.link
VITE_WS_URL=wss://game.atdl.link/ws
```

## Vercel Deployment

The Telegram Mini App frontend can be deployed to Vercel as a standalone project.

Current deployed URLs:

```bash
Project: https://vercel.com/contractopens-projects/mule-zjh-tg
Production: https://mule-zjh-2hrrjvxlf-contractopens-projects.vercel.app
Alias: https://mule-zjh-tg.vercel.app
```

1. Import the `frontend-tg/` directory as a separate Vercel project
2. Set `Framework Preset = Vite`
3. Use this build command:

```bash
npm run build
```

4. Use this output directory:

```bash
dist
```

5. Recommended environment variables:

```bash
VITE_API_BASE_URL=https://game.atdl.link
VITE_WS_URL=wss://game.atdl.link/ws
```

## TG-Specific Features

### Auto Login
No manual nickname entry. User info is read directly from `Telegram.WebApp.initDataUnsafe.user`.

### Haptic Feedback
Betting, looking at cards, and folding trigger `HapticFeedback` API calls:
- `impact` -- button actions
- `success` / `error` -- result feedback
- `selection` -- selection actions

### Theme Adaptation
Uses `--tg-theme-*` CSS variables to follow TG dark/light mode automatically.

### TON Wallet Identity
Prefers real TON address via TonConnect; falls back to `tg_<userId>`. All room operations use `tonIdentity` with `'ton'` as the chain identifier.

## URL Parameters & Deep Linking

| Parameter | Purpose | Example |
|-----------|---------|---------|
| `?room=CODE` | Join a specific room directly | `https://t.me/bot/app?room=ABC123` |
| `?startapp=CODE` | TG launch param, same as room | `https://t.me/bot/app?startapp=ABC123` |

Users clicking a shared link are taken directly into the room without entering a code manually.

## Composables

### `useTelegram.js`

| Export | Description |
|--------|-------------|
| `init()` | Calls `tg.ready()` / `tg.expand()`, parses user info |
| `getTonAddress()` | Tries TonConnect, falls back to `initDataUnsafe.wallet` |
| `hapticFeedback(type)` | Haptic feedback: impact / success / error / selection |
| `showAlert()` / `showConfirm()` | TG native popups, degrades to browser alerts |
| `setMainButton()` / `hideMainButton()` | TG bottom main button control |
| `userId` / `userName` / `userLanguage` | Reactive refs from TG user data |

### Shared from Main Frontend

- **`useGame.js`** -- WebSocket game logic (connect, rooms, actions, bonus)
- **`useI18n.js`** -- i18n with auto-detection from `userLanguage`

## Features

- **Quick Start** -- Creates a 2-player public room for instant matching
- **Create Room** -- Private rooms with shareable codes
- **Join Room** -- Enter code or use `?room=` / `?startapp=` URL params
- **Daily Bonus** -- Claim free chips daily
- **Simplified Game Table** -- Look, call, raise, all-in, fold with result overlay

## Configure TG Bot & Mini App

1. Create a bot via [@BotFather](https://t.me/BotFather)
2. Send `/newapp` to create a Mini App
3. Set the Web App URL to your deployment address (e.g., `https://your-domain.com/tg/`)
4. The backend must serve the `/tg/` path from the `static-tg/` directory
5. Optional: use `/setmenubutton` to add a quick-access button
