package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	outputPath := flag.String("output", "", "Output file path (default: docs/reference/models.md relative to repo root)")
	catwalkURL := flag.String("catwalk-url", "", "Optional Catwalk URL for enrichment (default: none)")
	flag.Parse()

	out := *outputPath
	if out == "" {
		repoRoot, err := findRepoRoot()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		out = filepath.Join(repoRoot, "docs", "reference", "models.md")
	}

	opts := Options{
		OutputPath: out,
		CatwalkURL: *catwalkURL,
	}

	gen := NewGenerator(opts)
	md, err := gen.Generate()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating docs: %v\n", err)
		os.Exit(1)
	}

	// Ensure directory exists
	dir := filepath.Dir(out)
	if err := os.MkdirAll(dir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating directory: %v\n", err)
		os.Exit(1)
	}

	// Write file
	if err := os.WriteFile(out, []byte(md), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Generated: %s\n", out)
}

func findRepoRoot() (string, error) {
	// Start from current directory and go up looking for go.mod
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			// Found go.mod, go up one more level to repo root
			return filepath.Dir(dir), nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("could not find go.mod")
		}
		dir = parent
	}
}
