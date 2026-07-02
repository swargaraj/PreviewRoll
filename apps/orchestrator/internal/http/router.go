package http

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/swargaraj/previewroll/apps/orchestrator/internal/config"
	"github.com/swargaraj/previewroll/apps/orchestrator/internal/database"
	"github.com/swargaraj/previewroll/apps/orchestrator/internal/http/handler"
	"github.com/swargaraj/previewroll/apps/orchestrator/internal/queue"
	"github.com/swargaraj/previewroll/apps/orchestrator/internal/worker"
)

// Server wraps http.Server with additional functionality
type Server struct {
	*http.Server
	logger *slog.Logger
}

// NewRouter creates a new HTTP server with all routes
func NewRouter(db *database.DB, cfg *config.Config, pool *worker.Pool, jobQueue *queue.MemoryQueue) *Server {
	mux := http.NewServeMux()
	logger := slog.With("component", "http")

	// Initialize handlers
	authHandler := handler.NewAuthHandler(db, cfg)
	healthHandler := handler.NewHealthHandler(db)
	deploymentHandler := handler.NewDeploymentHandler(db)
	workerHandler := handler.NewWorkerHandler(db, pool)
	projectHandler := handler.NewProjectHandler(db)
	domainHandler := handler.NewDomainHandler(db)
	webhookHandler := handler.NewWebhookHandler(db, jobQueue)

	// Health check endpoints (no auth required)
	mux.HandleFunc("GET /healthz", healthHandler.Healthz)
	mux.HandleFunc("GET /readyz", healthHandler.Readyz)

	// Auth endpoints (no auth required)
	mux.HandleFunc("POST /api/auth/register", authHandler.Register)
	mux.HandleFunc("POST /api/auth/login", authHandler.Login)
	mux.HandleFunc("POST /api/auth/logout", authHandler.Logout)
	mux.HandleFunc("GET /api/auth/me", authHandler.Me)

	// Protected endpoints (require auth)
	mux.Handle("GET /api/v1/deployments", withAuth(db, http.HandlerFunc(deploymentHandler.List)))
	mux.Handle("GET /api/v1/deployments/{id}", withAuth(db, http.HandlerFunc(deploymentHandler.GetByID)))
	mux.Handle("POST /api/v1/deployments/{id}/stop", withAuth(db, http.HandlerFunc(deploymentHandler.Stop)))
	mux.Handle("POST /api/v1/deployments/{id}/restart", withAuth(db, http.HandlerFunc(deploymentHandler.Restart)))
	mux.Handle("GET /api/v1/deployments/{id}/logs", withAuth(db, http.HandlerFunc(deploymentHandler.Logs)))

	mux.Handle("GET /api/v1/workers", withAuth(db, http.HandlerFunc(workerHandler.List)))
	mux.Handle("GET /api/v1/workers/{id}", withAuth(db, http.HandlerFunc(workerHandler.GetByID)))

	mux.Handle("GET /api/v1/projects", withAuth(db, http.HandlerFunc(projectHandler.List)))
	mux.Handle("GET /api/v1/projects/{id}", withAuth(db, http.HandlerFunc(projectHandler.GetByID)))
	mux.Handle("POST /api/v1/projects", withAuth(db, http.HandlerFunc(projectHandler.Create)))
	mux.Handle("DELETE /api/v1/projects/{id}", withAuth(db, http.HandlerFunc(projectHandler.Delete)))

	mux.Handle("GET /api/v1/domains", withAuth(db, http.HandlerFunc(domainHandler.List)))
	mux.Handle("POST /api/v1/domains", withAuth(db, http.HandlerFunc(domainHandler.Create)))
	mux.Handle("DELETE /api/v1/domains/{id}", withAuth(db, http.HandlerFunc(domainHandler.Delete)))
	mux.Handle("POST /api/v1/domains/{id}/projects", withAuth(db, http.HandlerFunc(domainHandler.ConnectProject)))
	mux.Handle("DELETE /api/v1/domains/{id}/projects/{projectId}", withAuth(db, http.HandlerFunc(domainHandler.DisconnectProject)))
	mux.Handle("GET /api/v1/projects/{projectId}/domains", withAuth(db, http.HandlerFunc(domainHandler.ListProjectDomains)))

	// Worker endpoints (no auth required - workers authenticate via other means)
	mux.HandleFunc("POST /api/workers/register", workerHandler.Register)
	mux.HandleFunc("POST /api/workers/heartbeat", workerHandler.Heartbeat)
	mux.HandleFunc("POST /api/workers/deregister", workerHandler.Deregister)

	// Webhook endpoints (no auth required - uses signature verification)
	mux.HandleFunc("POST /webhook/github", webhookHandler.HandleGitHubWebhook)

	// Wrap with middleware
	handler := withLogging(mux, logger)
	handler = withCORS(handler, cfg.CORSOrigins)
	handler = withRecovery(handler)

	return &Server{
		Server: &http.Server{
			Addr:         cfg.HTTPAddr,
			Handler:      handler,
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 15 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
		logger: logger,
	}
}

// withLogging adds request logging middleware
func withLogging(next http.Handler, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap response writer to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapped, r)

		logger.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", wrapped.statusCode,
			"duration", time.Since(start).String(),
			"remote_addr", r.RemoteAddr,
		)
	})
}

// withCORS adds CORS headers with dynamic origin support
func withCORS(next http.Handler, allowedOrigins []string) http.Handler {
	allowAll := false
	originSet := make(map[string]bool, len(allowedOrigins))
	for _, o := range allowedOrigins {
		if o == "*" {
			allowAll = true
		}
		originSet[o] = true
	}

	if len(allowedOrigins) > 0 {
		slog.Warn("CORS enabled", "allowed_origins", allowedOrigins)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		if allowAll {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		} else if originSet[origin] {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Max-Age", "86400")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// withRecovery adds panic recovery middleware
func withRecovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				slog.Error("panic recovered", "error", err, "path", r.URL.Path)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(w, r)
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// withAuth adds authentication middleware that validates sessions against DB
func withAuth(db *database.DB, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("__Host-sid")
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		if cookie.Value == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		userID, err := db.Q().GetSessionByToken(r.Context(), cookie.Value)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), handler.UserIDKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("shutting down http server")
	return s.Server.Shutdown(ctx)
}
