package repository

import (
	"gorm.io/gorm"

	"github.com/qs3c/anal_go_server/internal/model"
)

type CommentRepository struct {
	db *gorm.DB
}

func NewCommentRepository(db *gorm.DB) *CommentRepository {
	return &CommentRepository{db: db}
}

// Create 创建评论
func (r *CommentRepository) Create(comment *model.Comment) error {
	return r.db.Create(comment).Error
}

// GetByID 根据 ID 获取评论
func (r *CommentRepository) GetByID(id int64) (*model.Comment, error) {
	var comment model.Comment
	err := r.db.Where("id = ?", id).First(&comment).Error
	if err != nil {
		return nil, err
	}
	return &comment, nil
}

// GetByIDWithUser 获取评论及用户信息
func (r *CommentRepository) GetByIDWithUser(id int64) (*model.Comment, error) {
	var comment model.Comment
	err := r.db.Preload("User").Where("id = ?", id).First(&comment).Error
	if err != nil {
		return nil, err
	}
	return &comment, nil
}

// Delete 删除评论
func (r *CommentRepository) Delete(id int64) error {
	return r.db.Delete(&model.Comment{}, id).Error
}

// DeleteByParentID 删除子评论
func (r *CommentRepository) DeleteByParentID(parentID int64) (int64, error) {
	result := r.db.Where("parent_id = ?", parentID).Delete(&model.Comment{})
	return result.RowsAffected, result.Error
}

// ListByAnalysisID 获取分析的一级评论列表
func (r *CommentRepository) ListByAnalysisID(analysisID int64, page, pageSize int) ([]*model.Comment, int64, error) {
	var comments []*model.Comment
	var total int64

	query := r.db.Model(&model.Comment{}).
		Preload("User").
		Where("analysis_id = ? AND parent_id IS NULL", analysisID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&comments).Error
	if err != nil {
		return nil, 0, err
	}

	return comments, total, nil
}

// GetRepliesByParentIDs 批量获取回复
func (r *CommentRepository) GetRepliesByParentIDs(parentIDs []int64) ([]*model.Comment, error) {
	if len(parentIDs) == 0 {
		return nil, nil
	}

	var replies []*model.Comment
	err := r.db.Preload("User").
		Where("parent_id IN ?", parentIDs).
		Order("created_at ASC").
		Find(&replies).Error
	return replies, err
}

// CountByAnalysisID 获取分析的评论数
func (r *CommentRepository) CountByAnalysisID(analysisID int64) (int64, error) {
	var count int64
	err := r.db.Model(&model.Comment{}).Where("analysis_id = ?", analysisID).Count(&count).Error
	return count, err
}
