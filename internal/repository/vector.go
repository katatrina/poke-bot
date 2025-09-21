package repository

import (
	"context"
	"fmt"
	
	"github.com/katatrina/poke-bot/internal/config"
	"github.com/katatrina/poke-bot/internal/model"
	"github.com/qdrant/go-client/qdrant"
)

type VectorRepository struct {
	qdrantClient *qdrant.Client
	collection   string
}

func NewVectorRepository(cfg *config.Config, qdrantClient *qdrant.Client) (*VectorRepository, error) {
	repo := &VectorRepository{
		qdrantClient: qdrantClient,
		collection:   cfg.Qdrant.Collection,
	}
	
	// Ensure collection exists
	if err := repo.ensureCollection(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ensure collection: %w", err)
	}
	
	return repo, nil
}

func (repo *VectorRepository) ensureCollection(ctx context.Context) error {
	collections, err := repo.qdrantClient.ListCollections(ctx)
	if err != nil {
		return err
	}
	
	// Check if collection exists
	for _, col := range collections {
		if col == repo.collection {
			return nil // Collection exists
		}
	}
	
	// Create collection
	err = repo.qdrantClient.CreateCollection(ctx, &qdrant.CreateCollection{
		CollectionName: repo.collection,
		VectorsConfig: qdrant.NewVectorsConfig(&qdrant.VectorParams{
			Size:     768,                    // nomic-embed-text dimension
			Distance: qdrant.Distance_Cosine, // optimal for semantic search
		}),
	})
	if err != nil {
		return err
	}
	
	return nil
}

func (repo *VectorRepository) Upsert(ctx context.Context, documents []model.Document, embeddings [][]float32) error {
	if len(documents) != len(embeddings) {
		return fmt.Errorf("documents and embeddings count mismatch: %d vs %d", len(documents), len(embeddings))
	}
	
	var points []*qdrant.PointStruct
	for i, doc := range documents {
		// Convert metadata to Qdrant payload
		payload := make(map[string]any)
		payload["content"] = doc.Content
		for k, v := range doc.Metadata {
			payload[k] = v
		}
		
		point := qdrant.PointStruct{
			Id:      qdrant.NewIDUUID(doc.ID.String()),
			Vectors: qdrant.NewVectors(embeddings[i]...),
			Payload: qdrant.NewValueMap(payload),
		}
		
		points = append(points, &point)
	}
	
	_, err := repo.qdrantClient.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: repo.collection,
		Points:         points,
	})
	
	return err
}

func (repo *VectorRepository) Search(ctx context.Context, embedding []float32, limit int) ([]model.SearchResult, error) {
	searchResult, err := repo.qdrantClient.Query(ctx, &qdrant.QueryPoints{
		CollectionName: repo.collection,
		Query:          qdrant.NewQuery(embedding...),
		Limit:          qdrant.PtrOf(uint64(limit)),
		WithPayload:    qdrant.NewWithPayload(true),
		WithVectors:    qdrant.NewWithVectors(false),
	})
	if err != nil {
		return nil, err
	}
	
	var results []model.SearchResult
	for _, point := range searchResult {
		result := model.SearchResult{
			Score:    point.Score,
			Metadata: make(map[string]string),
		}
		
		// Extract content
		if contentValue, ok := point.Payload["content"]; ok {
			result.Content = contentValue.GetStringValue()
		}
		
		// Extract other metadata
		for k, v := range point.Payload {
			if k != "content" {
				result.Metadata[k] = v.GetStringValue()
			}
		}
		
		results = append(results, result)
	}
	
	return results, nil
}
