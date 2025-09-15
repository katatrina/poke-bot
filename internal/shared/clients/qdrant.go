package clients

import (
	"context"
	"fmt"

	"github.com/qdrant/go-client/qdrant"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type QdrantClient struct {
	client     qdrant.QdrantClient
	conn       *grpc.ClientConn
	collection string
}

func NewQdrantClient(url, collection string) (*QdrantClient, error) {
	conn, err := grpc.NewClient(url, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Qdrant: %w", err)
	}

	client := qdrant.NewQdrantClient(conn)

	qc := &QdrantClient{
		client:     client,
		conn:       conn,
		collection: collection,
	}

	return qc, nil
}

func (q *QdrantClient) Close() error {
	return q.conn.Close()
}

func (q *QdrantClient) CreateCollection(ctx context.Context, vectorSize uint64) error {
	_, err := q.client.CreateCollection(ctx, &qdrant.CreateCollection{
		CollectionName: q.collection,
		VectorsConfig: &qdrant.VectorsConfig{
			Config: &qdrant.VectorsConfig_Params{
				Params: &qdrant.VectorParams{
					Size:     vectorSize,
					Distance: qdrant.Distance_Cosine,
				},
			},
		},
	})
	return err
}

func (q *QdrantClient) Upsert(ctx context.Context, points []*qdrant.PointStruct) error {
	_, err := q.client.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: q.collection,
		Points:         points,
	})
	return err
}

func (q *QdrantClient) Search(ctx context.Context, vector []float32, limit uint64, threshold float32) ([]*qdrant.ScoredPoint, error) {
	response, err := q.client.Search(ctx, &qdrant.SearchPoints{
		CollectionName: q.collection,
		Vector:         vector,
		Limit:          limit,
		WithPayload:    &qdrant.WithPayloadSelector{SelectorOptions: &qdrant.WithPayloadSelector_Enable{Enable: true}},
		ScoreThreshold: &threshold,
	})
	if err != nil {
		return nil, err
	}
	return response.Result, nil
}

func (q *QdrantClient) CollectionExists(ctx context.Context) (bool, error) {
	response, err := q.client.CollectionInfo(ctx, &qdrant.CollectionInfoRequest{
		CollectionName: q.collection,
	})
	if err != nil {
		return false, nil
	}
	return response != nil, nil
}