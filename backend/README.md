# 炸金花后端服务

## 技术栈

- Go 1.21
- Gorilla WebSocket
- SQLite3 (WAL 模式)

## 目录结构

```
backend/
├── main.go                 # 入口文件
├── game/
│   ├── card.go             # 牌型定义与比较逻辑
│   ├── player.go           # 玩家状态管理
│   ├── room.go             # 房间与游戏流程控制
│   └── database.go         # 数据库操作
├── handler/
│   ├── ws.go               # WebSocket 连接处理
│   ├── wallet.go           # 多链钱包验证
│   └── config.go           # 配置加载
```

## 功能模块

| 模块 | 说明 |
|------|------|
| WebSocket 实时通信 | 基于 Gorilla WebSocket 的全双工通信 |
| 游戏逻辑引擎 | 发牌、比牌、下注、弃牌等核心逻辑 |
| 多链钱包验证 | 支持多条链的钱包地址签名验证 |
| 数据库持久化 | SQLite3 WAL 模式，保障并发读写性能 |
| 每日奖励系统 | 每日登录领取奖励 |
| 平台抽成 | 每局结算时按比例收取平台费用 |

## 数据库表

| 表名 | 用途 |
|------|------|
| `users` | 用户信息 |
| `transactions` | 交易记录 |
| `platform_fees` | 平台抽成记录 |
| `daily_bonus` | 每日奖励领取记录 |

## 环境变量

| 变量 | 说明 | 可选值 |
|------|------|--------|
| `APP_MODE` | 运行模式 | `te` (测试) / `pe` (生产) |
| `PORT` | 监听端口 | 默认见配置 |
| `DB_PATH` | SQLite 数据库文件路径 | 默认 `zhajinhua.db` |

## 运行方式

```bash
go build -o server
APP_MODE=te ./server
```

## Docker 独立部署

在 `backend/` 目录下可以单独构建和运行后端：

```bash
docker compose up -d --build
```

常用变量：

```bash
APP_MODE=pe
PORT=8080
ADMIN_PASSWORD=change-me
```

## WebSocket 消息协议

### Client -> Server

| 消息类型 | 说明 |
|----------|------|
| `create_room` | 创建房间 |
| `join_room` | 加入房间 |
| `join_by_code` | 通过邀请码加入 |
| `start_game` | 开始游戏 |
| `action` | 游戏操作（下注/跟注/弃牌等） |
| `new_round` | 发起新一轮 |
| `match` | 开始匹配 |
| `cancel_match` | 取消匹配 |
| `recharge` | 充值 |
| `leave_room` | 离开房间 |
| `claim_bonus` | 领取每日奖励 |

### Server -> Client

| 消息类型 | 说明 |
|----------|------|
| `connected` | 连接成功 |
| `room_created` | 房间已创建 |
| `room_state` | 房间状态同步 |
| `game_event` | 游戏事件推送 |
| `game_end` | 游戏结束结算 |
| `match_status` | 匹配状态更新 |
| `match_found` | 匹配成功 |
| `recharge_success` | 充值成功 |
| `next_round_status` | 下一轮状态 |
| `left_room` | 已离开房间 |
| `kicked` | 被踢出房间 |
| `error` | 错误信息 |
| `bankrupt` | 破产通知 |
| `bonus_claimed` | 奖励已领取 |

---

# Zha Jinhua Backend Service

## Tech Stack

- Go 1.21
- Gorilla WebSocket
- SQLite3 (WAL mode)

## Directory Structure

```
backend/
├── main.go                 # Entry point
├── game/
│   ├── card.go             # Card types and comparison logic
│   ├── player.go           # Player state management
│   ├── room.go             # Room and game flow control
│   └── database.go         # Database operations
├── handler/
│   ├── ws.go               # WebSocket connection handler
│   ├── wallet.go           # Multi-chain wallet verification
│   └── config.go           # Configuration loading
```

## Feature Modules

| Module | Description |
|--------|-------------|
| WebSocket Real-time Communication | Full-duplex communication via Gorilla WebSocket |
| Game Logic Engine | Core logic for dealing, comparing, betting, folding |
| Multi-chain Wallet Verification | Signature verification for multi-chain wallet addresses |
| Database Persistence | SQLite3 WAL mode for concurrent read/write performance |
| Daily Bonus System | Daily login rewards |
| Platform Commission | Per-round settlement with platform fee deduction |

## Database Tables

| Table | Purpose |
|-------|---------|
| `users` | User information |
| `transactions` | Transaction records |
| `platform_fees` | Platform commission records |
| `daily_bonus` | Daily bonus claim records |

## Environment Variables

| Variable | Description | Values |
|----------|-------------|--------|
| `APP_MODE` | Runtime mode | `te` (test) / `pe` (production) |
| `PORT` | Listening port | See configuration |
| `DB_PATH` | SQLite database file path | Defaults to `zhajinhua.db` |

## Running

```bash
go build -o server
APP_MODE=te ./server
```

## Standalone Docker Deployment

You can build and run the backend directly from the `backend/` directory:

```bash
docker compose up -d --build
```

Common variables:

```bash
APP_MODE=pe
PORT=8080
ADMIN_PASSWORD=change-me
```

## WebSocket Message Protocol

### Client -> Server

| Message Type | Description |
|--------------|-------------|
| `create_room` | Create a room |
| `join_room` | Join a room |
| `join_by_code` | Join by invite code |
| `start_game` | Start the game |
| `action` | Game action (bet/call/fold, etc.) |
| `new_round` | Start a new round |
| `match` | Start matchmaking |
| `cancel_match` | Cancel matchmaking |
| `recharge` | Recharge balance |
| `leave_room` | Leave the room |
| `claim_bonus` | Claim daily bonus |

### Server -> Client

| Message Type | Description |
|--------------|-------------|
| `connected` | Connection established |
| `room_created` | Room created |
| `room_state` | Room state sync |
| `game_event` | Game event push |
| `game_end` | Game end settlement |
| `match_status` | Match status update |
| `match_found` | Match found |
| `recharge_success` | Recharge successful |
| `next_round_status` | Next round status |
| `left_room` | Left the room |
| `kicked` | Kicked from room |
| `error` | Error message |
| `bankrupt` | Bankruptcy notice |
| `bonus_claimed` | Bonus claimed |
