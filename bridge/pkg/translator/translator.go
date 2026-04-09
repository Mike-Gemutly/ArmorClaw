// Package translator provides protocol translation between RPC and MCP.
package translator

import (
	"encoding/json"
)

// RPCToMCPTranslator translates between RPC protocol and MCP protocol.
type RPCToMCPTranslator struct{}

// NewRPCToMCPTranslator creates a new RPCToMCPTranslator.
func NewRPCToMCPTranslator() *RPCToMCPTranslator {
	return &RPCToMCPTranslator{}
}

// ToMCP extracts skill name and marshaled arguments from RPC skills.execute params.
// Input JSON: {"skill_name": "...", "params": {...}}
// Returns: skill name, arguments as json.RawMessage, error.
// Pure mapping — no business logic, no logging, no fmt.Sprintf.
func (t *RPCToMCPTranslator) ToMCP(rpcParams json.RawMessage) (string, json.RawMessage, error) {
	var params struct {
		SkillName string                 `json:"skill_name"`
		Params    map[string]interface{} `json:"params"`
	}
	if err := json.Unmarshal(rpcParams, &params); err != nil {
		return "", nil, err
	}
	args, err := json.Marshal(params.Params)
	if err != nil {
		return "", nil, err
	}
	return params.SkillName, json.RawMessage(args), nil
}

// FromMCP converts an MCP tools/call response result to an RPC-compatible result.
// Pure pass-through: returns result as-is, maps MCP error to (code, message).
// Returns: (result interface{}, errorExists bool, errorCode int, errorMessage string).
func (t *RPCToMCPTranslator) FromMCP(result interface{}, errCode int, errMsg string) (interface{}, bool, int, string) {
	if errCode != 0 {
		return nil, true, errCode, errMsg
	}
	return result, false, 0, ""
}
