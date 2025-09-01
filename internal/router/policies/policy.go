package policies

import (
	"context"
	"fmt"
	"time"
	"github.com/semantrix/semaroute/internal/models"
	"github.com/semantrix/semaroute/internal/providers"
)

// RoutingDecision represents the result of a routing policy decision.
type RoutingDecision struct {
	ProviderName string    `json:"provider_name"`
	Model        string    `json:"model"`
	Reason       string    `json:"reason"`
	EstimatedCost float64  `json:"estimated_cost,omitempty"`
	EstimatedLatency time.Duration `json:"estimated_latency,omitempty"`
	Confidence   float64   `json:"confidence"`
	Fallback     bool      `json:"fallback"`
}

// RoutingPolicy defines the interface for intelligent routing strategies.
type RoutingPolicy interface {
	// DecideRoute selects the best provider/model based on the request, cost, health, and latency.
	DecideRoute(ctx context.Context, req models.ChatRequest, availableProviders map[string]providers.Provider) (RoutingDecision, error)
	
	// GetName returns the name of this routing policy.
	GetName() string
	
	// GetDescription returns a description of how this policy works.
	GetDescription() string
	
	// ValidateRequest validates if the request can be handled by this policy.
	ValidateRequest(req models.ChatRequest) error
	
	// UpdateMetrics updates internal metrics after a routing decision.
	UpdateMetrics(decision RoutingDecision, success bool, latency time.Duration)
}

// BasePolicy provides common functionality for all routing policies.
type BasePolicy struct {
	name        string
	description string
	metrics     map[string]interface{}
}

// NewBasePolicy creates a new base policy.
func NewBasePolicy(name, description string) *BasePolicy {
	return &BasePolicy{
		name:        name,
		description: description,
		metrics:     make(map[string]interface{}),
	}
}

// GetName returns the policy name.
func (p *BasePolicy) GetName() string {
	return p.name
}

// GetDescription returns the policy description.
func (p *BasePolicy) GetDescription() string {
	return p.description
}

// ValidateRequest provides a basic validation implementation.
func (p *BasePolicy) ValidateRequest(req models.ChatRequest) error {
	if req.Model == "" {
		return fmt.Errorf("model is required")
	}
	if len(req.Messages) == 0 {
		return fmt.Errorf("at least one message is required")
	}
	return nil
}

// UpdateMetrics provides a basic metrics update implementation.
func (p *BasePolicy) UpdateMetrics(decision RoutingDecision, success bool, latency time.Duration) {
	// In production, this would update Prometheus metrics, etc.
	p.metrics["last_decision"] = decision
	p.metrics["last_success"] = success
	p.metrics["last_latency"] = latency
}

// GetMetrics returns the current metrics for this policy.
func (p *BasePolicy) GetMetrics() map[string]interface{} {
	return p.metrics
}

// Helper function to check if a provider supports the requested model.
func (p *BasePolicy) providerSupportsModel(provider providers.Provider, model string) bool {
	models, err := provider.GetModels()
	if err != nil {
		return false
	}
	
	for _, m := range models {
		if m == model {
			return true
		}
	}
	return false
}

// Helper function to get healthy providers.
func (p *BasePolicy) getHealthyProviders(availableProviders map[string]providers.Provider) map[string]providers.Provider {
	healthy := make(map[string]providers.Provider)
	for name, provider := range availableProviders {
		if provider.IsHealthy() {
			healthy[name] = provider
		}
	}
	return healthy
}
