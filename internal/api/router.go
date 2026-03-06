package api

import (
	"log/slog"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/ryank90/lora-trainer-example/internal/api/handler"
	"github.com/ryank90/lora-trainer-example/internal/api/middleware"
	"github.com/ryank90/lora-trainer-example/internal/config"
)

func NewRouter(
	cfg *config.Config,
	trainingHandler *handler.TrainingHandler,
	healthHandler *handler.HealthHandler,
	logger *slog.Logger,
) *chi.Mux {
	r := chi.NewRouter()

	r.Use(chimiddleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.Logging(logger))
	r.Use(chimiddleware.RealIP)

	r.Get("/healthz", healthHandler.Healthz)
	r.Get("/readyz", healthHandler.Readyz)

	if cfg.Telemetry.MetricsEnabled {
		r.Handle(cfg.Telemetry.MetricsPath, promhttp.Handler())
	}

	r.Route("/v1", func(r chi.Router) {
		r.Use(middleware.Auth(cfg.Auth.APIKeys))

		r.Route("/training/jobs", func(r chi.Router) {
			r.Post("/", trainingHandler.CreateJob)
			r.Get("/", trainingHandler.ListJobs)
			r.Get("/{jobID}", trainingHandler.GetJob)
			r.Post("/{jobID}/cancel", trainingHandler.CancelJob)
		})
	})

	return r
}
