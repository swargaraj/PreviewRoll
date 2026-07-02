package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds all configuration for the worker
type Config struct {
	// Worker
	WorkerName       string
	OrchestratorAddr string
	MaxConcurrent    int

	// Docker
	DockerSocket string
	DefaultImage string

	// Logging
	LogDir string

	// Heartbeat
	HeartbeatInterval int
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{
		WorkerName:       getEnv("WORKER_NAME", ""),
		OrchestratorAddr: getEnv("ORCHESTRATOR_ADDR", ""),
		MaxConcurrent:    getEnvInt("MAX_CONCURRENT", 5),
		DockerSocket:     getEnv("DOCKER_SOCKET", "unix:///var/run/docker.sock"),
		DefaultImage:     getEnv("DEFAULT_IMAGE", "node:20-alpine"),
		LogDir:           getEnv("LOG_DIR", "./data/logs"),
		HeartbeatInterval: getEnvInt("HEARTBEAT_INTERVAL", 15),
	}

	// Validate required fields
	if cfg.WorkerName == "" {
		return nil, fmt.Errorf("WORKER_NAME is required")
	}

	if cfg.OrchestratorAddr == "" {
		return nil, fmt.Errorf("ORCHESTRATOR_ADDR is required")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if value, exists := os.LookupEnv(key); exists {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return fallback
}
