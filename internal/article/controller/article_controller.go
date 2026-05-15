package controller

import (
	"event-platform/internal/article/dto"
	"event-platform/internal/article/model"
	"event-platform/internal/article/service"
	"event-platform/internal/utils"
	"time"

	"github.com/gin-gonic/gin"
)

// ArticleController 控制器
type ArticleController struct {
	articleService service.ArticleService
}

// NewArticleController 创建控制器实例
func NewArticleController(articleService service.ArticleService) *ArticleController {
	return &ArticleController{articleService: articleService}
}

// ListArticle 分页查询
func (ctr *ArticleController) ListArticle(ctx *gin.Context) {
	// 初始化参数结构体并绑定查询参数
	var req dto.ArticleListRequest
	if !utils.BindQuery(ctx, &req) {
		return
	}

	// page 默认1
	page := req.Page
	if page == 0 {
		page = 1
	}

	// pageSize 默认10
	pageSize := req.PageSize
	if pageSize == 0 {
		pageSize = 10
	}

	// 调用服务层
	results, total, err := ctr.articleService.ListArticle(ctx, page, pageSize, req.ArticleTitle, req.ArticleType, req.ReleaseTime, req.FieldType, req.IsSelection, req.QueryScope)
	// 处理异常
	if err != nil {
		utils.HandlerFunc(ctx, err)
		return
	}

	// 返回分页结果
	utils.SuccessPage(ctx, total, page, pageSize, results)
}

// GetArticleContent 获取文章内容
func (ctr *ArticleController) GetArticleContent(ctx *gin.Context) {
	// 初始化参数结构体并绑定查询参数
	var req dto.ArticleContentRequest
	if !utils.BindUrl(ctx, &req) {
		return
	}

	// 调用服务层
	result, err := ctr.articleService.GetArticleContent(ctx, req.ArticleID)
	// 处理异常
	if err != nil {
		utils.HandlerFunc(ctx, err)
		return
	}

	// 返回成功响应
	utils.Success(ctx, "success", result)

}

// CreateArticle 创建文章
func (ctr *ArticleController) CreateArticle(ctx *gin.Context) {
	// 初始化参数结构体并绑定请求体
	var req dto.CreateArticleRequest
	if !utils.BindJSON(ctx, &req) {
		return
	}

	// 获取userID
	userID, err := utils.GetUserID(ctx)
	if err != nil {
		utils.HandlerFunc(ctx, err)
		return
	}

	// 构造文章对象
	article := &model.Article{
		ArticleTitle:   req.ArticleTitle,
		ArticleType:    req.ArticleType,
		BriefContent:   req.BriefContent,
		ArticleContent: req.ArticleContent,
		IsSelection:    req.IsSelection,
		FieldType:      req.FieldType,
		CoverImageURL:  req.CoverImageURL,
		ArticleSource:  req.ArticleSource,
		ReleaseTime:    time.Now(),
		CreateUser:     userID,
		UpdateUser:     userID,
	}

	// 调用服务层
	if err := ctr.articleService.CreateArticle(ctx, article, req.ImageIDList); err != nil {
		utils.HandlerFunc(ctx, err)
		return
	}

	result := gin.H{
		"article_id": article.ArticleID,
	}

	// 返回成功响应
	utils.Success(ctx, "success", result)
}

// UpdateArticle 处理更新文章的请求
func (ctr *ArticleController) UpdateArticle(ctx *gin.Context) {
	// 从URL获取文章ID
	var urlReq dto.ArticleContentRequest
	if !utils.BindUrl(ctx, &urlReq) {
		return
	}

	// 从请求体获取更新参数
	var req dto.UpdateArticleRequest
	if !utils.BindJSON(ctx, &req) {
		return
	}

	// 获取当前用户ID
	userID, err := utils.GetUserID(ctx)
	if err != nil {
		utils.HandlerFunc(ctx, err)
		return
	}

	// 调用服务层更新文章
	err = ctr.articleService.UpdateArticle(ctx, urlReq.ArticleID, req, userID)
	if err != nil {
		utils.HandlerFunc(ctx, err)
		return
	}

	utils.Success(ctx, "success", nil)
}

// DeleteArticle 处理删除文章的请求
func (ctr *ArticleController) DeleteArticle(ctx *gin.Context) {
	// 从URL获取文章ID
	var req dto.ArticleContentRequest
	if !utils.BindUrl(ctx, &req) {
		return
	}

	// 获取当前用户ID
	userID, err := utils.GetUserID(ctx)
	if err != nil {
		utils.HandlerFunc(ctx, err)
		return
	}

	// 调用服务层删除文章
	err = ctr.articleService.DeleteArticle(ctx, req.ArticleID, userID)
	if err != nil {
		utils.HandlerFunc(ctx, err)
		return
	}

	utils.Success(ctx, "success", nil)
}
