package v1

import (
	"time"
)

// ChatCompletionRequest represents a chat completion request from a client.
type ChatCompletionRequest struct {
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
}

// Message represents a single message in a conversation.
type Message struct {
	Role      string `json:"role"`
	Content   string `json:"content"`
	Name      string `json:"name,omitempty"`
	Timestamp time.Time `json:"timestamp,omitempty"`
}

// ChatCompletionResponse represents a successful chat completion response.
type ChatCompletionResponse struct {
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

// ErrorResponse represents an error response from the API.
type ErrorResponse struct {
	Error   ErrorDetails `json:"error"`
	RequestID string     `json:"request_id,omitempty"`
}

// ErrorDetails provides detailed error information.
type ErrorDetails struct {
	Type        string `json:"type"`
	Message     string `json:"message"`
	StatusCode  int    `json:"status_code"`
	Provider    string `json:"provider,omitempty"`
	Retryable   bool   `json:"retryable"`
	Details     map[string]interface{} `json:"details,omitempty"`
}

// HealthResponse represents the health status of the service.
type HealthResponse struct {
	Status    string                 `json:"status"`
	Timestamp time.Time              `json:"timestamp"`
	Uptime    time.Duration          `json:"uptime"`
	Providers map[string]ProviderHealth `json:"providers"`
	Version   string                 `json:"version"`
}

// ProviderHealth represents the health status of a provider.
type ProviderHealth struct {
	Status    string        `json:"status"`
	Latency   time.Duration `json:"latency"`
	LastCheck time.Time     `json:"last_check"`
	Error     string        `json:"error,omitempty"`
}

// ModelsResponse represents the available models from all providers.
type ModelsResponse struct {
	Models   []ModelInfo `json:"models"`
	Total    int         `json:"total"`
	Providers []string   `json:"providers"`
}

// ModelInfo represents information about a specific model.
type ModelInfo struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Provider    string   `json:"provider"`
	Type        string   `json:"type"`
	ContextSize int      `json:"context_size,omitempty"`
	MaxTokens   int      `json:"max_tokens,omitempty"`
	SupportedFeatures []string `json:"supported_features,omitempty"`
}

// RoutingInfoResponse represents information about routing decisions.
type RoutingInfoResponse struct {
	RequestID      string         `json:"request_id"`
	RoutingPolicy  string         `json:"routing_policy"`
	Decision       RoutingDecision `json:"decision"`
	Alternatives   []RoutingDecision `json:"alternatives,omitempty"`
	Timestamp     time.Time       `json:"timestamp"`
}

// RoutingDecision represents a routing decision made by the system.
type RoutingDecision struct {
	ProviderName    string    `json:"provider_name"`
	Model           string    `json:"model"`
	Reason          string    `json:"reason"`
	EstimatedCost   float64   `json:"estimated_cost,omitempty"`
	EstimatedLatency time.Duration `json:"estimated_latency,omitempty"`
	Confidence      float64   `json:"confidence"`
	Fallback        bool      `json:"fallback"`
}

// MetricsResponse represents system metrics.
type MetricsResponse struct {
	Requests     RequestMetrics     `json:"requests"`
	Providers    ProviderMetrics    `json:"providers"`
	Routing      RoutingMetrics     `json:"routing"`
	Cache        CacheMetrics       `json:"cache"`
	Timestamp    time.Time          `json:"timestamp"`
}

// RequestMetrics represents request-related metrics.
type RequestMetrics struct {
	Total        int64   `json:"total"`
	Successful   int64   `json:"successful"`
	Failed       int64   `json:"failed"`
	AverageLatency time.Duration `json:"average_latency"`
	ErrorRate    float64 `json:"error_rate"`
}

// ProviderMetrics represents provider-related metrics.
type ProviderMetrics struct {
	Total        int64   `json:"total"`
	Healthy      int64   `json:"healthy"`
	Unhealthy   int64   `json:"unhealthy"`
	AverageLatency time.Duration `json:"average_latency"`
	TotalErrors  int64   `json:"total_errors"`
}

// RoutingMetrics represents routing-related metrics.
type RoutingMetrics struct {
	TotalDecisions int64   `json:"total_decisions"`
	AverageLatency time.Duration `json:"average_latency"`
	PolicyUsage    map[string]int64 `json:"policy_usage"`
}

// CacheMetrics represents cache-related metrics.
type CacheMetrics struct {
	Hits         int64   `json:"hits"`
	Misses       int64   `json:"misses"`
	HitRate      float64 `json:"hit_rate"`
	Size         int64   `json:"size"`
	MaxSize      int64   `json:"max_size"`
}
