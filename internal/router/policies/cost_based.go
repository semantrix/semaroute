package policies

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/semantrix/semaroute/internal/models"
	"github.com/semantrix/semaroute/internal/providers"
)

// CostBasedPolicy implements cost-optimized routing.
type CostBasedPolicy struct {
	*BasePolicy
	maxLatencyThreshold time.Duration
	costWeight          float64
	latencyWeight       float64
	healthWeight        float64
}

// NewCostBasedPolicy creates a new cost-based routing policy.
func NewCostBasedPolicy() *CostBasedPolicy {
	return &CostBasedPolicy{
		BasePolicy: NewBasePolicy(
			"cost_based",
			"Routes requests to the most cost-effective provider while considering latency and health",
		),
		maxLatencyThreshold: 5 * time.Second,
		costWeight:          0.6,
		latencyWeight:       0.3,
		healthWeight:        0.1,
	}
}

// DecideRoute selects the best provider based on cost, latency, and health.
func (p *CostBasedPolicy) DecideRoute(ctx context.Context, req models.ChatRequest, availableProviders map[string]providers.Provider) (RoutingDecision, error) {
	if err := p.ValidateRequest(req); err != nil {
		return RoutingDecision{}, fmt.Errorf("invalid request: %w", err)
	}

	// Get only healthy providers
	healthyProviders := p.getHealthyProviders(availableProviders)
	if len(healthyProviders) == 0 {
		return RoutingDecision{}, fmt.Errorf("no healthy providers available")
	}

	// Score each provider
	type providerScore struct {
		name  string
		score float64
		cost  float64
		latency time.Duration
		reason string
	}

	var scores []providerScore

	for name, provider := range healthyProviders {
		// Check if provider supports the requested model
		if !p.providerSupportsModel(provider, req.Model) {
			continue
		}

		// Get cost estimate
		cost, err := provider.GetCostEstimate(req)
		if err != nil {
			continue // Skip this provider if we can't get cost estimate
		}

		// Get latency estimate
		latency, err := provider.GetLatencyEstimate(req)
		if err != nil {
			latency = p.maxLatencyThreshold // Use max threshold as fallback
		}

		// Check if latency is within acceptable bounds
		if latency > p.maxLatencyThreshold {
			continue // Skip providers that are too slow
		}

		// Calculate composite score
		// Lower scores are better (like golf scoring)
		costScore := cost * p.costWeight
		latencyScore := float64(latency.Milliseconds()) / 1000.0 * p.latencyWeight
		healthScore := 0.0 // Healthy providers get 0 penalty
		
		totalScore := costScore + latencyScore + healthScore

		reason := fmt.Sprintf("Cost: $%.4f, Latency: %v, Health: Good", cost, latency)

		scores = append(scores, providerScore{
			name:    name,
			score:   totalScore,
			cost:    cost,
			latency: latency,
			reason:  reason,
		})
	}

	if len(scores) == 0 {
		return RoutingDecision{}, fmt.Errorf("no suitable providers found for model %s", req.Model)
	}

	// Sort by score (ascending - lower is better)
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].score < scores[j].score
	})

	// Select the best provider
	best := scores[0]

	// Calculate confidence based on score difference from next best
	confidence := 1.0
	if len(scores) > 1 {
		scoreDiff := scores[1].score - best.score
		if scoreDiff > 0 {
			confidence = 0.8 + (0.2 * (scoreDiff / best.score))
			if confidence > 1.0 {
				confidence = 1.0
			}
		}
	}

	decision := RoutingDecision{
		ProviderName:      best.name,
		Model:            req.Model,
		Reason:           best.reason,
		EstimatedCost:    best.cost,
		EstimatedLatency: best.latency,
		Confidence:       confidence,
		Fallback:         false,
	}

	// Update metrics
	p.UpdateMetrics(decision, true, 0) // We don't have actual latency yet

	return decision, nil
}

// SetWeights allows customization of the scoring weights.
func (p *CostBasedPolicy) SetWeights(cost, latency, health float64) error {
	total := cost + latency + health
	if total <= 0 {
		return fmt.Errorf("weights must sum to a positive number")
	}

	// Normalize weights
	p.costWeight = cost / total
	p.latencyWeight = latency / total
	p.healthWeight = health / total

	return nil
}

// SetMaxLatencyThreshold sets the maximum acceptable latency.
func (p *CostBasedPolicy) SetMaxLatencyThreshold(threshold time.Duration) {
	p.maxLatencyThreshold = threshold
}

// GetWeights returns the current scoring weights.
func (p *CostBasedPolicy) GetWeights() (cost, latency, health float64) {
	return p.costWeight, p.latencyWeight, p.healthWeight
}
