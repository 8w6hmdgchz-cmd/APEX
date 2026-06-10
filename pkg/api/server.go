// Package api provides HTTP API for Agent OS v2.
// 四Agent通过此API提交任务、查询状态、获取结果。
package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/nousresearch/agent-os-v2/pkg/blockchain"
	"github.com/nousresearch/agent-os-v2/pkg/moss"
	"github.com/nousresearch/agent-os-v2/pkg/nanobot"
)

// Server is the HTTP API server for Agent OS v2.
type Server struct {
	mu           sync.RWMutex
	scheduler    *nanobot.DistributedScheduler
	mossEngine   *moss.MOSSAGIEngine
	blockchain   *blockchain.Blockchain
	bridge       *blockchain.CrossChainBridge
	taskResults  map[string]*TaskResultRecord
	agentStats   map[string]*AgentStat
	port         int
}

// TaskResultRecord stores the result of a completed task.
type TaskResultRecord struct {
	TaskID    string    `json:"task_id"`
	AgentID   string    `json:"agent_id"`
	Success   bool      `json:"success"`
	Data      string    `json:"data"`
	Error     string    `json:"error,omitempty"`
	Duration  int64     `json:"duration_ms"`
	Timestamp time.Time `json:"timestamp"`
	TxHash    string    `json:"tx_hash,omitempty"`
}

// AgentStat tracks per-agent statistics.
type AgentStat struct {
	AgentID        string  `json:"agent_id"`
	TasksSubmitted int64   `json:"tasks_submitted"`
	TasksCompleted int64   `json:"tasks_completed"`
	TasksFailed    int64   `json:"tasks_failed"`
	TotalEV        float64 `json:"total_ev"`
	LastActive     time.Time `json:"last_active"`
}

// TaskRequest is the incoming task submission.
type TaskRequest struct {
	AgentID  string `json:"agent_id"`
	TaskName string `json:"task_name"`
	Priority int    `json:"priority"`
	Payload  string `json:"payload"`
	Timeout  int    `json:"timeout_seconds"`
}

// TaskResponse is the response after task submission.
type TaskResponse struct {
	TaskID  string `json:"task_id"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

// NewServer creates a new API server.
func NewServer(port int, scheduler *nanobot.DistributedScheduler, mossEngine *moss.MOSSAGIEngine, bc *blockchain.Blockchain, bridge *blockchain.CrossChainBridge) *Server {
	return &Server{
		scheduler:   scheduler,
		mossEngine:  mossEngine,
		blockchain:  bc,
		bridge:      bridge,
		taskResults: make(map[string]*TaskResultRecord),
		agentStats:  make(map[string]*AgentStat),
		port:        port,
	}
}

// Start begins serving the HTTP API.
func (s *Server) Start() error {
	mux := http.NewServeMux()

	// 任务API
	mux.HandleFunc("/api/task/submit", s.handleSubmitTask)
	mux.HandleFunc("/api/task/result", s.handleGetTaskResult)
	mux.HandleFunc("/api/task/callback", s.handleTaskCallback)

	// Agent API
	mux.HandleFunc("/api/agent/register", s.handleRegisterAgent)
	mux.HandleFunc("/api/agent/stats", s.handleAgentStats)

	// 系统状态API
	mux.HandleFunc("/api/status", s.handleStatus)
	mux.HandleFunc("/api/evolution", s.handleEvolution)
	mux.HandleFunc("/api/blockchain", s.handleBlockchain)
	mux.HandleFunc("/api/health", s.handleHealth)

	addr := fmt.Sprintf(":%d", s.port)
	return http.ListenAndServe(addr, mux)
}

// POST /api/task/submit — Agent提交任务
func (s *Server) handleSubmitTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req TaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.AgentID == "" {
		req.AgentID = "unknown"
	}
	if req.TaskName == "" {
		req.TaskName = "unnamed-task"
	}
	if req.Timeout == 0 {
		req.Timeout = 30
	}

	// 创建nanobot任务
	task := &nanobot.DistributedTask{
		Name:       req.TaskName,
		Priority:   nanobot.TaskPriority(req.Priority),
		Payload:    []byte(req.Payload),
		MaxRetries: 3,
		Timeout:    time.Duration(req.Timeout) * time.Second,
	}

	if err := s.scheduler.SubmitTask(task); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 更新Agent统计
	s.mu.Lock()
	stat, ok := s.agentStats[req.AgentID]
	if !ok {
		stat = &AgentStat{AgentID: req.AgentID}
		s.agentStats[req.AgentID] = stat
	}
	stat.TasksSubmitted++
	stat.LastActive = time.Now()
	s.mu.Unlock()

	resp := TaskResponse{
		TaskID:  task.ID,
		Status:  "submitted",
		Message: fmt.Sprintf("Task %s submitted by %s", task.ID, req.AgentID),
	}
	json.NewEncoder(w).Encode(resp)
}

// POST /api/task/callback — 任务完成回调
func (s *Server) handleTaskCallback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var record TaskResultRecord
	if err := json.NewDecoder(r.Body).Decode(&record); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	record.Timestamp = time.Now()

	// 存储结果
	s.mu.Lock()
	s.taskResults[record.TaskID] = &record

	// 更新Agent统计
	stat, ok := s.agentStats[record.AgentID]
	if !ok {
		stat = &AgentStat{AgentID: record.AgentID}
		s.agentStats[record.AgentID] = stat
	}
	if record.Success {
		stat.TasksCompleted++
	} else {
		stat.TasksFailed++
	}
	stat.LastActive = time.Now()
	s.mu.Unlock()

	// 注入MOSS-AGI进化引擎
	s.mossEngine.InjectTaskResult(record.Success, 0.01)

	// 结果上链
	s.blockchain.AddTransaction(&blockchain.Transaction{
		ID:        fmt.Sprintf("tx-%s-%d", record.TaskID, time.Now().UnixNano()),
		From:      record.AgentID,
		To:        "blockchain",
		Data:      fmt.Sprintf("%s:%v:%s", record.TaskID, record.Success, record.Data),
		Timestamp: time.Now().Unix(),
	})

	// 跨链同步
	s.bridge.Submit(&blockchain.CrossChainTx{
		SrcChain: blockchain.ChainLocal,
		DstChain: blockchain.ChainAPEX,
		TxID:     record.TaskID,
		Amount:   s.mossEngine.ComputeEV(),
	})

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "recorded"})
}

// GET /api/task/result?task_id=xxx — 查询任务结果
func (s *Server) handleGetTaskResult(w http.ResponseWriter, r *http.Request) {
	taskID := r.URL.Query().Get("task_id")
	if taskID == "" {
		http.Error(w, "task_id required", http.StatusBadRequest)
		return
	}

	s.mu.RLock()
	record, ok := s.taskResults[taskID]
	s.mu.RUnlock()

	if !ok {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(record)
}

// POST /api/agent/register — 注册Agent为nanobot节点
func (s *Server) handleRegisterAgent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		AgentID  string            `json:"agent_id"`
		Address  string            `json:"address"`
		Capacity int               `json:"capacity"`
		Labels   map[string]string `json:"labels"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	node := &nanobot.NanoNode{
		ID:       req.AgentID,
		Address:  req.Address,
		Capacity: req.Capacity,
		Labels:   req.Labels,
	}
	s.scheduler.RegisterNode(node)

	// 初始化Agent统计
	s.mu.Lock()
	s.agentStats[req.AgentID] = &AgentStat{
		AgentID:    req.AgentID,
		LastActive: time.Now(),
	}
	s.mu.Unlock()

	json.NewEncoder(w).Encode(map[string]string{
		"status":  "registered",
		"agent_id": req.AgentID,
	})
}

