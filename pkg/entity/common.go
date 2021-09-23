package entity

import (
	"github.com/shiningrush/fastflow/store"
	"time"
)

// BaseInfo
type BaseInfo struct {
	ID        string `yaml:"id" json:"id" bson:"_id"`
	CreatedAt int64  `yaml:"createdAt" json:"createdAt" bson:"createdAt"`
	UpdatedAt int64  `yaml:"updatedAt" json:"updatedAt" bson:"updatedAt"`
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
	b.CreatedAt = time.Now().Unix()
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
