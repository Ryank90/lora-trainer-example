package orchestrator

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/ryank90/lora-trainer-example/internal/config"
	"github.com/ryank90/lora-trainer-example/internal/domain"
	"github.com/ryank90/lora-trainer-example/internal/provider"
	"github.com/ryank90/lora-trainer-example/internal/queue"
	"github.com/ryank90/lora-trainer-example/internal/repository"
	"github.com/ryank90/lora-trainer-example/internal/storage"
	"github.com/ryank90/lora-trainer-example/internal/training"
)

type Orchestrator struct {
	cfg        config.OrchestratorConfig
	repo       repository.JobRepository
	queue      *queue.RedisQueue
	storage    storage.StorageService
	dispatcher *Dispatcher
	executor   training.TrainingExecutor
	registry   *training.Registry
	warmPool   *WarmPool
	logger     *slog.Logger

	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func New(
	cfg config.OrchestratorConfig,
	repo repository.JobRepository,
	q *queue.RedisQueue,
	store storage.StorageService,
	dispatcher *Dispatcher,
	executor training.TrainingExecutor,
	registry *training.Registry,
	warmPool *WarmPool,
	logger *slog.Logger,
) *Orchestrator {
	return &Orchestrator{
		cfg:        cfg,
		repo:       repo,
		queue:      q,
		storage:    store,
		dispatcher: dispatcher,
		executor:   executor,
		registry:   registry,
		warmPool:   warmPool,
		logger:     logger,
	}
}

func (o *Orchestrator) Start(ctx context.Context) {
	ctx, o.cancel = context.WithCancel(ctx)

	for i := 0; i < o.cfg.Workers; i++ {
		o.wg.Add(1)
		go o.worker(ctx, i)
	}

	o.logger.Info("orchestrator started", "workers", o.cfg.Workers)
}

func (o *Orchestrator) Stop() {
	o.logger.Info("stopping orchestrator")
	o.cancel()
	o.wg.Wait()
	o.logger.Info("orchestrator stopped")
}

func (o *Orchestrator) worker(ctx context.Context, id int) {
	defer o.wg.Done()

	logger := o.logger.With("worker_id", id)
	logger.Info("worker started")

	for {
		select {
		case <-ctx.Done():
			logger.Info("worker stopping")
			return
		default:
			jobID, err := o.queue.Dequeue(ctx, o.cfg.PollInterval)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				logger.Error("dequeue error", "error", err)
				time.Sleep(time.Second)
				continue
			}
			if jobID == "" {
				continue
			}

			logger.Info("processing job", "job_id", jobID)
			if err := o.processJob(ctx, jobID); err != nil {
				logger.Error("job processing failed", "job_id", jobID, "error", err)
				o.queue.Nack(ctx, jobID)
			} else {
				o.queue.Ack(ctx, jobID)
			}
		}
	}
}

func (o *Orchestrator) processJob(ctx context.Context, jobID string) error {
	job, err := o.repo.Get(ctx, jobID)
	if err != nil {
		return fmt.Errorf("loading job: %w", err)
	}

	if job.IsTerminal() {
		o.logger.Info("skipping terminal job", "job_id", jobID, "status", job.Status)
		return nil
	}

	modelConfig, err := o.registry.Get(job.Request.ModelType)
	if err != nil {
		return o.failJob(ctx, job, fmt.Sprintf("unknown model type: %s", job.Request.ModelType))
	}

	// Phase: Provisioning
	if err := job.TransitionTo(domain.JobStatusProvisioning); err != nil {
		return fmt.Errorf("transition to provisioning: %w", err)
	}
	o.repo.Update(ctx, job)

	instance, err := o.provision(ctx, job, modelConfig)
	if err != nil {
		return o.failJob(ctx, job, fmt.Sprintf("provisioning failed: %v", err))
	}
	defer o.cleanup(context.Background(), instance, job)

	job.Provider = instance.Provider
	job.GPUType = string(instance.GPUType)

	// Phase: Downloading
	if err := job.TransitionTo(domain.JobStatusDownloading); err != nil {
		return fmt.Errorf("transition to downloading: %w", err)
	}
	o.repo.Update(ctx, job)

	// Phase: Training
	if err := job.TransitionTo(domain.JobStatusTraining); err != nil {
		return fmt.Errorf("transition to training: %w", err)
	}
	o.repo.Update(ctx, job)

	if err := o.executor.Execute(ctx, job, instance, modelConfig); err != nil {
		return o.failJob(ctx, job, fmt.Sprintf("training failed: %v", err))
	}

	// Phase: Uploading
	if err := job.TransitionTo(domain.JobStatusUploading); err != nil {
		return fmt.Errorf("transition to uploading: %w", err)
	}
	o.repo.Update(ctx, job)

	result, err := o.collectResults(ctx, job)
	if err != nil {
		return o.failJob(ctx, job, fmt.Sprintf("upload failed: %v", err))
	}

	result.ProviderRef = instance.ID
	result.GPUType = string(instance.GPUType)

	// Phase: Completed
	if err := job.Complete(*result); err != nil {
		return fmt.Errorf("completing job: %w", err)
	}
	o.repo.Update(ctx, job)

	o.logger.Info("job completed", "job_id", jobID, "provider", instance.Provider)
	return nil
}

func (o *Orchestrator) provision(ctx context.Context, job *domain.Job, config training.ModelConfig) (*provider.Instance, error) {
	provisionCtx, cancel := context.WithTimeout(ctx, o.cfg.ProvisionTimeout)
	defer cancel()

	// Try warm pool first
	if o.warmPool != nil {
		if instance := o.warmPool.Acquire(config.GPUType); instance != nil {
			o.logger.Info("acquired warm instance", "job_id", job.ID, "instance_id", instance.ID)
			return instance, nil
		}
	}

	return o.dispatcher.Dispatch(provisionCtx, config)
}

func (o *Orchestrator) collectResults(ctx context.Context, job *domain.Job) (*domain.TrainingResult, error) {
	uploadCtx, cancel := context.WithTimeout(ctx, o.cfg.UploadTimeout)
	defer cancel()

	_ = uploadCtx
	// Results are uploaded by the training container via upload_results.py
	// which POSTs back to our API. For now, return placeholder.
	return &domain.TrainingResult{
		Files: []domain.FileRef{
			{
				Key:         fmt.Sprintf("jobs/%s/outputs/lora_weights.safetensors", job.ID),
				Filename:    "lora_weights.safetensors",
				ContentType: "application/octet-stream",
			},
			{
				Key:         fmt.Sprintf("jobs/%s/outputs/training_config.json", job.ID),
				Filename:    "training_config.json",
				ContentType: "application/json",
			},
		},
	}, nil
}

func (o *Orchestrator) failJob(ctx context.Context, job *domain.Job, errMsg string) error {
	o.logger.Error("job failed", "job_id", job.ID, "error", errMsg)
	job.Fail(errMsg)
	o.repo.Update(ctx, job)
	return fmt.Errorf("job %s failed: %s", job.ID, errMsg)
}

func (o *Orchestrator) cleanup(ctx context.Context, instance *provider.Instance, job *domain.Job) {
	if o.warmPool != nil && o.warmPool.CanReturn(instance) {
		o.warmPool.Return(instance)
		o.logger.Info("returned instance to warm pool", "instance_id", instance.ID)
		return
	}

	o.logger.Info("terminating instance", "instance_id", instance.ID)
	// Termination is handled by the dispatcher
}
