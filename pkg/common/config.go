package common

import "encoding/json"

// AgentConfig holds top-level configuration.
type AgentConfig struct {
	Server     ServerConfig     `yaml:"server"`
	Blockchain BlockchainConfig `yaml:"blockchain"`
	Evolution  EvolutionConfig  `yaml:"evolution"`
	APEX       APEXAppConfig    `yaml:"apex"`
	Reasoning  ReasoningConfig  `yaml:"reasoning"`
	Robot      RobotConfig      `yaml:"robot"`
	Security   SecurityConfig   `yaml:"security"`
	Scheduler  SchedulerConfig  `yaml:"scheduler"`
}

// APEXAppConfig holds APEX dynamic growth model settings.
type APEXAppConfig struct {
	Enabled       bool    `yaml:"enabled"`
	StatePath     string  `yaml:"state_path"`
	Beta          float64 `yaml:"beta"`
	Eta           float64 `yaml:"eta"`
	HarmRateMax   float64 `yaml:"harm_rate_max"`
	CycleInterval int     `yaml:"cycle_interval"`
	SignalRelay   bool    `yaml:"signal_relay"`
	Source        string  `yaml:"source"`
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Host         string `yaml:"host"`
	Port         int    `yaml:"port"`
	ReadTimeout  int    `yaml:"read_timeout"`
	WriteTimeout int    `yaml:"write_timeout"`
}

// BlockchainConfig holds blockchain settings.
type BlockchainConfig struct {
	Difficulty    int    `yaml:"difficulty"`
	BlockInterval int    `yaml:"block_interval"`
	DBPath        string `yaml:"db_path"`
}

// EvolutionConfig holds evolution settings.
type EvolutionConfig struct {
	PopulationSize int     `yaml:"population_size"`
	MutationRate   float64 `yaml:"mutation_rate"`
	CrossoverRate  float64 `yaml:"crossover_rate"`
	Generations    int     `yaml:"generations"`
}

// ReasoningConfig holds reasoning settings.
type ReasoningConfig struct {
	LLMEndpoint  string `yaml:"llm_endpoint"`
	MaxSteps     int    `yaml:"max_steps"`
	ShortTermTTL int    `yaml:"short_term_ttl"`
}

// RobotConfig holds robot/nanobot settings.
type RobotConfig struct {
	WorkerCount   int `yaml:"worker_count"`
	QueueSize     int `yaml:"queue_size"`
	HeartbeatRate int `yaml:"heartbeat_rate"`
}

// SecurityConfig holds security settings.
type SecurityConfig struct {
	EncryptionKey string `yaml:"encryption_key"`
	JWTSecret     string `yaml:"jwt_secret"`
	JWTExpiry     int    `yaml:"jwt_expiry"`
}

// SchedulerConfig holds scheduler settings.
type SchedulerConfig struct {
	CronInterval int `yaml:"cron_interval"`
	MaxRetries   int `yaml:"max_retries"`
	RetryDelay   int `yaml:"retry_delay"`
}

// DefaultConfig returns a config with sensible defaults.
func DefaultConfig() *AgentConfig {
	return &AgentConfig{
		Server:     ServerConfig{Host: "0.0.0.0", Port: 8080, ReadTimeout: 30, WriteTimeout: 30},
		Blockchain: BlockchainConfig{Difficulty: 4, BlockInterval: 10, DBPath: "./data/blocks"},
		Evolution:  EvolutionConfig{PopulationSize: 50, MutationRate: 0.1, CrossoverRate: 0.7, Generations: 100},
		APEX: APEXAppConfig{
			Enabled:       true,
			StatePath:     "_apex_state.json",
			Beta:          1.01,
			Eta:           0.00001,
			HarmRateMax:   0.5,
			CycleInterval: 30,
			SignalRelay:   true,
			Source:        "agent-os-v2",
		},
		Reasoning: ReasoningConfig{LLMEndpoint: "http://localhost:11434", MaxSteps: 10, ShortTermTTL: 300},
		Robot:     RobotConfig{WorkerCount: 4, QueueSize: 1000, HeartbeatRate: 30},
		Security:  SecurityConfig{JWTExpiry: 3600},
		Scheduler: SchedulerConfig{CronInterval: 60, MaxRetries: 3, RetryDelay: 5},
	}
}

// ToJSON serializes config to JSON.
func (c *AgentConfig) ToJSON() ([]byte, error) {
	return json.MarshalIndent(c, "", "  ")
}
