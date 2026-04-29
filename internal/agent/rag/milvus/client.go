package milvus

import (
	"context"
	"fmt"

	"activity-platform/internal/config"

	"github.com/milvus-io/milvus/client/v2/milvusclient"
	"github.com/sirupsen/logrus"
)

// NewClient 创建Milvus客户端
func NewClient(cfg config.MilvusConfig) (*milvusclient.Client, error) {
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)

	cli, err := milvusclient.New(context.Background(), &milvusclient.ClientConfig{
		Address: addr,
		DBName:  cfg.DBName,
	})
	if err != nil {
		return nil, fmt.Errorf("连接Milvus失败 [%s]: %w", addr, err)
	}

	// 验证连接：获取服务端版本
	version, err := cli.GetServerVersion(context.Background(), milvusclient.NewGetServerVersionOption())
	if err != nil {
		return nil, fmt.Errorf("Milvus连接验证失败: %w", err)
	}

	logrus.Infof("Milvus连接成功 [%s], 版本: %s", addr, version)
	return cli, nil
}
