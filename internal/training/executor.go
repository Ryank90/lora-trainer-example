package training

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/ryank90/lora-trainer-example/internal/domain"
	"github.com/ryank90/lora-trainer-example/internal/provider"
	"golang.org/x/crypto/ssh"
)

type TrainingExecutor interface {
	Execute(ctx context.Context, job *domain.Job, instance *provider.Instance, config ModelConfig) error
	Monitor(ctx context.Context, job *domain.Job, instance *provider.Instance) (*domain.TrainingMetrics, error)
}

type SSHExecutor struct {
	logger *slog.Logger
}

func NewSSHExecutor(logger *slog.Logger) *SSHExecutor {
	return &SSHExecutor{logger: logger}
}

func (e *SSHExecutor) Execute(ctx context.Context, job *domain.Job, instance *provider.Instance, config ModelConfig) error {
	client, err := e.connect(instance)
	if err != nil {
		return fmt.Errorf("SSH connect: %w", err)
	}
	defer client.Close()

	envVars := e.buildEnvVars(job, config)
	dockerCmd := e.buildDockerRunCommand(config, envVars)

	e.logger.Info("executing training container",
		"job_id", job.ID,
		"instance_id", instance.ID,
		"docker_image", config.DockerImage,
	)

	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("SSH session: %w", err)
	}
	defer session.Close()

	output, err := session.CombinedOutput(dockerCmd)
	if err != nil {
		return fmt.Errorf("training execution failed: %w\noutput: %s", err, string(output))
	}

	e.logger.Info("training completed", "job_id", job.ID)
	return nil
}

func (e *SSHExecutor) Monitor(ctx context.Context, job *domain.Job, instance *provider.Instance) (*domain.TrainingMetrics, error) {
	client, err := e.connect(instance)
	if err != nil {
		return nil, fmt.Errorf("SSH connect for monitoring: %w", err)
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return nil, fmt.Errorf("SSH session: %w", err)
	}
	defer session.Close()

	output, err := session.CombinedOutput("docker logs --tail 100 lora-trainer-" + job.ID)
	if err != nil {
		return nil, fmt.Errorf("reading training logs: %w", err)
	}

	e.logger.Debug("training logs", "job_id", job.ID, "output", string(output))
	return nil, nil
}

func (e *SSHExecutor) connect(instance *provider.Instance) (*ssh.Client, error) {
	config := &ssh.ClientConfig{
		User:            instance.SSHUser,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         30 * time.Second,
	}

	if instance.SSHKeyPath != "" {
		// In production, load the key from file
		config.Auth = []ssh.AuthMethod{
			ssh.Password(""), // placeholder
		}
	}

	addr := fmt.Sprintf("%s:%d", instance.IP, instance.SSHPort)
	return ssh.Dial("tcp", addr, config)
}

func (e *SSHExecutor) buildEnvVars(job *domain.Job, config ModelConfig) map[string]string {
	env := map[string]string{
		"JOB_ID":        job.ID,
		"MODEL_TYPE":    string(job.Request.ModelType),
		"TRIGGER_WORD":  job.Request.TriggerWord,
		"DATASET_URL":   job.Request.DatasetURL,
		"STEPS":         fmt.Sprintf("%d", job.Request.Steps),
		"LEARNING_RATE": fmt.Sprintf("%g", job.Request.LearningRate),
		"LORA_RANK":     fmt.Sprintf("%d", job.Request.LoRARank),
		"RESOLUTION":    fmt.Sprintf("%d", job.Request.Resolution),
		"BATCH_SIZE":    fmt.Sprintf("%d", job.Request.BatchSize),
	}

	if job.Request.Seed != nil {
		env["SEED"] = fmt.Sprintf("%d", *job.Request.Seed)
	}
	if job.Request.WebhookURL != "" {
		env["WEBHOOK_URL"] = job.Request.WebhookURL
	}

	if job.Request.ModelType.IsFluxVariant() {
		env["MODEL_VARIANT"] = job.Request.ModelType.FluxVariant()
	}

	for k, v := range config.ExtraEnv {
		env[k] = v
	}

	return env
}

func (e *SSHExecutor) buildDockerRunCommand(config ModelConfig, env map[string]string) string {
	parts := []string{
		"docker", "run",
		"--rm",
		"--gpus", "all",
		"--name", "lora-trainer-${JOB_ID}",
		"--shm-size", "16g",
	}

	for _, mount := range config.VolumeMounts {
		parts = append(parts, "-v", mount)
	}

	for k, v := range env {
		parts = append(parts, "-e", fmt.Sprintf("%s=%s", k, v))
	}

	parts = append(parts, config.DockerImage)

	return strings.Join(parts, " ")
}
