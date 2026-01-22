package dto

import "encoding/json"

// CreateAnalysisRequest 创建分析请求
type CreateAnalysisRequest struct {
	Title         string          `json:"title" binding:"required,max=200"`
	CreationType  string          `json:"creation_type" binding:"required,oneof=ai manual"`
	RepoURL       string          `json:"repo_url,omitempty" binding:"omitempty,url"`
	StartStruct   string          `json:"start_struct,omitempty" binding:"omitempty,max=100"`
	AnalysisDepth int             `json:"analysis_depth,omitempty" binding:"omitempty,min=1,max=10"`
	ModelName     string          `json:"model_name,omitempty" binding:"omitempty,max=50"`
	DiagramData   json.RawMessage `json:"diagram_data,omitempty"`
}

// CreateAnalysisResponse 创建分析响应
type CreateAnalysisResponse struct {
	AnalysisID int64 `json:"analysis_id"`
	JobID      int64 `json:"job_id,omitempty"`
}

// UpdateAnalysisRequest 更新分析请求
type UpdateAnalysisRequest struct {
	Title       *string         `json:"title,omitempty" binding:"omitempty,max=200"`
	Description *string         `json:"description,omitempty" binding:"omitempty,max=2000"`
	DiagramData json.RawMessage `json:"diagram_data,omitempty"`
}

// ShareAnalysisRequest 分享分析请求
type ShareAnalysisRequest struct {
	ShareTitle       string   `json:"share_title" binding:"required,max=200"`
	ShareDescription string   `json:"share_description,omitempty" binding:"omitempty,max=2000"`
	Tags             []string `json:"tags,omitempty" binding:"omitempty,max=5,dive,max=20"`
}

// AnalysisListItem 分析列表项
type AnalysisListItem struct {
	ID           int64    `json:"id"`
	Title        string   `json:"title"`
	CreationType string   `json:"creation_type"`
	Status       string   `json:"status"`
	IsPublic     bool     `json:"is_public"`
	ViewCount    int      `json:"view_count"`
	LikeCount    int      `json:"like_count"`
	CommentCount int      `json:"comment_count"`
	CreatedAt    string   `json:"created_at"`
	UpdatedAt    string   `json:"updated_at"`
	Tags         []string `json:"tags,omitempty"`
}

// AnalysisDetail 分析详情
type AnalysisDetail struct {
	ID               int64    `json:"id"`
	Title            string   `json:"title"`
	Description      string   `json:"description"`
	CreationType     string   `json:"creation_type"`
	RepoURL          string   `json:"repo_url,omitempty"`
	StartStruct      string   `json:"start_struct,omitempty"`
	AnalysisDepth    int      `json:"analysis_depth,omitempty"`
	ModelName        string   `json:"model_name,omitempty"`
	DiagramOSSURL    string   `json:"diagram_oss_url,omitempty"`
	DiagramSize      int      `json:"diagram_size,omitempty"`
	Status           string   `json:"status"`
	ErrorMessage     string   `json:"error_message,omitempty"`
	IsPublic         bool     `json:"is_public"`
	ShareTitle       string   `json:"share_title,omitempty"`
	ShareDescription string   `json:"share_description,omitempty"`
	Tags             []string `json:"tags,omitempty"`
	ViewCount        int      `json:"view_count"`
	LikeCount        int      `json:"like_count"`
	CommentCount     int      `json:"comment_count"`
	BookmarkCount    int      `json:"bookmark_count"`
	StartedAt        string   `json:"started_at,omitempty"`
	CompletedAt      string   `json:"completed_at,omitempty"`
	CreatedAt        string   `json:"created_at"`
	UpdatedAt        string   `json:"updated_at"`
}

// CommunityAnalysisItem 社区分析列表项
type CommunityAnalysisItem struct {
	ID               int64       `json:"id"`
	ShareTitle       string      `json:"share_title"`
	ShareDescription string      `json:"share_description"`
	Tags             []string    `json:"tags"`
	Author           *AuthorInfo `json:"author"`
	ViewCount        int         `json:"view_count"`
	LikeCount        int         `json:"like_count"`
	CommentCount     int         `json:"comment_count"`
	BookmarkCount    int         `json:"bookmark_count"`
	SharedAt         string      `json:"shared_at"`
}

// AuthorInfo 作者信息
type AuthorInfo struct {
	ID        int64  `json:"id"`
	Username  string `json:"username"`
	AvatarURL string `json:"avatar_url"`
	Bio       string `json:"bio,omitempty"`
}

// CommunityAnalysisDetail 社区分析详情
type CommunityAnalysisDetail struct {
	ID               int64            `json:"id"`
	ShareTitle       string           `json:"share_title"`
	ShareDescription string           `json:"share_description"`
	Tags             []string         `json:"tags"`
	Author           *AuthorInfo      `json:"author"`
	DiagramOSSURL    string           `json:"diagram_oss_url"`
	CreationType     string           `json:"creation_type"`
	RepoURL          string           `json:"repo_url,omitempty"`
	ViewCount        int              `json:"view_count"`
	LikeCount        int              `json:"like_count"`
	CommentCount     int              `json:"comment_count"`
	BookmarkCount    int              `json:"bookmark_count"`
	SharedAt         string           `json:"shared_at"`
	UserInteraction  *UserInteraction `json:"user_interaction,omitempty"`
}

// UserInteraction 用户互动状态
type UserInteraction struct {
	Liked      bool `json:"liked"`
	Bookmarked bool `json:"bookmarked"`
}

// JobStatusResponse 任务状态响应
type JobStatusResponse struct {
	JobID          int64  `json:"job_id"`
	AnalysisID     int64  `json:"analysis_id"`
	Status         string `json:"status"`
	CurrentStep    string `json:"current_step,omitempty"`
	ElapsedSeconds int    `json:"elapsed_seconds,omitempty"`
	ErrorMessage   string `json:"error_message,omitempty"`
	StartedAt      string `json:"started_at,omitempty"`
}
