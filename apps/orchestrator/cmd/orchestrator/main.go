package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/lmittmann/tint"
	"github.com/swargaraj/previewroll/apps/orchestrator/internal/config"
	"github.com/swargaraj/previewroll/apps/orchestrator/internal/database"
	httpserver "github.com/swargaraj/previewroll/apps/orchestrator/internal/http"
	"github.com/swargaraj/previewroll/apps/orchestrator/internal/queue"
	"github.com/swargaraj/previewroll/apps/orchestrator/internal/scheduler"
	"github.com/swargaraj/previewroll/apps/orchestrator/internal/worker"
	"gopkg.in/natefinch/lumberjack.v2"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	var level slog.Level
	if err := level.UnmarshalText([]byte(cfg.LogLevel)); err != nil {
		level = slog.LevelInfo
	}

	tintHandler := tint.NewHandler(os.Stdout, &tint.Options{Level: level})
	fileHandler := slog.NewTextHandler(&lumberjack.Logger{
		Filename: filepath.Join(cfg.LogDir, "orchestrator.log"),
		MaxAge:   cfg.LogMaxAge,
		Compress: true,
	}, &slog.HandlerOptions{Level: level})

	logger := slog.New(slog.NewMultiHandler(tintHandler, fileHandler))
	slog.SetDefault(logger)

	slog.Info("starting orchestrator",
		"http_addr", cfg.HTTPAddr,
		"grpc_addr", cfg.GRPCAddr,
		"database_path", cfg.DatabasePath,
		"log_dir", cfg.LogDir,
	)

	db, err := database.New(cfg.DatabasePath)
	if err != nil {
		slog.Error("failed to initialize database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := database.RunMigrations(db); err != nil {
		slog.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}

	workerPool := worker.NewPool(slog.With("component", "worker_pool"))

	existingWorkers, err := db.LoadWorkers()
	if err != nil {
		slog.Error("failed to load workers from database", "error", err)
		os.Exit(1)
	}
	for _, w := range existingWorkers {
		workerPool.Register(w)
	}
	slog.Info("loaded workers from database", "count", len(existingWorkers))

	jobQueue := queue.NewMemoryQueue()

	sched := scheduler.NewScheduler(slog.With("component", "scheduler"))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	workerPool.StartHealthCheck(ctx, 15*time.Second)

	go sched.Start(ctx, jobQueue, workerPool)

	router := httpserver.NewRouter(db, cfg, workerPool, jobQueue)

	errCh := make(chan error, 1)
	go func() {
		slog.Info("http server starting", "addr", cfg.HTTPAddr)
		if err := router.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-quit:
		slog.Info("shutting down orchestrator", "signal", sig)
	case err := <-errCh:
		slog.Error("http server failed", "error", err)
	}

	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := router.Shutdown(shutdownCtx); err != nil {
		slog.Error("server forced to shutdown", "error", err)
	}

	slog.Info("orchestrator stopped")
}
