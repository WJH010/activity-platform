package repository

import (
	"activity-platform/internal/event/dto"
	"activity-platform/internal/event/model"
	"activity-platform/internal/utils"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// EventRepository 数据访问接口，定义数据访问的方法集
type EventRepository interface {
	// ExecTransaction 执行事务
	ExecTransaction(ctx context.Context, fn func(tx *gorm.DB) error) error
	// List 分页查询
	List(ctx context.Context, page, pageSize int, eventStatus string, queryScope string, eventTitle string) ([]*dto.EventListResponse, int64, error)
	// GetEventDetail 获取活动详情
	GetEventDetail(ctx context.Context, eventID int) (*model.Event, error)
	// ListEventImage 获取活动图片列表
	ListEventImage(ctx context.Context, bizID int) []dto.Image
	// GetEventUserMap 查询活动-用户关联映射
	GetEventUserMap(ctx context.Context, eventID int, userID int) (*model.EventUserMapping, error)
	// CreatEventUserMap 创建活动-用户关联映射,将用户添加到活动中
	CreatEventUserMap(ctx context.Context, tx *gorm.DB, eventUserMapping *model.EventUserMapping) error
	// UpdateEUMapDeleteFlag 更新活动-用户关联删除标志
	UpdateEUMapDeleteFlag(ctx context.Context, tx *gorm.DB, eventID int, userID int, isDeleted string) error
	// IsUserRegistered 查询用户是否已报名活动
	IsUserRegistered(ctx context.Context, eventID int, userID int) (bool, error)
	// ListUserRegisteredEvents 获取用户已报名活动列表
	ListUserRegisteredEvents(ctx context.Context, page, pageSize int, userID int, eventStatus string) ([]*model.Event, int64, error)
	// CreateEvent 创建活动
	CreateEvent(ctx context.Context, tx *gorm.DB, event *model.Event) error
	// UpdateEvent 更新活动
	UpdateEvent(ctx context.Context, tx *gorm.DB, eventID int, updateFields map[string]interface{}) error
	// ListEventRegisteredUser 查询已报名活动的用户列表
	ListEventRegisteredUser(ctx context.Context, page, pageSize int, eventID int) ([]*dto.ListEventRegUserResponse, int64, error)
	// GetEventByTitle 根据活动标题查询活动
	GetEventByTitle(ctx context.Context, title string) (*model.Event, error)
	// CreateEventUserInfoMapping 创建活动用户信息映射
	CreateEventUserInfoMapping(ctx context.Context, tx *gorm.DB, mappings []*model.EventUserInfoMapping) error
	// GetEventUserInfoByEventID 根据活动ID查询所需用户信息
	GetEventUserInfoByEventID(ctx context.Context, eventID int) ([]*dto.EventUserInfo, error)
	// CreateEventRegistrationInfo 创建活动报名信息
	CreateEventRegistrationInfo(ctx context.Context, tx *gorm.DB, registrationInfo *model.EventRegistrationInfo) error
	// DeleteEventRegistrationInfo 删除活动报名信息
	DeleteEventRegistrationInfo(ctx context.Context, tx *gorm.DB, event_id int, user_id int) error
	// ListUpdatedSince 查询指定时间后有更新的活动（含已删除，含详情，用于增量同步）
	ListUpdatedSince(ctx context.Context, since time.Time, pageSize int, page int) ([]dto.EventUpdatedSinceResponse, int64, error)
}

// EventRepositoryImpl 实现接口的具体结构体
type EventRepositoryImpl struct {
	db *gorm.DB
}

// NewEventRepository 创建数据访问实例
func NewEventRepository(db *gorm.DB) EventRepository {
	return &EventRepositoryImpl{db: db}
}

// ExecTransaction 实现事务执行（使用 GORM 的 Transaction 方法）
func (repo *EventRepositoryImpl) ExecTransaction(ctx context.Context, fn func(tx *gorm.DB) error) error {
	return repo.db.WithContext(ctx).Transaction(fn)
}

// List 分页查询数据
func (repo *EventRepositoryImpl) List(ctx context.Context, page, pageSize int, eventStatus string, queryScope string, eventTitle string) ([]*dto.EventListResponse, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}

	offset := (page - 1) * pageSize
	var events []*dto.EventListResponse
	var total int64

	query := repo.db.WithContext(ctx)
	// 构建基础查询
	query = query.Table("events e").
		Select(`e.*, 
				COUNT(DISTINCT m.user_id) as member_count`).
		Joins("LEFT JOIN event_user_mappings m ON e.id = m.event_id AND m.is_deleted = ?", utils.DeletedFlagNo).
		Group("e.id")

	if queryScope != "" {
		// 如果传入了查询范围，则添加查询条件
		// 如果传入了查询范围为DELETED，则查询已删除的活动
		if queryScope == utils.QueryScopeDeleted {
			query = query.Where("e.is_deleted = ?", utils.DeletedFlagYes) // 查询已删除的活动
		}
		if queryScope == utils.QueryScopeAll {
			// 如果传入了查询范围为ALL，则查询所有活动
		}
	} else {
		// 默认查询未删除的活动
		query = query.Where("e.is_deleted = ?", utils.DeletedFlagNo)
	}

	// 根据活动状态拼接查询条件
	if eventStatus == model.EventStatusInProgress {
		// 进行中的活动：报名时间在当前时间范围内
		query = query.Where("e.registration_start_time <= ? AND e.registration_end_time >= ?", time.Now(), time.Now())
		// 按活动开始时间升序排列
		query = query.Order("e.event_start_time ASC")
	} else if eventStatus == model.EventStatusCompleted {
		// 已结束的活动：报名截止时间在当前时间之前
		query = query.Where("e.registration_end_time < ?", time.Now())
		// 按活动开始时间降序排列
		query = query.Order("e.event_start_time DESC")
	} else if eventStatus == model.EventStatusNotBegun {
		// 未开始的活动：报名开始时间在当前时间之后
		query = query.Where("e.registration_start_time > ?", time.Now())
		// 按活动开始时间升序排列
		query = query.Order("e.event_start_time ASC")
	}

	// 如果传入了活动标题，则添加标题查询条件
	if eventTitle != "" {
		query = query.Where("e.title LIKE ?", "%"+eventTitle+"%")
	}

	// 计算总数
	countQuery := query.Session(&gorm.Session{})
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, utils.NewSystemError(fmt.Errorf("计算总数时数据库查询失败: %v", err))
	}

	// 分页查询数据
	if err := query.Offset(offset).Limit(pageSize).Find(&events).Error; err != nil {
		return nil, 0, utils.NewSystemError(fmt.Errorf("数据库查询失败: %v", err))
	}

	fmt.Println(query.ToSQL(func(tx *gorm.DB) *gorm.DB {
		return tx.Offset(offset).Limit(pageSize).Find(&events)
	}))

	return events, total, nil
}

