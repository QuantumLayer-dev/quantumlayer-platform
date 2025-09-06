package engine

import (
	"sync"
	"time"
)

// Metrics tracks performance and usage metrics
type Metrics struct {
	TotalRequests int64              `json:"total_requests"`
	AvgLatency    float64            `json:"avg_latency_ms"`
	SuccessRate   float64            `json:"success_rate"`
	TemplateUsage map[string]int64   `json:"template_usage"`
	LastUpdated   time.Time          `json:"last_updated"`
	mu            sync.RWMutex
}

// NewMetrics creates a new metrics tracker
func NewMetrics() *Metrics {
	return &Metrics{
		TemplateUsage: make(map[string]int64),
		LastUpdated:   time.Now(),
	}
}

// RecordRequest records a request and its latency
func (m *Metrics) RecordRequest(success bool, latencyMs float64, template string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.TotalRequests++
	
	// Update average latency
	m.AvgLatency = (m.AvgLatency*float64(m.TotalRequests-1) + latencyMs) / float64(m.TotalRequests)
	
	// Update success rate
	if success {
		successCount := m.SuccessRate * float64(m.TotalRequests-1)
		m.SuccessRate = (successCount + 1) / float64(m.TotalRequests)
	} else {
		successCount := m.SuccessRate * float64(m.TotalRequests-1)
		m.SuccessRate = successCount / float64(m.TotalRequests)
	}
	
	// Update template usage
	if template != "" {
		m.TemplateUsage[template]++
	}
	
	m.LastUpdated = time.Now()
}

// GetSnapshot returns a copy of current metrics
func (m *Metrics) GetSnapshot() *Metrics {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	// Create a copy
	snapshot := &Metrics{
		TotalRequests: m.TotalRequests,
		AvgLatency:    m.AvgLatency,
		SuccessRate:   m.SuccessRate,
		LastUpdated:   m.LastUpdated,
		TemplateUsage: make(map[string]int64),
	}
	
	// Copy template usage
	for k, v := range m.TemplateUsage {
		snapshot.TemplateUsage[k] = v
	}
	
	return snapshot
}