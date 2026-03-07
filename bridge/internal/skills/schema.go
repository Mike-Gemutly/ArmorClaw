package skills

import (
	"encoding/json"
	"sync"
)

// ToolSchema represents an OpenAI function schema
type ToolSchema struct {
	Type       string                 `json:"type"`
	Function   FunctionSchema        `json:"function"`
}

// FunctionSchema represents an OpenAI function definition
type FunctionSchema struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  ParametersSchema       `json:"parameters"`
}

// ParametersSchema represents function parameters
type ParametersSchema struct {
	Type       string                 `json:"type"`
	Properties map[string]interface{} `json:"properties"`
	Required   []string               `json:"required"`
}

// SchemaGenerator generates OpenAI tool schemas from skills
type SchemaGenerator struct {
	registry *Registry
	cache    map[string][]ToolSchema
	mutex    sync.RWMutex
}

// NewSchemaGenerator creates a new schema generator
func NewSchemaGenerator(registry *Registry) *SchemaGenerator {
	return &SchemaGenerator{
		registry: registry,
		cache:    make(map[string][]ToolSchema),
	}
}

// GenerateSchema generates an OpenAI schema for a skill
func (sg *SchemaGenerator) GenerateSchema(skill *Skill) (*ToolSchema, error) {
	return sg.generateOpenAISchema(skill)
}

// generateOpenAISchema generates an OpenAI-compatible schema
func (sg *SchemaGenerator) generateOpenAISchema(skill *Skill) (*ToolSchema, error) {
	schema := &ToolSchema{
		Type: "function",
		Function: FunctionSchema{
			Name:        skill.Name,
			Description: skill.Description,
			Parameters: ParametersSchema{
				Type:       "object",
				Properties: make(map[string]interface{}),
				Required:   []string{},
			},
		},
	}

	// Add parameters based on skill
	switch skill.Name {
	case "weather.get":
		schema.Function.Parameters.Properties["location"] = map[string]interface{}{
			"type":        "string",
			"description": "Location to get weather for (e.g., 'London', 'New York', or '52.52,13.41')",
		}
		schema.Function.Parameters.Required = append(schema.Function.Parameters.Required, "location")

	case "github.repo.info":
		schema.Function.Parameters.Properties["repo"] = map[string]interface{}{
			"type":        "string",
			"description": "Repository in owner/repo format (e.g., 'octocat/hello-world')",
		}
		schema.Function.Parameters.Required = append(schema.Function.Parameters.Required, "repo")

	case "github.issue.create":
		schema.Function.Parameters.Properties["repo"] = map[string]interface{}{
			"type":        "string",
			"description": "Repository in owner/repo format (e.g., 'octocat/hello-world')",
		}
		schema.Function.Parameters.Properties["title"] = map[string]interface{}{
			"type":        "string",
			"description": "Title of the issue",
		}
		schema.Function.Parameters.Properties["body"] = map[string]interface{}{
			"type":        "string",
			"description": "Detailed description of the issue (optional)",
		}
		schema.Function.Parameters.Required = append(schema.Function.Parameters.Required, "repo", "title")

	case "web.fetch":
		schema.Function.Parameters.Properties["url"] = map[string]interface{}{
			"type":        "string",
			"description": "URL to fetch (must be HTTPS)",
		}
		schema.Function.Parameters.Required = append(schema.Function.Parameters.Required, "url")
	}

	return schema, nil
}

// GetSchemaForDomain returns schemas for all enabled skills in a domain
func (sg *SchemaGenerator) GetSchemaForDomain(domain string) ([]ToolSchema, error) {
	sg.mutex.RLock()
	cached, exists := sg.cache[domain]
	sg.mutex.RUnlock()

	if exists {
		return cached, nil
	}

	// Generate schemas
	skills := sg.registry.GetByDomain(domain)
	var schemas []ToolSchema

	for _, skill := range skills {
		if !skill.Enabled {
			continue
		}

		schema, err := sg.GenerateSchema(skill)
		if err != nil {
			continue // Skip skills that can't be schema-ified
		}

		schemas = append(schemas, *schema)
	}

	// Cache the result
	sg.mutex.Lock()
	sg.cache[domain] = schemas
	sg.mutex.Unlock()

	return schemas, nil
}

// GetSchemaForAllDomains returns schemas for all enabled skills organized by domain
func (sg *SchemaGenerator) GetSchemaForAllDomains() (map[string][]ToolSchema, error) {
	domains := map[string][]ToolSchema{
		"github":  nil,
		"weather": nil,
		"web":     nil,
		"general": nil,
	}

	for domain := range domains {
		schemas, err := sg.GetSchemaForDomain(domain)
		if err != nil {
			continue
		}
		domains[domain] = schemas
	}

	return domains, nil
}

// ClearCache clears the schema cache
func (sg *SchemaGenerator) ClearCache() {
	sg.mutex.Lock()
	defer sg.mutex.Unlock()
	sg.cache = make(map[string][]ToolSchema)
}

// ToJSON returns the schemas as JSON
func (sg *SchemaGenerator) ToJSON() ([]byte, error) {
	domains, err := sg.GetSchemaForAllDomains()
	if err != nil {
		return nil, err
	}

	return json.MarshalIndent(domains, "", "  ")
}

// GetDomainList returns all available domains
func (sg *SchemaGenerator) GetDomainList() []string {
	skills := sg.registry.GetEnabled()
	domains := make(map[string]bool)

	for _, skill := range skills {
		if skill.Enabled {
			domains[skill.Domain] = true
		}
	}

	var result []string
	for domain := range domains {
		result = append(result, domain)
	}

	return result
}

// RefreshCache refreshes the schema cache for a domain
func (sg *SchemaGenerator) RefreshCache(domain string) error {
	sg.mutex.Lock()
	defer sg.mutex.Unlock()

	// Remove from cache
	delete(sg.cache, domain)

	// Regenerate
	_, err := sg.GetSchemaForDomain(domain)
	if err != nil {
		return err
	}

	return nil
}

// GetSkillNamesForDomain returns all enabled skill names for a domain
func (sg *SchemaGenerator) GetSkillNamesForDomain(domain string) []string {
	skills := sg.registry.GetByDomain(domain)
	var names []string

	for _, skill := range skills {
		if skill.Enabled {
			names = append(names, skill.Name)
		}
	}

	return names
}