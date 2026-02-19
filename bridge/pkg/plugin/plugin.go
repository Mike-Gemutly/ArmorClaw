// Package plugin provides a plugin system for external adapters.
// This enables third-party platforms to be added without modifying the bridge core.
package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"plugin"
	"sync"
	"time"

	"github.com/armorclaw/bridge/internal/sdtw"
)

// PluginVersion defines the current plugin API version
const PluginAPIVersion = "1.0.0"

// PluginType defines the type of plugin
type PluginType string

const (
	PluginTypeAdapter   PluginType = "adapter"   // Platform adapter
	PluginTypeMiddleware PluginType = "middleware" // Message middleware
	PluginTypeNotifier  PluginType = "notifier"  // Notification handler
)

// PluginMetadata contains plugin identification and capability info
type PluginMetadata struct {
	// Required fields
	Name         string     `json:"name"`          // Unique plugin name (e.g., "telegram-adapter")
	Version      string     `json:"version"`       // Plugin version (semver)
	APIVersion   string     `json:"api_version"`   // Required API version (must match PluginAPIVersion)
	Type         PluginType `json:"type"`          // Plugin type
	Description  string     `json:"description"`   // Human-readable description
	Author       string     `json:"author"`        // Plugin author
	License      string     `json:"license"`       // License identifier (e.g., "MIT", "Apache-2.0")

	// Platform info (for adapter plugins)
	Platform     string     `json:"platform,omitempty"`     // Platform name (e.g., "telegram")
	Capabilities sdtw.CapabilitySet `json:"capabilities,omitempty"` // Platform capabilities

	// Dependencies
	Dependencies []string   `json:"dependencies,omitempty"` // Required other plugins

	// Configuration
	ConfigSchema json.RawMessage `json:"config_schema,omitempty"` // JSON Schema for configuration
}

// PluginConfig contains runtime configuration for a plugin
type PluginConfig struct {
	// Path to the plugin shared library (.so on Linux, .dylib on macOS)
	LibraryPath string `json:"library_path"`

	// Path to plugin metadata JSON file
	MetadataPath string `json:"metadata_path,omitempty"`

	// Enable the plugin
	Enabled bool `json:"enabled"`

	// Plugin-specific configuration (validated against ConfigSchema)
	Config map[string]interface{} `json:"config,omitempty"`

	// Credentials (injected from keystore)
	Credentials map[string]string `json:"credentials,omitempty"`
}

// PluginState represents the current state of a plugin
type PluginState string

const (
	PluginStateUnloaded  PluginState = "unloaded"  // Plugin not loaded
	PluginStateLoaded    PluginState = "loaded"    // Plugin loaded but not initialized
	PluginStateInit      PluginState = "initialized" // Plugin initialized
	PluginStateRunning   PluginState = "running"   // Plugin is running
	PluginStateError     PluginState = "error"     // Plugin encountered error
	PluginStateDisabled  PluginState = "disabled"  // Plugin is disabled
)

// PluginInfo contains runtime information about a loaded plugin
type PluginInfo struct {
	Metadata  PluginMetadata `json:"metadata"`
	State     PluginState    `json:"state"`
	LastError string         `json:"last_error,omitempty"`
	LoadTime  time.Time      `json:"load_time,omitempty"`
	StartTime time.Time      `json:"start_time,omitempty"`
}

// PluginInterface is the interface that all plugins must implement
// This matches Go's plugin system requirements
type PluginInterface interface {
	// Metadata returns plugin identification and capabilities
	Metadata() PluginMetadata

	// Initialize sets up the plugin with configuration
	Initialize(ctx context.Context, config PluginConfig) error

	// Start begins plugin operation
	Start(ctx context.Context) error

	// Stop gracefully stops the plugin
	Stop(ctx context.Context) error

	// HealthCheck returns the plugin's health status
	HealthCheck() error
}

// AdapterPlugin extends PluginInterface for adapter-specific functionality
type AdapterPlugin interface {
	PluginInterface

	// GetAdapter returns the SDTW adapter implementation
	GetAdapter() (sdtw.SDTWAdapter, error)
}

