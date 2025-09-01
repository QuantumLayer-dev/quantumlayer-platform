package orchestrator

import (
	"sync"
	"sync/atomic"
	"time"
)

// MetricsCollector collects metrics for the orchestrator
type MetricsCollector struct {
	totalTasks      int64
	completedTasks  int64
	failedTasks     int64
	taskDurations   []time.Duration
	mu              sync.RWMutex
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		taskDurations: make([]time.Duration, 0, 1000),
	}
}

// IncrementTotalTasks increments the total task counter
func (m *MetricsCollector) IncrementTotalTasks() {
	atomic.AddInt64(&m.totalTasks, 1)
}

// IncrementCompletedTasks increments the completed task counter
func (m *MetricsCollector) IncrementCompletedTasks() {
	atomic.AddInt64(&m.completedTasks, 1)
}

// IncrementFailedTasks increments the failed task counter
func (m *MetricsCollector) IncrementFailedTasks() {
	atomic.AddInt64(&m.failedTasks, 1)
}

// RecordTaskDuration records a task duration
func (m *MetricsCollector) RecordTaskDuration(duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.taskDurations = append(m.taskDurations, duration)
	
	// Keep only last 1000 durations
	if len(m.taskDurations) > 1000 {
		m.taskDurations = m.taskDurations[len(m.taskDurations)-1000:]
	}
}

// GetTotalTasks returns the total number of tasks
func (m *MetricsCollector) GetTotalTasks() int64 {
	return atomic.LoadInt64(&m.totalTasks)
}

// GetCompletedTasks returns the number of completed tasks
func (m *MetricsCollector) GetCompletedTasks() int64 {
	return atomic.LoadInt64(&m.completedTasks)
}

// GetFailedTasks returns the number of failed tasks
func (m *MetricsCollector) GetFailedTasks() int64 {
	return atomic.LoadInt64(&m.failedTasks)
}

// GetAverageTaskTime returns the average task time in milliseconds
func (m *MetricsCollector) GetAverageTaskTime() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if len(m.taskDurations) == 0 {
		return 0
	}
	
	var total time.Duration
	for _, d := range m.taskDurations {
		total += d
	}
	
	avg := total / time.Duration(len(m.taskDurations))
	return float64(avg.Milliseconds())
}

// GetMetrics returns all metrics
func (m *MetricsCollector) GetMetrics() map[string]interface{} {
	return map[string]interface{}{
		"total_tasks":      m.GetTotalTasks(),
		"completed_tasks":  m.GetCompletedTasks(),
		"failed_tasks":     m.GetFailedTasks(),
		"average_time_ms":  m.GetAverageTaskTime(),
		"success_rate":     m.GetSuccessRate(),
	}
}

// GetSuccessRate returns the task success rate
func (m *MetricsCollector) GetSuccessRate() float64 {
	completed := float64(m.GetCompletedTasks())
	failed := float64(m.GetFailedTasks())
	
	total := completed + failed
	if total == 0 {
		return 0
	}
	
	return (completed / total) * 100
}