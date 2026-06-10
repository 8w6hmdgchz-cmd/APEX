// Package nanobot implements a distributed task scheduling engine with
// patrol-based health checking, exponential-backoff retry, and priority queues.
package nanobot

import (
	"container/heap"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// ---------------------------------------------------------------------------
// Priority
// ---------------------------------------------------------------------------

// TaskPriority represents task importance level.
type TaskPriority int

const (
	PriorityLow      TaskPriority = iota // 0
	PriorityNormal                       // 1
	PriorityHigh                         // 2
	PriorityCritical                     // 3
)

func (p TaskPriority) String() string {
	switch p {
	case PriorityLow:
		return "low"
	case PriorityNormal:
		return "normal"
	case PriorityHigh:
		return "high"
	case PriorityCritical:
		return "critical"
	default:
		return "unknown"
	}
}

// ---------------------------------------------------------------------------
// DistributedTask
// ---------------------------------------------------------------------------

// DistributedTask is a unit of work to be scheduled across nodes.
type DistributedTask struct {
	ID           string
	Name         string
	Priority     TaskPriority
	Payload      []byte
	Status       string // pending, assigned, running, completed, failed
	AssignedNode string
	RetryCount   int
	MaxRetries   int
	CreatedAt    time.Time
	StartedAt    time.Time
	CompletedAt  time.Time
	Timeout      time.Duration
	Result       []byte
	Error        string
}

// TaskResult is sent back by a node when a task finishes.
type TaskResult struct {
	TaskID  string
	NodeID  string
	Result  []byte
	Error   error
	Elapsed time.Duration
}

// ---------------------------------------------------------------------------
// SchedulerConfig
// ---------------------------------------------------------------------------

// SchedulerConfig tunes the distributed scheduler.
type SchedulerConfig struct {
	MaxRetries      int
	DefaultTimeout  time.Duration
	MaxConcurrent   int
	NodeHeartbeat   time.Duration
	TaskQueueSize   int
	HeartbeatExpiry time.Duration // how long before a node is considered dead
}

// DefaultSchedulerConfig returns sensible defaults.
func DefaultSchedulerConfig() SchedulerConfig {
	return SchedulerConfig{
		MaxRetries:      3,
		DefaultTimeout:  30 * time.Second,
		MaxConcurrent:   64,
		NodeHeartbeat:   10 * time.Second,
		TaskQueueSize:   1024,
		HeartbeatExpiry: 30 * time.Second,
	}
}

// ---------------------------------------------------------------------------
// NanoNode
// ---------------------------------------------------------------------------

// Node status constants.
const (
	NodeStatusIdle     = 0
	NodeStatusBusy     = 1
	NodeStatusDraining = 2
	NodeStatusDead     = 3
)

// NanoNode represents a single worker node in the cluster.
type NanoNode struct {
	ID             string
	Address        string
	Status         int
	Capacity       int
	CurrentLoad    int
	LastHeartbeat  time.Time
	TasksCompleted int64
	TasksFailed    int64
	Labels         map[string]string // GPU, CPU, memory tags, etc.
	mu             sync.Mutex
}

// Available returns remaining capacity.
func (n *NanoNode) Available() int {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.Capacity - n.CurrentLoad
}

// IncLoad atomically increments load.
func (n *NanoNode) IncLoad() {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.CurrentLoad++
	if n.CurrentLoad >= n.Capacity {
		n.Status = NodeStatusBusy
	}
}

// DecLoad atomically decrements load.
func (n *NanoNode) DecLoad() {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.CurrentLoad > 0 {
		n.CurrentLoad--
	}
	if n.CurrentLoad < n.Capacity {
		n.Status = NodeStatusIdle
	}
}

// Touch updates the heartbeat timestamp.
func (n *NanoNode) Touch() {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.LastHeartbeat = time.Now()
}

// ---------------------------------------------------------------------------
// Priority queue (heap) for pending tasks
// ---------------------------------------------------------------------------

type taskQueue []*DistributedTask

func (q taskQueue) Len() int { return len(q) }

// Higher priority first; for equal priority, earlier CreatedAt first.
func (q taskQueue) Less(i, j int) bool {
	if q[i].Priority != q[j].Priority {
		return q[i].Priority > q[j].Priority
	}
	return q[i].CreatedAt.Before(q[j].CreatedAt)
}

func (q taskQueue) Swap(i, j int) { q[i], q[j] = q[j], q[i] }

func (q *taskQueue) Push(x interface{}) { *q = append(*q, x.(*DistributedTask)) }

func (q *taskQueue) Pop() interface{} {
	old := *q
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	*q = old[:n-1]
	return item
}

// ---------------------------------------------------------------------------
// SchedulerStats
// ---------------------------------------------------------------------------

// SchedulerStats holds runtime statistics.
type SchedulerStats struct {
	TotalSubmitted  int64
	TotalCompleted  int64
	TotalFailed     int64
	PendingTasks    int
	RunningTasks    int
	RegisteredNodes int
	AliveNodes      int
}

// ---------------------------------------------------------------------------
// DistributedScheduler
// ---------------------------------------------------------------------------

// DistributedScheduler is the central coordinator that assigns tasks to nodes.
type DistributedScheduler struct {
	mu         sync.RWMutex
	tasks      map[string]*DistributedTask
	pending    taskQueue
	nodes      map[string]*NanoNode
	taskChan   chan *DistributedTask
	resultChan chan *TaskResult
	cfg        SchedulerConfig
	running    bool
	ctx        context.Context
	cancel     context.CancelFunc

	// stats counters
	totalSubmitted int64
	totalCompleted int64
	totalFailed    int64

	// executor function — called when a task is dispatched to a node.
	// In production this would be an RPC; here it is a pluggable func.
	executor func(node *NanoNode, task *DistributedTask) (*TaskResult, error)
}

// newID generates a random hex ID (no external dependency).
func newID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// NewDistributedScheduler creates a new scheduler with the given config.
func NewDistributedScheduler(cfg SchedulerConfig) *DistributedScheduler {
	if cfg.TaskQueueSize <= 0 {
		cfg.TaskQueueSize = 1024
	}
	if cfg.MaxConcurrent <= 0 {
		cfg.MaxConcurrent = 64
	}
	if cfg.DefaultTimeout <= 0 {
		cfg.DefaultTimeout = 30 * time.Second
	}
	if cfg.NodeHeartbeat <= 0 {
		cfg.NodeHeartbeat = 10 * time.Second
	}
	if cfg.HeartbeatExpiry <= 0 {
		cfg.HeartbeatExpiry = 30 * time.Second
	}

	s := &DistributedScheduler{
		tasks:      make(map[string]*DistributedTask),
		pending:    make(taskQueue, 0),
		nodes:      make(map[string]*NanoNode),
		taskChan:   make(chan *DistributedTask, cfg.TaskQueueSize),
		resultChan: make(chan *TaskResult, cfg.TaskQueueSize),
		cfg:        cfg,
	}
	heap.Init(&s.pending)

	// default in-process executor (sleeps to simulate work)
	s.executor = func(node *NanoNode, task *DistributedTask) (*TaskResult, error) {
		// Placeholder: real implementation would do RPC.
		return &TaskResult{
			TaskID: task.ID,
			NodeID: node.ID,
			Result: task.Payload, // echo back
		}, nil
	}

	return s
}

// SetExecutor overrides the default task executor (useful for testing).
func (s *DistributedScheduler) SetExecutor(fn func(node *NanoNode, task *DistributedTask) (*TaskResult, error)) {
	s.executor = fn
}

// RegisterNode adds (or updates) a worker node.
func (s *DistributedScheduler) RegisterNode(n *NanoNode) {
	s.mu.Lock()
	defer s.mu.Unlock()
	n.Touch()
	s.nodes[n.ID] = n
}

// UnregisterNode removes a node.
func (s *DistributedScheduler) UnregisterNode(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.nodes, id)
}

