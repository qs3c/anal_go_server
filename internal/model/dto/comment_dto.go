package dto

// CreateCommentRequest 创建评论请求
type CreateCommentRequest struct {
	Content  string `json:"content" binding:"required,min=1,max=500"`
	ParentID *int64 `json:"parent_id,omitempty"`
}

// CommentItem 评论项
type CommentItem struct {
	ID        int64          `json:"id"`
	User      *CommentUser   `json:"user"`
	Content   string         `json:"content"`
	ParentID  *int64         `json:"parent_id"`
	Replies   []*CommentItem `json:"replies,omitempty"`
	CreatedAt string         `json:"created_at"`
}

// CommentUser 评论用户信息
type CommentUser struct {
	ID        int64  `json:"id"`
	Username  string `json:"username"`
	AvatarURL string `json:"avatar_url"`
}
