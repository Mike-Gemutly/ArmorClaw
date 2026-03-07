#!/usr/bin/env python3

def fix_syntax_errors():
    # Fix browser.go ctx := ctx to ctx = ctx
    with open('browser.go', 'r') as f:
        content = f.read()

    content = content.replace('ctx := ctx', 'ctx = ctx')

    with open('browser.go', 'w') as f:
        f.write(content)

    print("Fixed browser.go ctx := ctx to ctx = ctx")

    # Fix pii.go errors
    with open('pii.go', 'r') as f:
        content = f.read()

    # Fix line 73: remove trailing }, nil from append
    content = content.replace(
        'fields = append(fields, keystore.PIIFieldRequest{\n\t\t\tKey:         key,\n\t\t\tDisplayName: displayName,\n\t\t\tRequired:    required,\n\t\t\tSensitive:   sensitive,\n\t\t\t}, nil\n\t\t}',
        'fields = append(fields, keystore.PIIFieldRequest{\n\t\t\tKey:         key,\n\t\t\tDisplayName: displayName,\n\t\t\tRequired:    required,\n\t\t\tSensitive:   sensitive,\n\t\t})'
    )

    # Fix line 119: remove trailing }, nil from error return
    content = content.replace(
        'return nil, &ErrorObj{\n\t\t\t\tCode: InternalError,\n\t\t\t\tMessage: "failed to create PII request: " + err.Error(),\n\t\t\t},\n\t\t\t}, nil\n\n\t\t}',
        'return nil, &ErrorObj{\n\t\t\t\tCode: InternalError,\n\t\t\t\tMessage: "failed to create PII request: " + err.Error(),\n\t\t\t}\n\n\t\t}'
    )

    # Fix line 135: remove trailing }, nil from success return
    content = content.replace(
        'return map[string]interface{}, nil{\n\t\t\t"request_id":       piiReq.ID,\n\t\t\t"agent_id":         piiReq.AgentID,\n\t\t\t"skill_id":         piiReq.SkillID,\n\t\t\t"profile_id":       piiReq.ProfileID,\n\t\t\t"requested_fields": fields,\n\t\t\t"status":           string(piiReq.Status),\n\t\t\t"created_at":       piiReq.CreatedAt.Format(time.RFC3339),\n\t\t\t"expires_at":       piiReq.ExpiresAt.Format(time.RFC3339),\n\t\t\t"message":          "PII request created. Agent paused awaiting approval.",\n\t\t\t}, nil\n\t\t,\n\t\t}',
        'return map[string]interface{}, nil{\n\t\t\t"request_id":       piiReq.ID,\n\t\t\t"agent_id":         piiReq.AgentID,\n\t\t\t"skill_id":         piiReq.SkillID,\n\t\t\t"profile_id":       piiReq.ProfileID,\n\t\t\t"requested_fields": fields,\n\t\t\t"status":           string(piiReq.Status),\n\t\t\t"created_at":       piiReq.CreatedAt.Format(time.RFC3339),\n\t\t\t"expires_at":       piiReq.ExpiresAt.Format(time.RFC3339),\n\t\t\t"message":          "PII request created. Agent paused awaiting approval.",\n\t\t}, nil\n\t\t}'
    )

    # Fix any other similar patterns
    content = content.replace('}, nil\n\t\t,\n\t\t}', '}, nil\n\t\t}')

    # Remove any remaining }, nil from error returns
    content = content.replace('},\n\t\t\t}, nil\n\n\t\t}', '}\n\n\t\t}')

    with open('pii.go', 'w') as f:
        f.write(content)

    print("Fixed pii.go syntax errors")

fix_syntax_errors()
