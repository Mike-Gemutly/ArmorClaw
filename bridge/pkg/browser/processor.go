// Package browser provides browser service integration for the job queue
package browser

import (
	"context"
	"encoding/json"
	 "fmt"
  "sync"
  "time"

  "github.com/armorclaw/bridge/pkg/agent"
  "github.com/armorclaw/bridge/pkg/logger"
  "github.com/armorclaw/bridge/pkg/queue"
)

//=============================================================================
// Service Processor - Connects Queue to Browser Service
//=============================================================================

// ServiceProcessor implements queue.JobProcessor using the browser service HTTP client
type ServiceProcessor struct {
    client         *Client
    stateMachine    *agent.StateMachine
    log             *logger.Logger

    // Job tracking
    mu              sync.RWMutex
    activeJobs      map[string]context.CancelFunc

    // Configuration
    defaultTimeout  time.Duration
    maxRetries      int
}

// ServiceProcessorConfig holds configuration for the service processor
type ServiceProcessorConfig struct {
    // ServiceURL is the browser service HTTP endpoint
    ServiceURL string

    // DefaultTimeout is the default job timeout
    DefaultTimeout time.Duration

    // MaxRetries is the maximum retry count
    MaxRetries int
}

// DefaultServiceProcessorConfig returns sensible defaults
func DefaultServiceProcessorConfig() ServiceProcessorConfig {
    return ServiceProcessorConfig{
        ServiceURL:     "http://localhost:3000",
        DefaultTimeout: 5 * time.Minute,
        MaxRetries:     3,
    }
}

// NewServiceProcessor creates a new browser service processor
func NewServiceProcessor(config ServiceProcessorConfig, log *logger.Logger) *ServiceProcessor {
    if log == nil {
        log = logger.Global().WithComponent("browser_processor")
    }

    return &ServiceProcessor{
        client:         NewClient(config.ServiceURL),
        log:            log,
        activeJobs:     make(map[string]context.CancelFunc),
        defaultTimeout: config.DefaultTimeout,
        maxRetries:     config.MaxRetries,
    }
}

// SetStateMachine sets the agent state machine for status updates
func (p *ServiceProcessor) SetStateMachine(sm *agent.StateMachine) {
    p.stateMachine = sm
}

//=============================================================================
// JobProcessor Implementation
//=============================================================================

// ProcessJob implements queue.JobProcessor
func (p *ServiceProcessor) ProcessJob(ctx context.Context, job *queue.BrowserJob) error {
    p.log.Info("Processing browser job", "job_id", job.ID, "agent_id", job.AgentID)

    // Create cancellable context for this job
    jobCtx, cancel := context.WithCancel(ctx)
    defer cancel()

    // Track active job
    p.mu.Lock()
    p.activeJobs[job.ID] = cancel
    p.mu.Unlock()

    // Cleanup tracking when done
    defer func() {
        p.mu.Lock()
        delete(p.activeJobs, job.ID)
        p.mu.Unlock()
    }()

    // Ensure browser is initialized
    if err := p.ensureBrowserReady(jobCtx); err != nil {
        return fmt.Errorf("browser not ready: %w", err)
    }

    // Update agent state
    if p.stateMachine != nil {
        p.stateMachine.ForceTransition(agent.StatusBrowsing)
    }

    // Convert commands to workflow steps
    steps := p.convertCommandsToSteps(job.Commands)
    if len(steps) == 0 {
        return fmt.Errorf("no commands to execute")
    }

    // Execute workflow
    workflowCmd := ServiceWorkflowCommand{Steps: steps}

    p.log.Debug("Executing workflow", "job_id", job.ID, "steps", len(steps))

    response, err := p.client.Workflow(jobCtx, workflowCmd)
    if err != nil {
        p.log.Error("Workflow execution failed", "job_id", job.ID, "error", err)
        return fmt.Errorf("workflow failed: %w", err)
    }

    // Process response
    if !response.Success {
        if response.Data != nil && len(response.Data.Steps) > 0 {
            // Find the failed step
            for i, step := range response.Data.Steps {
                if !step.Success && step.Error != nil {
                    return fmt.Errorf("step %d failed: %s", i, step.Error.Message)
                }
            }
        }
        return fmt.Errorf("workflow failed with unknown error")
    }

    // Extract results
    job.Result = make(map[string]interface{})
    job.Screenshots = make([]string, 0)

    if response.Data != nil {
        // Collect results from all steps
        for i, step := range response.Data.Steps {
            if step.Success && step.Data != nil {
                // Store step results
                stepKey := fmt.Sprintf("step_%d", i)
                job.Result[stepKey] = step.Data

                // Extract screenshots if present
                if screenshot, ok := step.Data["screenshot"].(string); ok {
                    job.Screenshots = append(job.Screenshots, screenshot)
                }
            }
        }

        job.Result["completed_steps"] = response.Data.CompletedSteps
        job.Result["total_steps"] = response.Data.TotalSteps
    }

    // Update agent state to complete
    if p.stateMachine != nil {
        p.stateMachine.ForceTransition(agent.StatusComplete)
    }

    p.log.Info("Browser job completed successfully", "job_id", job.ID)
    return nil
}

