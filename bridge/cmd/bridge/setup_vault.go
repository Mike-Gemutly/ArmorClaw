// setup_vault.go — Vault governance client wiring
//
// Extracted from runBridgeServer(). Pure extraction, no logic changes.
package main

import (
	"context"
	"log"
	"os"

	"github.com/armorclaw/bridge/pkg/config"
	"github.com/armorclaw/bridge/pkg/eventbus"
	"github.com/armorclaw/bridge/pkg/vault"
)

// setupVaultClient initializes the vault governance client when v6_microkernel is enabled.
// Returns the client and event bridge (either may be nil if disabled or on error).
//
// Activation priority:
//  1. V6Microkernel=true in config → always enable
//  2. V6Microkernel=false in config → skip (explicit opt-out)
//  3. Config not set (default false) → auto-detect by checking socket presence
func setupVaultClient(cfg *config.Config, eventBus *eventbus.EventBus, shutdownCtx context.Context) (*vault.VaultGovernanceClient, *vault.VaultEventBridge) {
	var vaultClient *vault.VaultGovernanceClient
	var vaultEventBridge *vault.VaultEventBridge

	shouldEnable := cfg.Vault.V6Microkernel

	if !shouldEnable {
		if _, err := os.Stat(cfg.Vault.SocketPath); err == nil {
			log.Println("[VAULT] vault socket auto-detected at", cfg.Vault.SocketPath, "— enabling governance")
			shouldEnable = true
		}
	}

	if shouldEnable {
		log.Println("[VAULT] connecting to vault governance...")
		var err error
		vaultClient, err = vault.NewGovernanceClient(cfg.Vault.SocketPath)
		if err != nil {
			log.Printf("[VAULT] failed to connect to vault governance: %v (degrading gracefully)", err)
		} else {
			if eventBus != nil {
				vaultEventBridge = vault.NewVaultEventBridge(vaultClient, eventBus)
				go vaultEventBridge.StartSyncLoop(shutdownCtx)
				log.Println("[VAULT] vault governance client connected, event bridge started")
			} else {
				log.Println("[VAULT] vault governance client connected (event bus unavailable, event bridge skipped)")
			}
		}
	}

	return vaultClient, vaultEventBridge
}
