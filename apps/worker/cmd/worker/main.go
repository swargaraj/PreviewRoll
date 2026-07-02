package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/swargaraj/previewroll/apps/worker/internal/config"
	"github.com/swargaraj/previewroll/apps/worker/internal/docker"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// Initialize logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	slog.Info("starting worker",
		"name", cfg.WorkerName,
		"orchestrator_addr", cfg.OrchestratorAddr,
		"max_concurrent", cfg.MaxConcurrent,
	)

	// Initialize Docker client
	dockerClient, err := docker.NewClient(cfg.DockerSocket)
	if err != nil {
		slog.Error("failed to initialize docker client", "error", err)
		os.Exit(1)
	}
	defer dockerClient.Close()

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start heartbeat goroutine
	go startHeartbeat(ctx, cfg, logger)

	// TODO: Connect to orchestrator via gRPC
	// TODO: Listen for job assignments
	// TODO: Process jobs

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down worker")
	cancel()

	// Give some time for cleanup
	time.Sleep(2 * time.Second)

	slog.Info("worker stopped")
}

func startHeartbeat(ctx context.Context, cfg *config.Config, logger *slog.Logger) {
	ticker := time.NewTicker(time.Duration(cfg.HeartbeatInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// TODO: Send heartbeat to orchestrator
			logger.Debug("heartbeat sent")
		}
	}
}
