package models

import (
	"time"
)

// ChatRequest represents a unified chat completion request.
type ChatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Stream      bool      `json:"stream,omitempty"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Temperature float64   `json:"temperature,omitempty"`
	TopP        float64   `json:"top_p,omitempty"`
	TopK        int       `json:"top_k,omitempty"`
	Stop        []string  `json:"stop,omitempty"`
	PresencePenalty float64 `json:"presence_penalty,omitempty"`
	FrequencyPenalty float64 `json:"frequency_penalty,omitempty"`
	User        string    `json:"user,omitempty"`
	RequestID   string    `json:"request_id,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// Message represents a single message in a conversation.
type Message struct {
	Role      string `json:"role"`
	Content   string `json:"content"`
	Name      string `json:"name,omitempty"`
	Timestamp time.Time `json:"timestamp,omitempty"`
}

// ChatResponse represents a unified successful response.
type ChatResponse struct {
	ID      string   `json:"id"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
	Created int64    `json:"created"`
	Provider string  `json:"provider"`
	RequestID string `json:"request_id,omitempty"`
}

// Choice represents a single completion choice.
type Choice struct {
	Index   int     `json:"index"`
	Message Message `json:"message"`
	FinishReason string `json:"finish_reason"`
}

// Usage represents token usage statistics.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// StreamResponse represents a streaming response chunk.
type StreamResponse struct {
	ID      string   `json:"id"`
	Model   string   `json:"model"`
	Choices []StreamChoice `json:"choices"`
	Created int64    `json:"created"`
	Provider string  `json:"provider"`
	RequestID string `json:"request_id,omitempty"`
}

// StreamChoice represents a streaming choice.
type StreamChoice struct {
	Index   int     `json:"index"`
	Delta   Message `json:"delta"`
	FinishReason string `json:"finish_reason,omitempty"`
}

// ProviderError represents a standardized error from any provider.
type ProviderError struct {
	StatusCode int    `json:"status_code"`
	Err        error  `json:"error"`
	Provider   string `json:"provider"`
	RequestID  string `json:"request_id,omitempty"`
	Retryable  bool   `json:"retryable"`
}

// Error implements the error interface.
func (e *ProviderError) Error() string {
	return e.Err.Error()
}

// Unwrap returns the underlying error.
func (e *ProviderError) Unwrap() error {
	return e.Err
}

// HealthStatus represents the health status of a provider.
type HealthStatus struct {
	Healthy   bool      `json:"healthy"`
	Latency   time.Duration `json:"latency"`
	LastCheck time.Time `json:"last_check"`
	Error     string    `json:"error,omitempty"`
}

// RoutingRequest represents a request for routing decision.
type RoutingRequest struct {
	Request     ChatRequest `json:"request"`
	UserID      string      `json:"user_id,omitempty"`
	CostLimit   float64     `json:"cost_limit,omitempty"`
	LatencyRequirement time.Duration `json:"latency_requirement,omitempty"`
	Priority    string      `json:"priority,omitempty"`
}

// RoutingResponse represents the routing decision.
type RoutingResponse struct {
	ProviderName string    `json:"provider_name"`
	Model        string    `json:"model"`
	Reason       string    `json:"reason"`
	EstimatedCost float64  `json:"estimated_cost,omitempty"`
	EstimatedLatency time.Duration `json:"estimated_latency,omitempty"`
	Confidence   float64   `json:"confidence"`
}
