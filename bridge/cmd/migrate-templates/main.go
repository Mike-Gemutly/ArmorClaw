// migrate-templates adds "input": {} to WorkflowStep objects missing it.
//
// Usage:
//
//	go run ./bridge/cmd/migrate-templates <path>
//
// <path> can be a single JSON file or a directory (processed recursively).
// The tool is idempotent: running it multiple times produces the same output.
// Original files are only rewritten if changes were made.
//
// Flags:
//
//	--dry-run   Print changes without writing files
//	--validate  Only validate; do not migrate
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var validStepTypes = map[string]bool{
	"action":         true,
	"condition":      true,
	"parallel":       true,
	"parallel_split": true,
	"parallel_merge": true,
}

func main() {
	dryRun := flag.Bool("dry-run", false, "Print changes without writing files")
	validateOnly := flag.Bool("validate", false, "Only validate templates; do not migrate")
	flag.Parse()

	args := flag.Args()
	if len(args) != 1 {
		fmt.Fprintf(os.Stderr, "Usage: migrate-templates [--dry-run] [--validate] <file-or-directory>\n")
		os.Exit(1)
	}

	root := args[0]
	info, err := os.Stat(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	totalFiles := 0
	totalMigrated := 0
	totalErrors := 0

	if info.IsDir() {
		err = filepath.Walk(root, func(path string, fi os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if fi.IsDir() || !strings.HasSuffix(strings.ToLower(fi.Name()), ".json") {
				return nil
			}
			m, e, v := processFile(path, *dryRun, *validateOnly)
			totalFiles += v
			totalMigrated += m
			totalErrors += e
			return nil
		})
	} else {
		m, e, v := processFile(root, *dryRun, *validateOnly)
		totalFiles += v
		totalMigrated += m
		totalErrors += e
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error walking directory: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\nSummary: %d template(s) scanned, %d migrated, %d error(s)\n", totalFiles, totalMigrated, totalErrors)
	if totalErrors > 0 {
		os.Exit(1)
	}
}

// processFile handles a single JSON file. Returns (migrated, errors, templatesFound).
func processFile(path string, dryRun, validateOnly bool) (int, int, int) {
	raw, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  ERROR reading %s: %v\n", path, err)
		return 0, 1, 0
	}

	if !isTemplateJSON(raw) {
		return 0, 0, 0
	}

	var rawMap map[string]json.RawMessage
	if err := json.Unmarshal(raw, &rawMap); err != nil {
		fmt.Fprintf(os.Stderr, "  ERROR parsing %s: %v\n", path, err)
		return 0, 1, 0
	}

	stepsRaw, ok := rawMap["steps"]
	if !ok {
		fmt.Fprintf(os.Stderr, "  SKIP %s: no \"steps\" field found\n", path)
		return 0, 0, 0
	}

	var steps []json.RawMessage
	if err := json.Unmarshal(stepsRaw, &steps); err != nil {
		fmt.Fprintf(os.Stderr, "  ERROR parsing steps in %s: %v\n", path, err)
		return 0, 1, 0
	}

	changed := false
	errors := 0
	migrated := 0

	for i, stepRaw := range steps {
		var step map[string]json.RawMessage
		if err := json.Unmarshal(stepRaw, &step); err != nil {
			fmt.Fprintf(os.Stderr, "  ERROR parsing step[%d] in %s: %v\n", i, path, err)
			errors++
			continue
		}

		if typeRaw, ok := step["type"]; ok {
			var stepType string
			if err := json.Unmarshal(typeRaw, &stepType); err == nil {
				if !validStepTypes[stepType] {
					fmt.Fprintf(os.Stderr, "  WARN step[%d] in %s: unknown type %q\n", i, path, stepType)
				}
			}
		}

		if _, hasInput := step["input"]; !hasInput {
			if !validateOnly {
				step["input"] = []byte("{}")
				changed = true
				migrated++
				fmt.Printf("  ADD input:{} to step[%d] (%s) in %s\n", i, stepName(step), path)
			}
		}

		updatedStep, err := json.Marshal(step)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  ERROR re-marshaling step[%d] in %s: %v\n", i, path, err)
			errors++
			continue
		}
		steps[i] = updatedStep
	}

	if validateOnly {
		fmt.Printf("  VALID %s (%d steps)\n", path, len(steps))
		return 0, errors, 1
	}

	if changed {
		updatedSteps, err := json.Marshal(steps)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  ERROR marshaling steps in %s: %v\n", path, err)
			return 0, 1, 1
		}
		rawMap["steps"] = updatedSteps

		output, err := json.MarshalIndent(rawMap, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "  ERROR formatting %s: %v\n", path, err)
			return 0, 1, 1
		}
		output = append(output, '\n')

		if dryRun {
			fmt.Printf("  DRY-RUN would write %s (%d step(s) migrated)\n", path, migrated)
		} else {
			if err := os.WriteFile(path, output, 0); err != nil {
				fmt.Fprintf(os.Stderr, "  ERROR writing %s: %v\n", path, err)
				return 0, 1, 1
			}
			fmt.Printf("  WROTE %s (%d step(s) migrated)\n", path, migrated)
		}
		return migrated, errors, 1
	}

	fmt.Printf("  OK %s (already up-to-date, %d steps)\n", path, len(steps))
	return 0, errors, 1
}

// isTemplateJSON heuristically detects a template JSON by checking for
// the "steps" and "id" keys at the top level.
func isTemplateJSON(raw []byte) bool {
	var probe map[string]json.RawMessage
	if err := json.Unmarshal(raw, &probe); err != nil {
		return false
	}
	_, hasSteps := probe["steps"]
	_, hasID := probe["id"]
	return hasSteps && hasID
}

// stepName extracts the "name" field from a raw step map for logging.
func stepName(step map[string]json.RawMessage) string {
	if nameRaw, ok := step["name"]; ok {
		var name string
		if err := json.Unmarshal(nameRaw, &name); err == nil {
			return name
		}
	}
	if idRaw, ok := step["step_id"]; ok {
		var id string
		if err := json.Unmarshal(idRaw, &id); err == nil {
			return id
		}
	}
	return "?"
}
