// setup_mcp.go — MCP router wiring
//
// Extracted from runBridgeServer(). Pure extraction, no logic changes.
package main

import (
	"log"
	"time"

	"github.com/armorclaw/bridge/pkg/audit"
	"github.com/armorclaw/bridge/pkg/config"
	"github.com/armorclaw/bridge/pkg/governor"
	"github.com/armorclaw/bridge/pkg/mcp"
	"github.com/armorclaw/bridge/pkg/pii"
	"github.com/armorclaw/bridge/pkg/toolsidecar"
	"github.com/armorclaw/bridge/pkg/translator"
	"github.com/armorclaw/bridge/pkg/vault"
)

// setupMCPRouter initializes the v6 MCP Router when V6Microkernel is enabled.
// Returns the router and RPC-to-MCP translator (either may be nil if disabled or on error).
func setupMCPRouter(cfg *config.Config, toolsidecarDocker *toolsidecarDockerAdapter, vaultClient *vault.VaultGovernanceClient) (*mcp.MCPRouter, *translator.RPCToMCPTranslator) {
	var mcpRouter *mcp.MCPRouter
	var mcpTranslator *translator.RPCToMCPTranslator

	if cfg.Vault.V6Microkernel {
		log.Println("Initializing v6 Microkernel MCP Router...")
		mcpTranslator = translator.NewRPCToMCPTranslator()

		gov := governor.NewGovernor(nil, nil)
		prov, provErr := toolsidecar.NewProvisioner(toolsidecar.Config{
			DockerClient: toolsidecarDocker,
		})
		if provErr != nil {
			log.Printf("V6 Microkernel disabled: toolsidecar provisioner: %v", provErr)
		} else {
			consentMgr := pii.NewHITLConsentManager(pii.HITLConfig{
				Timeout: 60 * time.Second,
			})
			auditor, auditErr := audit.NewAuditLog(audit.DefaultConfig())
			if auditErr != nil {
				log.Printf("V6 Microkernel disabled: audit log: %v", auditErr)
			} else {
				var err error
				mcpRouter, err = mcp.New(mcp.Config{
					SkillGate:      gov,
					Provisioner:    prov,
					ConsentManager: consentMgr,
					Auditor:        auditor,
					VaultClient:    vaultClient,
					V6Microkernel:  true,
				})
				if err != nil {
					log.Printf("V6 Microkernel disabled: %v", err)
					mcpRouter = nil
				} else {
					log.Println("v6 Microkernel MCP Router initialized")
				}
			}
		}
	}

	return mcpRouter, mcpTranslator
}
