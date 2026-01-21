package dto

// RegisterRequest 注册请求
type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8,max=32"`
}

// RegisterResponse 注册响应
type RegisterResponse struct {
	UserID int64 `json:"user_id"`
}

// LoginRequest 登录请求
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse 登录响应
type LoginResponse struct {
	Token string    `json:"token"`
	User  *UserInfo `json:"user"`
}

// VerifyEmailRequest 邮箱验证请求
type VerifyEmailRequest struct {
	Code string `json:"code" binding:"required"`
}

// UserInfo 用户信息（返回给前端）
type UserInfo struct {
	ID                int64      `json:"id"`
	Username          string     `json:"username"`
	Email             string     `json:"email,omitempty"`
	AvatarURL         string     `json:"avatar_url"`
	Bio               string     `json:"bio"`
	SubscriptionLevel string     `json:"subscription_level"`
	EmailVerified     bool       `json:"email_verified,omitempty"`
	QuotaInfo         *QuotaInfo `json:"quota_info,omitempty"`
	CreatedAt         string     `json:"created_at,omitempty"`
}

// QuotaInfo 配额信息
type QuotaInfo struct {
	DailyQuota     int    `json:"daily_quota"`
	QuotaUsedToday int    `json:"quota_used_today"`
	QuotaRemaining int    `json:"quota_remaining"`
	QuotaResetAt   string `json:"quota_reset_at,omitempty"`
}

// UpdateProfileRequest 更新用户信息请求
type UpdateProfileRequest struct {
	Username *string `json:"username,omitempty" binding:"omitempty,min=3,max=50"`
	Bio      *string `json:"bio,omitempty" binding:"omitempty,max=500"`
}
