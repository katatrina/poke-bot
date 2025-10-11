package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/katatrina/poke-bot/internal/config"
	"github.com/katatrina/poke-bot/internal/crawler"
	"github.com/katatrina/poke-bot/internal/model"
	"github.com/katatrina/poke-bot/internal/repository"
	"github.com/pkoukk/tiktoken-go"
	"github.com/tmc/langchaingo/textsplitter"
	"resty.dev/v3"
)

const (
	pokemonDBSource = "pokemondb"
)

var (
	// Global tokenizer instance for cl100k_base encoding (used by GPT-3.5 and GPT-4)
	tokenizer *tiktoken.Tiktoken
)

func init() {
	var err error
	tokenizer, err = tiktoken.GetEncoding("cl100k_base")
	if err != nil {
		log.Printf("Warning: failed to initialize tokenizer: %v. Token counting will use character approximation.", err)
	}
}

// countTokens counts the number of tokens in the given text
func countTokens(text string) int {
	if tokenizer == nil {
		// Fallback: approximate tokens as characters / 4
		return len(text) / 4
	}
	return len(tokenizer.Encode(text, nil, nil))
}

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

// ErrConversationTooLong is returned when conversation history exceeds the maximum allowed length
var ErrConversationTooLong = errors.New("conversation too long, please start a new chat session")

func (req *ChatRequest) Validate() error {
	// 1. Sanitize the current message
	req.Message = SanitizeInput(req.Message)

	// 2. Validate message length
	if len(req.Message) == 0 {
		return ErrEmptyMessage
	}
	if len(req.Message) > 1000 {
		return ErrMessageTooLong
	}

	// 3. Check for prompt injection attempts
	if DetectPromptInjection(req.Message) {
		return ErrPromptInjection
	}

	// 4. Validate conversation history length
	// Frontend sends sliding window of last N turns (max_history_turns * 2 messages)
	// Allow a bit more (15 messages = ~7 turns) to account for edge cases
	if len(req.ConversationHistory) > 15 {
		return errors.New("conversation history too long (max 15 messages)")
	}

	// 5. Sanitize and validate conversation history
	totalTokens := countTokens(req.Message)
	for i := range req.ConversationHistory {
		// Validate message type
		if req.ConversationHistory[i].Type != "user" && req.ConversationHistory[i].Type != "assistant" {
			return fmt.Errorf("invalid message type: %s", req.ConversationHistory[i].Type)
		}

		// Sanitize content
		req.ConversationHistory[i].Content = SanitizeInput(req.ConversationHistory[i].Content)

		// Check for prompt injection in history
		if DetectPromptInjection(req.ConversationHistory[i].Content) {
			return fmt.Errorf("conversation history contains suspicious patterns")
		}

		// Validate length
		if len(req.ConversationHistory[i].Content) > 2000 {
			return errors.New("conversation message too long (max 2000 characters)")
		}

		totalTokens += countTokens(req.ConversationHistory[i].Content)
	}

	// 6. Hard limit on total tokens (2500 tokens for conversation)
	if totalTokens > 2500 {
		return ErrConversationTooLong
	}

	return nil
}

type ChatResponse struct {
	Response string `json:"response"`
	Context  string `json:"context"`
}

func (s *RAGService) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	// Generate embedding for user query
	embeddings, err := s.generateEmbeddings([]string{req.Message})
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	// Add timeout
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

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


