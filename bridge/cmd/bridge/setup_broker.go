package main

import (
	"log"

	"github.com/armorclaw/bridge/pkg/capability"
	"github.com/armorclaw/bridge/pkg/interfaces"
)

// setupBroker creates the CapabilityBroker with its dependencies.
// Returns nil when governance is not enabled (broker is optional).
func setupBroker(classifier interfaces.RiskClassifier, registry interfaces.CapabilityRegistry, consent interfaces.ConsentProvider, skillGate interfaces.SkillGate) *capability.Broker {
	if classifier == nil {
		log.Println("[BROKER] no risk classifier — broker not created")
		return nil
	}

	cfg := capability.DefaultBrokerConfig()
	broker := capability.NewBroker(classifier, registry, consent, skillGate, cfg)
	log.Println("[BROKER] capability broker created with default config")
	return broker
}
