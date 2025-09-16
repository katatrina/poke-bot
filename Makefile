.PHONY: run

qdrant:
	docker run -d --name qdrant-container -p 6333:6333 -p 6334:6334 qdrant/qdrant

ollama:
	docker run -d --name ollama-container -p 11434:11434 ollama/ollama
	curl http://localhost:11434/api/pull -d '{"name":"qwen2.5-coder:3b"}'
	curl http://localhost:11434/api/pull -d '{"name":"nomic-embed-text"}'

run:
	go run ./cmd
