package repository

import (
	"gorm.io/gorm"

	"github.com/qs3c/anal_go_server/internal/model"
)

type InteractionRepository struct {
	db *gorm.DB
}

func NewInteractionRepository(db *gorm.DB) *InteractionRepository {
	return &InteractionRepository{db: db}
}

// Create 创建互动记录
func (r *InteractionRepository) Create(interaction *model.Interaction) error {
	return r.db.Create(interaction).Error
}

// Delete 删除互动记录
func (r *InteractionRepository) Delete(userID, analysisID int64, interactionType string) error {
	return r.db.Where("user_id = ? AND analysis_id = ? AND type = ?", userID, analysisID, interactionType).
		Delete(&model.Interaction{}).Error
}

// Exists 检查互动是否存在
func (r *InteractionRepository) Exists(userID, analysisID int64, interactionType string) (bool, error) {
	var count int64
	err := r.db.Model(&model.Interaction{}).
		Where("user_id = ? AND analysis_id = ? AND type = ?", userID, analysisID, interactionType).
		Count(&count).Error
	return count > 0, err
}

// GetByUserAndAnalysis 获取用户对某分析的互动状态
func (r *InteractionRepository) GetByUserAndAnalysis(userID, analysisID int64) ([]*model.Interaction, error) {
	var interactions []*model.Interaction
	err := r.db.Where("user_id = ? AND analysis_id = ?", userID, analysisID).Find(&interactions).Error
	return interactions, err
}

// GetUserLikedAnalyses 获取用户点赞的分析列表
func (r *InteractionRepository) GetUserLikedAnalyses(userID int64, page, pageSize int) ([]int64, int64, error) {
	var total int64
	var ids []int64

	query := r.db.Model(&model.Interaction{}).Where("user_id = ? AND type = ?", userID, "like")

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Pluck("analysis_id", &ids).Error
	return ids, total, err
}

// GetUserBookmarkedAnalyses 获取用户收藏的分析列表
func (r *InteractionRepository) GetUserBookmarkedAnalyses(userID int64, page, pageSize int) ([]int64, int64, error) {
	var total int64
	var ids []int64

	query := r.db.Model(&model.Interaction{}).Where("user_id = ? AND type = ?", userID, "bookmark")

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Pluck("analysis_id", &ids).Error
	return ids, total, err
}
