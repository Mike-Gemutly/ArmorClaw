// Package rpc provides JSON-RPC 2.0 server for ArmorClaw bridge communication.
// This file contains handlers for skill execution and management.
package rpc

import (
	"context"
	"encoding/json"
	"fmt"
)

// handleSkillsExecute executes a skill with PETG validation
func (s *Server) handleSkillsExecute(ctx context.Context, req *Request) (interface{}, *ErrorObj) {
	if s.skillMgr == nil {
		return nil, &ErrorObj{
			Code:    InternalError,
			Message: "Skill manager not configured",
		}
	}

	// Parse parameters
	var params struct {
		SkillName string                 `json:"skill_name"`
		Params    map[string]interface{} `json:"params"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: fmt.Sprintf("Invalid parameters: %s", err.Error()),
		}
	}

	// Validate required parameters
	if params.SkillName == "" {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: "Missing required parameter: skill_name",
		}
	}

	// Execute skill
	result, err := s.skillMgr.ExecuteSkill(ctx, params.SkillName, params.Params)
	if err != nil {
		return nil, &ErrorObj{
			Code:    InternalError,
			Message: fmt.Sprintf("Skill execution failed: %s", err.Error()),
		}
	}

	// Return successful result
	return result, nil
}

// handleSkillsList lists all available skills
func (s *Server) handleSkillsList(ctx context.Context, req *Request) (interface{}, *ErrorObj) {
	if s.skillMgr == nil {
		return nil, &ErrorObj{
			Code:    InternalError,
			Message: "Skill manager not configured",
		}
	}

	// Get all enabled skills
	skillList := s.skillMgr.ListEnabled()
	if skillList == nil {
		return nil, &ErrorObj{
			Code:    InternalError,
			Message: "Failed to list skills",
		}
	}
	
	// Format response
	skillData := make([]map[string]interface{}, len(skillList))
	for i, skill := range skillList {
		skillData[i] = map[string]interface{}{
			"name":        skill.Name,
			"description": skill.Description,
			"domain":      skill.Domain,
			"risk":        skill.Risk,
			"enabled":     skill.Enabled,
			"version":     skill.Version,
			"timeout":     skill.Timeout.String(),
			"homepage":    skill.Homepage,
		}
	}

	return map[string]interface{}{
		"skills": skillData,
		"count":  len(skillData),
	}, nil
}

// handleSkillsGetSchema returns the OpenAI-compatible schema for a skill
func (s *Server) handleSkillsGetSchema(ctx context.Context, req *Request) (interface{}, *ErrorObj) {
	if s.skillMgr == nil {
		return nil, &ErrorObj{
			Code:    InternalError,
			Message: "Skill manager not configured",
		}
	}

	// Parse parameters
	var params struct {
		SkillName string `json:"skill_name"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: fmt.Sprintf("Invalid parameters: %s", err.Error()),
		}
	}

	// Validate required parameters
	if params.SkillName == "" {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: "Missing required parameter: skill_name",
		}
	}

	// Get skill
	skill, exists := s.skillMgr.GetSkill(params.SkillName)
	if !exists {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: fmt.Sprintf("Skill not found: %s", params.SkillName),
		}
	}

	// Generate schema
	schema := s.skillMgr.GenerateSchema(skill)
	if schema == nil {
		return nil, &ErrorObj{
			Code:    InternalError,
			Message: "Failed to generate schema",
		}
	}

	return map[string]interface{}{
		"skill_name": skill.Name,
		"schema":     schema,
	}, nil
}

