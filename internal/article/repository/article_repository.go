package repository

import (
	"context"
	"errors"
	"event-platform/internal/article/dto"
	"event-platform/internal/article/model"
	"event-platform/internal/utils"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// ArticleRepository 数据访问接口，定义数据访问的方法集
type ArticleRepository interface {
	// List 分页查询
	List(ctx context.Context, page, pageSize int, articleTitle string, articleType string, releaseTime string, fieldType string, isSelection int, queryScope string) ([]dto.ArticleListResponse, int64, error)
	// GetArticleContent 内容查询
	GetArticleContent(ctx context.Context, articleID int) (*dto.ArticleContentDTO, error)
	// GetArticleByTitle 根据标题查询文章
	GetArticleByTitle(ctx context.Context, title string) (*model.Article, error)
	// CreateArticle 创建文章
	CreateArticle(ctx context.Context, tx *gorm.DB, article *model.Article) error
	// UpdateArticle 更新文章
	UpdateArticle(ctx context.Context, tx *gorm.DB, articleID int, updateFields map[string]interface{}) error
	// ListArticleImage 获取关联图片列表
	ListArticleImage(ctx context.Context, bizID int) []dto.Image
	// ListUpdatedSince 查询指定时间后有更新的文章（含已删除，含正文，用于增量同步）
	ListUpdatedSince(ctx context.Context, since time.Time, pageSize int, page int) ([]dto.ArticleUpdatedSinceResponse, int64, error)
}

// ArticleRepositoryImpl 实现接口的具体结构体
type ArticleRepositoryImpl struct {
	db *gorm.DB
}

// NewArticleRepository 创建数据访问实例
func NewArticleRepository(db *gorm.DB) ArticleRepository {
	return &ArticleRepositoryImpl{db: db}
}

// List 分页查询数据
func (repo *ArticleRepositoryImpl) List(ctx context.Context, page, pageSize int, articleTitle string, articleType string, releaseTime string, fieldType string, isSelection int, queryScope string) ([]dto.ArticleListResponse, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}

	offset := (page - 1) * pageSize
	var articles []dto.ArticleListResponse
	query := repo.db.WithContext(ctx)

	// 构建基础查询
	query = query.Table("articles a").
		Select("a.article_id, a.article_title, a.article_type AS article_type_code, a.release_time, a.brief_content, a.is_selection, f.field_name, a.cover_image_url,a.article_source,at.type_name AS article_type").
		Joins("LEFT JOIN field_types f ON a.field_type = f.field_code").
		Joins("LEFT JOIN article_types at ON a.article_type = at.type_code")

	if queryScope != "" {
		// 如果传入了查询范围，则添加查询条件
		// 如果传入了查询范围为DELETED，则查询已删除的文章
		if queryScope == utils.QueryScopeDeleted {
			query = query.Where("a.is_deleted = ?", utils.DeletedFlagYes) // 查询已删除的文章
		}
		if queryScope == utils.QueryScopeAll {
			// 如果传入了查询范围为ALL，则查询所有文章（包括已删除和未删除的）
		}
	} else {
		// 默认查询未删除的
		query = query.Where("a.is_deleted = ?", utils.DeletedFlagNo)
	}

	// 添加条件查询
	if releaseTime != "" {
		query = query.Where("a.release_time >= ?", releaseTime)
	}
	if articleTitle != "" {
		query = query.Where("a.article_title LIKE ?", "%"+articleTitle+"%")
	}
	if fieldType != "" {
		query = query.Where("a.field_type = ?", fieldType)
	}
	if articleType != "" {
		query = query.Where("a.article_type = ?", articleType)
	}
	if isSelection != 0 {
		query = query.Where("a.is_selection = ?", isSelection)
	}

	// 按发布时间降序排列
	query = query.Order("a.release_time DESC")

	// 计算总数
	var total int64
	countQuery := query.Session(&gorm.Session{})
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, utils.NewSystemError(fmt.Errorf("计算总数时数据库查询失败: %v", err))
	}

	// 查询数据
	if err := query.Offset(offset).Limit(pageSize).Find(&articles).Error; err != nil {
		return nil, 0, utils.NewSystemError(fmt.Errorf("数据库查询失败: %v", err))
	}

	return articles, total, nil
}

