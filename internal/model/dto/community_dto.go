package dto

// CommunityListRequest 社区列表请求参数
type CommunityListRequest struct {
	Page     int    `form:"page,default=1"`
	PageSize int    `form:"page_size,default=20"`
	Sort     string `form:"sort,default=latest"` // latest, hot
	Tags     string `form:"tags"`                // 逗号分隔
}

// LikeResponse 点赞响应
type LikeResponse struct {
	Liked     bool `json:"liked"`
	LikeCount int  `json:"like_count"`
}

// BookmarkResponse 收藏响应
type BookmarkResponse struct {
	Bookmarked    bool `json:"bookmarked"`
	BookmarkCount int  `json:"bookmark_count"`
}
