package llmrouter

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	requestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "llm_requests_total",
		Help: "Total number of LLM requests",
	}, []string{"provider", "status"})

	requestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "llm_request_duration_seconds",
		Help:    "Duration of LLM requests",
		Buckets: prometheus.DefBuckets,
	}, []string{"provider", "model"})

	tokensUsed = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "llm_tokens_used_total",
		Help: "Total tokens used",
	}, []string{"provider", "type"})

	costTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "llm_cost_cents_total",
		Help: "Total cost in cents",
	}, []string{"provider", "org_id"})

	providerHealth = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "llm_provider_health",
		Help: "Provider health status (1=healthy, 0=unhealthy)",
	}, []string{"provider"})

	cacheHits = promauto.NewCounter(prometheus.CounterOpts{
		Name: "llm_cache_hits_total",
		Help: "Total number of cache hits",
	})

	cacheMisses = promauto.NewCounter(prometheus.CounterOpts{
		Name: "llm_cache_misses_total",
		Help: "Total number of cache misses",
	})

	httpRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total HTTP requests",
	}, []string{"method", "path", "status"})

	httpRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "HTTP request duration",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "path"})
)

// ProviderMetrics holds metrics for a specific provider
type ProviderMetrics struct {
	RequestCount    int64         `json:"request_count"`
	SuccessCount    int64         `json:"success_count"`
	FailureCount    int64         `json:"failure_count"`
	TotalTokens     int64         `json:"total_tokens"`
	TotalCostCents  float64       `json:"total_cost_cents"`
	AverageLatency  time.Duration `json:"average_latency_ms"`
	P95Latency      time.Duration `json:"p95_latency_ms"`
	P99Latency      time.Duration `json:"p99_latency_ms"`
	LastSuccess     time.Time     `json:"last_success"`
	LastFailure     time.Time     `json:"last_failure"`
	ErrorRate       float64       `json:"error_rate"`
	Availability    float64       `json:"availability"`
}

// MetricsCollector collects and aggregates metrics
type MetricsCollector struct {
	providerMetrics map[Provider]*ProviderMetrics
	mu              sync.RWMutex
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		providerMetrics: make(map[Provider]*ProviderMetrics),
	}
}

// RecordSuccess records a successful request
func (mc *MetricsCollector) RecordSuccess(provider Provider, latency time.Duration) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if _, ok := mc.providerMetrics[provider]; !ok {
		mc.providerMetrics[provider] = &ProviderMetrics{}
	}

	metrics := mc.providerMetrics[provider]
	metrics.RequestCount++
	metrics.SuccessCount++
	metrics.LastSuccess = time.Now()
	
	// Update average latency (simplified)
	if metrics.AverageLatency == 0 {
		metrics.AverageLatency = latency
	} else {
		metrics.AverageLatency = (metrics.AverageLatency + latency) / 2
	}

	// Update error rate and availability
	if metrics.RequestCount > 0 {
		metrics.ErrorRate = float64(metrics.FailureCount) / float64(metrics.RequestCount)
		metrics.Availability = float64(metrics.SuccessCount) / float64(metrics.RequestCount)
	}

	// Update Prometheus metrics
	requestsTotal.WithLabelValues(string(provider), "success").Inc()
	requestDuration.WithLabelValues(string(provider), "").Observe(latency.Seconds())
	providerHealth.WithLabelValues(string(provider)).Set(1)
}

// RecordFailure records a failed request
func (mc *MetricsCollector) RecordFailure(provider Provider, err error) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if _, ok := mc.providerMetrics[provider]; !ok {
		mc.providerMetrics[provider] = &ProviderMetrics{}
	}

	metrics := mc.providerMetrics[provider]
	metrics.RequestCount++
	metrics.FailureCount++
	metrics.LastFailure = time.Now()

	// Update error rate and availability
	if metrics.RequestCount > 0 {
		metrics.ErrorRate = float64(metrics.FailureCount) / float64(metrics.RequestCount)
		metrics.Availability = float64(metrics.SuccessCount) / float64(metrics.RequestCount)
	}

	// Update Prometheus metrics
	requestsTotal.WithLabelValues(string(provider), "failure").Inc()
	
	// Set health to 0 if error rate is too high
	if metrics.ErrorRate > 0.5 {
		providerHealth.WithLabelValues(string(provider)).Set(0)
	}
}

// RecordTokenUsage records token usage
func (mc *MetricsCollector) RecordTokenUsage(provider Provider, promptTokens, completionTokens int) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if _, ok := mc.providerMetrics[provider]; !ok {
		mc.providerMetrics[provider] = &ProviderMetrics{}
	}

	metrics := mc.providerMetrics[provider]
	metrics.TotalTokens += int64(promptTokens + completionTokens)

	// Update Prometheus metrics
	tokensUsed.WithLabelValues(string(provider), "prompt").Add(float64(promptTokens))
	tokensUsed.WithLabelValues(string(provider), "completion").Add(float64(completionTokens))
}

// RecordCost records cost
func (mc *MetricsCollector) RecordCost(provider Provider, costCents float64, orgID string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if _, ok := mc.providerMetrics[provider]; !ok {
		mc.providerMetrics[provider] = &ProviderMetrics{}
	}

	metrics := mc.providerMetrics[provider]
	metrics.TotalCostCents += costCents

	// Update Prometheus metrics
	costTotal.WithLabelValues(string(provider), orgID).Add(costCents)
}

// RecordCacheHit records a cache hit
func (mc *MetricsCollector) RecordCacheHit() {
	cacheHits.Inc()
}

// RecordCacheMiss records a cache miss
func (mc *MetricsCollector) RecordCacheMiss() {
	cacheMisses.Inc()
}

// RecordHTTPRequest records HTTP request metrics
func (mc *MetricsCollector) RecordHTTPRequest(method, path string, status int, duration time.Duration) {
	statusStr := string(rune(status))
	httpRequestsTotal.WithLabelValues(method, path, statusStr).Inc()
	httpRequestDuration.WithLabelValues(method, path).Observe(duration.Seconds())
}

// GetProviderMetrics returns metrics for a specific provider
func (mc *MetricsCollector) GetProviderMetrics(provider Provider) *ProviderMetrics {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	if metrics, ok := mc.providerMetrics[provider]; ok {
		// Return a copy to avoid race conditions
		metricsCopy := *metrics
		return &metricsCopy
	}

	return &ProviderMetrics{}
}

// GetAllMetrics returns metrics for all providers
func (mc *MetricsCollector) GetAllMetrics() map[Provider]*ProviderMetrics {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	// Create a copy of the map
	result := make(map[Provider]*ProviderMetrics)
	for provider, metrics := range mc.providerMetrics {
		metricsCopy := *metrics
		result[provider] = &metricsCopy
	}

	return result
}

// Reset resets all metrics (useful for testing)
func (mc *MetricsCollector) Reset() {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.providerMetrics = make(map[Provider]*ProviderMetrics)
}