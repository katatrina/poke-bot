.PHONY: run

qdrant:
	docker run -d --name qdrant-container -p 6333:6333 qdrant/qdrant:v1.2.0

run:
	go run ./cmd/server
