package model

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

// StringArray 用于 JSON 数组字段
type StringArray []string

func (s StringArray) Value() (driver.Value, error) {
	if s == nil {
		return "[]", nil
	}
	return json.Marshal(s)
}

func (s *StringArray) Scan(value interface{}) error {
	if value == nil {
		*s = []string{}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, s)
}

type Analysis struct {
	ID               int64       `gorm:"primaryKey" json:"id"`
	UserID           int64       `gorm:"not null;index" json:"user_id"`
	Title            string      `gorm:"size:200;not null" json:"title"`
	Description      string      `gorm:"type:text" json:"description"`
	CreationType     string      `gorm:"size:20;not null" json:"creation_type"` // ai, manual
	RepoURL          string      `gorm:"size:500" json:"repo_url,omitempty"`
	StartStruct      string      `gorm:"size:100" json:"start_struct,omitempty"`
	AnalysisDepth    int         `json:"analysis_depth,omitempty"`
	ModelName        string      `gorm:"size:50" json:"model_name,omitempty"`
	SourceType       string      `gorm:"size:20;default:github"` // github 或 upload
	UploadID         string      `gorm:"size:64"`
	StartFile        string      `gorm:"size:500"`
	DiagramOSSURL    string      `gorm:"size:500" json:"diagram_oss_url,omitempty"`
	DiagramSize      int         `json:"diagram_size,omitempty"`
	Status           string      `gorm:"size:20;default:draft;index" json:"status"` // draft, pending, analyzing, completed, failed
	ErrorMessage     string      `gorm:"type:text" json:"error_message,omitempty"`
	StartedAt        *time.Time  `json:"started_at,omitempty"`
	CompletedAt      *time.Time  `json:"completed_at,omitempty"`
	IsPublic         bool        `gorm:"default:false;index" json:"is_public"`
	SharedAt         *time.Time  `gorm:"index" json:"shared_at,omitempty"`
	ShareTitle       string      `gorm:"size:200" json:"share_title,omitempty"`
	ShareDescription string      `gorm:"type:text" json:"share_description,omitempty"`
	Tags             StringArray `gorm:"type:json" json:"tags,omitempty"`
	ViewCount        int         `gorm:"default:0" json:"view_count"`
	LikeCount        int         `gorm:"default:0" json:"like_count"`
	CommentCount     int         `gorm:"default:0" json:"comment_count"`
	BookmarkCount    int         `gorm:"default:0" json:"bookmark_count"`
	CreatedAt        time.Time   `gorm:"index" json:"created_at"`
	UpdatedAt        time.Time   `json:"updated_at"`

	// 关联
	User *User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (Analysis) TableName() string {
	return "analyses"
}
