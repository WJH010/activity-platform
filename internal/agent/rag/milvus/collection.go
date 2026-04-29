package milvus

import (
	"context"
	"fmt"

	"github.com/milvus-io/milvus/client/v2/entity"
	"github.com/milvus-io/milvus/client/v2/index"
	"github.com/milvus-io/milvus/client/v2/milvusclient"
	"github.com/sirupsen/logrus"
)

const (
	ArticleCollectionName = "articles_semantic"
	EventCollectionName   = "events_semantic"

	VectorDimension = 1024 // bge-m3向量维度
)

// InitCollections 初始化所有Collection（如不存在则创建）
func InitCollections(ctx context.Context, cli *milvusclient.Client) error {
	if err := initArticleCollection(ctx, cli); err != nil {
		return fmt.Errorf("初始化文章Collection失败: %w", err)
	}
	if err := initEventCollection(ctx, cli); err != nil {
		return fmt.Errorf("初始化活动Collection失败: %w", err)
	}
	return nil
}

// needsRecreate 检查Collection是否需要重建（主键字段不是pk的旧Schema）
func needsRecreate(ctx context.Context, cli *milvusclient.Client, collectionName string) bool {
	coll, err := cli.DescribeCollection(ctx, milvusclient.NewDescribeCollectionOption(collectionName))
	if err != nil {
		return false
	}
	for _, field := range coll.Schema.Fields {
		if field.PrimaryKey {
			// 如果主键字段不叫pk，说明是旧Schema，需要重建
			if field.Name != "pk" {
				return true
			}
			return false
		}
	}
	return false
}

// dropAndRecreate 删除旧Collection并重建
func dropAndRecreate(ctx context.Context, cli *milvusclient.Client, collectionName string, createFn func(context.Context, *milvusclient.Client) error) error {
	logrus.Warnf("检测到Collection [%s] 使用旧Schema，需要重建", collectionName)

	// 删除旧Collection
	err := cli.DropCollection(ctx, milvusclient.NewDropCollectionOption(collectionName))
	if err != nil {
		return fmt.Errorf("删除旧Collection [%s] 失败: %w", collectionName, err)
	}
	logrus.Infof("已删除旧Collection [%s]", collectionName)

	// 用新Schema重建
	return createFn(ctx, cli)
}

// initArticleCollection 创建文章语义Collection
func initArticleCollection(ctx context.Context, cli *milvusclient.Client) error {
	// 检查是否已存在
	exists, err := cli.HasCollection(ctx, milvusclient.NewHasCollectionOption(ArticleCollectionName))
	if err != nil {
		return fmt.Errorf("检查Collection存在性失败: %w", err)
	}

	if exists {
		// 检查是否需要重建（旧Schema）
		if needsRecreate(ctx, cli, ArticleCollectionName) {
			if err := dropAndRecreate(ctx, cli, ArticleCollectionName, createArticleCollection); err != nil {
				return err
			}
			return nil
		}
		logrus.Infof("Collection [%s] 已存在，跳过创建", ArticleCollectionName)
		return nil
	}

	return createArticleCollection(ctx, cli)
}

