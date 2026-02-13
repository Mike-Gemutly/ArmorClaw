// Package queue provides Prometheus metrics collection for message queue operations
package queue

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	// Queue metrics
	msgsEnqueued = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "sdtw_queue_enqueued_total",
			Help: "Total number of messages enqueued",
		},
		[]string{"platform"},
	)

	msgsDequeued = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "sdtw_queue_dequeued_total",
			Help: "Total number of messages dequeued",
		},
		[]string{"platform"},
	)

	msgsAcked = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "sdtw_queue_acked_total",
			Help: "Total number of messages acknowledged",
		},
		[]string{"platform"},
	)

	msgsRequeued = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "sdtw_queue_retried_total",
			Help: "Total number of messages requeued for retry",
		},
		[]string{"platform"},
	)

	msgsRetried = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "sdtw_queue_retry_total",
			Help: "Total number of message retry attempts",
		},
		[]string{"platform"},
	)

	msgsDLQ = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "sdtw_queue_dlq_total",
			Help: "Total number of messages moved to dead letter queue",
		},
		[]string{"platform"},
	)

	dlqReviewed = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "sdtw_queue_dlq_reviewed_total",
			Help: "Total number of DLQ messages reviewed",
		},
		[]string{"platform"},
	)

	dlqRetried = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "sdtw_queue_dlq_retried_total",
			Help: "Total number of DLQ messages retried",
		},
		[]string{"platform"},
	)

	dlqCleared = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "sdtw_queue_dlq_cleared_total",
			Help: "Total number of DLQ messages cleared",
		},
		[]string{"platform"},
	)

	// Gauge metrics
	queueDepth = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "sdtw_queue_depth",
			Help: "Current depth of message queue",
		},
		[]string{"platform", "state"},
	)

	queueInflight = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "sdtw_queue_inflight",
			Help: "Number of messages currently in flight",
		},
		[]string{"platform"},
	)

	queueFailed = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "sdtw_queue_failed",
			Help: "Number of failed messages in queue",
		},
		[]string{"platform"},
	)

	batchSize = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "sdtw_queue_batch_size",
			Help: "Size of batch dequeue operations",
		},
		[]string{"platform"},
	)

	// Histogram metrics
	msgWaitTime = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "sdtw_queue_wait_duration_seconds",
			Help:    "Time messages spend waiting in queue",
			Buckets: []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
		},
		[]string{"platform"},
	)
)

// QueueMetrics tracks queue performance metrics
type QueueMetrics struct {
	enqueued      int64
	dequeued      int64
	acked         int64
	requeued      int64
	retried       int64
	dlq           int64
	dlqReviewed   int64
	dlqRetried    int64
	dlqCleared    int64
	batchSize     int
	mu            sync.RWMutex
}

// NewQueueMetrics creates a new metrics collector
func NewQueueMetrics() *QueueMetrics {
	return &QueueMetrics{}
}

// RecordEnqueued records a message enqueue operation
func (qm *QueueMetrics) RecordEnqueued() {
	qm.mu.Lock()
	qm.enqueued++
	qm.mu.Unlock()
}

// RecordDequeued records a message dequeue operation
func (qm *QueueMetrics) RecordDequeued() {
	qm.mu.Lock()
	qm.dequeued++
	qm.mu.Unlock()
}

// RecordAcked records a message acknowledgment
func (qm *QueueMetrics) RecordAcked() {
	qm.mu.Lock()
	qm.acked++
	qm.mu.Unlock()
}

// RecordRequeued records a message requeue operation
func (qm *QueueMetrics) RecordRequeued() {
	qm.mu.Lock()
	qm.requeued++
	qm.mu.Unlock()
}

// RecordRetried records a message retry operation
func (qm *QueueMetrics) RecordRetried() {
	qm.mu.Lock()
	qm.retried++
	qm.mu.Unlock()
}

// RecordDLQ records a message moved to dead letter queue
func (qm *QueueMetrics) RecordDLQ() {
	qm.mu.Lock()
	qm.dlq++
	qm.mu.Unlock()
}

// RecordDLQReviewed records a DLQ message review
func (qm *QueueMetrics) RecordDLQReviewed() {
	qm.mu.Lock()
	qm.dlqReviewed++
	qm.mu.Unlock()
}

// RecordDLQRetried records a DLQ message retry
func (qm *QueueMetrics) RecordDLQRetried() {
	qm.mu.Lock()
	qm.dlqRetried++
	qm.mu.Unlock()
}

// RecordDLQCleared records DLQ cleanup operation
func (qm *QueueMetrics) RecordDLQCleared(count int) {
	qm.mu.Lock()
	qm.dlqCleared += int64(count)
	qm.mu.Unlock()
}

// RecordBatch records a batch dequeue operation
func (qm *QueueMetrics) RecordBatch(size int) {
	qm.mu.Lock()
	qm.batchSize = size
	qm.mu.Unlock()
}

// Reset clears all metrics (useful for testing)
func (qm *QueueMetrics) Reset() {
	qm.mu.Lock()
	qm.enqueued = 0
	qm.dequeued = 0
	qm.acked = 0
	qm.requeued = 0
	qm.retried = 0
	qm.dlq = 0
	qm.dlqReviewed = 0
	qm.dlqRetried = 0
	qm.dlqCleared = 0
	qm.batchSize = 0
	qm.mu.Unlock()
}

// ToPrometheusMetrics returns metrics for Prometheus scraping
func (qm *QueueMetrics) ToPrometheusMetrics(platform string) []prometheus.Collector {
	// Get snapshot
	qm.mu.RLock()
	defer qm.mu.RUnlock()

	// Set gauge values
	pending := qm.enqueued - qm.dequeued - qm.acked - qm.dlq
	queueDepth.WithLabelValues(platform, "pending").Set(float64(pending))
	queueInflight.WithLabelValues(platform).Set(float64(qm.requeued - qm.dequeued))
	queueFailed.WithLabelValues(platform).Set(float64(qm.retried))

	collectors := []prometheus.Collector{
		msgsEnqueued,
		msgsDequeued,
		msgsAcked,
		msgsRequeued,
		msgsRetried,
		msgsDLQ,
		dlqReviewed,
		dlqRetried,
		dlqCleared,
		queueDepth,
		queueInflight,
		queueFailed,
		batchSize,
		msgWaitTime,
	}

	return collectors
}
