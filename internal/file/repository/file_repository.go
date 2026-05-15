package repository

import (
	"context"
	"errors"
	"event-platform/internal/file/model"
	"event-platform/internal/utils"
	"fmt"

	"gorm.io/gorm"
)

// FileRepository 文件存储库接口
type FileRepository interface {
	// CreateImageFile 创建图片文件记录
	CreateImageFile(ctx context.Context, file *model.Image) error
	// BatchUpdateImageBizID 批量更新图片的biz_id和biz_type
	BatchUpdateImageBizID(ctx context.Context, tx *gorm.DB, imageIDs []int, bizID int, bizType string) error
	// GetImageByID 根据图片ID查询图片
	GetImageByID(ctx context.Context, imageID int) (*model.Image, error)
	// DeleteImage 删除图片
	DeleteImage(ctx context.Context, imageID int) error
}

// FileRepositoryImpl 文件存储库实现
type FileRepositoryImpl struct {
	db *gorm.DB
}

// NewFileRepository 创建文件存储库实例
func NewFileRepository(db *gorm.DB) FileRepository {
	return &FileRepositoryImpl{db: db}
}

// CreateImageFile 创建图片文件记录
func (repo *FileRepositoryImpl) CreateImageFile(ctx context.Context, image *model.Image) error {
	err := repo.db.WithContext(ctx).Create(image).Error
	if err != nil {
		return utils.NewSystemError(fmt.Errorf("创建图片文件记录失败: %w", err))
	}
	return err
}

// BatchUpdateImageBizID 批量更新图片的biz_id和biz_type
func (repo *FileRepositoryImpl) BatchUpdateImageBizID(ctx context.Context, tx *gorm.DB, imageIDs []int, bizID int, bizType string) error {
	if len(imageIDs) == 0 {
		return nil
	}

	// 更新images表中指定ID的记录，设置biz_id和biz_type
	result := tx.WithContext(ctx).
		Table("images").
		Where("(biz_id IS NULL OR biz_id = 0) "). // 确保只更新未关联的图片
		Where("id IN (?)", imageIDs).
		Updates(map[string]interface{}{
			"biz_id":   bizID,
			"biz_type": bizType, // 标记为活动相关图片
		})

	if result.Error != nil {
		return utils.NewSystemError(fmt.Errorf("更新图片关联关系失败: %w", result.Error))
	}
	return nil
}

// GetImageByID 根据ID查询图片
func (repo *FileRepositoryImpl) GetImageByID(ctx context.Context, imageID int) (*model.Image, error) {
	var image model.Image
	err := repo.db.WithContext(ctx).First(&image, imageID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, utils.NewSystemError(fmt.Errorf("查询图片失败: %w", err))
	}
	return &image, nil
}

// DeleteImage 删除图片记录
func (repo *FileRepositoryImpl) DeleteImage(ctx context.Context, imageID int) error {
	result := repo.db.WithContext(ctx).Delete(&model.Image{}, imageID)
	if result.Error != nil {
		return utils.NewSystemError(fmt.Errorf("删除图片记录失败: %w", result.Error))
	}
	if result.RowsAffected == 0 {
		return utils.NewBusinessError(utils.ErrCodeResourceNotFound, "图片不存在或已被删除")
	}
	return nil
}
