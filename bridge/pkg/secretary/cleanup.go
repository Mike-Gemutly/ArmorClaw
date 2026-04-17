package secretary

import (
	"log/slog"
	"os"
	"path/filepath"
)

// cleanupStateDir removes the entire state directory and all its contents.
// Empty string or nonexistent path are no-ops (return nil).
func cleanupStateDir(stateDir string) error {
	if stateDir == "" {
		return nil
	}

	if err := os.RemoveAll(stateDir); err != nil {
		slog.Error("failed to cleanup state directory", "path", stateDir, "error", err)
		return err
	}

	return nil
}

// stateDirExists checks whether a state directory exists by looking for
// the _events.jsonl marker file inside it.
func stateDirExists(stateDir string) bool {
	if stateDir == "" {
		return false
	}
	_, err := os.Stat(filepath.Join(stateDir, "_events.jsonl"))
	return err == nil
}
