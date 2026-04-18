// setup_vault.go — Vault governance client wiring
//
// Extracted from runBridgeServer(). Pure extraction, no logic changes.
package main

import (
	"context"
	"log"

	"github.com/armorclaw/bridge/pkg/config"
	"github.com/armorclaw/bridge/pkg/eventbus"
	"github.com/armorclaw/bridge/pkg/vault"
)

// setupVaultClient initializes the vault governance client when v6_microkernel is enabled.
// Returns the client and event bridge (either may be nil if disabled or on error).
func setupVaultClient(cfg config.Config, eventBus *eventbus.EventBus, shutdownCtx context.Context) (*vault.VaultGovernanceClient, *vault.VaultEventBridge) {
	var vaultClient *vault.VaultGovernanceClient
	var vaultEventBridge *vault.VaultEventBridge

	if cfg.Vault.V6Microkernel {
		log.Println("[VAULT] v6 microkernel enabled, connecting to vault governance...")
		var err error
		vaultClient, err = vault.NewGovernanceClient(cfg.Vault.SocketPath)
		if err != nil {
			log.Printf("[VAULT] Failed to connect to vault governance: %v (degrading gracefully)", err)
		} else {
			if eventBus != nil {
				vaultEventBridge = vault.NewVaultEventBridge(vaultClient, eventBus)
				go vaultEventBridge.StartSyncLoop(shutdownCtx)
				log.Println("[VAULT] Vault governance client connected, event bridge started")
			} else {
				log.Println("[VAULT] Vault governance client connected (event bus unavailable, event bridge skipped)")
			}
		}
	}

	return vaultClient, vaultEventBridge
}
