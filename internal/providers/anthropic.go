package providers

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/semantrix/semaroute/internal/models"
	"github.com/sethvargo/go-retry"
)

// AnthropicProvider implements the Provider interface for Anthropic.
type AnthropicProvider struct {
	*BaseProvider
	client *http.Client
}

// NewAnthropicProvider creates a new Anthropic provider instance.
func NewAnthropicProvider(config ProviderConfig) Provider {
	client := &http.Client{
		Timeout: config.Timeout,
	}

	return &AnthropicProvider{
		BaseProvider: NewBaseProvider(config),
		client:       client,
	}
}

// GetModels returns the list of available Anthropic models.
func (p *AnthropicProvider) GetModels() ([]string, error) {
	// For now, return a static list. In production, this would call the Anthropic models endpoint.
	return []string{
		"claude-3-opus-20240229",
		"claude-3-sonnet-20240229",
		"claude-3-haiku-20240307",
		"claude-2.1",
		"claude-2.0",
		"claude-instant-1.2",
	}, nil
}

// GetCostEstimate returns an estimated cost for the request.
func (p *AnthropicProvider) GetCostEstimate(req models.ChatRequest) (float64, error) {
	// Simplified cost estimation based on model and token count
	// In production, this would use actual pricing data
	model := req.Model
	var costPer1kTokens float64

	switch {
	case strings.Contains(model, "opus"):
		costPer1kTokens = 0.015
	case strings.Contains(model, "sonnet"):
		costPer1kTokens = 0.003
	case strings.Contains(model, "haiku"):
		costPer1kTokens = 0.00025
	case strings.Contains(model, "claude-2"):
		costPer1kTokens = 0.008
	case strings.Contains(model, "claude-instant"):
		costPer1kTokens = 0.0008
	default:
		costPer1kTokens = 0.005
	}

	// Estimate tokens (rough approximation)
	estimatedTokens := len(req.Messages) * 100 // Very rough estimate
	if req.MaxTokens > 0 {
		estimatedTokens += req.MaxTokens
	}

	return float64(estimatedTokens) * costPer1kTokens / 1000, nil
}

// GetLatencyEstimate returns an estimated latency for the request.
func (p *AnthropicProvider) GetLatencyEstimate(req models.ChatRequest) (time.Duration, error) {
	// Base latency + per-token latency
	baseLatency := 300 * time.Millisecond
	perTokenLatency := 15 * time.Millisecond

	estimatedTokens := len(req.Messages) * 100
	if req.MaxTokens > 0 {
		estimatedTokens += req.MaxTokens
	}

	return baseLatency + time.Duration(estimatedTokens)*perTokenLatency, nil
}

// CreateChatCompletion creates a chat completion using Anthropic's API.
func (p *AnthropicProvider) CreateChatCompletion(ctx context.Context, req models.ChatRequest) (*models.ChatResponse, error) {
	// Convert to Anthropic format
	anthropicReq := p.convertToAnthropicRequest(req)

	// Implement retry logic
	var response *models.ChatResponse
	err := retry.Do(ctx, retry.WithMaxRetries(uint64(p.config.MaxRetries), retry.NewConstant(p.config.RetryDelay)), func(ctx context.Context) error {
		var err error
		response, err = p.makeAnthropicRequest(ctx, anthropicReq)
		if err != nil {
			// Check if error is retryable
			if p.isRetryableError(err) {
				return retry.RetryableError(err)
			}
			return err
		}
		return nil
	})

	if err != nil {
		return nil, &models.ProviderError{
			StatusCode: 500,
			Err:        err,
			Provider:   p.GetName(),
			RequestID:  req.RequestID,
			Retryable:  p.isRetryableError(err),
		}
	}

	return response, nil
}

// CreateChatCompletionStream creates a streaming chat completion.
func (p *AnthropicProvider) CreateChatCompletionStream(ctx context.Context, req models.ChatRequest) (<-chan models.StreamResponse, error) {
	// For now, return an error indicating streaming is not yet implemented
	// In production, this would implement Server-Sent Events or similar
	return nil, fmt.Errorf("streaming not yet implemented for Anthropic provider")
}

// Close performs cleanup for the Anthropic provider.
func (p *AnthropicProvider) Close() error {
	if p.client != nil {
		p.client.CloseIdleConnections()
	}
	return p.BaseProvider.Close()
}

// convertToAnthropicRequest converts our unified request to Anthropic format.
func (p *AnthropicProvider) convertToAnthropicRequest(req models.ChatRequest) map[string]interface{} {
	// Convert messages to Anthropic format
	messages := make([]map[string]interface{}, len(req.Messages))
	for i, msg := range req.Messages {
		// Anthropic uses "user" and "assistant" roles
		role := msg.Role
		if role == "system" {
			role = "user" // Anthropic doesn't have a system role, so we use user
		}

		messages[i] = map[string]interface{}{
			"role":    role,
			"content": msg.Content,
		}
	}

	anthropicReq := map[string]interface{}{
		"model":       req.Model,
		"messages":    messages,
		"max_tokens":  req.MaxTokens,
		"temperature": req.Temperature,
	}

	if req.TopP > 0 {
		anthropicReq["top_p"] = req.TopP
	}
	if req.TopK > 0 {
		anthropicReq["top_k"] = req.TopK
	}
	if len(req.Stop) > 0 {
		anthropicReq["stop_sequences"] = req.Stop
	}

	return anthropicReq
}

// makeAnthropicRequest makes the actual HTTP request to Anthropic.
func (p *AnthropicProvider) makeAnthropicRequest(ctx context.Context, req map[string]interface{}) (*models.ChatResponse, error) {
	// This is a placeholder implementation
	// In production, this would make the actual HTTP request to Anthropic's API
	return nil, fmt.Errorf("Anthropic API request not yet implemented")
}

// isRetryableError determines if an error should trigger a retry.
func (p *AnthropicProvider) isRetryableError(err error) bool {
	// Check for retryable error conditions
	// In production, this would check for rate limits, timeouts, etc.
	return false
}
