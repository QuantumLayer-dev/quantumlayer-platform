package llmrouter

import (
	"context"
	"fmt"
	"time"

	"github.com/QuantumLayer-dev/quantumlayer-platform/packages/shared/circuitbreaker"
	"github.com/QuantumLayer-dev/quantumlayer-platform/packages/shared/tracing"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	openai "github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
)

// EnterpriseOpenAIClient implements production-ready OpenAI integration
type EnterpriseOpenAIClient struct {
	client         *openai.Client
	logger         *zap.Logger
	config         *ProviderConfig
	circuitBreaker *circuitbreaker.CircuitBreaker
	metrics        *ProviderMetrics
}

// ProviderMetrics tracks provider-specific metrics
type ProviderMetrics struct {
	totalRequests   int64
	successRequests int64
	failedRequests  int64
	totalLatency    time.Duration
	tokenCount      int64
}

// NewEnterpriseOpenAIClient creates a production-ready OpenAI client
func NewEnterpriseOpenAIClient(apiKey string, config *ProviderConfig, logger *zap.Logger) *EnterpriseOpenAIClient {
	// Configure OpenAI client with proper timeouts
	clientConfig := openai.DefaultConfig(apiKey)
	clientConfig.HTTPClient.Timeout = config.Timeout
	
	// Create circuit breaker
	cb := circuitbreaker.NewCircuitBreaker(
		circuitbreaker.Config{
			Name:         "openai-circuit",
			MaxFailures:  5,
			ResetTimeout: 60 * time.Second,
			HalfOpenMax:  3,
			OnStateChange: func(from, to circuitbreaker.State) {
				logger.Warn("OpenAI circuit breaker state changed",
					zap.String("from", from.String()),
					zap.String("to", to.String()),
				)
			},
		},
		logger,
	)
	
	return &EnterpriseOpenAIClient{
		client:         openai.NewClientWithConfig(clientConfig),
		logger:         logger,
		config:         config,
		circuitBreaker: cb,
		metrics:        &ProviderMetrics{},
	}
}

// Complete sends a completion request with full enterprise features
func (c *EnterpriseOpenAIClient) Complete(ctx context.Context, req *Request) (*Response, error) {
	// Start tracing span
	span, ctx := tracing.StartSpanFromContext(ctx, "openai.complete", map[string]interface{}{
		"provider": "openai",
		"model":    req.Model,
		"request_id": req.ID,
	})
	defer span.Finish()
	
	// Check rate limits
	if !c.config.RateLimiter.Allow() {
		err := fmt.Errorf("rate limit exceeded for OpenAI")
		tracing.SetSpanError(span, err)
		return nil, err
	}
	
	// Check token bucket
	estimatedTokens := int64(len(req.Messages) * 100) // Rough estimate
	if !c.config.TokenBucket.Consume(estimatedTokens) {
		err := fmt.Errorf("token quota exceeded for OpenAI")
		tracing.SetSpanError(span, err)
		return nil, err
	}
	
	// Execute with circuit breaker
	result, err := c.circuitBreaker.ExecuteWithFallback(
		ctx,
		func(ctx context.Context) (interface{}, error) {
			return c.executeCompletion(ctx, req, span)
		},
		func(ctx context.Context, cbErr error) (interface{}, error) {
			// Fallback to cached response or degraded mode
			c.logger.Warn("OpenAI circuit breaker triggered, using fallback",
				zap.Error(cbErr),
				zap.String("request_id", req.ID),
			)
			return c.fallbackResponse(req), nil
		},
	)
	
	if err != nil {
		c.recordMetrics(false, 0)
		tracing.SetSpanError(span, err)
		return nil, err
	}
	
	response := result.(*Response)
	c.recordMetrics(true, response.Usage.TotalTokens)
	
	// Record health check success
	c.config.HealthChecker.RecordSuccess()
	
	return response, nil
}

