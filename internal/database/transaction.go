package database

import (
	"event-platform/internal/utils"
	"fmt"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func WithTx(db *gorm.DB, fn func(tx *gorm.DB) error) (err error) {
	tx := db.Begin()
	if tx.Error != nil {
		return utils.NewSystemError(fmt.Errorf("开启事务失败: %w", tx.Error))
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			logrus.Errorf("事务执行异常，已回滚: %v", r)
			err = utils.NewSystemError(fmt.Errorf("事务执行异常: %v", r))
		}
	}()

	if err = fn(tx); err != nil {
		tx.Rollback()
		return
	}

	if err = tx.Commit().Error; err != nil {
		tx.Rollback()
		return utils.NewSystemError(fmt.Errorf("提交事务失败: %w", err))
	}
	return
}