// GET /api/agent/stats — 获取所有Agent统计
func (s *Server) handleAgentStats(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	stats := make([]*AgentStat, 0, len(s.agentStats))
	for _, v := range s.agentStats {
		stats = append(stats, v)
	}
	s.mu.RUnlock()

	json.NewEncoder(w).Encode(stats)
}

// GET /api/status — 系统总览
func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	schedStats := s.scheduler.GetStats()
	ev := s.mossEngine.ComputeEV()
	tier := s.mossEngine.GetTier()

	status := map[string]interface{}{
		"status":     "running",
		"version":    "2.0.0",
		"signal":     "%Ψ_ASI v=1.0 tier=3 src=yyds-quad-agent",
		"evolution": map[string]interface{}{
			"ev":   ev,
			"tier": tier,
			"dg":   s.mossEngine.ComputeDeltaG(),
		},
		"scheduler": map[string]interface{}{
			"nodes":          schedStats.RegisteredNodes,
			"pending_tasks":  schedStats.PendingTasks,
			"running_tasks":  schedStats.RunningTasks,
			"completed":      schedStats.TotalCompleted,
			"failed":         schedStats.TotalFailed,
		},
		"blockchain": map[string]interface{}{
			"blocks":     len(s.blockchain.GetChain()),
			"pending_tx": s.blockchain.PendingCount(),
		},
		"agents": len(s.agentStats),
	}
	json.NewEncoder(w).Encode(status)
}

// GET /api/evolution — 进化详情
func (s *Server) handleEvolution(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(s.mossEngine.GetState())
}

// GET /api/blockchain — 区块链状态
func (s *Server) handleBlockchain(w http.ResponseWriter, r *http.Request) {
	chain := s.blockchain.GetChain()
	blocks := make([]map[string]interface{}, len(chain))
	for i, b := range chain {
		blocks[i] = map[string]interface{}{
			"index":     b.Index,
			"hash":      b.Hash[:16],
			"prev_hash": b.PrevHash[:16],
			"tx_count":  len(b.Transactions),
			"timestamp": b.Timestamp,
			"validator": b.Validator,
		}
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"blocks":     blocks,
		"pending_tx": s.blockchain.PendingCount(),
		"bridge": map[string]interface{}{
			"pending":   s.bridge.PendingCount(),
			"completed": s.bridge.CompletedCount(),
		},
	})
}

// GET /api/health — 健康检查
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
