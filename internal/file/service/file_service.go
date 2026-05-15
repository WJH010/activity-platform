package service

import (
	"context"
	"event-platform/internal/file/dto"
	"event-platform/internal/utils"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"event-platform/internal/file/model"
	"event-platform/internal/file/repository"
)

// FileService 文件服务接口
type FileService interface {
	UploadFile(ctx context.Context, fileHeader *FileHeader, bizType string, bizID int, userID int) (dto.FileUploadResponse, error)
	// DeleteImage 删除图片
	DeleteImage(ctx context.Context, imageID int, userID int) error
}

// FileHeader 文件头信息
type FileHeader struct {
	OriginalFileName string
	ContentType      string
	Size             int64
	TemporaryFile    string
}

// FileServiceImpl 文件服务实现
type FileServiceImpl struct {
	minioRepo repository.MinIORepository
	fileRepo  repository.FileRepository
}

// NewFileService 创建文件服务实例
func NewFileService(minioRepo repository.MinIORepository, fileRepo repository.FileRepository) FileService {
	return &FileServiceImpl{
		minioRepo: minioRepo,
		fileRepo:  fileRepo,
	}
}

// UploadFile 上传文件
func (svc *FileServiceImpl) UploadFile(ctx context.Context, fileHeader *FileHeader, bizType string, bizID int, userID int) (dto.FileUploadResponse, error) {
	var response dto.FileUploadResponse
	// 确定文件类型
	fileType := string(detectFileType(fileHeader.OriginalFileName))

	// 根据文件类型设置存储路径前缀
	objectPrefix := getObjectPrefixByType(fileType)

	// 生成唯一的对象名
	ext := filepath.Ext(fileHeader.OriginalFileName)
	objectName := fmt.Sprintf("%s/%d%s", objectPrefix, time.Now().UnixNano(), ext)

	// 上传到MinIO
	url, err := svc.minioRepo.UploadFile(ctx, objectName, fileHeader.TemporaryFile)
	if err != nil {
		return response, err
	}

	// 图片与其他类型附件分开存储
	if fileType == string(model.FileTypeImage) {
		// 创建图片记录
		file := &model.Image{
			BizType:      bizType,
			BizID:        bizID,
			ObjectName:   objectName,
			URL:          url,
			FileName:     fileHeader.OriginalFileName,
			FileSize:     int(fileHeader.Size),
			ContentType:  fileHeader.ContentType,
			UploadUserID: userID,
		}
		if err := svc.fileRepo.CreateImageFile(ctx, file); err != nil {
			// 上传到数据库失败，删除MinIO中的文件
			_ = svc.minioRepo.DeleteFile(ctx, objectName)
			return response, err
		}
		response = dto.FileUploadResponse{
			ID:       file.ID,
			FileName: file.FileName,
			URL:      file.URL,
		}

		return response, nil
	} // 其他类型文件存储，暂不需要开发

	return response, nil
}

// detectFileType 检测文件类型
func detectFileType(filename string) model.FileType {
	ext := strings.ToLower(filepath.Ext(filename))

	// 图片类型
	imageExts := []string{".jpg", ".jpeg", ".png", ".gif", ".bmp", ".webp", ".svg", ".avif"}
	for _, e := range imageExts {
		if ext == e {
			return model.FileTypeImage
		}
	}

	return model.FileTypeOther
}

// getObjectPrefixByType 根据日期及文件类型获取存储路径前缀
func getObjectPrefixByType(fileType string) string {
	// 按月划分存储空间
	now := time.Now()
	yearMonth := now.Format("200601")

	switch fileType {
	case "image":
		return fmt.Sprintf("images/%s", yearMonth)
	default:
		return fmt.Sprintf("others/%s", yearMonth)
	}
}

// DeleteImage 删除图片
func (svc *FileServiceImpl) DeleteImage(ctx context.Context, imageID int, userID int) error {
	// 查询图片信息
	image, err := svc.fileRepo.GetImageByID(ctx, imageID)
	if err != nil {
		return err
	}
	if image == nil {
		return utils.NewBusinessError(utils.ErrCodeResourceNotFound, "图片不存在")
	}

	// 权限校验（仅上传者可删除）
	if image.UploadUserID != userID {
		return utils.NewBusinessError(utils.ErrCodePermissionDenied, "没有权限删除该图片")
	}

	// 删除MinIO中的文件
	if err := svc.minioRepo.DeleteFile(ctx, image.ObjectName); err != nil {
		return err
	}

	// 删除数据库记录
	if err := svc.fileRepo.DeleteImage(ctx, imageID); err != nil {
		return err
	}

	return nil
}
