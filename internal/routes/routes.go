package routes

import (
	"context"
	"event-platform/internal/agent/rag/embedding"
	"event-platform/internal/agent/rag/milvus"
	"event-platform/internal/agent/rag/pipeline"
	"event-platform/internal/agent/rag/search"
	"event-platform/internal/agent/skill"
	"event-platform/internal/agent/skill/builtin"
	"event-platform/internal/cache"
	"event-platform/internal/chat"
	"event-platform/internal/config"
	"event-platform/internal/database"
	"event-platform/internal/middleware"
	"event-platform/internal/utils"
	"time"

	articlectr "event-platform/internal/article/controller"
	articlerepo "event-platform/internal/article/repository"
	articlesvc "event-platform/internal/article/service"

	userctr "event-platform/internal/user/controller"
	userrepo "event-platform/internal/user/repository"
	usersvc "event-platform/internal/user/service"

	filectr "event-platform/internal/file/controller"
	filerepo "event-platform/internal/file/repository"
	filesvc "event-platform/internal/file/service"

	msgctr "event-platform/internal/message/controller"
	msgrepo "event-platform/internal/message/repository"
	msgsvc "event-platform/internal/message/service"

	eventctr "event-platform/internal/event/controller"
	eventmodel "event-platform/internal/event/model"
	eventrepo "event-platform/internal/event/repository"
	eventsvc "event-platform/internal/event/service"
	eventstock "event-platform/internal/event/stock"

	chatctr "event-platform/internal/chat/controller"
	chatrepo "event-platform/internal/chat/repository"
	chatsvc "event-platform/internal/chat/service"

	agentctr "event-platform/internal/agent/controller"
	agentrepo "event-platform/internal/agent/repository"
	agentsvc "event-platform/internal/agent/service"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// SetupRoutes 注册中间件和路由
func SetupRoutes(cfg *config.Config, router *gin.Engine, minioRepo filerepo.MinIORepository) {
	db := database.GetDB()

	// 初始化依赖
	// 初始化仓库
	articleRepo := articlerepo.NewArticleRepository(db)
	fieldTypeRepo := articlerepo.NewFieldTypeRepository(db)
	//noticeRepo := noticerepo.NewNoticeRepository(db)
	fileRepo := filerepo.NewFileRepository(db)
	userRepo := userrepo.NewUserRepository(db)
	industryRepo := userrepo.NewIndustryRepository(db)
	msgRepo := msgrepo.NewMessageRepository(db)
	eventRepo := eventrepo.NewEventRepository(db)
	eventUserInfoRepo := eventrepo.NewEventUserInfoRepository(db)
	msgGroupRepo := msgrepo.NewMsgGroupRepository(db, msgRepo)
	userRoleRepo := userrepo.NewUserRoleRepository(db)
	chatRepo := chatrepo.NewChatRepository(db)
	chatGroupRepo := chatrepo.NewChatGroupRepository(db)

	// 初始化服务
	articleService := articlesvc.NewArticleService(articleRepo, fileRepo)
	fieldService := articlesvc.NewFieldTypeService(fieldTypeRepo)
	//noticeService := noticesvc.NewNoticeService(noticeRepo)
	fileService := filesvc.NewFileService(minioRepo, fileRepo)
	msgService := msgsvc.NewMessageService(msgRepo, msgGroupRepo)
	msgGroupService := msgsvc.NewMsgGroupService(msgGroupRepo, msgRepo)
	userService := usersvc.NewUserService(userRepo, msgGroupService, cfg)
	industryService := usersvc.NewIndustryService(industryRepo)
	chatGroupService := chatsvc.NewChatGroupService(chatGroupRepo)
	eventService := eventsvc.NewEventService(eventRepo, eventUserInfoRepo, userRepo, fileRepo, chatGroupService, eventstock.NewStockService(), cache.New[int, *eventmodel.Event](3*time.Second))
	eventUserInfoService := eventsvc.NewEventUserInfoService(eventUserInfoRepo)
	userRoleService := usersvc.NewUserRoleService(userRoleRepo)
	chatService := chatsvc.NewChatService(chatRepo, chatGroupRepo, userRepo)

	// 创建 Hub 实例
	hub := chat.NewHub(chatService)
	// 在一个新的 goroutine 中运行 Hub，使其不阻塞主线程
	go hub.Run()

	// 初始化控制器
	articleController := articlectr.NewArticleController(articleService)
	fieldTypeController := articlectr.NewFieldTypeController(fieldService)
	//noticeController := noticectr.NewNoticeController(noticeService)
	fileController := filectr.NewFileController(fileService)
	userController := userctr.NewUserController(userService)
	industryController := userctr.NewIndustryController(industryService)
	msgController := msgctr.NewMessageController(msgService)
	eventController := eventctr.NewEventController(eventService)
	eventUserInfoController := eventctr.NewEventUserInfoController(eventUserInfoService)
	msgGroupController := msgctr.NewMsgGroupController(msgGroupService)
	userRoleController := userctr.NewUserRoleController(userRoleService)
	chatController := chatctr.NewChatController(chatService, chatGroupRepo, hub)
	chatGroupController := chatctr.NewChatGroupController(chatGroupService)

	// API分组
	api := router.Group("/api")
	{
		// 文章相关路由
		articles := api.Group("/articles")
		{
			// 公开接口 - 无需认证
			// 分页查询文章
			articles.GET("", articleController.ListArticle)
			// 查询文章详情
			articles.GET("/:id", articleController.GetArticleContent)
			// 需要认证的用户接口
			authArticles := articles.Group("")
			authArticles.Use(middleware.AuthMiddleware(cfg))
			{
				// 管理员接口 - 在认证基础上增加角色校验
				adminArticles := authArticles.Group("")
				adminArticles.Use(middleware.RoleMiddleware(utils.RoleAdmin))
				{
					// 创建文章
					adminArticles.POST("", articleController.CreateArticle)
					// 更新文章
					adminArticles.PUT("/:id", articleController.UpdateArticle)
					// 删除文章
					adminArticles.DELETE("/:id", articleController.DeleteArticle)
				}
			}
		}
		// 领域类型相关路由
		fieldTypes := api.Group("/field-types")
		{
			// 查询领域类型列表（不分页）
			fieldTypes.GET("", fieldTypeController.GetFieldType)
			authFieldTypes := fieldTypes.Group("")
			authFieldTypes.Use(middleware.AuthMiddleware(cfg))
			{
				// 管理员接口 - 在认证基础上增加角色校验
				adminFieldTypes := authFieldTypes.Group("")
				adminFieldTypes.Use(middleware.RoleMiddleware(utils.RoleAdmin))
				{
					// 创建领域类型
					adminFieldTypes.POST("", fieldTypeController.CreateFieldType)
					// 更新领域类型
					adminFieldTypes.PUT("/:id", fieldTypeController.UpdateFieldType)
					// 删除领域类型
					adminFieldTypes.DELETE("/:id", fieldTypeController.UpdateFieldType)
				}
			}
		}
		// 公告相关路由
		// notice := api.Group("/notice")
		// {
		// 	notice.GET("", noticeController.ListNotice)
		// 	notice.GET("/:id", noticeController.GetNoticeContent)
		// }
		// 用户相关路由
		users := api.Group("/users")
		{
			// 公开接口 - 无需认证
			// 登录
			users.POST("/login", userController.Login)
			// 后台系统登录
			users.POST("/bg-login", userController.BgLogin)
			// 刷新token接口
			users.POST("/refresh-token", userController.BgRefreshToken)
			// 需要认证的用户接口
			authUsers := users.Group("")
			authUsers.Use(middleware.AuthMiddleware(cfg))
			{
				// 获取当前用户信息
				authUsers.GET("/me", userController.GetUserInfo)
				// 更新当前用户信息
				authUsers.PUT("/me", userController.UpdateUserInfo)
				// 退出登录
				authUsers.POST("/logout", userController.Logout)
				// 管理员接口 - 在认证基础上增加角色校验
				adminUsers := authUsers.Group("")
				adminUsers.Use(middleware.RoleMiddleware(utils.RoleAdmin))
				{
					// 分页查询系统用户列表
					adminUsers.GET("", userController.ListAllUsers)
				}
				// 超级管理员接口
				superAdmin := authUsers.Group("")
				superAdmin.Use(middleware.RoleMiddleware(utils.RoleSuperAdmin))
				{
					// 新增管理员接口
					superAdmin.POST("", userController.CreateAdminUser)
					// 禁用/启用管理员接口
					superAdmin.PATCH("/:id/status", userController.UpdateAdminStatus)
					// 更新管理员信息
					superAdmin.PUT("/:id", userController.UpdateAdminUser)
				}
			}
		}
		// 行业路由
		industries := api.Group("/industries")
		{
			// 公开接口 - 无需认证
			// 获取行业列表（不分页）
			industries.GET("", industryController.ListIndustries)
			// 需要认证的用户接口
			authIndustries := industries.Group("")
			authIndustries.Use(middleware.AuthMiddleware(cfg))
			{
				// 管理员接口 - 在认证基础上增加角色校验
				adminIndustries := authIndustries.Group("")
				adminIndustries.Use(middleware.RoleMiddleware(utils.RoleAdmin))
				{
					// 创建行业
					adminIndustries.POST("", industryController.CreateIndustry)
					// 更新行业
					adminIndustries.PUT("/:id", industryController.UpdateIndustry)
				}
			}
		}
		// 用户角色路由
		userRoles := api.Group("/user-roles")
		{
			// 获取用户角色列表（不分页）
			userRoles.GET("", middleware.RoleMiddleware(utils.RoleAdmin), userRoleController.List)
		}
		// 文件相关路由
		files := api.Group("/files")
		files.Use(middleware.AuthMiddleware(cfg))
		{
			// 上传文件
			files.POST("", fileController.UploadFile)
			// 删除文件
			files.DELETE("/:id", fileController.DeleteImage)
		}
		// 消息相关路由
		messages := api.Group("/messages")
		messages.Use(middleware.AuthMiddleware(cfg))
		{
			// 获取消息详情
			messages.GET("/:id", msgController.GetMessageContent)
			// 查询是否有未读消息
			messages.GET("/unread", msgController.HasUnreadMessages)
			// 标记所有消息为已读
			messages.PATCH("/read", msgController.MarkAllMessagesAsRead)
			// 分页查询用户消息群组列表
			messages.GET("/groups", msgController.ListUserMessageGroups)
			// 分页查询组内消息列表
			messages.GET("/groups/:id/messages", msgController.ListMsgByGroups)
			// 消息群组管理，仅管理员可操作
			adminMessages := messages.Group("")
			adminMessages.Use(middleware.RoleMiddleware(utils.RoleAdmin))
			{
				// 根据消息群组ID查询消息列表
				adminMessages.GET("/groups/:id/all", msgController.ListMessagesByGroupID)
				// 分页获取消息群组列表
				adminMessages.GET("/groups", msgGroupController.ListMsgGroups)
				// 分页获取指定群组内用户列表
				adminMessages.GET("/groups/:id/users", msgGroupController.ListGroupsUsers)
				// 分页获取不在指定组内的用户列表
				adminMessages.GET("/groups/:id/users/not-in", msgGroupController.ListNotInGroupUsers)
				// 根据id获取指定消息群组信息
				adminMessages.GET("/groups/:id", msgGroupController.GetMsgGroupByID)
				// 创建消息群组
				adminMessages.POST("/groups", msgGroupController.CreateMsgGroup)
				// 将用户添加到指定群组
				adminMessages.POST("/groups/:id/members", msgGroupController.AddUserToGroup)
				// 向指定群组发送消息
				adminMessages.POST("/groups/:id/messages", msgController.SendMessage)
				// 更新指定群组信息
				adminMessages.PUT("/groups/:id", msgGroupController.UpdateMsgGroup)
				// 撤销指定消息
				adminMessages.DELETE("/:id", msgController.RevokeGroupMessage)
				// 从指定群组移除用户
				adminMessages.DELETE("/groups/:id/members", msgGroupController.DeleteUserFromGroup)
				// 删除群组
				adminMessages.DELETE("/groups/:id", msgGroupController.DeleteMsgGroup)
			}
		}
		// 活动相关路由
		events := api.Group("/events")
		{
			// 公开接口 - 无需认证
			// 分页查询活动列表
			events.GET("", eventController.ListEvent)
			// 获取指定活动详情
			events.GET("/:id", eventController.GetEventDetail)

			// 需要认证的用户接口
			authEvents := events.Group("")
			authEvents.Use(middleware.AuthMiddleware(cfg))
			{
				// 报名活动
				authEvents.POST("/:id/registrations", eventController.RegistrationEvent)
				// 查询当前用户是否报名指定活动
				authEvents.GET("/:id/registration", eventController.IsUserRegistered)
				// 取消报名活动
				authEvents.DELETE("/:id/registrations", eventController.CancelRegistrationEvent)
				// 获取当前用户已报名的活动列表
				authEvents.GET("/registered", eventController.ListUserRegisteredEvents)

				// 管理员接口 - 在认证基础上增加角色校验
				adminEvents := authEvents.Group("")
				adminEvents.Use(middleware.RoleMiddleware(utils.RoleAdmin))
				{
					// 创建活动
					adminEvents.POST("", eventController.CreateEvent)
					// 更新活动
					adminEvents.PUT("/:id", eventController.UpdateEvent)
					// 删除活动
					adminEvents.DELETE("/:id", eventController.DeleteEvent)
					// 分页查询报名指定活动的用户列表
					adminEvents.GET("/:id/registrations/users", eventController.ListEventRegisteredUsers)
					// 查询用户信息字段列表
					adminEvents.GET("/user-info-fields", eventUserInfoController.List)
					// 创建用户信息字段
					adminEvents.POST("/user-info-fields", eventUserInfoController.Create)
					// 更新用户信息字段
					adminEvents.PUT("/user-info-fields/:id", eventUserInfoController.Update)
					// 更新用户信息字段状态
					adminEvents.PATCH("/user-info-fields/:id/status", eventUserInfoController.UpdateStatus)
				}
			}
		}
		// 聊天相关路由
		chat := api.Group("/chat")
		{
			// WebSocket 连接端点需要认证
			authChat := chat.Group("")
			authChat.Use(middleware.AuthMiddleware(cfg))
			{
				// WebSocket连接
				authChat.GET("/ws/:groupId", chatController.ServeWs)
				// 个人通知频道
				authChat.GET("/ws/notifications", chatController.ServeNotificationWs)
				// 群组管理
				authChat.POST("/groups", chatGroupController.CreateGroup)
				authChat.GET("/groups", chatGroupController.ListUserGroups)
				authChat.GET("/groups/:groupId/members", chatGroupController.ListGroupMembers)
				authChat.GET("/groups/:groupId/not-in-members", chatGroupController.GetUsersNotInGroup)
				authChat.GET("/groups/:groupId/messages", chatGroupController.ListGroupMessages)
				authChat.POST("/groups/:groupId/members", chatGroupController.AddMembers)
				authChat.DELETE("/groups/:groupId/members", chatGroupController.RemoveMembers)
				authChat.GET("/unread", chatGroupController.HasUnreadMessages)
				authChat.DELETE("/groups/:groupId", chatGroupController.DeleteGroup)
				// 管理员接口 - 查询所有群组
				adminChat := authChat.Group("")
				adminChat.Use(middleware.RoleMiddleware(utils.RoleAdmin))
				{
					adminChat.GET("/groups/all", chatGroupController.ListAllGroups)
				}
			}
		}
		// Agent 智能助手相关路由
		agent := api.Group("/agent")
		{
			// 初始化RAG语义检索（可选，Milvus不可用则跳过）
			var searchSvc *search.SearchService
			var syncController *pipeline.SyncController
			milvusCli, milvusErr := milvus.NewClient(cfg.RAG.Milvus)
			if milvusErr == nil {
				// 初始化Collection
				if initErr := milvus.InitCollections(context.Background(), milvusCli); initErr != nil {
					logrus.Warnf("Milvus Collection初始化失败（语义检索不可用）: %v", initErr)
					milvusCli = nil
				}
			} else {
				logrus.Warnf("Milvus连接失败（语义检索不可用）: %v", milvusErr)
			}

			if milvusCli != nil {
				milvusRepo := milvus.NewRepository(milvusCli)
				embeddingSvc := embedding.NewBGEML3Service(cfg.RAG.Embedding)
				searchSvc = search.NewSearchService(milvusRepo, embeddingSvc)

				// 数据同步服务
				textProcessor := pipeline.NewTextProcessor()
				syncSvc := pipeline.NewSyncService(articleRepo, eventRepo, milvusRepo, embeddingSvc, textProcessor, db)

				// 同步管理控制器
				syncController = pipeline.NewSyncController(syncSvc)

				// 启动定时增量同步（内部自动判断是否需要全量同步）
				go syncSvc.StartIncrementalSync(context.Background(), cfg.RAG.Sync.IncrementInterval)
			}
			// --- 初始化Agent模块依赖 ---
			agentSessionRepo := agentrepo.NewSessionRepository(db)
			agentMessageRepo := agentrepo.NewMessageRepository(db)
			agentSkillRepo := agentrepo.NewSkillRepository(db)
			agentLLMConfigRepo := agentrepo.NewLLMConfigRepository(db)
			agentUserLLMPrefRepo := agentrepo.NewUserLLMPrefRepository(db)

			// 初始化Skill Registry并注册内置Skills
			skillRegistry := skill.NewRegistry()
			builtin.RegisterBuiltinSkills(skillRegistry, builtin.Dependencies{
				EventSvc:   eventService,
				ArticleSvc: articleService,
				UserSvc:    userService,
				SearchSvc:  searchSvc,
			})

			// 初始化Agent服务
			agentSessionSvc := agentsvc.NewSessionService(agentSessionRepo, agentMessageRepo)
			agentLLMConfigSvc := agentsvc.NewLLMConfigService(agentLLMConfigRepo, agentUserLLMPrefRepo)
			agentSkillSvc := agentsvc.NewSkillService(agentSkillRepo)
			agentChatSvc := agentsvc.NewAgentChatService(
				agentSessionSvc,
				agentLLMConfigSvc,
				agentLLMConfigRepo,
				agentMessageRepo,
				agentSkillSvc,
				skillRegistry,
				cfg,
			)

			// 初始化Agent控制器
			agentChatController := agentctr.NewAgentChatController(agentChatSvc)
			agentSessionController := agentctr.NewAgentSessionController(agentSessionSvc)
			agentSkillController := agentctr.NewAgentSkillController(agentSkillSvc)
			agentLLMConfigController := agentctr.NewAgentLLMConfigController(agentLLMConfigSvc)

			// 公开接口（部分）
			// Provider信息（无需认证，供前端展示配置选项）
			agent.GET("/llm-configs/providers", agentLLMConfigController.GetProviderInfo)

			// 需要认证的用户接口
			authAgent := agent.Group("")
			authAgent.Use(middleware.AuthMiddleware(cfg))
			{
				// 对话相关
				authAgent.POST("/chat", agentChatController.ChatStream)
				authAgent.GET("/sessions", agentSessionController.ListSessions)
				authAgent.GET("/sessions/:id/messages", agentChatController.GetSessionMessages)
				authAgent.DELETE("/sessions/:id", agentSessionController.DeleteSession)

				// LLM配置 - 用户可设置自己的偏好
				authAgent.GET("/llm-configs", agentLLMConfigController.ListConfigs)
				authAgent.PUT("/llm-configs/user-preference", agentLLMConfigController.SetUserLLM)

				// 管理员接口 - LLM配置管理
				adminAgent := authAgent.Group("")
				adminAgent.Use(middleware.RoleMiddleware(utils.RoleAdmin))
				{
					// LLM配置CRUD
					adminAgent.POST("/llm-configs", agentLLMConfigController.CreateConfig)
					adminAgent.GET("/llm-configs/:id", agentLLMConfigController.GetConfig)
					adminAgent.PUT("/llm-configs/:id", agentLLMConfigController.UpdateConfig)
					adminAgent.DELETE("/llm-configs/:id", agentLLMConfigController.DeleteConfig)

					// Skill管理
					adminAgent.GET("/skills", agentSkillController.ListSkills)
					adminAgent.GET("/skills/:id", agentSkillController.GetSkill)
					adminAgent.POST("/skills", agentSkillController.CreateSkill)
					adminAgent.PUT("/skills/:id", agentSkillController.UpdateSkill)
					adminAgent.DELETE("/skills/:id", agentSkillController.DeleteSkill)
					adminAgent.PUT("/skills/:id/toggle", agentSkillController.ToggleSkill)

					// RAG同步管理
					adminAgent.POST("/rag/sync/full", syncController.FullSync)
					adminAgent.GET("/rag/sync/status", syncController.GetSyncStatus)
				}
			}
		}
	}
}
