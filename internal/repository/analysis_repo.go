package repository

import (
	"gorm.io/gorm"

	"github.com/qs3c/anal_go_server/internal/model"
)

type AnalysisRepository struct {
	db *gorm.DB
}

func NewAnalysisRepository(db *gorm.DB) *AnalysisRepository {
	return &AnalysisRepository{db: db}
}

func (r *AnalysisRepository) Create(analysis *model.Analysis) error {
	return r.db.Create(analysis).Error
}

func (r *AnalysisRepository) GetByID(id int64) (*model.Analysis, error) {
	var analysis model.Analysis
	err := r.db.Where("id = ?", id).First(&analysis).Error
	if err != nil {
		return nil, err
	}
	return &analysis, nil
}

func (r *AnalysisRepository) GetByIDWithUser(id int64) (*model.Analysis, error) {
	var analysis model.Analysis
	err := r.db.Preload("User").Where("id = ?", id).First(&analysis).Error
	if err != nil {
		return nil, err
	}
	return &analysis, nil
}

func (r *AnalysisRepository) Update(analysis *model.Analysis) error {
	return r.db.Save(analysis).Error
}

func (r *AnalysisRepository) UpdateFields(id int64, fields map[string]interface{}) error {
	return r.db.Model(&model.Analysis{}).Where("id = ?", id).Updates(fields).Error
}

func (r *AnalysisRepository) UpdateStatus(id int64, status string) error {
	return r.db.Model(&model.Analysis{}).Where("id = ?", id).Update("status", status).Error
}

func (r *AnalysisRepository) Delete(id int64) error {
	return r.db.Delete(&model.Analysis{}, id).Error
}

// ListByUserID 获取用户的分析列表
func (r *AnalysisRepository) ListByUserID(userID int64, page, pageSize int, search, status string) ([]*model.Analysis, int64, error) {
	var analyses []*model.Analysis
	var total int64

	query := r.db.Model(&model.Analysis{}).Where("user_id = ?", userID)

	if search != "" {
		query = query.Where("title LIKE ?", "%"+search+"%")
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&analyses).Error; err != nil {
		return nil, 0, err
	}

	return analyses, total, nil
}

// ListPublic 获取公开的分析列表
func (r *AnalysisRepository) ListPublic(page, pageSize int, sortBy string, tags []string) ([]*model.Analysis, int64, error) {
	var analyses []*model.Analysis
	var total int64

	query := r.db.Model(&model.Analysis{}).Preload("User").Where("is_public = ?", true)

	// TODO: 标签过滤

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 排序
	switch sortBy {
	case "hot":
		query = query.Order("(like_count * 3 + comment_count * 2 + view_count) DESC")
	default: // latest
		query = query.Order("shared_at DESC")
	}

	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Find(&analyses).Error; err != nil {
		return nil, 0, err
	}

	return analyses, total, nil
}

// IncrementViewCount 增加浏览数
func (r *AnalysisRepository) IncrementViewCount(id int64) error {
	return r.db.Model(&model.Analysis{}).Where("id = ?", id).
		Update("view_count", gorm.Expr("view_count + 1")).Error
}

// IncrementLikeCount 增加点赞数
func (r *AnalysisRepository) IncrementLikeCount(id int64, delta int) error {
	return r.db.Model(&model.Analysis{}).Where("id = ?", id).
		Update("like_count", gorm.Expr("like_count + ?", delta)).Error
}

// IncrementBookmarkCount 增加收藏数
func (r *AnalysisRepository) IncrementBookmarkCount(id int64, delta int) error {
	return r.db.Model(&model.Analysis{}).Where("id = ?", id).
		Update("bookmark_count", gorm.Expr("bookmark_count + ?", delta)).Error
}

// IncrementCommentCount 增加评论数
func (r *AnalysisRepository) IncrementCommentCount(id int64, delta int) error {
	return r.db.Model(&model.Analysis{}).Where("id = ?", id).
		Update("comment_count", gorm.Expr("comment_count + ?", delta)).Error
}
