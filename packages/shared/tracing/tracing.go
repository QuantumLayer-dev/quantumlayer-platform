package tracing

import (
	"context"
	"fmt"
	"io"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/uber/jaeger-client-go"
	jaegercfg "github.com/uber/jaeger-client-go/config"
	jaegerlog "github.com/uber/jaeger-client-go/log"
	"github.com/uber/jaeger-lib/metrics"
	"go.uber.org/zap"
)

// Config holds tracing configuration
type Config struct {
	ServiceName         string
	JaegerEndpoint      string
	SamplingRate        float64
	BufferFlushInterval int
	LogSpans           bool
}

// Tracer wraps the OpenTracing tracer
type Tracer struct {
	tracer opentracing.Tracer
	closer io.Closer
	logger *zap.Logger
}

// NewTracer creates a new tracer instance
func NewTracer(config Config, logger *zap.Logger) (*Tracer, error) {
	if config.ServiceName == "" {
		config.ServiceName = "quantumlayer"
	}
	if config.JaegerEndpoint == "" {
		config.JaegerEndpoint = "jaeger-collector.istio-system.svc.cluster.local:14268"
	}
	if config.SamplingRate <= 0 {
		config.SamplingRate = 1.0
	}
	
	cfg := jaegercfg.Configuration{
		ServiceName: config.ServiceName,
		Sampler: &jaegercfg.SamplerConfig{
			Type:  jaeger.SamplerTypeConst,
			Param: config.SamplingRate,
		},
		Reporter: &jaegercfg.ReporterConfig{
			LogSpans:            config.LogSpans,
			BufferFlushInterval: config.BufferFlushInterval,
			CollectorEndpoint:   fmt.Sprintf("http://%s/api/traces", config.JaegerEndpoint),
		},
	}
	
	jLogger := &jaegerLogger{logger: logger}
	jMetrics := metrics.NullFactory
	
	tracer, closer, err := cfg.NewTracer(
		jaegercfg.Logger(jLogger),
		jaegercfg.Metrics(jMetrics),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create tracer: %w", err)
	}
	
	// Set as global tracer
	opentracing.SetGlobalTracer(tracer)
	
	return &Tracer{
		tracer: tracer,
		closer: closer,
		logger: logger,
	}, nil
}

// StartSpan starts a new span
func (t *Tracer) StartSpan(ctx context.Context, operationName string, opts ...opentracing.StartSpanOption) (opentracing.Span, context.Context) {
	// Extract parent span from context
	parentSpan := opentracing.SpanFromContext(ctx)
	if parentSpan != nil {
		opts = append(opts, opentracing.ChildOf(parentSpan.Context()))
	}
	
	span := t.tracer.StartSpan(operationName, opts...)
	return span, opentracing.ContextWithSpan(ctx, span)
}

// StartSpanFromContext starts a span from context
func StartSpanFromContext(ctx context.Context, operationName string, tags ...map[string]interface{}) (opentracing.Span, context.Context) {
	tracer := opentracing.GlobalTracer()
	
	// Extract parent span
	parentSpan := opentracing.SpanFromContext(ctx)
	
	var opts []opentracing.StartSpanOption
	if parentSpan != nil {
		opts = append(opts, opentracing.ChildOf(parentSpan.Context()))
	}
	
	span := tracer.StartSpan(operationName, opts...)
	
	// Add tags
	for _, tagMap := range tags {
		for k, v := range tagMap {
			span.SetTag(k, v)
		}
	}
	
	return span, opentracing.ContextWithSpan(ctx, span)
}

// InjectHTTPHeaders injects tracing headers into HTTP headers
func (t *Tracer) InjectHTTPHeaders(span opentracing.Span, headers map[string]string) error {
	carrier := opentracing.HTTPHeadersCarrier(headers)
	return t.tracer.Inject(span.Context(), opentracing.HTTPHeaders, carrier)
}

