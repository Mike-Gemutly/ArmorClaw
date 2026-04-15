// Package secretary provides core types for Secretary features including templates, workflows, and approvals.
package secretary

import (
	"encoding/json"
	"fmt"
	"os"
)

//=============================================================================
// Container Step Result (result.json convention)
//=============================================================================

// ContainerStepResult represents the structured output a container writes to
// result.json inside its bind-mounted state directory.
//
// Convention:
//
//	The container runs with NetworkMode "none" — its only output channels are
//	the process exit code and the bind-mounted state directory. A container
//	may (optionally) write a result.json file to /home/claw/.openclaw/result.json
//	inside the container, which maps to
//	/var/lib/armorclaw/agent-state/{definitionID}/result.json on the host.
//
//	The Bridge reads this file via ParseContainerStepResult, passing the
//	host-side state directory path. If the file does not exist the container
//	simply chose not to produce results — this is not an error.
//
//	Forward compatibility: unknown JSON fields are silently ignored so that
//	new container versions can add fields without breaking older Bridge code.
//
// Permission model:
//
//	The state directory is chown'd to UID 10001 (the container user) by
//	factory.go's Spawn() method. The container writes as 10001, Bridge reads
//	as root. The bind mount is synchronous (local filesystem), so there is no
//	race between the container writing the file and Bridge reading it — Docker
//	reports container exit only after all process file descriptors are closed.
type ContainerStepResult struct {
	// Status is the step outcome: "success", "failed", "partial", etc.
	Status string `json:"status"`

	// Output is human-readable output or log summary from the container step.
	Output string `json:"output"`

	// Data holds arbitrary structured data returned by the container step.
	Data map[string]any `json:"data,omitempty"`

	// Error describes the error if the step failed.
	Error string `json:"error,omitempty"`

	// DurationMS is the step execution duration in milliseconds.
	DurationMS int64 `json:"duration_ms"`
}

// ParseContainerStepResult reads and parses result.json from the given
// host-side state directory. The stateDir path is the bind-mount source
// (e.g., /var/lib/armorclaw/agent-state/{definitionID}).
//
// Returns (nil, nil) if result.json does not exist — the container chose
// not to produce results. Returns (nil, error) for I/O or JSON errors.
func ParseContainerStepResult(stateDir string) (*ContainerStepResult, error) {
	path := stateDir
	if len(path) > 0 && path[len(path)-1] != '/' {
		path += "/"
	}
	path += "result.json"

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read result.json: %w", err)
	}

	var result ContainerStepResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("parse result.json: %w", err)
	}

	return &result, nil
}
