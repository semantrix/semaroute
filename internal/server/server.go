package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/semantrix/semaroute/internal/cache"
	"github.com/semantrix/semaroute/internal/observability"
	"github.com/semantrix/semaroute/internal/providers"
	"github.com/semantrix/semaroute/internal/router/health"
	"github.com/semantrix/semaroute/internal/router/policies"
	"go.uber.org/zap"
)

// Server represents the main HTTP server for the semaroute service.
type Server struct {
	config        *Config
	router        *chi.Mux
	providers     map[string]providers.Provider
	routingPolicy policies.RoutingPolicy
	healthChecker *health.HealthChecker
	cache         cache.CacheClient
	logger        *zap.Logger
	metrics       *observability.Metrics
	tracing       *observability.Tracing
	server        *http.Server
}

// Config holds the server configuration.
type Config struct {
	Server struct {
		Port            int           `mapstructure:"port"`
		ReadTimeout     time.Duration `mapstructure:"read_timeout"`
		WriteTimeout    time.Duration `mapstructure:"write_timeout"`
		IdleTimeout     time.Duration `mapstructure:"idle_timeout"`
		ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout"`
	} `mapstructure:"server"`

	Providers map[string]providers.ProviderConfig `mapstructure:"providers"`

	RoutingPolicy struct {
		Type   string                 `mapstructure:"type"`
		Config map[string]interface{} `mapstructure:"config"`
	} `mapstructure:"routing_policy"`

	HealthCheck struct {
		Interval time.Duration `mapstructure:"interval"`
		Timeout  time.Duration `mapstructure:"timeout"`
	} `mapstructure:"health_check"`

	Cache cache.CacheConfig `mapstructure:"cache"`

	Observability struct {
		Logging observability.LoggerConfig  `mapstructure:"logging"`
		Metrics observability.MetricsConfig `mapstructure:"metrics"`
		Tracing observability.TracingConfig `mapstructure:"tracing"`
	} `mapstructure:"observability"`
}

// NewServer creates a new server instance.
func NewServer(config *Config) (*Server, error) {
	// Initialize logger
	logger, err := observability.NewLogger(config.Observability.Logging)
	if err != nil {
		return nil, fmt.Errorf("failed to create logger: %w", err)
	}

	// Initialize metrics
	metrics, err := observability.NewMetrics(config.Observability.Metrics, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create metrics: %w", err)
	}

	// Initialize tracing
	tracing := observability.NewTracing(config.Observability.Tracing, logger)

	// Initialize cache
	cacheClient := cache.NewMemoryCache(config.Cache)

	// Initialize providers
	providersMap, err := initializeProviders(config.Providers, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize providers: %w", err)
	}

	// Initialize routing policy
	routingPolicy, err := initializeRoutingPolicy(config.RoutingPolicy, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize routing policy: %w", err)
	}

	// Initialize health checker
	healthChecker := health.NewHealthChecker(
		config.HealthCheck.Interval,
		config.HealthCheck.Timeout,
		logger,
	)

	// Add providers to health checker
	for name, provider := range providersMap {
		healthChecker.AddProvider(name, provider)
	}

	// Create server instance
	server := &Server{
		config:        config,
		router:        chi.NewRouter(),
		providers:     providersMap,
		routingPolicy: routingPolicy,
		healthChecker: healthChecker,
		cache:         cacheClient,
		logger:        logger,
		metrics:       metrics,
		tracing:       tracing,
	}

	// Setup routes and middleware
	server.setupRoutes()

	// Create HTTP server
	server.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", config.Server.Port),
		Handler:      server.router,
		ReadTimeout:  config.Server.ReadTimeout,
		WriteTimeout: config.Server.WriteTimeout,
		IdleTimeout:  config.Server.IdleTimeout,
	}

	return server, nil
}

// setupRoutes configures the HTTP routes and middleware.
func (s *Server) setupRoutes() {
	// Add middleware
	s.router.Use(middleware.RequestID)
	s.router.Use(middleware.RealIP)
	s.router.Use(middleware.Logger)
	s.router.Use(middleware.Recoverer)
	s.router.Use(s.observabilityMiddleware)
	s.router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Health check endpoint
	s.router.Get("/health", s.handleHealthCheck)

	// API v1 routes
	s.router.Route("/v1", func(r chi.Router) {
		r.Post("/chat/completions", s.handleChatCompletion)
		r.Get("/models", s.handleGetModels)
		r.Get("/routing/info", s.handleGetRoutingInfo)
		r.Get("/metrics", s.handleGetMetrics)
	})

	// Admin routes
	s.router.Route("/admin", func(r chi.Router) {
		r.Get("/providers", s.handleGetProviders)
		r.Get("/providers/{name}/health", s.handleGetProviderHealth)
		r.Post("/providers/{name}/health-check", s.handleForceHealthCheck)
		r.Get("/routing/policy", s.handleGetRoutingPolicy)
		r.Put("/routing/policy", s.handleUpdateRoutingPolicy)
	})
}