// executeCompletion performs the actual API call
func (c *EnterpriseOpenAIClient) executeCompletion(ctx context.Context, req *Request, span opentracing.Span) (*Response, error) {
	startTime := time.Now()
	
	// Convert messages with validation
	messages, err := c.convertMessages(req.Messages)
	if err != nil {
		return nil, fmt.Errorf("invalid message format: %w", err)
	}
	
	// Build OpenAI request with all parameters
	openaiReq := openai.ChatCompletionRequest{
		Model:            c.mapModel(req.Model),
		Messages:         messages,
		MaxTokens:        req.MaxTokens,
		Temperature:      req.Temperature,
		TopP:             req.TopP,
		N:                1,
		Stop:             req.Stop,
		PresencePenalty:  req.PresencePenalty,
		FrequencyPenalty: req.FrequencyPenalty,
		Stream:           false,
		User:             req.UserID,
	}
	
	// Add function calling if specified
	if req.Functions != nil {
		openaiReq.Functions = c.convertFunctions(req.Functions)
	}
	
	// Set span tags for request details
	span.SetTag("openai.model", openaiReq.Model)
	span.SetTag("openai.max_tokens", openaiReq.MaxTokens)
	span.SetTag("openai.temperature", openaiReq.Temperature)
	
	// Make API call with timeout context
	apiCtx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()
	
	resp, err := c.client.CreateChatCompletion(apiCtx, openaiReq)
	if err != nil {
		c.config.HealthChecker.RecordFailure()
		c.logger.Error("OpenAI API error",
			zap.Error(err),
			zap.String("request_id", req.ID),
			zap.String("model", string(req.Model)),
			zap.Duration("latency", time.Since(startTime)),
		)
		return nil, fmt.Errorf("OpenAI API error: %w", err)
	}
	
	// Set span tags for response
	span.SetTag("openai.usage.prompt_tokens", resp.Usage.PromptTokens)
	span.SetTag("openai.usage.completion_tokens", resp.Usage.CompletionTokens)
	span.SetTag("openai.usage.total_tokens", resp.Usage.TotalTokens)
	span.SetTag("openai.latency_ms", time.Since(startTime).Milliseconds())
	
	// Convert to our response format with proper error handling
	return c.convertResponse(&resp, req), nil
}

// convertMessages converts our message format to OpenAI format with validation
func (c *EnterpriseOpenAIClient) convertMessages(messages []Message) ([]openai.ChatCompletionMessage, error) {
	if len(messages) == 0 {
		return nil, fmt.Errorf("no messages provided")
	}
	
	result := make([]openai.ChatCompletionMessage, 0, len(messages))
	for i, msg := range messages {
		if msg.Content == "" && msg.FunctionCall == nil {
			return nil, fmt.Errorf("message %d has empty content", i)
		}
		
		openaiMsg := openai.ChatCompletionMessage{
			Role:    msg.Role,
			Content: msg.Content,
			Name:    msg.Name,
		}
		
		if msg.FunctionCall != nil {
			openaiMsg.FunctionCall = &openai.FunctionCall{
				Name:      msg.FunctionCall.Name,
				Arguments: msg.FunctionCall.Arguments,
			}
		}
		
		result = append(result, openaiMsg)
	}
	
	return result, nil
}

// convertResponse converts OpenAI response to our format
func (c *EnterpriseOpenAIClient) convertResponse(resp *openai.ChatCompletionResponse, req *Request) *Response {
	choices := make([]Choice, 0, len(resp.Choices))
	for _, choice := range resp.Choices {
		ourChoice := Choice{
			Index: choice.Index,
			Message: Message{
				Role:    choice.Message.Role,
				Content: choice.Message.Content,
				Name:    choice.Message.Name,
			},
			FinishReason: string(choice.FinishReason),
		}
		
		if choice.Message.FunctionCall != nil {
			ourChoice.Message.FunctionCall = &FunctionCall{
				Name:      choice.Message.FunctionCall.Name,
				Arguments: choice.Message.FunctionCall.Arguments,
			}
		}
		
		choices = append(choices, ourChoice)
	}
	
	return &Response{
		ID:       resp.ID,
		Object:   resp.Object,
		Created:  resp.Created,
		Model:    Model(resp.Model),
		Provider: ProviderOpenAI,
		Choices:  choices,
		Usage: Usage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
		RequestID: req.ID,
		Latency:   time.Since(time.Unix(resp.Created, 0)),
	}
}

