package skills

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Skill represents a single skill from a SKILL.md file
type Skill struct {
	Name         string                           `json:"name"`
	Description  string                           `json:"description"`
	Homepage     string                           `json:"homepage"`
	Domain       string                           `json:"domain"`
	Risk         string                           `json:"risk"`
	Command      string                           `json:"command"`
	Enabled      bool                             `json:"enabled"`
	Timeout      time.Duration                    `json:"timeout"`
	Version      string                           `json:"version"`
	Parameters   map[string]Param                 `json:"parameters"`
	Metadata     map[string]interface{}           `json:"metadata"`
	SecurityCheck func(params map[string]interface{}) error `json:"-"`
}

// Param represents a parameter for a skill
type Param struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
	Validation  string `json:"validation,omitempty"`
}

// Registry manages all available skills
type Registry struct {
	skills      map[string]*Skill
	domainIndex map[string][]*Skill
	mutex       sync.RWMutex
}

// NewRegistry creates a new skill registry
func NewRegistry() *Registry {
	return &Registry{
		skills:      make(map[string]*Skill),
		domainIndex: make(map[string][]*Skill),
	}
}

// ScanSkills scans the skills directory for SKILL.md files
func (r *Registry) ScanSkills(skillsDir string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Clear existing data
	r.skills = make(map[string]*Skill)
	r.domainIndex = make(map[string][]*Skill)

	return filepath.WalkDir(skillsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.Name() == "SKILL.md" {
			skill, err := parseSkillFile(path)
			if err != nil {
				fmt.Printf("Warning: failed to parse skill file %s: %v\n", path, err)
				return nil // Continue scanning other files
			}

			r.skills[skill.Name] = skill

			// Add to domain index
			r.domainIndex[skill.Domain] = append(r.domainIndex[skill.Domain], skill)
		}

		return nil
	})
}

// parseSkillFile parses a single SKILL.md file
func parseSkillFile(path string) (*Skill, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	content := string(data)
	skill := &Skill{
		Enabled: true, // Skills are enabled by default
	}

	// Parse frontmatter
	if !strings.HasPrefix(content, "---") {
		return nil, fmt.Errorf("invalid SKILL.md format: no frontmatter")
	}

	parts := strings.SplitN(content, "---", 3)
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid SKILL.md format: missing closing ---")
	}

	frontmatter := strings.TrimSpace(parts[1])
	body := strings.TrimSpace(parts[2])

	// Parse YAML frontmatter (simplified for Phase 1)
	// In production, use a proper YAML parser like gopkg.in/yaml.v2
	var frontmatterMap map[string]interface{}
	if err := parseYAMLFrontmatter(frontmatter, &frontmatterMap); err != nil {
		return nil, fmt.Errorf("parse frontmatter: %w", err)
	}

	// Extract basic fields
	if name, ok := frontmatterMap["name"].(string); ok {
		skill.Name = name
	} else {
		return nil, fmt.Errorf("skill missing name")
	}

	if desc, ok := frontmatterMap["description"].(string); ok {
		skill.Description = desc
	}

	if homepage, ok := frontmatterMap["homepage"].(string); ok {
		skill.Homepage = homepage
	}

	// Extract domain from skill name or metadata
	skill.Domain = extractDomainFromName(skill.Name)

	// Set risk level based on domain
	skill.Risk = getRiskForDomain(skill.Domain)

	// Extract command
	skill.Command = getCommandForSkill(skill.Name)

	// Extract parameters (simplified for now)
	skill.Parameters = make(map[string]Param)

	// Extract metadata
	if metadata, ok := frontmatterMap["metadata"].(map[interface{}]interface{}); ok {
		skill.Metadata = convertInterfaceMap(metadata)
	}

	// Extract parameters from body (simplified)
	skill.Parameters = extractParametersFromBody(body)

	return skill, nil
}

// extractDomainFromName extracts domain from skill name
func extractDomainFromName(name string) string {
	parts := strings.Split(name, ".")
	if len(parts) > 0 {
		switch parts[0] {
		case "github":
			return "github"
		case "weather":
			return "weather"
		case "web":
			return "web"
		default:
			return "general"
		}
	}
	return "general"
}

// getRiskForDomain returns risk level for domain
func getRiskForDomain(domain string) string {
	switch domain {
	case "weather":
		return "low"
	case "github":
		return "medium"
	case "web":
		return "medium"
	default:
		return "low"
	}
}