// PluginManager manages plugin lifecycle
type PluginManager struct {
	mu      sync.RWMutex
	plugins map[string]*loadedPlugin
	config  ManagerConfig
}

// ManagerConfig configures the plugin manager
type ManagerConfig struct {
	// Plugin directory containing .so files and metadata
	PluginDir string `json:"plugin_dir"`

	// Enable hot reloading (watch for file changes)
	EnableHotReload bool `json:"enable_hot_reload"`

	// Auto-discover plugins in directory
	AutoDiscover bool `json:"auto_discover"`

	// Plugin search patterns
	SearchPatterns []string `json:"search_patterns"`
}

// loadedPlugin represents a loaded plugin instance
type loadedPlugin struct {
	info     PluginInfo
	config   PluginConfig
	instance PluginInterface
	rawLib   *plugin.Plugin
}

// NewPluginManager creates a new plugin manager
func NewPluginManager(config ManagerConfig) *PluginManager {
	if config.PluginDir == "" {
		config.PluginDir = "/var/lib/armorclaw/plugins"
	}
	if len(config.SearchPatterns) == 0 {
		config.SearchPatterns = []string{"*.so", "*.plugin"}
	}

	return &PluginManager{
		plugins: make(map[string]*loadedPlugin),
		config:  config,
	}
}

// DiscoverPlugins finds all available plugins in the plugin directory
func (pm *PluginManager) DiscoverPlugins() ([]PluginMetadata, error) {
	var plugins []PluginMetadata

	// Ensure plugin directory exists
	if _, err := os.Stat(pm.config.PluginDir); os.IsNotExist(err) {
		return plugins, nil // No plugins directory yet
	}

	// Walk the plugin directory
	err := filepath.Walk(pm.config.PluginDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip files we can't access
		}

		// Check for metadata files
		if filepath.Ext(path) == ".json" {
			metadata, err := pm.loadMetadata(path)
			if err == nil {
				plugins = append(plugins, metadata)
			}
		}

		return nil
	})

	return plugins, err
}

// LoadPlugin loads a plugin from the given configuration
func (pm *PluginManager) LoadPlugin(config PluginConfig) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Check if already loaded
	if _, exists := pm.plugins[config.LibraryPath]; exists {
		return fmt.Errorf("plugin already loaded: %s", config.LibraryPath)
	}

	// Verify library exists
	if _, err := os.Stat(config.LibraryPath); os.IsNotExist(err) {
		return fmt.Errorf("plugin library not found: %s", config.LibraryPath)
	}

	// Load metadata
	var metadata PluginMetadata
	if config.MetadataPath != "" {
		loadedMeta, err := pm.loadMetadata(config.MetadataPath)
		if err != nil {
			return fmt.Errorf("failed to load metadata: %w", err)
		}
		metadata = loadedMeta
	}

	// Verify API version
	if metadata.APIVersion != "" && metadata.APIVersion != PluginAPIVersion {
		return fmt.Errorf("plugin API version mismatch: plugin=%s, bridge=%s",
			metadata.APIVersion, PluginAPIVersion)
	}

	// Load the shared library
	rawLib, err := plugin.Open(config.LibraryPath)
	if err != nil {
		return fmt.Errorf("failed to load plugin library: %w", err)
	}

	// Look up the plugin symbol
	sym, err := rawLib.Lookup("Plugin")
	if err != nil {
		return fmt.Errorf("plugin does not export 'Plugin' symbol: %w", err)
	}

	// Type assertion to PluginInterface
	pluginInstance, ok := sym.(PluginInterface)
	if !ok {
		return fmt.Errorf("plugin does not implement PluginInterface")
	}

	// Store the loaded plugin
	pm.plugins[config.LibraryPath] = &loadedPlugin{
		info: PluginInfo{
			Metadata: metadata,
			State:    PluginStateLoaded,
			LoadTime: time.Now(),
		},
		config:   config,
		instance: pluginInstance,
		rawLib:   rawLib,
	}

	return nil
}

