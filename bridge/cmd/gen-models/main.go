// Package main provides model documentation generation.
package main

import (
	"fmt"
	"os"
)

func main() {
	opts := Options{
		OutputPath: "MODELS.md",
		CatwalkURL: os.Getenv("CATWALK_URL"),
	}

	gen := NewGenerator(opts)
	markdown, err := gen.Generate()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if opts.OutputPath != "" {
		if err := os.WriteFile(opts.OutputPath, []byte(markdown), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing file: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Generated %s\n", opts.OutputPath)
	} else {
		fmt.Println(markdown)
	}
}
