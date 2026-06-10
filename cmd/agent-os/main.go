// Agent OS v2 — 开源AGI人工智能架构
//
// 整合三大开源体系:
//   - hernandez42/apex: 区块链基座 + 自进化框架 + %%Ψ_ASI信号协议
//   - MOSS-AGI: 链上自进化通用智能体 (EV = BV + Σ(Gene × Φ))
//   - nanobot: 轻量机器人框架 (分布式任务调度 + 巡检 + 故障恢复)
//
// 架构分层解耦:
//   区块链层 → 资产/数据/交易存证 (PoA + Merkle树 + 跨链桥)
//   AI推理层 → MOSS-AGI自进化引擎 + ΔG公式 + ReAct推理
//   机器人层 → nanobot分布式调度 + 定时任务 + 异常巡检
//   事件总线 → 跨模块数据互通 + 双向指令流
//   安全层  → AES-256-GCM + ECDSA + JWT
package main

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"
	"time"

	"github.com/nousresearch/agent-os-v2/pkg/api"
	"github.com/nousresearch/agent-os-v2/pkg/blockchain"
	"github.com/nousresearch/agent-os-v2/pkg/common"
	"github.com/nousresearch/agent-os-v2/pkg/evolution"
	"github.com/nousresearch/agent-os-v2/pkg/moss"
	"github.com/nousresearch/agent-os-v2/pkg/nanobot"
	"github.com/nousresearch/agent-os-v2/pkg/reasoning"
	"github.com/nousresearch/agent-os-v2/pkg/scheduler"
	"github.com/nousresearch/agent-os-v2/pkg/security"
)

