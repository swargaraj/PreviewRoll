package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds all configuration for the orchestrator
type Config struct {
	// Server
	HTTPAddr string
	GRPCAddr string
	LogLevel string

	// Database
	DatabasePath string

	// Auth
	SessionSecret string
	SessionMaxAge int

	// Docker
	DockerSocket string
	DefaultImage string

	// Worker
	WorkerName       string
	OrchestratorAddr string
	MaxConcurrent    int

	// Logging
	LogDir    string
	LogMaxAge int

	// CORS
	CORSOrigins []string

	// Proxy
	BaseDomain  string
	TraefikAddr string
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{
		HTTPAddr:         getEnv("HTTP_ADDR", ":8080"),
		GRPCAddr:         getEnv("GRPC_ADDR", ":9090"),
		LogLevel:         getEnv("LOG_LEVEL", "info"),
		DatabasePath:     getEnv("DATABASE_PATH", "./data/previewroll.db"),
		SessionSecret:    getEnv("SESSION_SECRET", ""),
		SessionMaxAge:    getEnvInt("SESSION_MAX_AGE", 604800), // 7 days
		DockerSocket:     getEnv("DOCKER_SOCKET", "/var/run/docker.sock"),
		DefaultImage:     getEnv("DEFAULT_IMAGE", "node:20-alpine"),
		WorkerName:       getEnv("WORKER_NAME", ""),
		OrchestratorAddr: getEnv("ORCHESTRATOR_ADDR", ""),
		MaxConcurrent:    getEnvInt("MAX_CONCURRENT", 5),
		LogDir:           getEnv("LOG_DIR", "./data/logs"),
		LogMaxAge:        getEnvInt("LOG_MAX_AGE", 30), // days
		CORSOrigins:      getEnvList("CORS_ORIGINS", "*"),
		BaseDomain:       getEnv("BASE_DOMAIN", "preview.localhost"),
		TraefikAddr:      getEnv("TRAEFIK_ADDR", "http://localhost:8080"),
	}

	// Validate required fields
	if cfg.SessionSecret == "" {
		return nil, fmt.Errorf("SESSION_SECRET is required")
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

func getEnvList(key, fallback string) []string {
	value := getEnv(key, fallback)
	var list []string
	for _, item := range strings.Split(value, ",") {
		trimmed := strings.TrimSpace(item)
		if trimmed != "" {
			list = append(list, trimmed)
		}
	}
	return list
}