// GetEventDetail 获取活动详情
func (repo *EventRepositoryImpl) GetEventDetail(ctx context.Context, eventID int) (*model.Event, error) {
	var event model.Event

	// 查询活动详情
	result := repo.db.WithContext(ctx).First(&event, eventID)
	err := result.Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, utils.NewBusinessError(utils.ErrCodeResourceNotFound, "活动不存在或已被删除，请刷新页面后重试")
		}
		return nil, utils.NewSystemError(fmt.Errorf("数据库查询失败: %v", err))
	}

	return &event, nil
}

// ListEventImage 获取活动图片列表
func (repo *EventRepositoryImpl) ListEventImage(ctx context.Context, bizID int) []dto.Image {
	var images []dto.Image

	err := repo.db.WithContext(ctx).
		Table("images").
		Select("id AS image_id, url").
		Where("biz_type = ? AND biz_id = ?", utils.TypeEvent, bizID).
		Find(&images).Error

	if err != nil {
		logrus.Errorf("获取活动图片失败: %v", err) // 只记录异常，不影响活动信息的返回
		return nil
	}

	return images
}

// GetEventUserMap 查询活动-用户关联映射
func (repo *EventRepositoryImpl) GetEventUserMap(ctx context.Context, eventID int, userID int) (*model.EventUserMapping, error) {
	var mapping model.EventUserMapping

	// 查询活动-用户关联映射
	err := repo.db.WithContext(ctx).
		Where("event_id = ? AND user_id = ?", eventID, userID).
		First(&mapping).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // 映射不存在，只用来判断用户是否已报名活动，所以不返回异常
		}
		return nil, utils.NewSystemError(fmt.Errorf("数据库查询失败: %v", err))
	}

	return &mapping, nil
}

// CreatEventUserMap 创建活动-用户关联映射,将用户添加到活动中
func (repo *EventRepositoryImpl) CreatEventUserMap(ctx context.Context, tx *gorm.DB, eventUserMapping *model.EventUserMapping) error {
	err := tx.WithContext(ctx).Create(eventUserMapping).Error
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return utils.NewBusinessError(utils.ErrCodeResourceExists, "已报名该活动，请勿重复报名")
		} else {
			return utils.NewSystemError(fmt.Errorf("创建活动-用户关联映射失败: %w", err))
		}
	}
	return nil
}

