package provisioning

import (
	"fmt"
	"os"
)

// ConfigLoader loads provisioning configuration from TOML
type ConfigLoader struct {
	SigningSecret       string
	DefaultExpirySecs   int
	MaxExpirySecs       int
	OneTimeUse          bool
	BridgePublicKey     string
	MatrixHomeserver    string
	RPCURL              string
	WSURL               string
	PushGateway         string
	ServerName          string
	Region              string
}

// LoadFromEnv loads configuration from environment variables
func LoadFromEnv() *ConfigLoader {
	return &ConfigLoader{
		SigningSecret:     getEnv("ARMORCLAW_PROVISIONING_SECRET", ""),
		DefaultExpirySecs: getEnvInt("ARMORCLAW_PROVISIONING_DEFAULT_EXPIRY", 60),
		MaxExpirySecs:     getEnvInt("ARMORCLAW_PROVISIONING_MAX_EXPIRY", 300),
		OneTimeUse:        getEnvBool("ARMORCLAW_PROVISIONING_ONE_TIME_USE", true),
		BridgePublicKey:   getEnv("ARMORCLAW_BRIDGE_PUBLIC_KEY", ""),
		MatrixHomeserver:  getEnv("ARMORCLAW_MATRIX_HOMESERVER", ""),
		RPCURL:            getEnv("ARMORCLAW_RPC_URL", ""),
		WSURL:             getEnv("ARMORCLAW_WS_URL", ""),
		PushGateway:       getEnv("ARMORCLAW_PUSH_GATEWAY", ""),
		ServerName:        getEnv("ARMORCLAW_SERVER_NAME", ""),
		Region:            getEnv("ARMORCLAW_REGION", ""),
	}
}

// Validate validates the configuration
func (c *ConfigLoader) Validate() error {
	if c.SigningSecret == "" {
		return fmt.Errorf("provisioning.signing_secret is required")
	}
	if c.MatrixHomeserver == "" {
		return fmt.Errorf("matrix homeserver URL is required")
	}
	if c.RPCURL == "" {
		return fmt.Errorf("RPC URL is required")
	}
	if c.WSURL == "" {
		return fmt.Errorf("WebSocket URL is required")
	}
	return nil
}

// ToManagerConfig converts to ManagerConfig
func (c *ConfigLoader) ToManagerConfig() *ManagerConfig {
	return &ManagerConfig{
		SigningSecret:       c.SigningSecret,
		DefaultExpirySeconds: c.DefaultExpirySecs,
		MaxExpirySeconds:     c.MaxExpirySecs,
		OneTimeUse:          c.OneTimeUse,
		BridgePublicKey:    c.BridgePublicKey,
		ServerConfig:       c,
	}
}

// Implement ServerConfigProvider
func (c *ConfigLoader) GetMatrixHomeserver() string { return c.MatrixHomeserver }
func (c *ConfigLoader) GetRPCURL() string           { return c.RPCURL }
func (c *ConfigLoader) GetWSURL() string            { return c.WSURL }
func (c *ConfigLoader) GetPushGateway() string      { return c.PushGateway }
func (c *ConfigLoader) GetServerName() string       { return c.ServerName }
func (c *ConfigLoader) GetRegion() string           { return c.Region }

// Helper functions
func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		var result int
		if _, err := fmt.Sscanf(val, "%d", &result); err == nil {
			return result
		}
	}
	return defaultVal
}

func getEnvBool(key string, defaultVal bool) bool {
	if val := os.Getenv(key); val != "" {
		return val == "true" || val == "1" || val == "yes"
	}
	return defaultVal
}
