// Package errors provides a comprehensive error handling system for ArmorClaw
// with detailed traces, structured error codes, and LLM-friendly notifications.
//
// # Overview
//
// The errors package implements a robust error handling system that:
//   - Assigns structured error codes (RPC-001, CTX-042, MAT-103, etc.)
//   - Captures detailed traces with call stacks and state snapshots
//   - Tracks component-scoped events for context
//   - Rate-limits notifications to prevent spam
//   - Persists errors to SQLite for later retrieval
//   - Sends LLM-friendly notifications to admins via Matrix
//
// # Quick Start
//
// Basic usage:
//
//	err := errors.NewBuilder("CTX-001").
//	    Wrap(originalError).
//	    WithFunction("StartContainer").
//	    WithInputs(map[string]any{"id": containerID}).
//	    WithState(map[string]any{"running": true}).
//	    Build()
//
//	// Send notification
//	errors.GlobalNotify(context.Background(), err)
//
// # Error Codes
//
// Error codes follow the format CATEGORY-NUMBER:
//   - RPC-001 to RPC-999: RPC layer errors
//   - CTX-001 to CTX-999: Container/runtime errors
//   - MAT-001 to MAT-999: Matrix adapter errors
//   - SYS-001 to SYS-999: System/infrastructure errors
//   - BGT-001+: Budget errors
//   - VOX-001+: Voice/WebRTC errors
//
// # Severity Levels
//
//   - Warning: Non-critical issues that don't break functionality
//   - Error: Operation failed but system continues
//   - Critical: System-level failure requiring immediate attention
//
// # Notification Rules
//
//   - Critical errors: Always notify immediately
//   - First occurrence of code: Notify
//   - Repeats within 5-minute window: Count but don't notify
//   - After window expires: Notify with accumulated count
//
// # Admin Resolution
//
// The system resolves the admin recipient using a 3-tier fallback:
//  1. Explicit config (armorclaw.toml admin_mxid)
//  2. First setup user (captured during wizard)
//  3. Admin room members (first with power level >= 50)
//
// # Message Format
//
// Notifications use a hybrid format for LLM consumption:
//
//	🔴 ERROR: CTX-042
//
//	Container failed to start: permission denied on socket
//
//	📍 Location: StartContainer @ docker/client.go:142
//	🏷️ Trace ID: tr_8f3a2b1c
//	⏰ 2026-02-15 18:32:05 UTC
//
//	```json
//	{
//	  "code": "CTX-042",
//	  "category": "container",
//	  "severity": "error",
//	  ...
//	}
//	```
//
//	📋 Copy the JSON block above to analyze with an LLM.
//
// # Component Tracking
//
// Each package can track events for trace context:
//
//	tracker := errors.GetComponentTracker("docker")
//	tracker.Event("start_container", map[string]any{"id": "abc123"})
//	tracker.Success("start_container", nil)
//	tracker.Failure("start_container", err, nil)
//
// # Integration
//
// Initialize the system at startup:
//
//	system, err := errors.Initialize(errors.Config{
//	    StorePath:       "/var/lib/armorclaw/errors.db",
//	    SetupUserMXID:   "@admin:example.com",
//	    MatrixSender:    matrixAdapter,
//	    Enabled:         true,
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer system.Stop()
//
// # RPC Methods
//
// The system exposes RPC methods for error management:
//   - GetErrors: Query stored errors
//   - ResolveError: Mark an error as resolved
//
// # Thread Safety
//
// All components are thread-safe and can be used concurrently.
package errors
