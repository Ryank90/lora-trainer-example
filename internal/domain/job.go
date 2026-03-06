package domain

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type JobStatus string

const (
	JobStatusPending      JobStatus = "pending"
	JobStatusProvisioning JobStatus = "provisioning"
	JobStatusDownloading  JobStatus = "downloading"
	JobStatusTraining     JobStatus = "training"
	JobStatusUploading    JobStatus = "uploading"
	JobStatusCompleted    JobStatus = "completed"
	JobStatusFailed       JobStatus = "failed"
	JobStatusCancelled    JobStatus = "cancelled"
)

var validTransitions = map[JobStatus][]JobStatus{
	JobStatusPending:      {JobStatusProvisioning, JobStatusFailed, JobStatusCancelled},
	JobStatusProvisioning: {JobStatusDownloading, JobStatusFailed, JobStatusCancelled},
	JobStatusDownloading:  {JobStatusTraining, JobStatusFailed, JobStatusCancelled},
	JobStatusTraining:     {JobStatusUploading, JobStatusFailed, JobStatusCancelled},
	JobStatusUploading:    {JobStatusCompleted, JobStatusFailed, JobStatusCancelled},
}

type Job struct {
	ID        string         `json:"id"`
	Status    JobStatus      `json:"status"`
	Request   TrainingRequest `json:"request"`
	Result    *TrainingResult `json:"result,omitempty"`
	Error     string         `json:"error,omitempty"`
	Provider  string         `json:"provider,omitempty"`
	GPUType   string         `json:"gpu_type,omitempty"`

	IdempotencyKey string `json:"idempotency_key,omitempty"`

	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	ProvisionedAt  *time.Time `json:"provisioned_at,omitempty"`
	DownloadStartAt *time.Time `json:"download_start_at,omitempty"`
	TrainingStartAt *time.Time `json:"training_start_at,omitempty"`
	UploadStartAt  *time.Time `json:"upload_start_at,omitempty"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
	FailedAt       *time.Time `json:"failed_at,omitempty"`
	CancelledAt    *time.Time `json:"cancelled_at,omitempty"`
}

func NewJob(req TrainingRequest) *Job {
	now := time.Now().UTC()
	return &Job{
		ID:        uuid.New().String(),
		Status:    JobStatusPending,
		Request:   req,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func (j *Job) TransitionTo(status JobStatus) error {
	if j.Status == JobStatusCompleted {
		return ErrJobAlreadyComplete
	}
	if j.Status == JobStatusCancelled {
		return ErrJobAlreadyCancelled
	}
	if j.Status == JobStatusFailed {
		return fmt.Errorf("%w: cannot transition from failed", ErrInvalidTransition)
	}

	allowed := validTransitions[j.Status]
	for _, s := range allowed {
		if s == status {
			j.Status = status
			j.UpdatedAt = time.Now().UTC()
			j.setPhaseTimestamp(status)
			return nil
		}
	}

	return fmt.Errorf("%w: %s -> %s", ErrInvalidTransition, j.Status, status)
}

func (j *Job) setPhaseTimestamp(status JobStatus) {
	now := time.Now().UTC()
	switch status {
	case JobStatusProvisioning:
		j.ProvisionedAt = &now
	case JobStatusDownloading:
		j.DownloadStartAt = &now
	case JobStatusTraining:
		j.TrainingStartAt = &now
	case JobStatusUploading:
		j.UploadStartAt = &now
	case JobStatusCompleted:
		j.CompletedAt = &now
	case JobStatusFailed:
		j.FailedAt = &now
	case JobStatusCancelled:
		j.CancelledAt = &now
	}
}

func (j *Job) Fail(errMsg string) error {
	if err := j.TransitionTo(JobStatusFailed); err != nil {
		return err
	}
	j.Error = errMsg
	return nil
}

func (j *Job) Cancel() error {
	return j.TransitionTo(JobStatusCancelled)
}

func (j *Job) Complete(result TrainingResult) error {
	if err := j.TransitionTo(JobStatusCompleted); err != nil {
		return err
	}
	j.Result = &result
	return nil
}

func (j *Job) IsTerminal() bool {
	return j.Status == JobStatusCompleted || j.Status == JobStatusFailed || j.Status == JobStatusCancelled
}

func (j *Job) RequestJSON() ([]byte, error) {
	return json.Marshal(j.Request)
}

func (j *Job) ResultJSON() ([]byte, error) {
	if j.Result == nil {
		return nil, nil
	}
	return json.Marshal(j.Result)
}
