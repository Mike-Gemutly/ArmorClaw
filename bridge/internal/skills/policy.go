package skills

import "time"

// ToolPolicy defines security policies for a tool
type ToolPolicy struct {
	Risk              string        // low, medium, high
	AutoExecute       bool          // Whether to auto-execute without confirmation
	Timeout           time.Duration // Maximum execution time
	MaxOutput         int           // Maximum output size in bytes
	AllowedSchemes    []string      // Allowed URL schemes (e.g., ["https"])
	BlockPrivateIPs   bool          // Whether to block private IP addresses
	BlockMetadata     bool          // Whether to block cloud metadata endpoints
	ResolveDNSFirst   bool          // Whether to resolve DNS before request
	MaxConcurrent     int           // Maximum concurrent executions
	RateLimit         int           // Maximum executions per minute
}

// Policy holds all tool policies
var Policy = map[string]ToolPolicy{
	"weather": {
		Risk:            "low",
		AutoExecute:     true,
		Timeout:         5 * time.Second,
		MaxOutput:       4096,
		AllowedSchemes:  []string{"https"},
		BlockPrivateIPs: true,
		BlockMetadata:   true,
		ResolveDNSFirst: true,
		MaxConcurrent:   1,
		RateLimit:       10,
	},
	"weather.get": {
		Risk:            "low",
		AutoExecute:     true,
		Timeout:         5 * time.Second,
		MaxOutput:       4096,
		AllowedSchemes:  []string{"https"},
		BlockPrivateIPs: true,
		BlockMetadata:   true,
		ResolveDNSFirst: true,
		MaxConcurrent:   1,
		RateLimit:       10,
	},

	"github.repo.info": {
		Risk:            "low",
		AutoExecute:     true,
		Timeout:         10 * time.Second,
		MaxOutput:       4096,
		AllowedSchemes:  []string{"https"},
		BlockPrivateIPs: true,
		BlockMetadata:   true,
		ResolveDNSFirst: true,
		MaxConcurrent:   1,
		RateLimit:       15,
	},

	"github.issue.create": {
		Risk:            "medium",
		AutoExecute:     true,
		Timeout:         10 * time.Second,
		MaxOutput:       4096,
		AllowedSchemes:  []string{"https"},
		BlockPrivateIPs: true,
		BlockMetadata:   true,
		ResolveDNSFirst: true,
		MaxConcurrent:   1,
		RateLimit:       5, // Write actions are more restricted
	},

	"web.fetch": {
		Risk:            "medium",
		AutoExecute:     true,
		Timeout:         15 * time.Second,
		MaxOutput:       8192, // Larger for web content
		AllowedSchemes:  []string{"https"},
		BlockPrivateIPs: true,
		BlockMetadata:   true,
		ResolveDNSFirst: true,
		MaxConcurrent:   2,
		RateLimit:       20,
	},
}

// GetPolicy returns the policy for a specific tool
func GetPolicy(toolName string) (*ToolPolicy, bool) {
	policy, exists := Policy[toolName]
	return &policy, exists
}

// IsToolAllowed checks if a tool is allowed based on policy
func IsToolAllowed(toolName string) bool {
	_, exists := Policy[toolName]
	return exists
}

// GetRiskLevel returns the risk level for a tool
func GetRiskLevel(toolName string) string {
	if policy, exists := Policy[toolName]; exists {
		return policy.Risk
	}
	return "high" // Default to high if not found
}

// ValidateTimeout checks if the execution time is within policy limits
func ValidateTimeout(toolName string, duration time.Duration) bool {
	if policy, exists := Policy[toolName]; exists {
		return duration <= policy.Timeout
	}
	return false
}

// ValidateOutputSize checks if the output size is within policy limits
func ValidateOutputSize(toolName string, size int) bool {
	if policy, exists := Policy[toolName]; exists {
		return size <= policy.MaxOutput
	}
	return false
}

// ValidateScheme checks if the URL scheme is allowed
func ValidateScheme(toolName string, scheme string) bool {
	if policy, exists := Policy[toolName]; exists {
		if len(policy.AllowedSchemes) == 0 {
			return true // No restriction
		}
		for _, allowed := range policy.AllowedSchemes {
			if scheme == allowed {
				return true
			}
		}
	}
	return false
}

// ShouldBlockPrivateIPs returns whether to block private IPs for a tool
func ShouldBlockPrivateIPs(toolName string) bool {
	if policy, exists := Policy[toolName]; exists {
		return policy.BlockPrivateIPs
	}
	return true // Default to blocking
}

// ShouldBlockMetadata returns whether to block metadata endpoints for a tool
func ShouldBlockMetadata(toolName string) bool {
	if policy, exists := Policy[toolName]; exists {
		return policy.BlockMetadata
	}
	return true // Default to blocking
}

// ShouldResolveDNSFirst returns whether to resolve DNS before request
func ShouldResolveDNSFirst(toolName string) bool {
	if policy, exists := Policy[toolName]; exists {
		return policy.ResolveDNSFirst
	}
	return true // Default to resolving first
}

// GetMaxConcurrent returns the maximum concurrent executions for a tool
func GetMaxConcurrent(toolName string) int {
	if policy, exists := Policy[toolName]; exists {
		return policy.MaxConcurrent
	}
	return 1 // Default to 1
}

// GetRateLimit returns the rate limit per minute for a tool
func GetRateLimit(toolName string) int {
	if policy, exists := Policy[toolName]; exists {
		return policy.RateLimit
	}
	return 5 // Default to 5
}