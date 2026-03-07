package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	TasksTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "armorclaw_agent_tasks_total",
		Help: "Total number of agent tasks processed",
	}, []string{"status", "room_id"})

	TasksDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "armorclaw_agent_task_duration_seconds",
		Help:    "Duration of agent tasks in seconds",
		Buckets: prometheus.DefBuckets,
	}, []string{"status"})

	StepsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "armorclaw_agent_steps_total",
		Help: "Total number of reasoning steps",
	}, []string{"type"})

	ToolCallsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "armorclaw_agent_tool_calls_total",
		Help: "Total number of tool calls",
	}, []string{"tool_name", "status"})

	ToolCallDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "armorclaw_agent_tool_call_duration_seconds",
		Help:    "Duration of tool calls in seconds",
		Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
	}, []string{"tool_name"})

	TokensUsed = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "armorclaw_agent_tokens_total",
		Help: "Total tokens used by the agent",
	}, []string{"type"})

	SpeculativeCalls = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "armorclaw_agent_speculative_calls_total",
		Help: "Total speculative tool calls",
	}, []string{"outcome"})

	CacheHits = promauto.NewCounter(prometheus.CounterOpts{
		Name: "armorclaw_cache_hits_total",
		Help: "Total cache hits",
	})

	CacheMisses = promauto.NewCounter(prometheus.CounterOpts{
		Name: "armorclaw_cache_misses_total",
		Help: "Total cache misses",
	})

	CacheEvictions = promauto.NewCounter(prometheus.CounterOpts{
		Name: "armorclaw_cache_evictions_total",
		Help: "Total cache evictions",
	})

	ActiveTasks = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "armorclaw_agent_active_tasks",
		Help: "Number of currently active tasks",
	})

	QueueSize = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "armorclaw_agent_queue_size",
		Help: "Current queue size",
	})

	CircuitBreakerState = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "armorclaw_circuit_breaker_state",
		Help: "Circuit breaker state (0=closed, 1=open, 2=half-open)",
	}, []string{"name"})

	LLMRequests = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "armorclaw_llm_requests_total",
		Help: "Total LLM API requests",
	}, []string{"provider", "model", "status"})

	LLMLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "armorclaw_llm_latency_seconds",
		Help:    "LLM API request latency",
		Buckets: []float64{0.1, 0.25, 0.5, 1, 2.5, 5, 10, 15, 30, 60},
	}, []string{"provider", "model"})

	MemoryOperations = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "armorclaw_memory_operations_total",
		Help: "Total memory store operations",
	}, []string{"operation"})

	RateLimitHits = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "armorclaw_rate_limit_hits_total",
		Help: "Total rate limit hits",
	}, []string{"type"})
)

func RecordTaskStart(roomID string) {
	ActiveTasks.Inc()
}

func RecordTaskComplete(status string, roomID string, duration float64) {
	TasksTotal.WithLabelValues(status, roomID).Inc()
	TasksDuration.WithLabelValues(status).Observe(duration)
	ActiveTasks.Dec()
}

func RecordStep(stepType string) {
	StepsTotal.WithLabelValues(stepType).Inc()
}

func RecordToolCall(toolName string, status string, duration float64) {
	ToolCallsTotal.WithLabelValues(toolName, status).Inc()
	ToolCallDuration.WithLabelValues(toolName).Observe(duration)
}

func RecordTokens(prompt, completion int) {
	TokensUsed.WithLabelValues("prompt").Add(float64(prompt))
	TokensUsed.WithLabelValues("completion").Add(float64(completion))
}

func RecordSpeculativeCall(outcome string) {
	SpeculativeCalls.WithLabelValues(outcome).Inc()
}

func RecordCacheHit() {
	CacheHits.Inc()
}

func RecordCacheMiss() {
	CacheMisses.Inc()
}

func RecordCacheEviction() {
	CacheEvictions.Inc()
}

func RecordLLMRequest(provider, model, status string, latency float64) {
	LLMRequests.WithLabelValues(provider, model, status).Inc()
	LLMLatency.WithLabelValues(provider, model).Observe(latency)
}

func SetCircuitBreakerState(name string, state float64) {
	CircuitBreakerState.WithLabelValues(name).Set(state)
}

func RecordMemoryOperation(op string) {
	MemoryOperations.WithLabelValues(op).Inc()
}

func RecordRateLimitHit(limitType string) {
	RateLimitHits.WithLabelValues(limitType).Inc()
}

func SetQueueSize(size float64) {
	QueueSize.Set(size)
}
