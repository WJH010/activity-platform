package service

import (
	"context"
	"event-platform/internal/cache"
	chatmodel "event-platform/internal/chat/model"
	chatSvc "event-platform/internal/chat/service"
	"event-platform/internal/event/dto"
	"event-platform/internal/event/model"
	"event-platform/internal/event/repository"
	"event-platform/internal/event/stock"
	filerepo "event-platform/internal/file/repository"
	userrepo "event-platform/internal/user/repository"
	"event-platform/internal/utils"
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// EventService 定义事件服务接口，提供事件相关的业务逻辑方法
type EventService interface {
	// GetEventStatus 根据开始时间和结束时间计算活动状态
	GetEventStatus(registrationStartTime time.Time, registrationEndTime time.Time) string
	// ListEvent 分页查询活动列表
	ListEvent(ctx context.Context, page, pageSize int, eventStatus string, queryScope string, eventTitle string) ([]*dto.EventListResponse, int64, error)
	// GetEventDetail 获取活动详情
	GetEventDetail(ctx context.Context, eventID int) (*model.Event, error)
	// RegistrationEvent 活动报名
	RegistrationEvent(ctx context.Context, eventID int, userID int) error
	// CancelRegistrationEvent 取消活动报名
	CancelRegistrationEvent(ctx context.Context, eventID int, userID int) error
	// IsUserRegistered 查询用户是否已报名活动
	IsUserRegistered(ctx context.Context, eventID int, userID int) (bool, error)
	// ListUserRegisteredEvents 获取用户已报名的活动列表
	ListUserRegisteredEvents(ctx context.Context, page, pageSize int, userID int, eventStatus string) ([]*model.Event, int64, error)
	// CreateEvent 创建活动
	CreateEvent(ctx context.Context, event *model.Event, imageIDList []int, userFieldIDList []int) error
	// UpdateEvent 更新活动
	UpdateEvent(ctx context.Context, eventID int, req dto.UpdateEventRequest, userID int) error
	// DeleteEvent 删除活动
	DeleteEvent(ctx context.Context, eventID int, userID int) error
	// ListEventRegisteredUser 获取活动报名用户列表
	ListEventRegisteredUser(ctx context.Context, page, pageSize int, eventID int) ([]*dto.ListEventRegUserResponse, int64, error)
}

// EventServiceImpl 实现 EventService 接口，提供事件相关的业务逻辑
type EventServiceImpl struct {
	eventRepo         repository.EventRepository         // 事件数据访问接口
	eventUserInfoRepo repository.EventUserInfoRepository // 用户信息字段数据访问接口
	userRepo          userrepo.UserRepository            // 用户数据访问接口
	fileRepo          filerepo.FileRepository            // 文件数据访问接口
	chatSvc           chatSvc.ChatGroupService           // 聊天服务接口
	stockSvc          *stock.StockService
	eventCache        *cache.Cache[int, *model.Event]
	//msgSvc            msgsvc.MsgGroupService             // 消息群组服务接口
}

// NewEventService 创建服务实例
func NewEventService(
	eventRepo repository.EventRepository,
	eventUserInfoRepo repository.EventUserInfoRepository,
	userRepo userrepo.UserRepository,
	fileRepo filerepo.FileRepository,
	chatSvc chatSvc.ChatGroupService,
	stockSvc *stock.StockService,
	eventCache *cache.Cache[int, *model.Event],
	//msgSvc msgsvc.MsgGroupService,
) EventService {
	return &EventServiceImpl{
		eventRepo:         eventRepo,
		eventUserInfoRepo: eventUserInfoRepo,
		userRepo:          userRepo,
		fileRepo:          fileRepo,
		chatSvc:           chatSvc,
		stockSvc:          stockSvc,
		eventCache:        eventCache,
		//msgSvc:            msgSvc,
	}
}

// GetEventStatus 根据开始时间和结束时间计算活动状态
func (svc *EventServiceImpl) GetEventStatus(registrationStartTime time.Time, registrationEndTime time.Time) string {
	if registrationStartTime.After(time.Now()) {
		return "未开始"
	}
	if registrationStartTime.Before(time.Now()) && registrationEndTime.After(time.Now()) {
		return "正在进行"
	}
	if registrationEndTime.Before(time.Now()) {
		return "已结束"
	}
	return ""
}

// ListEvent 分页查询活动列表
func (svc *EventServiceImpl) ListEvent(ctx context.Context, page, pageSize int, eventStatus string, queryScope string, eventTitle string) ([]*dto.EventListResponse, int64, error) {
	events, total, err := svc.eventRepo.List(ctx, page, pageSize, eventStatus, queryScope, eventTitle)
	if err != nil {
		return nil, 0, err
	}
	for _, e := range events {
		if e.MaxRegistrants > 0 {
			e.RemainingQuota = e.MaxRegistrants - e.CurrentRegistrants
		}
	}
	return events, total, nil
}

// GetEventDetail 获取活动详情
func (svc *EventServiceImpl) GetEventDetail(ctx context.Context, eventID int) (*model.Event, error) {
	event, err := svc.eventRepo.GetEventDetail(ctx, eventID)
	if err != nil {
		return nil, err
	}

	// 获取关联图片列表
	images := svc.eventRepo.ListEventImage(ctx, eventID)
	// 添加图片到活动详情
	event.Images = make([]dto.Image, 0, len(images)) // 预分配空间，提高性能
	for _, img := range images {
		event.Images = append(event.Images, dto.Image{
			ImageID: img.ImageID,
			URL:     img.URL,
		})
	}

	// 获取关联用户信息字段列表
	userInfoList, err := svc.eventRepo.GetEventUserInfoByEventID(ctx, eventID)
	if err != nil {
		return nil, err
	}
	// 添加用户信息字段到活动详情
	event.UserInfo = make([]dto.EventUserInfo, 0, len(userInfoList)) // 预分配空间，提高性能
	for _, mapping := range userInfoList {
		event.UserInfo = append(event.UserInfo, dto.EventUserInfo{
			UserInfoID: mapping.UserInfoID,
			Code:       mapping.Code,
			Name:       mapping.Name,
		})
	}

	return event, nil
}

// RegistrationEvent 活动报名实现
func (svc *EventServiceImpl) RegistrationEvent(ctx context.Context, eventID int, userID int) error {
	var mapping *model.EventUserMapping
	// 检查活动是否存在（本地缓存优先，内置 singleflight 防击穿）
	event, err := svc.eventCache.GetOrLoad(eventID, func() (*model.Event, error) {
		return svc.eventRepo.GetEventDetail(ctx, eventID)
	})
	if err != nil {
		return err
	}
	// 检查活动是否已删除
	if event.IsDeleted == utils.DeletedFlagYes {
		return utils.NewBusinessError(utils.ErrCodeBusinessLogicError, "活动已失效")
	}
	// 检查活动是否在报名时间内
	if event.RegistrationStartTime.After(time.Now()) || event.RegistrationEndTime.Before(time.Now()) {
		return utils.NewBusinessError(utils.ErrCodeBusinessLogicError, "未在活动报名时间内")
	}

	// 检查活动是否已满员
	if event.MaxRegistrants > 0 {
		decrResult, err := svc.stockSvc.Decr(ctx, eventID)
		if err != nil {
			logrus.Warnf("Redis库存预扣失败[eventID=%d]: %v", eventID, err)
		}
		if decrResult == stock.DecrResultSoldOut {
			return utils.NewBusinessError(utils.ErrCodeResourceQuotaExceeded, "报名人数已满")
		}
	}

	// 检查用户信息是否完整
	// 根据活动ID获取所需信息
	userInfoList, err := svc.eventRepo.GetEventUserInfoByEventID(ctx, eventID)
	if err != nil {
		svc.stockSvc.Incr(ctx, eventID)
		return err
	}
	// 获取用户信息
	user, err := svc.userRepo.GetUserByID(ctx, userID)
	if err != nil || user == nil {
		svc.stockSvc.Incr(ctx, eventID)
		return utils.NewBusinessError(utils.ErrCodeBusinessLogicError, "加载用户信息失败")
	}

	// 检查用户是否提供了所有必填的用户信息字段，并组装报名信息
	registration := &model.EventRegistrationInfo{
		EventID: eventID,
		UserID:  userID,
	}
	for _, field := range userInfoList {
		if field.Code == "name" {
			if user.Name == "" {
				svc.stockSvc.Incr(ctx, eventID)
				return utils.NewBusinessError(utils.ErrCodeBusinessLogicError, "请填写姓名")
			}
			registration.Name = user.Name
		}
		if field.Code == "phone_number" {
			if user.PhoneNumber == "" {
				svc.stockSvc.Incr(ctx, eventID)
				return utils.NewBusinessError(utils.ErrCodeBusinessLogicError, "请填写手机号")
			}
			registration.PhoneNumber = user.PhoneNumber
		}
		if field.Code == "email" {
			if user.Email == "" {
				svc.stockSvc.Incr(ctx, eventID)
				return utils.NewBusinessError(utils.ErrCodeBusinessLogicError, "请填写邮箱")
			}
			registration.Email = user.Email
		}
		if field.Code == "unit" {
			if user.Unit == "" {
				svc.stockSvc.Incr(ctx, eventID)
				return utils.NewBusinessError(utils.ErrCodeBusinessLogicError, "请填写单位")
			}
			registration.Unit = user.Unit
		}
		if field.Code == "department" {
			if user.Department == "" {
				svc.stockSvc.Incr(ctx, eventID)
				return utils.NewBusinessError(utils.ErrCodeBusinessLogicError, "请填写部门")
			}
			registration.Department = user.Department
		}
		if field.Code == "position" {
			if user.Position == "" {
				svc.stockSvc.Incr(ctx, eventID)
				return utils.NewBusinessError(utils.ErrCodeBusinessLogicError, "请填写职位")
			}
			registration.Position = user.Position
		}
		if field.Code == "industry" {
			if user.Industry == "" {
				svc.stockSvc.Incr(ctx, eventID)
				return utils.NewBusinessError(utils.ErrCodeBusinessLogicError, "请填写行业")
			}
			registration.Industry = user.Industry
		}
	}

	mapping, err = svc.eventRepo.GetEventUserMap(ctx, eventID, userID)
	if err != nil {
		svc.stockSvc.Incr(ctx, eventID)
		return err
	}

	// 使用 GORM 函数式事务
	err = svc.eventRepo.ExecTransaction(ctx, func(tx *gorm.DB) error {
		// 执行活动报名逻辑
		// 如果关联关系不存在，则创建新的关联关系
		if mapping == nil {
			mapping = &model.EventUserMapping{
				UserID:  userID,
				EventID: eventID,
			}
			err = svc.eventRepo.CreatEventUserMap(ctx, tx, mapping)
			if err != nil {
				return err
			}
		} else if mapping.IsDeleted == utils.DeletedFlagYes {
			// 如果关联关系软删除了，则恢复
			err = svc.eventRepo.UpdateEUMapDeleteFlag(ctx, tx, eventID, userID, utils.DeletedFlagNo)
			if err != nil {
				return err
			}
		} else if mapping.IsDeleted == utils.DeletedFlagNo {
			// 如果关联关系存在且有效，则返回错误提示
			return utils.NewBusinessError(utils.ErrCodeResourceExists, "已报名该活动，请勿重复报名")
		}

		// 新建活动报名信息
		err = svc.eventRepo.CreateEventRegistrationInfo(ctx, tx, registration)
		if err != nil {
			return err
		}

		// 条件更新：原子校验名额 + 递增计数
		if err := svc.eventRepo.IncrementRegistrants(ctx, tx, eventID); err != nil {
			return err
		}

		return nil // 返回 nil，GORM 自动提交
	})

	// 处理事务执行结果
	if err != nil {
		if event.MaxRegistrants > 0 {
			svc.stockSvc.Incr(ctx, eventID)
		}
		return err
	}

	// 报名成功，失效本地缓存（current_registrants已变化）
	svc.eventCache.Delete(eventID)

	// 报名成功后，添加用户到活动对应的消息群组，消息群组添加失败不影响报名成功
	// 查询活动对应的消息群组
	// group, err := svc.msgSvc.GetMsgGroupByEventID(ctx, eventID)
	// if err != nil || group == nil {
	// 	// 不存在对应的消息群组，返回错误
	// 	return utils.NewBusinessError(utils.ErrCodeResourceNotFound, "进入活动消息群组失败，请联系管理员")
	// }
	// // 将用户添加到消息群组
	// err = svc.msgSvc.AddUserToGroup(ctx, group.ID, []int{userID}, userID)
	// if err != nil {
	// 	return utils.NewBusinessError(utils.ErrCodeBusinessLogicError, "进入活动消息群组失败，请联系管理员")
	// }

	// 报名成功后，添加用户到活动对应的聊天群组，聊天群组添加失败不影响报名成功
	group, err := svc.chatSvc.GetGroupByID(ctx, event.ChatGroupID)
	if err != nil || group == nil {
		return utils.NewBusinessError(utils.ErrCodeResourceNotFound, "进入活动聊天群组失败，请联系管理员")
	}
	// 将用户添加到聊天群组
	err = svc.chatSvc.AddUserToGroup(ctx, group.ID, userID)
	if err != nil {
		return utils.NewBusinessError(utils.ErrCodeBusinessLogicError, "进入活动聊天群组失败，请联系管理员")
	}

	return nil
}

// CancelRegistrationEvent 取消活动报名
func (svc *EventServiceImpl) CancelRegistrationEvent(ctx context.Context, eventID int, userID int) error {
	// 检查活动是否存在（本地缓存优先，内置 singleflight 防击穿）
	event, err := svc.eventCache.GetOrLoad(eventID, func() (*model.Event, error) {
		return svc.eventRepo.GetEventDetail(ctx, eventID)
	})
	if err != nil {
		return err
	}
	// 检查活动是否已删除
	if event.IsDeleted == utils.DeletedFlagYes {
		return utils.NewBusinessError(utils.ErrCodeBusinessLogicError, "活动已失效")
	}

	// 检查活动是否已开始
	if event.EventStartTime.Before(time.Now()) {
		return utils.NewBusinessError(utils.ErrCodeBusinessLogicError, "活动已开始，无法取消报名")
	}
	// 使用 GORM 函数式事务
	err = svc.eventRepo.ExecTransaction(ctx, func(tx *gorm.DB) error {
		// 执行取消报名逻辑
		err = svc.eventRepo.UpdateEUMapDeleteFlag(ctx, tx, eventID, userID, utils.DeletedFlagYes)
		if err != nil {
			return err
		}
		// 删除报名信息
		err = svc.eventRepo.DeleteEventRegistrationInfo(ctx, tx, eventID, userID)
		if err != nil {
			return err
		}
		if err := svc.eventRepo.DecrementRegistrants(ctx, tx, eventID); err != nil {
			return err
		}
		return nil // 返回 nil，GORM 自动提交
	})

	// 处理事务执行结果
	if err != nil {
		return err
	}

	// 取消报名成功后，将用户从活动对应的消息群组移除
	// group, err := svc.msgSvc.GetMsgGroupByEventID(ctx, eventID)
	// if err != nil {
	// 	return utils.NewBusinessError(utils.ErrCodeResourceNotFound, "退出活动消息群组失败，请联系管理员处理")
	// }
	// // 将用户从消息群组移除
	// err = svc.msgSvc.DeleteUserFromGroup(ctx, group.ID, []int{userID}, userID)
	// if err != nil {
	// 	return utils.NewBusinessError(utils.ErrCodeBusinessLogicError, "退出活动消息群组失败，请联系管理员处理")
	// }

	if event.MaxRegistrants > 0 {
		svc.stockSvc.Incr(ctx, eventID)
	}

	// 取消报名成功后，将用户从活动对应的聊天群组移除
	group, err := svc.chatSvc.GetGroupByID(ctx, eventID)
	if err != nil || group == nil {
		return utils.NewBusinessError(utils.ErrCodeResourceNotFound, "退出活动聊天群组失败，请联系管理员处理")
	}
	// 将用户从聊天群组移除
	err = svc.chatSvc.DeleteUserFromGroup(ctx, group.ID, userID)
	if err != nil {
		return utils.NewBusinessError(utils.ErrCodeBusinessLogicError, "退出活动聊天群组失败，请联系管理员处理")
	}

	// 取消报名成功，失效本地缓存（current_registrants已变化）
	svc.eventCache.Delete(eventID)

	return nil
}

// IsUserRegistered 查询用户是否已报名活动
func (svc *EventServiceImpl) IsUserRegistered(ctx context.Context, eventID int, userID int) (bool, error) {
	return svc.eventRepo.IsUserRegistered(ctx, eventID, userID)
}

// ListUserRegisteredEvents 获取用户已报名的活动列表
func (svc *EventServiceImpl) ListUserRegisteredEvents(ctx context.Context, page, pageSize int, userID int, eventStatus string) ([]*model.Event, int64, error) {
	return svc.eventRepo.ListUserRegisteredEvents(ctx, page, pageSize, userID, eventStatus)
}

// CreateEvent 创建活动
func (svc *EventServiceImpl) CreateEvent(ctx context.Context, event *model.Event, imageIDList []int, userInfoIDList []int) error {
	// 检查是否有重复的活动标题
	existingEvent, err := svc.eventRepo.GetEventByTitle(ctx, event.Title)
	if err != nil {
		return err
	}
	if existingEvent != nil {
		return utils.NewBusinessError(utils.ErrCodeResourceExists, "已存在同名活动，请修改标题后重试")
	}

	// 检查活动时间是否合理
	if event.EventStartTime.After(event.EventEndTime) {
		return utils.NewBusinessError(utils.ErrCodeBusinessLogicError, "活动开始时间不能晚于结束时间")
	}
	if event.RegistrationStartTime.After(event.RegistrationEndTime) {
		return utils.NewBusinessError(utils.ErrCodeBusinessLogicError, "报名开始时间不能晚于结束时间")
	}

	// 使用 GORM 函数式事务
	err = svc.eventRepo.ExecTransaction(ctx, func(tx *gorm.DB) error {
		// 创建活动
		if err := svc.eventRepo.CreateEvent(ctx, tx, event); err != nil {
			return err
		}

		// 如果有图片，更新images表的biz_id和biz_type
		if len(imageIDList) > 0 {
			if err := svc.fileRepo.BatchUpdateImageBizID(ctx, tx, imageIDList, event.ID, utils.TypeEvent); err != nil {
				return err
			}
		}

		// 创建用户信息自定义字段
		if len(userInfoIDList) > 0 {
			mappings := make([]*model.EventUserInfoMapping, 0, len(userInfoIDList))
			for _, userInfoID := range userInfoIDList {
				mappings = append(mappings, &model.EventUserInfoMapping{
					EventID:    event.ID,
					UserInfoID: userInfoID,
				})
			}
			if err := svc.eventRepo.CreateEventUserInfoMapping(ctx, tx, mappings); err != nil {
				return err
			}
		} else {
			// 默认需要全部用户信息
			// 从数据库查询所有用户信息ID
			userInfoList, err := svc.eventUserInfoRepo.List(ctx, dto.ListEventUserInfoRequest{
				IsDeleted: utils.DeletedFlagNo,
			})
			if err != nil {
				return err
			}
			mappings := make([]*model.EventUserInfoMapping, 0, len(userInfoList))
			for _, userInfo := range userInfoList {
				mappings = append(mappings, &model.EventUserInfoMapping{
					EventID:    event.ID,
					UserInfoID: userInfo.ID,
				})
			}
			if err := svc.eventRepo.CreateEventUserInfoMapping(ctx, tx, mappings); err != nil {
				return err
			}
		}

		return nil // 返回 nil，GORM 自动提交
	})

	// 处理事务执行结果
	if err != nil {
		return err
	}

	// 活动创建成功后，创建活动消息群组，消息群组创建失败不影响活动创建成功
	// 构建消息群组模型
	// 检查是否已存在对应的消息群组，理论上不应该存在
	// group, err := svc.msgSvc.GetMsgGroupByEventID(ctx, event.ID)
	// if err != nil || group != nil {
	// 	// 已存在对应的消息群组，直接返回成功，(在正确的业务流程下不应该出现这种情况)
	// 	return nil
	// }
	// msgGroup := &msgmodel.UserMessageGroup{
	// 	GroupName:      event.Title,
	// 	Desc:           "由活动" + event.Title + "自动创建",
	// 	EventID:        event.ID,
	// 	IncludeAllUser: utils.FlagNo,
	// 	CreateUser:     event.CreateUser,
	// 	UpdateUser:     event.UpdateUser,
	// }
	// if err = svc.msgSvc.CreateMsgGroup(ctx, msgGroup, []int{}); err != nil {
	// 	return utils.NewBusinessError(utils.ErrCodeServerInternalError, "自动创建活动消息群组失败"+err.Error())
	// }

	// // 更新活动的消息群组ID
	// updateFields := make(map[string]interface{})
	// updateFields["group_id"] = msgGroup.ID

	// if err = svc.eventRepo.UpdateEvent(ctx, nil, event.ID, updateFields); err != nil {
	// 	return err
	// }

	// 初始化活动名额
	if event.MaxRegistrants > 0 {
		if err := svc.stockSvc.InitWithTTL(ctx, event.ID, event.MaxRegistrants, 0, event.RegistrationEndTime); err != nil {
			logrus.Warnf("初始化活动名额失败[eventID=%d]: %v", event.ID, err)
		}
	}

	chatGroup := &chatmodel.ChatGroup{
		GroupName: event.Title,
		Desc:      "由活动 '" + event.Title + "' 自动创建",
	}

	createdGroup, err := svc.chatSvc.CreateGroup(ctx, chatGroup, event.CreateUser)
	if err != nil {
		return utils.NewBusinessError(utils.ErrCodeServerInternalError, "自动创建活动聊天群组失败"+err.Error())
	} else {
		// 更新活动中的 ChatGroupID
		updateFields := make(map[string]interface{})
		updateFields["chat_group_id"] = createdGroup.ID
		if err = svc.eventRepo.UpdateEvent(ctx, nil, event.ID, updateFields); err != nil {
			return utils.NewBusinessError(utils.ErrCodeServerInternalError, "自动更新活动聊天群组ID失败"+err.Error())
		}
	}

	return nil
}

// UpdateEvent 更新活动
func (svc *EventServiceImpl) UpdateEvent(ctx context.Context, eventID int, req dto.UpdateEventRequest, userID int) error {
	// 检查活动是否存在
	event, err := svc.eventRepo.GetEventDetail(ctx, eventID)
	if err != nil {
		return err
	}

	// 当标题修改时，检查是否有重复的活动标题
	if req.Title != nil && *req.Title != event.Title {
		existingEvent, err := svc.eventRepo.GetEventByTitle(ctx, *req.Title)
		if err != nil {
			return err
		}
		if existingEvent != nil {
			return utils.NewBusinessError(utils.ErrCodeResourceExists, "已存在同名活动，请修改标题后重试")
		}
	}

	// 构建更新字段映射
	updateFields, err := makeUpdateFields(event, req)
	if err != nil {
		return err
	}

	var imageIDList []int
	if req.ImageIDList != nil {
		imageIDList = *req.ImageIDList
	}

	if len(updateFields) == 0 && len(imageIDList) == 0 {
		return nil // 无更新内容
	}

	// 设置更新人
	updateFields["update_user"] = userID

	// 使用 GORM 函数式事务
	err = svc.eventRepo.ExecTransaction(ctx, func(tx *gorm.DB) error {
		// 更新活动
		if err := svc.eventRepo.UpdateEvent(ctx, tx, eventID, updateFields); err != nil {
			return err
		}

		// 如果有图片，更新images表的biz_id和biz_type
		if len(imageIDList) > 0 {
			if err := svc.fileRepo.BatchUpdateImageBizID(ctx, tx, imageIDList, eventID, utils.TypeEvent); err != nil {
				return err
			}
		}

		return nil // 返回 nil，GORM 自动提交
	})

	// 处理事务执行结果
	if err != nil {
		return err
	}

	// 更新活动库存
	if req.MaxRegistrants != nil {
		currentRegistrants := event.CurrentRegistrants
		if err := svc.stockSvc.Init(ctx, eventID, *req.MaxRegistrants, currentRegistrants); err != nil {
			logrus.Warnf("更新活动库存失败[eventID=%d]: %v", eventID, err)
		}
	}

	// 更新活动缓存
	svc.eventCache.Delete(eventID)

	return nil
}

// makeUpdateFields 构建更新字段映射
func makeUpdateFields(event *model.Event, req dto.UpdateEventRequest) (map[string]interface{}, error) {
	updateFields := make(map[string]interface{})

	// 先处理时间字段，然后校验时间是否合理
	var eventStartTime, eventEndTime, registrationStartTime, registrationEndTime time.Time
	var err error
	if req.EventStartTime != nil {
		eventStartTime, err = utils.StringToTime(*req.EventStartTime)
		if err != nil {
			return nil, err
		}
		updateFields["event_start_time"] = eventStartTime
	} else {
		eventStartTime = event.EventStartTime
	}
	if req.EventEndTime != nil {
		eventEndTime, err = utils.StringToTime(*req.EventEndTime)
		if err != nil {
			return nil, err
		}
		updateFields["event_end_time"] = eventEndTime
	} else {
		eventEndTime = event.EventEndTime
	}
	// 检查活动时间是否合理
	if eventStartTime.After(eventEndTime) {
		return nil, utils.NewBusinessError(utils.ErrCodeBusinessLogicError, "活动开始时间不能晚于结束时间")
	}
	if req.RegistrationStartTime != nil {
		registrationStartTime, err = utils.StringToTime(*req.RegistrationStartTime)
		if err != nil {
			return nil, err
		}
		updateFields["registration_start_time"] = registrationStartTime

	} else {
		registrationStartTime = event.RegistrationStartTime
	}
	if req.RegistrationEndTime != nil {
		registrationEndTime, err = utils.StringToTime(*req.RegistrationEndTime)
		if err != nil {
			return nil, err
		}
		updateFields["registration_end_time"] = registrationEndTime
	} else {
		registrationEndTime = event.RegistrationEndTime
	}
	// 检查报名时间是否合理
	if registrationStartTime.After(registrationEndTime) {
		return nil, utils.NewBusinessError(utils.ErrCodeBusinessLogicError, "报名开始时间不能晚于结束时间")
	}

	if req.Title != nil {
		updateFields["title"] = *req.Title
	}
	if req.Detail != nil {
		updateFields["detail"] = *req.Detail
	}
	if req.EventAddress != nil {
		updateFields["event_address"] = *req.EventAddress
	}
	if req.RegistrationFee != nil {
		updateFields["registration_fee"] = *req.RegistrationFee
	}
	if req.CoverImageURL != nil {
		updateFields["cover_image_url"] = *req.CoverImageURL
	}
	if req.MaxRegistrants != nil {
		if *req.MaxRegistrants > 0 && *req.MaxRegistrants < event.CurrentRegistrants {
			return nil, utils.NewBusinessError(utils.ErrCodeBusinessLogicError, "最大报名人数不能小于当前已报名人数")
		}
		updateFields["max_registrants"] = *req.MaxRegistrants
	}

	return updateFields, nil
}

// DeleteEvent 删除活动
func (svc *EventServiceImpl) DeleteEvent(ctx context.Context, eventID int, userID int) error {
	// 检查活动是否存在
	event, err := svc.eventRepo.GetEventDetail(ctx, eventID)
	if err != nil {
		return err
	}
	// 检查活动是否已删除
	if event.IsDeleted == utils.DeletedFlagYes {
		return utils.NewBusinessError(utils.ErrCodeBusinessLogicError, "活动已失效")
	}

	// 使用 GORM 函数式事务
	err = svc.eventRepo.ExecTransaction(ctx, func(tx *gorm.DB) error {
		// 软删除（更新is_deleted为Y，记录更新人）
		updateFields := map[string]interface{}{
			"is_deleted":  utils.DeletedFlagYes,
			"update_user": userID,
		}
		if err := svc.eventRepo.UpdateEvent(ctx, tx, eventID, updateFields); err != nil {
			tx.Rollback()
			return err
		}

		return nil // 返回 nil，GORM 自动提交
	})

	// 处理事务执行结果
	if err != nil {
		return err
	}
	// 删除活动成功后，删除活动对应的消息群组，消息群组删除失败不影响活动删除成功
	// 查询活动对应的消息群组
	// group, err := svc.msgSvc.GetMsgGroupByEventID(ctx, eventID)
	// if err != nil || group == nil {
	// 	// 不存在对应的消息群组，直接返回成功
	// 	return nil
	// }
	// // 删除消息群组
	// err = svc.msgSvc.DeleteMsgGroup(ctx, group.ID, userID)
	// if err != nil {
	// 	return utils.NewBusinessError(utils.ErrCodeServerInternalError, "删除活动消息群组失败"+err.Error())
	// }

	// 删除关联群聊
	err = svc.chatSvc.DeleteGroup(ctx, event.ChatGroupID, userID)
	if err != nil {
		return utils.NewBusinessError(utils.ErrCodeServerInternalError, "删除活动关联群聊失败"+err.Error())
	}

	// 清理Redis库存缓存
	if err := svc.stockSvc.Delete(ctx, eventID); err != nil {
		logrus.Warnf("删除活动库存缓存失败[eventID=%d]: %v", eventID, err)
	}

	// 更新活动缓存
	svc.eventCache.Delete(eventID)

	return nil
}

// ListEventRegisteredUser 获取活动报名用户列表
func (svc *EventServiceImpl) ListEventRegisteredUser(ctx context.Context, page, pageSize int, eventID int) ([]*dto.ListEventRegUserResponse, int64, error) {
	return svc.eventRepo.ListEventRegisteredUser(ctx, page, pageSize, eventID)
}
