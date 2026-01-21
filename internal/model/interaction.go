package model

import (
	"time"
)

type Interaction struct {
	ID         int64     `gorm:"primaryKey" json:"id"`
	UserID     int64     `gorm:"not null;index" json:"user_id"`
	AnalysisID int64     `gorm:"not null;index" json:"analysis_id"`
	Type       string    `gorm:"size:20;not null" json:"type"` // like, bookmark
	CreatedAt  time.Time `json:"created_at"`
}

func (Interaction) TableName() string {
	return "interactions"
}