// createArticleCollection 创建文章Collection（新Schema：pk主键）
func createArticleCollection(ctx context.Context, cli *milvusclient.Client) error {
	logrus.Infof("开始创建Collection [%s]...", ArticleCollectionName)

	// 定义Schema - 使用pk作为唯一主键，格式: {article_id}_{chunk_index}
	schema := entity.NewSchema().
		WithField(entity.NewField().
			WithName("pk").WithDataType(entity.FieldTypeVarChar).WithIsPrimaryKey(true).WithMaxLength(64).
			WithDescription("联合主键，格式: {article_id}_{chunk_index}")).
		WithField(entity.NewField().
			WithName("article_id").WithDataType(entity.FieldTypeInt64).
			WithDescription("文章ID")).
		WithField(entity.NewField().
			WithName("chunk_index").WithDataType(entity.FieldTypeInt32).
			WithDescription("分块序号")).
		WithField(entity.NewField().
			WithName("text_content").WithDataType(entity.FieldTypeVarChar).
			WithMaxLength(8192).
			WithEnableAnalyzer(true).
			WithDescription("原始文本（BM25输入）")).
		WithField(entity.NewField().
			WithName("dense_vector").WithDataType(entity.FieldTypeFloatVector).
			WithTypeParams("dim", "1024").
			WithDescription("bge-m3 dense向量")).
		WithField(entity.NewField().
			WithName("sparse_vector").WithDataType(entity.FieldTypeSparseVector).
			WithDescription("BM25稀疏向量（自动生成）")).
		WithField(entity.NewField().
			WithName("article_title").WithDataType(entity.FieldTypeVarChar).
			WithMaxLength(512).
			WithDescription("文章标题（冗余存储）")).
		WithField(entity.NewField().
			WithName("brief_content").WithDataType(entity.FieldTypeVarChar).
			WithMaxLength(1024).
			WithDescription("摘要（冗余存储）")).
		WithField(entity.NewField().
			WithName("field_type").WithDataType(entity.FieldTypeVarChar).
			WithMaxLength(64).
			WithDescription("领域类型")).
		WithField(entity.NewField().
			WithName("article_type").WithDataType(entity.FieldTypeVarChar).
			WithMaxLength(64).
			WithDescription("文章类型")).
		WithField(entity.NewField().
			WithName("is_selection").WithDataType(entity.FieldTypeInt32).
			WithDescription("是否精选")).
		WithField(entity.NewField().
			WithName("create_time").WithDataType(entity.FieldTypeInt64).
			WithDescription("创建时间戳")).
		// BM25函数：text_content → sparse_vector
		WithFunction(entity.NewFunction().
			WithName("bm25_func").
			WithType(entity.FunctionTypeBM25).
			WithInputFields("text_content").
			WithOutputFields("sparse_vector"),
		)

	// 创建Collection
	err := cli.CreateCollection(ctx, milvusclient.NewCreateCollectionOption(ArticleCollectionName, schema))
	if err != nil {
		return fmt.Errorf("创建Collection失败: %w", err)
	}

	// 创建索引
	if err := createArticleIndexes(ctx, cli); err != nil {
		return fmt.Errorf("创建索引失败: %w", err)
	}

	// 加载Collection到内存
	_, err = cli.LoadCollection(ctx, milvusclient.NewLoadCollectionOption(ArticleCollectionName))
	if err != nil {
		return fmt.Errorf("加载Collection失败: %w", err)
	}

	logrus.Infof("Collection [%s] 创建并加载完成", ArticleCollectionName)
	return nil
}

// createArticleIndexes 创建文章Collection的索引
func createArticleIndexes(ctx context.Context, cli *milvusclient.Client) error {
	// dense_vector: HNSW索引
	_, err := cli.CreateIndex(ctx, milvusclient.NewCreateIndexOption(ArticleCollectionName, "dense_vector",
		index.NewHNSWIndex(entity.COSINE, 16, 256),
	))
	if err != nil {
		return fmt.Errorf("创建dense_vector索引失败: %w", err)
	}

	// sparse_vector: 稀疏倒排索引
	_, err = cli.CreateIndex(ctx, milvusclient.NewCreateIndexOption(ArticleCollectionName, "sparse_vector",
		index.NewSparseInvertedIndex(entity.BM25, 0.2),
	))
	if err != nil {
		return fmt.Errorf("创建sparse_vector索引失败: %w", err)
	}

	// 标量字段: Inverted索引（加速过滤和查询）
	for _, field := range []string{"article_id", "field_type", "article_type"} {
		_, err = cli.CreateIndex(ctx, milvusclient.NewCreateIndexOption(ArticleCollectionName, field,
			index.NewInvertedIndex(),
		))
		if err != nil {
			logrus.Warnf("创建标量索引 [%s] 失败（不影响核心功能）: %v", field, err)
		}
	}

	return nil
}

// initEventCollection 创建活动语义Collection
func initEventCollection(ctx context.Context, cli *milvusclient.Client) error {
	// 检查是否已存在
	exists, err := cli.HasCollection(ctx, milvusclient.NewHasCollectionOption(EventCollectionName))
	if err != nil {
		return fmt.Errorf("检查Collection存在性失败: %w", err)
	}

	if exists {
		// 检查是否需要重建（旧Schema）
		if needsRecreate(ctx, cli, EventCollectionName) {
			if err := dropAndRecreate(ctx, cli, EventCollectionName, createEventCollection); err != nil {
				return err
			}
			return nil
		}
		logrus.Infof("Collection [%s] 已存在，跳过创建", EventCollectionName)
		return nil
	}

	return createEventCollection(ctx, cli)
}

