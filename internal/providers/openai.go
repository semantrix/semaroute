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

// OpenAIProvider implements the Provider interface for OpenAI.
type OpenAIProvider struct {
	*BaseProvider
	client *http.Client
}

// NewOpenAIProvider creates a new OpenAI provider instance.
func NewOpenAIProvider(config ProviderConfig) Provider {
	client := &http.Client{
		Timeout: config.Timeout,
	}

	return &OpenAIProvider{
		BaseProvider: NewBaseProvider(config),
		client:       client,
	}
}

// GetModels returns the list of available OpenAI models.
func (p *OpenAIProvider) GetModels() ([]string, error) {
	// For now, return a static list. In production, this would call the OpenAI models endpoint.
	return []string{
		"gpt-4",
		"gpt-4-turbo-preview",
		"gpt-4-32k",
		"gpt-3.5-turbo",
		"gpt-3.5-turbo-16k",
	}, nil
}

// GetCostEstimate returns an estimated cost for the request.
func (p *OpenAIProvider) GetCostEstimate(req models.ChatRequest) (float64, error) {
	// Simplified cost estimation based on model and token count
	// In production, this would use actual pricing data
	model := req.Model
	var costPer1kTokens float64

	switch {
	case strings.Contains(model, "gpt-4"):
		costPer1kTokens = 0.03
	case strings.Contains(model, "gpt-3.5"):
		costPer1kTokens = 0.002
	default:
		costPer1kTokens = 0.01
	}

	// Estimate tokens (rough approximation)
	estimatedTokens := len(req.Messages) * 100 // Very rough estimate
	if req.MaxTokens > 0 {
		estimatedTokens += req.MaxTokens
	}

	return float64(estimatedTokens) * costPer1kTokens / 1000, nil
}

// GetLatencyEstimate returns an estimated latency for the request.
func (p *OpenAIProvider) GetLatencyEstimate(req models.ChatRequest) (time.Duration, error) {
	// Base latency + per-token latency
	baseLatency := 200 * time.Millisecond
	perTokenLatency := 10 * time.Millisecond

	estimatedTokens := len(req.Messages) * 100
	if req.MaxTokens > 0 {
		estimatedTokens += req.MaxTokens
	}

	return baseLatency + time.Duration(estimatedTokens)*perTokenLatency, nil
}

// CreateChatCompletion creates a chat completion using OpenAI's API.
func (p *OpenAIProvider) CreateChatCompletion(ctx context.Context, req models.ChatRequest) (*models.ChatResponse, error) {
	// Convert to OpenAI format
	openAIReq := p.convertToOpenAIRequest(req)

	// Implement retry logic
	var response *models.ChatResponse
	err := retry.Do(ctx, retry.WithMaxRetries(uint64(p.config.MaxRetries), retry.NewConstant(p.config.RetryDelay)), func(ctx context.Context) error {
		var err error
		response, err = p.makeOpenAIRequest(ctx, openAIReq)
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
func (p *OpenAIProvider) CreateChatCompletionStream(ctx context.Context, req models.ChatRequest) (<-chan models.StreamResponse, error) {
	// For now, return an error indicating streaming is not yet implemented
	// In production, this would implement Server-Sent Events or similar
	return nil, fmt.Errorf("streaming not yet implemented for OpenAI provider")
}

// Close performs cleanup for the OpenAI provider.
func (p *OpenAIProvider) Close() error {
	if p.client != nil {
		p.client.CloseIdleConnections()
	}
	return p.BaseProvider.Close()
}

// convertToOpenAIRequest converts our unified request to OpenAI format.
func (p *OpenAIProvider) convertToOpenAIRequest(req models.ChatRequest) map[string]interface{} {
	// Convert messages to OpenAI format
	messages := make([]map[string]interface{}, len(req.Messages))
	for i, msg := range req.Messages {
		messages[i] = map[string]interface{}{
			"role":    msg.Role,
			"content": msg.Content,
		}
		if msg.Name != "" {
			messages[i]["name"] = msg.Name
		}
	}

	openAIReq := map[string]interface{}{
		"model":       req.Model,
		"messages":    messages,
		"stream":      req.Stream,
		"temperature": req.Temperature,
	}

	if req.MaxTokens > 0 {
		openAIReq["max_tokens"] = req.MaxTokens
	}
	if req.TopP > 0 {
		openAIReq["top_p"] = req.TopP
	}
	if req.TopK > 0 {
		openAIReq["top_k"] = req.TopK
	}
	if len(req.Stop) > 0 {
		openAIReq["stop"] = req.Stop
	}
	if req.PresencePenalty != 0 {
		openAIReq["presence_penalty"] = req.PresencePenalty
	}
	if req.FrequencyPenalty != 0 {
		openAIReq["frequency_penalty"] = req.FrequencyPenalty
	}
	if req.User != "" {
		openAIReq["user"] = req.User
	}

	return openAIReq
}

// makeOpenAIRequest makes the actual HTTP request to OpenAI.
func (p *OpenAIProvider) makeOpenAIRequest(ctx context.Context, req map[string]interface{}) (*models.ChatResponse, error) {
	// This is a placeholder implementation
	// In production, this would make the actual HTTP request to OpenAI's API
	return nil, fmt.Errorf("OpenAI API request not yet implemented")
}

// isRetryableError determines if an error should trigger a retry.
func (p *OpenAIProvider) isRetryableError(err error) bool {
	// Check for retryable error conditions
	// In production, this would check for rate limits, timeouts, etc.
	return false
}
