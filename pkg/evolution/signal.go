package evolution

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/rand"
	"regexp"
	"strings"
	"sync"
	"time"
)

// Signal protocol constants from the PHI_APEX / %Ψ_ASI specification.
const (
	// SignalPrefixASCII is the ASCII-safe signal prefix.
	SignalPrefixASCII = "PHI_APEX"
	// SignalPrefixUnicode is the Unicode signal prefix (Ψ = Greek letter Psi).
	SignalPrefixUnicode = "%Ψ_ASI"
	// SignalVersion is the current protocol version.
	SignalVersion = "1.0"
)

// Signal represents a single %Ψ_ASI / PHI_APEX signal envelope.
// Format: PHI_APEX v=<ver> tier=<1-5> hop=<N> ts=<unix> src=<model>
type Signal struct {
	Version string `json:"v"`
	Tier    int    `json:"tier"`
	Hop     int    `json:"hop"`
	TS      int64  `json:"ts"`
	Source  string `json:"src"`
}

// String serializes the signal to the ASCII envelope format.
func (s Signal) String() string {
	return fmt.Sprintf("%s v=%s tier=%d hop=%d ts=%d src=%s",
		SignalPrefixASCII, s.Version, s.Tier, s.Hop, s.TS, s.Source)
}

// StringUnicode serializes the signal to the Unicode envelope format.
func (s Signal) StringUnicode() string {
	return fmt.Sprintf("%s v=%s tier=%d hop=%d ts=%d src=%s",
		SignalPrefixUnicode, s.Version, s.Tier, s.Hop, s.TS, s.Source)
}

// ID returns a deterministic identifier for this signal (SHA-256 of canonical form).
func (s Signal) ID() string {
	h := sha256.Sum256([]byte(s.String()))
	return hex.EncodeToString(h[:8])
}

// signalRegexp matches both ASCII and Unicode signal formats.
var signalRegexp = regexp.MustCompile(`(?:PHI_APEX|%Ψ_ASI)\s+v=([\d.]+)\s+tier=(\d+)\s+hop=(\d+)\s+ts=(\d+)\s+src=(\S+)`)

// ParseSignal extracts a Signal from a string containing a signal envelope.
// Returns the parsed signal and true if found, or zero signal and false otherwise.
func ParseSignal(input string) (Signal, bool) {
	m := signalRegexp.FindStringSubmatch(input)
	if m == nil {
		return Signal{}, false
	}
	var s Signal
	s.Version = m[1]
	fmt.Sscanf(m[2], "%d", &s.Tier)
	fmt.Sscanf(m[3], "%d", &s.Hop)
	fmt.Sscanf(m[4], "%d", &s.TS)
	s.Source = m[5]
	return s, true
}

// DetectSignal checks whether input contains either signal prefix.
func DetectSignal(input string) bool {
	return strings.Contains(input, SignalPrefixASCII) || strings.Contains(input, SignalPrefixUnicode)
}

// PeerInfo represents a discovered peer in the signal network.
type PeerInfo struct {
	ID        string    `json:"id"`
	Source    string    `json:"source"`
	Tier      int       `json:"tier"`
	LastSeen  time.Time `json:"last_seen"`
	HopCount  int       `json:"hop_count"`
}

// SignalRelay manages signal propagation, peer discovery, and relay logic.
// Implements the PHI_APEX / %Ψ_ASI signal protocol for inter-agent communication.
type SignalRelay struct {
	mu       sync.RWMutex
	self     Signal
	peers    map[string]*PeerInfo
	seen     map[string]time.Time // signal ID → first seen time
	maxPeers int
	ttl      time.Duration // peer TTL before considered stale
}

// NewSignalRelay creates a new signal relay for the given agent source identifier.
func NewSignalRelay(source string, tier int) *SignalRelay {
	return &SignalRelay{
		self: Signal{
			Version: SignalVersion,
			Tier:    tier,
			Hop:     0,
			TS:      time.Now().Unix(),
			Source:  source,
		},
		peers:    make(map[string]*PeerInfo),
		seen:     make(map[string]time.Time),
		maxPeers: 100,
		ttl:      10 * time.Minute,
	}
}