// createEventCollection 创建活动Collection（新Schema：pk主键）
func createEventCollection(ctx context.Context, cli *milvusclient.Client) error {
	logrus.Infof("开始创建Collection [%s]...", EventCollectionName)

	// 定义Schema - 使用pk作为唯一主键，格式: {event_id}_{chunk_index}
	schema := entity.NewSchema().
		WithField(entity.NewField().
			WithName("pk").WithDataType(entity.FieldTypeVarChar).WithIsPrimaryKey(true).WithMaxLength(64).
			WithDescription("联合主键，格式: {event_id}_{chunk_index}")).
		WithField(entity.NewField().
			WithName("event_id").WithDataType(entity.FieldTypeInt64).
			WithDescription("活动ID")).
		WithField(entity.NewField().
			WithName("chunk_index").WithDataType(entity.FieldTypeInt32).
			WithDescription("分块序号")).
		WithField(entity.NewField().
			WithName("text_content").WithDataType(entity.FieldTypeVarChar).
			WithMaxLength(8192).
			WithEnableAnalyzer(true).
			WithDescription("原始文本（BM25输入）")).
		WithField(entity.NewField().
			WithName("dense_vector").WithDataType(entity.FieldTypeFloatVector).
			WithTypeParams("dim", "1024").
			WithDescription("bge-m3 dense向量")).
		WithField(entity.NewField().
			WithName("sparse_vector").WithDataType(entity.FieldTypeSparseVector).
			WithDescription("BM25稀疏向量（自动生成）")).
		WithField(entity.NewField().
			WithName("event_title").WithDataType(entity.FieldTypeVarChar).
			WithMaxLength(512).
			WithDescription("活动标题（冗余存储）")).
		WithField(entity.NewField().
			WithName("event_status").WithDataType(entity.FieldTypeVarChar).
			WithMaxLength(64).
			WithDescription("活动状态")).
		WithField(entity.NewField().
			WithName("event_address").WithDataType(entity.FieldTypeVarChar).
			WithMaxLength(512).
			WithDescription("活动地址")).
		WithField(entity.NewField().
			WithName("event_start_time").WithDataType(entity.FieldTypeInt64).
			WithDescription("活动开始时间戳")).
		WithField(entity.NewField().
			WithName("event_end_time").WithDataType(entity.FieldTypeInt64).
			WithDescription("活动结束时间戳")).
		// BM25函数
		WithFunction(entity.NewFunction().
			WithName("bm25_func").
			WithType(entity.FunctionTypeBM25).
			WithInputFields("text_content").
			WithOutputFields("sparse_vector"),
		)

	err := cli.CreateCollection(ctx, milvusclient.NewCreateCollectionOption(EventCollectionName, schema))
	if err != nil {
		return fmt.Errorf("创建Collection失败: %w", err)
	}

	if err := createEventIndexes(ctx, cli); err != nil {
		return fmt.Errorf("创建索引失败: %w", err)
	}

	_, err = cli.LoadCollection(ctx, milvusclient.NewLoadCollectionOption(EventCollectionName))
	if err != nil {
		return fmt.Errorf("加载Collection失败: %w", err)
	}

	logrus.Infof("Collection [%s] 创建并加载完成", EventCollectionName)
	return nil
}

// createEventIndexes 创建活动Collection的索引
func createEventIndexes(ctx context.Context, cli *milvusclient.Client) error {
	// dense_vector: HNSW
	_, err := cli.CreateIndex(ctx, milvusclient.NewCreateIndexOption(EventCollectionName, "dense_vector",
		index.NewHNSWIndex(entity.COSINE, 16, 256),
	))
	if err != nil {
		return fmt.Errorf("创建dense_vector索引失败: %w", err)
	}

	// sparse_vector
	_, err = cli.CreateIndex(ctx, milvusclient.NewCreateIndexOption(EventCollectionName, "sparse_vector",
		index.NewSparseInvertedIndex(entity.BM25, 0.2),
	))
	if err != nil {
		return fmt.Errorf("创建sparse_vector索引失败: %w", err)
	}

	// 标量字段
	for _, field := range []string{"event_id", "event_status"} {
		_, err = cli.CreateIndex(ctx, milvusclient.NewCreateIndexOption(EventCollectionName, field,
			index.NewInvertedIndex(),
		))
		if err != nil {
			logrus.Warnf("创建标量索引 [%s] 失败（不影响核心功能）: %v", field, err)
		}
	}

	return nil
}
