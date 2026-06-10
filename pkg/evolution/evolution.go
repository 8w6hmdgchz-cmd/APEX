package evolution

import (
	"math"
	"math/rand"
	"sync"
)

// GeneType represents a gene category.
type GeneType int

const (
	GeneTypeCognition GeneType = iota
	GeneTypeMemory
	GeneTypePerception
	GeneTypeAction
	GeneTypeSocial
	GeneTypeAdaptation
)

// GeneName represents a specific gene.
type GeneName string

const (
	GeneFocus       GeneName = "focus"
	GeneCreativity  GeneName = "creativity"
	GeneRetention   GeneName = "retention"
	GeneRecall      GeneName = "recall"
	GeneSensitivity GeneName = "sensitivity"
	GeneResolution  GeneName = "resolution"
	GeneSpeed       GeneName = "speed"
	GeneAccuracy    GeneName = "accuracy"
	GeneEmpathy     GeneName = "empathy"
	GeneCooperation GeneName = "cooperation"
	GeneResilience  GeneName = "resilience"
	GeneFlexibility GeneName = "flexibility"
)

// Gene represents a single gene with a value.
type Gene struct {
	Name  GeneName
	Type  GeneType
	Value float64 // 0.0 - 1.0
}

// Genome is a collection of 12 genes.
type Genome struct {
	Genes []Gene
}

// NewDefaultGenome creates a genome with default values.
func NewDefaultGenome() *Genome {
	return &Genome{Genes: []Gene{
		{Name: GeneFocus, Type: GeneTypeCognition, Value: 0.5},
		{Name: GeneCreativity, Type: GeneTypeCognition, Value: 0.5},
		{Name: GeneRetention, Type: GeneTypeMemory, Value: 0.5},
		{Name: GeneRecall, Type: GeneTypeMemory, Value: 0.5},
		{Name: GeneSensitivity, Type: GeneTypePerception, Value: 0.5},
		{Name: GeneResolution, Type: GeneTypePerception, Value: 0.5},
		{Name: GeneSpeed, Type: GeneTypeAction, Value: 0.5},
		{Name: GeneAccuracy, Type: GeneTypeAction, Value: 0.5},
		{Name: GeneEmpathy, Type: GeneTypeSocial, Value: 0.5},
		{Name: GeneCooperation, Type: GeneTypeSocial, Value: 0.5},
		{Name: GeneResilience, Type: GeneTypeAdaptation, Value: 0.5},
		{Name: GeneFlexibility, Type: GeneTypeAdaptation, Value: 0.5},
	}}
}

// DGResult holds the DeltaG computation result.
type DGResult struct {
	DeltaG     float64
	Lambda     float64 // Learning rate
	Theta      float64 // Task complexity
	K          float64 // Knowledge depth
	Xi         float64 // Adaptation factor
	Psi        float64 // Social influence
	Phi        float64 // Resource efficiency
	Sigma      float64 // Security factor
	H          float64 // Entropy
	T          float64 // Time pressure
	Epsilon    float64 // Error tolerance
}

// ComputeDeltaG calculates: ΔG = (Λ×Θ×K×ξ×Ψ×Φ×Σ) / (H×T×ε)
func ComputeDeltaG(lambda, theta, k, xi, psi, phi, sigma, h, t, epsilon float64) DGResult {
	// Clamp small denominators to prevent division by zero
	denom := h * t * epsilon
	if denom < 1e-10 {
		denom = 1e-10
	}
	dg := (lambda * theta * k * xi * psi * phi * sigma) / denom
	return DGResult{
		DeltaG: dg, Lambda: lambda, Theta: theta, K: k, Xi: xi,
		Psi: psi, Phi: phi, Sigma: sigma, H: h, T: t, Epsilon: epsilon,
	}
}

// FitnessFunc evaluates a genome's fitness.
type FitnessFunc func(*Genome) float64

// Evolver manages evolutionary operations.
type Evolver struct {
	mu             sync.RWMutex
	population     []*Genome
	mutationRate   float64
	crossoverRate  float64
	fitness        FitnessFunc
	lastDeltaG     DGResult
}

// NewEvolver creates a new evolver.
func NewEvolver(popSize int, mutationRate, crossoverRate float64, fitness FitnessFunc) *Evolver {
	pop := make([]*Genome, popSize)
	for i := range pop {
		g := NewDefaultGenome()
		// Randomize initial values
		for j := range g.Genes {
			g.Genes[j].Value = rand.Float64()
		}
		pop[i] = g
	}
	return &Evolver{population: pop, mutationRate: mutationRate, crossoverRate: crossoverRate, fitness: fitness}
}

// Evolve performs one generation of evolution.
func (e *Evolver) Evolve() {
	e.mu.Lock()
	defer e.mu.Unlock()

	n := len(e.population)
	fitnesses := make([]float64, n)
	totalFitness := 0.0
	for i, g := range e.population {
		fitnesses[i] = e.fitness(g)
		totalFitness += fitnesses[i]
	}

	newPop := make([]*Genome, n)
	for i := 0; i < n; i++ {
		parent1 := e.selectParent(fitnesses, totalFitness)
		parent2 := e.selectParent(fitnesses, totalFitness)

		var child *Genome
		if rand.Float64() < e.crossoverRate {
			child = e.crossover(parent1, parent2)
		} else {
			child = e.clone(parent1)
		}

		if rand.Float64() < e.mutationRate {
			e.mutate(child)
		}

		// Self-repair: clamp values
		e.selfRepair(child)
		newPop[i] = child
	}
	e.population = newPop
}

func (e *Evolver) selectParent(fitnesses []float64, total float64) *Genome {
	r := rand.Float64() * total
	cumulative := 0.0
	for i, f := range fitnesses {
		cumulative += f
		if cumulative >= r {
			return e.population[i]
		}
	}
	return e.population[len(e.population)-1]
}

func (e *Evolver) crossover(a, b *Genome) *Genome {
	child := NewDefaultGenome()
	for i := range child.Genes {
		if rand.Float64() < 0.5 {
			child.Genes[i].Value = a.Genes[i].Value
		} else {
			child.Genes[i].Value = b.Genes[i].Value
		}
	}
	return child
}

func (e *Evolver) mutate(g *Genome) {
	for i := range g.Genes {
		if rand.Float64() < 0.3 {
			g.Genes[i].Value += (rand.Float64() - 0.5) * 0.2
		}
	}
}

func (e *Evolver) selfRepair(g *Genome) {
	for i := range g.Genes {
		g.Genes[i].Value = math.Max(0, math.Min(1, g.Genes[i].Value))
	}
}

func (e *Evolver) clone(g *Genome) *Genome {
	c := NewDefaultGenome()
	for i := range c.Genes {
		c.Genes[i].Value = g.Genes[i].Value
	}
	return c
}

// GetBest returns the best genome and its fitness.
func (e *Evolver) GetBest() (*Genome, float64) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	var best *Genome
	var bestFit float64
	for _, g := range e.population {
		f := e.fitness(g)
		if f > bestFit {
			bestFit = f
			best = g
		}
	}
	return best, bestFit
}

// LastDeltaG returns the last computed DeltaG.
func (e *Evolver) LastDeltaG() DGResult {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.lastDeltaG
}

// SetLastDeltaG stores a DeltaG result.
func (e *Evolver) SetLastDeltaG(dg DGResult) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.lastDeltaG = dg
}
