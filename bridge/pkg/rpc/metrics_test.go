package rpc

import (
	"strings"
	"testing"
)

func TestMetricsExport(t *testing.T) {
	m := NewMetrics()

	m.IncrementCounter("armorclaw_rpc_requests_total", "ai.chat")
	m.IncrementCounter("armorclaw_rpc_requests_total", "matrix.send")
	m.IncrementCounter("armorclaw_rpc_requests_total", "ai.chat")

	m.IncrementCounter("armorclaw_matrix_messages_total", "sent")
	m.IncrementCounter("armorclaw_matrix_messages_total", "received")

	m.IncrementCounter("armorclaw_keystore_operations_total", "store")
	m.IncrementCounter("armorclaw_keystore_operations_total", "retrieve")

	m.SetGauge("armorclaw_active_agents", 3)
	m.SetGauge("armorclaw_uptime_seconds", 123.45)

	output := m.Export()

	t.Run("Has HELP annotations", func(t *testing.T) {
		if !strings.Contains(output, "# HELP") {
			t.Errorf("Output should contain HELP annotations")
		}
	})

	t.Run("Has TYPE annotations", func(t *testing.T) {
		if !strings.Contains(output, "# TYPE") {
			t.Errorf("Output should contain TYPE annotations")
		}
	})

	t.Run("Has rpc_requests_total", func(t *testing.T) {
		if !strings.Contains(output, "armorclaw_rpc_requests_total{method=\"ai.chat\"} 2") {
			t.Errorf("Output should contain rpc_requests_total with correct count")
		}
	})

	t.Run("Has matrix_messages_total", func(t *testing.T) {
		if !strings.Contains(output, "armorclaw_matrix_messages_total{direction=\"sent\"} 1") {
			t.Errorf("Output should contain matrix_messages_total with correct count")
		}
	})

	t.Run("Has keystore_operations_total", func(t *testing.T) {
		if !strings.Contains(output, "armorclaw_keystore_operations_total{operation=\"store\"} 1") {
			t.Errorf("Output should contain keystore_operations_total with correct count")
		}
	})

	t.Run("Has active_agents gauge", func(t *testing.T) {
		if !strings.Contains(output, "armorclaw_active_agents 3") {
			t.Errorf("Output should contain active_agents gauge")
		}
	})

	t.Run("Has uptime_seconds gauge", func(t *testing.T) {
		if !strings.Contains(output, "armorclaw_uptime_seconds 123.45") {
			t.Errorf("Output should contain uptime_seconds gauge")
		}
	})

	t.Run("No sensitive data", func(t *testing.T) {
		sensitive := []string{"password", "secret", "credential", "api_key"}
		lowerOutput := strings.ToLower(output)
		for _, word := range sensitive {
			if strings.Contains(lowerOutput, word) {
				t.Errorf("Output should not contain sensitive word: %s", word)
			}
		}
	})
}

func TestMetricsIncrementCounter(t *testing.T) {
	m := NewMetrics()

	t.Run("Counter increments", func(t *testing.T) {
		m.IncrementCounter("armorclaw_rpc_requests_total", "test.method")
		m.IncrementCounter("armorclaw_rpc_requests_total", "test.method")
		output := m.Export()
		if !strings.Contains(output, "armorclaw_rpc_requests_total{method=\"test.method\"} 2") {
			t.Errorf("Counter should increment correctly")
		}
	})

	t.Run("Unknown counter ignored", func(t *testing.T) {
		outputBefore := m.Export()
		m.IncrementCounter("unknown_metric", "test")
		outputAfter := m.Export()
		if outputBefore != outputAfter {
			t.Errorf("Unknown metrics should be ignored")
		}
	})
}

func TestMetricsSetGauge(t *testing.T) {
	m := NewMetrics()

	t.Run("Gauge sets value", func(t *testing.T) {
		m.SetGauge("armorclaw_active_agents", 5)
		output := m.Export()
		if !strings.Contains(output, "armorclaw_active_agents 5") {
			t.Errorf("Gauge should set value correctly")
		}
	})

	t.Run("Gauge overwrites value", func(t *testing.T) {
		m.SetGauge("armorclaw_active_agents", 5)
		m.SetGauge("armorclaw_active_agents", 10)
		output := m.Export()
		if !strings.Contains(output, "armorclaw_active_agents 10") ||
			strings.Contains(output, "armorclaw_active_agents 5") {
			t.Errorf("Gauge should overwrite previous value")
		}
	})

	t.Run("Unknown gauge ignored", func(t *testing.T) {
		outputBefore := m.Export()
		m.SetGauge("unknown_gauge", 5)
		outputAfter := m.Export()
		if outputBefore != outputAfter {
			t.Errorf("Unknown gauges should be ignored")
		}
	})

	t.Run("Negative gauge ignored", func(t *testing.T) {
		m.SetGauge("armorclaw_active_agents", 5)
		m.SetGauge("armorclaw_active_agents", -1)
		output := m.Export()
		if !strings.Contains(output, "armorclaw_active_agents 5") ||
			strings.Contains(output, "armorclaw_active_agents -1") {
			t.Errorf("Negative gauge values should be ignored")
		}
	})
}
