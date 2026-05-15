package repository

import (
	"context"
	"event-platform/internal/utils"
	"fmt"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// MinIORepository MinIO存储接口
type MinIORepository interface {
	UploadFile(ctx context.Context, objectName, filePath string) (string, error)
	DeleteFile(ctx context.Context, objectName string) error
	//GetFileURL(ctx context.Context, objectName string) (string, error)
}

// MinIORepositoryImpl MinIO存储实现
type MinIORepositoryImpl struct {
	client     *minio.Client
	bucketName string
}

// NewMinIORepository 创建MinIO存储实例
func NewMinIORepository(endpoint, accessKeyID, secretAccessKey string, useSSL bool, bucketName string) (MinIORepository, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, utils.NewSystemError(fmt.Errorf("创建MinIO客户端失败: %w", err))
	}

	// 检查存储桶是否存在，不存在则创建
	ctx := context.Background()
	exists, err := client.BucketExists(ctx, bucketName)
	if err != nil {
		return nil, utils.NewSystemError(fmt.Errorf("检查MinIO存储桶是否存在失败: %w", err))
	}
	if !exists {
		err = client.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{})
		if err != nil {
			return nil, utils.NewSystemError(fmt.Errorf("创建MinIO存储桶失败: %w", err))
		}
	}

	return &MinIORepositoryImpl{
		client:     client,
		bucketName: bucketName,
	}, nil
}

// UploadFile 上传文件到MinIO
func (repo *MinIORepositoryImpl) UploadFile(ctx context.Context, objectName, filePath string) (string, error) {
	info, err := repo.client.FPutObject(ctx, repo.bucketName, objectName, filePath, minio.PutObjectOptions{})
	if err != nil {
		return "", utils.NewSystemError(fmt.Errorf("上传文件到MinIO失败: %w", err))
	}

	// 生成文件的访问URL
	url := fmt.Sprintf("%s/%s/%s", repo.client.EndpointURL().String(), repo.bucketName, info.Key)
	return url, nil
}

// DeleteFile 从MinIO删除文件
func (repo *MinIORepositoryImpl) DeleteFile(ctx context.Context, objectName string) error {
	err := repo.client.RemoveObject(ctx, repo.bucketName, objectName, minio.RemoveObjectOptions{})
	if err != nil {
		return utils.NewSystemError(fmt.Errorf("删除MinIO文件失败: %w", err))
	}
	return err
}

// GetFileURL 获取文件预签名URL
//func (repo *MinIORepositoryImpl) GetFileURL(ctx context.Context, objectName string) (string, error) {
//	// 生成预签名URL，有效期1小时
//	url, err := repo.client.PresignedGetObject(ctx, repo.bucketName, objectName, time.Hour, nil)
//	if err != nil {
//		return "", utils.NewSystemError(fmt.Errorf("生成文件URL失败: %w", err))
//	}
//	return url.String(), nil
//}
