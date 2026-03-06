package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ryank90/lora-trainer-example/internal/domain"
	"github.com/ryank90/lora-trainer-example/internal/repository"
)

type JobRepo struct {
	pool *pgxpool.Pool
}

func NewJobRepo(pool *pgxpool.Pool) *JobRepo {
	return &JobRepo{pool: pool}
}

func (r *JobRepo) Create(ctx context.Context, job *domain.Job) error {
	reqJSON, err := job.RequestJSON()
	if err != nil {
		return fmt.Errorf("marshaling request: %w", err)
	}

	resultJSON, err := job.ResultJSON()
	if err != nil {
		return fmt.Errorf("marshaling result: %w", err)
	}

	_, err = r.pool.Exec(ctx, `
		INSERT INTO jobs (id, status, model_type, request, result, error, provider, gpu_type,
			idempotency_key, created_at, updated_at, provisioned_at, download_start_at,
			training_start_at, upload_start_at, completed_at, failed_at, cancelled_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)`,
		job.ID, job.Status, string(job.Request.ModelType), reqJSON, resultJSON, nilIfEmpty(job.Error),
		nilIfEmpty(job.Provider), nilIfEmpty(job.GPUType), nilIfEmpty(job.IdempotencyKey),
		job.CreatedAt, job.UpdatedAt, job.ProvisionedAt, job.DownloadStartAt,
		job.TrainingStartAt, job.UploadStartAt, job.CompletedAt, job.FailedAt, job.CancelledAt)
	if err != nil {
		return fmt.Errorf("inserting job: %w", err)
	}
	return nil
}