// SubmitTask enqueues a task for scheduling.
func (s *DistributedScheduler) SubmitTask(t *DistributedTask) error {
	if t.ID == "" {
		t.ID = newID()
	}
	if t.Status == "" {
		t.Status = "pending"
	}
	if t.CreatedAt.IsZero() {
		t.CreatedAt = time.Now()
	}
	if t.Timeout <= 0 {
		t.Timeout = s.cfg.DefaultTimeout
	}

	s.mu.Lock()
	s.tasks[t.ID] = t
	heap.Push(&s.pending, t)
	atomic.AddInt64(&s.totalSubmitted, 1)
	s.mu.Unlock()

	return nil
}

// Start launches the scheduler goroutines.
func (s *DistributedScheduler) Start() {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.ctx, s.cancel = context.WithCancel(context.Background())
	s.running = true
	s.mu.Unlock()

	go s.dispatchLoop()
	go s.resultLoop()
	go s.heartbeatLoop()
}

// Stop gracefully shuts down the scheduler.
func (s *DistributedScheduler) Stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	s.running = false
	s.cancel()
	s.mu.Unlock()
}

// GetStats returns a snapshot of scheduler statistics.
func (s *DistributedScheduler) GetStats() SchedulerStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var alive int
	for _, n := range s.nodes {
		if n.Status != NodeStatusDead {
			alive++
		}
	}

	var running int
	for _, t := range s.tasks {
		if t.Status == "running" || t.Status == "assigned" {
			running++
		}
	}

	return SchedulerStats{
		TotalSubmitted:  atomic.LoadInt64(&s.totalSubmitted),
		TotalCompleted:  atomic.LoadInt64(&s.totalCompleted),
		TotalFailed:     atomic.LoadInt64(&s.totalFailed),
		PendingTasks:    s.pending.Len(),
		RunningTasks:    running,
		RegisteredNodes: len(s.nodes),
		AliveNodes:      alive,
	}
}

