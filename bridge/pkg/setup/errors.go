// Package setup provides setup and installation utilities for ArmorClaw.
// This includes typed errors with actionable guidance for common setup failures.
package setup

import (
	"fmt"
	"strings"
)

// SetupError represents a setup failure with actionable guidance.
// It implements the error interface and provides rich context for
// debugging and user guidance.
type SetupError struct {
	Code        string   // Error code (e.g., "INS-001")
	Title       string   // Short human-readable title
	Cause       string   // Root cause explanation
	Fix         []string // Step-by-step remediation instructions
	DocLink     string   // Link to documentation (optional)
	ExitCode    int      // Suggested exit code (default: 1)
}

// Error implements the error interface.
func (e *SetupError) Error() string {
	return fmt.Sprintf("[%s] %s", e.Code, e.Title)
}

// FullError returns the complete error message with fix instructions.
func (e *SetupError) FullError() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("\n  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n"))
	sb.WriteString(fmt.Sprintf("  ERROR [%s]: %s\n", e.Code, e.Title))
	sb.WriteString(fmt.Sprintf("  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n"))

	if e.Cause != "" {
		sb.WriteString(fmt.Sprintf("  Cause:\n  %s\n\n", e.Cause))
	}

	if len(e.Fix) > 0 {
		sb.WriteString("  Fix:\n")
		for i, step := range e.Fix {
			sb.WriteString(fmt.Sprintf("  %d. %s\n", i+1, step))
		}
		sb.WriteString("\n")
	}

	if e.DocLink != "" {
		sb.WriteString(fmt.Sprintf("  Learn more: %s\n", e.DocLink))
	}

	return sb.String()
}

// Predefined setup errors

