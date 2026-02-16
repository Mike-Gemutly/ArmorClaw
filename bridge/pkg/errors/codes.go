package errors

import "sync"

// ErrorCodeDefinition defines an error code's properties
type ErrorCodeDefinition struct {
	Code     string   `json:"code"`
	Category string   `json:"category"`
	Severity Severity `json:"severity"`
	Message  string   `json:"message"`
	Help     string   `json:"help"`
}

// registry stores all registered error codes
var (
	registry = make(map[string]ErrorCodeDefinition)
	registryMu sync.RWMutex
)

// Default error code definitions
var defaultCodes = map[string]ErrorCodeDefinition{
	// Container errors (CTX-001 to CTX-099: lifecycle)
	"CTX-001": {
		Code:     "CTX-001",
		Category: "container",
		Severity: SeverityError,
		Message:  "container start failed",
		Help:     "Check Docker daemon status, image availability, and resource limits",
	},
	"CTX-002": {
		Code:     "CTX-002",
		Category: "container",
		Severity: SeverityError,
		Message:  "container exec failed",
		Help:     "Verify container is running and command is valid",
	},
	"CTX-003": {
		Code:     "CTX-003",
		Category: "container",
		Severity: SeverityCritical,
		Message:  "container health check timeout",
		Help:     "Container may be hung; check logs and consider restart",
	},
	"CTX-010": {
		Code:     "CTX-010",
		Category: "container",
		Severity: SeverityCritical,
		Message:  "permission denied on docker socket",
		Help:     "Bridge needs docker group membership or sudo",
	},
	"CTX-011": {
		Code:     "CTX-011",
		Category: "container",
		Severity: SeverityError,
		Message:  "container not found",
		Help:     "Container may have been removed or ID is incorrect",
	},
	"CTX-012": {
		Code:     "CTX-012",
		Category: "container",
		Severity: SeverityError,
		Message:  "container already running",
		Help:     "Stop the container first or use a different ID",
	},
	"CTX-020": {
		Code:     "CTX-020",
		Category: "container",
		Severity: SeverityError,
		Message:  "image pull failed",
		Help:     "Check image name, registry access, and network connectivity",
	},
	"CTX-021": {
		Code:     "CTX-021",
		Category: "container",
		Severity: SeverityError,
		Message:  "image not found",
		Help:     "Verify image exists in registry and name is correct",
	},

	// Matrix errors (MAT-001 to MAT-099: connection)
	"MAT-001": {
		Code:     "MAT-001",
		Category: "matrix",
		Severity: SeverityError,
		Message:  "matrix connection failed",
		Help:     "Check homeserver URL and network connectivity",
	},
	"MAT-002": {
		Code:     "MAT-002",
		Category: "matrix",
		Severity: SeverityError,
		Message:  "matrix authentication failed",
		Help:     "Verify access token or device credentials",
	},
	"MAT-003": {
		Code:     "MAT-003",
		Category: "matrix",
		Severity: SeverityWarning,
		Message:  "matrix sync timeout",
		Help:     "Homeserver may be slow; will retry automatically",
	},
	"MAT-010": {
		Code:     "MAT-010",
		Category: "matrix",
		Severity: SeverityError,
		Message:  "E2EE decryption failed",
		Help:     "Device keys may be missing or rotated",
	},
	"MAT-011": {
		Code:     "MAT-011",
		Category: "matrix",
		Severity: SeverityError,
		Message:  "E2EE encryption failed",
		Help:     "Device keys may be missing; try re-verifying",
	},
	"MAT-020": {
		Code:     "MAT-020",
		Category: "matrix",
		Severity: SeverityError,
		Message:  "room join failed",
		Help:     "Check room ID and user permissions",
	},
	"MAT-021": {
		Code:     "MAT-021",
		Category: "matrix",
		Severity: SeverityError,
		Message:  "message send failed",
		Help:     "Check room membership and message content",
	},
	"MAT-030": {
		Code:     "MAT-030",
		Category: "matrix",
		Severity: SeverityCritical,
		Message:  "voice call failed",
		Help:     "Check TURN server configuration and network",
	},

	// RPC errors (RPC-001 to RPC-099: protocol)
	"RPC-001": {
		Code:     "RPC-001",
		Category: "rpc",
		Severity: SeverityWarning,
		Message:  "invalid JSON-RPC request",
		Help:     "Check request format matches JSON-RPC 2.0 spec",
	},
	"RPC-002": {
		Code:     "RPC-002",
		Category: "rpc",
		Severity: SeverityError,
		Message:  "method not found",
		Help:     "Verify method name against RPC API docs",
	},
	"RPC-003": {
		Code:     "RPC-003",
		Category: "rpc",
		Severity: SeverityError,
		Message:  "invalid params",
		Help:     "Check parameter types and required fields",
	},
	"RPC-010": {
		Code:     "RPC-010",
		Category: "rpc",
		Severity: SeverityError,
		Message:  "socket connection failed",
		Help:     "Check bridge is running and socket permissions",
	},
	"RPC-011": {
		Code:     "RPC-011",
		Category: "rpc",
		Severity: SeverityWarning,
		Message:  "request timeout",
		Help:     "Operation took too long; may need retry",
	},
	"RPC-020": {
		Code:     "RPC-020",
		Category: "rpc",
		Severity: SeverityError,
		Message:  "unauthorized",
		Help:     "Check authentication credentials and permissions",
	},

	// System errors (SYS-001+)
	"SYS-001": {
		Code:     "SYS-001",
		Category: "system",
		Severity: SeverityCritical,
		Message:  "keystore decryption failed",
		Help:     "Master key may be wrong or keystore corrupted",
	},
	"SYS-002": {
		Code:     "SYS-002",
		Category: "system",
		Severity: SeverityError,
		Message:  "audit log write failed",
		Help:     "Check disk space and permissions on /var/lib/armorclaw",
	},
	"SYS-003": {
		Code:     "SYS-003",
		Category: "system",
		Severity: SeverityError,
		Message:  "configuration load failed",
		Help:     "Check config file syntax and file permissions",
	},
	"SYS-010": {
		Code:     "SYS-010",
		Category: "system",
		Severity: SeverityCritical,
		Message:  "secret injection failed",
		Help:     "Check secrets file format and permissions",
	},
	"SYS-011": {
		Code:     "SYS-011",
		Category: "system",
		Severity: SeverityError,
		Message:  "secret cleanup failed",
		Help:     "Secrets may persist; manual cleanup may be needed",
	},
	"SYS-020": {
		Code:     "SYS-020",
		Category: "system",
		Severity: SeverityCritical,
		Message:  "out of memory",
		Help:     "Increase system memory or reduce concurrent operations",
	},
	"SYS-021": {
		Code:     "SYS-021",
		Category: "system",
		Severity: SeverityCritical,
		Message:  "disk full",
		Help:     "Free up disk space or increase storage",
	},

	// Budget errors (BGT-001+)
	"BGT-001": {
		Code:     "BGT-001",
		Category: "budget",
		Severity: SeverityWarning,
		Message:  "budget warning threshold reached",
		Help:     "Monitor usage; consider adjusting limits",
	},
	"BGT-002": {
		Code:     "BGT-002",
		Category: "budget",
		Severity: SeverityCritical,
		Message:  "budget exceeded",
		Help:     "Operation blocked; increase budget or wait for reset",
	},

	// Voice/WebRTC errors (VOX-001+)
	"VOX-001": {
		Code:     "VOX-001",
		Category: "voice",
		Severity: SeverityError,
		Message:  "WebRTC connection failed",
		Help:     "Check ICE/TURN configuration and network connectivity",
	},
	"VOX-002": {
		Code:     "VOX-002",
		Category: "voice",
		Severity: SeverityError,
		Message:  "audio capture failed",
		Help:     "Check microphone permissions and device availability",
	},
	"VOX-003": {
		Code:     "VOX-003",
		Category: "voice",
		Severity: SeverityError,
		Message:  "audio playback failed",
		Help:     "Check speaker configuration and permissions",
	},
}

