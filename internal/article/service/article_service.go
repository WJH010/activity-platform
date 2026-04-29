package service

import (
	"activity-platform/internal/article/dto"
	"activity-platform/internal/article/model"
	"activity-platform/internal/article/repository"
	db "activity-platform/internal/database"
	filerepo "activity-platform/internal/file/repository"
	"activity-platform/internal/utils"
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
)

// ArticleService 服务接口，定义方法，接收 context.Context 和数据模型。
type ArticleService interface {
	// ListArticle 分页查询文章列表
	ListArticle(ctx context.Context, page, pageSize int, articleTitle string, articleType string, releaseTime string, fieldType string, isSelection int, queryScope string) ([]dto.ArticleListResponse, int64, error)
	// GetArticleContent 获取文章内容
	GetArticleContent(ctx context.Context, articleID int) (*dto.ArticleContentResponse, error)
	// CreateArticle 创建文章
	CreateArticle(ctx context.Context, article *model.Article, imageIDList []int) error
	// UpdateArticle 更新文章
	UpdateArticle(ctx context.Context, articleID int, req dto.UpdateArticleRequest, userID int) error
	// DeleteArticle 删除文章
	DeleteArticle(ctx context.Context, articleID int, userID int) error
}

// ArticleServiceImpl 实现接口的具体结构体，持有数据访问层接口 Repository 的实例
type ArticleServiceImpl struct {
	articleRepo repository.ArticleRepository
	fileRepo    filerepo.FileRepository
}

// NewArticleService 创建服务实例
func NewArticleService(articleRepo repository.ArticleRepository, fileRepo filerepo.FileRepository) ArticleService {
	return &ArticleServiceImpl{articleRepo: articleRepo, fileRepo: fileRepo}
}

// ListArticle 分页查询数据
func (svc *ArticleServiceImpl) ListArticle(ctx context.Context, page, pageSize int, articleTitle string, articleType string, releaseTime string, fieldType string, isSelection int, queryScope string) ([]dto.ArticleListResponse, int64, error) {
	return svc.articleRepo.List(ctx, page, pageSize, articleTitle, articleType, releaseTime, fieldType, isSelection, queryScope)
}

// GetArticleContent 获取文章内容
func (svc *ArticleServiceImpl) GetArticleContent(ctx context.Context, articleID int) (*dto.ArticleContentResponse, error) {
	article, err := svc.articleRepo.GetArticleContent(ctx, articleID)
	if err != nil {
		return nil, err
	}

	// 获取关联图片列表
	images := svc.articleRepo.ListArticleImage(ctx, articleID)

	// 拼接文章内容和图片列表
	res := dto.ArticleContentResponse{
		ArticleID:       article.ArticleID,
		ArticleTitle:    article.ArticleTitle,
		BriefContent:    article.BriefContent,
		FieldType:       article.FieldType,
		FieldName:       article.FieldName,
		ReleaseTime:     article.ReleaseTime,
		ArticleContent:  article.ArticleContent,
		ArticleTypeCode: article.ArticleTypeCode,
		ArticleType:     article.ArticleType,
		ArticleSource:   article.ArticleSource,
		IsSelection:     article.IsSelection,
		CoverImageURL:   article.CoverImageURL,
	}
	res.Images = make([]dto.Image, 0, len(images)) // 预分配空间，提高性能
	for _, img := range images {
		res.Images = append(res.Images, dto.Image{
			ImageID: img.ImageID,
			URL:     img.URL,
		})
	}
	return &res, err
}

// CreateArticle 创建文章
func (svc *ArticleServiceImpl) CreateArticle(ctx context.Context, article *model.Article, imageIDList []int) error {
	// 检查是否存在重复标题的文章
	existingArticle, err := svc.articleRepo.GetArticleByTitle(ctx, article.ArticleTitle)
	if err != nil {
		return err
	}
	if existingArticle != nil {
		return utils.NewBusinessError(utils.ErrCodeResourceExists, "已存在同名文章，请修改标题后重试")
	}

	// 开启事务
	tx := db.GetDB().Begin()
	if tx.Error != nil {
		return utils.NewSystemError(fmt.Errorf("开启事务失败: %w", tx.Error))
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			logrus.Panic("事务回滚，发生异常: ", r)
		}
	}()

	// 创建文章
	if err := svc.articleRepo.CreateArticle(ctx, tx, article); err != nil {
		tx.Rollback()
		return err
	}

	// 如果有图片，更新images表的biz_id和biz_type
	if len(imageIDList) > 0 {
		if err := svc.fileRepo.BatchUpdateImageBizID(ctx, tx, imageIDList, article.ArticleID, utils.TypeArticle); err != nil {
			tx.Rollback()
			return err
		}
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return utils.NewSystemError(fmt.Errorf("提交事务失败: %w", err))
	}
	return nil
}

