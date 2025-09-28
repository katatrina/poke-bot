package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	
	"github.com/google/uuid"
	"github.com/katatrina/poke-bot/internal/config"
	"github.com/katatrina/poke-bot/internal/crawler"
	"github.com/katatrina/poke-bot/internal/model"
	"github.com/katatrina/poke-bot/internal/repository"
	"github.com/tmc/langchaingo/textsplitter"
	"resty.dev/v3"
)

const (
	pokemonDBSource = "pokemondb"
)

type RAGService struct {
	config     *config.Config
	vectorRepo *repository.VectorRepository
	restClient *resty.Client
	crawler    *crawler.PokemonDBCrawler
}

func NewRAGService(
	cfg *config.Config,
	vectorRepo *repository.VectorRepository,
	restClient *resty.Client,
) *RAGService {
	return &RAGService{
		config:     cfg,
		vectorRepo: vectorRepo,
		restClient: restClient,
		crawler:    crawler.NewPokemonDBCrawler(),
	}
}

type IngestRequest struct {
	Source     string `json:"source,omitempty"` // "pokemondb" or "text"
	CrawlLimit int    `json:"crawl_limit"`      // Number of Pokemon to crawl (default 10)
	StartFrom  int    `json:"start_from"`       // Start from Pokemon number (for pagination)
}

func (req *IngestRequest) Validate() error {
	if req.Source != pokemonDBSource {
		return fmt.Errorf("unsupported source: %s (must be 'pokemondb')", req.Source)
	}
	
	if req.CrawlLimit <= 0 {
		req.CrawlLimit = 10 // Default to 10 Pokemon
	}
	
	if req.CrawlLimit > 151 {
		req.CrawlLimit = 151 // Max Gen 1 Pokemon
	}
	
	return nil
}

func (s *RAGService) IngestPokemonData(ctx context.Context, req *IngestRequest) error {
	log.Printf("Starting Pokemon crawl with limit=%d", req.CrawlLimit)
	
	// Step 1: Get list of Pokemon URLs
	pokemonURLs, err := s.crawler.CrawlPokemonList(ctx, req.CrawlLimit)
	if err != nil {
		return fmt.Errorf("failed to crawl pokemon list: %w", err)
	}
	
	log.Printf("Found %d Pokemon URLs to crawl", len(pokemonURLs))
	
	// Process start_from if specified
	if req.StartFrom > 0 && req.StartFrom < len(pokemonURLs) {
		pokemonURLs = pokemonURLs[req.StartFrom:]
	}
	
	successCount := 0
	failCount := 0
	
	// Step 2: Crawl each Pokemon and ingest
	for i, url := range pokemonURLs {
		log.Printf("Crawling Pokemon %d/%d: %s", i+1, len(pokemonURLs), url)
		
		// Crawl Pokemon details
		pokemonData, err := s.crawler.CrawlPokemonDetails(ctx, url)
		if err != nil {
			log.Printf("Failed to crawl %s: %v", url, err)
			failCount++
			continue
		}
		
		// Format Pokemon data for RAG
		content := s.crawler.FormatPokemonForRAG(pokemonData)
		
		// Split into chunks if needed
		chunks, err := s.splitText(content)
		if err != nil {
			log.Printf("Failed to split text for %s: %v", pokemonData.Name, err)
			failCount++
			continue
		}
		
		// Generate embeddings
		embeddings, err := s.generateEmbeddings(chunks)
		if err != nil {
			log.Printf("Failed to generate embeddings for %s: %v", pokemonData.Name, err)
			failCount++
			continue
		}
		
		// Create documents
		var documents []model.Document
		for j, chunk := range chunks {
			documentID, _ := uuid.NewV7()
			doc := model.Document{
				ID:      documentID,
				Content: chunk,
				Metadata: map[string]string{
					"source":  pokemonDBSource,
					"pokemon": pokemonData.Name,
					"number":  pokemonData.Number,
					"types":   strings.Join(pokemonData.Types, ","),
					"chunk":   fmt.Sprintf("%d/%d", j+1, len(chunks)),
				},
			}
			documents = append(documents, doc)
		}
		
		// Store in vector database
		if err = s.vectorRepo.Upsert(ctx, documents, embeddings); err != nil {
			log.Printf("Failed to store %s: %v", pokemonData.Name, err)
			failCount++
			continue
		}
		
		successCount++
		log.Printf("Successfully ingested %s (%d chunks)", pokemonData.Name, len(chunks))
	}
	
	log.Printf("Pokemon crawl completed: %d success, %d failed", successCount, failCount)
	
	if successCount == 0 {
		return fmt.Errorf("failed to ingest any Pokemon data")
	}
	
	return nil
}

