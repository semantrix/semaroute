package policies

import (
	"context"
	"fmt"
	"time"

	"github.com/semantrix/semaroute/internal/models"
	"github.com/semantrix/semaroute/internal/providers"
)

// FailoverPolicy implements primary/backup provider routing with automatic fallback.
type FailoverPolicy struct {
	*BasePolicy
	primaryProvider   string
	backupProviders  []string
	failoverDelay    time.Duration
	healthCheckInterval time.Duration
	lastFailover     time.Time
}

// NewFailoverPolicy creates a new failover routing policy.
func NewFailoverPolicy(primaryProvider string, backupProviders []string) *FailoverPolicy {
	return &FailoverPolicy{
		BasePolicy: NewBasePolicy(
			"failover",
			"Routes requests to primary provider with automatic failover to backup providers",
		),
		primaryProvider:    primaryProvider,
		backupProviders:   backupProviders,
		failoverDelay:     30 * time.Second, // Wait before trying primary again
		healthCheckInterval: 10 * time.Second,
		lastFailover:      time.Time{},
	}
}

// DecideRoute selects the best provider based on failover logic.
func (p *FailoverPolicy) DecideRoute(ctx context.Context, req models.ChatRequest, availableProviders map[string]providers.Provider) (RoutingDecision, error) {
	if err := p.ValidateRequest(req); err != nil {
		return RoutingDecision{}, fmt.Errorf("invalid request: %w", err)
	}

	// Check if primary provider is available and healthy
	if p.shouldUsePrimary() {
		if provider, exists := availableProviders[p.primaryProvider]; exists && provider.IsHealthy() {
			if p.providerSupportsModel(provider, req.Model) {
				decision := RoutingDecision{
					ProviderName: p.primaryProvider,
					Model:        req.Model,
					Reason:       "Primary provider is healthy and available",
					Confidence:   1.0,
					Fallback:     false,
				}
				p.UpdateMetrics(decision, true, 0)
				return decision, nil
			}
		}
	}

	// Try backup providers in order
	for _, backupName := range p.backupProviders {
		if provider, exists := availableProviders[backupName]; exists && provider.IsHealthy() {
			if p.providerSupportsModel(provider, req.Model) {
				decision := RoutingDecision{
					ProviderName: backupName,
					Model:        req.Model,
					Reason:       fmt.Sprintf("Using backup provider %s (primary unavailable)", backupName),
					Confidence:   0.8,
					Fallback:     true,
				}
				p.UpdateMetrics(decision, true, 0)
				return decision, nil
			}
		}
	}

	// If we get here, no providers are available
	return RoutingDecision{}, fmt.Errorf("no available providers for model %s", req.Model)
}

// shouldUsePrimary determines if we should try the primary provider.
func (p *FailoverPolicy) shouldUsePrimary() bool {
	// If we've never failed over, use primary
	if p.lastFailover.IsZero() {
		return true
	}

	// Check if enough time has passed since last failover
	return time.Since(p.lastFailover) > p.failoverDelay
}

// MarkFailover records that a failover occurred.
func (p *FailoverPolicy) MarkFailover(providerName string) {
	if providerName == p.primaryProvider {
		p.lastFailover = time.Now()
	}
}

// SetFailoverDelay sets the delay before retrying the primary provider.
func (p *FailoverPolicy) SetFailoverDelay(delay time.Duration) {
	p.failoverDelay = delay
}

// GetFailoverDelay returns the current failover delay.
func (p *FailoverPolicy) GetFailoverDelay() time.Duration {
	return p.failoverDelay
}

// SetPrimaryProvider sets the primary provider.
func (p *FailoverPolicy) SetPrimaryProvider(providerName string) {
	p.primaryProvider = providerName
	p.lastFailover = time.Time{} // Reset failover timer
}

// GetPrimaryProvider returns the current primary provider.
func (p *FailoverPolicy) GetPrimaryProvider() string {
	return p.primaryProvider
}

// SetBackupProviders sets the list of backup providers.
func (p *FailoverPolicy) SetBackupProviders(providers []string) {
	p.backupProviders = providers
}

// GetBackupProviders returns the current backup providers.
func (p *FailoverPolicy) GetBackupProviders() []string {
	return p.backupProviders
}

// GetLastFailover returns when the last failover occurred.
func (p *FailoverPolicy) GetLastFailover() time.Time {
	return p.lastFailover
}

// IsInFailoverMode returns true if we're currently using backup providers.
func (p *FailoverPolicy) IsInFailoverMode() bool {
	return !p.shouldUsePrimary()
}