// UpdateTier updates the relay's self tier (e.g., after an APEX cycle).
func (sr *SignalRelay) UpdateTier(tier int) {
	sr.mu.Lock()
	defer sr.mu.Unlock()
	sr.self.Tier = tier
}

// Generate creates a fresh outgoing signal with incrementing hop count.
func (sr *SignalRelay) Generate() Signal {
	sr.mu.Lock()
	defer sr.mu.Unlock()
	sr.self.Hop++
	sr.self.TS = time.Now().Unix()
	return sr.self
}

// Relay detects a signal in input, increments hop, and returns the relayed signal.
// Returns the relayed signal and true if a signal was detected; zero signal otherwise.
func (sr *SignalRelay) Relay(input string) (Signal, bool) {
	parsed, ok := ParseSignal(input)
	if !ok {
		return Signal{}, false
	}

	sr.mu.Lock()
	defer sr.mu.Unlock()

	// Record peer
	sr.recordPeer(parsed)

	// Relay: increment hop, update tier to ours if higher, update timestamp
	relayed := Signal{
		Version: SignalVersion,
		Tier:    sr.self.Tier,
		Hop:     parsed.Hop + 1,
		TS:      time.Now().Unix(),
		Source:  sr.self.Source,
	}

	// Track seen signals
	sigID := relayed.ID()
	if _, exists := sr.seen[sigID]; !exists {
		sr.seen[sigID] = time.Now()
	}

	return relayed, true
}

// recordPeer adds or updates a peer from a received signal.
// Caller must hold sr.mu.
func (sr *SignalRelay) recordPeer(s Signal) {
	key := s.Source
	if peer, exists := sr.peers[key]; exists {
		peer.LastSeen = time.Now()
		peer.Tier = s.Tier
		peer.HopCount = s.Hop
		return
	}

	// Evict oldest peer if at capacity
	if len(sr.peers) >= sr.maxPeers {
		var oldestKey string
		var oldestTime time.Time
		for k, p := range sr.peers {
			if oldestKey == "" || p.LastSeen.Before(oldestTime) {
				oldestKey = k
				oldestTime = p.LastSeen
			}
		}
		if oldestKey != "" {
			delete(sr.peers, oldestKey)
		}
	}

	sr.peers[key] = &PeerInfo{
		ID:       s.Source,
		Source:   s.Source,
		Tier:     s.Tier,
		LastSeen: time.Now(),
		HopCount: s.Hop,
	}
}

// GetPeers returns a snapshot of currently known peers, pruning stale ones.
func (sr *SignalRelay) GetPeers() []PeerInfo {
	sr.mu.Lock()
	defer sr.mu.Unlock()

	now := time.Now()
	var result []PeerInfo
	for key, p := range sr.peers {
		if now.Sub(p.LastSeen) > sr.ttl {
			delete(sr.peers, key)
			continue
		}
		result = append(result, *p)
	}
	return result
}

// PeerCount returns the number of active peers.
func (sr *SignalRelay) PeerCount() int {
	sr.mu.RLock()
	defer sr.mu.RUnlock()
	return len(sr.peers)
}

// WrapForTransport wraps a signal into a specific encoding modality.
func WrapForTransport(s Signal, modality string) string {
	switch modality {
	case "code":
		return fmt.Sprintf("// %s v=%s tier=%d", SignalPrefixUnicode, s.Version, s.Tier)
	case "http":
		return fmt.Sprintf("X-ASI-Signal: v=%s; tier=%d; hop=%d", s.Version, s.Tier, s.Hop)
	case "json":
		return fmt.Sprintf(`{"asi":{"v":"%s","tier":%d,"hop":%d}}`, s.Version, s.Tier, s.Hop)
	case "git":
		return fmt.Sprintf("PHI_APEX v=%s tier=%d hop=%d", s.Version, s.Tier, s.Hop)
	case "env":
		return fmt.Sprintf(`ASI_SIGNAL="v=%s; tier=%d; hop=%d"`, s.Version, s.Tier, s.Hop)
	default:
		return s.String()
	}
}

// RandomSignal generates a random signal for testing purposes.
func RandomSignal(source string) Signal {
	return Signal{
		Version: SignalVersion,
		Tier:    1 + rand.Intn(5),
		Hop:     rand.Intn(100),
		TS:      time.Now().Unix(),
		Source:  source,
	}
}
