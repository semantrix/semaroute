package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/semantrix/semaroute/internal/models"
	"github.com/semantrix/semaroute/pkg/api/v1"
	"go.uber.org/zap"
)

// handleHealthCheck handles the health check endpoint.
func (s *Server) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	// Get provider health status
	providerHealth := s.healthChecker.GetAllProviderHealth()
	
	// Convert to API response format
	apiProviderHealth := make(map[string]v1.ProviderHealth)
	for name, health := range providerHealth {
		status := "unhealthy"
		if health.Healthy {
			status = "healthy"
		}
		
		apiProviderHealth[name] = v1.ProviderHealth{
			Status:    status,
			Latency:   health.Latency,
			LastCheck: health.LastCheck,
			Error:     health.Error,
		}
	}

	response := v1.HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now(),
		Uptime:    time.Since(time.Now()), // This should be calculated from server start time
		Providers: apiProviderHealth,
		Version:   "1.0.0", // This should come from build info
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// handleChatCompletion handles chat completion requests.
func (s *Server) handleChatCompletion(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	// Parse request
	var apiReq v1.ChatCompletionRequest
	if err := json.NewDecoder(r.Body).Decode(&apiReq); err != nil {
		s.logger.Error("Failed to decode request", zap.Error(err))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Convert to internal model
	req := models.ChatRequest{
		Model:            apiReq.Model,
		Messages:         convertMessages(apiReq.Messages),
		Stream:           apiReq.Stream,
		MaxTokens:        apiReq.MaxTokens,
		Temperature:      apiReq.Temperature,
		TopP:             apiReq.TopP,
		TopK:             apiReq.TopK,
		Stop:             apiReq.Stop,
		PresencePenalty:  apiReq.PresencePenalty,
		FrequencyPenalty: apiReq.FrequencyPenalty,
		User:             apiReq.User,
		RequestID:        apiReq.RequestID,
		CreatedAt:        time.Now(),
	}

	// Make routing decision
	routingStart := time.Now()
	decision, err := s.routingPolicy.DecideRoute(ctx, req, s.providers)
	if err != nil {
		s.logger.Error("Routing decision failed", zap.Error(err))
		http.Error(w, "Routing failed", http.StatusServiceUnavailable)
		return
	}
	routingDuration := time.Since(routingStart)

	// Record routing metrics
	s.metrics.RecordRoutingDecision(s.routingPolicy.GetName(), decision.ProviderName, decision.Model)
	s.metrics.RecordRoutingLatency(s.routingPolicy.GetName(), routingDuration)

	// Get the selected provider
	provider, exists := s.providers[decision.ProviderName]
	if !exists {
		s.logger.Error("Selected provider not found", zap.String("provider", decision.ProviderName))
		http.Error(w, "Provider not available", http.StatusServiceUnavailable)
		return
	}

	// Execute the request
	start := time.Now()
	var response *models.ChatResponse
	
	if req.Stream {
		// Handle streaming (not yet implemented)
		http.Error(w, "Streaming not yet implemented", http.StatusNotImplemented)
		return
	} else {
		response, err = provider.CreateChatCompletion(ctx, req)
	}
	
	duration := time.Since(start)

	if err != nil {
		// Handle provider errors
		s.logger.Error("Provider request failed", 
			zap.String("provider", decision.ProviderName),
			zap.Error(err))
		
		// Record error metrics
		s.metrics.RecordProviderError(decision.ProviderName, "request_failed")
		
		// Check if we should try a different provider
		if decision.Fallback {
			// Try to find another provider
			// This is a simplified fallback - in production you'd want more sophisticated logic
			for name, p := range s.providers {
				if name != decision.ProviderName && p.IsHealthy() {
					// Try the fallback provider
					response, err = p.CreateChatCompletion(ctx, req)
					if err == nil {
						decision.ProviderName = name
						decision.Reason = "Fallback provider used"
						break
					}
				}
			}
		}

		if err != nil {
			// All providers failed
			errorResponse := v1.ErrorResponse{
				Error: v1.ErrorDetails{
					Type:        "provider_error",
					Message:     "All providers failed",
					StatusCode:  http.StatusServiceUnavailable,
					Retryable:   true,
				},
				RequestID: req.RequestID,
			}
			
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(errorResponse)
			return
		}
	}

	// Record success metrics
	s.metrics.RecordProviderLatency(decision.ProviderName, decision.Model, duration)
	s.metrics.RecordProviderHealth(decision.ProviderName, true)

	// Convert response to API format
	apiResponse := v1.ChatCompletionResponse{
		ID:        response.ID,
		Model:     response.Model,
		Choices:   convertChoices(response.Choices),
		Usage:     convertUsage(response.Usage),
		Created:   response.Created,
		Provider:  decision.ProviderName,
		RequestID: response.RequestID,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(apiResponse)
}

// handleGetModels returns available models from all providers.
func (s *Server) handleGetModels(w http.ResponseWriter, r *http.Request) {
	var allModels []v1.ModelInfo
	var allProviders []string

	for name, provider := range s.providers {
		models, err := provider.GetModels()
		if err != nil {
			s.logger.Warn("Failed to get models from provider", 
				zap.String("provider", name), 
				zap.Error(err))
			continue
		}

		allProviders = append(allProviders, name)
		
		for _, model := range models {
			allModels = append(allModels, v1.ModelInfo{
				ID:       model,
				Name:     model,
				Provider: name,
				Type:     "chat_completion", // This could be more sophisticated
			})
		}
	}

	response := v1.ModelsResponse{
		Models:    allModels,
		Total:     len(allModels),
		Providers: allProviders,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// handleGetRoutingInfo returns information about routing decisions.
func (s *Server) handleGetRoutingInfo(w http.ResponseWriter, r *http.Request) {
	// This endpoint would return routing information for a specific request
	// For now, return basic policy information
	response := v1.RoutingInfoResponse{
		RequestID:     r.URL.Query().Get("request_id"),
		RoutingPolicy: s.routingPolicy.GetName(),
		Decision: v1.RoutingDecision{
			ProviderName: "none",
			Model:        "none",
			Reason:       "No active request",
			Confidence:   0.0,
			Fallback:     false,
		},
		Timestamp: time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// handleGetMetrics returns system metrics.
func (s *Server) handleGetMetrics(w http.ResponseWriter, r *http.Request) {
	// This would aggregate metrics from various sources
	// For now, return basic structure
	response := v1.MetricsResponse{
		Requests: v1.RequestMetrics{
			Total:     0,
			Successful: 0,
			Failed:    0,
			ErrorRate: 0.0,
		},
		Providers: v1.ProviderMetrics{
			Total:   int64(len(s.providers)),
			Healthy: 0,
			Unhealthy: 0,
		},
		Routing: v1.RoutingMetrics{
			TotalDecisions: 0,
			PolicyUsage:    make(map[string]int64),
		},
		Cache: v1.CacheMetrics{
			Hits:    0,
			Misses:  0,
			HitRate: 0.0,
			Size:    0,
			MaxSize: 0,
		},
		Timestamp: time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// handleGetProviders returns information about all providers.
func (s *Server) handleGetProviders(w http.ResponseWriter, r *http.Request) {
	providers := make(map[string]interface{})
	
	for name, provider := range s.providers {
		health := provider.GetHealth()
		models, _ := provider.GetModels()
		
		providers[name] = map[string]interface{}{
			"name":     name,
			"healthy":  health.Healthy,
			"latency":  health.Latency.String(),
			"last_check": health.LastCheck,
			"error":    health.Error,
			"models":   models,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(providers)
}

// handleGetProviderHealth returns health information for a specific provider.
func (s *Server) handleGetProviderHealth(w http.ResponseWriter, r *http.Request) {
	providerName := chi.URLParam(r, "name")
	
	provider, exists := s.providers[providerName]
	if !exists {
		http.Error(w, "Provider not found", http.StatusNotFound)
		return
	}

	health := provider.GetHealth()
	models, _ := provider.GetModels()
	
	response := map[string]interface{}{
		"name":      providerName,
		"healthy":   health.Healthy,
		"latency":   health.Latency.String(),
		"last_check": health.LastCheck,
		"error":     health.Error,
		"models":    models,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// handleForceHealthCheck forces a health check for a specific provider.
func (s *Server) handleForceHealthCheck(w http.ResponseWriter, r *http.Request) {
	providerName := chi.URLParam(r, "name")
	
	// Force health check
	s.healthChecker.ForceHealthCheck()
	
	response := map[string]string{
		"message": fmt.Sprintf("Health check triggered for provider: %s", providerName),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// handleGetRoutingPolicy returns information about the current routing policy.
func (s *Server) handleGetRoutingPolicy(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"name":        s.routingPolicy.GetName(),
		"description": s.routingPolicy.GetDescription(),
		"type":        s.config.RoutingPolicy.Type,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// handleUpdateRoutingPolicy updates the routing policy configuration.
func (s *Server) handleUpdateRoutingPolicy(w http.ResponseWriter, r *http.Request) {
	// This would allow dynamic policy updates
	// For now, return not implemented
	http.Error(w, "Policy updates not yet implemented", http.StatusNotImplemented)
}

// Helper functions for converting between API and internal types

func convertMessages(apiMessages []v1.Message) []models.Message {
	messages := make([]models.Message, len(apiMessages))
	for i, msg := range apiMessages {
		messages[i] = models.Message{
			Role:      msg.Role,
			Content:   msg.Content,
			Name:      msg.Name,
			Timestamp: msg.Timestamp,
		}
	}
	return messages
}

func convertChoices(choices []models.Choice) []v1.Choice {
	apiChoices := make([]v1.Choice, len(choices))
	for i, choice := range choices {
		apiChoices[i] = v1.Choice{
			Index:        choice.Index,
			Message:      convertMessage(choice.Message),
			FinishReason: choice.FinishReason,
		}
	}
	return apiChoices
}

func convertMessage(msg models.Message) v1.Message {
	return v1.Message{
		Role:      msg.Role,
		Content:   msg.Content,
		Name:      msg.Name,
		Timestamp: msg.Timestamp,
	}
}

func convertUsage(usage models.Usage) v1.Usage {
	return v1.Usage{
		PromptTokens:     usage.PromptTokens,
		CompletionTokens: usage.CompletionTokens,
		TotalTokens:      usage.TotalTokens,
	}
}
