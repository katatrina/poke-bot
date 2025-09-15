package config

import (
	"fmt"
	
	"github.com/spf13/viper"
)

type AppConfig struct {
	Server ServerConfig `mapstructure:"server"`
	Qdrant QdrantConfig `mapstructure:"qdrant"`
	LLM    LLMConfig    `mapstructure:"llm"`
	KB     KBConfig     `mapstructure:"kb"`
}

type ServerConfig struct {
	Port int    `mapstructure:"port"`
	Host string `mapstructure:"host"`
}

type QdrantConfig struct {
	URL        string `mapstructure:"url"`
	Collection string `mapstructure:"collection"`
}

type LLMConfig struct {
	Provider       string  `mapstructure:"provider"`
	Model          string  `mapstructure:"model"`
	EmbeddingModel string  `mapstructure:"embedding_model"`
	Temperature    float64 `mapstructure:"temperature"`
	MaxTokens      int     `mapstructure:"max_tokens"`
	APIKey         string  `mapstructure:"api_key"`
	BaseURL        string  `mapstructure:"base_url"`
}

type KBConfig struct {
	ChunkSize      int     `mapstructure:"chunk_size"`
	ChunkOverlap   int     `mapstructure:"chunk_overlap"`
	TopK           int     `mapstructure:"top_k"`
	ScoreThreshold float64 `mapstructure:"score_threshold"`
}

type KBSourceConfig struct {
	Sources []Source `yaml:"sources"`
}

type Source struct {
	Type        string            `yaml:"type"`
	URL         string            `yaml:"url,omitempty"`
	Path        string            `yaml:"path,omitempty"`
	Metadata    map[string]string `yaml:"metadata,omitempty"`
	Enabled     bool              `yaml:"enabled"`
	Description string            `yaml:"description,omitempty"`
}

func Load() (*AppConfig, error) {
	var cfg AppConfig
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unable to decode config: %w", err)
	}
	return &cfg, nil
}

func LoadKBSources() (*KBSourceConfig, error) {
	kbViper := viper.New()
	kbViper.SetConfigName("kb-config")
	kbViper.SetConfigType("yaml")
	kbViper.AddConfigPath("./configs")
	
	if err := kbViper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read kb-config.yaml: %w", err)
	}
	
	var kbConfig KBSourceConfig
	if err := kbViper.Unmarshal(&kbConfig); err != nil {
		return nil, fmt.Errorf("unable to decode kb config: %w", err)
	}
	
	return &kbConfig, nil
}
