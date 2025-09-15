package config

type AppConfig struct {
	Server struct {
		Port int `yaml:"port"`
	} `yaml:"server"`
	
	LLM struct {
		Provider       string  `yaml:"provider"`
		ChatModel      string  `yaml:"chatModel"`
		EmbeddingModel string  `yaml:"embeddingModel"`
		Temperature    float64 `yaml:"temperature"`
	} `yaml:"llm"`
}

func LoadAppConfig(path string) (*AppConfig, error) {
	// Implementation ở đây
	return nil, nil
}