// OnTaskComplete is called externally when a task finishes (alternative to resultChan).
func (s *DistributedScheduler) OnTaskComplete(taskID string, result []byte, err error) {
	r := &TaskResult{TaskID: taskID, Result: result}
	if err != nil {
		r.Error = err
	}
	select {
	case s.resultChan <- r:
	default:
		// channel full — handle synchronously
		s.handleResult(r)
	}
}

// GetTask returns a task by ID.
func (s *DistributedScheduler) GetTask(id string) *DistributedTask {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.tasks[id]
}

// ListTasks returns all tasks, optionally filtered by status.
func (s *DistributedScheduler) ListTasks(status string) []*DistributedTask {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*DistributedTask
	for _, t := range s.tasks {
		if status == "" || t.Status == status {
			cp := *t
			out = append(out, &cp)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out
}

// --- internal goroutines ---------------------------------------------------

func (s *DistributedScheduler) dispatchLoop() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	sem := make(chan struct{}, s.cfg.MaxConcurrent)

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.mu.Lock()
			for s.pending.Len() > 0 {
				node := s.pickNode()
				if node == nil {
					break // no available node
				}
				task := heap.Pop(&s.pending).(*DistributedTask)
				task.Status = "assigned"
				task.AssignedNode = node.ID
				task.StartedAt = time.Now()
				node.IncLoad()
				sem <- struct{}{}
				go func(t *DistributedTask, n *NanoNode) {
					defer func() { <-sem }()
					s.executeTask(t, n)
				}(task, node)
			}
			s.mu.Unlock()
		}
	}
}

func (s *DistributedScheduler) executeTask(task *DistributedTask, node *NanoNode) {
	task.Status = "running"

	type execResult struct {
		res *TaskResult
		err error
	}
	ch := make(chan execResult, 1)
	go func() {
		res, err := s.executor(node, task)
		ch <- execResult{res, err}
	}()

	select {
	case <-s.ctx.Done():
		return
	case r := <-ch:
		if r.err != nil {
			r.res = &TaskResult{TaskID: task.ID, NodeID: node.ID, Error: r.err}
		}
		s.resultChan <- r.res
	case <-time.After(task.Timeout):
		s.resultChan <- &TaskResult{
			TaskID: task.ID,
			NodeID: node.ID,
			Error:  fmt.Errorf("task %s timed out after %s", task.ID, task.Timeout),
		}
	}
}

func (s *DistributedScheduler) resultLoop() {
	for {
		select {
		case <-s.ctx.Done():
			return
		case r := <-s.resultChan:
			s.handleResult(r)
		}
	}
}

