package model

import (
	"time"
)

type Comment struct {
	ID         int64     `gorm:"primaryKey" json:"id"`
	UserID     int64     `gorm:"not null;index" json:"user_id"`
	AnalysisID int64     `gorm:"not null;index" json:"analysis_id"`
	ParentID   *int64    `gorm:"index" json:"parent_id,omitempty"`
	Content    string    `gorm:"type:text;not null" json:"content"`
	CreatedAt  time.Time `gorm:"index" json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`

	// 关联
	User    *User      `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Replies []*Comment `gorm:"-" json:"replies,omitempty"`
}

func (Comment) TableName() string {
	return "comments"
}