// ExtractHTTPHeaders extracts span context from HTTP headers
func (t *Tracer) ExtractHTTPHeaders(headers map[string]string) (opentracing.SpanContext, error) {
	carrier := opentracing.HTTPHeadersCarrier(headers)
	return t.tracer.Extract(opentracing.HTTPHeaders, carrier)
}

// Close closes the tracer
func (t *Tracer) Close() error {
	if t.closer != nil {
		return t.closer.Close()
	}
	return nil
}

// TraceFunction wraps a function with tracing
func TraceFunction(ctx context.Context, name string, fn func(context.Context) error) error {
	span, ctx := StartSpanFromContext(ctx, name)
	defer span.Finish()
	
	err := fn(ctx)
	if err != nil {
		ext.Error.Set(span, true)
		span.SetTag("error.message", err.Error())
	}
	
	return err
}

// TraceFunctionWithResult wraps a function that returns a result
func TraceFunctionWithResult[T any](ctx context.Context, name string, fn func(context.Context) (T, error)) (T, error) {
	span, ctx := StartSpanFromContext(ctx, name)
	defer span.Finish()
	
	result, err := fn(ctx)
	if err != nil {
		ext.Error.Set(span, true)
		span.SetTag("error.message", err.Error())
	}
	
	return result, err
}

// SetSpanError sets error on span
func SetSpanError(span opentracing.Span, err error) {
	if span == nil || err == nil {
		return
	}
	
	ext.Error.Set(span, true)
	span.SetTag("error.message", err.Error())
}

// SetSpanTag sets a tag on the span from context
func SetSpanTag(ctx context.Context, key string, value interface{}) {
	span := opentracing.SpanFromContext(ctx)
	if span != nil {
		span.SetTag(key, value)
	}
}

// LogToSpan logs to the span from context
func LogToSpan(ctx context.Context, fields ...opentracing.LogData) {
	span := opentracing.SpanFromContext(ctx)
	if span != nil {
		for _, field := range fields {
			span.LogKV(field)
		}
	}
}

// jaegerLogger implements jaeger logger interface
type jaegerLogger struct {
	logger *zap.Logger
}

func (l *jaegerLogger) Error(msg string) {
	l.logger.Error(msg)
}

func (l *jaegerLogger) Infof(msg string, args ...interface{}) {
	l.logger.Info(fmt.Sprintf(msg, args...))
}

func (l *jaegerLogger) Debugf(msg string, args ...interface{}) {
	l.logger.Debug(fmt.Sprintf(msg, args...))
}

// Middleware for Gin framework with tracing
func GinTracingMiddleware(tracer *Tracer) func(c *gin.Context) {
	return func(c *gin.Context) {
		// Extract parent span from headers
		spanContext, _ := tracer.ExtractHTTPHeaders(c.Request.Header)
		
		// Start span
		var span opentracing.Span
		if spanContext != nil {
			span = tracer.tracer.StartSpan(
				fmt.Sprintf("%s %s", c.Request.Method, c.Request.URL.Path),
				opentracing.ChildOf(spanContext),
			)
		} else {
			span = tracer.tracer.StartSpan(
				fmt.Sprintf("%s %s", c.Request.Method, c.Request.URL.Path),
			)
		}
		defer span.Finish()
		
		// Set standard tags
		ext.HTTPMethod.Set(span, c.Request.Method)
		ext.HTTPUrl.Set(span, c.Request.URL.String())
		ext.Component.Set(span, "gin")
		
		// Add span to context
		ctx := opentracing.ContextWithSpan(c.Request.Context(), span)
		c.Request = c.Request.WithContext(ctx)
		
		// Process request
		c.Next()
		
		// Set response tags
		ext.HTTPStatusCode.Set(span, uint16(c.Writer.Status()))
		if c.Writer.Status() >= 500 {
			ext.Error.Set(span, true)
		}
	}
}