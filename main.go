package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	
	"github.com/gin-gonic/gin"
	"github.com/qdrant/go-client/qdrant"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Qdrant struct {
		Host       string `yaml:"host"`
		Port       int    `yaml:"port"`
		Collection string `yaml:"collection"`
	} `yaml:"qdrant"`
	
	Ollama struct {
		BaseURL        string `yaml:"base_url"`
		Model          string `yaml:"model"`
		EmbeddingModel string `yaml:"embedding_model"`
	} `yaml:"ollama"`
	
	HTTPServer struct {
		Port int `yaml:"port"`
	} `yaml:"http_server"`
}

type App struct {
	config       *Config
	qdrantClient *qdrant.Client
}

func main() {
	// Load config
	configData, err := os.ReadFile("configs/config.yaml")
	if err != nil {
		log.Fatal("Read config failed:", err)
	}
	
	var config Config
	if err = yaml.Unmarshal(configData, &config); err != nil {
		log.Fatal("Parse config failed:", err)
	}
	
	// Connect to Qdrant
	qdrantConfig := qdrant.Config{
		Host: config.Qdrant.Host,
		Port: config.Qdrant.Port,
	}
	
	qdrantClient, err := qdrant.NewClient(&qdrantConfig)
	if err != nil {
		log.Fatal("Connect to Qdrant failed:", err)
	}
	defer qdrantClient.Close()
	
	// Check/Create collection
	ctx := context.Background()
	collections, err := qdrantClient.ListCollections(ctx)
	if err != nil {
		log.Fatal("List collections failed:", err)
	}
	
	collectionExists := false
	for _, col := range collections {
		if col == config.Qdrant.Collection {
			collectionExists = true
			break
		}
	}
	
	if !collectionExists {
		err = qdrantClient.CreateCollection(ctx, &qdrant.CreateCollection{
			CollectionName: config.Qdrant.Collection,
			VectorsConfig: qdrant.NewVectorsConfig(&qdrant.VectorParams{
				Size:     768, // nomic-embed-text dimension
				Distance: qdrant.Distance_Cosine,
			}),
		})
		if err != nil {
			log.Fatal("Create collection failed:", err)
		}
		log.Println("Created collection:", config.Qdrant.Collection)
	}
	
	app := &App{
		config:       &config,
		qdrantClient: qdrantClient,
	}
	
	// Setup routes
	r := gin.Default()
	r.GET("/health", app.healthCheck)
	r.POST("/ingest", app.ingestHandler)
	r.POST("/chat", app.chatHandler)
	
	// Server
	r.Run(fmt.Sprintf(":%d", config.HTTPServer.Port))
}

func (app *App) healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"qdrant": "connected",
	})
}

func (app *App) ingestHandler(c *gin.Context) {

}

func (app *App) chatHandler(c *gin.Context) {

}
