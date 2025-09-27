package config

import (
	"os"
	
	"gopkg.in/yaml.v3"
)

type Config struct {
	Server struct {
		Port int `yaml:"port"`
	} `yaml:"server"`
	
	Qdrant QdrantConfig `yaml:"qdrant"`
	
	Ollama OllamaConfig `yaml:"ollama"`
	
	RAG RAGConfig `yaml:"rag"`
}

type QdrantConfig struct {
	Host       string `yaml:"host"`
	Port       int    `yaml:"port"`
	Collection string `yaml:"collection"`
}

type OllamaConfig struct {
	BaseURL        string `yaml:"base_url"`
	ChatModel      string `yaml:"chat_model"`
	EmbeddingModel string `yaml:"embedding_model"`
}

type RAGConfig struct {
	ChunkSize    int `yaml:"chunk_size"`
	ChunkOverlap int `yaml:"chunk_overlap"`
	TopK         int `yaml:"top_k"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	
	var cfg Config
	if err = yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	
	return &cfg, nil
}