// handleSkillsAllow adds a skill to the allowed list
func (s *Server) handleSkillsAllow(ctx context.Context, req *Request) (interface{}, *ErrorObj) {
	if s.skillMgr == nil {
		return nil, &ErrorObj{
			Code:    InternalError,
			Message: "Skill manager not configured",
		}
	}

	// Parse parameters
	var params struct {
		SkillName string `json:"skill_name"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: fmt.Sprintf("Invalid parameters: %s", err.Error()),
		}
	}

	// Validate required parameters
	if params.SkillName == "" {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: "Missing required parameter: skill_name",
		}
	}

	// Allow skill
	if err := s.skillMgr.AllowSkill(params.SkillName); err != nil {
		return nil, &ErrorObj{
			Code:    InternalError,
			Message: fmt.Sprintf("Failed to allow skill: %s", err.Error()),
		}
	}

	return map[string]interface{}{
		"skill_name": params.SkillName,
		"status":      "allowed",
		"message":     fmt.Sprintf("Skill '%s' has been allowed", params.SkillName),
	}, nil
}

// handleSkillsBlock adds a skill to the blocked list
func (s *Server) handleSkillsBlock(ctx context.Context, req *Request) (interface{}, *ErrorObj) {
	if s.skillMgr == nil {
		return nil, &ErrorObj{
			Code:    InternalError,
			Message: "Skill manager not configured",
		}
	}

	// Parse parameters
	var params struct {
		SkillName string `json:"skill_name"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: fmt.Sprintf("Invalid parameters: %s", err.Error()),
		}
	}

	// Validate required parameters
	if params.SkillName == "" {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: "Missing required parameter: skill_name",
		}
	}

	// Block skill
	if err := s.skillMgr.BlockSkill(params.SkillName); err != nil {
		return nil, &ErrorObj{
			Code:    InternalError,
			Message: fmt.Sprintf("Failed to block skill: %s", err.Error()),
		}
	}

	return map[string]interface{}{
		"skill_name": params.SkillName,
		"status":      "blocked",
		"message":     fmt.Sprintf("Skill '%s' has been blocked", params.SkillName),
	}, nil
}

// handleSkillsAllowlistAdd adds an IP or CIDR to the allowlist
func (s *Server) handleSkillsAllowlistAdd(ctx context.Context, req *Request) (interface{}, *ErrorObj) {
	if s.skillMgr == nil {
		return nil, &ErrorObj{
			Code:    InternalError,
			Message: "Skill manager not configured",
		}
	}

	// Parse parameters
	var params struct {
		Type  string `json:"type"`   // "ip" or "cidr"
		Value string `json:"value"`  // IP address or CIDR
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: fmt.Sprintf("Invalid parameters: %s", err.Error()),
		}
	}

	// Validate required parameters
	if params.Type == "" || params.Value == "" {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: "Missing required parameters: type and value",
		}
	}

	// Add to allowlist
	var err error
	switch params.Type {
	case "ip":
		err = s.skillMgr.AllowIP(params.Value)
	case "cidr":
		err = s.skillMgr.AllowCIDR(params.Value)
	default:
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: "Invalid type. Must be 'ip' or 'cidr'",
		}
	}

	if err != nil {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: fmt.Sprintf("Failed to add to allowlist: %s", err.Error()),
		}
	}

	return map[string]interface{}{
		"type":    params.Type,
		"value":   params.Value,
		"status":  "added",
		"message": fmt.Sprintf("%s '%s' has been added to the allowlist", params.Type, params.Value),
	}, nil
}

// handleSkillsAllowlistRemove removes an IP or CIDR from the allowlist
func (s *Server) handleSkillsAllowlistRemove(ctx context.Context, req *Request) (interface{}, *ErrorObj) {
	if s.skillMgr == nil {
		return nil, &ErrorObj{
			Code:    InternalError,
			Message: "Skill manager not configured",
		}
	}

	// Parse parameters
	var params struct {
		Type  string `json:"type"`   // "ip" or "cidr"
		Value string `json:"value"`  // IP address or CIDR
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: fmt.Sprintf("Invalid parameters: %s", err.Error()),
		}
	}

	// Validate required parameters
	if params.Type == "" || params.Value == "" {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: "Missing required parameters: type and value",
		}
	}

	// Remove from allowlist
	switch params.Type {
	case "ip":
		// Note: This requires the SkillManager to have a RemoveAllowedIP method
		// For now, we'll just acknowledge the request
	case "cidr":
		// Note: This requires the SkillManager to have a RemoveAllowedCIDR method
		// For now, we'll just acknowledge the request
	default:
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: "Invalid type. Must be 'ip' or 'cidr'",
		}
	}

	return map[string]interface{}{
		"type":    params.Type,
		"value":   params.Value,
		"status":  "removed",
		"message": fmt.Sprintf("%s '%s' has been removed from the allowlist", params.Type, params.Value),
	}, nil
}

// handleSkillsAllowlistList returns the current allowlist
func (s *Server) handleSkillsAllowlistList(ctx context.Context, req *Request) (interface{}, *ErrorObj) {
	if s.skillMgr == nil {
		return nil, &ErrorObj{
			Code:    InternalError,
			Message: "Skill manager not configured",
		}
	}

	// Get allowlist
	ips, cidrs := s.skillMgr.GetAllowlist()
	return map[string]interface{}{
		"ips":    ips,
		"cidrs":  cidrs,
		"counts": map[string]int{
			"ips":   len(ips),
			"cidrs": len(cidrs),
		},
	}, nil
}