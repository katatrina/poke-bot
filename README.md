# 🤖 Pokemon RAG Chatbot

A production-ready RAG (Retrieval-Augmented Generation) chatbot that answers questions about Pokemon using local LLMs and vector search.

## ✨ Features

- **Semantic Search**: Uses vector embeddings (nomic-embed-text) for accurate Pokemon information retrieval
- **Local LLM**: Runs entirely locally using Ollama (qwen2.5-coder:3b)
- **Real-time Crawling**: Automatically crawls and indexes Pokemon data from PokemonDB
- **Conversation History**: Maintains context across multiple questions
- **Type Safety**: Full TypeScript support for frontend
- **Scalable**: Vector database with Qdrant for fast similarity search

## 🏗️ Architecture

```
┌─────────────┐      ┌──────────────┐      ┌─────────────┐
│   React UI  │ ──── │ Go REST API  │ ──── │   Qdrant    │
│  (Frontend) │ HTTP │  (Backend)   │      │  (Vectors)  │
└─────────────┘      └──────────────┘      └─────────────┘
                              │
                              │ HTTP
                              ▼
                     ┌──────────────┐
                     │    Ollama    │
                     │   (LLM +     │
                     │  Embeddings) │
                     └──────────────┘
```

**Tech Stack:**
- **Backend**: Go 1.25, Gin, Qdrant Go Client, Langchain Go
- **Frontend**: React 19, TypeScript, Vite, TailwindCSS
- **ML/AI**: Ollama (qwen2.5-coder:3b, nomic-embed-text)
- **Vector DB**: Qdrant (768-dim embeddings, Cosine similarity)
- **Scraping**: Colly v2

## 🚀 Quick Start

### Prerequisites
- Docker & Docker Compose
- Go 1.25+ (for local dev)
- Node.js 20+ (for frontend dev)

### 1. Start Infrastructure

```bash
# Start Qdrant + Ollama
make qdrant
make ollama

# Or use docker-compose
docker-compose up -d
```

### 2. Run Backend

```bash
go run .
```

### 3. Ingest Pokemon Data

```bash
curl -X POST http://localhost:8080/api/v1/ingest \
  -H "Content-Type: application/json" \
  -d '{"source": "pokemondb", "crawl_limit": 151}'
```

### 4. Chat with the Bot

```bash
curl -X POST http://localhost:8080/api/v1/chat \
  -H "Content-Type: application/json" \
  -d '{
    "message": "What are Charizard base stats?",
    "conversation_history": []
  }'
```

### 5. Run Frontend (Optional)

```bash
cd web
npm install
npm run dev
```

Visit: http://localhost:5173

## 📚 API Endpoints

### Health Check

```http
GET /api/v1/health
```

### Ingest Pokemon Data

```http
POST /api/v1/ingest
Content-Type: application/json
```

Request body:
```json
{
  "source": "pokemondb",
  "crawl_limit": 151,
  "start_from": 0
}
```

### Chat

```http
POST /api/v1/chat
Content-Type: application/json
```

Request body:
```json
{
  "message": "Which Pokemon is strongest against Fire types?",
  "conversation_history": [
    {
      "type": "user",
      "content": "Tell me about Pikachu"
    },
    {
      "type": "assistant", 
      "content": "Pikachu is an Electric-type Pokemon..."
    }
  ]
}
```

Response:
```json
{
  "response": "Water, Rock, and Ground type Pokemon are strongest...",
  "sources": ["Pokemon: Blastoise", "Pokemon: Geodude"],
  "context": "Which Pokemon is strongest against Fire types?"
}
```

## 🔧 Configuration

Edit `config.yaml`:

```yaml
server:
  port: 8080

qdrant:
  host: "localhost"
  port: 6334
  collection: "pokemons"

ollama:
  base_url: "http://localhost:11434"
  chat_model: "qwen2.5-coder:3b"
  embedding_model: "nomic-embed-text"

rag:
  chunk_size: 600
  chunk_overlap: 100
  top_k: 5
  temperature: 0.3
```

## 🧪 Example Queries

- "What type is Charizard?"
- "Compare Pikachu and Raichu stats"
- "Which Pokemon evolves into Gyarados?"
- "What are Mewtwo abilities?"
- "What is Dragonite weak against?"

## 📁 Project Structure

```
.
├── internal/
│   ├── config/          # Configuration loading
│   ├── crawler/         # PokemonDB web scraping
│   ├── handler/         # HTTP handlers
│   ├── model/           # Domain models
│   ├── repository/      # Vector DB operations
│   ├── server/          # HTTP server setup
│   └── service/         # Business logic (RAG)
├── web/                 # React frontend
├── config.yaml          # Application config
├── Makefile             # Common tasks
└── main.go              # Entry point
```

## 🎯 How RAG Works

**Ingestion Phase:**
1. Crawl Pokemon data from PokemonDB
2. Split text into chunks (600 chars, 100 overlap)
3. Generate embeddings using nomic-embed-text (768-dim)
4. Store vectors in Qdrant with metadata

**Query Phase:**
1. User asks question
2. Generate query embedding
3. Search top-K similar vectors (cosine similarity)
4. Build context from retrieved chunks
5. Send context + question to LLM
6. Return generated response + sources

## 🛠️ Development

### Run Tests

```bash
make test
```

### View Test Coverage

```bash
make test-coverage
```

### Start with Docker Compose

```bash
make docker-up
```

### View Logs

```bash
make docker-logs
```

### Clean Up

```bash
make clean
```

## 📝 License

MIT

---

⚡ Built with Go, React, and local AI - no API keys needed!