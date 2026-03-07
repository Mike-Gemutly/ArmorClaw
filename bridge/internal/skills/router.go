package skills

import (
	"strings"
)

// Router handles domain routing for tool selection
type Router struct {
	keywordMap map[string]string
}

// DomainScore represents a domain with its score
type DomainScore struct {
	Domain string
	Score  int
}

// NewRouter creates a new domain router
func NewRouter() *Router {
	return &Router{
		keywordMap: map[string]string{
			// Weather keywords
			"weather": "weather",
			"forecast": "weather",
			"temperature": "weather",
			"rain": "weather",
			"cloud": "weather",
			"sun": "weather",
			"wind": "weather",
			
			// GitHub keywords
			"github": "github",
			"repo": "github",
			"repository": "github",
			"issue": "github",
			"pr": "github",
			"pull": "github",
			"commit": "github",
			"code": "github",
			"git": "github",
			
			// Web keywords
			"web": "web",
			"fetch": "web",
			"http": "web",
			"https": "web",
			"url": "web",
			"website": "web",
			"page": "web",
			"download": "web",
			"link": "web",
		},
	}
}

// Route determines the best domain for a user message
func (r *Router) Route(message string) string {
	message = strings.ToLower(message)
	
	// Score each domain
	domains := map[string]int{
		"github":  r.scoreDomain(message, "github"),
		"weather": r.scoreDomain(message, "weather"),
		"web":     r.scoreDomain(message, "web"),
	}
	
	// Find domain with highest score
	bestDomain := "general"
	bestScore := 0
	
	for domain, score := range domains {
		if score > bestScore {
			bestScore = score
			bestDomain = domain
		}
	}
	
	// If no domain scored above 0, return general
	if bestScore == 0 {
		return "general"
	}
	
	return bestDomain
}

// scoreDomain calculates a score for a domain based on keyword matches
func (r *Router) scoreDomain(message string, domain string) int {
	score := 0
	
	// Keywords with different weights
	keywordWeights := map[string]int{
		// High weight (primary keywords)
		"github": 3,
		"weather": 3,
		"web": 3,
		
		// Medium weight (related keywords)
		"repo": 2,
		"repository": 2,
		"forecast": 2,
		"fetch": 2,
		"http": 2,
		"https": 2,
		
		// Low weight (associated keywords)
		"issue": 1,
		"pr": 1,
		"temperature": 1,
		"cloud": 1,
		"url": 1,
		"website": 1,
	}
	
	for keyword, targetDomain := range r.keywordMap {
		if targetDomain != domain {
			continue
		}
		
		if strings.Contains(message, keyword) {
			weight := keywordWeights[keyword]
			if weight == 0 {
				weight = 1 // Default weight if not defined
			}
			score += weight
		}
	}
	
	return score
}

// GetDomains returns all available domains
func (r *Router) GetDomains() []string {
	return []string{"github", "weather", "web", "general"}
}

// AddKeyword adds a keyword to the router
func (r *Router) AddKeyword(keyword, domain string) {
	r.keywordMap[strings.ToLower(keyword)] = domain
}

// RemoveKeyword removes a keyword from the router
func (r *Router) RemoveKeyword(keyword string) {
	delete(r.keywordMap, strings.ToLower(keyword))
}

// GetKeywordsForDomain returns all keywords for a domain
func (r *Router) GetKeywordsForDomain(domain string) []string {
	var keywords []string
	for keyword, targetDomain := range r.keywordMap {
		if targetDomain == domain {
			keywords = append(keywords, keyword)
		}
	}
	return keywords
}

// ScoreMessage returns the domain scores for a message
func (r *Router) ScoreMessage(message string) []DomainScore {
	message = strings.ToLower(message)
	
	domains := []DomainScore{
		{Domain: "github", Score: r.scoreDomain(message, "github")},
		{Domain: "weather", Score: r.scoreDomain(message, "weather")},
		{Domain: "web", Score: r.scoreDomain(message, "web")},
		{Domain: "general", Score: 0},
	}
	
	return domains
}