// buildPromptWithHistory builds the prompt with smart truncation to fit within context window
// Priority: Instructions > Current Question > Recent History > RAG Context
func (s *RAGService) buildPromptWithHistory(ragContext, question string, conversationHistory []ConversationMessage) string {
	// Get max context tokens from config
	maxContextTokens := s.config.RAG.MaxContextTokens
	if maxContextTokens == 0 {
		maxContextTokens = 4000 // Default fallback
	}

	// Define fixed components (highest priority)
	systemPrompt := "You are a helpful Pokemon expert assistant. Answer questions based on the provided context about Pokemon.\n\n"
	instructions := "\nInstructions:\n" +
		"- Answer based on the context above and conversation history\n" +
		"- Use conversation context to understand references (it, that Pokemon, etc.)\n" +
		"- Be specific and accurate about Pokemon stats, types, and abilities\n" +
		"- If comparing Pokemon, use specific numbers when available\n" +
		"- If the context doesn't contain the information, say so clearly\n" +
		"- Keep your answer concise but informative\n\n" +
		"Answer:"

	// Count tokens for fixed components (always included)
	questionWithLabel := fmt.Sprintf("Current Question: %s\n", question)
	tokensUsed := countTokens(systemPrompt + questionWithLabel + instructions)

	// Fit as much recent history as possible (second priority)
	recentHistory := []ConversationMessage{}
	historyTruncated := false
	for i := len(conversationHistory) - 1; i >= 0; i-- {
		role := "Human"
		if conversationHistory[i].Type == "assistant" {
			role = "Assistant"
		}
		msgText := fmt.Sprintf("%s: %s\n", role, conversationHistory[i].Content)
		msgTokens := countTokens(msgText)

		if tokensUsed+msgTokens > maxContextTokens {
			historyTruncated = true
			break
		}

		recentHistory = append([]ConversationMessage{conversationHistory[i]}, recentHistory...)
		tokensUsed += msgTokens
	}

	// Calculate remaining tokens for RAG context
	remainingTokens := maxContextTokens - tokensUsed
	if remainingTokens < 0 {
		remainingTokens = 0
	}

	// Truncate RAG context if needed (lowest priority)
	truncatedRagContext, ragTruncated := s.truncateToTokens(ragContext, remainingTokens)

	// Log truncation for monitoring
	if historyTruncated {
		log.Printf("Truncated conversation history from %d to %d messages",
			len(conversationHistory), len(recentHistory))
	}
	if ragTruncated {
		originalTokens := countTokens(ragContext)
		log.Printf("Truncated RAG context from %d to %d tokens", originalTokens, remainingTokens)
	}

	// Build final prompt
	var promptBuilder strings.Builder
	promptBuilder.WriteString(systemPrompt)

	// Add RAG context
	if len(truncatedRagContext) > 0 {
		if ragTruncated {
			promptBuilder.WriteString("Context Information (truncated):\n\n")
		} else {
			promptBuilder.WriteString("Context Information:\n\n")
		}
		promptBuilder.WriteString(truncatedRagContext)
		promptBuilder.WriteString("\n")
	}

	// Add conversation history
	if len(recentHistory) > 0 {
		if historyTruncated {
			promptBuilder.WriteString("=== Recent Conversation (earlier messages omitted) ===\n")
		} else {
			promptBuilder.WriteString("=== Recent Conversation ===\n")
		}
		for _, msg := range recentHistory {
			role := "Human"
			if msg.Type == "assistant" {
				role = "Assistant"
			}
			promptBuilder.WriteString(fmt.Sprintf("%s: %s\n", role, msg.Content))
		}
		promptBuilder.WriteString("\n")
	}

	// Add current question and instructions
	promptBuilder.WriteString(questionWithLabel)
	promptBuilder.WriteString(instructions)

	return promptBuilder.String()
}

// truncateToTokens truncates text to fit within a token budget
// Returns the truncated text and whether truncation occurred
func (s *RAGService) truncateToTokens(text string, maxTokens int) (string, bool) {
	if maxTokens <= 0 {
		return "", true
	}

	currentTokens := countTokens(text)
	if currentTokens <= maxTokens {
		return text, false
	}

	// Binary search for the right length
	// Approximate: 1 token ≈ 4 characters
	estimatedChars := maxTokens * 4
	if estimatedChars > len(text) {
		estimatedChars = len(text)
	}

	// Start with estimated length and adjust
	low, high := 0, len(text)
	result := ""

	for low < high {
		mid := (low + high + 1) / 2
		if mid > len(text) {
			mid = len(text)
		}

		candidate := text[:mid]
		tokens := countTokens(candidate)

		if tokens <= maxTokens {
			result = candidate
			low = mid
		} else {
			high = mid - 1
		}

		// Prevent infinite loop
		if high-low < 10 {
			break
		}
	}

	// Add truncation indicator
	if len(result) < len(text) {
		result += "\n... (content truncated)"
	}

	return result, true
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