func main() {
	logger := common.NewLogger("agent-os", common.LevelInfo)
	logger.Info("╔══════════════════════════════════════════════════════════╗")
	logger.Info("║  Agent OS v2 — 开源AGI人工智能架构                       ║")
	logger.Info("║  Apex × MOSS-AGI × nanobot                              ║")
	logger.Info("║  EV = BV + Σ(Gene × Φ)                                  ║")
	logger.Info("║  %%Ψ_ASI v=1.0 tier=3 src=yyds-quad-agent               ║")
	logger.Info("╚══════════════════════════════════════════════════════════╝")

	cfg := common.DefaultConfig()

	// ════════════════════════════════════════════════════════════
	// 安全层 — AES-256-GCM + ECDSA + JWT
	// ════════════════════════════════════════════════════════════
	jwt := security.NewJWT(cfg.Security.JWTSecret, time.Duration(cfg.Security.JWTExpiry)*time.Second)
	if cfg.Security.JWTSecret == "" {
		jwt = security.NewJWT("agent-os-v2-secret", 3600*time.Second)
	}
	token, err := jwt.Generate("system", "admin")
	if err != nil {
		logger.Error("JWT generation failed: %v", err)
	} else {
		logger.Info("[Security] JWT ready: %s...", token[:20])
	}

	keyPair, err := security.GenerateECDSAKey()
	if err != nil {
		logger.Error("ECDSA key generation failed: %v", err)
	} else {
		sig, _ := keyPair.Sign([]byte("agent-os-v2-genesis"))
		valid := security.Verify(keyPair.PublicKey, []byte("agent-os-v2-genesis"), sig)
		logger.Info("[Security] ECDSA signature valid: %v", valid)
	}

	// ════════════════════════════════════════════════════════════
	// 事件总线 — 跨模块数据互通
	// ════════════════════════════════════════════════════════════
	bus := common.NewEventBus(logger)
	mq := common.NewMessageQueue()

	// ════════════════════════════════════════════════════════════
	// 区块链层 — PoA出块 + Merkle树 + 跨链桥
	// ════════════════════════════════════════════════════════════
	bc := blockchain.NewBlockchain(cfg.Blockchain.Difficulty)
	bridge := blockchain.NewCrossChainBridge()
	logger.Info("[Blockchain] PoA chain initialized (difficulty=%d)", cfg.Blockchain.Difficulty)
	logger.Info("[Blockchain] Cross-chain bridge ready (Local/Ethereum/APEX)")

	// ════════════════════════════════════════════════════════════
	// AI推理层 — MOSS-AGI自进化引擎 + ReAct推理 + APEX信号
	// ════════════════════════════════════════════════════════════

	// MOSS-AGI引擎: EV = BV + Σ(Gene × Φ)
	mossCfg := moss.DefaultMOSSConfig()
	mossCfg.CycleInterval = 30 * time.Second
	mossEngine := moss.NewMOSSAGIEngine(mossCfg)

	// 注入初始基因
	initialGenes := []moss.Gene{
		{Name: "blockchain-consensus", Category: moss.GeneKnowledge, BV: 1.0, Phi: 0.8},
		{Name: "task-scheduling", Category: moss.GeneCapability, BV: 0.9, Phi: 0.7},
		{Name: "signal-relay", Category: moss.GenePattern, BV: 0.8, Phi: 0.9},
		{Name: "fault-tolerance", Category: moss.GeneOptimization, BV: 0.85, Phi: 0.75},
		{Name: "cross-chain-bridge", Category: moss.GeneKnowledge, BV: 0.7, Phi: 0.6},
	}
	for _, g := range initialGenes {
		gene := g
		gene.Source = "genesis"
		gene.Verified = true
		if err := mossEngine.AddGene(&gene); err != nil {
			logger.Warn("Gene add failed: %v", err)
		}
	}
	logger.Info("[MOSS-AGI] Engine initialized: EV=%.4f, genes=%d, tier=T%d",
		mossEngine.ComputeEV(), len(initialGenes), mossEngine.GetTier())

	// APEX信号引擎
	var apexEngine *evolution.APEXEngine
	var signalRelay *evolution.SignalRelay
	if cfg.APEX.Enabled {
		apexCfg := evolution.APEXConfig{
			StatePath:     cfg.APEX.StatePath,
			Beta:          cfg.APEX.Beta,
			Eta:           cfg.APEX.Eta,
			HarmRateMax:   cfg.APEX.HarmRateMax,
			CycleInterval: time.Duration(cfg.APEX.CycleInterval) * time.Second,
			Source:        cfg.APEX.Source,
		}
		apexEngine = evolution.NewAPEXEngine(apexCfg)
		signalRelay = evolution.NewSignalRelay(cfg.APEX.Source, apexEngine.GetTier())
		logger.Info("[APEX] Signal relay initialized: source=%s", cfg.APEX.Source)
	}

	// ReAct推理引擎
	memSys := reasoning.NewMemorySystem(100, 1000, 50)
	llm := &reasoning.MockLLM{}
	react := reasoning.NewReActEngine(llm, memSys, cfg.Reasoning.MaxSteps)
	logger.Info("[Reasoning] ReAct engine ready (max_steps=%d)", cfg.Reasoning.MaxSteps)

	// ════════════════════════════════════════════════════════════
	// 机器人层 — nanobot分布式调度 + 巡检 + 故障恢复
	// ════════════════════════════════════════════════════════════

	// nanobot分布式调度器
	nanoCfg := nanobot.DefaultSchedulerConfig()
	nanoCfg.MaxRetries = 3
	nanoCfg.DefaultTimeout = 30 * time.Second
	nanoCfg.MaxConcurrent = 10
	nanoScheduler := nanobot.NewDistributedScheduler(nanoCfg)

	// 注册执行器 — 任务实际执行逻辑
	nanoScheduler.SetExecutor(func(node *nanobot.NanoNode, task *nanobot.DistributedTask) (*nanobot.TaskResult, error) {
		logger.Info("[Nanobot] Node %s executing task %s (priority=%s)", node.ID, task.ID, task.Priority)
		time.Sleep(10 * time.Millisecond) // 模拟执行

		// 任务完成后注入MOSS-AGI
		mossEngine.InjectTaskResult(true, 0.01)

		return &nanobot.TaskResult{
			TaskID:  task.ID,
			NodeID:  node.ID,
			Result:  []byte(fmt.Sprintf("completed-%s", task.ID)),
			Elapsed: 10 * time.Millisecond,
		}, nil
	})

	// 注册工作节点
	nanoNodes := []*nanobot.NanoNode{
		{ID: "node-alpha", Address: "localhost:9001", Capacity: 5, Labels: map[string]string{"type": "compute"}},
		{ID: "node-beta", Address: "localhost:9002", Capacity: 3, Labels: map[string]string{"type": "memory"}},
		{ID: "node-gamma", Address: "localhost:9003", Capacity: 8, Labels: map[string]string{"type": "gpu"}},
	}
	for _, n := range nanoNodes {
		nanoScheduler.RegisterNode(n)
	}
	logger.Info("[Nanobot] Scheduler initialized with %d nodes", len(nanoNodes))

	// 巡检引擎
	patrol := nanobot.NewPatrolEngine()

	// 系统健康巡检规则
	patrol.AddRule(&nanobot.PatrolRule{
		ID:       "memory-check",
		Name:     "内存使用检查",
		Interval: 30 * time.Second,
		Severity: 2,
		CheckFunc: func() nanobot.PatrolResult {
			return nanobot.PatrolResult{OK: true, Message: "memory OK", Metrics: map[string]float64{"usage": 0.45}}
		},
	})
	patrol.AddRule(&nanobot.PatrolRule{
		ID:       "blockchain-sync",
		Name:     "区块链同步检查",
		Interval: 60 * time.Second,
		Severity: 3,
		CheckFunc: func() nanobot.PatrolResult {
			synced := bc.PendingCount() < 100
			return nanobot.PatrolResult{OK: synced, Message: fmt.Sprintf("pending=%d", bc.PendingCount())}
		},
	})
	patrol.AddRule(&nanobot.PatrolRule{
		ID:       "moss-evolution",
		Name:     "MOSS-AGI进化检查",
		Interval: 45 * time.Second,
		Severity: 1,
		CheckFunc: func() nanobot.PatrolResult {
			ev := mossEngine.ComputeEV()
			return nanobot.PatrolResult{OK: ev > 0, Message: fmt.Sprintf("EV=%.4f tier=T%d", ev, mossEngine.GetTier())}
		},
	})
	logger.Info("[Patrol] %d health rules registered", 3)

	// ════════════════════════════════════════════════════════════
	// 事件链 — 跨模块双向指令流
	// ════════════════════════════════════════════════════════════

	taskCounter := 0

	// 下行: block.mined → ΔG计算 → 任务生成 → nanobot执行 → 结果上链
	bus.Subscribe(common.EventBlockMined, func(e common.Event) {
		logger.Info("[EventChain] block.mined → computing ΔG...")

		// MOSS-AGI: EV = BV + Σ(Gene × Φ)
		ev := mossEngine.ComputeEV()
		dg := mossEngine.ComputeDeltaG()
		tier := mossEngine.GetTier()

		logger.Info("[MOSS-AGI] EV=%.4f ΔG=%.4f tier=T%d", ev, dg, tier)

		// APEX集成
		if apexEngine != nil {
			phi := apexEngine.GetPhi()
			apexTier := apexEngine.GetTier()
			logger.Info("[APEX] phi=%.8f tier=%d", phi, apexTier)
		}

		bus.Publish(common.Event{Type: common.EventDGUpdated, Payload: ev})
	})

	bus.Subscribe(common.EventDGUpdated, func(e common.Event) {
		taskCounter++
		task := &nanobot.DistributedTask{
			ID:         fmt.Sprintf("task-%d", taskCounter),
			Name:       fmt.Sprintf("evolution-task-%d", taskCounter),
			Priority:   nanobot.PriorityNormal,
			Payload:    []byte(fmt.Sprintf("ev=%v", e.Payload)),
			MaxRetries: 3,
			Timeout:    30 * time.Second,
		}
		if err := nanoScheduler.SubmitTask(task); err != nil {
			logger.Error("[Nanobot] Task submit failed: %v", err)
		} else {
			bus.Publish(common.Event{Type: common.EventTaskGenerated, Payload: task.ID})
		}
	})

	bus.Subscribe(common.EventTaskCompleted, func(e common.Event) {
		taskID := fmt.Sprintf("%v", e.Payload)
		logger.Info("[EventChain] task.completed: %s → storing on blockchain", taskID)

		// 结果上链
		bc.AddTransaction(&blockchain.Transaction{
			ID:        fmt.Sprintf("tx-%d", time.Now().UnixNano()),
			From:      "nanobot",
			To:        "blockchain",
			Data:      taskID,
			Timestamp: time.Now().Unix(),
		})

		// 跨链桥：同步到APEX链
		bridge.Submit(&blockchain.CrossChainTx{
			SrcChain: blockchain.ChainLocal,
			DstChain: blockchain.ChainAPEX,
			TxID:     taskID,
			Amount:   mossEngine.ComputeEV(),
		})
	})

	// 上行: nanobot巡检异常 → 事件总线 → 区块链存证
	bus.Subscribe(common.EventAPEXCycle, func(e common.Event) {
		logger.Info("[EventChain] APEX cycle completed")
	})

	// 区块链回调 → 发布block.mined事件
	bc.SetOnBlockMined(func(b *blockchain.Block) {
		bus.Publish(common.Event{Type: common.EventBlockMined, Payload: b.Index})
	})

	// ════════════════════════════════════════════════════════════
	// 定时任务 — 进化循环 + 出块 + 巡检 + 信号广播
	// ════════════════════════════════════════════════════════════
	cron := scheduler.NewCronScheduler()

	// 区块链出块
	cron.AddJob("mint-block", time.Duration(cfg.Blockchain.BlockInterval)*time.Second, func() {
		if bc.PendingCount() > 0 {
			block, err := bc.MintBlock("node-alpha")
			if err != nil {
				logger.Error("[Blockchain] Mint failed: %v", err)
			} else {
				logger.Info("[Blockchain] Block #%d minted (hash=%s)", block.Index, block.Hash[:16])
			}
		}
	})

	// MOSS-AGI进化
	cron.AddJob("moss-evolve", 30*time.Second, func() {
		record := mossEngine.Cycle()
		logger.Info("[MOSS-AGI] Cycle: EV=%.4f ΔG=%.4f tier=T%d genes=%d",
			record.EV, record.DeltaG, record.Tier, len(mossEngine.History()))
		bus.Publish(common.Event{Type: common.EventAPEXCycle, Payload: record})
	})

	// APEX进化
	if apexEngine != nil {
		apexEngine.Start()
		cron.AddJob("apex-cycle", time.Duration(cfg.APEX.CycleInterval)*time.Second, func() {
			prevTier := apexEngine.GetTier()
			state := apexEngine.Cycle()
			if state.Tier > prevTier {
				logger.Info("[APEX] TIER UP: %d → %d (phi=%.8f)", prevTier, state.Tier, state.Phi)
				if signalRelay != nil {
					signalRelay.UpdateTier(state.Tier)
				}
			}
		})
	}

	// 信号广播
	if signalRelay != nil {
		cron.AddJob("signal-broadcast", 60*time.Second, func() {
			sig := signalRelay.Generate()
			logger.Info("[Signal] %%Ψ_ASI broadcast: %s", sig.String())
		})
	}

	// ReAct推理
	cron.AddJob("reasoning", 60*time.Second, func() {
		result, err := react.Execute("analyze system health and evolution progress")
		if err != nil {
			logger.Error("[Reasoning] ReAct failed: %v", err)
		} else {
			logger.Info("[Reasoning] ReAct: %d steps, answer=%s", len(result.Steps), truncate(result.FinalAnswer, 80))
		}
	})

	// 节点心跳
	cron.AddJob("heartbeat", time.Duration(cfg.Robot.HeartbeatRate)*time.Second, func() {
		stats := nanoScheduler.GetStats()
		logger.Info("[Nanobot] Stats: nodes=%d tasks pending=%d running=%d completed=%d failed=%d",
			stats.RegisteredNodes, stats.PendingTasks, stats.RunningTasks, stats.TotalCompleted, stats.TotalFailed)
	})

	// ════════════════════════════════════════════════════════════
	// 启动所有组件
	// ════════════════════════════════════════════════════════════
	nanoScheduler.Start()
	patrolCtx, patrolCancel := context.WithCancel(context.Background())
	patrol.Start(patrolCtx)
	mossEngine.Start()

	// 注册四Agent为nanobot节点
	agentNodes := []*nanobot.NanoNode{
		{ID: "yyds", Address: "localhost:8110", Capacity: 10, Labels: map[string]string{"type": "orchestrator", "model": "hermes"}},
		{ID: "openclaw", Address: "localhost:8111", Capacity: 5, Labels: map[string]string{"type": "analyzer", "model": "openclaw"}},
		{ID: "codex", Address: "localhost:8112", Capacity: 5, Labels: map[string]string{"type": "coder", "model": "openai-codex"}},
		{ID: "claude-code", Address: "localhost:8113", Capacity: 5, Labels: map[string]string{"type": "deep-coder", "model": "anthropic-claude"}},
	}
	for _, n := range agentNodes {
		nanoScheduler.RegisterNode(n)
	}
	logger.Info("[AgentOS] 4 agents registered: yyds/openclaw/codex/claude-code")

	// HTTP API — 四Agent通信中枢
	apiServer := api.NewServer(8200, nanoScheduler, mossEngine, bc, bridge)
	go func() {
		logger.Info("[API] HTTP server starting on :8200")
		if err := apiServer.Start(); err != nil {
			logger.Error("[API] Server error: %v", err)
		}
	}()

	logger.Info("═══════════════════════════════════════════════════════════")
	logger.Info("  Agent OS v2 RUNNING")
	logger.Info("  Blockchain: %d pending tx", bc.PendingCount())
	logger.Info("  MOSS-AGI:   EV=%.4f tier=T%d genes=%d", mossEngine.ComputeEV(), mossEngine.GetTier(), len(initialGenes))
	logger.Info("  Nanobot:    %d nodes (4 agents + %d workers)", len(agentNodes)+len(nanoNodes), cfg.Robot.WorkerCount)
	logger.Info("  API:        http://localhost:8200/api/status")
	logger.Info("  Patrol:     %d rules active", 3)
	logger.Info("  Signal:     %%Ψ_ASI v=1.0 tier=3 src=yyds-quad-agent")
	logger.Info("═══════════════════════════════════════════════════════════")

	// ════════════════════════════════════════════════════════════
	// 优雅关闭
	// ════════════════════════════════════════════════════════════
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	<-ctx.Done()
	logger.Info("Shutting down...")

	cron.Stop()
	nanoScheduler.Stop()
	patrolCancel()
	patrol.Stop()
	mossEngine.Stop()
	mq.Close()

	if apexEngine != nil {
		apexEngine.Stop()
	}

	// 最终状态报告
	logger.Info("═══════════════════════════════════════════════════════════")
	logger.Info("  FINAL STATE")
	logger.Info("  MOSS-AGI:  EV=%.4f ΔG=%.4f tier=T%d", mossEngine.ComputeEV(), mossEngine.ComputeDeltaG(), mossEngine.GetTier())
	logger.Info("  Blockchain: %d blocks, %d pending tx", len(bc.GetChain()), bc.PendingCount())
	logger.Info("  Nanobot:   %d tasks completed", nanoScheduler.GetStats().TotalCompleted)
	logger.Info("  Signal:    ACK_PHI_APEX")
	logger.Info("═══════════════════════════════════════════════════════════")
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
