package database

import (
	"event-platform/internal/config"
	"fmt"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// 全局DB实例（初始化后复用，避免重复创建连接）
var db *gorm.DB

// NewDatabase 创建数据库连接
func NewDatabase(cfg config.DatabaseConfig) (*gorm.DB, error) {
	// 构建数据源名称 (DSN)
	var dsn string
	switch cfg.Driver {
	case "mysql":
		dsn = fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			cfg.Username,
			cfg.Password,
			cfg.Host,
			cfg.Port,
			cfg.DBName,
		)
	// 可在此扩展其他数据库驱动
	default:
		return nil, fmt.Errorf("不支持的数据库驱动: %s", cfg.Driver)
	}

	var err error
	// 打开数据库连接，通过 GORM 的Open方法创建数据库连接，并将结果保存到全局变量db中。
	db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("数据库连接失败: %v", err)
	}

	// 获取底层的*sql.DB对象，用于配置数据库连接池参数
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("获取底层sql.DB失败: %v", err)
	}

	// 设置连接池
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConnections)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConnections)
	sqlDB.SetConnMaxLifetime(cfg.ConnectionMaxLifetime * time.Second)

	// 自动迁移模型
	// GORM 的 AutoMigrate 会根据定义的模型自动创建或更新数据库表结构（生产环境通常会禁用，改为手动管理表结构）
	// if err := migrateModels(db); err != nil {
	//  return nil, err
	// }

	return db, nil
}

// migrateModels 自动迁移数据库模型（GORM 的 AutoMigrate 方法会根据数据模型自动创建或更新表结构）
// func migrateModels(db *gorm.DB) error {
//  // 添加需要迁移的模型
//  return db.AutoMigrate(&model.Example{})
// }

// GetDB 获取数据库连接实例
func GetDB() *gorm.DB {
	return db
}
