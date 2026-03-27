# Project Plan / 项目计划

## Goal / 目标

Evolve the current `v1` centralized game service into a community-node network where users can run the compiled backend on their own computers and become service providers, while keeping the system operable, auditable, and progressively safer.

将当前的 `v1` 中心化游戏服务逐步演进为一个“社区节点网络”：用户可以在自己的电脑上运行编译后的后端，成为服务提供者之一，同时保证系统仍然可运行、可审计，并且能逐步提升安全性与可信度。

This plan is intentionally staged. The near-term goal is not full decentralization, but validating the product and building toward a federated architecture step by step.

这份计划采用分阶段推进的方式。近期目标不是一步做到完全去中心化，而是先验证产品，再逐步演进到联邦式、社区节点参与的架构。

## Stage 1: Community Node Runtime / 第一阶段：社区节点运行

### Objective / 阶段目标

Allow users to run the backend locally as their own game node for private rooms, friend rooms, and early community-hosted matches.

允许用户在本地运行后端，作为自己的游戏节点，用于私有房、好友房，以及早期的社区托管房间。

### Deliverables / 交付物

- Build and package the backend so it can be run easily on user machines.
- Add basic node identity generation and persistence.
- Expose node metadata such as `node_id`, version, mode, and host info.
- Support self-hosted room creation tied to the hosting node.
- Document how to run a node and how clients connect to it.

- 将后端打包为可在用户机器上方便运行的版本。
- 增加基础节点身份生成与持久化能力。
- 提供节点元信息，例如 `node_id`、版本、模式、主机信息等。
- 支持与托管节点绑定的自托管房间创建能力。
- 编写社区节点运行与客户端连接方式说明。

### Why This Stage Matters / 为什么这一阶段重要

This is the fastest way to test whether people are willing to host, join, and trust community-run rooms without overbuilding the final network too early.

这是验证“用户是否愿意托管、愿意加入、愿意信任社区节点房间”的最快方式，同时避免过早把系统做得过重。

### Key Risks / 关键风险

- Nodes are easy to run, but not yet strongly trusted.
- Room hosting is possible, but still mostly based on social trust.
- Different users may run different versions unless versioning is enforced.

- 节点虽然容易运行，但可信度还不够强。
- 房间虽然能托管，但早期更多仍依赖社交信任。
- 如果不做版本约束，不同用户运行不同版本会导致兼容性问题。

## Stage 2: Node Discovery, Registration, and Reputation / 第二阶段：节点发现、注册与信誉

### Objective / 阶段目标

Make community nodes discoverable and distinguish between official, community, and friend-hosted services.

让社区节点可以被发现，并且能够区分官方节点、社区节点、好友托管节点等不同服务类型。

### Deliverables / 交付物

- Introduce node registration with signed node identity.
- Add a public directory or coordinator service for node discovery.
- Show node labels such as official/community/friend-hosted.
- Track basic node health, uptime, version, and room availability.
- Add first-pass reputation signals such as hosted games count, complaints, and reliability.

- 引入带签名的节点注册机制。
- 增加公共目录或协调服务，用于节点发现。
- 展示节点标签，例如官方、社区、好友托管等。
- 跟踪节点基础状态，如健康度、在线时长、版本、房间可用性。
- 增加第一版信誉信号，例如托管局数、投诉情况、稳定性等。

### Why This Stage Matters / 为什么这一阶段重要

Once users can self-host, the next challenge is helping players find rooms and decide which nodes they trust enough to join.

当用户已经可以自托管后，接下来的关键问题就是：玩家如何发现节点，以及如何判断哪些节点值得信任并加入。

### Key Risks / 关键风险

- Fake or low-quality nodes can pollute the network.
- Discovery without reputation will create poor first impressions.
- Version drift across nodes can break compatibility.

- 虚假节点或低质量节点会污染网络。
- 如果只有发现能力却没有信誉系统，用户第一次体验会很差。
- 节点间版本漂移会造成协议和功能兼容问题。

## Stage 3: Verifiable Dealing and Auditable Settlement / 第三阶段：可验证发牌与可审计结算

### Objective / 阶段目标

Reduce the trust required in community nodes by making core game outcomes verifiable or at least auditable after the fact.

通过让核心对局结果变得可验证，或至少在事后可审计，来降低用户对社区节点“必须完全信任”的要求。

