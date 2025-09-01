package observability

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// TracingConfig holds configuration for tracing.
type TracingConfig struct {
	Enabled     bool   `mapstructure:"enabled"`
	ServiceName string `mapstructure:"service_name"`
	Environment string `mapstructure:"environment"`
}

// Tracing provides OpenTelemetry tracing functionality.
type Tracing struct {
	config TracingConfig
	logger *zap.Logger
	tracer trace.Tracer
}

// NewTracing creates a new tracing instance.
func NewTracing(config TracingConfig, logger *zap.Logger) *Tracing {
	// Set global tracer provider if not already set
	if otel.GetTracerProvider() == nil {
		// In production, you would configure a proper tracer provider here
		// For now, we'll use the default no-op tracer
		logger.Info("Using default no-op tracer - configure proper tracer provider for production")
	}

	tracer := otel.Tracer(config.ServiceName)

	return &Tracing{
		config: config,
		logger: logger,
		tracer: tracer,
	}
}

// StartSpan starts a new span for the given operation.
func (t *Tracing) StartSpan(ctx context.Context, operationName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return t.tracer.Start(ctx, operationName, opts...)
}

// StartSpanWithAttributes starts a new span with the given attributes.
func (t *Tracing) StartSpanWithAttributes(ctx context.Context, operationName string, attributes map[string]string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	// Convert string attributes to OpenTelemetry attributes
	otelAttrs := make([]attribute.KeyValue, 0, len(attributes))
	for k, v := range attributes {
		otelAttrs = append(otelAttrs, attribute.String(k, v))
	}

	// Add attributes to span start options
	spanOpts := append(opts, trace.WithAttributes(otelAttrs...))

	return t.tracer.Start(ctx, operationName, spanOpts...)
}

// AddEvent adds an event to the current span.
func (t *Tracing) AddEvent(ctx context.Context, name string, attributes map[string]string) {
	span := trace.SpanFromContext(ctx)
	if span == nil {
		return
	}

	otelAttrs := make([]attribute.KeyValue, 0, len(attributes))
	for k, v := range attributes {
		otelAttrs = append(otelAttrs, attribute.String(k, v))
	}

	span.AddEvent(name, trace.WithAttributes(otelAttrs...))
}

// SetAttributes sets attributes on the current span.
func (t *Tracing) SetAttributes(ctx context.Context, attributes map[string]string) {
	span := trace.SpanFromContext(ctx)
	if span == nil {
		return
	}

	otelAttrs := make([]attribute.KeyValue, 0, len(attributes))
	for k, v := range attributes {
		otelAttrs = append(otelAttrs, attribute.String(k, v))
	}

	span.SetAttributes(otelAttrs...)
}

// RecordError records an error on the current span.
func (t *Tracing) RecordError(ctx context.Context, err error, attributes map[string]string) {
	span := trace.SpanFromContext(ctx)
	if span == nil {
		return
	}

	otelAttrs := make([]attribute.KeyValue, 0, len(attributes))
	for k, v := range attributes {
		otelAttrs = append(otelAttrs, attribute.String(k, v))
	}

	span.RecordError(err, trace.WithAttributes(otelAttrs...))
}

// IsEnabled returns true if tracing is enabled.
func (t *Tracing) IsEnabled() bool {
	return t.config.Enabled
}

// GetTracer returns the underlying tracer.
func (t *Tracing) GetTracer() trace.Tracer {
	return t.tracer
}

// TraceFunction traces the execution of a function.
func (t *Tracing) TraceFunction(ctx context.Context, functionName string, fn func(context.Context) error) error {
	if !t.IsEnabled() {
		return fn(ctx)
	}

	ctx, span := t.StartSpan(ctx, functionName)
	defer span.End()

	start := time.Now()
	err := fn(ctx)
	duration := time.Since(start)

	// Record function execution metrics
	span.SetAttributes(
		attribute.String("function.name", functionName),
		attribute.Int64("function.duration_ms", duration.Milliseconds()),
	)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "")
	}

	return err
}

// TraceFunctionWithResult traces the execution of a function that returns a result.
func (t *Tracing) TraceFunctionWithResult(ctx context.Context, functionName string, fn func(context.Context) (interface{}, error)) (interface{}, error) {
	if !t.IsEnabled() {
		return fn(ctx)
	}

	ctx, span := t.StartSpan(ctx, functionName)
	defer span.End()

	start := time.Now()
	result, err := fn(ctx)
	duration := time.Since(start)

	// Record function execution metrics
	span.SetAttributes(
		attribute.String("function.name", functionName),
		attribute.Int64("function.duration_ms", duration.Milliseconds()),
	)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "")
	}

	return result, err
}
