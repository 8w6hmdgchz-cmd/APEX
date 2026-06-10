// Package moss implements the MOSS-AGI Self-Evolution Engine.
//
// Core formula: EV = BV + Σ(Gene × Φ)
// Evolution:    ΔG = (EV_t - EV_{t-1}) / EV_{t-1}
// Tier:         T1–T5 based on EV thresholds.
package moss

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"sort"
	"sync"
	"time"
)

// ---------------------------------------------------------------------------
// Gene — knowledge / capability unit
// ---------------------------------------------------------------------------

// GeneCategory classifies a gene.
type GeneCategory string

const (
	GeneKnowledge    GeneCategory = "knowledge"
	GeneCapability   GeneCategory = "capability"
	GeneOptimization GeneCategory = "optimization"
	GenePattern      GeneCategory = "pattern"
)

// Gene is the atomic unit of value in the MOSS gene pool.
type Gene struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Category    GeneCategory `json:"category"`
	BV          float64      `json:"bv"`  // base value
	Phi         float64      `json:"phi"` // Φ weight
	DeltaG      float64      `json:"delta_g"`
	Source      string       `json:"source"`
	Verified    bool         `json:"verified"`
	CreatedAt   time.Time    `json:"created_at"`
	AppliedCount int         `json:"applied_count"`
}

// EffectiveValue returns BV × Φ.
func (g *Gene) EffectiveValue() float64 {
	return g.BV * g.Phi
}

// ---------------------------------------------------------------------------
// GenePool — pool management with crossover / mutation
// ---------------------------------------------------------------------------

// GenePool manages a bounded collection of genes.
type GenePool struct {
	mu         sync.RWMutex
	genes      map[string]*Gene
	maxCapacity int
}

// NewGenePool creates a pool with the given max capacity.
func NewGenePool(capacity int) *GenePool {
	return &GenePool{
		genes:       make(map[string]*Gene),
		maxCapacity: capacity,
	}
}

var (
	ErrGeneExists   = errors.New("moss: gene already exists")
	ErrGeneNotFound = errors.New("moss: gene not found")
	ErrPoolFull     = errors.New("moss: gene pool at capacity")
	ErrNilGene      = errors.New("moss: gene is nil")
)

// Add inserts a gene after dedup + capacity checks.
func (p *GenePool) Add(g *Gene) error {
	if g == nil {
		return ErrNilGene
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, ok := p.genes[g.ID]; ok {
		return ErrGeneExists
	}
	if len(p.genes) >= p.maxCapacity {
		return ErrPoolFull
	}
	// store a copy
	cp := *g
	p.genes[g.ID] = &cp
	return nil
}

// Remove deletes a gene by ID.
func (p *GenePool) Remove(id string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, ok := p.genes[id]; !ok {
		return ErrGeneNotFound
	}
	delete(p.genes, id)
	return nil
}

// Get returns a gene by ID.
func (p *GenePool) Get(id string) (*Gene, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	g, ok := p.genes[id]
	return g, ok
}

// All returns a snapshot slice of every gene.
func (p *GenePool) All() []*Gene {
	p.mu.RLock()
	defer p.mu.RUnlock()
	out := make([]*Gene, 0, len(p.genes))
	for _, g := range p.genes {
		cp := *g
		out = append(out, &cp)
	}
	return out
}

// Len returns current pool size.
func (p *GenePool) Len() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.genes)
}

// Select returns the top-n genes ranked by EffectiveValue (BV×Φ).
func (p *GenePool) Select(n int) []*Gene {
	p.mu.RLock()
	defer p.mu.RUnlock()

	all := make([]*Gene, 0, len(p.genes))
	for _, g := range p.genes {
		cp := *g
		all = append(all, &cp)
	}
	sort.Slice(all, func(i, j int) bool {
		return all[i].EffectiveValue() > all[j].EffectiveValue()
	})
	if n > len(all) {
		n = len(all)
	}
	return all[:n]
}

