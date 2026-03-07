package router

import (
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/armorclaw/bridge/internal/cache"
	"github.com/armorclaw/bridge/internal/capability"
)

type Router struct {
	mu          sync.RWMutex
	domains     map[string]*Domain
	cache       *cache.LRU
	policy      *capability.Policy
}

type Domain struct {
	Name        string
	Keywords    []string
	Patterns    []*regexp.Regexp
	Tools       []string
	Priority    int
}

type RouterConfig struct {
	Policy     *capability.Policy
	CacheSize  int
}

func NewRouter(cfg RouterConfig) *Router {
	if cfg.CacheSize <= 0 {
		cfg.CacheSize = 1000
	}

	r := &Router{
		domains: make(map[string]*Domain),
		cache:   cache.NewLRU(cache.LRUConfig{MaxSize: cfg.CacheSize}),
		policy:  cfg.Policy,
	}

	r.initDefaultDomains()
	return r
}

func (r *Router) initDefaultDomains() {
	r.RegisterDomain(Domain{
		Name:     "weather",
		Keywords: []string{"weather", "temperature", "forecast", "rain", "sunny", "cloudy", "snow"},
		Tools:    []string{"weather.get", "weather.forecast"},
		Priority: 10,
	})

	r.RegisterDomain(Domain{
		Name:     "github",
		Keywords: []string{"github", "repo", "repository", "pull request", "issue", "commit", "branch"},
		Tools:    []string{"github.list", "github.get", "github.create"},
		Priority: 20,
	})

	r.RegisterDomain(Domain{
		Name:     "web",
		Keywords: []string{"url", "website", "http", "search", "browse", "fetch"},
		Patterns: []*regexp.Regexp{
			regexp.MustCompile(`https?://[^\s]+`),
		},
		Tools:    []string{"web.fetch", "web.search", "web.scrape"},
		Priority: 15,
	})

	r.RegisterDomain(Domain{
		Name:     "code",
		Keywords: []string{"code", "function", "variable", "debug", "error", "compile", "run"},
		Tools:    []string{"code.analyze", "code.run", "code.debug"},
		Priority: 25,
	})

	r.RegisterDomain(Domain{
		Name:     "general",
		Keywords: []string{},
		Tools:    []string{},
		Priority: 0,
	})
}

func (r *Router) RegisterDomain(d Domain) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.domains[d.Name] = &d
}

func (r *Router) Route(message string, roomID string) *RouteResult {
	cacheKey := message + ":" + roomID
	if cached, ok := r.cache.Get(cacheKey); ok {
		return cached.(*RouteResult)
	}

	r.mu.RLock()
	domains := make([]*Domain, 0, len(r.domains))
	for _, d := range r.domains {
		domains = append(domains, d)
	}
	r.mu.RUnlock()

	messageLower := strings.ToLower(message)

	var bestMatch *Domain
	bestScore := 0

	for _, domain := range domains {
		score := r.scoreDomain(domain, messageLower)
		if score > bestScore {
			bestScore = score
			bestMatch = domain
		}
	}

	if bestMatch == nil {
		bestMatch = r.domains["general"]
	}

	tools := bestMatch.Tools
	if r.policy != nil {
		tools = r.policy.FilterTools(roomID, tools)
	}

	result := &RouteResult{
		Domain:    bestMatch.Name,
		Tools:     tools,
		Confidence: float64(bestScore) / 100.0,
		RoutedAt:  time.Now(),
	}

	r.cache.Set(cacheKey, result)
	return result
}

func (r *Router) scoreDomain(domain *Domain, message string) int {
	score := 0

	for _, keyword := range domain.Keywords {
		if strings.Contains(message, strings.ToLower(keyword)) {
			score += 10
		}
	}

	for _, pattern := range domain.Patterns {
		if pattern.MatchString(message) {
			score += 20
		}
	}

	score += domain.Priority

	return score
}

func (r *Router) ClearCache() {
	r.cache.Clear()
}

type RouteResult struct {
	Domain     string    `json:"domain"`
	Tools      []string  `json:"tools"`
	Confidence float64   `json:"confidence"`
	RoutedAt   time.Time `json:"routed_at"`
}
