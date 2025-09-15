package ingest

import (
	"time"
)

// Document represents a document to ingest
type Document struct {
	ID       string                 `json:"id"`
	Filepath string                 `json:"filepath"`
	Name     string                 `json:"name"`
	Content  string                 `json:"content"`
	Metadata map[string]interface{} `json:"metadata"`
	CreateAt time.Time              `json:"create_at"`
}

// Chunk represents a small part of the document after splitting
type Chunk struct {
	ID         string                 `json:"id"`
	DocumentID string                 `json:"document_id"`
	Content    string                 `json:"content"`
	Metadata   map[string]interface{} `json:"metadata"`
	Embedding  []float32              `json:"embedding,omitempty"`
	Index      int                    `json:"index"` // chunk order in document
}
