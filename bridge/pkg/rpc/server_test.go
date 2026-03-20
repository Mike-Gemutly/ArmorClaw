package rpc

import (
	"testing"
)

func TestMethodRegistration(t *testing.T) {
	criticalMethods := []string{
		"matrix.status",
		"matrix.login",
		"matrix.send",
		"ai.chat",
	}

	server := &Server{}
	server.registerHandlers()

	for _, method := range criticalMethods {
		t.Run(method, func(t *testing.T) {
			if _, exists := server.handlers[method]; !exists {
				t.Errorf("critical method %q not registered in handlers map", method)
			}
		})
	}
}

func TestMethodRegistrationCompleteness(t *testing.T) {
	expectedMethods := []string{
		"ai.chat",
		"browser.navigate",
		"browser.fill",
		"browser.click",
		"browser.status",
		"matrix.status",
		"matrix.login",
		"matrix.send",
	}

	server := &Server{}
	server.registerHandlers()

	for _, method := range expectedMethods {
		t.Run(method, func(t *testing.T) {
			if _, exists := server.handlers[method]; !exists {
				t.Errorf("expected method %q not registered in handlers map", method)
			}
		})
	}
}
