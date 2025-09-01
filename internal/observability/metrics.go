package observability

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	otelprometheus "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.uber.org/zap"
)

// MetricsConfig holds configuration for metrics collection.
type MetricsConfig struct {
	Enabled         bool          `mapstructure:"enabled"`
	Port            int           `mapstructure:"port"`
	Path            string        `mapstructure:"path"`
	CollectInterval time.Duration `mapstructure:"collect_interval"`
}

// Metrics provides Prometheus metrics for the router.
type Metrics struct {
	config   MetricsConfig
	logger   *zap.Logger
	registry *prometheus.Registry
	exporter *otelprometheus.Exporter
	provider *metric.MeterProvider

	// Request metrics
	requestsTotal    *prometheus.CounterVec
	requestsDuration *prometheus.HistogramVec
	requestsErrors   *prometheus.CounterVec

	// Provider metrics
	providerHealth  *prometheus.GaugeVec
	providerLatency *prometheus.HistogramVec
	providerErrors  *prometheus.CounterVec

	// Routing metrics
	routingDecisions *prometheus.CounterVec
	routingLatency   *prometheus.HistogramVec

	// Cache metrics (for future use)
	cacheHits   *prometheus.CounterVec
	cacheMisses *prometheus.CounterVec
	cacheSize   *prometheus.GaugeVec
}

// NewMetrics creates a new metrics instance.
func NewMetrics(config MetricsConfig, logger *zap.Logger) (*Metrics, error) {
	// Create Prometheus registry
	registry := prometheus.NewRegistry()

	// Create OpenTelemetry Prometheus exporter
	exporter, err := otelprometheus.New()
	if err != nil {
		return nil, err
	}

	// Create meter provider
	provider := metric.NewMeterProvider(metric.WithReader(exporter))

	// Create metrics instance
	m := &Metrics{
		config:   config,
		logger:   logger,
		registry: registry,
		exporter: exporter,
		provider: provider,
	}

	// Initialize metrics
	if err := m.initMetrics(); err != nil {
		return nil, err
	}

	return m, nil
}

// initMetrics initializes all Prometheus metrics.
func (m *Metrics) initMetrics() error {
	// Request metrics
	m.requestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "semaroute_requests_total",
			Help: "Total number of requests processed",
		},
		[]string{"method", "endpoint", "status_code"},
	)

	m.requestsDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "semaroute_request_duration_seconds",
			Help:    "Request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint"},
	)

	m.requestsErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "semaroute_request_errors_total",
			Help: "Total number of request errors",
		},
		[]string{"method", "endpoint", "error_type"},
	)

	// Provider metrics
	m.providerHealth = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "semaroute_provider_health",
			Help: "Provider health status (1 = healthy, 0 = unhealthy)",
		},
		[]string{"provider_name"},
	)

	m.providerLatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "semaroute_provider_latency_seconds",
			Help:    "Provider response latency in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"provider_name", "model"},
	)

	m.providerErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "semaroute_provider_errors_total",
			Help: "Total number of provider errors",
		},
		[]string{"provider_name", "error_type"},
	)

	// Routing metrics
	m.routingDecisions = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "semaroute_routing_decisions_total",
			Help: "Total number of routing decisions made",
		},
		[]string{"policy_name", "provider_name", "model"},
	)

	m.routingLatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "semaroute_routing_latency_seconds",
			Help:    "Routing decision latency in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"policy_name"},
	)

	// Cache metrics
	m.cacheHits = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "semaroute_cache_hits_total",
			Help: "Total number of cache hits",
		},
		[]string{"cache_type"},
	)

	m.cacheMisses = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "semaroute_cache_misses_total",
			Help: "Total number of cache misses",
		},
		[]string{"cache_type"},
	)

	m.cacheSize = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "semaroute_cache_size",
			Help: "Current cache size",
		},
		[]string{"cache_type"},
	)

	// Register all metrics
	metrics := []prometheus.Collector{
		m.requestsTotal,
		m.requestsDuration,
		m.requestsErrors,
		m.providerHealth,
		m.providerLatency,
		m.providerErrors,
		m.routingDecisions,
		m.routingLatency,
		m.cacheHits,
		m.cacheMisses,
		m.cacheSize,
	}

	for _, metric := range metrics {
		if err := m.registry.Register(metric); err != nil {
			return err
		}
	}

	return nil
}

