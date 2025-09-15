# Tài Liệu Tổng Quan Phát Triển Dự Án RAG Chatbot Bằng Golang

---

## 1. Mục Tiêu Dự Án

Xây dựng một hệ thống chatbot truy xuất tăng cường (Retrieval-Augmented Generation - RAG) nội bộ ở mức độ POC. Hệ thống phát triển chủ yếu bằng **Golang**, dùng các công nghệ hiện đại như **LangChainGo**, vector database **Qdrant**, và LLM backend (Ollama/OpenAI API).

Mục đích là có một chatbot nội bộ hiệu quả, dễ mở rộng, dễ phát triển, vận hành bằng một cli, phù hợp cho các use case thực tế với ingest, embedding, indexing và chat.

---

## 2. Kiến Trúc Tổng Thể

### 2.1. Các Thành Phần Chính

| Thành phần | Mô tả |
| --- | --- |
| Go RAG Chatbot Server | Trung tâm orchestrator, xử lý logic RAG pipeline với các module ingest, chat. |
| Qdrant Vector Database | Lưu trữ vector embeddings, hỗ trợ truy vấn tương đồng hiệu quả. |
| Ollama LLM Service | Cung cấp khả năng embedding và text generation qua API. |
| LangChainGo | Thư viện quản lý luồng pipeline RAG, prompt, kết hợp vector search và LLM. |
| Document Sources | Ban đầu chỉ hỗ trợ 1 kiểu file như pdf, nhưng sẽ thiết kế để dễ mở rộng sau này, được định nghĩa trong cấu hình `kb-config.yaml`. |
| Client/User | Giao diện người dùng qua HTTP REST API, frontend Alpine.js nhẹ nhàng và responsive. |

### 2.2. Áp Dụng Modular Monolith cho toàn bộ hệ thống & Hexagonal Architecture chỗ mỗi module

- **Infrastructure Layer (Driving Ports):**
    - `controller` (HTTP, CLI handlers)
    - `repository` (Qdrant, LLM clients, file loaders...)
- **Service Layer:**
    - Xử lý business logic, chia chunk dữ liệu, gọi embedding, tìm kiếm vector, xây dựng prompt, gọi LLM.
    - Ứng dụng LangChainGo tạo thành pipeline hiệu quả, giảm boilerplate.
- **Model:** Các entity domain riêng biệt, error, event...

⇒ Giúp tách biệt business logic với framework details (infrastructure)

---

## 3. Chiến Lược Sử Dụng LangChainGo

### 3.1. Sử Dụng LangChainGo (Tích Cực)

### Core RAG Components:

- **Embeddings:** `langchaingo/embeddings` cho embedding generation
- **Vector Stores:** `langchaingo/vectorstores` cho similarity search
- **Chains:** `chains.LoadStuffQA()` cho RAG question-answering
- **Document Processing:** `documentloaders` + `textsplitter` cho text chunking
- **LLM Integration:** Hỗ trợ cả Ollama và OpenAI thông qua langchaingo wrappers

### Workflow được cover:

- Document loading và splitting
- Embedding generation và vector storage
- Retrieval-augmented chat với StuffDocuments chain
- Vector similarity search với score thresholds

### 3.2. Custom Implementation (Linh hoạt)

### Business Logic Layer:

- HTTP server và API endpoints hoàn toàn custom
- Configuration management (YAML configs)
- Web crawling logic với Colly + custom filtering

### Adapter Pattern (ví dụ mẫu):

```go
// interfaces không phụ thuộc vào langchaingo
type LLMHandler interface {
    Chat(ctx context.Context, vectorStore vectorstores.VectorStore,
         query string) (response string, err error)
    // ...
}

```

### Provider Flexibility:

- Support cả Ollama (local) và OpenAI API
- Vector DB abstraction (hiện tại Qdrant, dễ thêm khác)

### 3.3. Đánh Giá Về Flexibility

**✅ Ưu điểm:**

- Langchaingo chỉ dùng cho "heavy lifting" components (embeddings, chains, document processing)
- Business logic và API layer hoàn toàn độc lập
- Adapter pattern cho dễ swap implementations
- TODO comments shows họ cũng muốn reduce dependency: `//TODO it should not be specific to langchain`

**⚠️ Dependency concerns:**

- Core interfaces vẫn expose langchaingo types (`vectorstores.VectorStore`, `schema.Document`)
- Khó migrate sang framework khác mà không refactor interfaces

**💡 Recommendations:**

1. **Dùng langchaingo cho:** Document processing, embeddings, basic RAG chains
2. **Tự implement:** HTTP APIs, business logic, configuration, custom workflows
3. **Abstraction strategy:** Tạo internal types và map từ/sang langchaingo types ở boundary layer

---

## 4. Quy Trình Hệ Thống

| Bước quy trình | Mô tả |
| --- | --- |
| Ingest | Crawl hoặc đọc tài liệu theo config (kb-config.yaml), chia chunk, embed, upsert vector vào Qdrant. Hỗ trợ tự động hóa qua flag. |
| Chat | Nhận query, embed câu hỏi, truy tìm vector top-k, tổng hợp thông tin bằng LangChainGo, gọi LLM, phản hồi streaming. |

---

## 5. API Exposed

| Endpoint | Mô tả |
| --- | --- |
| `/health` | Kiểm tra tình trạng hệ thống |
| `/crawl-docs` | Crawl từ kb-config.yaml và thực hiện ingest |
| `/add-docs` | Nhận docs trực tiếp từ request body và thực hiện ingest |
| `/chat` | Nhận truy vấn, trả lời chat dựa trên RAG pipeline. |