// GetArticleContent 内容查询
func (repo *ArticleRepositoryImpl) GetArticleContent(ctx context.Context, articleID int) (*dto.ArticleContentDTO, error) {
	var article dto.ArticleContentDTO

	query := repo.db.WithContext(ctx).Table("articles a").
		Select(`a.article_id, a.article_title, f.field_name, a.release_time, a.article_content, 
				a.article_type AS article_type_code, at.type_name AS article_type, a.article_source, 
				a.cover_image_url, a.brief_content, a.is_selection, a.field_type`).
		Joins("LEFT JOIN field_types f ON a.field_type = f.field_code").
		Joins("LEFT JOIN article_types at ON a.article_type = at.type_code").
		Where("a.article_id = ?", articleID)

	result := query.First(&article, articleID)
	err := result.Error

	// 查询文章内容
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, utils.NewBusinessError(utils.ErrCodeResourceNotFound, "文章不存在或已被删除，请刷新页面后重试")
		}
		return nil, utils.NewSystemError(fmt.Errorf("数据库查询失败: %v", err))
	}

	return &article, nil
}

// CreateArticle 创建文章
func (repo *ArticleRepositoryImpl) CreateArticle(ctx context.Context, tx *gorm.DB, article *model.Article) error {
	// 插入文章数据
	if err := tx.WithContext(ctx).Create(article).Error; err != nil {
		return utils.NewSystemError(fmt.Errorf("创建文章失败: %w", err))
	}

	return nil
}

// GetArticleByTitle 根据标题查询文章
func (repo *ArticleRepositoryImpl) GetArticleByTitle(ctx context.Context, title string) (*model.Article, error) {
	var article model.Article

	// 查询文章
	if err := repo.db.WithContext(ctx).Where("article_title = ? AND is_deleted = ?", title, utils.DeletedFlagNo).First(&article).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, utils.NewSystemError(fmt.Errorf("数据库查询失败: %v", err))
	}

	return &article, nil
}

// UpdateArticle 更新文章字段
func (repo *ArticleRepositoryImpl) UpdateArticle(ctx context.Context, tx *gorm.DB, articleID int, updateFields map[string]interface{}) error {
	// 执行更新（仅更新未删除的文章）
	result := tx.WithContext(ctx).
		Model(&model.Article{}).
		Where("article_id = ? AND is_deleted = ?", articleID, utils.DeletedFlagNo).
		Updates(updateFields)

	if result.Error != nil {
		return utils.NewSystemError(fmt.Errorf("更新文章失败: %w", result.Error))
	}
	if result.RowsAffected == 0 {
		return utils.NewBusinessError(utils.ErrCodeResourceNotFound, "文章不存在或已被删除，请刷新页面后重试")
	}

	return nil
}

// ListArticleImage 获取关联图片列表
func (repo *ArticleRepositoryImpl) ListArticleImage(ctx context.Context, bizID int) []dto.Image {
	var images []dto.Image

	err := repo.db.WithContext(ctx).
		Table("images").
		Where("biz_type = ? AND biz_id = ?", utils.TypeArticle, bizID).
		Find(&images).Error

	if err != nil {
		logrus.Errorf("获取文章关联图片失败: %v", err) // 只记录异常，不影响活动信息的返回
		return nil
	}

	return images
}

// ListUpdatedSince 查询指定时间后有更新的文章
func (repo *ArticleRepositoryImpl) ListUpdatedSince(ctx context.Context, since time.Time, pageSize int, page int) ([]dto.ArticleUpdatedSinceResponse, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 50
	}

	offset := (page - 1) * pageSize
	var articles []dto.ArticleUpdatedSinceResponse

	query := repo.db.WithContext(ctx).Table("articles a").
		Select("a.article_id, a.article_title, a.article_content, a.article_type AS article_type_code, " +
			"a.field_type, a.brief_content, a.is_selection, a.release_time, a.is_deleted, a.update_time")

	// 核心条件：update_time >= since（不过滤is_deleted，已删除的也要查出来用于同步删除）
	query = query.Where("a.update_time >= ?", since)

	// 按更新时间升序排列，确保顺序处理
	query = query.Order("a.update_time ASC")

	// 计算总数
	var total int64
	countQuery := query.Session(&gorm.Session{})
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, utils.NewSystemError(fmt.Errorf("计算总数时数据库查询失败: %v", err))
	}

	// 分页查询
	if err := query.Offset(offset).Limit(pageSize).Find(&articles).Error; err != nil {
		return nil, 0, utils.NewSystemError(fmt.Errorf("数据库查询失败: %v", err))
	}

	return articles, total, nil
}
