package browser

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/armorclaw/bridge/pkg/capability"
)

// BrokerHandler wraps a BrowserBroker into the same
// func(ctx context.Context, config json.RawMessage) (json.RawMessage, error)
// signature used by Handler(client). It dispatches browser actions the same
// way as handler.go:72-87 but calls broker methods instead of client methods.
//
// Each broker call is logged with method, jobID, duration, and error.
func BrokerHandler(broker BrowserBroker) func(ctx context.Context, config json.RawMessage) (json.RawMessage, error) {
	return func(ctx context.Context, config json.RawMessage) (json.RawMessage, error) {
		var req browserRequest
		if err := json.Unmarshal(config, &req); err != nil {
			return nil, fmt.Errorf("browser_execute: parse config: %w", err)
		}

		intent := req.Intent
		if intent == nil {
			intent = &capability.BrowserIntent{
				URL:        req.URL,
				Action:     req.Action,
				FormFields: req.FormFields,
			}
		}

		if err := intent.Validate(); err != nil {
			return nil, fmt.Errorf("browser_execute: %w", err)
		}

		result, err := brokerDispatchAction(ctx, broker, intent)
		if err != nil {
			return nil, fmt.Errorf("browser_execute: %w", err)
		}

		raw, err := json.Marshal(result)
		if err != nil {
			return nil, fmt.Errorf("browser_execute: marshal result: %w", err)
		}
		return raw, nil
	}
}

func logBrokerCall(method string, jobID JobID, start time.Time, err error) {
	duration := time.Since(start)
	if err != nil {
		log.Printf("browser broker: %s job_id=%s duration=%v error=%v", method, jobID, duration, err)
	} else {
		log.Printf("browser broker: %s job_id=%s duration=%v ok", method, jobID, duration)
	}
}

// brokerDispatchAction dispatches a browser action to the broker.
// It produces the same BrowserResultCompat output as handler.go:72-87.
func brokerDispatchAction(ctx context.Context, broker BrowserBroker, intent *capability.BrowserIntent) (*BrowserResultCompat, error) {
	switch intent.Action {
	case "navigate":
		return brokerNavigateAction(ctx, broker, intent)
	case "fill":
		return brokerFillAction(ctx, broker, intent)
	case "extract":
		return brokerExtractAction(ctx, broker, intent)
	case "screenshot":
		return brokerScreenshotAction(ctx, broker, intent)
	case "workflow":
		return brokerWorkflowAction(ctx, broker, intent)
	default:
		return nil, fmt.Errorf("unsupported browser action: %s", intent.Action)
	}
}

func brokerNavigateAction(ctx context.Context, broker BrowserBroker, intent *capability.BrowserIntent) (*BrowserResultCompat, error) {
	// Start a job for this navigation.
	start := time.Now()
	jobID, err := broker.StartJob(ctx, StartJobRequest{URL: intent.URL})
	if err != nil {
		return nil, fmt.Errorf("broker navigate: start job: %w", err)
	}
	defer func() {
		_, _ = broker.Complete(ctx, jobID)
	}()

	br, err := broker.Navigate(ctx, jobID, intent.URL)
	logBrokerCall("navigate", jobID, start, err)
	if err != nil {
		return nil, err
	}

	return br.ToBrowserResult(), nil
}

func brokerFillAction(ctx context.Context, broker BrowserBroker, intent *capability.BrowserIntent) (*BrowserResultCompat, error) {
	start := time.Now()
	jobID, err := broker.StartJob(ctx, StartJobRequest{URL: intent.URL})
	if err != nil {
		return nil, fmt.Errorf("broker fill: start job: %w", err)
	}
	defer func() {
		_, _ = broker.Complete(ctx, jobID)
	}()

	var fields []FillRequest
	for _, ff := range intent.FormFields {
		fields = append(fields, FillRequest{Selector: ff})
	}

	br, err := broker.Fill(ctx, jobID, fields)
	logBrokerCall("fill", jobID, start, err)
	if err != nil {
		return nil, err
	}

	return br.ToBrowserResult(), nil
}

