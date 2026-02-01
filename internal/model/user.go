package model

import (
	"time"
)

type User struct {
	ID                    int64      `gorm:"primaryKey" json:"id"`
	Username              string     `gorm:"size:50;uniqueIndex;not null" json:"username"`
	Email                 *string    `gorm:"size:100;uniqueIndex" json:"email,omitempty"`
	PasswordHash          *string    `gorm:"size:255" json:"-"`
	AvatarURL             string     `gorm:"size:500" json:"avatar_url"`
	Bio                   string     `gorm:"type:text" json:"bio"`
	GithubID              *string    `gorm:"column:github_id;size:50;uniqueIndex" json:"-"`
	WechatOpenID          *string    `gorm:"column:wechat_openid;size:100;uniqueIndex" json:"-"`
	SubscriptionLevel     string     `gorm:"size:20;default:free" json:"subscription_level"`
	DailyQuota            int        `gorm:"default:5" json:"daily_quota"`
	QuotaUsedToday        int        `gorm:"default:0" json:"quota_used_today"`
	QuotaResetAt          *time.Time `json:"quota_reset_at,omitempty"`
	SubscriptionExpiresAt *time.Time `json:"subscription_expires_at,omitempty"`
	EmailVerified         bool       `gorm:"default:false" json:"email_verified"`
	VerificationCode      *string    `gorm:"size:100" json:"-"`
	VerificationExpiresAt *time.Time `json:"-"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
}

func (User) TableName() string {
	return "users"
}
