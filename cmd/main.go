package main

import (
	"activity-platform/internal/config"
	"activity-platform/internal/database"
	filerepo "activity-platform/internal/file/repository"
	"activity-platform/internal/middleware"
	"activity-platform/internal/redis"
	"activity-platform/internal/routes"
	"activity-platform/internal/utils"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/sirupsen/logrus"
)

func main() {
	// 1.初始化日志记录器
	utils.InitLogger()

	// 2.加载配置
	cfg, err := config.LoadConfig("../config.yaml")
	if err != nil {
		logrus.Fatalf("服务器启动失败: %v", err)
	}

	// 3.初始化数据库
	db, err := database.NewDatabase(cfg.Database)
	if err != nil {
		log.Fatalf("初始化数据库失败: %v", err)
	}

	sqlDB, _ := db.DB() // 获取底层的 SQL 数据库连接
	defer sqlDB.Close() // 确保程序退出时关闭数据库连接，释放资源

	// 初始化Redis
	rdb, err := redis.NewRedisClient(cfg.Redis)
	if err != nil {
		logrus.Warnf("Redis初始化失败（Token黑名单不可用）: %v", err)
		// Redis不可用不阻塞启动，黑名单功能降级为不可用
	}
	defer func() {
		if rdb != nil {
			rdb.Close()
		}
	}()

	// 4.创建MinIO存储实例
	minioRepo, err := filerepo.NewMinIORepository(
		cfg.MinIO.Endpoint,
		cfg.MinIO.AccessKeyID,
		cfg.MinIO.SecretAccessKey,
		cfg.MinIO.UseSSL,
		cfg.MinIO.BucketName,
	)
	if err != nil {
		logrus.Panic("创建MinIO存储实例失败: ", err)
	}

	// 6.设置Gin模式
	if cfg.App.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// 7. 创建默认的Gin引擎，但不使用默认中间件
	router := gin.New()

	// 替换Gin的默认验证器为自定义验证器
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		// 在现有验证器上注册自定义规则
		utils.RegisterCustomValidators(v)
	}

	// 7.注册中间件
	router.Use(middleware.Logger())
	router.Use(middleware.Recovery())
	router.Use(middleware.RequestIdMiddleware())

	// 8.初始化依赖及注册路由
	routes.SetupRoutes(cfg, router, minioRepo)

	// pprof.Register(router) // 开启 pprof 后门

	PORT := cfg.App.Port
	logrus.Infof("服务器运行在端口 %d", PORT)
	if err := http.ListenAndServe(":"+strconv.Itoa(PORT), router); err != nil {
		logrus.Fatalf("服务器启动失败: %v", err)
	}
}
