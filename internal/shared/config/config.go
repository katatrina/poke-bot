package config

import (
	"fmt"
	"os"
	"strings"
	
	"github.com/spf13/viper"
)

// AppConfig represents application configuration
type AppConfig struct {
	Server ServerConfig `yaml:"server"`
	LLM    LLMConfig    `yaml:"llm"`
}

type ServerConfig struct {
	Port int `yaml:"port"`
}

type LLMConfig struct {
	Provider       string  `yaml:"provider"`
	ChatModel      string  `yaml:"chatModel"`
	EmbeddingModel string  `yaml:"embeddingModel"`
	Temperature    float64 `yaml:"temperature"`
}

// KBConfig represents knowledge base configuration
type KBConfig struct {
	Crawler  CrawlerConfig           `yaml:"crawler"`
	Chunking ChunkingConfig          `yaml:"chunking"`
	Sources  map[string]SourceConfig `yaml:"sources"`
}

type CrawlerConfig struct {
	RateLimit      string `yaml:"rateLimit"`
	MaxConcurrency int    `yaml:"maxConcurrency"`
	Timeout        string `yaml:"timeout"`
}

type ChunkingConfig struct {
	MaxTokens int `yaml:"maxTokens"`
	Overlap   int `yaml:"overlap"`
}

type SourceConfig struct {
	Enabled   bool   `yaml:"enabled"`
	ListURL   string `yaml:"listURL,omitempty"`
	DetailURL string `yaml:"detailURL,omitempty"`
	MaxItems  int    `yaml:"maxItems,omitempty"`
	BatchSize int    `yaml:"batchSize,omitempty"`
}

// LoadAppConfig loads application configuration using Viper
func LoadAppConfig(path string) (*AppConfig, error) {
	v := viper.New()
	
	// Set defaults
	setAppDefaults(v)
	
	v.SetConfigFile(path)
	v.SetEnvPrefix("RAG")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
	
	// Read config file
	if err := v.ReadInConfig(); err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		// File doesn't exist, use defaults + env vars
	}
	
	// Unmarshal to struct
	var config AppConfig
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	
	return &config, nil
}

// setAppDefaults sets default values for app config
func setAppDefaults(v *viper.Viper) {
	v.SetDefault("server.port", 8080)
	v.SetDefault("llm.provider", "ollama")
	v.SetDefault("llm.chatModel", "qwen2.5-coder-3b-instruct")
	v.SetDefault("llm.embeddingModel", "text-embedding-mxbai-embed-large-v1")
	v.SetDefault("llm.temperature", 0.8)
}

// setKBDefaults sets default values for KB config
func setKBDefaults(v *viper.Viper) {
	v.SetDefault("crawler.rateLimit", "100ms")
	v.SetDefault("crawler.maxConcurrency", 3)
	v.SetDefault("crawler.timeout", "30s")
	v.SetDefault("chunking.maxTokens", 800)
	v.SetDefault("chunking.overlap", 100)
}

// validateAppConfig validates application configuration
func validateAppConfig(config *AppConfig) error {
	if config.Server.Port <= 0 || config.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", config.Server.Port)
	}
	
	if config.LLM.Provider != "ollama" && config.LLM.Provider != "openai" {
		return fmt.Errorf("unsupported LLM provider: %s", config.LLM.Provider)
	}
	
	if config.LLM.ChatModel == "" {
		return fmt.Errorf("chat model cannot be empty")
	}
	
	if config.LLM.EmbeddingModel == "" {
		return fmt.Errorf("embedding model cannot be empty")
	}
	
	if config.LLM.Temperature < 0 || config.LLM.Temperature > 2 {
		return fmt.Errorf("invalid temperature: %f (must be between 0 and 2)", config.LLM.Temperature)
	}
	
	// Check required env vars for OpenAI
	if config.LLM.Provider == "openai" {
		if os.Getenv("OPENAI_API_KEY") == "" {
			return fmt.Errorf("OPENAI_API_KEY environment variable required for OpenAI provider")
		}
	}
	
	return nil
}

// validateKBConfig validates knowledge base configuration
func validateKBConfig(config *KBConfig) error {
	if config.Chunking.MaxTokens <= 0 {
		return fmt.Errorf("chunking maxTokens must be positive: %d", config.Chunking.MaxTokens)
	}
	
	if config.Chunking.Overlap < 0 {
		return fmt.Errorf("chunking overlap cannot be negative: %d", config.Chunking.Overlap)
	}
	
	if config.Chunking.Overlap >= config.Chunking.MaxTokens {
		return fmt.Errorf("chunking overlap (%d) must be less than maxTokens (%d)",
			config.Chunking.Overlap, config.Chunking.MaxTokens)
	}
	
	// Validate enabled sources
	enabledCount := 0
	for name, source := range config.Sources {
		if source.Enabled {
			enabledCount++
			if source.ListURL == "" {
				return fmt.Errorf("source %s: listURL cannot be empty", name)
			}
			if source.MaxItems <= 0 {
				return fmt.Errorf("source %s: maxItems must be positive", name)
			}
			if source.BatchSize <= 0 {
				return fmt.Errorf("source %s: batchSize must be positive", name)
			}
		}
	}
	
	if enabledCount == 0 {
		return fmt.Errorf("at least one source must be enabled")
	}
	
	return nil
}

// GetEnabledSources returns only enabled sources
func (kb *KBConfig) GetEnabledSources() map[string]SourceConfig {
	enabled := make(map[string]SourceConfig)
	for name, source := range kb.Sources {
		if source.Enabled {
			enabled[name] = source
		}
	}
	return enabled
}

// GetVectorSize returns vector size based on embedding model
func (c *AppConfig) GetVectorSize() int {
	sizes := map[string]int{
		"text-embedding-mxbai-embed-large-v1": 1024,
		"text-embedding-ada-002":              1536,
		"text-embedding-3-small":              1536,
		"text-embedding-3-large":              3072,
	}
	
	if size, exists := sizes[c.LLM.EmbeddingModel]; exists {
		return size
	}
	
	// Default fallback
	return 1024
}
