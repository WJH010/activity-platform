package embedding

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"event-platform/internal/config"

	"github.com/sirupsen/logrus"
)

// bgeM3Service 基于OpenAI兼容API的bge-m3实现
// 支持 Infinity、TEI、Ollama 等提供的OpenAI兼容Embedding端点
type bgeM3Service struct {
	apiURL    string
	modelName string
	batchSize int
	client    *http.Client
}

// NewBGEML3Service 创建bge-m3 Embedding服务
func NewBGEML3Service(cfg config.EmbeddingConfig) EmbeddingService {
	modelName := "bge-m3"
	if cfg.Provider == "bge_m3_api" && cfg.ApiURL == "" {
		logrus.Warn("Embedding API URL为空，语义检索将不可用")
	}

	batchSize := cfg.BatchSize
	if batchSize <= 0 {
		batchSize = 32
	}
	return &bgeM3Service{
		apiURL:    cfg.ApiURL,
		modelName: modelName,
		batchSize: batchSize,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// embeddingRequest OpenAI兼容Embedding请求
type embeddingRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

// embeddingResponse OpenAI兼容Embedding响应
type embeddingResponse struct {
	Data  []embeddingData `json:"data"`
	Usage struct {
		TotalTokens int `json:"total_tokens"`
	} `json:"usage"`
}

type embeddingData struct {
	Index     int       `json:"index"`
	Embedding []float32 `json:"embedding"`
}

// embeddingErrorResponse 错误响应
type embeddingErrorResponse struct {
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}

// Embed 将单条文本转为向量
func (s *bgeM3Service) Embed(ctx context.Context, text string) ([]float32, error) {
	vectors, err := s.EmbedBatch(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	if len(vectors) == 0 {
		return nil, fmt.Errorf("嵌入结果为空")
	}
	return vectors[0], nil
}

// EmbedBatch 批量将文本转为向量
func (s *bgeM3Service) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	if s.apiURL == "" {
		return nil, fmt.Errorf("Embedding API URL未配置")
	}

	// 分批处理
	var allVectors [][]float32
	for i := 0; i < len(texts); i += s.batchSize {
		end := i + s.batchSize
		if end > len(texts) {
			end = len(texts)
		}
		batch := texts[i:end]

		vectors, err := s.callAPI(ctx, batch)
		if err != nil {
			return nil, fmt.Errorf("批次 %d-%d 嵌入失败: %w", i, end, err)
		}
		allVectors = append(allVectors, vectors...)
	}

	return allVectors, nil
}

// callAPI 调用Embedding API
func (s *bgeM3Service) callAPI(ctx context.Context, texts []string) ([][]float32, error) {
	reqBody := embeddingRequest{
		Model: s.modelName,
		Input: texts,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", s.apiURL, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("调用Embedding API失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp embeddingErrorResponse
		if json.Unmarshal(body, &errResp) == nil && errResp.Error.Message != "" {
			return nil, fmt.Errorf("Embedding API错误 [%d]: %s", resp.StatusCode, errResp.Error.Message)
		}
		return nil, fmt.Errorf("Embedding API错误 [%d]: %s", resp.StatusCode, string(body))
	}

	var embResp embeddingResponse
	if err := json.Unmarshal(body, &embResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	// 按index排序确保顺序正确
	vectors := make([][]float32, len(embResp.Data))
	for _, d := range embResp.Data {
		if d.Index >= len(vectors) {
			continue
		}
		vectors[d.Index] = d.Embedding
	}

	logrus.Debugf("Embedding API调用成功, 输入%d条, 输出%d条向量, tokens=%d",
		len(texts), len(vectors), embResp.Usage.TotalTokens)

	return vectors, nil
}
