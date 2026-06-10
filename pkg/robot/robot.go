package robot

import (
	"container/heap"
	"context"
	"fmt"
	"sync"
	"time"
)

// TaskStatus represents the status of a task.
type TaskStatus int

const (
	TaskPending TaskStatus = iota
	TaskRunning
	TaskCompleted
	TaskFailed
)

func (s TaskStatus) String() string {
	switch s {
	case TaskPending:
		return "pending"
	case TaskRunning:
		return "running"
	case TaskCompleted:
		return "completed"
	case TaskFailed:
		return "failed"
	default:
		return "unknown"
	}
}

// Task represents a unit of work.
type Task struct {
	ID       string
	Priority int
	Status   TaskStatus
	Payload  interface{}
	Result   interface{}
	CreatedAt time.Time
}

// PriorityQueue implements heap.Interface.
type PriorityQueue []*Task

func (pq PriorityQueue) Len() int            { return len(pq) }
func (pq PriorityQueue) Less(i, j int) bool   { return pq[i].Priority > pq[j].Priority }
func (pq PriorityQueue) Swap(i, j int)        { pq[i], pq[j] = pq[j], pq[i] }
func (pq *PriorityQueue) Push(x interface{})  { *pq = append(*pq, x.(*Task)) }
func (pq *PriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	*pq = old[:n-1]
	return item
}

// TaskFunc is a function that processes a task.
type TaskFunc func(*Task) error

// WorkerPool manages a pool of task workers.
type WorkerPool struct {
	mu       sync.RWMutex
	workers  int
	taskQueue *PriorityQueue
	taskFunc  TaskFunc
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	running  bool
}

// NewWorkerPool creates a new worker pool.
func NewWorkerPool(workers int, taskFunc TaskFunc) *WorkerPool {
	pq := &PriorityQueue{}
	heap.Init(pq)
	ctx, cancel := context.WithCancel(context.Background())
	return &WorkerPool{
		workers:   workers,
		taskQueue: pq,
		taskFunc:  taskFunc,
		ctx:       ctx,
		cancel:    cancel,
	}
}

// Start begins processing tasks.
func (wp *WorkerPool) Start() {
	wp.mu.Lock()
	if wp.running {
		wp.mu.Unlock()
		return
	}
	wp.running = true
	wp.mu.Unlock()

	for i := 0; i < wp.workers; i++ {
		wp.wg.Add(1)
		go wp.worker(i)
	}
}

func (wp *WorkerPool) worker(id int) {
	defer wp.wg.Done()
	for {
		select {
		case <-wp.ctx.Done():
			return
		default:
			wp.mu.Lock()
			if wp.taskQueue.Len() == 0 {
				wp.mu.Unlock()
				time.Sleep(50 * time.Millisecond)
				continue
			}
			task := heap.Pop(wp.taskQueue).(*Task)
			wp.mu.Unlock()

			task.Status = TaskRunning
			if err := wp.taskFunc(task); err != nil {
				task.Status = TaskFailed
			} else {
				task.Status = TaskCompleted
			}
		}
	}
}

// Submit adds a task to the queue.
func (wp *WorkerPool) Submit(task *Task) {
	wp.mu.Lock()
	defer wp.mu.Unlock()
	heap.Push(wp.taskQueue, task)
}

// Stop gracefully shuts down the pool.
func (wp *WorkerPool) Stop() {
	wp.cancel()
	wp.wg.Wait()
	wp.mu.Lock()
	wp.running = false
	wp.mu.Unlock()
}

// PendingCount returns the number of pending tasks.
func (wp *WorkerPool) PendingCount() int {
	wp.mu.RLock()
	defer wp.mu.RUnlock()
	return wp.taskQueue.Len()
}

// NodeStatus represents a nanobot node's status.
type NodeStatus int

const (
	NodeOnline NodeStatus = iota
	NodeOffline
	NodeBusy
)

// Node represents a nanobot worker node.
type Node struct {
	ID        string
	Status    NodeStatus
	LastBeat  time.Time
	TasksDone int
}

// NodeManager manages worker nodes and heartbeats.
type NodeManager struct {
	mu           sync.RWMutex
	nodes        map[string]*Node
	heartbeatTTL time.Duration
}

// NewNodeManager creates a new node manager.
func NewNodeManager(heartbeatTTL time.Duration) *NodeManager {
	return &NodeManager{
		nodes:        make(map[string]*Node),
		heartbeatTTL: heartbeatTTL,
	}
}

// Register adds a node.
func (nm *NodeManager) Register(id string) {
	nm.mu.Lock()
	defer nm.mu.Unlock()
	nm.nodes[id] = &Node{ID: id, Status: NodeOnline, LastBeat: time.Now()}
}

// Heartbeat updates a node's heartbeat.
func (nm *NodeManager) Heartbeat(id string) error {
	nm.mu.Lock()
	defer nm.mu.Unlock()
	node, ok := nm.nodes[id]
	if !ok {
		return fmt.Errorf("node %s not found", id)
	}
	node.LastBeat = time.Now()
	node.Status = NodeOnline
	return nil
}

// Prune removes offline nodes.
func (nm *NodeManager) Prune() int {
	nm.mu.Lock()
	defer nm.mu.Unlock()
	pruned := 0
	for id, node := range nm.nodes {
		if time.Since(node.LastBeat) > nm.heartbeatTTL {
			node.Status = NodeOffline
			delete(nm.nodes, id)
			pruned++
		}
	}
	return pruned
}

// GetNodes returns all registered nodes.
func (nm *NodeManager) GetNodes() []*Node {
	nm.mu.RLock()
	defer nm.mu.RUnlock()
	nodes := make([]*Node, 0, len(nm.nodes))
	for _, n := range nm.nodes {
		nodes = append(nodes, n)
	}
	return nodes
}

// NodeCount returns the number of registered nodes.
func (nm *NodeManager) NodeCount() int {
	nm.mu.RLock()
	defer nm.mu.RUnlock()
	return len(nm.nodes)
}