// InitializePlugin initializes a loaded plugin
func (pm *PluginManager) InitializePlugin(name string, config PluginConfig) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	plugin, exists := pm.plugins[name]
	if !exists {
		return fmt.Errorf("plugin not found: %s", name)
	}

	if err := plugin.instance.Initialize(context.Background(), config); err != nil {
		plugin.info.State = PluginStateError
		plugin.info.LastError = err.Error()
		return fmt.Errorf("failed to initialize plugin: %w", err)
	}

	plugin.info.State = PluginStateInit
	plugin.config = config

	return nil
}

// StartPlugin starts an initialized plugin
func (pm *PluginManager) StartPlugin(name string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	plugin, exists := pm.plugins[name]
	if !exists {
		return fmt.Errorf("plugin not found: %s", name)
	}

	if plugin.info.State != PluginStateInit {
		return fmt.Errorf("plugin not initialized: current state=%s", plugin.info.State)
	}

	if err := plugin.instance.Start(context.Background()); err != nil {
		plugin.info.State = PluginStateError
		plugin.info.LastError = err.Error()
		return fmt.Errorf("failed to start plugin: %w", err)
	}

	plugin.info.State = PluginStateRunning
	plugin.info.StartTime = time.Now()

	return nil
}

// StopPlugin stops a running plugin
func (pm *PluginManager) StopPlugin(name string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	plugin, exists := pm.plugins[name]
	if !exists {
		return fmt.Errorf("plugin not found: %s", name)
	}

	if err := plugin.instance.Stop(context.Background()); err != nil {
		plugin.info.LastError = err.Error()
		return fmt.Errorf("failed to stop plugin: %w", err)
	}

	plugin.info.State = PluginStateInit

	return nil
}

// UnloadPlugin unloads a plugin completely
func (pm *PluginManager) UnloadPlugin(name string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	plugin, exists := pm.plugins[name]
	if !exists {
		return fmt.Errorf("plugin not found: %s", name)
	}

	// Stop if running
	if plugin.info.State == PluginStateRunning {
		plugin.instance.Stop(context.Background())
	}

	// Remove from map (Go plugins can't be unloaded, but we can remove the reference)
	delete(pm.plugins, name)

	return nil
}

// GetPlugin returns information about a loaded plugin
func (pm *PluginManager) GetPlugin(name string) (*PluginInfo, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	plugin, exists := pm.plugins[name]
	if !exists {
		return nil, fmt.Errorf("plugin not found: %s", name)
	}

	info := plugin.info
	return &info, nil
}

// ListPlugins returns information about all loaded plugins
func (pm *PluginManager) ListPlugins() []PluginInfo {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	var list []PluginInfo
	for _, plugin := range pm.plugins {
		list = append(list, plugin.info)
	}

	return list
}

// GetAdapter returns the SDTW adapter for an adapter plugin
func (pm *PluginManager) GetAdapter(name string) (sdtw.SDTWAdapter, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	plugin, exists := pm.plugins[name]
	if !exists {
		return nil, fmt.Errorf("plugin not found: %s", name)
	}

	adapterPlugin, ok := plugin.instance.(AdapterPlugin)
	if !ok {
		return nil, fmt.Errorf("plugin is not an adapter plugin: %s", name)
	}

	return adapterPlugin.GetAdapter()
}

// HealthCheck checks the health of all plugins
func (pm *PluginManager) HealthCheck() map[string]error {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	results := make(map[string]error)
	for name, plugin := range pm.plugins {
		if plugin.info.State == PluginStateRunning {
			results[name] = plugin.instance.HealthCheck()
		}
	}

	return results
}

// loadMetadata loads plugin metadata from a JSON file
func (pm *PluginManager) loadMetadata(path string) (PluginMetadata, error) {
	var metadata PluginMetadata

	data, err := os.ReadFile(path)
	if err != nil {
		return metadata, fmt.Errorf("failed to read metadata file: %w", err)
	}

	if err := json.Unmarshal(data, &metadata); err != nil {
		return metadata, fmt.Errorf("failed to parse metadata: %w", err)
	}

	// Validate required fields
	if metadata.Name == "" {
		return metadata, fmt.Errorf("metadata missing required field: name")
	}
	if metadata.Version == "" {
		return metadata, fmt.Errorf("metadata missing required field: version")
	}
	if metadata.Type == "" {
		return metadata, fmt.Errorf("metadata missing required field: type")
	}

	return metadata, nil
}
