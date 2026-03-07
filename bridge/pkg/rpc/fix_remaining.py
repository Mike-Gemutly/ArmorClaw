#!/usr/bin/env python3

def fix_all_handlers():
    with open('browser.go', 'r') as f:
        content = f.read()

    # Fix handleBrowserStatus completely
    status_fix = '''// handleBrowserStatus handles browser.status RPC method
// Polls for job completion and returns current state
func (s *Server) handleBrowserStatus(ctx context.Context, req *Request) (interface{}, *ErrorObj) {
	var params struct {
		JobID string `json:"job_id"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: "invalid parameters: " + err.Error(),
		}
	}

	if params.JobID == "" {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: "job_id is required",
		}
	}

	// Get job
	job, exists := s.browserJobs.GetJob(params.JobID)
	if !exists {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: "job not found: " + params.JobID,
		}
	}

	result := map[string]interface{}{
		"job_id":     job.ID,
		"agent_id":   job.AgentID,
		"status":     job.Status,
		"url":        job.URL,
		"created_at": job.CreatedAt.Format(time.RFC3339),
	}

	if job.CompletedAt != nil {
		result["completed_at"] = job.CompletedAt.Format(time.RFC3339)
	}

	if job.Error != "" {
		result["error"] = job.Error
	}

	if job.Session != nil {
		result["session"] = map[string]interface{}{
			"id":            job.Session.ID,
			"url":           job.Session.URL,
			"status":        string(job.Session.Status),
			"title":         job.Session.Title,
			"last_activity": job.Session.LastActivity.Format(time.RFC3339),
		}
	}

	// Include screenshot placeholder (would be base64 in real implementation)
	// In a full implementation, this would capture actual screenshots
	result["screenshot_available"] = false

	return result, nil
}'''

    # Fix handleBrowserWaitForElement
    wait_for_element_fix = '''// handleBrowserWaitForElement handles browser.wait_for_element RPC method
// Waits for an element to appear on the page
func (s *Server) handleBrowserWaitForElement(ctx context.Context, req *Request) (interface{}, *ErrorObj) {
	var params struct {
		JobID    string `json:"job_id"`
		Selector string `json:"selector"`
		Timeout  int    `json:"timeout"` // seconds
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: "invalid parameters: " + err.Error(),
		}
	}

	if params.JobID == "" {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: "job_id is required",
		}
	}

	if params.Selector == "" {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: "selector is required",
		}
	}

	if params.Timeout == 0 {
		params.Timeout = 30
	}

	// Get job
	job, exists := s.browserJobs.GetJob(params.JobID)
	if !exists {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: "job not found: " + params.JobID,
		}
	}

	// Wait for element
	ctx, cancel := context.WithTimeout(ctx, time.Duration(params.Timeout)*time.Second)
	defer cancel()

	err := job.skill.WaitForElement(ctx, params.Selector, time.Duration(params.Timeout)*time.Second)
	if err != nil {
		return nil, &ErrorObj{
			Code:    InternalError,
			Message: "element not found: " + err.Error(),
		}
	}

	return map[string]interface{}{
		"job_id":   params.JobID,
		"status":   "element_found",
		"selector": params.Selector,
		"success":  true,
	}, nil
}'''

    # Fix handleBrowserWaitForCaptcha
    wait_for_captcha_fix = '''// handleBrowserWaitForCaptcha handles browser.wait_for_captcha RPC method
// Signals that the agent is waiting for captcha resolution
func (s *Server) handleBrowserWaitForCaptcha(ctx context.Context, req *Request) (interface{}, *ErrorObj) {
	var params struct {
		JobID string `json:"job_id"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: "invalid parameters: " + err.Error(),
		}
	}

	if params.JobID == "" {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: "job_id is required",
		}
	}

	// Get job
	job, exists := s.browserJobs.GetJob(params.JobID)
	if !exists {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: "job not found: " + params.JobID,
		}
	}

	// Emit captcha status
	job.skill.WaitForCaptcha(ctx)

	return map[string]interface{}{
		"job_id":  params.JobID,
		"status":  "awaiting_captcha",
		"message": "Captcha detection started. Mobile app will be notified.",
	}, nil
}'''

    # Fix handleBrowserWaitFor2FA
    wait_for_2fa_fix = '''// handleBrowserWaitFor2FA handles browser.wait_for_2fa RPC method
// Signals that the agent is waiting for 2FA code
func (s *Server) handleBrowserWaitFor2FA(ctx context.Context, req *Request) (interface{}, *ErrorObj) {
	var params struct {
		JobID string `json:"job_id"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: "invalid parameters: " + err.Error(),
		}
	}

	if params.JobID == "" {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: "job_id is required",
		}
	}

	// Get job
	job, exists := s.browserJobs.GetJob(params.JobID)
	if !exists {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: "job not found: " + params.JobID,
		}
	}

	// Emit 2FA status
	job.skill.WaitFor2FA(ctx)

	return map[string]interface{}{
		"job_id":  params.JobID,
		"status":  "awaiting_2fa",
		"message": "2FA detection started. Mobile app will be notified.",
	}, nil
}'''

    # Fix handleBrowserComplete
    complete_fix = '''// handleBrowserComplete handles browser.complete RPC method
// Marks a browser job as complete
func (s *Server) handleBrowserComplete(ctx context.Context, req *Request) (interface{}, *ErrorObj) {
	var params struct {
		JobID string `json:"job_id"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: "invalid parameters: " + err.Error(),
		}
	}

	if params.JobID == "" {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: "job_id is required",
		}
	}

	// Get job
	job, exists := s.browserJobs.GetJob(params.JobID)
	if !exists {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: "job not found: " + params.JobID,
		}
	}

	// Mark complete
	job.skill.Complete(ctx)
	job.Status = "completed"
	now := time.Now()
	job.CompletedAt = &now
	job.Session = job.skill.GetSession()

	return map[string]interface{}{
		"job_id":       params.JobID,
		"status":       "completed",
		"completed_at": job.CompletedAt.Format(time.RFC3339),
		"success":      true,
	}, nil
}'''

    # Fix handleBrowserFail
    fail_fix = '''// handleBrowserFail handles browser.fail RPC method
// Marks a browser job as failed
func (s *Server) handleBrowserFail(ctx context.Context, req *Request) (interface{}, *ErrorObj) {
	var params struct {
		JobID  string `json:"job_id"`
		Reason string `json:"reason"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: "invalid parameters: " + err.Error(),
		}
	}

	if params.JobID == "" {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: "job_id is required",
		}
	}

	// Get job
	job, exists := s.browserJobs.GetJob(params.JobID)
	if !exists {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: "job not found: " + params.JobID,
		}
	}

	// Mark failed
	job.skill.Fail(ctx, fmt.Errorf("%s", params.Reason))
	job.Status = "failed"
	job.Error = params.Reason
	now := time.Now()
	job.CompletedAt = &now

	return map[string]interface{}{
		"job_id":  params.JobID,
		"status":  "failed",
		"error":   params.Reason,
		"success": true,
	}, nil
}'''

    # Fix handleBrowserList
    list_fix = '''// handleBrowserList handles browser.list RPC method
// Lists all active browser jobs
func (s *Server) handleBrowserList(ctx context.Context, req *Request) (interface{}, *ErrorObj) {
	jobs := s.browserJobs.ListJobs()

	result := make([]map[string]interface{}, 0, len(jobs))
	for _, job := range jobs {
		jobInfo := map[string]interface{}{
			"job_id":     job.ID,
			"agent_id":   job.AgentID,
			"url":        job.URL,
			"status":     job.Status,
			"created_at": job.CreatedAt.Format(time.RFC3339),
		}
		if job.CompletedAt != nil {
			jobInfo["completed_at"] = job.CompletedAt.Format(time.RFC3339)
		}
		if job.Error != "" {
			jobInfo["error"] = job.Error
		}
		result = append(result, jobInfo)
	}

	return map[string]interface{}{
		"jobs":  result,
		"count": len(result),
	}, nil
}'''

    # Fix handleBrowserCancel
    cancel_fix = '''// handleBrowserCancel handles browser.cancel RPC method
// Cancels an active browser job
func (s *Server) handleBrowserCancel(ctx context.Context, req *Request) (interface{}, *ErrorObj) {
	var params struct {
		JobID string `json:"job_id"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: "invalid parameters: " + err.Error(),
		}
	}

	if params.JobID == "" {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: "job_id is required",
		}
	}

	// Get job
	job, exists := s.browserJobs.GetJob(params.JobID)
	if !exists {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: "job not found: " + params.JobID,
		}
	}

	// Cancel job
	if job.cancelFunc != nil {
		job.cancelFunc()
	}
	job.Status = "cancelled"
	now := time.Now()
	job.CompletedAt = &now

	return map[string]interface{}{
		"job_id":       params.JobID,
		"status":       "cancelled",
		"cancelled_at": job.CompletedAt.Format(time.RFC3339),
		"success":      true,
	}, nil
}'''

    # Apply all fixes
    content = content.replace('''// handleBrowserStatus handles browser.status RPC method
// Polls for job completion and returns current state
func (s *Server) handleBrowserStatus(ctx context.Context, req *Request) (interface{}, *ErrorObj) {
	var params struct {
		JobID string `json:"job_id"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, &ErrorObj{
				Code: InvalidParams,
				Message: "invalid parameters: " + err.Error(),
			},
			}, nil

	}

	if params.JobID == "" {
		return nil, &ErrorObj{
				Code: InvalidParams,
				Message: "job_id is required",
			},
			}, nil

	}

	// Get job
	job, exists := s.browserJobs.GetJob(params.JobID)
	if !exists {
		return nil, &ErrorObj{
				Code: InvalidParams,
				Message: "job not found: " + params.JobID,
			},
			}, nil

	}

	result := map[string]interface{}{
		"job_id":     job.ID,
		"agent_id":   job.AgentID,
		"status":     job.Status,
		"url":        job.URL,
		"created_at": job.CreatedAt.Format(time.RFC3339),
	}

	if job.CompletedAt != nil {
		result["completed_at"] = job.CompletedAt.Format(time.RFC3339)
	}

	if job.Error != "" {
		result["error"] = job.Error
	}

	if job.Session != nil {
		result["session"] = map[string]interface{}{
			"id":            job.Session.ID,
			"url":           job.Session.URL,
			"status":        string(job.Session.Status),
			"title":         job.Session.Title,
			"last_activity": job.Session.LastActivity.Format(time.RFC3339),
		}, nil

	}

	// Include screenshot placeholder (would be base64 in real implementation)
	// In a full implementation, this would capture actual screenshots
	result["screenshot_available"] = false

	return result, nil
}''', status_fix)

    content = content.replace('''// handleBrowserWaitForElement handles browser.wait_for_element RPC method
// Waits for an element to appear on the page
func (s *Server) handleBrowserWaitForElement(ctx context.Context, req *Request) (interface{}, *ErrorObj) {
	var params struct {
		JobID    string `json:"job_id"`
		Selector string `json:"selector"`
		Timeout  int    `json:"timeout"` // seconds
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, &ErrorObj{
				Code: InvalidParams,
				Message: "invalid parameters: " + err.Error(),
			},
			}, nil

	}

	if params.JobID == "" {
		return nil, &ErrorObj{
				Code: InvalidParams,
				Message: "job_id is required",
			},
			}, nil

	}

	if params.Selector == "" {
		return nil, &ErrorObj{
				Code: InvalidParams,
				Message: "selector is required",
			},
			}, nil

	}

	if params.Timeout == 0 {
		params.Timeout = 30
	}

	// Get job
	job, exists := s.browserJobs.GetJob(params.JobID)
	if !exists {
		return nil, &ErrorObj{
				Code: InvalidParams,
				Message: "job not found: " + params.JobID,
			},
			}, nil

	}

	// Wait for element
	ctx, cancel := context.WithTimeout(s.ctx, time.Duration(params.Timeout)*time.Second)
	defer cancel()

	err := job.skill.WaitForElement(ctx, params.Selector, time.Duration(params.Timeout)*time.Second)
	if err != nil {
		return nil, &ErrorObj{
				Code: InternalError,
				Message: "element not found: " + err.Error(),
			},
			}, nil

	}

	return map[string]interface{}, nil{
			"job_id":   params.JobID,
			"status":   "element_found",
			"selector": params.Selector,
			"success":  true,
		}, nil

	}
}''', wait_for_element_fix)

    content = content.replace('''// handleBrowserWaitForCaptcha handles browser.wait_for_captcha RPC method
// Signals that the agent is waiting for captcha resolution
func (s *Server) handleBrowserWaitForCaptcha(ctx context.Context, req *Request) (interface{}, *ErrorObj) {
	var params struct {
		JobID string `json:"job_id"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, &ErrorObj{
				Code: InvalidParams,
				Message: "invalid parameters: " + err.Error(),
			},
			}, nil

	}

	if params.JobID == "" {
		return nil, &ErrorObj{
				Code: InvalidParams,
				Message: "job_id is required",
			},
			}, nil

	}

	// Get job
	job, exists := s.browserJobs.GetJob(params.JobID)
	if !exists {
		return nil, &ErrorObj{
				Code: InvalidParams,
				Message: "job not found: " + params.JobID,
			},
			}, nil

	}

	// Emit captcha status
	ctx := s.ctx
	job.skill.WaitForCaptcha(ctx)

	return map[string]interface{}, nil{
			"job_id":  params.JobID,
			"status":  "awaiting_captcha",
			"message": "Captcha detection started. Mobile app will be notified.",
		}, nil

	}
}''', wait_for_captcha_fix)

    content = content.replace('''// handleBrowserWaitFor2FA handles browser.wait_for_2fa RPC method
// Signals that the agent is waiting for 2FA code
func (s *Server) handleBrowserWaitFor2FA(ctx context.Context, req *Request) (interface{}, *ErrorObj) {
	var params struct {
		JobID string `json:"job_id"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, &ErrorObj{
				Code: InvalidParams,
				Message: "invalid parameters: " + err.Error(),
			},
			}, nil

	}

	if params.JobID == "" {
		return nil, &ErrorObj{
				Code: InvalidParams,
				Message: "job_id is required",
			},
			}, nil

	}

	// Get job
	job, exists := s.browserJobs.GetJob(params.JobID)
	if !exists {
		return nil, &ErrorObj{
				Code: InvalidParams,
				Message: "job not found: " + params.JobID,
			},
			}, nil

	}

	// Emit 2FA status
	ctx := s.ctx
	job.skill.WaitFor2FA(ctx)

	return map[string]interface{}, nil{
			"job_id":  params.JobID,
			"status":  "awaiting_2fa",
			"message": "2FA detection started. Mobile app will be notified.",
		}, nil

	}
}''', wait_for_2fa_fix)

    content = content.replace('''// handleBrowserComplete handles browser.complete RPC method
// Marks a browser job as complete
func (s *Server) handleBrowserComplete(ctx context.Context, req *Request) (interface{}, *ErrorObj) {
	var params struct {
		JobID string `json:"job_id"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, &ErrorObj{
				Code: InvalidParams,
				Message: "invalid parameters: " + err.Error(),
			},
			}, nil

	}

	if params.JobID == "" {
		return nil, &ErrorObj{
				Code: InvalidParams,
				Message: "job_id is required",
			},
			}, nil

	}

	// Get job
	job, exists := s.browserJobs.GetJob(params.JobID)
	if !exists {
		return nil, &ErrorObj{
				Code: InvalidParams,
				Message: "job not found: " + params.JobID,
			},
			}, nil

	}

	// Mark complete
	ctx := s.ctx
	job.skill.Complete(ctx)
	job.Status = "completed"
	now := time.Now()
	job.CompletedAt = &now
	job.Session = job.skill.GetSession()

	return map[string]interface{}, nil{
			"job_id":       params.JobID,
			"status":       "completed",
			"completed_at": job.CompletedAt.Format(time.RFC3339),
			"success":      true,
		}, nil

	}
}''', complete_fix)

    content = content.replace('''// handleBrowserFail handles browser.fail RPC method
// Marks a browser job as failed
func (s *Server) handleBrowserFail(ctx context.Context, req *Request) (interface{}, *ErrorObj) {
	var params struct {
		JobID  string `json:"job_id"`
		Reason string `json:"reason"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, &ErrorObj{
				Code: InvalidParams,
				Message: "invalid parameters: " + err.Error(),
			},
			}, nil

	}

	if params.JobID == "" {
		return nil, &ErrorObj{
				Code: InvalidParams,
				Message: "job_id is required",
			},
			}, nil

	}

	// Get job
	job, exists := s.browserJobs.GetJob(params.JobID)
	if !exists {
		return nil, &ErrorObj{
				Code: InvalidParams,
				Message: "job not found: " + params.JobID,
			},
			}, nil

	}

	// Mark failed
	ctx := s.ctx
	job.skill.Fail(ctx, fmt.Errorf("%s", params.Reason))
	job.Status = "failed"
	job.Error = params.Reason
	now := time.Now()
	job.CompletedAt = &now

	return map[string]interface{}, nil{
			"job_id":  params.JobID,
			"status":  "failed",
			"error":   params.Reason,
			"success": true,
		}, nil

	}
}''', fail_fix)

    content = content.replace('''// handleBrowserList handles browser.list RPC method
// Lists all active browser jobs
func (s *Server) handleBrowserList(ctx context.Context, req *Request) (interface{}, *ErrorObj) {
	jobs := s.browserJobs.ListJobs()

	result := make([]map[string]interface{}, 0, len(jobs))
	for _, job := range jobs {
		jobInfo := map[string]interface{}{
			"job_id":     job.ID,
			"agent_id":   job.AgentID,
			"url":        job.URL,
			"status":     job.Status,
			"created_at": job.CreatedAt.Format(time.RFC3339),
		}, nil

		if job.CompletedAt != nil {
			jobInfo["completed_at"] = job.CompletedAt.Format(time.RFC3339)
		}
		if job.Error != "" {
			jobInfo["error"] = job.Error
		}
		result = append(result, jobInfo)
	}

	return map[string]interface{}, nil{
			"jobs":  result,
			"count": len(result),
		}, nil

	}
}''', list_fix)

    content = content.replace('''// handleBrowserCancel handles browser.cancel RPC method
// Cancels an active browser job
func (s *Server) handleBrowserCancel(ctx context.Context, req *Request) (interface{}, *ErrorObj) {
	var params struct {
		JobID string `json:"job_id"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, &ErrorObj{
				Code: InvalidParams,
				Message: "invalid parameters: " + err.Error(),
			},
			}, nil

	}

	if params.JobID == "" {
		return nil, &ErrorObj{
				Code: InvalidParams,
				Message: "job_id is required",
			},
			}, nil

	}

	// Get job
	job, exists := s.browserJobs.GetJob(params.JobID)
	if !exists {
		return nil, &ErrorObj{
				Code: InvalidParams,
				Message: "job not found: " + params.JobID,
			},
			}, nil

	}

	// Cancel job
	if job.cancelFunc != nil {
		job.cancelFunc()
	}
	job.Status = "cancelled"
	now := time.Now()
	job.CompletedAt = &now

	return map[string]interface{}, nil{
			"job_id":       params.JobID,
			"status":       "cancelled",
			"cancelled_at": job.CompletedAt.Format(time.RFC3339),
			"success":      true,
		}, nil

	}
}''', cancel_fix)

    # Final cleanup
    content = content.replace(',\n,', ',\n')
    content = content.replace(',\t,', '\t,')
    content = content.replace('}, nil\n,', '}, nil\n')
    content = content.replace('}, nil\n,', '}, nil\n')
    content = content.replace('}, nil\n,', '}, nil\n')
    content = content.replace('}, nil\n,', '}, nil\n')

    with open('browser.go', 'w') as f:
        f.write(content)

    print("Fixed all remaining handlers in browser.go")

fix_all_handlers()