func (s *RAGService) splitText(text string) ([]string, error) {
	// For smaller Pokemon entries, don't split unnecessarily
	if len(text) < s.config.RAG.ChunkSize {
		return []string{text}, nil
	}
	
	splitter := textsplitter.NewRecursiveCharacter(
		textsplitter.WithChunkSize(s.config.RAG.ChunkSize),
		textsplitter.WithChunkOverlap(s.config.RAG.ChunkOverlap),
		textsplitter.WithSeparators([]string{"\n\n===", "\n\n", "\n", ". ", " "}),
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
	
	resp, err := s.restClient.R().
		SetBody(reqBody).
		SetResult(&result).
		Post(s.config.Ollama.BaseURL + "/api/embed")
	
	if err != nil {
		return nil, err
	}
	
	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("embedding API returned status %d: %s", resp.StatusCode(), resp.String())
	}
	
	if len(result.Embeddings) == 0 {
		return nil, errors.New("no embeddings returned from API")
	}
	
	return result.Embeddings, nil
}

type ConversationMessage struct {
	Type    string `json:"type"` // "user" | "assistant"
	Content string `json:"content"`
}

type ChatRequest struct {
	Message             string                `json:"message"`
	ConversationHistory []ConversationMessage `json:"conversation_history"`
}

func (req ChatRequest) Validate() error {
	// Validate message length
	if len(req.Message) == 0 {
		return errors.New("message cannot be empty")
	}
	
	if len(req.Message) > 1000 {
		return model.ErrMessageTooLong
	}
	
	// Validate conversation history
	if len(req.ConversationHistory) > 20 {
		return errors.New("conversation history too long (max 20 messages)")
	}
	
	for _, msg := range req.ConversationHistory {
		if msg.Type != "user" && msg.Type != "assistant" {
			return fmt.Errorf("invalid message type: %s (must be 'user' or 'assistant')", msg.Type)
		}
		
		if len(msg.Content) > 2000 {
			return errors.New("conversation message too long (max 2000 characters)")
		}
	}
	
	return nil
}

type ChatResponse struct {
	Response string   `json:"response"`
	Sources  []string `json:"sources"`
	Context  string   `json:"context"`
}

func (s *RAGService) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	// Generate embedding for user query
	embeddings, err := s.generateEmbeddings([]string{req.Message})
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}
	
	// Search for relevant documents
	searchResults, err := s.vectorRepo.Search(ctx, embeddings[0], s.config.RAG.TopK)
	if err != nil {
		return nil, fmt.Errorf("failed to search documents: %w", err)
	}
	
	// Build RAG context from search results
	ragContext := s.buildRAGContext(searchResults)
	
	// Build prompt with conversation history
	prompt := s.buildPromptWithHistory(ragContext, req.Message, req.ConversationHistory)
	
	// Generate response from LLM
	resp, err := s.generateResponse(prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to generate response: %w", err)
	}
	
	return &ChatResponse{
		Response: resp,
		Sources:  s.extractSources(searchResults),
		Context:  req.Message, // Store for follow-up questions
	}, nil
}

