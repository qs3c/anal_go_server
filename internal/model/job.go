package model

import (
	"time"
)

type AnalysisJob struct {
	ID             int64      `gorm:"primaryKey" json:"id"`
	AnalysisID     int64      `gorm:"not null;index" json:"analysis_id"`
	UserID         int64      `gorm:"not null;index" json:"user_id"`
	RepoURL        string     `gorm:"size:500;not null" json:"repo_url"`
	StartStruct    string     `gorm:"size:100;not null" json:"start_struct"`
	Depth          int        `gorm:"not null" json:"depth"`
	ModelName      string     `gorm:"size:50;not null" json:"model_name"`
	Status         string     `gorm:"size:20;default:queued;index" json:"status"` // queued, processing, completed, failed, cancelled
	CurrentStep    string     `gorm:"size:200" json:"current_step,omitempty"`
	ErrorMessage   string     `gorm:"type:text" json:"error_message,omitempty"`
	CreatedAt      time.Time  `gorm:"index" json:"created_at"`
	StartedAt      *time.Time `json:"started_at,omitempty"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
	ElapsedSeconds int        `json:"elapsed_seconds,omitempty"`
}

func (AnalysisJob) TableName() string {
	return "analysis_jobs"
}