// UpdateEUMapDeleteFlag 更新活动-用户关联删除标志
func (repo *EventRepositoryImpl) UpdateEUMapDeleteFlag(ctx context.Context, tx *gorm.DB, eventID int, userID int, isDeleted string) error {
	result := tx.WithContext(ctx).Model(&model.EventUserMapping{}).
		Where("event_id = ?", eventID).
		Where("user_id = ?", userID).
		Updates(map[string]interface{}{
			"is_deleted": isDeleted,
		})

	if result.Error != nil {
		return utils.NewSystemError(fmt.Errorf("数据更新异常: %w", result.Error))
	}
	if result.RowsAffected == 0 {
		return utils.NewBusinessError(utils.ErrCodeResourceNotFound, "数据更新异常，未找到活动或状态已更新，请刷新页面后重试")
	}

	return nil
}

// IsUserRegistered 查询用户是否已报名活动
func (repo *EventRepositoryImpl) IsUserRegistered(ctx context.Context, eventID int, userID int) (bool, error) {
	var count int64
	err := repo.db.WithContext(ctx).
		Model(&model.EventUserMapping{}).
		Where("event_id = ? AND user_id = ? AND is_deleted = ?", eventID, userID, utils.DeletedFlagNo).
		Count(&count).Error

	if err != nil {
		return false, utils.NewSystemError(fmt.Errorf("查询用户是否已报名活动失败: %w", err))
	}

	return count > 0, nil
}

// ListUserRegisteredEvents 获取用户已报名活动列表
func (repo *EventRepositoryImpl) ListUserRegisteredEvents(ctx context.Context, page, pageSize int, userID int, eventStatus string) ([]*model.Event, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}

	offset := (page - 1) * pageSize
	var events []*model.Event
	var total int64

	query := repo.db.WithContext(ctx)

	query = query.Table("events e").
		Joins("JOIN event_user_mappings eum ON e.id = eum.event_id").
		Where("eum.user_id = ? AND e.is_deleted = ? AND eum.is_deleted = ?", userID, utils.DeletedFlagNo, utils.DeletedFlagNo).
		Find(&events)

	// 根据活动状态拼接查询条件
	if eventStatus == model.EventStatusInProgress {
		// 进行中的活动：报名时间在当前时间范围内
		query = query.Where("e.registration_start_time <= ? AND e.registration_end_time >= ?", time.Now(), time.Now())
		// 按活动开始时间升序排列
		query = query.Order("e.event_start_time ASC")
	} else if eventStatus == model.EventStatusCompleted {
		// 已结束的活动：报名截止时间在当前时间之前
		query = query.Where("e.registration_end_time < ?", time.Now())
		// 按活动开始时间降序排列
		query = query.Order("e.event_start_time DESC")
	}

	// 计算总数
	countQuery := query.Session(&gorm.Session{})
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, utils.NewSystemError(fmt.Errorf("计算总数时数据库查询失败: %v", err))
	}

	// 分页查询数据
	if err := query.Offset(offset).Limit(pageSize).Find(&events).Error; err != nil {
		return nil, 0, utils.NewSystemError(fmt.Errorf("数据库查询失败: %v", err))
	}

	return events, total, nil
}

// CreateEvent 创建活动
func (repo *EventRepositoryImpl) CreateEvent(ctx context.Context, tx *gorm.DB, event *model.Event) error {
	// 插入新活动
	if err := tx.WithContext(ctx).Create(event).Error; err != nil {
		return utils.NewSystemError(fmt.Errorf("创建活动失败: %w", err))
	}

	return nil
}

// UpdateEvent 更新活动
func (repo *EventRepositoryImpl) UpdateEvent(ctx context.Context, tx *gorm.DB, eventID int, updateFields map[string]interface{}) error {
	// 更新活动信息
	if tx == nil {
		tx = repo.db
	}
	result := tx.WithContext(ctx).Model(&model.Event{}).
		Where("id = ?", eventID).
		Updates(updateFields)

	if result.Error != nil {
		return utils.NewSystemError(fmt.Errorf("更新活动信息失败: %w", result.Error))
	}
	if result.RowsAffected == 0 {
		return utils.NewBusinessError(utils.ErrCodeResourceNotFound, "更新活动信息失败，活动数据异常，请刷新页面后重试")
	}
	return nil
}

// ListEventRegisteredUser 查询已报名活动的用户列表
func (repo *EventRepositoryImpl) ListEventRegisteredUser(ctx context.Context, page, pageSize int, eventID int) ([]*dto.ListEventRegUserResponse, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}

	offset := (page - 1) * pageSize
	var users []*dto.ListEventRegUserResponse
	var total int64

	query := repo.db.WithContext(ctx).
		Table("event_registration_info u").
		Select("u.name, u.phone_number, u.email, u.unit, u.department, u.position, u.industry, i.industry_name").
		Joins("LEFT JOIN industries i ON u.industry = i.industry_code").
		Where("u.event_id = ?", eventID)

	// 计算总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, utils.NewSystemError(fmt.Errorf("计算总数时数据库查询失败: %v", err))
	}

	// 分页查询数据
	if err := query.Offset(offset).Limit(pageSize).Find(&users).Error; err != nil {
		return nil, 0, utils.NewSystemError(fmt.Errorf("数据库查询失败: %v", err))
	}

	return users, total, nil
}

