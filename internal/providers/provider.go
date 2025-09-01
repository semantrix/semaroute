package providers

import (
	"context"
	"time"

	"github.com/semantrix/semaroute/internal/models"
)

// Provider defines the interface that all LLM providers must implement.
type Provider interface {
	// GetName returns the unique name identifier for this provider.
	GetName() string

	// GetModels returns the list of available models for this provider.
	GetModels() ([]string, error)

	// GetHealth returns the current health status of this provider.
	GetHealth() models.HealthStatus

	// IsHealthy returns true if the provider is currently healthy and available.
	IsHealthy() bool

	// SetHealth updates the health status of this provider.
	SetHealth(healthy bool, latency time.Duration, err string)

	// GetCostEstimate returns an estimated cost for the given request.
	GetCostEstimate(req models.ChatRequest) (float64, error)

	// GetLatencyEstimate returns an estimated latency for the given request.
	GetLatencyEstimate(req models.ChatRequest) (time.Duration, error)

	// CreateChatCompletion creates a synchronous chat completion.
	CreateChatCompletion(ctx context.Context, req models.ChatRequest) (*models.ChatResponse, error)

	// CreateChatCompletionStream creates a streaming chat completion.
	CreateChatCompletionStream(ctx context.Context, req models.ChatRequest) (<-chan models.StreamResponse, error)

	// Close performs any necessary cleanup when the provider is no longer needed.
	Close() error
}

// ProviderConfig holds common configuration for all providers.
type ProviderConfig struct {
	Name                string        `mapstructure:"name"`
	APIKey              string        `mapstructure:"api_key"`
	BaseURL             string        `mapstructure:"base_url"`
	Timeout             time.Duration `mapstructure:"timeout"`
	MaxRetries          int           `mapstructure:"max_retries"`
	RetryDelay          time.Duration `mapstructure:"retry_delay"`
	HealthCheckURL      string        `mapstructure:"health_check_url"`
	HealthCheckInterval time.Duration `mapstructure:"health_check_interval"`
	Enabled             bool          `mapstructure:"enabled"`
}

// BaseProvider provides common functionality for all providers.
type BaseProvider struct {
	config ProviderConfig
	health models.HealthStatus
	models []string
}

// NewBaseProvider creates a new base provider with the given configuration.
func NewBaseProvider(config ProviderConfig) *BaseProvider {
	return &BaseProvider{
		config: config,
		health: models.HealthStatus{
			Healthy:   true,
			LastCheck: time.Now(),
		},
	}
}

// GetName returns the provider name.
func (p *BaseProvider) GetName() string {
	return p.config.Name
}

// GetHealth returns the current health status.
func (p *BaseProvider) GetHealth() models.HealthStatus {
	return p.health
}

// IsHealthy returns true if the provider is healthy.
func (p *BaseProvider) IsHealthy() bool {
	return p.health.Healthy
}

// SetHealth updates the health status.
func (p *BaseProvider) SetHealth(healthy bool, latency time.Duration, err string) {
	p.health.Healthy = healthy
	p.health.Latency = latency
	p.health.LastCheck = time.Now()
	p.health.Error = err
}

// GetConfig returns the provider configuration.
func (p *BaseProvider) GetConfig() ProviderConfig {
	return p.config
}

// Close performs cleanup for the base provider.
func (p *BaseProvider) Close() error {
	// Base implementation does nothing
	return nil
}