func (s *DistributedScheduler) handleResult(r *TaskResult) {
	s.mu.Lock()
	defer s.mu.Unlock()

	task, ok := s.tasks[r.TaskID]
	if !ok {
		return
	}

	// release node load
	if node, exists := s.nodes[r.NodeID]; exists {
		node.DecLoad()
	}

	if r.Error != nil {
		task.RetryCount++
		task.Error = r.Error.Error()

		if task.RetryCount <= task.MaxRetries {
			// re-enqueue for retry
			task.Status = "pending"
			task.AssignedNode = ""
			heap.Push(&s.pending, task)
		} else {
			task.Status = "failed"
			task.CompletedAt = time.Now()
			atomic.AddInt64(&s.totalFailed, 1)
			if node, exists := s.nodes[r.NodeID]; exists {
				atomic.AddInt64(&node.TasksFailed, 1)
			}
		}
	} else {
		task.Status = "completed"
		task.Result = r.Result
		task.CompletedAt = time.Now()
		atomic.AddInt64(&s.totalCompleted, 1)
		if node, exists := s.nodes[r.NodeID]; exists {
			atomic.AddInt64(&node.TasksCompleted, 1)
		}
	}
}

func (s *DistributedScheduler) heartbeatLoop() {
	ticker := time.NewTicker(s.cfg.NodeHeartbeat)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.mu.Lock()
			now := time.Now()
			for _, n := range s.nodes {
				n.mu.Lock()
				if now.Sub(n.LastHeartbeat) > s.cfg.HeartbeatExpiry {
					n.Status = NodeStatusDead
				}
				n.mu.Unlock()
			}
			s.mu.Unlock()
		}
	}
}

// pickNode selects the best available node (most available capacity).
func (s *DistributedScheduler) pickNode() *NanoNode {
	var best *NanoNode
	bestAvail := 0
	for _, n := range s.nodes {
		if n.Status == NodeStatusDead || n.Status == NodeStatusDraining {
			continue
		}
		avail := n.Available()
		if avail > bestAvail {
			bestAvail = avail
			best = n
		}
	}
	return best
}

// ---------------------------------------------------------------------------
// RetryPolicy — exponential back-off
// ---------------------------------------------------------------------------

// RetryPolicy defines how retries are spaced.
type RetryPolicy struct {
	MaxRetries int
	BaseDelay  time.Duration
	MaxDelay   time.Duration
	Multiplier float64
}

// DefaultRetryPolicy returns a policy with sensible defaults.
func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{
		MaxRetries: 3,
		BaseDelay:  500 * time.Millisecond,
		MaxDelay:   30 * time.Second,
		Multiplier: 2.0,
	}
}

// Execute runs fn up to MaxRetries+1 times with exponential back-off.
func (p *RetryPolicy) Execute(fn func() error) error {
	var lastErr error
	delay := p.BaseDelay

	for attempt := 0; attempt <= p.MaxRetries; attempt++ {
		if err := fn(); err != nil {
			lastErr = err
			if attempt < p.MaxRetries {
				time.Sleep(delay)
				delay = time.Duration(float64(delay) * p.Multiplier)
				if p.MaxDelay > 0 && delay > p.MaxDelay {
					delay = p.MaxDelay
				}
			}
		} else {
			return nil
		}
	}
	return fmt.Errorf("all %d retries exhausted: %w", p.MaxRetries, lastErr)
}

// ---------------------------------------------------------------------------
// PatrolEngine — health-check / patrol framework
// ---------------------------------------------------------------------------

// PatrolRule defines a single health-check rule.
type PatrolRule struct {
	ID        string
	Name      string
	CheckFunc func() PatrolResult
	Interval  time.Duration
	Severity  int // 0=info, 1=warn, 2=critical
}

// PatrolResult is the output of a single check execution.
type PatrolResult struct {
	OK      bool
	Message string
	Metrics map[string]float64
}

// PatrolRecord stores a historical check result.
type PatrolRecord struct {
	RuleID    string
	OK        bool
	Message   string
	Timestamp time.Time
}

