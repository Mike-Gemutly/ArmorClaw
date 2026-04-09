package cdp

import (
	"log"
	"strings"
)

type MethodRoute struct {
	Pattern string
	Handler MessageHandler
	Action  RouteAction
}

type MessageHandler func(*CDPMessage) (*CDPMessage, error)

type RouteAction string

const (
	ActionTranslate   RouteAction = "translate"
	ActionPassthrough RouteAction = "passthrough"
	ActionUnsupported RouteAction = "unsupported"
)

type MethodRouter struct {
	routes     []MethodRoute
	defaults   map[string]RouteAction
	wildcards  []MethodRoute
	translator *Translator
}

func NewMethodRouter(translator *Translator) *MethodRouter {
	r := &MethodRouter{
		defaults:   make(map[string]RouteAction),
		translator: translator,
	}
	r.configureDefaults()
	r.configureRoutes()
	return r
}

func (r *MethodRouter) configureDefaults() {
	r.defaults["Page"] = ActionPassthrough
	r.defaults["Runtime"] = ActionTranslate
	r.defaults["Input"] = ActionPassthrough
	r.defaults["Network"] = ActionPassthrough
	r.defaults["DOM"] = ActionPassthrough
	r.defaults["Target"] = ActionPassthrough
	r.defaults["Browser"] = ActionPassthrough
	r.defaults["Emulation"] = ActionPassthrough
	r.defaults["Fetch"] = ActionPassthrough
	r.defaults["Security"] = ActionPassthrough
	r.defaults["Performance"] = ActionPassthrough
	r.defaults["Schema"] = ActionPassthrough
}

func (r *MethodRouter) configureRoutes() {
	r.routes = []MethodRoute{
		{
			Pattern: "Input.dispatchMouseEvent",
			Handler: r.handleMouseClick,
			Action:  ActionTranslate,
		},
		{
			Pattern: "Input.dispatchKeyEvent",
			Handler: r.handleKeyInput,
			Action:  ActionTranslate,
		},
		{
			Pattern: "Input.insertText",
			Handler: r.handleTextInsert,
			Action:  ActionTranslate,
		},
		{
			Pattern: "Target.*",
			Handler: nil,
			Action:  ActionPassthrough,
		},
	}
	r.identifyWildcards()
}

func (r *MethodRouter) identifyWildcards() {
	r.wildcards = make([]MethodRoute, 0, len(r.routes))
	for _, route := range r.routes {
		if strings.Contains(route.Pattern, "*") {
			r.wildcards = append(r.wildcards, route)
		}
	}
}

func matchWildcard(pattern, method string) bool {
	if strings.HasSuffix(pattern, ".*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(method, prefix)
	}
	return pattern == method
}

func (r *MethodRouter) Route(method string) *MethodRoute {
	for _, route := range r.routes {
		if route.Pattern == method {
			return &route
		}
	}

	for _, wildcard := range r.wildcards {
		if matchWildcard(wildcard.Pattern, method) {
			return &wildcard
		}
	}

	domain := strings.Split(method, ".")[0]
	if action, exists := r.defaults[domain]; exists {
		return &MethodRoute{
			Pattern: method,
			Action:  action,
			Handler: nil,
		}
	}

	return &MethodRoute{
		Pattern: method,
		Action:  ActionUnsupported,
		Handler: r.handleUnsupported,
	}
}

func (r *MethodRouter) handleMouseClick(msg *CDPMessage) (*CDPMessage, error) {
	return r.translate(msg)
}

func (r *MethodRouter) handleKeyInput(msg *CDPMessage) (*CDPMessage, error) {
	return r.translate(msg)
}

func (r *MethodRouter) handleTextInsert(msg *CDPMessage) (*CDPMessage, error) {
	return r.translate(msg)
}

func (r *MethodRouter) translate(msg *CDPMessage) (*CDPMessage, error) {
	result, err := r.translator.Translate(msg)
	if err != nil {
		log.Printf("[JETSKI ROUTER]: translation failed for %s: %v", msg.Method, err)
		return msg, nil
	}
	return result, nil
}

func (r *MethodRouter) handleUnsupported(msg *CDPMessage) (*CDPMessage, error) {
	return msg, nil
}
