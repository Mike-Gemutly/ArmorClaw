package rpc

import (
	"fmt"
	"sync"
	"time"
)

// Metrics holds Prometheus metrics for ArmorClaw bridge
// Thread-safe using RWMutex for concurrent access
type Metrics struct {
	mu sync.RWMutex

	startTime           time.Time
	rpcRequestsTotal    map[string]uint64
	matrixMessagesTotal map[string]uint64
	keystoreOpsTotal    map[string]uint64
	activeAgents        int
	uptime              float64
}

// NewMetrics creates a new Metrics instance
func NewMetrics() *Metrics {
	return &Metrics{
		startTime:           time.Now(),
		rpcRequestsTotal:    make(map[string]uint64),
		matrixMessagesTotal: make(map[string]uint64),
		keystoreOpsTotal:    make(map[string]uint64),
		activeAgents:        0,
	}
}

// IncrementCounter increments a counter metric with labels
func (m *Metrics) IncrementCounter(name string, labels string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	switch name {
	case "armorclaw_rpc_requests_total":
		m.rpcRequestsTotal[labels]++
	case "armorclaw_matrix_messages_total":
		m.matrixMessagesTotal[labels]++
	case "armorclaw_keystore_operations_total":
		m.keystoreOpsTotal[labels]++
	}
}

// SetGauge sets a gauge metric value
func (m *Metrics) SetGauge(name string, value float64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	switch name {
	case "armorclaw_active_agents":
		if value >= 0 {
			m.activeAgents = int(value)
		}
	case "armorclaw_uptime_seconds":
		if value >= 0 {
			m.uptime = value
		}
	}
}

// UpdateUptime updates the uptime gauge based on start time
func (m *Metrics) UpdateUptime() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.uptime = time.Since(m.startTime).Seconds()
}

// Export returns metrics in Prometheus text format
// Format: https://prometheus.io/docs/instrumenting/exposition_formats/
// Content-Type: text/plain; version=0.0.4
func (m *Metrics) Export() string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var output string

	output += "# HELP armorclaw_rpc_requests_total Total number of RPC requests handled\n"
	output += "# TYPE armorclaw_rpc_requests_total counter\n"
	for method, count := range m.rpcRequestsTotal {
		output += fmt.Sprintf("armorclaw_rpc_requests_total{method=\"%s\"} %d\n", method, count)
	}

	output += "# HELP armorclaw_active_agents Number of currently active agent containers\n"
	output += "# TYPE armorclaw_active_agents gauge\n"
	output += fmt.Sprintf("armorclaw_active_agents %d\n", m.activeAgents)

	output += "# HELP armorclaw_matrix_messages_total Total number of Matrix messages sent/received\n"
	output += "# TYPE armorclaw_matrix_messages_total counter\n"
	for direction, count := range m.matrixMessagesTotal {
		output += fmt.Sprintf("armorclaw_matrix_messages_total{direction=\"%s\"} %d\n", direction, count)
	}

	output += "# HELP armorclaw_keystore_operations_total Total number of keystore operations\n"
	output += "# TYPE armorclaw_keystore_operations_total counter\n"
	for op, count := range m.keystoreOpsTotal {
		output += fmt.Sprintf("armorclaw_keystore_operations_total{operation=\"%s\"} %d\n", op, count)
	}

	output += "# HELP armorclaw_uptime_seconds Server uptime in seconds\n"
	output += "# TYPE armorclaw_uptime_seconds gauge\n"
	output += fmt.Sprintf("armorclaw_uptime_seconds %.2f\n", m.uptime)

	return output
}
