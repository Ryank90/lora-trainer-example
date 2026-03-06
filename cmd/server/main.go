package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"
	"github.com/ryank90/lora-trainer-example/internal/api"
	"github.com/ryank90/lora-trainer-example/internal/api/handler"
	"github.com/ryank90/lora-trainer-example/internal/config"
	"github.com/ryank90/lora-trainer-example/internal/orchestrator"
	"github.com/ryank90/lora-trainer-example/internal/provider"
	"github.com/ryank90/lora-trainer-example/internal/provider/runpod"
	"github.com/ryank90/lora-trainer-example/internal/provider/voltagepark"
	"github.com/ryank90/lora-trainer-example/internal/queue"
	"github.com/ryank90/lora-trainer-example/internal/repository/postgres"
	"github.com/ryank90/lora-trainer-example/internal/storage/r2"
	"github.com/ryank90/lora-trainer-example/internal/telemetry"
	"github.com/ryank90/lora-trainer-example/internal/training"
)

func main() {
	configPath := flag.String("config", "configs/config.yaml", "config file path")
	flag.Parse()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	cfg, err := config.Load(*configPath)
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Database
	dbPool, err := pgxpool.New(ctx, cfg.Database.DSN())
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer dbPool.Close()

	// Redis
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	defer redisClient.Close()

	if err := redisClient.Ping(ctx).Err(); err != nil {
		logger.Error("failed to connect to redis", "error", err)
		os.Exit(1)
	}

	// Storage
	storageClient, err := r2.NewClient(ctx, cfg.Storage)
	if err != nil {
		logger.Error("failed to create storage client", "error", err)
		os.Exit(1)
	}

	// Telemetry
	if cfg.Telemetry.MetricsEnabled {
		telemetry.NewMetrics(prometheus.DefaultRegisterer)
	}

	if cfg.Telemetry.TracingEnabled {
		shutdown, err := telemetry.InitTracer(ctx, cfg.Telemetry.ServiceName, cfg.Telemetry.OTLPEndpoint)
		if err != nil {
			logger.Error("failed to init tracer", "error", err)
			os.Exit(1)
		}
		defer shutdown(context.Background())
	}

	// Repository & Queue
	jobRepo := postgres.NewJobRepo(dbPool)
	jobQueue := queue.NewRedisQueue(redisClient)

	// Providers
	providers := map[string]provider.GPUProvider{}
	if cfg.Providers.VoltagePark.Enabled {
		providers["voltagepark"] = voltagepark.NewProvider(
			cfg.Providers.VoltagePark.Endpoint,
			cfg.Providers.VoltagePark.APIKey,
		)
	}
	if cfg.Providers.Runpod.Enabled {
		providers["runpod"] = runpod.NewProvider(
			cfg.Providers.Runpod.Endpoint,
			cfg.Providers.Runpod.APIKey,
		)
	}

	providerRouter := provider.NewProviderRouter(providers, cfg.Providers.Preference, logger)

	// Training
	registry := training.NewRegistry()
	executor := training.NewSSHExecutor(logger)

	// Orchestrator
	warmPool := orchestrator.NewWarmPool(cfg.Orchestrator.WarmPool, logger)
	dispatcher := orchestrator.NewDispatcher(providerRouter, logger)
	orch := orchestrator.New(
		cfg.Orchestrator, jobRepo, jobQueue, storageClient,
		dispatcher, executor, registry, warmPool, logger,
	)

	// Handlers
	trainingHandler := handler.NewTrainingHandler(jobRepo, jobQueue, logger)
	healthHandler := handler.NewHealthHandler(dbPool, redisClient)

	// Router & Server
	router := api.NewRouter(cfg, trainingHandler, healthHandler, logger)
	server := api.NewServer(router, cfg.Server.Host, cfg.Server.Port,
		cfg.Server.ReadTimeout, cfg.Server.WriteTimeout, logger)

	// Start orchestrator
	orch.Start(ctx)

	// Graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := server.Start(); err != nil {
			logger.Error("server error", "error", err)
			cancel()
		}
	}()

	logger.Info("server started", "port", cfg.Server.Port)

	sig := <-sigCh
	logger.Info("received signal, shutting down", "signal", sig)

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer shutdownCancel()

	orch.Stop()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("server shutdown error", "error", err)
	}

	logger.Info("shutdown complete")
}
