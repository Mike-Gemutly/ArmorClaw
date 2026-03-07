package petg

import (
	"context"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"
)

type CircuitState int

const (
	StateClosed CircuitState = iota
	StateOpen
	StateHalfOpen
)

type CircuitBreaker struct {
	mu               sync.RWMutex
	state            CircuitState
	failures         int
	successes        int
	failureThreshold int
	successThreshold int
	timeout          time.Duration
	lastFailure      time.Time
	name             string
}

type CircuitBreakerConfig struct {
	Name             string
	FailureThreshold int
	SuccessThreshold int
	Timeout          time.Duration
}

func NewCircuitBreaker(cfg CircuitBreakerConfig) *CircuitBreaker {
	if cfg.FailureThreshold <= 0 {
		cfg.FailureThreshold = 5
	}
	if cfg.SuccessThreshold <= 0 {
		cfg.SuccessThreshold = 2
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 30 * time.Second
	}
	return &CircuitBreaker{
		name:             cfg.Name,
		state:            StateClosed,
		failureThreshold: cfg.FailureThreshold,
		successThreshold: cfg.SuccessThreshold,
		timeout:          cfg.Timeout,
	}
}

func (cb *CircuitBreaker) Allow() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	switch cb.state {
	case StateClosed:
		return true
	case StateOpen:
		return time.Since(cb.lastFailure) > cb.timeout
	case StateHalfOpen:
		return true
	default:
		return false
	}
}

func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateClosed:
		cb.failures = 0
	case StateHalfOpen:
		cb.successes++
		if cb.successes >= cb.successThreshold {
			cb.state = StateClosed
			cb.failures = 0
			cb.successes = 0
		}
	}
}

func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.lastFailure = time.Now()

	switch cb.state {
	case StateClosed:
		cb.failures++
		if cb.failures >= cb.failureThreshold {
			cb.state = StateOpen
			cb.successes = 0
		}
	case StateHalfOpen:
		cb.state = StateOpen
		cb.successes = 0
	}
}

func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

func (cb *CircuitBreaker) Execute(fn func() error) error {
	if !cb.Allow() {
		return ErrCircuitOpen
	}

	err := fn()
	if err != nil {
		cb.RecordFailure()
		return err
	}

	cb.RecordSuccess()
	return nil
}

var ErrCircuitOpen = &CircuitError{Message: "circuit breaker is open"}

type CircuitError struct {
	Message string
}

func (e *CircuitError) Error() string {
	return e.Message
}

type Gateway struct {
	circuitBreaker *CircuitBreaker
	ssrfChecker    *SSRFChecker
	sanitizer      *Sanitizer
	outputFilter   *OutputFilter
}

type GatewayConfig struct {
	CircuitBreaker *CircuitBreaker
}

func NewGateway(cfg GatewayConfig) *Gateway {
	if cfg.CircuitBreaker == nil {
		cfg.CircuitBreaker = NewCircuitBreaker(CircuitBreakerConfig{
			Name: "petg",
		})
	}
	return &Gateway{
		circuitBreaker: cfg.CircuitBreaker,
		ssrfChecker:    NewSSRFChecker(),
		sanitizer:      NewSanitizer(),
		outputFilter:   NewOutputFilter(),
	}
}

func (g *Gateway) ValidateToolCall(ctx context.Context, toolName string, args map[string]interface{}) error {
	return g.circuitBreaker.Execute(func() error {
		if err := g.sanitizer.Sanitize(args); err != nil {
			return err
		}
		return g.ssrfChecker.Check(args)
	})
}

func (g *Gateway) FilterOutput(output string) string {
	return g.outputFilter.Filter(output)
}

func (g *Gateway) CircuitState() CircuitState {
	return g.circuitBreaker.State()
}

type SSRFChecker struct {
	privateIPPatterns []*regexp.Regexp
	allowedSchemes    map[string]bool
}

func NewSSRFChecker() *SSRFChecker {
	return &SSRFChecker{
		privateIPPatterns: []*regexp.Regexp{
			regexp.MustCompile(`^127\.`),
			regexp.MustCompile(`^10\.`),
			regexp.MustCompile(`^172\.(1[6-9]|2[0-9]|3[0-1])\.`),
			regexp.MustCompile(`^192\.168\.`),
			regexp.MustCompile(`^0\.0\.0\.0`),
			regexp.MustCompile(`^localhost`),
			regexp.MustCompile(`^::1$`),
		},
		allowedSchemes: map[string]bool{
			"http":  true,
			"https": true,
		},
	}
}

func (s *SSRFChecker) Check(args map[string]interface{}) error {
	for key, val := range args {
		if strings.Contains(strings.ToLower(key), "url") {
			urlStr, ok := val.(string)
			if !ok {
				continue
			}

			parsed, err := url.Parse(urlStr)
			if err != nil {
				return &SSRFError{Message: "invalid URL format"}
			}

			if !s.allowedSchemes[parsed.Scheme] {
				return &SSRFError{Message: "URL scheme not allowed: " + parsed.Scheme}
			}

			host := parsed.Hostname()
			for _, pattern := range s.privateIPPatterns {
				if pattern.MatchString(host) {
					return &SSRFError{Message: "access to private/internal addresses is blocked"}
				}
			}
		}
	}
	return nil
}

type SSRFError struct {
	Message string
}

func (e *SSRFError) Error() string {
	return e.Message
}

type Sanitizer struct {
	dangerousPatterns []*regexp.Regexp
}

func NewSanitizer() *Sanitizer {
	return &Sanitizer{
		dangerousPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)\b(password|secret|api[_-]?key|token|credential)\b`),
			regexp.MustCompile(`(?i)\b(private[_-]?key)\b`),
		},
	}
}

func (s *Sanitizer) Sanitize(args map[string]interface{}) error {
	for key, val := range args {
		strVal, ok := val.(string)
		if !ok {
			continue
		}

		for _, pattern := range s.dangerousPatterns {
			if pattern.MatchString(key) {
				return &SanitizeError{Message: "potentially sensitive field detected: " + key}
			}
		}

		for _, pattern := range s.dangerousPatterns {
			if pattern.MatchString(strVal) {
				args[key] = "[REDACTED]"
			}
		}
	}
	return nil
}

type SanitizeError struct {
	Message string
}

func (e *SanitizeError) Error() string {
	return e.Message
}

type OutputFilter struct {
	patterns []*regexp.Regexp
}

func NewOutputFilter() *OutputFilter {
	return &OutputFilter{
		patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)(password|secret|api[_-]?key|token)\s*[=:]\s*\S+`),
			regexp.MustCompile(`(?i)(bearer)\s+\S+`),
		},
	}
}

func (f *OutputFilter) Filter(output string) string {
	result := output
	for _, pattern := range f.patterns {
		result = pattern.ReplaceAllString(result, "$1: [REDACTED]")
	}
	return result
}
