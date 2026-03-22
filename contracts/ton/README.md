# 炸金花筹码合约 -- TON 部署指南

## 概述

`zhajinhua_chips.fc` 是一个 FunC 智能合约，允许用户发送 TON 来购买游戏内筹码（积分）。管理员可以提取已收集的资金并更新价格。**定价模型：** 价值 1 美元的 TON = 1000 游戏积分。合约存储一个可配置的 `price_per_point`（单位为 nanoTON），管理员可根据 TON/USD 汇率变化进行调整。

## 编译

需要 TON FunC 编译器（`func`）和 Fift 汇编器。推荐使用 `blueprint`：

```bash
npx blueprint build zhajinhua_chips
```

或手动编译：
```bash
func -o zhajinhua_chips.fif -SPA stdlib.fc zhajinhua_chips.fc
fift -s zhajinhua_chips.fif
```

## 初始状态

合约的 `c4` 数据单元在部署时必须初始化以下字段：
| 字段             | 类型      | 说明                                     |
|------------------|-----------|------------------------------------------|
| admin_address    | MsgAddr   | 管理员钱包（标准地址）                   |
| price_per_point  | uint64    | 1 游戏积分的 nanoTON 成本                |
| total_deposits   | uint128   | 部署时设为 0                             |
| user_records     | dict      | 部署时为空字典（null cell ref）          |

示例：若 TON = 2 美元，则 1 美元 = 0.5 TON = 500000000 nanoTON。每美元 1000 积分，`price_per_point = 500000000 / 1000 = 500000` nanoTON。

## 部署

使用 `blueprint`：
```bash
npx blueprint run deployZhajinhuaChips
```
发送部署消息，op 为 `0x4`，附带包含编译代码和上述初始数据单元的 StateInit。

## 操作码
| Op 码   | 名称          | 发送者 | op+query_id 之后的消息体              |
|---------|---------------|--------|---------------------------------------|
| 0x1     | buy_chips     | 任何人 | （无 -- 转账金额即为支付）            |
| 0x2     | withdraw      | 管理员 | `amount:uint128`（0 = 全部提取）      |
| 0x3     | update_price  | 管理员 | `new_price:uint64`                    |
| 0x4     | deploy        | 管理员 | （无）                                |

空消息体的裸转账视为 `buy_chips`。

## GET 方法
- `get_contract_info()` -- 返回 `(price_per_point, total_deposits)`
- `get_user_record(addr_hash)` -- 返回用户的 `(total_points, total_spent)`
- `get_admin()` -- 返回管理员地址 slice

## 更新价格
当 TON/USD 汇率变化时，管理员发送 `update_price` 消息：

```
op:       0x3
query_id: 0
body:     new_price_per_point (uint64, 单位 nanoTON)
```

## 安全说明
- 仅部署时存储的管理员地址可调用 `withdraw` 和 `update_price`。
- 合约保留最低 0.05 TON 余额以支付存储租金。
- 弹回消息将被静默忽略。

---

# Zhajinhua Chips Contract -- TON Deployment Guide

## Overview

`zhajinhua_chips.fc` is a FunC smart contract that lets users send TON to purchase in-game chips (points). The admin can withdraw collected funds and update the price.

**Pricing model:** 1 USD worth of TON = 1000 game points. The contract stores a configurable `price_per_point` in nanoTON so the admin can adjust it as the TON/USD rate changes.

## Build

Requires the TON FunC compiler (`func`) and Fift assembler. Using `blueprint` (recommended):

```bash
npx blueprint build zhajinhua_chips
```

Or compile manually:

```bash
func -o zhajinhua_chips.fif -SPA stdlib.fc zhajinhua_chips.fc
fift -s zhajinhua_chips.fif
```

## Initial State

The contract's `c4` data cell must be initialised at deploy time with:

| Field            | Type      | Description                              |
|------------------|-----------|------------------------------------------|
| admin_address    | MsgAddr   | Admin wallet (std address)               |
| price_per_point  | uint64    | nanoTON cost of 1 game point             |
| total_deposits   | uint128   | Set to 0 at deploy                       |
| user_records     | dict      | Empty dict (null cell ref) at deploy     |

Example: if TON = $2 USD, then 1 USD = 0.5 TON = 500000000 nanoTON. For 1000 points per USD, `price_per_point = 500000000 / 1000 = 500000` nanoTON.

## Deploy

Using `blueprint`:

```bash
npx blueprint run deployZhajinhuaChips
```

Send the deploy message with op `0x4` and the StateInit containing the compiled code and initial data cell described above.

## Operations

| Op Code | Name          | Sender | Body after op+query_id                |
|---------|---------------|--------|---------------------------------------|
| 0x1     | buy_chips     | Any    | (none -- value is the payment)        |
| 0x2     | withdraw      | Admin  | `amount:uint128` (0 = withdraw all)   |
| 0x3     | update_price  | Admin  | `new_price:uint64`                    |
| 0x4     | deploy        | Admin  | (none)                                |

Bare transfers (empty body) are treated as `buy_chips`.

## GET Methods

- `get_contract_info()` -- returns `(price_per_point, total_deposits)`
- `get_user_record(addr_hash)` -- returns `(total_points, total_spent)` for a user
- `get_admin()` -- returns admin address slice

## Updating the Price

When the TON/USD exchange rate changes, the admin sends an `update_price` message:

```
op:       0x3
query_id: 0
body:     new_price_per_point (uint64, in nanoTON)
```

## Security Notes

- Only the admin address stored at deploy time can call `withdraw` and `update_price`.
- The contract keeps a minimum balance of 0.05 TON to cover storage rent.
- Bounced messages are silently ignored.