// getCommandForSkill returns base command for skill
func getCommandForSkill(name string) string {
	parts := strings.Split(name, ".")
	if len(parts) > 0 {
		switch parts[0] {
		case "weather":
			return "curl"
		case "github":
			return "gh"
		case "web":
			return "curl"
		}
	}
	return "curl" // default
}

// extractParametersFromBody extracts parameters from skill body (simplified)
func extractParametersFromBody(body string) map[string]Param {
	params := make(map[string]Param)

	// For Phase 1, we'll define minimal parameters based on skill
	// In a full implementation, this would parse the body more carefully

	return params
}

// GetByDomain returns all skills for a given domain
func (r *Registry) GetByDomain(domain string) []*Skill {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	skills := r.domainIndex[domain]
	result := make([]*Skill, len(skills))
	copy(result, skills)
	return result
}

// GetEnabled returns all enabled skills
func (r *Registry) GetEnabled() []*Skill {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var result []*Skill
	for _, skill := range r.skills {
		if skill.Enabled {
			result = append(result, skill)
		}
	}
	return result
}

// GetSkill returns a specific skill by name
func (r *Registry) GetSkill(name string) (*Skill, bool) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	skill, exists := r.skills[name]
	return skill, exists
}

// ValidateParameters validates parameters against a skill's schema
func (r *Registry) ValidateParameters(skillName string, params map[string]interface{}) error {
	skill, exists := r.GetSkill(skillName)
	if !exists {
		return fmt.Errorf("skill not found: %s", skillName)
	}

	// Check required parameters
	for paramName, param := range skill.Parameters {
		if param.Required {
			if _, exists := params[paramName]; !exists {
				return fmt.Errorf("required parameter '%s' is missing", paramName)
			}
		}
	}

	// For Phase 1, simple type validation
	for paramName, paramValue := range params {
		if paramDef, exists := skill.Parameters[paramName]; exists {
			if err := validateParameterType(paramName, paramValue, paramDef.Type); err != nil {
				return err
			}
		}
	}

	return nil
}

// validateParameterType validates a parameter's type
func validateParameterType(name string, value interface{}, expectedType string) error {
	switch expectedType {
	case "string":
		if _, ok := value.(string); !ok {
			return fmt.Errorf("parameter '%s' must be a string", name)
		}
	case "number":
		if _, ok := value.(float64); !ok {
			return fmt.Errorf("parameter '%s' must be a number", name)
		}
	case "boolean":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("parameter '%s' must be a boolean", name)
		}
	case "array":
		if _, ok := value.([]interface{}); !ok {
			return fmt.Errorf("parameter '%s' must be an array", name)
		}
	case "object":
		if _, ok := value.(map[string]interface{}); !ok {
			return fmt.Errorf("parameter '%s' must be an object", name)
		}
	}

	return nil
}

// SetEnabled enables or disables a skill
func (r *Registry) SetEnabled(name string, enabled bool) bool {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	skill, exists := r.skills[name]
	if !exists {
		return false
	}

	skill.Enabled = enabled
	return true
}

// convertInterfaceMap converts map[interface{}]interface{} to map[string]interface{}
func convertInterfaceMap(m map[interface{}]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range m {
		if key, ok := k.(string); ok {
			result[key] = v
		}
	}
	return result
}

// parseYAMLFrontmatter parses simple YAML frontmatter for Phase 1
// In production, use a proper YAML parser like gopkg.in/yaml.v2
func parseYAMLFrontmatter(data string, v interface{}) error {
	// Simplified YAML parsing for Phase 1
	// Parse basic key: value pairs
	lines := strings.Split(data, "\n")
	
	frontmatterMap, ok := v.(*map[string]interface{})
	if !ok {
		return fmt.Errorf("expected map[string]interface{}, got %T", v)
	}
	
	*frontmatterMap = make(map[string]interface{})
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		
		// Parse key: value pairs
		if strings.Contains(line, ":") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				
				// Remove quotes if present
				if strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"") {
					value = strings.Trim(value, "\"")
				} else if strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'") {
					value = strings.Trim(value, "'")
				}
				
				// Handle special cases like metadata
				if key == "metadata" && strings.HasPrefix(value, "{") && strings.HasSuffix(value, "}") {
					// For now, just store as string - in Phase 2 we'd parse JSON
					(*frontmatterMap)[key] = value
				} else {
					(*frontmatterMap)[key] = value
				}
			}
		}
	}
	
	return nil
}