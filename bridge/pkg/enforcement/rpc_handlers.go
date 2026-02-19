// Package enforcement provides RPC handlers for license management
package enforcement

import (
	"encoding/json"

	"github.com/armorclaw/bridge/pkg/rpc"
)

// ServerInterface defines the interface for RPC server (to avoid import cycle)
// This is a local copy for the handlers package

// HandleLicenseStatus handles license.status RPC method
func HandleLicenseStatus(manager *Manager, req *rpc.Request) *rpc.Response {
	info := manager.GetLicenseInfo()

	return &rpc.Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]interface{}{
			"tier":            info["tier"],
			"valid":           info["valid"],
			"compliance_mode": manager.GetComplianceMode(),
			"expires_at":      info["expires_at"],
			"features":        info["features"],
		},
	}
}

// HandleLicenseFeatures handles license.features RPC method
func HandleLicenseFeatures(manager *Manager, req *rpc.Request) *rpc.Response {
	return &rpc.Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]interface{}{
			"available": manager.GetAvailableFeatures(),
			"all":       manager.GetAllFeatures(),
		},
	}
}

// HandleLicenseCheckFeature handles license.check_feature RPC method
func HandleLicenseCheckFeature(manager *Manager, req *rpc.Request) *rpc.Response {
	var params struct {
		Feature string `json:"feature"`
	}

	if len(req.Params) == 0 {
		return &rpc.Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &rpc.ErrorObj{
				Code:    rpc.InvalidParams,
				Message: "feature parameter required",
			},
		}
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return &rpc.Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &rpc.ErrorObj{
				Code:    rpc.InvalidParams,
				Message: err.Error(),
			},
		}
	}

	allowed, err := manager.CheckFeature(Feature(params.Feature))
	if err != nil {
		return &rpc.Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &rpc.ErrorObj{
				Code:    rpc.InternalError,
				Message: err.Error(),
			},
		}
	}

	return &rpc.Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]interface{}{
			"feature":       params.Feature,
			"allowed":       allowed,
			"current_tier":  manager.GetTier(),
			"compliance_mode": manager.GetComplianceMode(),
		},
	}
}

// HandleComplianceStatus handles compliance.status RPC method
func HandleComplianceStatus(manager *Manager, req *rpc.Request) *rpc.Response {
	mode := manager.GetComplianceMode()

	// Build compliance details based on mode
	details := map[string]interface{}{
		"mode": mode,
	}

	switch mode {
	case ComplianceModeNone:
		details["phi_scrubbing"] = false
		details["audit_logging"] = false
		details["tamper_evidence"] = false
		details["quarantine"] = false
	case ComplianceModeBasic:
		details["phi_scrubbing"] = false
		details["audit_logging"] = false
		details["tamper_evidence"] = false
		details["quarantine"] = false
	case ComplianceModeStandard:
		details["phi_scrubbing"] = true
		details["audit_logging"] = true
		details["tamper_evidence"] = false
		details["quarantine"] = false
	case ComplianceModeFull:
		details["phi_scrubbing"] = true
		details["audit_logging"] = true
		details["tamper_evidence"] = true
		details["quarantine"] = false
	case ComplianceModeStrict:
		details["phi_scrubbing"] = true
		details["audit_logging"] = true
		details["tamper_evidence"] = true
		details["quarantine"] = true
	}

	return &rpc.Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  details,
	}
}

// HandlePlatformLimits handles platform.limits RPC method
func HandlePlatformLimits(manager *Manager, req *rpc.Request) *rpc.Response {
	platforms := []string{"slack", "discord", "teams", "whatsapp"}
	limits := make(map[string]interface{})

	for _, p := range platforms {
		limit, err := manager.GetPlatformLimit(p)
		if err != nil {
			continue
		}

		allowed, _ := manager.CanBridgePlatform(p)

		limits[p] = map[string]interface{}{
			"enabled":       allowed,
			"max_channels":  limit.MaxChannels,
			"max_users":     limit.MaxUsers,
			"message_limit": limit.MessageLimit,
			"phi_scrubbing": limit.PHIScrubbing,
			"audit_logging": limit.AuditLogging,
		}
	}

	return &rpc.Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]interface{}{
			"tier":      manager.GetTier(),
			"platforms": limits,
		},
	}
}

// HandlePlatformCheck handles platform.check RPC method
func HandlePlatformCheck(manager *Manager, req *rpc.Request) *rpc.Response {
	var params struct {
		Platform string `json:"platform"`
	}

	if len(req.Params) == 0 {
		return &rpc.Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &rpc.ErrorObj{
				Code:    rpc.InvalidParams,
				Message: "platform parameter required",
			},
		}
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return &rpc.Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &rpc.ErrorObj{
				Code:    rpc.InvalidParams,
				Message: err.Error(),
			},
		}
	}

	allowed, err := manager.CanBridgePlatform(params.Platform)
	if err != nil {
		return &rpc.Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &rpc.ErrorObj{
				Code:    rpc.InternalError,
				Message: err.Error(),
			},
		}
	}

	limit, _ := manager.GetPlatformLimit(params.Platform)

	return &rpc.Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]interface{}{
			"platform":      params.Platform,
			"allowed":       allowed,
			"current_tier":  manager.GetTier(),
			"max_channels":  limit.MaxChannels,
			"max_users":     limit.MaxUsers,
			"message_limit": limit.MessageLimit,
		},
	}
}

// ServerInterface defines the interface for RPC server (to avoid import cycle)
type ServerInterface interface {
	RegisterMethod(method string, handler func(*rpc.Request) *rpc.Response)
}
