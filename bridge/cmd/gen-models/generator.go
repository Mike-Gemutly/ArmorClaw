// Package main provides model documentation generation.
package main

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
	"time"

	"github.com/armorclaw/bridge/internal/wizard"
	"github.com/armorclaw/bridge/pkg/providers"
)

// Options configures the generator.
type Options struct {
	OutputPath string // Where to write the markdown file
	CatwalkURL string // Optional Catwalk URL for enrichment
}

// ProviderDoc represents a provider with its models for documentation.
type ProviderDoc struct {
	ID       string
	Name     string
	Protocol string
	Aliases  []string
	Models   []ModelDoc
}

// ModelDoc represents a model for documentation.
type ModelDoc struct {
	ID          string
	Name        string
	ContextSize int
}

// Generator creates model documentation.
type Generator struct {
	opts Options
}

// NewGenerator creates a new generator.
func NewGenerator(opts Options) *Generator {
	return &Generator{opts: opts}
}

// Generate creates the markdown documentation.
func (g *Generator) Generate() (string, error) {
	docs := g.loadProviders()

	// Optional Catwalk enrichment
	if g.opts.CatwalkURL != "" {
		g.enrichFromCatwalk(docs)
	}

	return g.renderMarkdown(docs)
}

func (g *Generator) loadProviders() []*ProviderDoc {
	registry := providers.LoadEmbeddedRegistry()
	docs := make([]*ProviderDoc, 0, registry.GetProviderCount())

	for _, p := range registry.Providers {
		doc := &ProviderDoc{
			ID:       p.ID,
			Name:     p.Name,
			Protocol: p.Protocol,
			Aliases:  p.Aliases,
			Models:   make([]ModelDoc, 0),
		}

		// Load fallback models
		if models := providers.GetModels(p.ID); len(models) > 0 {
			for _, m := range models {
				doc.Models = append(doc.Models, ModelDoc{
					ID:          m.ID,
					Name:        m.Name,
					ContextSize: m.ContextSize,
				})
			}
		}

		docs = append(docs, doc)
	}

	return docs
}

func (g *Generator) enrichFromCatwalk(docs []*ProviderDoc) {
	client := wizard.NewCatwalkClient(g.opts.CatwalkURL)
	if !client.IsAvailable() {
		return
	}

	catwalkProviders, err := client.FetchProviders()
	if err != nil {
		return
	}

	// Build lookup map
	docMap := make(map[string]*ProviderDoc)
	for _, d := range docs {
		docMap[d.ID] = d
	}

	// Enrich with Catwalk data
	for _, cp := range catwalkProviders {
		if doc, ok := docMap[cp.ID]; ok && len(cp.Models) > 0 {
			// Replace models with Catwalk data (richer)
			doc.Models = make([]ModelDoc, 0, len(cp.Models))
			for _, m := range cp.Models {
				doc.Models = append(doc.Models, ModelDoc{
					ID:          m.ID,
					Name:        m.Name,
					ContextSize: m.ContextSize,
				})
			}
		}
	}
}

func (g *Generator) renderMarkdown(docs []*ProviderDoc) (string, error) {
	tmpl := `# AI Provider Models

> Generated from embedded provider registry.
> Last updated: {{.Timestamp}}

## Providers

| ID | Name | Protocol | Aliases |
|----|------|----------|---------|
{{- range .Providers}}
| {{.ID}} | {{.Name}} | {{.Protocol}} | {{aliasList .Aliases}} |
{{- end}}

## Models by Provider

{{- range .Providers}}
{{- if .Models}}

### {{.Name}}{{if .Aliases}} ({{join .Aliases ", "}}){{end}}

| Model ID | Display Name | Context |
|----------|--------------|---------|
{{- range .Models}}
| {{.ID}} | {{.Name}} | {{contextSize .ContextSize}} |
{{- end}}
{{- end}}
{{- end}}
`

	funcs := template.FuncMap{
		"aliasList": func(aliases []string) string {
			if len(aliases) == 0 {
				return "-"
			}
			return strings.Join(aliases, ", ")
		},
		"join": strings.Join,
		"contextSize": func(size int) string {
			if size == 0 {
				return "-"
			}
			return fmt.Sprintf("%d", size)
		},
	}

	t, err := template.New("models").Funcs(funcs).Parse(tmpl)
	if err != nil {
		return "", fmt.Errorf("parse template: %w", err)
	}

	data := struct {
		Timestamp string
		Providers []*ProviderDoc
	}{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Providers: docs,
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}

	return buf.String(), nil
}