// RecordRequest records metrics for an HTTP request.
func (m *Metrics) RecordRequest(method, endpoint string, statusCode int, duration time.Duration) {
	statusStr := strconv.Itoa(statusCode)

	m.requestsTotal.WithLabelValues(method, endpoint, statusStr).Inc()
	m.requestsDuration.WithLabelValues(method, endpoint).Observe(duration.Seconds())
}

// RecordRequestError records metrics for a request error.
func (m *Metrics) RecordRequestError(method, endpoint, errorType string) {
	m.requestsErrors.WithLabelValues(method, endpoint, errorType).Inc()
}

// RecordProviderHealth updates the health status of a provider.
func (m *Metrics) RecordProviderHealth(providerName string, healthy bool) {
	value := 0.0
	if healthy {
		value = 1.0
	}
	m.providerHealth.WithLabelValues(providerName).Set(value)
}

// RecordProviderLatency records the response latency of a provider.
func (m *Metrics) RecordProviderLatency(providerName, model string, duration time.Duration) {
	m.providerLatency.WithLabelValues(providerName, model).Observe(duration.Seconds())
}

// RecordProviderError records an error from a provider.
func (m *Metrics) RecordProviderError(providerName, errorType string) {
	m.providerErrors.WithLabelValues(providerName, errorType).Inc()
}

// RecordRoutingDecision records a routing decision made by a policy.
func (m *Metrics) RecordRoutingDecision(policyName, providerName, model string) {
	m.routingDecisions.WithLabelValues(policyName, providerName, model).Inc()
}

// RecordRoutingLatency records the time taken to make a routing decision.
func (m *Metrics) RecordRoutingLatency(policyName string, duration time.Duration) {
	m.routingLatency.WithLabelValues(policyName).Observe(duration.Seconds())
}

// RecordCacheHit records a cache hit.
func (m *Metrics) RecordCacheHit(cacheType string) {
	m.cacheHits.WithLabelValues(cacheType).Inc()
}

// RecordCacheMiss records a cache miss.
func (m *Metrics) RecordCacheMiss(cacheType string) {
	m.cacheMisses.WithLabelValues(cacheType).Inc()
}

// RecordCacheSize records the current size of a cache.
func (m *Metrics) RecordCacheSize(cacheType string, size int) {
	m.cacheSize.WithLabelValues(cacheType).Set(float64(size))
}

// GetRegistry returns the Prometheus registry.
func (m *Metrics) GetRegistry() *prometheus.Registry {
	return m.registry
}

// GetMeterProvider returns the OpenTelemetry meter provider.
func (m *Metrics) GetMeterProvider() *metric.MeterProvider {
	return m.provider
}

// StartMetricsServer starts the metrics HTTP server.
func (m *Metrics) StartMetricsServer(ctx context.Context) error {
	if !m.config.Enabled {
		m.logger.Info("Metrics server disabled")
		return nil
	}

	mux := http.NewServeMux()
	mux.Handle(m.config.Path, promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{}))

	server := &http.Server{
		Addr:    ":" + strconv.Itoa(m.config.Port),
		Handler: mux,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			m.logger.Error("Metrics server error", zap.Error(err))
		}
	}()

	m.logger.Info("Metrics server started",
		zap.Int("port", m.config.Port),
		zap.String("path", m.config.Path))

	// Wait for context cancellation
	<-ctx.Done()

	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		m.logger.Error("Error shutting down metrics server", zap.Error(err))
	}

	return nil
}
