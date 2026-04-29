package embedding

import "context"

// EmbeddingService 向量嵌入服务接口
type EmbeddingService interface {
	// Embed 将单条文本转为向量
	Embed(ctx context.Context, text string) ([]float32, error)

	// EmbedBatch 批量将文本转为向量
	EmbedBatch(ctx context.Context, texts []string) ([][]float32, error)
}