// observabilityMiddleware adds observability features to requests.
func (s *Server) observabilityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Start tracing span
		ctx, span := s.tracing.StartSpan(r.Context(), "http_request")
		defer span.End()

		// Add request attributes
		s.tracing.SetAttributes(ctx, map[string]string{
			"http.method":     r.Method,
			"http.url":        r.URL.String(),
			"http.user_agent": r.UserAgent(),
		})

		// Create response writer wrapper for status code
		wrappedWriter := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Process request
		next.ServeHTTP(wrappedWriter, r.WithContext(ctx))

		// Record metrics
		duration := time.Since(start)
		s.metrics.RecordRequest(r.Method, r.URL.Path, wrappedWriter.statusCode, duration)

		// Add response attributes
		s.tracing.SetAttributes(ctx, map[string]string{
			"http.status_code": fmt.Sprintf("%d", wrappedWriter.statusCode),
			"http.duration_ms": fmt.Sprintf("%d", duration.Milliseconds()),
		})
	})
}

// responseWriter wraps http.ResponseWriter to capture status code.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Start starts the server and begins accepting requests.
func (s *Server) Start() error {
	// Start health checker
	s.healthChecker.Start()

	// Start metrics server if enabled
	if s.config.Observability.Metrics.Enabled {
		metricsCtx, cancel := context.WithCancel(context.Background())
		defer cancel()
		go func() {
			if err := s.metrics.StartMetricsServer(metricsCtx); err != nil {
				s.logger.Error("Failed to start metrics server", zap.Error(err))
			}
		}()
	}

	s.logger.Info("Starting semaroute server",
		zap.Int("port", s.config.Server.Port),
		zap.Int("providers", len(s.providers)))

	// Start server in goroutine
	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("Server error", zap.Error(err))
		}
	}()

	return nil
}

// Stop gracefully shuts down the server.
func (s *Server) Stop() error {
	s.logger.Info("Shutting down server...")

	// Stop health checker
	s.healthChecker.Stop()

	// Create shutdown context
	ctx, cancel := context.WithTimeout(context.Background(), s.config.Server.ShutdownTimeout)
	defer cancel()

	// Shutdown server
	if err := s.server.Shutdown(ctx); err != nil {
		s.logger.Error("Error during server shutdown", zap.Error(err))
		return err
	}

	// Close cache
	if err := s.cache.Close(); err != nil {
		s.logger.Error("Error closing cache", zap.Error(err))
	}

	// Close providers
	for name, provider := range s.providers {
		if err := provider.Close(); err != nil {
			s.logger.Error("Error closing provider", zap.String("provider", name), zap.Error(err))
		}
	}

	// Sync logger
	observability.SyncLogger(s.logger)

	s.logger.Info("Server stopped")
	return nil
}

// WaitForShutdown waits for shutdown signals and gracefully stops the server.
func (s *Server) WaitForShutdown() {
	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	s.logger.Info("Received shutdown signal")
	s.Stop()
}

// GetRouter returns the underlying chi router for testing purposes.
func (s *Server) GetRouter() *chi.Mux {
	return s.router
}

// GetProviders returns the providers map for testing purposes.
func (s *Server) GetProviders() map[string]providers.Provider {
	return s.providers
}

// initializeProviders creates and configures all provider instances.
func initializeProviders(configs map[string]providers.ProviderConfig, logger *zap.Logger) (map[string]providers.Provider, error) {
	providersMap := make(map[string]providers.Provider)

	for name, config := range configs {
		if !config.Enabled {
			continue
		}

		var provider providers.Provider

		switch name {
		case "openai":
			provider = providers.NewOpenAIProvider(config)
		case "anthropic":
			provider = providers.NewAnthropicProvider(config)
		default:
			logger.Warn("Unknown provider type", zap.String("provider", name))
			continue
		}

		providersMap[name] = provider
		logger.Info("Initialized provider", zap.String("name", name))
	}

	return providersMap, nil
}

// initializeRoutingPolicy creates and configures the routing policy.
func initializeRoutingPolicy(config struct {
	Type   string                 `mapstructure:"type"`
	Config map[string]interface{} `mapstructure:"config"`
}, logger *zap.Logger) (policies.RoutingPolicy, error) {
	switch config.Type {
	case "cost_based":
		return policies.NewCostBasedPolicy(), nil
	case "failover":
		// Extract failover configuration
		primary, _ := config.Config["primary_provider"].(string)
		backups, _ := config.Config["backup_providers"].([]string)
		return policies.NewFailoverPolicy(primary, backups), nil
	default:
		logger.Warn("Unknown routing policy, using cost-based", zap.String("policy", config.Type))
		return policies.NewCostBasedPolicy(), nil
	}
}
