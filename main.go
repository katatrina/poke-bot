// main.go
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/qdrant/go-client/qdrant"
	"github.com/tmc/langchaingo/textsplitter"
	"gopkg.in/yaml.v3"
	"resty.dev/v3"
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
	restyClient  *resty.Client
}

func main() {
	// Load config
	configData, err := os.ReadFile("configs/config.yaml")
	if err != nil {
		log.Fatal("Read config file failed:", err)
	}
	
	var config Config
	if err = yaml.Unmarshal(configData, &config); err != nil {
		log.Fatal("Parse config failed:", err)
	}
	
	// Connect to Qdrant
	qdrantClient, err := qdrant.NewClient(&qdrant.Config{
		Host: config.Qdrant.Host,
		Port: config.Qdrant.Port,
	})
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
				Size:     768,                    // nomic-embed-text model
				Distance: qdrant.Distance_Cosine, // optimal for sematic search
			}),
		})
		if err != nil {
			log.Fatal("Create collection failed:", err)
		}
		log.Println("Created collection:", config.Qdrant.Collection)
	}
	
	restyClient := resty.New()
	defer restyClient.Close()
	
	app := &App{
		config:       &config,
		qdrantClient: qdrantClient,
		restyClient:  restyClient,
	}
	
	r := gin.Default()
	r.GET("/health", app.healthCheck)
	r.POST("/ingest", app.ingestHandler)
	r.POST("/chat", app.chatHandler)
	
	r.Run(fmt.Sprintf(":%d", config.Server.Port))
}

func (app *App) healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"qdrant": "connected",
	})
}

// Text splitter with langchaingo
func splitText(text string, charSize int, charOverlap int) ([]string, error) {
	splitter := textsplitter.NewRecursiveCharacter(
		textsplitter.WithChunkSize(charSize),
		textsplitter.WithChunkOverlap(charOverlap),
		textsplitter.WithSeparators([]string{"\n\n", "\n", ". ", " "}),
	)
	
	chunks, err := splitter.SplitText(text)
	if err != nil {
		return nil, err
	}
	
	return chunks, nil
}

type EmbedRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

type EmbedResponse struct {
	Embeddings [][]float32 `json:"embeddings"`
}

func (app *App) embedText(texts ...string) ([][]float32, error) {
	reqBody := EmbedRequest{
		Model: app.config.Ollama.EmbeddingModel,
		Input: texts,
	}
	
	var result EmbedResponse
	
	_, err := app.restyClient.R().
		SetBody(reqBody).
		SetResult(&result).
		Post(app.config.Ollama.BaseURL + "/api/embed")
	if err != nil {
		return nil, err
	}
	
	return result.Embeddings, nil
}

func (app *App) ingestHandler(c *gin.Context) {
	// Read text from the request body
	bodyBytes, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "cannot read body",
		})
		return
	}
	
	text := string(bodyBytes)
	if text == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "empty text body",
		})
		return
	}
	
	// Split into chunks
	chunks, err := splitText(text, 1000, 200)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("chunking failed: %v", err.Error()),
		})
		return
	}
	
	embeddings, err := app.embedText(chunks...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("embedding failed: %v", err.Error()),
		})
		return
	}
	
	ctx := context.Background()
	var points []*qdrant.PointStruct
	
	// Process each embedding
	for i, embedding := range embeddings {
		pointID, _ := uuid.NewV7()
		point := qdrant.PointStruct{
			Id:      qdrant.NewIDUUID(pointID.String()),
			Vectors: qdrant.NewVectors(embedding...),
			Payload: qdrant.NewValueMap(map[string]any{
				"text": chunks[i],
			}),
		}
		points = append(points, &point)
	}
	
	// Upsert to Qdrant
	_, err = app.qdrantClient.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: app.config.Qdrant.Collection,
		Points:         points,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Upsert failed: %s", err.Error()),
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"message": "Ingested successfully",
		"chunks":  len(chunks),
	})
}

type ChatRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type ChatResponse struct {
	Response string `json:"response"`
}

// Chat với Ollama
func (app *App) generateResponse(prompt string) (string, error) {
	reqBody := ChatRequest{
		Model:  app.config.Ollama.Model,
		Prompt: prompt,
		Stream: false,
	}
	
	var result ChatResponse
	
	_, err := app.restyClient.R().
		SetBody(reqBody).
		SetResult(&result).
		Post(app.config.Ollama.BaseURL + "/api/generate")
	if err != nil {
		return "", err
	}
	
	return result.Response, nil
}

func (app *App) chatHandler(c *gin.Context) {
	var req struct {
		Prompt string `json:"prompt"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request",
		})
		return
	}
	
	// Get embedding for prompt
	promptEmbedding, err := app.embedText(req.Prompt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Embeddings failed",
		})
		return
	}
	
	// Search similar vectors
	ctx := context.Background()
	searchResult, err := app.qdrantClient.Query(ctx, &qdrant.QueryPoints{
		CollectionName: app.config.Qdrant.Collection,
		Query:          qdrant.NewQuery(promptEmbedding[0]...),
		Limit:          qdrant.PtrOf(uint64(5)), // top 5 similar chunks
		WithPayload:    qdrant.NewWithPayload(true),
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Search failed",
		})
		return
	}
	
	// Build context from results
	var builderContext strings.Builder
	builderContext.WriteString("Context:\n")
	for _, point := range searchResult {
		if textValue, ok := point.Payload["text"]; ok {
			builderContext.WriteString(fmt.Sprintf("- %v\n", textValue.GetStringValue()))
		}
	}
	
	// Build prompt
	prompt := fmt.Sprintf(`Based on the above context, answer the question.
%s

Question: %s

Answer:`, builderContext.String(), req.Prompt)
	
	// Generate response
	answer, err := app.generateResponse(prompt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Generate response failed",
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"answer": answer,
	})
}
