package repository

import (
	"gorm.io/gorm"

	"github.com/qs3c/anal_go_server/internal/model"
)

type JobRepository struct {
	db *gorm.DB
}

func NewJobRepository(db *gorm.DB) *JobRepository {
	return &JobRepository{db: db}
}

func (r *JobRepository) Create(job *model.AnalysisJob) error {
	return r.db.Create(job).Error
}

func (r *JobRepository) GetByID(id int64) (*model.AnalysisJob, error) {
	var job model.AnalysisJob
	err := r.db.Where("id = ?", id).First(&job).Error
	if err != nil {
		return nil, err
	}
	return &job, nil
}

func (r *JobRepository) GetByAnalysisID(analysisID int64) (*model.AnalysisJob, error) {
	var job model.AnalysisJob
	err := r.db.Where("analysis_id = ?", analysisID).Order("created_at DESC").First(&job).Error
	if err != nil {
		return nil, err
	}
	return &job, nil
}

func (r *JobRepository) Update(job *model.AnalysisJob) error {
	return r.db.Save(job).Error
}

func (r *JobRepository) UpdateStatus(id int64, status string) error {
	return r.db.Model(&model.AnalysisJob{}).Where("id = ?", id).Update("status", status).Error
}

func (r *JobRepository) UpdateStep(id int64, step string) error {
	return r.db.Model(&model.AnalysisJob{}).Where("id = ?", id).Update("current_step", step).Error
}

// GetPendingJobs 获取待处理的任务
func (r *JobRepository) GetPendingJobs(limit int) ([]*model.AnalysisJob, error) {
	var jobs []*model.AnalysisJob
	err := r.db.Where("status = ?", "queued").
		Order("created_at ASC").
		Limit(limit).
		Find(&jobs).Error
	return jobs, err
}

// CancelByAnalysisID 取消指定分析的任务
func (r *JobRepository) CancelByAnalysisID(analysisID int64) error {
	return r.db.Model(&model.AnalysisJob{}).
		Where("analysis_id = ? AND status IN ?", analysisID, []string{"queued", "processing"}).
		Update("status", "cancelled").Error
}