func init() {
	// Register default codes
	for code, def := range defaultCodes {
		registry[code] = def
	}
}

// Register adds a new error code to the registry
func Register(def ErrorCodeDefinition) {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry[def.Code] = def
}

// Lookup retrieves an error code definition
func Lookup(code string) ErrorCodeDefinition {
	registryMu.RLock()
	defer registryMu.RUnlock()

	if def, ok := registry[code]; ok {
		return def
	}

	// Return unknown code definition
	return ErrorCodeDefinition{
		Code:     code,
		Category: "unknown",
		Severity: SeverityError,
		Message:  "unknown error",
		Help:     "No additional help available for this error code",
	}
}

// AllCodes returns all registered error codes
func AllCodes() map[string]ErrorCodeDefinition {
	registryMu.RLock()
	defer registryMu.RUnlock()

	result := make(map[string]ErrorCodeDefinition, len(registry))
	for k, v := range registry {
		result[k] = v
	}
	return result
}

// CodesByCategory returns all codes in a given category
func CodesByCategory(category string) []ErrorCodeDefinition {
	registryMu.RLock()
	defer registryMu.RUnlock()

	var result []ErrorCodeDefinition
	for _, def := range registry {
		if def.Category == category {
			result = append(result, def)
		}
	}
	return result
}

// CodesBySeverity returns all codes with a given severity
func CodesBySeverity(severity Severity) []ErrorCodeDefinition {
	registryMu.RLock()
	defer registryMu.RUnlock()

	var result []ErrorCodeDefinition
	for _, def := range registry {
		if def.Severity == severity {
			result = append(result, def)
		}
	}
	return result
}