func brokerExtractAction(ctx context.Context, broker BrowserBroker, intent *capability.BrowserIntent) (*BrowserResultCompat, error) {
	start := time.Now()
	jobID, err := broker.StartJob(ctx, StartJobRequest{URL: intent.URL})
	if err != nil {
		return nil, fmt.Errorf("broker extract: start job: %w", err)
	}
	defer func() {
		_, _ = broker.Complete(ctx, jobID)
	}()

	spec := ExtractSpec{}
	for _, ff := range intent.FormFields {
		spec.Fields = append(spec.Fields, ExtractField{Name: ff, Selector: ff})
	}

	extractResult, err := broker.Extract(ctx, jobID, spec)
	logBrokerCall("extract", jobID, start, err)
	if err != nil {
		return nil, err
	}

	result := &BrowserResultCompat{URL: intent.URL}
	if extractResult != nil {
		for k, v := range extractResult.Fields {
			result.ExtractedData = append(result.ExtractedData, fmt.Sprintf("%s=%v", k, v))
		}
		if extractResult.Screenshot != "" {
			result.Screenshots = append(result.Screenshots, extractResult.Screenshot)
		}
	}
	return result, nil
}

func brokerScreenshotAction(ctx context.Context, broker BrowserBroker, intent *capability.BrowserIntent) (*BrowserResultCompat, error) {
	start := time.Now()
	jobID, err := broker.StartJob(ctx, StartJobRequest{URL: intent.URL})
	if err != nil {
		return nil, fmt.Errorf("broker screenshot: start job: %w", err)
	}
	defer func() {
		_, _ = broker.Complete(ctx, jobID)
	}()

	br, err := broker.Screenshot(ctx, jobID, false)
	logBrokerCall("screenshot", jobID, start, err)
	if err != nil {
		return nil, err
	}

	return br.ToBrowserResult(), nil
}

func brokerWorkflowAction(ctx context.Context, broker BrowserBroker, intent *capability.BrowserIntent) (*BrowserResultCompat, error) {
	start := time.Now()

	steps := make([]string, len(intent.FormFields))
	copy(steps, intent.FormFields)

	jobID, err := broker.StartJob(ctx, StartJobRequest{
		URL:   intent.URL,
		Steps: steps,
	})
	if err != nil {
		return nil, fmt.Errorf("broker workflow: start job: %w", err)
	}
	defer func() {
		_, _ = broker.Complete(ctx, jobID)
	}()

	// Execute navigate as the first step.
	br, err := broker.Navigate(ctx, jobID, intent.URL)
	if err != nil {
		logBrokerCall("workflow", jobID, start, err)
		return nil, err
	}

	result := br.ToBrowserResult()

	// Execute remaining steps via fill (matches handler.go workflow behavior).
	for _, step := range intent.FormFields {
		_, err := broker.Fill(ctx, jobID, []FillRequest{{Selector: step}})
		if err != nil {
			logBrokerCall("workflow", jobID, start, err)
			return nil, err
		}
	}

	logBrokerCall("workflow", jobID, start, nil)
	return result, nil
}

// FallbackHandler wraps a primary handler and falls back to a secondary handler
// when the primary fails. It logs a warning on fallback.
func FallbackHandler(primary, fallback func(ctx context.Context, config json.RawMessage) (json.RawMessage, error)) func(ctx context.Context, config json.RawMessage) (json.RawMessage, error) {
	return func(ctx context.Context, config json.RawMessage) (json.RawMessage, error) {
		result, err := primary(ctx, config)
		if err != nil {
			log.Printf("browser broker: jetski failed, falling back to legacy: %v", err)
			return fallback(ctx, config)
		}
		return result, nil
	}
}
