package repository

import (
	"time"

	"gorm.io/gorm"

	"github.com/qs3c/anal_go_server/internal/model"
)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(user *model.User) error {
	return r.db.Create(user).Error
}

func (r *UserRepository) GetByID(id int64) (*model.User, error) {
	var user model.User
	err := r.db.Where("id = ?", id).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) GetByEmail(email string) (*model.User, error) {
	var user model.User
	err := r.db.Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) GetByUsername(username string) (*model.User, error) {
	var user model.User
	err := r.db.Where("username = ?", username).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) GetByGithubID(githubID string) (*model.User, error) {
	var user model.User
	err := r.db.Where("github_id = ?", githubID).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) GetByVerificationCode(code string) (*model.User, error) {
	var user model.User
	err := r.db.Where("verification_code = ?", code).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) Update(user *model.User) error {
	return r.db.Save(user).Error
}

func (r *UserRepository) UpdateFields(id int64, fields map[string]interface{}) error {
	return r.db.Model(&model.User{}).Where("id = ?", id).Updates(fields).Error
}

func (r *UserRepository) IncrementQuotaUsed(id int64) error {
	return r.db.Model(&model.User{}).Where("id = ?", id).
		Update("quota_used_today", gorm.Expr("quota_used_today + 1")).Error
}

func (r *UserRepository) DecrementQuotaUsed(id int64) error {
	return r.db.Model(&model.User{}).Where("id = ?", id).
		Update("quota_used_today", gorm.Expr("GREATEST(quota_used_today - 1, 0)")).Error
}

func (r *UserRepository) ResetQuota(id int64, nextResetAt time.Time) error {
	return r.db.Model(&model.User{}).Where("id = ?", id).Updates(map[string]interface{}{
		"quota_used_today": 0,
		"quota_reset_at":   nextResetAt,
	}).Error
}

func (r *UserRepository) ResetAllQuotas(nextResetAt time.Time) error {
	return r.db.Model(&model.User{}).Where("1 = 1").Updates(map[string]interface{}{
		"quota_used_today": 0,
		"quota_reset_at":   nextResetAt,
	}).Error
}

func (r *UserRepository) ExistsByEmail(email string) (bool, error) {
	var count int64
	err := r.db.Model(&model.User{}).Where("email = ?", email).Count(&count).Error
	return count > 0, err
}

func (r *UserRepository) ExistsByUsername(username string) (bool, error) {
	var count int64
	err := r.db.Model(&model.User{}).Where("username = ?", username).Count(&count).Error
	return count > 0, err
}