// UpdateArticle 更新文章
func (svc *ArticleServiceImpl) UpdateArticle(ctx context.Context, articleID int, req dto.UpdateArticleRequest, userID int) error {
	// 检查文章是否存在
	article, err := svc.articleRepo.GetArticleContent(ctx, articleID)
	if err != nil {
		return err
	}
	if article == nil {
		return utils.NewBusinessError(utils.ErrCodeResourceNotFound, "文章不存在或已被删除")
	}

	// 检查标题是否重复（仅当标题被修改时）
	if req.ArticleTitle != nil && *req.ArticleTitle != article.ArticleTitle {
		existing, err := svc.articleRepo.GetArticleByTitle(ctx, *req.ArticleTitle)
		if err != nil {
			return err
		}
		if existing != nil {
			return utils.NewBusinessError(utils.ErrCodeResourceExists, "已存在同名文章，请修改标题后重试")
		}
	}

	// 构建更新字段
	updateFields, err := makeArticleUpdateFields(req)
	if err != nil {
		return err
	}

	// 处理图片ID列表
	var imageIDList []int
	if req.ImageIDList != nil {
		imageIDList = *req.ImageIDList
	}

	if len(updateFields) == 0 && len(imageIDList) == 0 {
		return nil // 无更新内容
	}

	// 设置更新人
	updateFields["update_user"] = userID

	// 开启事务
	tx := db.GetDB().Begin()
	if tx.Error != nil {
		return utils.NewSystemError(fmt.Errorf("开启事务失败: %w", tx.Error))
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			logrus.Panic("事务回滚，发生异常: ", r)
		}
	}()

	// 更新文章基本信息
	if err := svc.articleRepo.UpdateArticle(ctx, tx, articleID, updateFields); err != nil {
		tx.Rollback()
		return err
	}

	// 更新图片关联（如果有）
	if len(imageIDList) > 0 {
		if err := svc.fileRepo.BatchUpdateImageBizID(ctx, tx, imageIDList, articleID, utils.TypeArticle); err != nil {
			tx.Rollback()
			return err
		}
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return utils.NewSystemError(fmt.Errorf("提交事务失败: %w", err))
	}
	return nil
}

// 辅助函数：构建更新字段映射
func makeArticleUpdateFields(req dto.UpdateArticleRequest) (map[string]interface{}, error) {
	updateFields := make(map[string]interface{})

	// 处理其他字段
	if req.ArticleTitle != nil {
		updateFields["article_title"] = *req.ArticleTitle
	}
	if req.ArticleType != nil {
		updateFields["article_type"] = *req.ArticleType
	}
	if req.BriefContent != nil {
		updateFields["brief_content"] = *req.BriefContent
	}
	if req.ArticleContent != nil {
		updateFields["article_content"] = *req.ArticleContent
	}
	if req.IsSelection != nil {
		updateFields["is_selection"] = *req.IsSelection
	}
	if req.FieldType != nil {
		updateFields["field_type"] = *req.FieldType
	}
	if req.CoverImageURL != nil {
		updateFields["cover_image_url"] = *req.CoverImageURL
	}
	if req.ArticleSource != nil {
		updateFields["article_source"] = *req.ArticleSource
	}

	return updateFields, nil
}

// DeleteArticle 软删除文章（更新is_deleted标志）
func (svc *ArticleServiceImpl) DeleteArticle(ctx context.Context, articleID int, userID int) error {
	// 检查文章是否存在
	article, err := svc.articleRepo.GetArticleContent(ctx, articleID)
	if err != nil {
		return err
	}
	if article == nil {
		return utils.NewBusinessError(utils.ErrCodeResourceNotFound, "文章不存在或已被删除，请刷新后重试")
	}

	// 开启事务
	tx := db.GetDB().Begin()
	if tx.Error != nil {
		return utils.NewSystemError(fmt.Errorf("开启事务失败: %w", tx.Error))
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			logrus.Panic("事务回滚，发生异常: ", r)
		}
	}()

	// 软删除（更新is_deleted为Y，记录更新人）
	updateFields := map[string]interface{}{
		"is_deleted":  utils.DeletedFlagYes,
		"update_user": userID,
	}
	if err := svc.articleRepo.UpdateArticle(ctx, tx, articleID, updateFields); err != nil {
		tx.Rollback()
		return err
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return utils.NewSystemError(fmt.Errorf("提交事务失败: %w", err))
	}
	return nil
}
