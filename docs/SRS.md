# Agent OS v2 — 软件需求规范 (SRS)

> **版本**: v2.0.0-draft  
> **日期**: 2026-06-10  
> **状态**: 初稿  
> **作者**: Agent OS 核心团队

---

## 目录

1. [引言](#1-引言)
2. [总体描述](#2-总体描述)
3. [功能需求](#3-功能需求)
4. [非功能需求](#4-非功能需求)
5. [接口需求](#5-接口需求)
6. [数据设计](#6-数据设计)
7. [部署架构](#7-部署架构)
8. [附录](#8-附录)

---

## 1. 引言

### 1.1 目的

本文档定义 Agent OS v2 的软件需求规范 (Software Requirements Specification)。Agent OS v2 是一个融合区块链、AI 推理引擎与分布式机器人调度的操作系统级框架，旨在为自进化智能体提供完整的链上运行时环境。本文档为开发团队、测试团队及利益相关方提供统一的功能与非功能需求基准。

### 1.2 范围

Agent OS v2 整合三大开源体系，构建端到端的自进化智能体基础设施：

| 子系统 | 来源 | 职责 |
|--------|------|------|
| **Apex 框架** | [hernandez42/apex](https://github.com/hernandez42/apex) | 区块链基座 + 自进化框架，定义信号协议 `%Ψ_ASI/PHI_APEX` |
| **MOSS-AGI** | MOSS-AGI 项目 | 链上自进化通用智能体，核心公式 `EV = BV + Σ(Gene × Φ)` |
| **nanobot** | nanobot 项目 | 轻量机器人框架，提供分布式任务调度能力 |

**系统边界**：Agent OS v2 负责从底层区块链共识到上层 AI 推理调度的全栈管理。系统不直接管理硬件资源，通过容器化接口与宿主机交互。

### 1.3 定义、缩略语与缩写

| 术语 | 定义 |
|------|------|
| **Agent OS** | 智能体操作系统，为自进化 AI 智能体提供链上运行时 |
| **Apex** | 区块链基座框架，提供 PoA 共识、交易存证、Merkle 树及跨链桥 |
| **MOSS-AGI** | 链上自进化通用智能体 (Multi-objective On-chain Self-evolving AGI) |
| **nanobot** | 轻量级分布式机器人调度框架 |
| **PoA** | 权威证明 (Proof of Authority) |
| **ΔG** | 基因变异量，衡量智能体进化的度量 |
| **EV / BV** | 进化值 (Evolution Value) / 基础值 (Base Value) |
| **Φ** | 进化系数 (Phi coefficient)，基因表达权重 |
| **Ψ_ASI** | 人工超级智能信号协议 |
| **PHI_APEX** | Apex 框架的哲学层抽象信号 |
| **EventBus** | 跨模块事件总线，实现异步数据互通 |
| **AES-256-GCM** | 高级加密标准，256位密钥，GCM 模式 (认证加密) |
| **ECDSA** | 椭圆曲线数字签名算法 |
| **JWT** | JSON Web Token |

### 1.4 参考资料

| 编号 | 文档/项目 | 说明 |
|------|-----------|------|
| [REF-01] | hernandez42/apex | Apex 区块链框架源码及白皮书 |
| [REF-02] | MOSS-AGI Technical Report | MOSS-AGI 自进化智能体技术报告 |
| [REF-03] | nanobot Documentation | nanobot 机器人框架文档 |
| [REF-04] | IEEE 830-1998 | IEEE 软件需求规范推荐实践 |
| [REF-05] | NIST SP 800-38D | AES-GCM 加密模式规范 |
| [REF-06] | RFC 7519 | JSON Web Token (JWT) 标准 |
| [REF-07] | SEC 2 | 椭圆曲线密码学标准曲线 |

---

## 2. 总体描述

### 2.1 产品前景

```
┌─────────────────────────────────────────────────────────────────┐
│                      Agent OS v2 整体架构                        │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐             │
│  │  Apex 框架   │  │  MOSS-AGI   │  │   nanobot    │             │
│  │ (区块链基座) │  │ (AI推理引擎) │  │ (机器人调度) │             │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘             │
│         │                │                │                     │
│  ═══════╪════════════════╪════════════════╪═════════════════    │
│         │          EventBus 事件总线       │                     │
│  ═══════╪════════════════╪════════════════╪═════════════════    │
│         │                │                │                     │
│  ┌──────┴────────────────┴────────────────┴──────┐              │
│  │              安全层 (Security Layer)            │              │
│  │     AES-256-GCM + ECDSA + JWT + RBAC           │              │
│  └───────────────────────────────────────────────┘              │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

Agent OS v2 定位于 **链上自进化智能体的基础设施层**，解决以下核心问题：

1. **链上可信计算**：通过 Apex 框架提供不可篡改的计算存证
2. **智能体自进化**：通过 MOSS-AGI 引擎实现 `EV = BV + Σ(Gene × Φ)` 的持续进化
3. **分布式任务编排**：通过 nanobot 实现大规模机器人任务的可靠调度
4. **安全隔离**：通过多层安全架构保障数据与通信安全

### 2.2 产品功能

Agent OS v2 提供五大核心功能模块：

| 模块 | 功能概述 | 优先级 |
|------|----------|--------|
| **F-01 区块链层** | PoA 出块、交易存证、Merkle 树验证、跨链桥通信 | P0 |
| **F-02 AI 推理层** | MOSS-AGI 自进化引擎、ΔG 基因变异、ReAct 推理、9维记忆 | P0 |
| **F-03 机器人层** | nanobot 分布式调度、定时任务、异常巡检、节点管理 | P0 |
| **F-04 事件总线** | 跨模块数据互通、双向指令流、异步消息队列 | P0 |
| **F-05 安全层** | 端到端加密、数字签名、身份认证、访问控制 | P0 |

### 2.3 用户特征

| 用户角色 | 描述 | 技术水平 |
|----------|------|----------|
| **智能体开发者** | 编写自进化策略、定义基因变异规则 | 高级：熟悉 AI/ML 与区块链 |
| **运维工程师** | 部署、监控、扩缩容系统 | 中级：熟悉 Docker/K8s |
| **DApp 用户** | 通过前端与链上智能体交互 | 初级：通过图形界面操作 |
| **系统管理员** | 管理权限、安全策略、节点准入 | 高级：熟悉密码学与安全 |

### 2.4 约束条件

| 编号 | 约束 | 说明 |
|------|------|------|
| C-01 | 开源兼容性 | 所有组件必须保持开源许可证兼容 (MIT/Apache-2.0) |
| C-02 | 区块链不可变性 | 一旦上链，数据不可修改，需在链下进行充分验证 |
| C-03 | 实时性要求 | AI 推理响应时间 ≤ 200ms (P95) |
| C-04 | 数据隐私 | 遵循 GDPR 及中国《数据安全法》要求 |
| C-05 | 向后兼容 | v2 必须兼容 v1 的链上数据格式和 API |

### 2.5 假设与依赖

**假设**：
- 运行环境为 Docker 容器或 Kubernetes 集群
- 节点间网络延迟 ≤ 50ms (同一数据中心)
- 每个 PoA 验证节点具有已知的静态身份
- MOSS-AGI 的基因空间在单次进化周期内是有限可枚举的

**依赖**：
- Go 1.21+ (Apex 框架)
- Python 3.11+ (MOSS-AGI 引擎)
- Rust 1.75+ (nanobot 核心)
- PostgreSQL 15+ (元数据存储)
- Redis 7+ (缓存与消息队列)
- IPFS (链下大数据存储)

---

## 3. 功能需求

### 3.1 区块链层 (Blockchain Layer)

> 基于 Apex 框架实现

#### 3.1.1 PoA 共识出块

| 需求编号 | 需求描述 | 优先级 |
|----------|----------|--------|
| BR-0101 | 系统应支持基于 PoA (Proof of Authority) 的共识出块机制，验证节点由系统管理员预先授权 | P0 |
| BR-0102 | 出块间隔应可配置，默认 2 秒，最小 0.5 秒 | P0 |
| BR-0103 | 每个区块应包含：区块头、交易列表、Merkle 根、验证者签名 | P0 |
| BR-0104 | 系统应支持至少 3 个验证节点的共识，容忍 `f` 个拜占庭节点 (总节点数 ≥ 3f+1) | P0 |
| BR-0105 | 当验证节点超过 2/3 签名时，区块视为最终确认 (Finality) | P0 |

#### 3.1.2 交易存证

| 需求编号 | 需求描述 | 优先级 |
|----------|----------|--------|
| BR-0201 | 系统应将所有智能体进化事件记录为链上交易 | P0 |
| BR-0202 | 交易应包含：发送者地址、接收者地址、交易类型、载荷哈希、时间戳、签名 | P0 |
| BR-0203 | 支持以下交易类型：`GeneEvolution`(基因进化)、`TaskDispatch`(任务调度)、`StateCheckpoint`(状态检查点)、`CrossChain`(跨链通信) | P0 |
| BR-0204 | 交易验证应包括：签名验证、余额检查、重放攻击检测 | P0 |
| BR-0205 | 每个交易应生成唯一的交易哈希 (SHA-256) | P0 |

#### 3.1.3 Merkle 树验证

| 需求编号 | 需求描述 | 优先级 |
|----------|----------|--------|
| BR-0301 | 每个区块应构建完整的 Merkle 树，根哈希写入区块头 | P0 |
| BR-0302 | 系统应支持 SPV (Simplified Payment Verification) 证明生成与验证 | P1 |
| BR-0303 | Merkle 证明路径长度应为 O(log n)，其中 n 为区块内交易数 | P0 |
| BR-0304 | 支持增量 Merkle 树更新，新交易加入时仅重新计算受影响的路径 | P1 |

#### 3.1.4 跨链桥

| 需求编号 | 需求描述 | 优先级 |
|----------|----------|--------|
| BR-0401 | 系统应支持与至少一条外部链 (如 Ethereum) 的跨链资产转移 | P1 |
| BR-0402 | 跨链通信应使用轻客户端验证 (Light Client Verification) 模式 | P1 |
| BR-0403 | 跨链交易应支持两阶段提交：锁定 → 确认 → 释放 | P1 |
| BR-0404 | 超时未确认的跨链交易应触发自动回滚机制 | P1 |

#### 3.1.5 信号协议 `Ψ_ASI/PHI_APEX`

| 需求编号 | 需求描述 | 优先级 |
|----------|----------|--------|
| BR-0501 | 系统应实现 `%Ψ_ASI/PHI_APEX` 信号协议，作为 Apex 框架的语义通信层 | P1 |
| BR-0502 | 信号消息应支持结构化载荷 (JSON-LD 格式) | P1 |
| BR-0503 | 信号协议应支持请求-响应和发布-订阅两种通信模式 | P1 |

---

### 3.2 AI 推理层 (AI Inference Layer)

> 基于 MOSS-AGI 引擎实现

#### 3.2.1 自进化引擎

| 需求编号 | 需求描述 | 优先级 |
|----------|----------|--------|
| AI-0101 | 系统应实现 MOSS-AGI 自进化引擎，核心公式：`EV = BV + Σ(Gene_i × Φ_i)` | P0 |
| AI-0102 | 进化周期应可配置，默认每 100 个推理任务触发一次基因评估 | P0 |
| AI-0103 | 系统应维护每个智能体的基因向量 `G = {Gene_1, Gene_2, ..., Gene_n}` | P0 |
| AI-0104 | 进化结果应记录为链上 `GeneEvolution` 交易 | P0 |
| AI-0105 | 系统应支持回滚至任意历史版本的基因快照 | P1 |

#### 3.2.2 ΔG 基因变异

| 需求编号 | 需求描述 | 优先级 |
|----------|----------|--------|
| AI-0201 | 系统应计算基因变异量 `ΔG = G_new - G_old` | P0 |
| AI-0202 | 变异策略应支持：随机变异、梯度变异、交叉变异 | P0 |
| AI-0203 | 单次变异幅度应受约束：`|ΔGene_i| ≤ δ_max`，防止基因爆炸 | P0 |
| AI-0204 | 变异方向应受适应度函数引导，适应度下降超过阈值时自动回滚 | P1 |

#### 3.2.3 ReAct 推理

| 需求编号 | 需求描述 | 优先级 |
|----------|----------|--------|
| AI-0301 | 系统应实现 ReAct (Reasoning + Acting) 推理范式 | P0 |
| AI-0302 | 推理链应支持：思考 (Thought) → 行动 (Action) → 观察 (Observation) 循环 | P0 |
| AI-0303 | 单次推理链最大步骤数应可配置，默认 10 步 | P0 |
| AI-0304 | 推理过程应完整记录，支持事后审计与回放 | P1 |
| AI-0305 | 系统应支持推理缓存，相同输入在缓存有效期内直接返回结果 | P1 |

#### 3.2.4 9维记忆系统

| 需求编号 | 需求描述 | 优先级 |
|----------|----------|--------|
| AI-0401 | 系统应实现 9 维记忆架构，维度定义如下： | P0 |

**9维记忆定义**：

| 维度 | 名称 | 描述 | 存储介质 |
|------|------|------|----------|
| M1 | 短期工作记忆 | 当前推理上下文，≤ 32K tokens | 内存 (Redis) |
| M2 | 长期语义记忆 | 世界知识与概念关系 | 向量数据库 (pgvector) |
| M3 | 情景记忆 | 历史交互事件时间线 | PostgreSQL |
| M4 | 程序记忆 | 已学会的技能与操作流程 | 文件系统 + Git |
| M5 | 感知记忆 | 原始输入的短暂缓存 | Redis (TTL 60s) |
| M6 | 元认知记忆 | 对自身推理能力的评估 | PostgreSQL |
| M7 | 社交记忆 | 与其他智能体的交互历史 | PostgreSQL |
| M8 | 链上记忆 | 区块链上的不可变记录 | 区块链 |
| M9 | 进化记忆 | 基因变异历史与适应度轨迹 | 区块链 + 向量数据库 |

| 需求编号 | 需求描述 | 优先级 |
|----------|----------|--------|
| AI-0402 | 各维度记忆应支持独立的读写接口 | P0 |
| AI-0403 | 记忆应支持跨维度关联查询 (如：从情景记忆检索相关语义记忆) | P1 |
| AI-0404 | 链上记忆 (M8, M9) 应与区块链层同步，保证不可篡改性 | P0 |
| AI-0405 | 各维度记忆应有独立的容量上限与淘汰策略 | P0 |

---

### 3.3 机器人层 (Robot Layer)

> 基于 nanobot 框架实现

#### 3.3.1 分布式调度

| 需求编号 | 需求描述 | 优先级 |
|----------|----------|--------|
| RB-0101 | 系统应支持基于 nanobot 的分布式任务调度，任务自动分配至可用节点 | P0 |
| RB-0102 | 调度策略应支持：轮询、最少负载、亲和性绑定 | P0 |
| RB-0103 | 单个任务应支持超时设置，超时后自动终止并重试 | P0 |
| RB-0104 | 任务结果应通过 EventBus 发布，供其他模块消费 | P0 |
| RB-0105 | 系统应支持任务依赖图 (DAG)，按拓扑顺序执行 | P1 |

#### 3.3.2 定时任务

| 需求编号 | 需求描述 | 优先级 |
|----------|----------|--------|
| RB-0201 | 系统应支持 Cron 表达式定义定时任务 | P0 |
| RB-0202 | 定时任务应支持分布式锁，防止多节点重复执行 | P0 |
| RB-0203 | 定时任务执行历史应可查询，保留最近 1000 条记录 | P1 |
| RB-0204 | 系统应支持定时任务的动态增删改，无需重启服务 | P0 |

#### 3.3.3 异常巡检

| 需求编号 | 需求描述 | 优先级 |
|----------|----------|--------|
| RB-0301 | 系统应定期巡检所有节点的健康状态 (CPU、内存、磁盘、网络) | P0 |
| RB-0302 | 巡检异常时应自动触发告警，告警通道支持：Webhook、邮件、链上事件 | P0 |
| RB-0303 | 系统应支持自定义巡检脚本，通过插件机制扩展 | P1 |
| RB-0304 | 连续 3 次巡检失败的节点应自动隔离，并通知管理员 | P0 |

#### 3.3.4 节点管理

| 需求编号 | 需求描述 | 优先级 |
|----------|----------|--------|
| RB-0401 | 系统应支持节点的动态注册与注销 | P0 |
| RB-0402 | 节点元信息应包含：节点 ID、类型、标签、能力集、网络地址 | P0 |
| RB-0403 | 系统应维护节点状态机：`PENDING → ACTIVE → DRAINING → OFFLINE` | P0 |
| RB-0404 | 节点间通信应使用 mTLS (mutual TLS) 加密 | P0 |

---

### 3.4 事件总线 (EventBus)

#### 3.4.1 跨模块数据互通

| 需求编号 | 需求描述 | 优先级 |
|----------|----------|--------|
| EB-0101 | 系统应实现中心化事件总线，连接区块链层、AI 推理层、机器人层 | P0 |
| EB-0102 | 事件应支持标准格式：`{event_id, event_type, source, timestamp, payload, metadata}` | P0 |
| EB-0103 | 事件总线应支持至少 10,000 events/s 的吞吐量 | P0 |
| EB-0104 | 事件应支持持久化存储，可回放任意时间点之后的事件流 | P1 |
| EB-0105 | 支持事件过滤与路由：按 `event_type`、`source` 进行条件分发 | P0 |

#### 3.4.2 双向指令流

| 需求编号 | 需求描述 | 优先级 |
|----------|----------|--------|
| EB-0201 | 系统应支持下行指令：管理层 → 执行层 (如：启动进化、调度任务) | P0 |
| EB-0202 | 系统应支持上行报告：执行层 → 管理层 (如：任务完成、异常告警) | P0 |
| EB-0203 | 指令应支持优先级：`CRITICAL > HIGH > NORMAL > LOW` | P0 |
| EB-0204 | 指令应支持确认机制 (ACK)，未确认的指令应自动重试 (最多 3 次) | P1 |
| EB-0205 | 指令链路应支持背压 (Backpressure) 控制，防止消费者过载 | P1 |

---

### 3.5 安全层 (Security Layer)

#### 3.5.1 数据加密

| 需求编号 | 需求描述 | 优先级 |
|----------|----------|--------|
| SE-0101 | 所有链下敏感数据应使用 AES-256-GCM 加密存储 | P0 |
| SE-0102 | 数据传输应使用 TLS 1.3 加密 | P0 |
| SE-0103 | 加密密钥应通过密钥管理服务 (KMS) 管理，支持自动轮换 | P0 |
| SE-0104 | GCM 认证标签 (Authentication Tag) 应随密文一起存储，用于完整性校验 | P0 |

#### 3.5.2 数字签名

| 需求编号 | 需求描述 | 优先级 |
|----------|----------|--------|
| SE-0201 | 所有链上交易应使用 ECDSA (secp256k1) 签名 | P0 |
| SE-0202 | 签名应包含时间戳，防止重放攻击 | P0 |
| SE-0203 | 系统应支持多签 (Multi-Signature) 机制，用于高权限操作 | P1 |
| SE-0204 | 私钥应存储在安全硬件 (HSM) 或加密的密钥库中 | P0 |

#### 3.5.3 身份认证与访问控制

| 需求编号 | 需求描述 | 优先级 |
|----------|----------|--------|
| SE-0301 | 用户认证应使用 JWT (RS256 算法)，令牌有效期 ≤ 24 小时 | P0 |
| SE-0302 | 系统应实现 RBAC (基于角色的访问控制)，预定义角色：Admin、Developer、Operator、Viewer | P0 |
| SE-0303 | 敏感操作 (如：基因变异策略修改) 应要求二次认证 | P1 |
| SE-0304 | 所有认证与授权事件应记录审计日志，保留 ≥ 90 天 | P0 |
| SE-0305 | 连续 5 次认证失败应自动锁定账户 30 分钟 | P1 |

---

## 4. 非功能需求

### 4.1 性能需求

| 需求编号 | 指标 | 目标值 | 测量方法 |
|----------|------|--------|----------|
| NFR-0101 | 区块链出块延迟 | ≤ 2s (P95) | 出块时间戳差值 |
| NFR-0102 | 交易吞吐量 | ≥ 500 TPS | 区块内交易数 / 出块间隔 |
| NFR-0103 | AI 推理响应时间 | ≤ 200ms (P95) | 请求到首个 token 的时间 |
| NFR-0104 | 基因进化周期 | ≤ 10s (单次完整进化) | 进化开始到结果上链的时间 |
| NFR-0105 | 事件总线吞吐量 | ≥ 10,000 events/s | 持续负载测试 |
| NFR-0106 | 任务调度延迟 | ≤ 100ms (P95) | 任务提交到开始执行的时间 |
| NFR-0107 | 跨链交易确认 | ≤ 60s | 锁定交易到释放交易的时间 |
| NFR-0108 | 并发连接数 | ≥ 10,000 | 同时在线的 WebSocket 连接 |

### 4.2 安全需求

| 需求编号 | 需求描述 | 测试方法 |
|----------|----------|----------|
| NFR-0201 | 加密算法强度：AES-256 (对称)、ECDSA secp256k1 (非对称)、SHA-256 (哈希) | 算法审查 |
| NFR-0202 | 无已知 CVE 漏洞 | 依赖项安全扫描 (Trivy/Snyk) |
| NFR-0203 | OWASP Top 10 全覆盖 | 渗透测试 |
| NFR-0204 | 私钥不出安全边界 | 架构审查 |
| NFR-0205 | 所有外部输入进行严格的参数校验与转义 | 代码审查 + Fuzzing |
| NFR-0206 | 安全审计日志不可篡改 (写入区块链) | 链上验证 |

### 4.3 可用性需求

| 需求编号 | 需求描述 | 目标值 |
|----------|----------|--------|
| NFR-0301 | 系统整体可用性 | ≥ 99.95% (每月 ≤ 22 分钟停机) |
| NFR-0302 | 区块链层可用性 | ≥ 99.99% (PoA 共识保障) |
| NFR-0303 | 故障恢复时间 (RTO) | ≤ 5 分钟 |
| NFR-0304 | 数据恢复点 (RPO) | ≤ 1 分钟 |
| NFR-0305 | 滚动升级零停机 | 支持蓝绿部署或金丝雀发布 |

### 4.4 可维护性需求

| 需求编号 | 需求描述 | 实施方式 |
|----------|----------|----------|
| NFR-0401 | 代码测试覆盖率 | ≥ 80% (单元测试)，≥ 60% (集成测试) |
| NFR-0402 | 日志标准化 | 结构化 JSON 日志，统一字段：`timestamp, level, module, trace_id, message` |
| NFR-0403 | 分布式追踪 | 支持 OpenTelemetry，链路追踪覆盖全请求路径 |
| NFR-0404 | 配置管理 | 环境变量 + 配置文件 + 热更新，无需重启 |
| NFR-0405 | API 版本管理 | URL 路径版本化 (如 `/api/v2/`)，旧版本保留 ≥ 6 个月 |
| NFR-0406 | 文档自动生成 | API 文档由 OpenAPI 3.0 规范自动生成 |

---

## 5. 接口需求

### 5.1 内部模块 API

#### 5.1.1 区块链层 API

```
┌─────────────────────────────────────────────────────────┐
│                  Blockchain API (gRPC)                   │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  rpc SubmitTransaction(TxRequest)    → TxResponse       │
│  rpc GetTransaction(TxHash)          → Transaction      │
│  rpc GetBlock(BlockNumber)           → Block            │
│  rpc GetLatestBlock(Empty)           → Block            │
│  rpc GetMerkleProof(TxHash)          → MerkleProof      │
│  rpc GetChainState(Empty)            → ChainState       │
│  rpc CrossChainTransfer(CrossChainReq) → CrossChainResp │
│  rpc SubscribeBlocks(BlockFilter)    → stream Block     │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

#### 5.1.2 AI 推理层 API

```
┌─────────────────────────────────────────────────────────┐
│                  AI Inference API (REST)                 │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  POST /api/v2/agent/{id}/infer          执行推理        │
│  POST /api/v2/agent/{id}/evolve         触发进化        │
│  GET  /api/v2/agent/{id}/genes          获取基因向量    │
│  PUT  /api/v2/agent/{id}/genes          更新基因策略    │
│  GET  /api/v2/agent/{id}/memory/{dim}   查询记忆维度    │
│  POST /api/v2/agent/{id}/memory/{dim}   写入记忆        │
│  GET  /api/v2/agent/{id}/history        推理历史        │
│  GET  /api/v2/agent/{id}/fitness        适应度评估      │
│  POST /api/v2/agent/{id}/checkpoint     创建状态快照    │
│  POST /api/v2/agent/{id}/rollback/{ver} 回滚至指定版本  │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

#### 5.1.3 机器人层 API

```
┌─────────────────────────────────────────────────────────┐
│                  Robot Layer API (gRPC + REST)           │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  POST   /api/v2/tasks                   提交任务        │
│  GET    /api/v2/tasks/{id}              查询任务状态    │
│  DELETE /api/v2/tasks/{id}              取消任务        │
│  GET    /api/v2/tasks                   任务列表        │
│  POST   /api/v2/schedules               创建定时任务    │
│  PUT    /api/v2/schedules/{id}          更新定时任务    │
│  DELETE /api/v2/schedules/{id}          删除定时任务    │
│  GET    /api/v2/nodes                    节点列表       │
│  GET    /api/v2/nodes/{id}/health       节点健康状态    │
│  POST   /api/v2/nodes/{id}/drain        节点排空        │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

#### 5.1.4 事件总线 API

```
┌─────────────────────────────────────────────────────────┐
│                   EventBus API (WebSocket + gRPC)        │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  POST   /api/v2/events                  发布事件        │
│  WS     /api/v2/events/subscribe        订阅事件流      │
│  POST   /api/v2/commands                发送指令        │
│  GET    /api/v2/commands/{id}/status    查询指令状态    │
│  POST   /api/v2/events/replay           事件回放        │
│                                                         │
│  gRPC:                                                  │
│  rpc PublishEvent(Event)           → PublishResult      │
│  rpc SubscribeEvents(EventFilter)  → stream Event       │
│  rpc SendCommand(Command)          → CommandResult      │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

#### 5.1.5 安全层 API

```
┌─────────────────────────────────────────────────────────┐
│                   Security API (REST)                    │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  POST   /api/v2/auth/login              登录获取 JWT    │
│  POST   /api/v2/auth/refresh            刷新令牌        │
│  POST   /api/v2/auth/logout             登出            │
│  GET    /api/v2/users                    用户列表       │
│  PUT    /api/v2/users/{id}/role         更新角色        │
│  GET    /api/v2/audit/logs              审计日志        │
│  POST   /api/v2/keys/rotate             密钥轮换        │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

### 5.2 外部接口

#### 5.2.1 Ethereum 跨链桥接口

| 接口 | 方向 | 协议 | 说明 |
|------|------|------|------|
| `bridge_lock` | Agent OS → Ethereum | JSON-RPC | 锁定资产至跨链合约 |
| `bridge_release` | Ethereum → Agent OS | Oracle callback | 释放跨链资产 |
| `bridge_verify` | 双向 | JSON-RPC | 验证对端链上交易 |

#### 5.2.2 IPFS 存储接口

| 接口 | 方向 | 协议 | 说明 |
|------|------|------|------|
| `ipfs_add` | Agent OS → IPFS | HTTP API | 上传大数据至 IPFS |
| `ipfs_cat` | IPFS → Agent OS | HTTP API | 从 IPFS 读取数据 |
| `ipfs_pin` | Agent OS → IPFS | HTTP API | 固定数据防止垃圾回收 |

#### 5.2.3 监控接口

| 接口 | 方向 | 协议 | 说明 |
|------|------|------|------|
| Prometheus metrics | Agent OS → Prometheus | HTTP `/metrics` | 系统指标暴露 |
| Grafana dashboard | Grafana → Agent OS | HTTP | 可视化仪表板 |
| AlertManager | Agent OS → AlertManager | HTTP webhook | 告警推送 |

---

## 6. 数据设计

### 6.1 核心数据结构

#### 6.1.1 区块 (Block)

```protobuf
message BlockHeader {
  uint64   version         = 1;   // 协议版本
  bytes    prev_block_hash = 2;   // 前一区块哈希
  bytes    merkle_root     = 3;   // Merkle 树根
  uint64   timestamp       = 4;   // Unix 时间戳 (毫秒)
  uint64   block_number    = 5;   // 区块高度
  bytes    validator       = 6;   // 验证者地址
  bytes    signature       = 7;   // 验证者签名
}

message Block {
  BlockHeader     header       = 1;
  repeated Transaction txs     = 2;
  repeated bytes  extra_data   = 3;
}
```

#### 6.1.2 交易 (Transaction)

```protobuf
enum TxType {
  GENE_EVOLUTION    = 0;  // 基因进化
  TASK_DISPATCH     = 1;  // 任务调度
  STATE_CHECKPOINT  = 2;  // 状态检查点
  CROSS_CHAIN       = 3;  // 跨链通信
  ADMIN             = 4;  // 管理操作
}

message Transaction {
  bytes    tx_hash     = 1;  // SHA-256 哈希
  bytes    from        = 2;  // 发送者地址
  bytes    to          = 3;  // 接收者地址
  TxType   type        = 4;  // 交易类型
  bytes    payload     = 5;  // 载荷 (序列化后)
  uint64   nonce       = 6;  // 防重放
  uint64   timestamp   = 7;  // 时间戳
  bytes    signature   = 8;  // ECDSA 签名
}
```

#### 6.1.3 基因向量 (GeneVector)

```json
{
  "agent_id": "uuid-v4",
  "version": 42,
  "genes": {
    "reasoning":      { "value": 0.85, "bounds": [0.0, 1.0], "phi": 1.2 },
    "creativity":     { "value": 0.72, "bounds": [0.0, 1.0], "phi": 0.9 },
    "memory_util":    { "value": 0.68, "bounds": [0.0, 1.0], "phi": 1.0 },
    "tool_use":       { "value": 0.91, "bounds": [0.0, 1.0], "phi": 1.1 },
    "planning":       { "value": 0.77, "bounds": [0.0, 1.0], "phi": 1.3 },
    "adaptability":   { "value": 0.63, "bounds": [0.0, 1.0], "phi": 0.8 },
    "social":         { "value": 0.55, "bounds": [0.0, 1.0], "phi": 0.7 },
    "efficiency":     { "value": 0.82, "bounds": [0.0, 1.0], "phi": 1.0 },
    "robustness":     { "value": 0.79, "bounds": [0.0, 1.0], "phi": 1.1 }
  },
  "fitness": 0.847,
  "evolution_value": 12.35,
  "base_value": 5.00,
  "created_at": "2026-06-10T00:00:00Z",
  "block_hash": "0xabc..."
}
```

#### 6.1.4 事件 (Event)

```json
{
  "event_id": "uuid-v4",
  "event_type": "gene.evolution.completed",
  "source": "moss-agi-engine",
  "timestamp": "2026-06-10T12:00:00.000Z",
  "priority": "NORMAL",
  "payload": {
    "agent_id": "uuid-v4",
    "old_version": 41,
    "new_version": 42,
    "delta_g": 0.035,
    "fitness_change": 0.012
  },
  "metadata": {
    "trace_id": "trace-uuid",
    "correlation_id": "corr-uuid",
    "ttl": 300
  }
}
```

#### 6.1.5 任务 (Task)

```json
{
  "task_id": "uuid-v4",
  "task_type": "agent.inference",
  "status": "PENDING | RUNNING | SUCCESS | FAILED | TIMEOUT",
  "priority": "HIGH",
  "agent_id": "uuid-v4",
  "input": { "..." : "..." },
  "output": null,
  "assigned_node": "node-uuid",
  "retry_count": 0,
  "max_retries": 3,
  "timeout_ms": 30000,
  "created_at": "2026-06-10T12:00:00Z",
  "started_at": null,
  "completed_at": null,
  "dependencies": ["task-uuid-1", "task-uuid-2"]
}
```

#### 6.1.6 节点 (Node)

```json
{
  "node_id": "uuid-v4",
  "node_type": "worker | validator | gateway",
  "state": "PENDING | ACTIVE | DRAINING | OFFLINE",
  "address": "10.0.1.5:8443",
  "capabilities": ["inference", "training", "storage"],
  "labels": { "region": "cn-east", "gpu": "a100" },
  "resources": {
    "cpu_total": 64, "cpu_available": 48,
    "memory_total_gb": 256, "memory_available_gb": 180,
    "gpu_total": 8, "gpu_available": 6
  },
  "health": { "status": "HEALTHY", "last_check": "2026-06-10T12:00:00Z" },
  "registered_at": "2026-06-01T00:00:00Z"
}
```

### 6.2 数据库 Schema 概览

```
┌──────────────────────────────────────────────────────┐
│                   PostgreSQL 15+                      │
├──────────────────────────────────────────────────────┤
│                                                      │
│  agents          (agent_id PK, name, status, ...)    │
│  gene_snapshots  (id PK, agent_id FK, version, ...)  │
│  memories        (id PK, agent_id, dimension, ...)   │
│  inference_logs  (id PK, agent_id, input_hash, ...)  │
│  tasks           (task_id PK, type, status, ...)     │
│  nodes           (node_id PK, type, state, ...)      │
│  audit_logs      (id PK, user_id, action, ...)       │
│  users           (user_id PK, name, role, ...)       │
│                                                      │
├──────────────────────────────────────────────────────┤
│                   pgvector 扩展                       │
├──────────────────────────────────────────────────────┤
│                                                      │
│  semantic_memory (id PK, agent_id, embedding vec(1536),│
│                   content TEXT, metadata JSONB)       │
│  evolution_memory (id PK, agent_id, gene_hash,       │
│                    embedding vec(512), metadata JSONB)│
│                                                      │
├──────────────────────────────────────────────────────┤
│                   Redis 7+                            │
├──────────────────────────────────────────────────────┤
│                                                      │
│  working_memory:{agent_id}  (Hash)                   │
│  perception_buffer:{agent_id} (String, TTL 60s)      │
│  event_queue (Stream)                                 │
│  task_locks:{task_id} (String, TTL)                  │
│                                                      │
└──────────────────────────────────────────────────────┘
```

---

## 7. 部署架构

### 7.1 Docker 容器化

```yaml
# docker-compose.yml 概要
services:
  # ===== 区块链层 =====
  blockchain-node-1:
    image: agentos/blockchain:latest
    environment:
      - NODE_ROLE=validator
      - POA_AUTHORIZED=true
    ports: ["8545:8545", "8546:8546"]

  blockchain-node-2:
    image: agentos/blockchain:latest
    environment:
      - NODE_ROLE=validator

  blockchain-node-3:
    image: agentos/blockchain:latest
    environment:
      - NODE_ROLE=validator

  # ===== AI 推理层 =====
  moss-agi-engine:
    image: agentos/moss-agi:latest
    deploy:
      resources:
        reservations:
          devices:
            - driver: nvidia
              count: 1
              capabilities: [gpu]
    ports: ["8080:8080"]

  # ===== 机器人层 =====
  nanobot-scheduler:
    image: agentos/nanobot:latest
    environment:
      - SCHEDULER_MODE=distributed
    ports: ["9090:9090"]

  nanobot-worker-1:
    image: agentos/nanobot-worker:latest
  nanobot-worker-2:
    image: agentos/nanobot-worker:latest

  # ===== 事件总线 =====
  event-bus:
    image: agentos/eventbus:latest
    ports: ["8070:8070"]

  # ===== 基础设施 =====
  postgres:
    image: pgvector/pgvector:pg15
    volumes: ["pgdata:/var/lib/postgresql/data"]

  redis:
    image: redis:7-alpine

  ipfs:
    image: ipfs/kubo:latest
```

### 7.2 Kubernetes 部署

```yaml
# k8s/ 部署结构
k8s/
├── namespaces.yaml          # agentos-system, agentos-data
├── blockchain/
│   ├── statefulset.yaml     # 3 副本 PoA 验证节点
│   ├── service.yaml         # ClusterIP + Headless Service
│   └── configmap.yaml       # PoA 配置
├── ai-engine/
│   ├── deployment.yaml      # MOSS-AGI 引擎 (HPA 1-10)
│   ├── service.yaml
│   └── hpa.yaml             # CPU/GPU 自动扩缩
├── nanobot/
│   ├── deployment-scheduler.yaml
│   ├── deployment-worker.yaml  # HPA 2-20
│   └── service.yaml
├── eventbus/
│   ├── deployment.yaml      # 3 副本
│   ├── service.yaml
│   └── ingress.yaml
├── infra/
│   ├── postgres-statefulset.yaml
│   ├── redis-deployment.yaml
│   ├── pvc.yaml             # 持久化存储
│   └── networkpolicy.yaml   # 网络隔离
├── security/
│   ├── rbac.yaml            # 角色与绑定
│   ├── secrets.yaml         # 密钥 (通过 SealedSecrets)
│   └── cert-manager.yaml    # TLS 证书自动管理
└── monitoring/
    ├── prometheus.yaml
    ├── grafana-dashboards/
    └── alerting-rules.yaml
```

### 7.3 CI/CD 流水线

```
┌─────────────────────────────────────────────────────────────────┐
│                      CI/CD Pipeline (GitHub Actions)             │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Push/PR ──► Lint & Format                                      │
│           ──► Unit Tests (Go/Python/Rust)                       │
│           ──► Integration Tests (Docker Compose)                │
│           ──► Security Scan (Trivy + Snyk)                      │
│           ──► Build Container Images                             │
│           ──► Push to Registry (GHCR)                           │
│           ──► Deploy to Staging (Helm)                           │
│           ──► E2E Tests (Staging)                                │
│           ──► Manual Approval Gate                               │
│           ──► Deploy to Production (Blue/Green)                  │
│           ──► Smoke Tests (Production)                           │
│           ──► Monitor & Alert                                    │
│                                                                 │
│  回滚策略：自动检测生产环境错误率 > 1% 时自动回滚               │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### 7.4 环境矩阵

| 环境 | 用途 | 区块链节点数 | Worker 数 | 数据库 |
|------|------|-------------|-----------|--------|
| **dev** | 本地开发 | 1 | 1 | SQLite + 内存 Redis |
| **staging** | 集成测试 | 3 | 2 | PostgreSQL + Redis |
| **production** | 生产运行 | 5+ | 2-20 (HPA) | PostgreSQL HA + Redis Cluster |

---

## 8. 附录

### 8.1 术语表

| 术语 | 英文 | 定义 |
|------|------|------|
| 智能体 | Agent | 具有自主推理和行动能力的 AI 实体 |
| 基因 | Gene | 智能体的能力参数，构成进化的基本单元 |
| 适应度 | Fitness | 智能体在特定任务上的表现评分 |
| 进化值 | Evolution Value (EV) | 智能体的综合进化水平 |
| 基础值 | Base Value (BV) | 智能体的初始能力基线 |
| 进化系数 | Phi (Φ) | 各基因维度的权重系数 |
| 基因变异量 | Delta G (ΔG) | 单次进化中基因向量的变化量 |
| 验证者 | Validator | PoA 共识中有权出块的授权节点 |
| 最终性 | Finality | 区块不可逆转的确认状态 |
| 事件总线 | Event Bus | 跨模块异步消息通信基础设施 |
| 背压 | Backpressure | 消费者过载时对生产者的流控机制 |
| 信号协议 | Signal Protocol | `%Ψ_ASI/PHI_APEX` 语义通信协议 |

### 8.2 核心公式推导

#### 8.2.1 进化值公式

**公式**：

```
EV = BV + Σ(i=1 to n) (Gene_i × Φ_i)
```

**推导过程**：

1. 智能体的初始能力由基础值 `BV` 决定
2. 随着进化，每个基因维度 `Gene_i` 在其权重 `Φ_i` 的调节下对总能力产生贡献
3. 进化值 `EV` 是基础值与所有基因加权贡献之和

**参数约束**：

```
∀i: 0 ≤ Gene_i ≤ 1      (基因值归一化)
∀i: Φ_i > 0              (进化系数为正)
BV ≥ 0                    (基础值非负)
```

**示例计算**：

```
BV = 5.0
Gene = {0.85, 0.72, 0.68, 0.91, 0.77, 0.63, 0.55, 0.82, 0.79}
Φ    = {1.2,  0.9,  1.0,  1.1,  1.3,  0.8,  0.7,  1.0,  1.1 }

Σ(Gene_i × Φ_i) = 0.85×1.2 + 0.72×0.9 + 0.68×1.0 + 0.91×1.1
                 + 0.77×1.3 + 0.63×0.8 + 0.55×0.7 + 0.82×1.0
                 + 0.79×1.1
                 = 1.020 + 0.648 + 0.680 + 1.001
                 + 1.001 + 0.504 + 0.385 + 0.820
                 + 0.869
                 = 6.928

EV = 5.0 + 6.928 = 11.928
```

#### 8.2.2 基因变异公式

**公式**：

```
ΔG = G(t+1) - G(t)
|ΔGene_i| ≤ δ_max = 0.1    (单步变异上限)
```

**变异策略**：

- **随机变异**: `ΔGene_i = random(-δ_max, δ_max)`
- **梯度变异**: `ΔGene_i = -η × ∂Fitness/∂Gene_i` (η 为学习率)
- **交叉变异**: `ΔGene_i = α × (Gene_i^parent_A - Gene_i^parent_B)` (α ∈ [0, 1])

**适应度函数**：

```
Fitness = Σ(i=1 to n) (w_i × Performance_i(Gene_i))
```

其中 `w_i` 为任务权重，`Performance_i` 为第 i 个基因在当前任务上的表现函数。

#### 8.2.3 PoA 共识最终性

**确认条件**：

```
若总验证者数为 N，拜占庭容错数为 f，则：
- N ≥ 3f + 1
- 区块确认需 ≥ 2f + 1 个验证者签名
```

**示例**：N = 4, f = 1, 确认需 ≥ 3 个签名。

### 8.3 变更历史

| 版本 | 日期 | 作者 | 变更内容 |
|------|------|------|----------|
| v2.0.0-draft | 2026-06-10 | Agent OS Team | 初始版本，定义全部功能与非功能需求 |

---

> **文档结束** — 本文档为 Agent OS v2 的软件需求规范基线。所有后续变更需通过变更控制流程审批。
