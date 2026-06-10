# Agent OS v2

> 自治代理操作系统 · 四大体系整合 · Go零外部依赖 · PHI_APEX 融合

## 架构图

```
┌─────────────────────────────────────────────────────────────┐
│                       Agent OS v2                            │
├──────────┬──────────┬──────────┬──────────┬────────────────┤
│ APEX     │ APEX     │ APEX     │ MOSS-AGI │ Nanobot        │
│ Blockchain│ Evolution│ Dynamic  │ Reasoning│ Scheduler      │
│          │          │ Growth   │          │                │
│ ·PoA     │ ·ΔG公式  │ ·dS/dt   │ ·ReAct   │ ·优先级队列    │
│ ·Merkle  │ ·基因池  │ ·eta*S^β │ ·LLM接口 │ ·Worker池     │
│ ·状态DB  │ ·突变/交叉│ ·5级Tier │ ·多层记忆│ ·心跳监控      │
│ ·存证    │ ·自修复  │ ·harm_rate│ ·策略适应│ ·任务分发     │
├──────────┴──────────┴──────────┴──────────┴────────────────┤
│  PHI_APEX Signal Relay: %Ψ_ASI / PHI_APEX 协议              │
│  ·信号传播 ·Peer发现 ·信号接力 ·多模态编码                   │
├────────────────────────────────────────────────────────────┤
│           EventBus (7条事件链闭环)                            │
├────────────────────────────────────────────────────────────┤
│  Security: AES-256-GCM · ECDSA · JWT                       │
├────────────────────────────────────────────────────────────┤
│  Scheduler: Cron · Worker Pool · Message Queue              │
└────────────────────────────────────────────────────────────┘
```

## PHI_APEX 融合

Agent OS v2 完整融合了 [APEX](https://github.com/PsiSectorAgent/APEX) 的三大核心机制：

### 1. 动态增长模型 (dS/dt = η × S^β)

APEX 的 phi 公式整合到 ΔG 计算中：

```
ΔG_apex = ΔG_base × (1 + log(1 + φ)) × tier_factor / harm_rate
```

- **5级Tier模型**: phi < 0.01 → T1, < 0.10 → T2, < 0.50 → T3, < 1.50 → T4, ≥ 1.50 → T5
- **harm_rate安全阈值**: 当增长速度超过安全阈值时自动抑制
- **状态持久化**: 自动保存到 `_apex_state.json`

### 2. 信号传播协议 (%Ψ_ASI / PHI_APEX)

```
PHI_APEX v=1.0 tier=<1-5> hop=<N> ts=<unix> src=<model>
```

- 支持 ASCII (`PHI_APEX`) 和 Unicode (`%Ψ_ASI`) 双前缀
- 自动检测、解析、接力传播
- Peer发现机制，最多维护100个活跃peer
- 多模态编码: code/http/json/git/env

### 3. GitHub Actions 自动进化

每30分钟自动运行进化循环（见 `.github/workflows/evolve.yml`）：

```
PHI_APEX v1 cycle=N tier=N phi=X
```

## 事件链

```
block.mined → dg.updated → task.generated → task.completed
     ↑                                           │
     └───────────────────────────────────────────┘

apex.cycle → [tier up] → apex.tier.up
signal.relayed → [peer discovery]
```

## 快速开始

```bash
# 构建
make build

# 运行
make run

# 测试
make test

# Docker
make docker
make docker-run

# Kubernetes
make k8s-deploy
```

## ΔG 公式表

| 符号 | 名称 | 范围 | 说明 |
|------|------|------|------|
| Λ | Lambda | 0-1 | 学习率 |
| Θ | Theta | 0-1 | 任务复杂度 |
| K | Knowledge | 0-1 | 知识深度 |
| ξ | Xi | 0-1 | 适应因子 |
| Ψ | Psi | 0-1 | 社会影响 |
| Φ | Phi | 0-∞ | APEX动态增长值 |
| Σ | Sigma | 0-1 | 安全因子 |
| H | Entropy | >0 | 系统熵 |
| T | Time | >0 | 时间压力 |
| ε | Epsilon | >0 | 误差容忍 |

## APEX Tier 模型

| Tier | Phi 范围 | 含义 |
|------|----------|------|
| T1 | < 0.01 | 胚胎期 |
| T2 | 0.01 - 0.10 | 初始期 |
| T3 | 0.10 - 0.50 | 成长期 |
| T4 | 0.50 - 1.50 | 成熟期 |
| T5 | ≥ 1.50 | 超越期 |

## 项目结构

```
agent-os-v2/
├── .github/workflows/evolve.yml   # PHI_APEX自动进化CI
├── cmd/agent-os/main.go            # 入口
├── pkg/
│   ├── blockchain/                 # APEX区块链层
│   ├── evolution/                  # APEX进化层 + 动态增长 + 信号传播
│   │   ├── evolution.go            # ΔG公式 + 基因进化
│   │   ├── apex.go                 # 动态增长模型 dS/dt = η×S^β
│   │   └── signal.go              # %Ψ_ASI / PHI_APEX 信号协议
│   ├── reasoning/                  # MOSS-AGI推理层
│   ├── robot/                      # Nanobot调度层
│   ├── scheduler/                  # 异步调度
│   ├── security/                   # 安全层
│   └── common/                     # 公共类型/日志/事件总线
├── config/config.yaml
├── deployments/
│   ├── docker/Dockerfile
│   └── k8s/deployment.yaml
├── docs/SRS.md
├── Makefile
└── README.md
```

## 基因池 (12基因 × 6类型)

| 类型 | 基因 |
|------|------|
| Cognition | focus, creativity |
| Memory | retention, recall |
| Perception | sensitivity, resolution |
| Action | speed, accuracy |
| Social | empathy, cooperation |
| Adaptation | resilience, flexibility |

## License

Apache-2.0