// Crossover produces a child gene from two parents.
// The child inherits averaged BV/Phi with a small random perturbation.
func (p *GenePool) Crossover(a, b *Gene) *Gene {
	if a == nil || b == nil {
		return nil
	}
	child := &Gene{
		ID:       fmt.Sprintf("cross_%s_%s_%d", a.ID, b.ID, time.Now().UnixNano()),
		Name:     fmt.Sprintf("cross(%s,%s)", a.Name, b.Name),
		Category: a.Category,
		BV:       (a.BV+b.BV)/2 + (rand.Float64()-0.5)*0.1,
		Phi:      (a.Phi+b.Phi)/2 + (rand.Float64()-0.5)*0.02,
		Source:   "crossover",
		CreatedAt: time.Now(),
	}
	if child.BV < 0 {
		child.BV = 0
	}
	if child.Phi < 0.01 {
		child.Phi = 0.01
	}
	return child
}

// Mutate returns a mutated copy of a gene.
func (p *GenePool) Mutate(g *Gene) *Gene {
	if g == nil {
		return nil
	}
	mut := *g
	mut.ID = fmt.Sprintf("mut_%s_%d", g.ID, time.Now().UnixNano())
	mut.Name = g.Name + "+"
	mut.Source = "mutation"
	mut.CreatedAt = time.Now()
	mut.AppliedCount = 0
	mut.Verified = false

	// perturb BV ±20%
	mut.BV = g.BV * (1 + (rand.Float64()-0.5)*0.4)
	if mut.BV < 0 {
		mut.BV = 0
	}
	// perturb Phi ±10%
	mut.Phi = g.Phi * (1 + (rand.Float64()-0.5)*0.2)
	if mut.Phi < 0.01 {
		mut.Phi = 0.01
	}
	return &mut
}

// ---------------------------------------------------------------------------
// DeltaGRecord — snapshot per evolution cycle
// ---------------------------------------------------------------------------

// DeltaGRecord captures the state at the end of a cycle.
type DeltaGRecord struct {
	Timestamp time.Time `json:"timestamp"`
	EV        float64   `json:"ev"`
	BV        float64   `json:"bv"`
	PhiSum    float64   `json:"phi_sum"`
	DeltaG    float64   `json:"delta_g"`
	Tier      int       `json:"tier"`
}

// ---------------------------------------------------------------------------
// MOSSConfig
// ---------------------------------------------------------------------------

// MOSSConfig tunes the evolution engine.
type MOSSConfig struct {
	GenePoolSize         int           `json:"gene_pool_size"`
	MutationRate         float64       `json:"mutation_rate"`   // 0–1
	CrossoverRate        float64       `json:"crossover_rate"`  // 0–1
	PhiAdaptRate         float64       `json:"phi_adapt_rate"`  // default 0.05
	TaskSuccessThreshold float64       `json:"task_success_threshold"`
	CycleInterval        time.Duration `json:"cycle_interval"`
}

// DefaultMOSSConfig returns sensible defaults.
func DefaultMOSSConfig() MOSSConfig {
	return MOSSConfig{
		GenePoolSize:         1000,
		MutationRate:         0.1,
		CrossoverRate:        0.05,
		PhiAdaptRate:         0.05,
		TaskSuccessThreshold: 0.6,
		CycleInterval:        30 * time.Second,
	}
}

// ---------------------------------------------------------------------------
// MOSSAGIEngine — the self-evolution core
// ---------------------------------------------------------------------------

// MOSSAGIEngine is the central self-evolution engine.
type MOSSAGIEngine struct {
	mu          sync.RWMutex
	pool        *GenePool
	bv          float64   // base value synced from EVM
	phiSum      float64   // Σ(Gene × Φ)
	ev          float64   // EV = BV + Σ(Gene × Φ)
	prevEV      float64   // EV from previous cycle
	deltaGHistory []DeltaGRecord
	tier        int
	running     bool
	cfg         MOSSConfig
	stopCh      chan struct{}
}

