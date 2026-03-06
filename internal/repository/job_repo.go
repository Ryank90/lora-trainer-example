package repository

import (
	"context"

	"github.com/ryank90/lora-trainer-example/internal/domain"
)

type JobFilter struct {
	Status    *domain.JobStatus
	ModelType *domain.ModelType
	Limit     int
	Offset    int
}

type JobRepository interface {
	Create(ctx context.Context, job *domain.Job) error
	Get(ctx context.Context, id string) (*domain.Job, error)
	Update(ctx context.Context, job *domain.Job) error
	UpdateStatus(ctx context.Context, id string, status domain.JobStatus) error
	List(ctx context.Context, filter JobFilter) ([]*domain.Job, error)
	GetByIdempotencyKey(ctx context.Context, key string) (*domain.Job, error)
}
