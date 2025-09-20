package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	
	"github.com/google/uuid"
	"github.com/katatrina/poke-bot/internal/config"
	"github.com/katatrina/poke-bot/internal/model"
	"github.com/katatrina/poke-bot/internal/repository"
	"github.com/tmc/langchaingo/textsplitter"
	"resty.dev/v3"
)

type RAGService struct {
	config     *config.Config
	vectorRepo *repository.VectorRepository
	restClient *resty.Client
}

func NewRAGService(cfg *config.Config, vectorRepo *repository.VectorRepository, restClient *resty.Client) *RAGService {
	return &RAGService{
		config:     cfg,
		vectorRepo: vectorRepo,
		restClient: restClient,
	}
}

type IngestRequest struct {
	Content     string `json:"content"`
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
}

func (req *IngestRequest) Validate() error {
	if req.ContentType != "text" {
		return fmt.Errorf("unsupported content type: %s", req.ContentType)
	}
	
	if len(req.Content) == 0 {
		return model.ErrEmptyDocContent
	}
	
	if len(req.Content) > 10*1024*1024 { // 10MB limit
		return model.ErrDocContentTooLarge
	}
	
	return nil
}

func (s *RAGService) IngestDocument(ctx context.Context, req *IngestRequest) error {
	// Split text into chunks
	chunks, err := s.splitText(req.Content)
	if err != nil {
		return fmt.Errorf("failed to split text: %w", err)
	}
	
	// Generate embeddings for chunks
	embeddings, err := s.generateEmbeddings(chunks)
	if err != nil {
		return fmt.Errorf("failed to generate embeddings: %s", err)
	}
	
	// Convert chunks to documents
	var documents []model.Document
	for _, chunk := range chunks {
		documentID, _ := uuid.NewV7()
		doc := model.Document{
			ID:      documentID,
			Content: chunk,
			Metadata: map[string]string{
				"source": req.Filename,
			},
		}
		documents = append(documents, doc)
	}
	
	// Store in vector db
	if err = s.vectorRepo.Upsert(ctx, documents, embeddings); err != nil {
		return fmt.Errorf("failed to store documents: %w", err)
	}
	
	return nil
}

func (s *RAGService) splitText(text string) ([]string, error) {
	splitter := textsplitter.NewRecursiveCharacter(
		textsplitter.WithChunkSize(s.config.RAG.ChunkSize),
		textsplitter.WithChunkOverlap(s.config.RAG.ChunkOverlap),
		textsplitter.WithSeparators([]string{"\n\n", "\n", ". ", " "}),
	)
	
	chunks, err := splitter.SplitText(text)
	if err != nil {
		return nil, err
	}
	
	return chunks, nil
}

type OllamaEmbedRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

type OllamaEmbedResponse struct {
	Embeddings [][]float32 `json:"embeddings"`
}

func (s *RAGService) generateEmbeddings(texts []string) ([][]float32, error) {
	reqBody := OllamaEmbedRequest{
		Model: s.config.Ollama.EmbeddingModel,
		Input: texts,
	}
	
	var result OllamaEmbedResponse
	
	_, err := s.restClient.R().
		SetBody(reqBody).
		SetResult(&result).
		Post(s.config.Ollama.BaseURL + "/api/embed")
	if err != nil {
		return nil, err
	}
	
	return result.Embeddings, nil
}

type ChatRequest struct {
	Message string `json:"message"`
	Context string `json:"context"`
}

func (req ChatRequest) Validate() error {
	// Validate message length
	if len(req.Message) == 0 {
		return errors.New("message cannot be empty")
	}

	if len(req.Message) > 1000 {
		return model.ErrMessageTooLong
	}

	return nil
}

type ChatResponse struct {
	Response string   `json:"response"`
	Sources  []string `json:"sources"`
	Context  string   `json:"context"`
}

func (s *RAGService) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	// Generate embedding for user usage
	embeddings, err := s.generateEmbeddings([]string{req.Message})
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}
	
	// Search for relevant documents
	searchResults, err := s.vectorRepo.Search(ctx, embeddings[0], s.config.RAG.TopK)
	if err != nil {
		return nil, fmt.Errorf("failed to search documents: %w", err)
	}
	
	// Build context from search results
	var contextBuilder strings.Builder
	var sources []string
	
	contextBuilder.WriteString("Context:\n")
	for i, result := range searchResults {
		contextBuilder.WriteString(fmt.Sprintf("%d. %s\n", i+1, result.Content))
		if source, ok := result.Metadata["source"]; ok && source != "" {
			sources = append(sources, source)
		}
	}
	
	// Build prompt for LLM
	prompt := s.buildPrompt(contextBuilder.String(), req.Message, req.Context)
	
	// Generate response from LLM
	resp, err := s.generateResponse(prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to generate response: %w", err)
	}
	
	// Remove duplicate sources
	sources = removeDuplicates(sources)
	
	return &ChatResponse{
		Response: resp,
		Sources:  sources,
		Context:  req.Message, // Simple context for follow-up questions
	}, nil
}

func (s *RAGService) buildPrompt(context, question, previousContext string) string {
	var promptBuilder strings.Builder
	
	promptBuilder.WriteString("You are a helpful AI assistant. Answer the question based on the provided context.\n\n")
	promptBuilder.WriteString(context)
	promptBuilder.WriteString("\n")
	
	if previousContext != "" {
		promptBuilder.WriteString(fmt.Sprintf("Previous question: %s\n", previousContext))
	}
	
	promptBuilder.WriteString(fmt.Sprintf("Question: %s\n\n", question))
	promptBuilder.WriteString("Answer based on the context above. If the context doesn't contain relevant information, say so clearly.\n\n")
	promptBuilder.WriteString("Answer:")
	
	return promptBuilder.String()
}

type OllamaChatRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type OllamaChatResponse struct {
	Response string `json:"response"`
}

func (s *RAGService) generateResponse(prompt string) (string, error) {
	reqBody := OllamaChatRequest{
		Model:  s.config.Ollama.ChatModel,
		Prompt: prompt,
		Stream: false,
	}
	
	var result OllamaChatResponse
	_, err := s.restClient.R().
		SetBody(reqBody).
		SetResult(&result).
		Post(s.config.Ollama.BaseURL + "/api/generate")
	if err != nil {
		return "", err
	}
	
	return result.Response, nil
}

// Helper function to remove duplicate strings
func removeDuplicates(slice []string) []string {
	keys := make(map[string]bool)
	var result []string
	
	for _, item := range slice {
		if !keys[item] {
			keys[item] = true
			result = append(result, item)
		}
	}
	
	return result
}
