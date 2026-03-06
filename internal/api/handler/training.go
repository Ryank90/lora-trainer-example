package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/ryank90/lora-trainer-example/internal/domain"
	"github.com/ryank90/lora-trainer-example/internal/queue"
	"github.com/ryank90/lora-trainer-example/internal/repository"
)

type TrainingHandler struct {
	repo   repository.JobRepository
	queue  *queue.RedisQueue
	logger *slog.Logger
}

func NewTrainingHandler(repo repository.JobRepository, queue *queue.RedisQueue, logger *slog.Logger) *TrainingHandler {
	return &TrainingHandler{
		repo:   repo,
		queue:  queue,
		logger: logger,
	}
}

func (h *TrainingHandler) CreateJob(w http.ResponseWriter, r *http.Request) {
	var req domain.TrainingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.ApplyDefaults()

	if err := req.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Idempotency check
	idempotencyKey := r.Header.Get("Idempotency-Key")
	if idempotencyKey != "" {
		existing, err := h.repo.GetByIdempotencyKey(r.Context(), idempotencyKey)
		if err != nil {
			h.logger.Error("idempotency check failed", "error", err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		if existing != nil {
			writeJSON(w, http.StatusOK, existing)
			return
		}
	}

	job := domain.NewJob(req)
	if idempotencyKey != "" {
		job.IdempotencyKey = idempotencyKey
	}

	if err := h.repo.Create(r.Context(), job); err != nil {
		h.logger.Error("failed to create job", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to create job")
		return
	}

	if err := h.queue.Enqueue(r.Context(), job.ID); err != nil {
		h.logger.Error("failed to enqueue job", "error", err, "job_id", job.ID)
		writeError(w, http.StatusInternalServerError, "failed to enqueue job")
		return
	}

	h.logger.Info("job created", "job_id", job.ID, "model_type", req.ModelType)
	writeJSON(w, http.StatusCreated, job)
}

func (h *TrainingHandler) GetJob(w http.ResponseWriter, r *http.Request) {
	jobID := chi.URLParam(r, "jobID")

	job, err := h.repo.Get(r.Context(), jobID)
	if err != nil {
		if errors.Is(err, domain.ErrJobNotFound) {
			writeError(w, http.StatusNotFound, "job not found")
			return
		}
		h.logger.Error("failed to get job", "error", err, "job_id", jobID)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, job)
}

func (h *TrainingHandler) ListJobs(w http.ResponseWriter, r *http.Request) {
	filter := repository.JobFilter{}

	if s := r.URL.Query().Get("status"); s != "" {
		status := domain.JobStatus(s)
		filter.Status = &status
	}
	if mt := r.URL.Query().Get("model_type"); mt != "" {
		modelType := domain.ModelType(mt)
		filter.ModelType = &modelType
	}
	if l := r.URL.Query().Get("limit"); l != "" {
		if limit, err := strconv.Atoi(l); err == nil {
			filter.Limit = limit
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if offset, err := strconv.Atoi(o); err == nil {
			filter.Offset = offset
		}
	}

	jobs, err := h.repo.List(r.Context(), filter)
	if err != nil {
		h.logger.Error("failed to list jobs", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"jobs":  jobs,
		"count": len(jobs),
	})
}

func (h *TrainingHandler) CancelJob(w http.ResponseWriter, r *http.Request) {
	jobID := chi.URLParam(r, "jobID")

	job, err := h.repo.Get(r.Context(), jobID)
	if err != nil {
		if errors.Is(err, domain.ErrJobNotFound) {
			writeError(w, http.StatusNotFound, "job not found")
			return
		}
		h.logger.Error("failed to get job", "error", err, "job_id", jobID)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	if err := job.Cancel(); err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}

	if err := h.repo.Update(r.Context(), job); err != nil {
		h.logger.Error("failed to cancel job", "error", err, "job_id", jobID)
		writeError(w, http.StatusInternalServerError, "failed to cancel job")
		return
	}

	h.logger.Info("job cancelled", "job_id", jobID)
	writeJSON(w, http.StatusOK, job)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