// PatrolAlert is emitted when a check fails.
type PatrolAlert struct {
	RuleID    string
	Result    PatrolRecord
	Timestamp time.Time
}

// PatrolEngine runs periodic health checks and emits alerts.
type PatrolEngine struct {
	mu       sync.RWMutex
	rules    map[string]*PatrolRule
	results  []PatrolRecord
	alerts   []PatrolAlert
	alertChan chan PatrolAlert
	running  bool
	ctx      context.Context
	cancel   context.CancelFunc
}

// NewPatrolEngine creates a new patrol engine.
func NewPatrolEngine() *PatrolEngine {
	return &PatrolEngine{
		rules:     make(map[string]*PatrolRule),
		alertChan: make(chan PatrolAlert, 256),
	}
}

// AddRule registers a patrol rule.
func (p *PatrolEngine) AddRule(rule *PatrolRule) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if rule.ID == "" {
		rule.ID = newID()
	}
	p.rules[rule.ID] = rule
}

// RemoveRule unregisters a patrol rule.
func (p *PatrolEngine) RemoveRule(id string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.rules, id)
}

// Start launches patrol goroutines.
func (p *PatrolEngine) Start(ctx context.Context) {
	p.mu.Lock()
	if p.running {
		p.mu.Unlock()
		return
	}
	p.ctx, p.cancel = context.WithCancel(ctx)
	p.running = true
	rules := make([]*PatrolRule, 0, len(p.rules))
	for _, r := range p.rules {
		rules = append(rules, r)
	}
	p.mu.Unlock()

	for _, rule := range rules {
		go p.runRule(rule)
	}
}

// Stop halts all patrol goroutines.
func (p *PatrolEngine) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.running {
		return
	}
	p.running = false
	p.cancel()
}

// GetAlerts returns accumulated alerts and clears the buffer.
func (p *PatrolEngine) GetAlerts() []PatrolAlert {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]PatrolAlert, len(p.alerts))
	copy(out, p.alerts)
	p.alerts = p.alerts[:0]
	return out
}

// AlertChan returns a read-only channel of live alerts.
func (p *PatrolEngine) AlertChan() <-chan PatrolAlert {
	return p.alertChan
}

// GetResults returns recent patrol records.
func (p *PatrolEngine) GetResults(limit int) []PatrolRecord {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if limit <= 0 || limit > len(p.results) {
		limit = len(p.results)
	}
	start := len(p.results) - limit
	out := make([]PatrolRecord, limit)
	copy(out, p.results[start:])
	return out
}

func (p *PatrolEngine) runRule(rule *PatrolRule) {
	ticker := time.NewTicker(rule.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			result := rule.CheckFunc()
			rec := PatrolRecord{
				RuleID:    rule.ID,
				OK:        result.OK,
				Message:   result.Message,
				Timestamp: time.Now(),
			}

			p.mu.Lock()
			p.results = append(p.results, rec)
			// keep at most 1000 records
			if len(p.results) > 1000 {
				p.results = p.results[len(p.results)-1000:]
			}
			if !result.OK {
				alert := PatrolAlert{
					RuleID:    rule.ID,
					Result:    rec,
					Timestamp: rec.Timestamp,
				}
				p.alerts = append(p.alerts, alert)
				select {
				case p.alertChan <- alert:
				default:
				}
			}
			p.mu.Unlock()
		}
	}
}

// RunOnce executes all rules once and returns results (useful for testing / CLI).
func (p *PatrolEngine) RunOnce() []PatrolRecord {
	p.mu.RLock()
	rules := make([]*PatrolRule, 0, len(p.rules))
	for _, r := range p.rules {
		rules = append(rules, r)
	}
	p.mu.RUnlock()

	var records []PatrolRecord
	for _, rule := range rules {
		result := rule.CheckFunc()
		rec := PatrolRecord{
			RuleID:    rule.ID,
			OK:        result.OK,
			Message:   result.Message,
			Timestamp: time.Now(),
		}
		records = append(records, rec)
	}
	return records
}

// Ensure heap interface is satisfied at compile time.
var _ heap.Interface = (*taskQueue)(nil)

// avoid unused import
var _ = math.MaxFloat64
