package health

import (
	"fmt"
	"sync"
	"time"

	"github.com/semantrix/semaroute/internal/models"
	"github.com/semantrix/semaroute/internal/providers"
	"go.uber.org/zap"
)

// HealthChecker monitors the health of all providers.
type HealthChecker struct {
	providers     map[string]providers.Provider
	checkInterval time.Duration
	timeout       time.Duration
	stopChan      chan struct{}
	wg            sync.WaitGroup
	logger        *zap.Logger
	metrics       map[string]*ProviderMetrics
	metricsMutex  sync.RWMutex
}

// ProviderMetrics tracks health metrics for a provider.
type ProviderMetrics struct {
	TotalChecks      int64         `json:"total_checks"`
	SuccessfulChecks int64         `json:"successful_checks"`
	FailedChecks     int64         `json:"failed_checks"`
	LastCheck        time.Time     `json:"last_check"`
	LastLatency      time.Duration `json:"last_latency"`
	AverageLatency   time.Duration `json:"average_latency"`
	Uptime           float64       `json:"uptime"`
}

// NewHealthChecker creates a new health checker instance.
func NewHealthChecker(checkInterval, timeout time.Duration, logger *zap.Logger) *HealthChecker {
	return &HealthChecker{
		providers:     make(map[string]providers.Provider),
		checkInterval: checkInterval,
		timeout:       timeout,
		stopChan:      make(chan struct{}),
		logger:        logger,
		metrics:       make(map[string]*ProviderMetrics),
	}
}

// AddProvider adds a provider to be monitored.
func (hc *HealthChecker) AddProvider(name string, provider providers.Provider) {
	hc.metricsMutex.Lock()
	defer hc.metricsMutex.Unlock()

	hc.providers[name] = provider
	hc.metrics[name] = &ProviderMetrics{
		LastCheck: time.Now(),
	}
}

// RemoveProvider removes a provider from monitoring.
func (hc *HealthChecker) RemoveProvider(name string) {
	delete(hc.providers, name)
	hc.metricsMutex.Lock()
	delete(hc.metrics, name)
	hc.metricsMutex.Unlock()
}

// Start begins the health checking process.
func (hc *HealthChecker) Start() {
	hc.wg.Add(1)
	go hc.run()
	hc.logger.Info("Health checker started", zap.Duration("interval", hc.checkInterval))
}

// Stop stops the health checking process.
func (hc *HealthChecker) Stop() {
	close(hc.stopChan)
	hc.wg.Wait()
	hc.logger.Info("Health checker stopped")
}

// run is the main health checking loop.
func (hc *HealthChecker) run() {
	defer hc.wg.Done()

	ticker := time.NewTicker(hc.checkInterval)
	defer ticker.Stop()

	// Run initial health check
	hc.checkAllProviders()

	for {
		select {
		case <-ticker.C:
			hc.checkAllProviders()
		case <-hc.stopChan:
			return
		}
	}
}

// checkAllProviders performs health checks on all registered providers.
func (hc *HealthChecker) checkAllProviders() {
	var wg sync.WaitGroup

	hc.metricsMutex.RLock()
	providersCopy := make(map[string]providers.Provider)
	for name, provider := range hc.providers {
		providersCopy[name] = provider
	}
	hc.metricsMutex.RUnlock()

	for name, provider := range providersCopy {
		wg.Add(1)
		go func(providerName string, p providers.Provider) {
			defer wg.Done()
			hc.checkProvider(providerName, p)
		}(name, provider)
	}

	wg.Wait()
}

// checkProvider performs a health check on a single provider.
func (hc *HealthChecker) checkProvider(name string, provider providers.Provider) {
	start := time.Now()

	// Try to get models as a health check
	_, err := provider.GetModels()
	latency := time.Since(start)

	hc.metricsMutex.Lock()
	metrics := hc.metrics[name]
	if metrics == nil {
		metrics = &ProviderMetrics{}
		hc.metrics[name] = metrics
	}

	metrics.TotalChecks++
	metrics.LastCheck = time.Now()
	metrics.LastLatency = latency

	if err == nil {
		// Successful health check
		metrics.SuccessfulChecks++
		// Update provider health status
		provider.SetHealth(true, latency, "")
		hc.logger.Debug("Provider health check successful",
			zap.String("provider", name),
			zap.Duration("latency", latency))
	} else {
		// Failed health check
		metrics.FailedChecks++
		// Update provider health status
		provider.SetHealth(false, latency, err.Error())
		hc.logger.Warn("Provider health check failed",
			zap.String("provider", name),
			zap.Duration("latency", latency),
			zap.Error(err))
	}

	// Calculate uptime percentage
	if metrics.TotalChecks > 0 {
		metrics.Uptime = float64(metrics.SuccessfulChecks) / float64(metrics.TotalChecks) * 100
	}

	// Update average latency (simple moving average)
	if metrics.SuccessfulChecks > 0 {
		if metrics.AverageLatency == 0 {
			metrics.AverageLatency = latency
		} else {
			// Simple exponential moving average
			alpha := 0.1
			metrics.AverageLatency = time.Duration(
				float64(metrics.AverageLatency)*(1-alpha) + float64(latency)*alpha,
			)
		}
	}

	hc.metricsMutex.Unlock()
}

// GetProviderHealth returns the current health status of a provider.
func (hc *HealthChecker) GetProviderHealth(name string) (models.HealthStatus, error) {
	provider, exists := hc.providers[name]
	if !exists {
		return models.HealthStatus{}, fmt.Errorf("provider %s not found", name)
	}

	return provider.GetHealth(), nil
}

// GetAllProviderHealth returns health status for all providers.
func (hc *HealthChecker) GetAllProviderHealth() map[string]models.HealthStatus {
	result := make(map[string]models.HealthStatus)

	for name, provider := range hc.providers {
		result[name] = provider.GetHealth()
	}

	return result
}

// GetProviderMetrics returns metrics for a specific provider.
func (hc *HealthChecker) GetProviderMetrics(name string) (*ProviderMetrics, error) {
	hc.metricsMutex.RLock()
	defer hc.metricsMutex.RUnlock()

	metrics, exists := hc.metrics[name]
	if !exists {
		return nil, fmt.Errorf("metrics for provider %s not found", name)
	}

	return metrics, nil
}

// GetAllProviderMetrics returns metrics for all providers.
func (hc *HealthChecker) GetAllProviderMetrics() map[string]*ProviderMetrics {
	hc.metricsMutex.RLock()
	defer hc.metricsMutex.RUnlock()

	result := make(map[string]*ProviderMetrics)
	for name, metrics := range hc.metrics {
		result[name] = metrics
	}

	return result
}

// ForceHealthCheck triggers an immediate health check for all providers.
func (hc *HealthChecker) ForceHealthCheck() {
	hc.logger.Info("Forcing health check for all providers")
	hc.checkAllProviders()
}

// SetCheckInterval updates the health check interval.
func (hc *HealthChecker) SetCheckInterval(interval time.Duration) {
	hc.checkInterval = interval
	hc.logger.Info("Health check interval updated", zap.Duration("new_interval", interval))
}

// GetCheckInterval returns the current health check interval.
func (hc *HealthChecker) GetCheckInterval() time.Duration {
	return hc.checkInterval
}
