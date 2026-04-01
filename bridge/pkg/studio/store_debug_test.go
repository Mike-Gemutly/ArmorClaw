package studio

import (
    "testing"
    "time"
)

func TestStoreDebug(t *testing.T) {
    store, err := NewStore(StoreConfig{Path: ":memory:"})
    if err != nil {
        t.Fatalf("failed to create store: %v", err)
    }
    defer store.Close()

    def := &AgentDefinition{
        ID:           "debug-test-001",
        Name:         "Debug Test Agent",
        Skills:       []string{"browser_navigate"},
        PIIAccess:    []string{"client_name"},
        ResourceTier: "medium",
        CreatedBy:    "@test:example.com",
        CreatedAt:    time.Now(),
        UpdatedAt:    time.Now(),
        IsActive:     true,
    }

    if err := store.CreateDefinition(def); err != nil {
        t.Fatalf("failed to create definition: %v", err)
    }

    // Try to get it back
    retrieved, err := store.GetDefinition("debug-test-001")
    if err != nil {
        t.Fatalf("failed to get definition: %v", err)
    }

    if retrieved.ID != "debug-test-001" {
        t.Errorf("expected ID debug-test-001, got %s", retrieved.ID)
    }
}