// Stream implements streaming with proper error handling
func (c *EnterpriseOpenAIClient) Stream(ctx context.Context, req *Request) (<-chan *Response, error) {
	// Start tracing span
	span, ctx := tracing.StartSpanFromContext(ctx, "openai.stream", map[string]interface{}{
		"provider": "openai",
		"model":    req.Model,
		"request_id": req.ID,
	})
	
	// Check rate limits
	if !c.config.RateLimiter.Allow() {
		span.Finish()
		return nil, fmt.Errorf("rate limit exceeded for OpenAI")
	}
	
	respChan := make(chan *Response, 100)
	
	go func() {
		defer close(respChan)
		defer span.Finish()
		
		// Execute with circuit breaker
		_, err := c.circuitBreaker.Execute(ctx, func(ctx context.Context) (interface{}, error) {
			return nil, c.executeStream(ctx, req, respChan, span)
		})
		
		if err != nil {
			c.logger.Error("OpenAI stream error", zap.Error(err))
			tracing.SetSpanError(span, err)
		}
	}()
	
	return respChan, nil
}

// executeStream performs the streaming API call
func (c *EnterpriseOpenAIClient) executeStream(ctx context.Context, req *Request, respChan chan<- *Response, span opentracing.Span) error {
	messages, err := c.convertMessages(req.Messages)
	if err != nil {
		return err
	}
	
	openaiReq := openai.ChatCompletionRequest{
		Model:       c.mapModel(req.Model),
		Messages:    messages,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		Stream:      true,
	}
	
	stream, err := c.client.CreateChatCompletionStream(ctx, openaiReq)
	if err != nil {
		c.config.HealthChecker.RecordFailure()
		return err
	}
	defer stream.Close()
	
	for {
		response, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		
		if len(response.Choices) > 0 {
			resp := &Response{
				ID:       response.ID,
				Object:   response.Object,
				Created:  response.Created,
				Model:    Model(response.Model),
				Provider: ProviderOpenAI,
				Choices: []Choice{{
					Index: response.Choices[0].Index,
					Message: Message{
						Role:    response.Choices[0].Delta.Role,
						Content: response.Choices[0].Delta.Content,
					},
					FinishReason: string(response.Choices[0].FinishReason),
				}},
			}
			
			select {
			case respChan <- resp:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
	
	c.config.HealthChecker.RecordSuccess()
	return nil
}

// fallbackResponse generates a fallback response when circuit is open
func (c *EnterpriseOpenAIClient) fallbackResponse(req *Request) *Response {
	return &Response{
		ID:       generateRequestID(),
		Object:   "chat.completion",
		Created:  time.Now().Unix(),
		Model:    req.Model,
		Provider: ProviderOpenAI,
		Choices: []Choice{{
			Index: 0,
			Message: Message{
				Role:    "assistant",
				Content: "Service temporarily unavailable. Please try again later.",
			},
			FinishReason: "error",
		}},
		Usage: Usage{
			PromptTokens:     0,
			CompletionTokens: 0,
			TotalTokens:      0,
		},
		RequestID: req.ID,
		Error:     "Circuit breaker open - service degraded",
	}
}

// recordMetrics records provider metrics
func (c *EnterpriseOpenAIClient) recordMetrics(success bool, tokens int64) {
	c.metrics.totalRequests++
	if success {
		c.metrics.successRequests++
	} else {
		c.metrics.failedRequests++
	}
	c.metrics.tokenCount += tokens
}

// GetMetrics returns provider metrics
func (c *EnterpriseOpenAIClient) GetMetrics() map[string]interface{} {
	successRate := float64(0)
	if c.metrics.totalRequests > 0 {
		successRate = float64(c.metrics.successRequests) / float64(c.metrics.totalRequests) * 100
	}
	
	return map[string]interface{}{
		"total_requests":   c.metrics.totalRequests,
		"success_requests": c.metrics.successRequests,
		"failed_requests":  c.metrics.failedRequests,
		"success_rate":     successRate,
		"total_tokens":     c.metrics.tokenCount,
		"circuit_state":    c.circuitBreaker.GetState().String(),
	}
}

// convertFunctions converts function definitions
func (c *EnterpriseOpenAIClient) convertFunctions(functions []Function) []openai.FunctionDefinition {
	result := make([]openai.FunctionDefinition, 0, len(functions))
	for _, fn := range functions {
		result = append(result, openai.FunctionDefinition{
			Name:        fn.Name,
			Description: fn.Description,
			Parameters:  fn.Parameters,
		})
	}
	return result
}