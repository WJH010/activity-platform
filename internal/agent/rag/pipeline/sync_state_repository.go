// sync_state_repository.go 完整替换

package pipeline

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// SyncState RAG同步状态持久化模型
type SyncState struct {
	ID         int       `json:"id" gorm:"primaryKey;autoIncrement;column:id"`
	Name       string    `json:"name" gorm:"column:name;uniqueIndex"`
	LastSyncAt time.Time `json:"last_sync_at" gorm:"column:last_sync_at"`
	Status     string    `json:"status" gorm:"column:status"`
	Log        string    `json:"log" gorm:"column:log;type:text"`
	CreateTime time.Time `json:"create_time" gorm:"column:create_time;autoCreateTime"`
	UpdateTime time.Time `json:"update_time" gorm:"column:update_time;autoUpdateTime"`
}

// TableName 设置表名
func (SyncState) TableName() string {
	return "rag_sync_state"
}

// SyncStateRepository 同步状态数据访问
type SyncStateRepository struct {
	db *gorm.DB
}

// NewSyncStateRepository 创建同步状态Repository
func NewSyncStateRepository(db *gorm.DB) *SyncStateRepository {
	return &SyncStateRepository{db: db}
}

// Get 获取指定数据源的同步状态（取最新一条）
func (r *SyncStateRepository) Get(ctx context.Context, name string) (*SyncState, error) {
	var state SyncState
	err := r.db.WithContext(ctx).Where("name = ?", name).Order("id DESC").First(&state).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &state, nil
}

// Upsert 插入或更新同步状态
func (r *SyncStateRepository) Upsert(ctx context.Context, state *SyncState) error {
	var existing SyncState
	err := r.db.WithContext(ctx).Where("name = ?", state.Name).Order("id DESC").First(&existing).Error
	if err == gorm.ErrRecordNotFound {
		// 不存在，插入新记录
		return r.db.WithContext(ctx).Create(state).Error
	}
	if err != nil {
		return err
	}
	// 已存在，更新
	return r.db.WithContext(ctx).Model(&existing).Updates(map[string]interface{}{
		"last_sync_at": state.LastSyncAt,
		"status":       state.Status,
		"log":          state.Log,
		"update_time":  time.Now(),
	}).Error
}

// UpdateStatusOnly 只更新状态和日志，不更新 LastSyncAt
func (r *SyncStateRepository) UpdateStatusOnly(ctx context.Context, state *SyncState) error {
	var existing SyncState
	err := r.db.WithContext(ctx).Where("name = ?", state.Name).Order("id DESC").First(&existing).Error
	if err == gorm.ErrRecordNotFound {
		// 不存在，插入新记录（不含last_sync_at）
		return r.db.WithContext(ctx).Create(state).Error
	}
	if err != nil {
		return err
	}
	// 已存在，更新
	return r.db.WithContext(ctx).Model(&existing).Updates(map[string]interface{}{
		"status":      state.Status,
		"log":         state.Log,
		"update_time": time.Now(),
	}).Error
}
