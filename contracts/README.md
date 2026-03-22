# Zhajinhua 智能合约

三条链的游戏积分购买合约，用户使用原生代币购买游戏积分（1 USD = 1000 积分），管理员可提取资金。

## 合约概览

| 链 | 语言 | 文件 | 定价方式 |
|----|------|------|----------|
| EVM (Ethereum/BSC/等) | Solidity ^0.8.19 | `evm/ZhajinhuaChips.sol` | Chainlink 预言机 + 回退价格 |
| TON | FunC | `ton/zhajinhua_chips.fc` | 管理员设定 price_per_point |
| Solana | Anchor/Rust | `sol/zhajinhua_chips.rs` | 管理员设定 sol_per_usd |

## 核心功能

所有合约实现相同的业务逻辑：购买积分、管理员提取资金、价格更新、链上用户记录。

## EVM 合约 — `ZhajinhuaChips.sol`

- **Chainlink 预言机** — 实时获取 ETH/USD 价格（8 位小数）
- **回退机制** — 预言机过期或失败时使用 `fallbackPriceUsd`
- **过期阈值** — `stalenessThreshold` 控制预言机数据的有效时间
- **直接转账购买** — 支持 `buyChips()` 函数调用或直接发送 ETH（`receive()`）
- **预览报价** — `quotePoints(ethAmount)` 查看可获得的积分数
- **支持所有 EVM 链** — 只需配置对应链的 Chainlink 喂价地址

```bash
# Hardhat 部署: npx hardhat run scripts/deploy.js --network sepolia
# 参数: priceFeed地址, fallbackPriceUsd(如 200000000000=$2000), stalenessThreshold(如 3600)
```

## TON 合约 — `zhajinhua_chips.fc`

- **FunC 语言** — 使用 TON 标准开发范式
- **操作码协议** — `0x1` 购买 / `0x2` 提取 / `0x3` 更新价格 / `0x4` 部署
- **裸转账** — 不带 op code 的转账自动视为购买积分
- **最低余额保护** — 提取时保留 0.05 TON 防止合约被销毁
- **GET 方法** — `get_contract_info()` / `get_user_record(addr_hash)` / `get_admin()`
- **存储布局** — admin_address + price_per_point(uint64) + total_deposits(uint128) + user_records(dict)

> 完整的 TON 编译、部署、测试流程请查看 [ton/README.md](ton/README.md)。

## Solana 合约 — `zhajinhua_chips.rs`

- **Anchor 框架** — 使用 Anchor 宏简化账户验证和序列化
- **PDA 账户体系**：
  - `game_state` PDA (seed: `"game_state"`) — 全局配置（管理员、价格、累计数据）
  - `vault` PDA (seed: `"vault"`) — 存放所有 SOL 的金库
  - `user_account` PDA (seed: `"user_account" + buyer_pubkey`) — 用户购买记录
- **指令**：`initialize` / `buy_chips` / `withdraw` / `update_price`
- **溢出保护** — 所有算术使用 `checked_*` 操作，中间值用 u128

```bash
# Anchor 部署: anchor build && anchor deploy
# 初始化: initialize(sol_per_usd)，如 SOL=$200 则传入 5_000_000
```

## 积分计算公式

| 链 | 公式 |
|----|------|
| EVM | `points = ethAmount * ethPriceUsd * 1000 / (1e18 * 1e8)` |
| TON | `points = msg_value / price_per_point` |
| SOL | `chips = sol_amount * 1000 / sol_per_usd` |

最终效果相同：**1 美元价值的原生币 = 1000 游戏积分**。

---

# Zhajinhua Smart Contracts

Cross-chain game chip purchase contracts. Users send native tokens to receive game points (1 USD = 1000 points). Admins can withdraw accumulated funds.

## Overview

| Chain | Language | File | Pricing |
|-------|----------|------|---------|
| EVM (Ethereum/BSC/etc.) | Solidity ^0.8.19 | `evm/ZhajinhuaChips.sol` | Chainlink oracle + fallback |
| TON | FunC | `ton/zhajinhua_chips.fc` | Admin-set price_per_point |
| Solana | Anchor/Rust | `sol/zhajinhua_chips.rs` | Admin-set sol_per_usd |

## Core Features (all chains)

Buy chips, admin withdraw, price update, on-chain user records.

## EVM — `ZhajinhuaChips.sol`

- Chainlink oracle for real-time ETH/USD pricing (8 decimals), with configurable fallback price and staleness threshold
- Supports `buyChips()` or direct ETH transfer via `receive()`
- `quotePoints(ethAmount)` for previewing chip yield
- Deployable on any EVM chain with a Chainlink price feed

## TON — `zhajinhua_chips.fc`

- Op-code protocol: `0x1` buy / `0x2` withdraw / `0x3` update price / `0x4` deploy
- Bare transfers (no op-code) default to chip purchase
- Retains 0.05 TON minimum balance on withdrawal
- GET methods: `get_contract_info()`, `get_user_record(addr_hash)`, `get_admin()`

> See [ton/README.md](ton/README.md) for detailed compilation and deployment instructions.

## Solana — `zhajinhua_chips.rs`

- Built with Anchor framework; PDA-based account structure
- Accounts: `game_state` (global config), `vault` (SOL treasury), `user_account` (per-user records)
- Instructions: `initialize`, `buy_chips`, `withdraw`, `update_price`
- Overflow-safe arithmetic via `checked_*` operations with u128 intermediates

## Points Formula

| Chain | Formula |
|-------|---------|
| EVM | `points = ethAmount * ethPriceUsd * 1000 / (1e18 * 1e8)` |
| TON | `points = msg_value / price_per_point` |
| SOL | `chips = sol_amount * 1000 / sol_per_usd` |

Net result across all chains: **1 USD worth of native token = 1000 game points**.
