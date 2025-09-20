package model

import (
	"github.com/google/uuid"
)

type Document struct {
	ID       uuid.UUID         `json:"id"`
	Content  string            `json:"content"`
	Metadata map[string]string `json:"metadata"`
}

type SearchResult struct {
	Content  string            `json:"content"`
	Score    float32           `json:"score"`
	Metadata map[string]string `json:"metadata"`
}