// GetEventByTitle 根据活动标题查询活动
func (repo *EventRepositoryImpl) GetEventByTitle(ctx context.Context, title string) (*model.Event, error) {
	var event model.Event

	result := repo.db.WithContext(ctx).Where("title = ? AND is_deleted = ?", title, utils.DeletedFlagNo).First(&event)
	err := result.Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, utils.NewSystemError(fmt.Errorf("数据库查询失败: %v", err))
	}

	return &event, nil
}

// CreateEventUserInfoMapping 创建活动所需用户信息映射
func (repo *EventRepositoryImpl) CreateEventUserInfoMapping(ctx context.Context, tx *gorm.DB, mappings []*model.EventUserInfoMapping) error {
	// 插入新映射
	result := tx.WithContext(ctx).Create(mappings)
	err := result.Error
	// 异常处理
	if err != nil {
		// 检查是否是重复键错误（需在数据库层配置唯一约束）
		if exist, value, _ := utils.IsUniqueConstraintError(err); exist {
			return utils.NewBusinessError(utils.ErrCodeResourceExists, fmt.Sprintf("关联关系（%s）已存在，不可重复创建", value))
		}
		return utils.NewSystemError(fmt.Errorf("数据库插入失败: %w", err))
	}

	return nil
}

// GetEventUserInfoByEventID 根据活动ID查询所需用户信息
func (repo *EventRepositoryImpl) GetEventUserInfoByEventID(ctx context.Context, eventID int) ([]*dto.EventUserInfo, error) {
	var mappings []*dto.EventUserInfo

	result := repo.db.WithContext(ctx).Table("event_user_info_mappings eum").
		Select("eum.user_info_id, eui.code, eui.name").
		Joins("JOIN event_user_info eui ON eum.user_info_id = eui.id").
		Where("eum.event_id = ?", eventID).
		Find(&mappings)
	err := result.Error

	if err != nil {
		return nil, utils.NewSystemError(fmt.Errorf("数据库查询失败: %v", err))
	}

	return mappings, nil
}

// CreateEventRegistrationInfo 创建活动报名信息
func (repo *EventRepositoryImpl) CreateEventRegistrationInfo(ctx context.Context, tx *gorm.DB, registrationInfo *model.EventRegistrationInfo) error {
	// 插入新报名信息
	if err := tx.WithContext(ctx).Create(registrationInfo).Error; err != nil {
		return utils.NewSystemError(fmt.Errorf("创建活动报名信息失败: %w", err))
	}

	return nil
}

// DeleteEventRegistrationInfo 删除活动报名信息
func (repo *EventRepositoryImpl) DeleteEventRegistrationInfo(ctx context.Context, tx *gorm.DB, event_id int, user_id int) error {
	// 删除报名信息
	result := repo.db.WithContext(ctx).
		Model(&model.EventRegistrationInfo{}).
		Where("event_id = ? AND user_id = ?", event_id, user_id).
		Delete(&model.EventRegistrationInfo{})
	err := result.Error
	if err != nil {
		return utils.NewSystemError(fmt.Errorf("删除活动报名信息失败: %w", err))
	}

	return nil
}

// ListUpdatedSince 查询指定时间后有更新的活动
func (repo *EventRepositoryImpl) ListUpdatedSince(ctx context.Context, since time.Time, pageSize int, page int) ([]dto.EventUpdatedSinceResponse, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 50
	}

	offset := (page - 1) * pageSize
	var events []dto.EventUpdatedSinceResponse

	query := repo.db.WithContext(ctx).Table("events e").
		Select("e.id, e.title, e.detail, e.event_start_time, e.event_end_time, " +
			"e.registration_start_time, e.registration_end_time, e.event_address, e.is_deleted, e.update_time")

	// 核心条件：update_time >= since（不过滤is_deleted）
	query = query.Where("e.update_time >= ?", since)

	// 按更新时间升序排列
	query = query.Order("e.update_time ASC")

	// 计算总数
	var total int64
	countQuery := query.Session(&gorm.Session{})
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, utils.NewSystemError(fmt.Errorf("计算总数时数据库查询失败: %v", err))
	}

	// 分页查询
	if err := query.Offset(offset).Limit(pageSize).Find(&events).Error; err != nil {
		return nil, 0, utils.NewSystemError(fmt.Errorf("数据库查询失败: %v", err))
	}

	return events, total, nil
}