---

## 6. Cobra CLI với Flag `load-kb`

- Dự án sử dụng **Cobra** để quản lý CLI trong một binary duy nhất.
- Chatbot server hỗ trợ flag boolean `-load-kb` (mặc định `false`), khi bật (`true`), server tự động crawl và ingest tài liệu theo `kb-config.yaml` lúc khởi động.
- Ví dụ chạy server đồng thời ingest tự động:

    ```bash
    ./main --load-kb=true
    ```


---

## 7. Công Nghệ & Thư Viện Chủ Đạo

| Công nghệ/thư viện | Vai trò |
| --- | --- |
| Golang | Ngôn ngữ chính, modular monolith |
| Cobra | CLI |
| Gin | HTTP server handlers |
| net/http | Kết hợp với gin để triển khai graceful shutdown |
| Viper | Cấu hình |
| log/slog | Logging |
| LangChainGo | Pipeline RAG, quản lý prompt, dịch vụ LLM |
| Qdrant | Vector database |
| Ollama / OpenAI | LLM API, embedding API |
| Alpine.js | Frontend chat nhẹ |

---

## 8. Project Structure

```
rag-chatbot-go/
├── cmd/
│   └── root.go                    # Cobra CLI setup, call each module setup
│
├── internal/
│   ├── modules/                   # Các module nghiệp vụ theo chiều dọc
│   │   ├── ingest/                # Module ingest tài liệu
│   │   │   ├── controller.go      # HTTP handlers, CLI handlers
│   │   │   ├── service.go         # Business logic ingest, chunking, embedding
│   │   │   ├── repository.go      # Qdrant interaction, file readers
│   │   │   ├── model.go           # Domain types (Document, Chunk, etc.)
│   │   │   └── module.go          # Setup module, wire dependencies
│   │   ├── chat/                  # Module chat realtime
│   │   │   ├── controller.go      # HTTP handlers cho /chat API
│   │   │   ├── service.go         # Chat logic, LangChainGo integration
│   │   │   ├── repository.go      # Vector search, LLM calls
│   │   │   ├── model.go           # Domain types (Query, Response, Session)
│   │   │   └── module.go          # Setup module, wire dependencies
│   │
│   ├── shared/                    # Shared utilities, common components
│   │   ├── config/                # Configuration management
│   │   │   ├── config.go          # Config struct definitions
│   │   │
│   │   ├── logger/                # Logging wrapper
│   │   │   └── logger.go          # log/slog logger setup and utilities
│   │   │
│   │   ├── clients/               # External service clients
│   │   │   ├── qdrant.go          # Qdrant client wrapper
│   │   │   ├── ollama.go          # Ollama client wrapper
│   │   │   └── langchain.go       # LangChainGo integration
│   │   │
│   │   └── types/                 # Shared common types
│   │       ├── errors.go          # Common error types
│   │       ├── response.go        # API response wrappers
│   │       └── const.go           # System constants
│   │
│
├── web/                           # Frontend static files
│   ├── index.html                 # Main HTML page with Alpine.js
│   ├── static/
│   │   ├── css/
│   │   │   └── style.css          # Simple styling
│   │   └── js/
│   │       └── chat.js            # Alpine.js chat component
│   └── templates/                 # HTML templates (nếu cần)
│
├── configs/                       # Configuration files
│   ├── app-config.yaml            # Main application config
│   ├── kb-config.yaml             # Knowledge base sources config
│
├── .gitignore                     # Git ignore file
├── main.go                        # Main entry point, call Cobra root command
├── go.mod                         # Go module definition
├── go.sum                         # Go module dependencies
├── Makefile                       # Build and development commands
└── README.md                      # Project documentation

```

---

## 9. Roadmap & Future Enhancements

### Phase 1 - POC (Current)

- ✅ Basic RAG pipeline với LangChainGo
- ✅ Qdrant vector database integration
- ✅ Support Ollama/OpenAI providers
- ✅ Simple web UI với Alpine.js
- ✅ Cobra CLI với auto-ingest flag

### Phase 2 - Optimization (SKIP)

- 🔄 Reduce LangChainGo dependency ở core interfaces
- 🔄 Implement internal abstraction types
- 🔄 Add more document formats (docx, xlsx, html)
- 🔄 Caching layer cho frequent queries
- 🔄 Advanced chunking strategies

### Phase 3 - Production Ready (SKIP)

- 📋 Multi-tenancy support
- 📋 Authentication & authorization
- 📋 Monitoring & observability
- 📋 Rate limiting & quotas
- 📋 Horizontal scaling support

---

## 10. Kết Luận

Hệ thống này tập trung phát huy sức mạnh của Golang với cấu trúc modular monolith gọn nhẹ, dễ phát triển, dễ bảo trì, áp dụng kiến trúc hexagonal cho từng module.

**Chiến lược hybrid approach:**

- Leverage LangChainGo cho các component phức tạp (embeddings, document processing, RAG chains)
- Giữ business logic và API layer độc lập hoàn toàn
- Sử dụng adapter pattern để dễ dàng swap implementations khi cần

Việc dùng LangChainGo giúp giảm thiểu công sức xây dựng pipeline RAG phức tạp, trong khi vẫn maintain control over architecture thông qua custom implementations ở các layer quan trọng. Cobra CLI đa command với flag `load-kb` giúp linh hoạt triển khai ingest tự động theo cấu hình, phù hợp triển khai thực tế nội bộ.

Sự kết hợp các thành phần giúp hệ thống vừa dễ vận hành, vừa sẵn sàng mở rộng và tối ưu hóa hiệu năng theo nhu cầu.