package reasoning

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// MemoryType represents the type of memory.
type MemoryType int

const (
	MemoryShortTerm MemoryType = iota
	MemoryLongTerm
	MemoryWorking
)

func (m MemoryType) String() string {
	switch m {
	case MemoryShortTerm:
		return "short_term"
	case MemoryLongTerm:
		return "long_term"
	case MemoryWorking:
		return "working"
	default:
		return "unknown"
	}
}

// MemoryEntry is a single memory item.
type MemoryEntry struct {
	ID        string
	Type      MemoryType
	Content   string
	Metadata  map[string]interface{}
	CreatedAt time.Time
	AccessCount int
}

// MemorySystem manages multi-layer memory.
type MemorySystem struct {
	mu          sync.RWMutex
	shortTerm   []*MemoryEntry
	longTerm    []*MemoryEntry
	working     []*MemoryEntry
	maxShort    int
	maxLong     int
	maxWorking  int
}

// NewMemorySystem creates a new memory system.
func NewMemorySystem(maxShort, maxLong, maxWorking int) *MemorySystem {
	return &MemorySystem{
		maxShort:   maxShort,
		maxLong:    maxLong,
		maxWorking: maxWorking,
	}
}

// Store adds a memory entry.
func (ms *MemorySystem) Store(entry *MemoryEntry) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	entry.CreatedAt = time.Now()
	switch entry.Type {
	case MemoryShortTerm:
		ms.shortTerm = append(ms.shortTerm, entry)
		if len(ms.shortTerm) > ms.maxShort {
			ms.shortTerm = ms.shortTerm[1:]
		}
	case MemoryLongTerm:
		ms.longTerm = append(ms.longTerm, entry)
		if len(ms.longTerm) > ms.maxLong {
			ms.longTerm = ms.longTerm[1:]
		}
	case MemoryWorking:
		ms.working = append(ms.working, entry)
		if len(ms.working) > ms.maxWorking {
			ms.working = ms.working[1:]
		}
	}
}

// Recall retrieves memories of a given type matching a query.
func (ms *MemorySystem) Recall(memType MemoryType, query string, limit int) []*MemoryEntry {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	var source []*MemoryEntry
	switch memType {
	case MemoryShortTerm:
		source = ms.shortTerm
	case MemoryLongTerm:
		source = ms.longTerm
	case MemoryWorking:
		source = ms.working
	}
	var results []*MemoryEntry
	for i := len(source) - 1; i >= 0 && len(results) < limit; i-- {
		if query == "" || contains(source[i].Content, query) {
			source[i].AccessCount++
			results = append(results, source[i])
		}
	}
	return results
}

func contains(s, substr string) bool {
	return len(substr) == 0 || len(s) >= len(substr) && simpleContains(s, substr)
}

func simpleContains(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// Consolidate moves frequently accessed short-term memories to long-term.
func (ms *MemorySystem) Consolidate(threshold int) int {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	moved := 0
	var remaining []*MemoryEntry
	for _, entry := range ms.shortTerm {
		if entry.AccessCount >= threshold {
			entry.Type = MemoryLongTerm
			ms.longTerm = append(ms.longTerm, entry)
			moved++
		} else {
			remaining = append(remaining, entry)
		}
	}
	ms.shortTerm = remaining
	return moved
}

// ReActStep represents one step in the ReAct loop.
type ReActStep int

const (
	StepThink ReActStep = iota
	StepAct
	StepObserve
)

func (s ReActStep) String() string {
	switch s {
	case StepThink:
		return "think"
	case StepAct:
		return "act"
	case StepObserve:
		return "observe"
	default:
		return "unknown"
	}
}

// LLMInterface defines the interface for LLM calls.
type LLMInterface interface {
	Complete(prompt string) (string, error)
}

// MockLLM is a simple mock LLM for testing.
type MockLLM struct{}

// Complete returns a mock response.
func (m *MockLLM) Complete(prompt string) (string, error) {
	return fmt.Sprintf("Mock response for: %s", prompt[:min(50, len(prompt))]), nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ReActResult holds the result of a ReAct cycle.
type ReActResult struct {
	Steps      []ReActCycleStep
	FinalAnswer string
	DeltaG     float64
}

// ReActCycleStep is one step in a ReAct cycle.
type ReActCycleStep struct {
	Step    ReActStep
	Content string
}

// ReActEngine implements the ReAct reasoning loop.
type ReActEngine struct {
	mu           sync.RWMutex
	llm          LLMInterface
	memory       *MemorySystem
	maxSteps     int
	dgThreshold  float64
}

// NewReActEngine creates a new ReAct engine.
func NewReActEngine(llm LLMInterface, memory *MemorySystem, maxSteps int) *ReActEngine {
	return &ReActEngine{llm: llm, memory: memory, maxSteps: maxSteps, dgThreshold: 0.5}
}

// SetDGThreshold sets the DeltaG threshold for strategy adaptation.
func (re *ReActEngine) SetDGThreshold(threshold float64) {
	re.mu.Lock()
	defer re.mu.Unlock()
	re.dgThreshold = threshold
}

// Execute runs a ReAct loop: Think → Act → Observe.
func (re *ReActEngine) Execute(task string) (*ReActResult, error) {
	re.mu.RLock()
	maxSteps := re.maxSteps
	re.mu.RUnlock()

	result := &ReActResult{}
	context := task

	for i := 0; i < maxSteps; i++ {
		// Think
		thought, err := re.llm.Complete(fmt.Sprintf("Think about: %s", context))
		if err != nil {
			return nil, fmt.Errorf("think step failed: %w", err)
		}
		result.Steps = append(result.Steps, ReActCycleStep{Step: StepThink, Content: thought})

		// Act
		action, err := re.llm.Complete(fmt.Sprintf("Act on thought: %s", thought))
		if err != nil {
			return nil, fmt.Errorf("act step failed: %w", err)
		}
		result.Steps = append(result.Steps, ReActCycleStep{Step: StepAct, Content: action})

		// Observe
		observation, err := re.llm.Complete(fmt.Sprintf("Observe result of action: %s", action))
		if err != nil {
			return nil, fmt.Errorf("observe step failed: %w", err)
		}
		result.Steps = append(result.Steps, ReActCycleStep{Step: StepObserve, Content: observation})

		// Store in memory
		re.memory.Store(&MemoryEntry{
			ID:      fmt.Sprintf("react-%d", i),
			Type:    MemoryShortTerm,
			Content: observation,
			Metadata: map[string]interface{}{"task": task, "step": i},
		})

		context = observation

		// Check if we have a final answer
		if isComplete(observation) {
			result.FinalAnswer = observation
			break
		}
	}

	if result.FinalAnswer == "" && len(result.Steps) > 0 {
		result.FinalAnswer = result.Steps[len(result.Steps)-1].Content
	}

	return result, nil
}

func isComplete(observation string) bool {
	// Simple heuristic for completion
	_ = json.Marshal // use encoding/json to avoid unused import
	return len(observation) > 10
}