func (r *JobRepo) Get(ctx context.Context, id string) (*domain.Job, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, status, model_type, request, result, error, provider, gpu_type,
			idempotency_key, created_at, updated_at, provisioned_at, download_start_at,
			training_start_at, upload_start_at, completed_at, failed_at, cancelled_at
		FROM jobs WHERE id = $1`, id)

	return scanJob(row)
}

func (r *JobRepo) Update(ctx context.Context, job *domain.Job) error {
	reqJSON, err := job.RequestJSON()
	if err != nil {
		return fmt.Errorf("marshaling request: %w", err)
	}

	resultJSON, err := job.ResultJSON()
	if err != nil {
		return fmt.Errorf("marshaling result: %w", err)
	}

	_, err = r.pool.Exec(ctx, `
		UPDATE jobs SET status=$2, request=$3, result=$4, error=$5, provider=$6, gpu_type=$7,
			updated_at=$8, provisioned_at=$9, download_start_at=$10, training_start_at=$11,
			upload_start_at=$12, completed_at=$13, failed_at=$14, cancelled_at=$15
		WHERE id=$1`,
		job.ID, job.Status, reqJSON, resultJSON, nilIfEmpty(job.Error),
		nilIfEmpty(job.Provider), nilIfEmpty(job.GPUType),
		job.UpdatedAt, job.ProvisionedAt, job.DownloadStartAt,
		job.TrainingStartAt, job.UploadStartAt, job.CompletedAt, job.FailedAt, job.CancelledAt)
	if err != nil {
		return fmt.Errorf("updating job: %w", err)
	}
	return nil
}

func (r *JobRepo) UpdateStatus(ctx context.Context, id string, status domain.JobStatus) error {
	_, err := r.pool.Exec(ctx, `UPDATE jobs SET status=$2, updated_at=NOW() WHERE id=$1`, id, status)
	if err != nil {
		return fmt.Errorf("updating job status: %w", err)
	}
	return nil
}

func (r *JobRepo) List(ctx context.Context, filter repository.JobFilter) ([]*domain.Job, error) {
	query := `SELECT id, status, model_type, request, result, error, provider, gpu_type,
		idempotency_key, created_at, updated_at, provisioned_at, download_start_at,
		training_start_at, upload_start_at, completed_at, failed_at, cancelled_at
		FROM jobs WHERE 1=1`

	args := []any{}
	argIdx := 1

	if filter.Status != nil {
		query += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, string(*filter.Status))
		argIdx++
	}
	if filter.ModelType != nil {
		query += fmt.Sprintf(" AND model_type = $%d", argIdx)
		args = append(args, string(*filter.ModelType))
		argIdx++
	}

	query += " ORDER BY created_at DESC"

	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	query += fmt.Sprintf(" LIMIT $%d", argIdx)
	args = append(args, limit)
	argIdx++

	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argIdx)
		args = append(args, filter.Offset)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("listing jobs: %w", err)
	}
	defer rows.Close()

	var jobs []*domain.Job
	for rows.Next() {
		job, err := scanJobFromRows(rows)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}

	return jobs, rows.Err()
}

func (r *JobRepo) GetByIdempotencyKey(ctx context.Context, key string) (*domain.Job, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, status, model_type, request, result, error, provider, gpu_type,
			idempotency_key, created_at, updated_at, provisioned_at, download_start_at,
			training_start_at, upload_start_at, completed_at, failed_at, cancelled_at
		FROM jobs WHERE idempotency_key = $1`, key)

	job, err := scanJob(row)
	if err != nil {
		if errors.Is(err, domain.ErrJobNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return job, nil
}

func scanJob(row pgx.Row) (*domain.Job, error) {
	var job domain.Job
	var reqJSON, resultJSON []byte
	var errStr, provider, gpuType, idempotencyKey *string
	var modelType string

	err := row.Scan(
		&job.ID, &job.Status, &modelType, &reqJSON, &resultJSON, &errStr,
		&provider, &gpuType, &idempotencyKey,
		&job.CreatedAt, &job.UpdatedAt, &job.ProvisionedAt, &job.DownloadStartAt,
		&job.TrainingStartAt, &job.UploadStartAt, &job.CompletedAt, &job.FailedAt, &job.CancelledAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrJobNotFound
		}
		return nil, fmt.Errorf("scanning job: %w", err)
	}

	if err := json.Unmarshal(reqJSON, &job.Request); err != nil {
		return nil, fmt.Errorf("unmarshaling request: %w", err)
	}
	if resultJSON != nil {
		var result domain.TrainingResult
		if err := json.Unmarshal(resultJSON, &result); err != nil {
			return nil, fmt.Errorf("unmarshaling result: %w", err)
		}
		job.Result = &result
	}

	if errStr != nil {
		job.Error = *errStr
	}
	if provider != nil {
		job.Provider = *provider
	}
	if gpuType != nil {
		job.GPUType = *gpuType
	}
	if idempotencyKey != nil {
		job.IdempotencyKey = *idempotencyKey
	}

	return &job, nil
}

func scanJobFromRows(rows pgx.Rows) (*domain.Job, error) {
	var job domain.Job
	var reqJSON, resultJSON []byte
	var errStr, provider, gpuType, idempotencyKey *string
	var modelType string

	err := rows.Scan(
		&job.ID, &job.Status, &modelType, &reqJSON, &resultJSON, &errStr,
		&provider, &gpuType, &idempotencyKey,
		&job.CreatedAt, &job.UpdatedAt, &job.ProvisionedAt, &job.DownloadStartAt,
		&job.TrainingStartAt, &job.UploadStartAt, &job.CompletedAt, &job.FailedAt, &job.CancelledAt)
	if err != nil {
		return nil, fmt.Errorf("scanning job row: %w", err)
	}

	if err := json.Unmarshal(reqJSON, &job.Request); err != nil {
		return nil, fmt.Errorf("unmarshaling request: %w", err)
	}
	if resultJSON != nil {
		var result domain.TrainingResult
		if err := json.Unmarshal(resultJSON, &result); err != nil {
			return nil, fmt.Errorf("unmarshaling result: %w", err)
		}
		job.Result = &result
	}

	if errStr != nil {
		job.Error = *errStr
	}
	if provider != nil {
		job.Provider = *provider
	}
	if gpuType != nil {
		job.GPUType = *gpuType
	}
	if idempotencyKey != nil {
		job.IdempotencyKey = *idempotencyKey
	}

	return &job, nil
}

func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
