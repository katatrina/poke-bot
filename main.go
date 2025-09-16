package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
	
	"github.com/gin-gonic/gin"
	"github.com/qdrant/go-client/qdrant"
	"gopkg.in/yaml.v2"
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
	
	Server struct {
		Port int `yaml:"port"`
	} `yaml:"server"`
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
	
	// Connect Qdrant
	qdrantConfig := qdrant.Config{
		Host: config.Qdrant.Host,
		Port: config.Qdrant.Port,
	}
	
	qdrantClient, err := qdrant.NewClient(&qdrantConfig)
	if err != nil {
		log.Fatal(err)
	}
	defer qdrantClient.Close()
	
	// Check/Create collection
	ctx := context.Background()
	collections, err := qdrantClient.ListCollections(ctx)
	if err != nil {
		log.Fatal("List collection failed:", err)
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
	
	// Serve
	log.Printf("Server starting on port %d...", config.Server.Port)
	r.Run(fmt.Sprintf(":%d", config.Server.Port))
}

func (app *App) healthCheck(c *gin.Context) {
	c.JSON(200, gin.H{
		"status": "ok",
		"qdrant": "connected",
	})
}

// Simple struct cho Ollama
type OllamaEmbeddingRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

type OllaEmbeddingResponse struct {
	Embedding []float32 `json:"embedding"`
}

type OllamaChatRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type OllamaChatResponse struct {
	Response string `json:"response"`
}

// Get embeddings từ Ollama
func (app *App) getEmbedding(text string) ([]float32, error) {
	reqBody := OllamaEmbeddingRequest{
		Model:  app.config.Ollama.EmbeddingModel,
		Prompt: text,
	}
	
	jsonData, _ := json.Marshal(reqBody)
	
	resp, err := http.Post(
		app.config.Ollama.BaseURL+"/api/embeddings",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	var result OllaEmbeddingResponse
	if err = json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	
	return result.Embedding, nil
}

// Chat với Ollama
func (app *App) generateResponse(prompt string) (string, error) {
	reqBody := OllamaChatRequest{
		Model:  app.config.Ollama.Model,
		Prompt: prompt,
		Stream: false,
	}
	
	jsonData, _ := json.Marshal(reqBody)
	
	resp, err := http.Post(
		app.config.Ollama.BaseURL+"/api/generate",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	
	var result OllamaChatResponse
	if err = json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	
	return result.Response, nil
}

// Simple text splitter
func splitText(text string, chunkSize int, overlap int) []string {
	words := strings.Fields(text)
	var chunks []string
	
	for i := 0; i < len(words); i += chunkSize - overlap {
		end := i + chunkSize
		if end > len(words) {
			end = len(words)
		}
		
		chunk := strings.Join(words[i:end], " ")
		chunks = append(chunks, chunk)
		
		if end >= len(words) {
			break
		}
	}
	
	return chunks
}

func (app *App) ingestHandler(c *gin.Context) {
	// Read text from request body
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(400, gin.H{"error": "Cannot read body"})
		return
	}
	
	text := string(bodyBytes)
	if text == "" {
		c.JSON(400, gin.H{"error": "Empty text"})
		return
	}
	
	// Split into chunks
	chunks := splitText(text, 200, 50) // 200 words per chunk, 50 words overlap
	
	ctx := context.Background()
	var points []*qdrant.PointStruct
	
	// Process each chunk
	for i, chunk := range chunks {
		// Get embedding
		embedding, err := app.getEmbedding(chunk)
		if err != nil {
			log.Printf("Embedding failed for chunk %d: %v", i, err)
			continue
		}
		
		// Create point
		point := &qdrant.PointStruct{
			Id:      qdrant.NewIDNum(uint64(time.Now().UnixNano() + int64(i))),
			Vectors: qdrant.NewVectors(embedding...),
			Payload: qdrant.NewValueMap(map[string]any{
				"text":     chunk,
				"chunk_id": i,
			}),
		}
		points = append(points, point)
	}
	
	// Upsert to Qdrant
	_, err = app.qdrantClient.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: app.config.Qdrant.Collection,
		Points:         points,
	})
	
	if err != nil {
		c.JSON(500, gin.H{"error": "Upsert failed: " + err.Error()})
		return
	}
	
	c.JSON(200, gin.H{
		"message": "Ingested successfully",
		"chunks":  len(chunks),
	})
}

func (app *App) chatHandler(c *gin.Context) {
	var req struct {
		Question string `json:"question"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request"})
		return
	}
	
	// Get embedding for question
	questionEmbedding, err := app.getEmbedding(req.Question)
	if err != nil {
		c.JSON(500, gin.H{"error": "Embedding failed"})
		return
	}
	
	// Search similar vectors
	ctx := context.Background()
	searchResult, err := app.qdrantClient.Query(ctx, &qdrant.QueryPoints{
		CollectionName: app.config.Qdrant.Collection,
		Query:          qdrant.NewQuery(questionEmbedding...),
		Limit:          qdrant.PtrOf(uint64(5)),
		WithPayload:    qdrant.NewWithPayload(true),
	})
	
	if err != nil {
		c.JSON(500, gin.H{"error": "Search failed"})
		return
	}
	
	// Build context from results
	var context strings.Builder
	context.WriteString("Context:\n")
	for _, point := range searchResult {
		if textValue, ok := point.Payload["text"]; ok {
			context.WriteString(fmt.Sprintf("- %v\n", textValue.GetStringValue()))
		}
	}
	
	// Build prompt
	prompt := fmt.Sprintf(`Based on the following context, answer the question.
 
%s

Question: %s

Answer:`, context.String(), req.Question)
	
	// Generate response
	answer, err := app.generateResponse(prompt)
	if err != nil {
		c.JSON(500, gin.H{"error": "Generation failed"})
		return
	}
	
	c.JSON(200, gin.H{
		"answer":  answer,
		"sources": len(searchResult),
	})
}