// NewMOSSAGIEngine creates a new engine with the given config.
func NewMOSSAGIEngine(cfg MOSSConfig) *MOSSAGIEngine {
	if cfg.GenePoolSize <= 0 {
		cfg.GenePoolSize = 1000
	}
	if cfg.PhiAdaptRate <= 0 {
		cfg.PhiAdaptRate = 0.05
	}
	if cfg.CycleInterval <= 0 {
		cfg.CycleInterval = 30 * time.Second
	}
	return &MOSSAGIEngine{
		pool:   NewGenePool(cfg.GenePoolSize),
		cfg:    cfg,
		stopCh: make(chan struct{}),
	}
}

// SetBV updates the base value (typically synced from an EVM layer).
func (e *MOSSAGIEngine) SetBV(bv float64) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.bv = bv
}

// AddGene adds a gene to the internal pool.
func (e *MOSSAGIEngine) AddGene(g *Gene) error {
	return e.pool.Add(g)
}

// RemoveGene removes a gene by ID.
func (e *MOSSAGIEngine) RemoveGene(id string) error {
	return e.pool.Remove(id)
}

// ComputeEV recalculates EV = BV + Σ(Gene × Φ) and returns it.
func (e *MOSSAGIEngine) ComputeEV() float64 {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.recompute()
	return e.ev
}

// recompute (caller must hold e.mu).
func (e *MOSSAGIEngine) recompute() {
	sum := 0.0
	for _, g := range e.pool.All() {
		sum += g.EffectiveValue()
	}
	e.phiSum = sum
	e.ev = e.bv + e.phiSum
	e.tier = computeTier(e.ev)
}

// ComputeDeltaG returns ΔG = (EV_t - EV_{t-1}) / EV_{t-1}.
// Returns 0 when prevEV is 0 (first cycle).
func (e *MOSSAGIEngine) ComputeDeltaG() float64 {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.prevEV == 0 {
		return 0
	}
	return (e.ev - e.prevEV) / e.prevEV
}

// AdaptPhi adjusts every gene's Φ based on task outcome.
// success == true  → Φ += adaptRate
// success == false → Φ -= adaptRate (floored at 0.01)
func (e *MOSSAGIEngine) AdaptPhi(taskSuccess bool) {
	e.mu.Lock()
	defer e.mu.Unlock()

	rate := e.cfg.PhiAdaptRate
	if rate <= 0 {
		rate = 0.05
	}

	all := e.pool.All()
	for _, g := range all {
		if taskSuccess {
			g.Phi += rate
		} else {
			g.Phi -= rate
			if g.Phi < 0.01 {
				g.Phi = 0.01
			}
		}
		// write back
		if existing, ok := e.pool.Get(g.ID); ok {
			existing.Phi = g.Phi
		}
	}
	e.recompute()
}

// InjectTaskResult lets external callers feed success/failure + a BV boost.
func (e *MOSSAGIEngine) InjectTaskResult(success bool, boost float64) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if success {
		e.bv += boost
	} else {
		e.bv -= boost * 0.5
		if e.bv < 0 {
			e.bv = 0
		}
	}

	rate := e.cfg.PhiAdaptRate
	if rate <= 0 {
		rate = 0.05
	}
	all := e.pool.All()
	for _, g := range all {
		if success {
			g.Phi += rate
		} else {
			g.Phi -= rate
			if g.Phi < 0.01 {
				g.Phi = 0.01
			}
		}
		if existing, ok := e.pool.Get(g.ID); ok {
			existing.Phi = g.Phi
		}
	}
	e.recompute()
}

// ---------------------------------------------------------------------------
// Cycle — one round of evolution
// ---------------------------------------------------------------------------

