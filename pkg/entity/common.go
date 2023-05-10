package entity

import (
	"context"
	"time"

	"github.com/shiningrush/fastflow/store"
)

// BaseInfo
type BaseInfo struct {
	ID        string `yaml:"id" json:"id" bson:"_id" gorm:"primaryKey;type:VARCHAR(256);not null"`
	CreatedAt int64  `yaml:"createdAt" json:"createdAt" bson:"createdAt" gorm:"autoCreateTime;type:bigint(20) unsigned;not null;<-:create"`
	UpdatedAt int64  `yaml:"updatedAt" json:"updatedAt" bson:"updatedAt" gorm:"autoUpdateTime;type:bigint(20) unsigned;index;"`
}

// GetBaseInfo getter
func (b *BaseInfo) GetBaseInfo() *BaseInfo {
	return b
}

// Initial base info
func (b *BaseInfo) Initial() {
	if b.ID == "" {
		b.ID = store.NextStringID()
	}
	if b.CreatedAt == 0 {
		b.CreatedAt = time.Now().Unix()
	}
	b.UpdatedAt = time.Now().Unix()
}

// Update
func (b *BaseInfo) Update() {
	b.UpdatedAt = time.Now().Unix()
}

// BaseInfoGetter
type BaseInfoGetter interface {
	GetBaseInfo() *BaseInfo
}

type CtxKey string

const (
	CtxKeyRunningTaskIns CtxKey = "running-task"
)

func CtxWithRunningTaskIns(ctx context.Context, task *TaskInstance) context.Context {
	return context.WithValue(ctx, CtxKeyRunningTaskIns, task)
}

func CtxRunningTaskIns(ctx context.Context) (*TaskInstance, bool) {
	ins, ok := ctx.Value(CtxKeyRunningTaskIns).(*TaskInstance)
	return ins, ok
}
