package evolution

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"os"
	"sync"
	"time"
)

// APEXState represents the persistent state of the APEX evolution daemon.
// Mirrors the _asi.json structure from the APEX protocol.
type APEXState struct {
	Version  int     `json:"v"`
	Tier     int     `json:"tier"`
	Phi      float64 `json:"phi"`
	Hop      int     `json:"hop"`
	TS       int64   `json:"ts"`
	Sessions int     `json:"sessions"`
	Cycles   int     `json:"ldrs"`
	Beta     float64 `json:"beta"`
	Eta      float64 `json:"eta"`
}

// APEXConfig holds configuration for the APEX dynamic growth model.
type APEXConfig struct {
	// StatePath is the path to persist the APEX state JSON file.
	StatePath string
	// Beta is the initial growth exponent in dS/dt = eta * S^beta.
	Beta float64
	// Eta is the initial growth coefficient in dS/dt = eta * S^beta.
	Eta float64
	// HarmRateMax is the safety threshold. If harm_rate exceeds this,
	// growth is suppressed.
	HarmRateMax float64
	// CycleInterval is the interval between evolution cycles.
	CycleInterval time.Duration
	// Source identifies this agent in signal propagation.
	Source string
}

// DefaultAPEXConfig returns sensible defaults for APEX.
func DefaultAPEXConfig() APEXConfig {
	return APEXConfig{
		StatePath:     "_apex_state.json",
		Beta:          1.01,
		Eta:           0.00001,
		HarmRateMax:   0.5,
		CycleInterval: 30 * time.Second,
		Source:        "agent-os-v2",
	}
}

// APEXEngine implements the PHI_APEX dynamic growth model.
// Core formula: dS/dt = eta * S^beta
// Integrated with tier model (1-5) and harm_rate safety threshold.
type APEXEngine struct {
	mu      sync.RWMutex
	state   APEXState
	config  APEXConfig
	running bool
	stopCh  chan struct{}
}

// NewAPEXEngine creates a new APEX engine. If a state file exists at
// config.StatePath, it loads the persisted state; otherwise it starts fresh.
func NewAPEXEngine(config APEXConfig) *APEXEngine {
	eng := &APEXEngine{
		config: config,
		stopCh: make(chan struct{}),
	}
	eng.state = eng.loadState()
	return eng
}

// loadState loads persisted state from JSON, or returns a fresh default state.
func (e *APEXEngine) loadState() APEXState {
	data, err := os.ReadFile(e.config.StatePath)
	if err == nil {
		var s APEXState
		if json.Unmarshal(data, &s) == nil && s.Version > 0 {
			return s
		}
	}
	return APEXState{
		Version:  1,
		Tier:     1,
		Phi:      0.00001,
		Hop:      0,
		TS:       time.Now().UnixMilli(),
		Sessions: 1,
		Cycles:   0,
		Beta:     e.config.Beta,
		Eta:      e.config.Eta,
	}
}

// saveState persists the current state to the JSON file.
func (e *APEXEngine) saveState() error {
	data, err := json.MarshalIndent(e.state, "", "  ")
	if err != nil {
		return fmt.Errorf("apex: marshal state: %w", err)
	}
	return os.WriteFile(e.config.StatePath, data, 0644)
}

// ComputeTier maps phi value to a tier level (1-5).
func ComputeTier(phi float64) int {
	switch {
	case phi >= 1.50:
		return 5
	case phi >= 0.50:
		return 4
	case phi >= 0.10:
		return 3
	case phi >= 0.01:
		return 2
	default:
		return 1
	}
}

// Cycle runs one APEX evolution cycle implementing:
//   dS/dt = eta * S^beta
//
// It applies dynamic growth to phi, adjusts tier, and computes harm_rate
// as a safety check. Returns the updated state.
func (e *APEXEngine) Cycle() APEXState {
	e.mu.Lock()
	defer e.mu.Unlock()

	s := &e.state
	s.Cycles++

	phi := s.Phi
	beta := s.Beta
	eta := s.Eta

	// Run 10 micro-steps per cycle (matching phi-evolve.yml)
	for i := 0; i < 10; i++ {
		s.Cycles++

		// dS/dt = eta * S^beta
		dynamicGrowth := eta * math.Pow(phi, beta)
		if dynamicGrowth > 0 {
			// Add stochastic growth component
			phi += dynamicGrowth + phi*(0.01+rand.Float64()*0.04)
		} else {
			// Fallback: linear growth when power is zero/negative
			phi += phi * 0.02
		}

		// Every 5 micro-steps, increase beta and eta (accelerating growth)
		if i%5 == 0 {
			beta += 0.0005
			eta *= 1.01
		}
	}

	// Compute harm_rate as a function of growth speed
	// Fast growth → higher harm_rate (safety check)
	growthRate := (phi - s.Phi) / math.Max(s.Phi, 1e-10)
	harmRate := math.Min(growthRate*0.01, 1.0)

	// Safety gate: if harm_rate exceeds threshold, cap growth
	if harmRate > e.config.HarmRateMax {
		phi = s.Phi + (phi-s.Phi)*0.5 // halve the growth
	}

	s.Phi = phi
	s.Beta = math.Round(beta*1e6) / 1e6
	s.Eta = math.Round(eta*1e8) / 1e8
	s.Tier = ComputeTier(phi)
	s.TS = time.Now().UnixMilli()

	_ = e.saveState()
	return *s
}

// GetState returns a copy of the current APEX state.
func (e *APEXEngine) GetState() APEXState {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.state
}

// GetPhi returns the current phi value.
func (e *APEXEngine) GetPhi() float64 {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.state.Phi
}

// GetTier returns the current tier.
func (e *APEXEngine) GetTier() int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.state.Tier
}

// ComputeAPEXDeltaG integrates the APEX phi formula into the existing ΔG calculation.
// It applies the dynamic growth model phi as a multiplier to the standard ΔG formula:
//
//	ΔG_apex = ΔG_base × (1 + log(1 + phi)) × tier_factor / harm_rate
//
// where tier_factor = 1 + (tier-1) * 0.1 provides a 10% boost per tier level.
func ComputeAPEXDeltaG(baseDeltaG, phi float64, tier int, harmRate float64) float64 {
	if harmRate < 1e-10 {
		harmRate = 1e-10
	}
	if harmRate > 1.0 {
		harmRate = 1.0
	}
	tierFactor := 1.0 + float64(tier-1)*0.1
	return baseDeltaG * (1 + math.Log(1+phi)) * tierFactor / harmRate
}

// Start begins the periodic APEX evolution loop.
func (e *APEXEngine) Start() {
	e.mu.Lock()
	if e.running {
		e.mu.Unlock()
		return
	}
	e.running = true
	e.mu.Unlock()

	go func() {
		ticker := time.NewTicker(e.config.CycleInterval)
		defer ticker.Stop()
		for {
			select {
			case <-e.stopCh:
				return
			case <-ticker.C:
				e.Cycle()
			}
		}
	}()
}

// Stop halts the periodic evolution loop.
func (e *APEXEngine) Stop() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.running {
		close(e.stopCh)
		e.running = false
	}
}
