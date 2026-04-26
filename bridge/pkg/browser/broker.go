package browser

import "context"

// BrowserBroker manages browser automation jobs and exposes semantic
// operations consumed by the handler layer and queue subsystem.
type BrowserBroker interface {
	StartJob(ctx context.Context, req StartJobRequest) (JobID, error)
	Status(ctx context.Context, id JobID) (*JobSummary, error)
	Complete(ctx context.Context, id JobID) (*BrokerResult, error)
	Fail(ctx context.Context, id JobID, reason string) error
	List(ctx context.Context, agentID string) ([]JobSummary, error)
	Cancel(ctx context.Context, id JobID) error

	Navigate(ctx context.Context, id JobID, url string) (*BrokerResult, error)

	// Fill injects values into form fields. When a field's Sensitive flag
	// is true, the broker routes it through the PII approval path before
	// injecting via ServiceFillField.ValueRef.
	Fill(ctx context.Context, id JobID, fields []FillRequest) (*BrokerResult, error)

	Click(ctx context.Context, id JobID, selector string) (*BrokerResult, error)
	WaitForElement(ctx context.Context, id JobID, selector string, timeoutMs int) (*BrokerResult, error)
	WaitForCaptcha(ctx context.Context, id JobID, timeoutMs int) (*BrokerResult, error)
	WaitFor2FA(ctx context.Context, id JobID, timeoutMs int) (*BrokerResult, error)
	Extract(ctx context.Context, id JobID, spec ExtractSpec) (*ExtractResult, error)
	Screenshot(ctx context.Context, id JobID, fullPage bool) (*BrokerResult, error)
}