### Deliverables / 交付物

- Introduce a signed event log for room lifecycle and player actions.
- Add a verifiable dealing design such as commit-reveal or multi-party entropy mixing.
- Persist settlement records in a form that can be replayed and audited.
- Add dispute-review tooling for suspicious matches.
- Define protocol versioning for rules, dealing, and settlement behavior.

- 为房间生命周期和玩家动作引入签名事件日志。
- 增加可验证发牌方案，例如 commit-reveal 或多方熵混合。
- 以可回放、可审计的形式持久化结算记录。
- 为可疑对局提供争议审查工具。
- 定义规则、发牌和结算协议的版本机制。

### Why This Stage Matters / 为什么这一阶段重要

Community-hosted game nodes only become credible when players can verify that the node did not secretly manipulate cards, outcomes, or payouts.

只有当玩家能够验证节点没有偷偷操控牌局、结果或结算时，社区托管节点才会真正具备可信度。

### Key Risks / 关键风险

- This is the most technically sensitive phase.
- Poor protocol design can still leave room for host manipulation.
- Verification that is too complex will slow product adoption.

- 这是技术上最敏感、最核心的阶段。
- 如果协议设计不严谨，节点仍然可能存在操控空间。
- 如果验证机制过于复杂，会拖慢产品落地与用户接受速度。

## Stage 4: Asset, Trust, and Network Economics / 第四阶段：资产、信任与网络经济

### Objective / 阶段目标

Add stronger trust incentives and a path toward a durable service-provider ecosystem.

建立更强的信任激励机制，并为长期可持续的服务提供者生态打基础。

### Deliverables / 交付物

- Connect node trust to on-chain or otherwise verifiable identity.
- Add operator staking, bonding, or other economic commitment if appropriate.
- Define how recharge, settlement, and operator incentives work in a multi-node network.
- Introduce policies for suspending or isolating malicious nodes.
- Build dashboards for node status, trust level, and economic activity.

- 将节点信任与链上身份或其他可验证身份体系关联起来。
- 在适合的时候增加运营者质押、保证金或其他经济约束。
- 定义多节点网络下的充值、结算与节点激励机制。
- 建立恶意节点暂停、隔离、降权等治理策略。
- 构建节点状态、信任等级和经济活动的可视化面板。

### Why This Stage Matters / 为什么这一阶段重要

This is the phase where the system stops being just “people can host nodes” and starts becoming a defensible network with incentives, accountability, and long-term expansion potential.

到了这个阶段，系统就不再只是“大家都能开节点”，而是开始变成一个有激励、有约束、有责任归属，并且具备长期扩张潜力的网络。

### Key Risks / 关键风险

- Economic design can become overly complex too early.
- Poor incentive design may reward bad actors.
- Legal, compliance, and platform risks become more important here.

- 经济模型如果过早做复杂，会拖累整体推进。
- 激励机制设计不当，可能反而奖励了不良行为。
- 法律、合规和平台风险在这一阶段会显著上升。

## Near-Term Execution Priority / 近期执行优先级

The next concrete plan should start with `Stage 1`.

下一步的实际执行计划应该从 `Stage 1` 开始。

Recommended immediate work:

1. Define a persistent `node_id` and node config file format.
2. Separate room ownership from the current in-memory singleton assumptions.
3. Add node metadata endpoints for health, identity, and version.
4. Decide how clients select or connect to a specific node.
5. Write deployment instructions for self-hosted community nodes.

建议优先做的事项：

1. 定义持久化的 `node_id` 与节点配置文件格式。
2. 将房间归属从当前单进程内存单例假设中拆出来。
3. 增加节点健康检查、身份信息、版本信息接口。
4. 明确客户端如何选择节点、连接节点。
5. 编写社区节点的部署与运行说明。

## Working Principle / 工作原则

Do not optimize for “fully decentralized” too early.

不要过早为了“完全去中心化”而做过度设计。

The product path is:

1. Prove users want to play.
2. Prove users will host.
3. Prove users can trust hosted games.
4. Then build the stronger network economics around that behavior.

产品推进路径应该是：

1. 先证明用户愿意玩。
2. 再证明用户愿意托管节点。
3. 再证明用户能够信任托管出来的对局。
4. 最后围绕这些真实行为建立更强的网络经济与治理机制。