// Cycle runs one evolution round: record previous EV, apply crossover/mutation, recompute.
func (e *MOSSAGIEngine) Cycle() *DeltaGRecord {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.prevEV = e.ev
	e.recompute()

	// crossover
	if e.cfg.CrossoverRate > 0 {
		all := e.pool.All()
		n := int(math.Round(float64(len(all)) * e.cfg.CrossoverRate))
		if n >= 2 && len(all) >= 2 {
			for i := 0; i < n; i++ {
				a := all[rand.Intn(len(all))]
				b := all[rand.Intn(len(all))]
				if a.ID == b.ID {
					continue
				}
				child := e.pool.Crossover(a, b)
				if child != nil {
					_ = e.pool.Add(child) // ignore if full
				}
			}
		}
	}

	// mutation
	if e.cfg.MutationRate > 0 {
		all := e.pool.All()
		n := int(math.Round(float64(len(all)) * e.cfg.MutationRate))
		for i := 0; i < n && i < len(all); i++ {
			g := all[rand.Intn(len(all))]
			mut := e.pool.Mutate(g)
			if mut != nil {
				_ = e.pool.Add(mut)
			}
		}
	}

	e.recompute()

	deltaG := 0.0
	if e.prevEV > 0 {
		deltaG = (e.ev - e.prevEV) / e.prevEV
	}

	rec := DeltaGRecord{
		Timestamp: time.Now(),
		EV:        e.ev,
		BV:        e.bv,
		PhiSum:    e.phiSum,
		DeltaG:    deltaG,
		Tier:      e.tier,
	}
	e.deltaGHistory = append(e.deltaGHistory, rec)
	return &rec
}

// ---------------------------------------------------------------------------
// Start / Stop — periodic evolution loop
// ---------------------------------------------------------------------------

// Start launches a goroutine that calls Cycle at the configured interval.
func (e *MOSSAGIEngine) Start() {
	e.mu.Lock()
	if e.running {
		e.mu.Unlock()
		return
	}
	e.running = true
	e.stopCh = make(chan struct{})
	e.mu.Unlock()

	go e.loop()
}

func (e *MOSSAGIEngine) loop() {
	ticker := time.NewTicker(e.cfg.CycleInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			e.Cycle()
		case <-e.stopCh:
			return
		}
	}
}

// Stop halts the periodic evolution loop.
func (e *MOSSAGIEngine) Stop() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if !e.running {
		return
	}
	e.running = false
	close(e.stopCh)
}

// ---------------------------------------------------------------------------
// Queries
// ---------------------------------------------------------------------------

// GetTier returns the current tier (1–5).
func (e *MOSSAGIEngine) GetTier() int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.tier
}

// GetState returns a JSON-serialisable snapshot.
func (e *MOSSAGIEngine) GetState() map[string]interface{} {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return map[string]interface{}{
		"bv":            e.bv,
		"phi_sum":       e.phiSum,
		"ev":            e.ev,
		"prev_ev":       e.prevEV,
		"tier":          e.tier,
		"gene_count":    e.pool.Len(),
		"running":       e.running,
		"history_len":   len(e.deltaGHistory),
		"last_delta_g":  e.lastDeltaG(),
	}
}

// History returns a copy of the delta-G history.
func (e *MOSSAGIEngine) History() []DeltaGRecord {
	e.mu.RLock()
	defer e.mu.RUnlock()
	out := make([]DeltaGRecord, len(e.deltaGHistory))
	copy(out, e.deltaGHistory)
	return out
}

// MarshalJSON implements json.Marshaler for the engine snapshot.
func (e *MOSSAGIEngine) MarshalJSON() ([]byte, error) {
	return json.Marshal(e.GetState())
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

func (e *MOSSAGIEngine) lastDeltaG() float64 {
	if len(e.deltaGHistory) == 0 {
		return 0
	}
	return e.deltaGHistory[len(e.deltaGHistory)-1].DeltaG
}

// computeTier maps EV → tier.
func computeTier(ev float64) int {
	switch {
	case ev >= 20:
		return 5
	case ev >= 10:
		return 4
	case ev >= 5:
		return 3
	case ev >= 2:
		return 2
	default:
		return 1
	}
}