func (s *RAGService) buildRAGContext(searchResults []model.SearchResult) string {
	var contextBuilder strings.Builder
	var sources []string
	seenSources := make(map[string]bool)
	
	contextBuilder.WriteString("Context Information:\n\n")
	for i, result := range searchResults {
		contextBuilder.WriteString(fmt.Sprintf("[%d] %s\n\n", i+1, result.Content))
		
		// Collect unique sources
		if pokemon, ok := result.Metadata["pokemon"]; ok && pokemon != "" {
			sourceStr := fmt.Sprintf("Pokemon: %s", pokemon)
			if !seenSources[sourceStr] {
				sources = append(sources, sourceStr)
				seenSources[sourceStr] = true
			}
		}
	}
	
	return contextBuilder.String()
}

// New method to extract sources
func (s *RAGService) extractSources(searchResults []model.SearchResult) []string {
	var sources []string
	seenSources := make(map[string]bool)
	
	for _, result := range searchResults {
		if pokemon, ok := result.Metadata["pokemon"]; ok && pokemon != "" {
			sourceStr := fmt.Sprintf("Pokemon: %s", pokemon)
			if !seenSources[sourceStr] {
				sources = append(sources, sourceStr)
				seenSources[sourceStr] = true
			}
		} else if source, ok := result.Metadata["source"]; ok && source != "" {
			if !seenSources[source] {
				sources = append(sources, source)
				seenSources[source] = true
			}
		}
	}
	
	return sources
}

// Update buildPrompt method to handle conversation history
func (s *RAGService) buildPromptWithHistory(ragContext, question string, conversationHistory []ConversationMessage) string {
	var promptBuilder strings.Builder
	
	promptBuilder.WriteString("You are a helpful Pokemon expert assistant. Answer questions based on the provided context about Pokemon.\n\n")
	
	// Add RAG context
	promptBuilder.WriteString(ragContext)
	promptBuilder.WriteString("\n")
	
	// Add conversation history if available
	if len(conversationHistory) > 0 {
		promptBuilder.WriteString("=== Recent Conversation ===\n")
		for _, msg := range conversationHistory {
			role := "Human"
			if msg.Type == "assistant" {
				role = "Assistant"
			}
			promptBuilder.WriteString(fmt.Sprintf("%s: %s\n", role, msg.Content))
		}
		promptBuilder.WriteString("\n")
	}
	
	promptBuilder.WriteString(fmt.Sprintf("Current Question: %s\n\n", question))
	promptBuilder.WriteString("Instructions:\n")
	promptBuilder.WriteString("- Answer based on the context above and conversation history\n")
	promptBuilder.WriteString("- Use conversation context to understand references (it, that Pokemon, etc.)\n")
	promptBuilder.WriteString("- Be specific and accurate about Pokemon stats, types, and abilities\n")
	promptBuilder.WriteString("- If comparing Pokemon, use specific numbers when available\n")
	promptBuilder.WriteString("- If the context doesn't contain the information, say so clearly\n")
	promptBuilder.WriteString("- Keep your answer concise but informative\n\n")
	promptBuilder.WriteString("Answer:")
	
	return promptBuilder.String()
}

type OllamaChatRequest struct {
	Model   string                 `json:"model"`
	Prompt  string                 `json:"prompt"`
	Stream  bool                   `json:"stream"`
	Options map[string]interface{} `json:"options,omitempty"`
}

type OllamaChatResponse struct {
	Response string `json:"response"`
}

func (s *RAGService) generateResponse(prompt string) (string, error) {
	reqBody := OllamaChatRequest{
		Model:  s.config.Ollama.ChatModel,
		Prompt: prompt,
		Stream: false,
		Options: map[string]interface{}{
			"temperature": 0.3, // Lower temperature for factual responses
			"top_p":       0.9,
		},
	}
	
	var result OllamaChatResponse
	resp, err := s.restClient.R().
		SetBody(reqBody).
		SetResult(&result).
		Post(s.config.Ollama.BaseURL + "/api/generate")
	
	if err != nil {
		return "", err
	}
	
	if resp.StatusCode() != 200 {
		return "", fmt.Errorf("chat API returned status %d: %s", resp.StatusCode(), resp.String())
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
