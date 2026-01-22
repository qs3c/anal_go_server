package service

import (
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/qs3c/anal_go_server/config"
	"github.com/qs3c/anal_go_server/internal/model"
	"github.com/qs3c/anal_go_server/internal/model/dto"
	"github.com/qs3c/anal_go_server/internal/repository"
)

var (
	ErrCommentNotFound     = errors.New("评论不存在")
	ErrCommentPermission   = errors.New("无权操作此评论")
	ErrParentNotFound      = errors.New("父评论不存在")
	ErrParentNotInAnalysis = errors.New("父评论不属于该分析")
)

type CommentService struct {
	commentRepo  *repository.CommentRepository
	analysisRepo *repository.AnalysisRepository
	userRepo     *repository.UserRepository
	cfg          *config.Config
}

func NewCommentService(
	commentRepo *repository.CommentRepository,
	analysisRepo *repository.AnalysisRepository,
	userRepo *repository.UserRepository,
	cfg *config.Config,
) *CommentService {
	return &CommentService{
		commentRepo:  commentRepo,
		analysisRepo: analysisRepo,
		userRepo:     userRepo,
		cfg:          cfg,
	}
}

// Create 创建评论
func (s *CommentService) Create(userID, analysisID int64, req *dto.CreateCommentRequest) (*dto.CommentItem, error) {
	// 验证分析存在且公开
	analysis, err := s.analysisRepo.GetByID(analysisID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAnalysisNotFound
		}
		return nil, err
	}

	if !analysis.IsPublic {
		return nil, ErrAnalysisNotPublic
	}

	// 如果是回复，验证父评论
	if req.ParentID != nil {
		parent, err := s.commentRepo.GetByID(*req.ParentID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, ErrParentNotFound
			}
			return nil, err
		}

		// 验证父评论属于同一分析
		if parent.AnalysisID != analysisID {
			return nil, ErrParentNotInAnalysis
		}

		// 只支持一级回复
		if parent.ParentID != nil {
			req.ParentID = parent.ParentID
		}
	}

	// 获取用户信息
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return nil, err
	}

	// 创建评论
	comment := &model.Comment{
		UserID:     userID,
		AnalysisID: analysisID,
		ParentID:   req.ParentID,
		Content:    req.Content,
	}

	if err := s.commentRepo.Create(comment); err != nil {
		return nil, err
	}

	// 增加评论数
	s.analysisRepo.IncrementCommentCount(analysisID, 1)

	return &dto.CommentItem{
		ID:       comment.ID,
		ParentID: comment.ParentID,
		Content:  comment.Content,
		User: &dto.CommentUser{
			ID:        user.ID,
			Username:  user.Username,
			AvatarURL: user.AvatarURL,
		},
		CreatedAt: comment.CreatedAt.Format(time.RFC3339),
	}, nil
}

// Delete 删除评论
func (s *CommentService) Delete(userID, commentID int64) error {
	comment, err := s.commentRepo.GetByID(commentID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrCommentNotFound
		}
		return err
	}

	// 验证权限
	if comment.UserID != userID {
		return ErrCommentPermission
	}

	// 删除子回复并计算删除数量
	deletedReplies, _ := s.commentRepo.DeleteByParentID(commentID)

	// 删除评论
	if err := s.commentRepo.Delete(commentID); err != nil {
		return err
	}

	// 减少评论数（包括子回复）
	totalDeleted := 1 + int(deletedReplies)
	s.analysisRepo.IncrementCommentCount(comment.AnalysisID, -totalDeleted)

	return nil
}

// ListByAnalysisID 获取分析的评论列表
func (s *CommentService) ListByAnalysisID(analysisID int64, page, pageSize int) ([]*dto.CommentItem, int64, error) {
	// 验证分析存在且公开
	analysis, err := s.analysisRepo.GetByID(analysisID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, 0, ErrAnalysisNotFound
		}
		return nil, 0, err
	}

	if !analysis.IsPublic {
		return nil, 0, ErrAnalysisNotPublic
	}

	// 获取一级评论
	comments, total, err := s.commentRepo.ListByAnalysisID(analysisID, page, pageSize)
	if err != nil {
		return nil, 0, err
	}

	if len(comments) == 0 {
		return []*dto.CommentItem{}, 0, nil
	}

	// 收集一级评论ID
	parentIDs := make([]int64, len(comments))
	for i, c := range comments {
		parentIDs[i] = c.ID
	}

	// 批量获取回复
	replies, _ := s.commentRepo.GetRepliesByParentIDs(parentIDs)

	// 构建回复映射
	repliesMap := make(map[int64][]*model.Comment)
	for _, r := range replies {
		if r.ParentID != nil {
			repliesMap[*r.ParentID] = append(repliesMap[*r.ParentID], r)
		}
	}

	// 组装结果
	items := make([]*dto.CommentItem, len(comments))
	for i, c := range comments {
		items[i] = s.buildCommentItem(c)

		// 添加回复
		if childReplies, ok := repliesMap[c.ID]; ok {
			items[i].Replies = make([]*dto.CommentItem, len(childReplies))
			for j, r := range childReplies {
				items[i].Replies[j] = s.buildCommentItem(r)
			}
		}
	}

	return items, total, nil
}

func (s *CommentService) buildCommentItem(c *model.Comment) *dto.CommentItem {
	item := &dto.CommentItem{
		ID:        c.ID,
		ParentID:  c.ParentID,
		Content:   c.Content,
		CreatedAt: c.CreatedAt.Format(time.RFC3339),
	}

	if c.User != nil {
		item.User = &dto.CommentUser{
			ID:        c.User.ID,
			Username:  c.User.Username,
			AvatarURL: c.User.AvatarURL,
		}
	}

	return item
}