//=============================================================================
// Job Management
//=============================================================================

// CancelJob implements queue.JobProcessor
func (p *ServiceProcessor) CancelJob(jobID string) error {
    p.mu.Lock()
    cancel, exists := p.activeJobs[jobID]
    p.mu.Unlock()

    if !exists {
        return fmt.Errorf("job not found or not active: %s", jobID)
    }

    // Cancel the job context
    cancel()
    p.log.Info("Browser job cancelled", "job_id", jobID)
    return nil
}

//=============================================================================
// Helper Methods
//=============================================================================

// ensureBrowserReady ensures the browser service is initialized
func (p *ServiceProcessor) ensureBrowserReady(ctx context.Context) error {
    // Check health first
    health, err := p.client.Health(ctx)
    if err != nil {
        return fmt.Errorf("browser service health check failed: %w", err)
    }

    // If browser not initialized, initialize it
    if health.Browser != "initialized" {
        p.log.Info("Initializing browser service")
        if err := p.client.Initialize(ctx); err != nil {
            return fmt.Errorf("browser initialization failed: %w", err)
        }
    }
    return nil
}

// convertCommandsToSteps converts queue commands to service workflow steps
func (p *ServiceProcessor) convertCommandsToSteps(commands []queue.BrowserCommand) []ServiceWorkflowStep {
    steps := make([]ServiceWorkflowStep, 0, len(commands))
    for _, cmd := range commands {
        step := ServiceWorkflowStep{
            Action: cmd.Type,
        }
        // Parse command content based on type
        switch cmd.Type {
        case "navigate":
            var navCmd struct {
                URL       string `json:"url"`
                WaitUntil string `json:"waitUntil"`
                Timeout   int    `json:"timeout"`
            }
            if err := json.Unmarshal(cmd.Content, &navCmd); err == nil {
                step.URL = navCmd.URL
                step.WaitUntil = ServiceWaitUntil(navCmd.WaitUntil)
                step.Timeout = navCmd.Timeout
            }
        case "fill":
            var fillCmd struct {
                Fields []struct {
                    Selector string `json:"selector"`
                    Value    string `json:"value"`
                    ValueRef string `json:"value_ref"`
                } `json:"fields"`
                AutoSubmit  bool `json:"auto_submit"`
                SubmitDelay int `json:"submit_delay"`
            }
            if err := json.Unmarshal(cmd.Content, &fillCmd); err == nil {
                step.Fields = make([]ServiceFillField, len(fillCmd.Fields))
                for i, f := range fillCmd.Fields {
                    step.Fields[i] = ServiceFillField{
                        Selector: f.Selector,
                        Value:    f.Value,
                        ValueRef: f.ValueRef,
                    }
                }
                step.AutoSubmit = fillCmd.AutoSubmit
                step.SubmitDelay = fillCmd.SubmitDelay
            }
        case "click":
            var clickCmd struct {
                Selector string `json:"selector"`
                WaitFor  string `json:"waitFor"`
                Timeout  int    `json:"timeout"`
            }
            if err := json.Unmarshal(cmd.Content, &clickCmd); err == nil {
                step.Selector = clickCmd.Selector
                step.WaitFor = clickCmd.WaitFor
                step.Timeout = clickCmd.Timeout
            }
        case "wait":
            var waitCmd struct {
                Condition string `json:"condition"`
                Value     string `json:"value"`
                Timeout   int    `json:"timeout"`
            }
            if err := json.Unmarshal(cmd.Content, &waitCmd); err == nil {
                step.Condition = waitCmd.Condition
                step.Value = waitCmd.Value
                step.Timeout = waitCmd.Timeout
            }
        case "extract":
            var extractCmd struct {
                Fields []struct {
                    Name      string `json:"name"`
                    Selector  string `json:"selector"`
                    Attribute string `json:"attribute"`
                } `json:"fields"`
            }
            if err := json.Unmarshal(cmd.Content, &extractCmd); err == nil {
                for _, f := range extractCmd.Fields {
                    // Create a step for each field extraction
                    extractStep := ServiceWorkflowStep{
                        Action:    "extract",
                        Name:      f.Name,
                        Selector:  f.Selector,
                        Attribute: f.Attribute,
                    }
                    steps = append(steps, extractStep)
                }
                continue // Skip the default append
            }
        case "screenshot":
            var screenshotCmd struct {
                FullPage bool   `json:"fullPage"`
                Selector string `json:"selector"`
                Format   string `json:"format"`
            }
            if err := json.Unmarshal(cmd.Content, &screenshotCmd); err == nil {
                step.FullPage = screenshotCmd.FullPage
                step.Selector = screenshotCmd.Selector
                step.Format = screenshotCmd.Format
            }
        }
        steps = append(steps, step)
    }
    return steps
}
