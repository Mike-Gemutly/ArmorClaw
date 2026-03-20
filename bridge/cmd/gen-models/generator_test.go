package main

import (
	"strings"
	"testing"
)

func TestGenerator_LoadProviders(t *testing.T) {
	gen := NewGenerator(Options{})
	docs := gen.loadProviders()

	if len(docs) == 0 {
		t.Error("expected at least one provider")
	}

	// Check OpenAI exists
	found := false
	for _, d := range docs {
		if d.ID == "openai" {
			found = true
			if len(d.Models) == 0 {
				t.Error("openai should have models")
			}
			break
		}
	}
	if !found {
		t.Error("openai provider not found")
	}
}

func TestGenerator_RenderMarkdown(t *testing.T) {
	gen := NewGenerator(Options{})
	docs := []*ProviderDoc{
		{
			ID:       "test",
			Name:     "Test Provider",
			Protocol: "openai",
			Aliases:  []string{"t", "testing"},
			Models: []ModelDoc{
				{ID: "model-1", Name: "Model 1", ContextSize: 1000},
			},
		},
	}

	md, err := gen.renderMarkdown(docs)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}

	if !strings.Contains(md, "# AI Provider Models") {
		t.Error("missing header")
	}
	if !strings.Contains(md, "Test Provider") {
		t.Error("missing provider name")
	}
	if !strings.Contains(md, "model-1") {
		t.Error("missing model ID")
	}
	if !strings.Contains(md, "t, testing") {
		t.Error("missing aliases")
	}
}

func TestGenerator_CatwalkEnrichment_SkippedWhenUnavailable(t *testing.T) {
	gen := NewGenerator(Options{
		CatwalkURL: "http://localhost:9999", // Unreachable
	})

	docs := gen.loadProviders()
	if len(docs) == 0 {
		t.Fatal("need at least one provider for test")
	}
	originalCount := len(docs[0].Models)

	gen.enrichFromCatwalk(docs)

	// Should not change models when Catwalk unavailable
	if len(docs[0].Models) != originalCount {
		t.Error("models changed despite Catwalk being unavailable")
	}
}
