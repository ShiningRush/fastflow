package mysql

import (
	"time"
)

type Heartbeat struct {
	WorkerKey string    `gorm:"primaryKey;type:VARCHAR(256);not null"`
	CreatedAt time.Time `gorm:"autoCreateTime;type:timestamp;not null;<-:create"`
	UpdatedAt time.Time `gorm:"autoUpdateTime;type:timestamp;index;"`
}

type Election struct {
	ID        string    `gorm:"primaryKey;type:VARCHAR(256);not null"`
	WorkerKey string    `gorm:"type:VARCHAR(256);not null"`
	UpdatedAt time.Time `gorm:"autoUpdateTime;type:timestamp;index;"`
}