var (
	// ErrDockerSocket is returned when the Docker socket is not accessible.
	ErrDockerSocket = &SetupError{
		Code:     "INS-001",
		Title:    "Docker socket not accessible",
		Cause:    "The Docker socket at /var/run/docker.sock is not accessible. ArmorClaw requires Docker to orchestrate the Matrix stack.",
		Fix: []string{
			"Ensure Docker socket is mounted: -v /var/run/docker.sock:/var/run/docker.sock",
			"Run with --user root or add your user to the docker group",
			"Verify Docker is running: docker ps",
		},
		DocLink:  "https://docs.armorclaw.com/errors/ins-001",
		ExitCode: 1,
	}

	// ErrDockerPermission is returned when the user lacks Docker permissions.
	ErrDockerPermission = &SetupError{
		Code:     "INS-010",
		Title:    "Docker permission denied",
		Cause:    "Your user account lacks permission to access the Docker socket.",
		Fix: []string{
			"Add your user to the docker group: sudo usermod -aG docker $USER",
			"Log out and log back in, or run: newgrp docker",
			"Verify access with: docker ps",
		},
		DocLink:  "https://docs.docker.com/engine/install/linux-postinstall/",
		ExitCode: 1,
	}

	// ErrTerminalNotTTY is returned when TUI is launched without a proper terminal.
	ErrTerminalNotTTY = &SetupError{
		Code:     "INS-002",
		Title:    "Interactive terminal required",
		Cause:    "The TUI wizard requires an interactive terminal (TTY). You may be running in a pipe or without -it flags.",
		Fix: []string{
			"Run with -it flags: docker run -it ...",
			"Or use environment variables for non-interactive setup:",
			"  -e ARMORCLAW_API_KEY=sk-your-key",
			"  -e ARMORCLAW_SERVER_NAME=your-server",
		},
		DocLink:  "https://docs.armorclaw.com/errors/ins-002",
		ExitCode: 1,
	}

	// ErrTerminalTooNarrow is returned when terminal is too narrow for TUI.
	ErrTerminalTooNarrow = &SetupError{
		Code:     "INS-011",
		Title:    "Terminal too narrow",
		Cause:    "The TUI wizard requires at least 60 columns to display properly.",
		Fix: []string{
			"Resize your terminal window to be wider",
			"Or use environment variables for non-interactive setup",
		},
		ExitCode: 1,
	}

	// ErrConfigWriteFailed is returned when configuration cannot be written.
	ErrConfigWriteFailed = &SetupError{
		Code:     "INS-003",
		Title:    "Configuration write failed",
		Cause:    "Unable to write configuration file. This may be due to permission issues or insufficient disk space.",
		Fix: []string{
			"Check volume permissions: ls -la /etc/armorclaw",
			"Verify disk space: df -h",
			"Run with --user root if needed",
		},
		DocLink:  "https://docs.armorclaw.com/errors/ins-003",
		ExitCode: 1,
	}

	// ErrMatrixConnection is returned when Matrix homeserver is unreachable.
	ErrMatrixConnection = &SetupError{
		Code:     "INS-004",
		Title:    "Matrix connection failed",
		Cause:    "Cannot connect to the Matrix homeserver. The Conduit container may not be running or the port is not exposed.",
		Fix: []string{
			"Check if Conduit is running: docker ps | grep conduit",
			"Verify port 6167 is exposed: docker port armorclaw",
			"Check Conduit logs: docker logs armorclaw-conduit",
		},
		DocLink:  "https://docs.armorclaw.com/errors/ins-004",
		ExitCode: 1,
	}

	// ErrAPIKeyInvalid is returned when API key validation fails.
	ErrAPIKeyInvalid = &SetupError{
		Code:     "INS-005",
		Title:    "API key validation failed",
		Cause:    "The provided API key is invalid or has insufficient length.",
		Fix: []string{
			"Verify your API key is correct",
			"Ensure the key is at least 20 characters",
			"Check that the key matches your provider (OpenAI, Anthropic, etc.)",
		},
		ExitCode: 1,
	}

	// ErrProviderSelection is returned when provider selection fails.
	ErrProviderSelection = &SetupError{
		Code:     "INS-015",
		Title:    "Invalid provider selection",
		Cause:    "The selected provider format was not recognized.",
		Fix: []string{
			"Try running setup again",
			"If the issue persists, report this as a bug",
		},
		ExitCode: 1,
	}

	// ErrFormInput is returned when form input fails.
	ErrFormInput = &SetupError{
		Code:     "INS-016",
		Title:    "Form input error",
		Cause:    "The interactive form encountered an input error.",
		Fix: []string{
			"Try running setup again",
			"If using a limited terminal, try: -e ARMORCLAW_ACCESSIBLE=true",
			"Or use environment variables for non-interactive setup",
		},
		ExitCode: 1,
	}

	// ErrNetworkUnavailable is returned when network connectivity fails.
	ErrNetworkUnavailable = &SetupError{
		Code:     "INS-012",
		Title:    "Network unavailable",
		Cause:    "Cannot reach external services. Network connectivity is required for Docker image pulls and API calls.",
		Fix: []string{
			"Check your network connection",
			"Verify DNS resolution: nslookup google.com",
			"Check if firewall is blocking outbound connections",
		},
		ExitCode: 1,
	}

	// ErrDiskSpaceLow is returned when disk space is insufficient.
	ErrDiskSpaceLow = &SetupError{
		Code:     "INS-013",
		Title:    "Insufficient disk space",
		Cause:    "Less than 2GB of free disk space available. ArmorClaw needs space for Docker images and data.",
		Fix: []string{
			"Free up disk space: docker system prune -a",
			"Check available space: df -h",
			"Remove unused Docker volumes: docker volume prune",
		},
		ExitCode: 1,
	}

	// ErrSSLCertFailed is returned when SSL certificate generation fails.
	ErrSSLCertFailed = &SetupError{
		Code:     "INS-014",
		Title:    "SSL certificate generation failed",
		Cause:    "Unable to generate self-signed SSL certificate.",
		Fix: []string{
			"Verify openssl is installed: openssl version",
			"Check /etc/armorclaw/ssl directory permissions",
			"Run with --user root if needed",
		},
		ExitCode: 1,
	}

	// ErrUserAborted is returned when user cancels the setup.
	ErrUserAborted = &SetupError{
		Code:     "INS-099",
		Title:    "Setup cancelled by user",
		Cause:    "The setup wizard was cancelled before completion.",
		Fix: []string{
			"Re-run the setup to continue",
			"Use environment variables for non-interactive setup",
		},
		ExitCode: 130, // Standard exit code for Ctrl+C
	}
)

// WrapError wraps a standard error with a SetupError for context.
func WrapError(err error, setupErr *SetupError) *SetupError {
	return &SetupError{
		Code:     setupErr.Code,
		Title:    setupErr.Title,
		Cause:    fmt.Sprintf("%s\n  Original error: %v", setupErr.Cause, err),
		Fix:      setupErr.Fix,
		DocLink:  setupErr.DocLink,
		ExitCode: setupErr.ExitCode,
	}
}

// IsSetupError checks if an error is a SetupError.
func IsSetupError(err error) bool {
	_, ok := err.(*SetupError)
	return ok
}

// GetSetupError extracts a SetupError from an error, or returns nil.
func GetSetupError(err error) *SetupError {
	if setupErr, ok := err.(*SetupError); ok {
		return setupErr
	}
	return nil
}
