#!/usr/bin/env python3
"""
P0-CRIT-3: Replace file-based secret injection with socket-based injection.
This script modifies server.go handleStart function.
"""

import re

def modify_server():
    with open('server.go', 'r', encoding='utf-8') as f:
        content = f.read()

    # Lines to delete: 622-670 (file-based secret injection)
    # Keep everything before and after these lines
    lines = content.split('\n')

    # Find the section to replace (after "containerName := fmt.Sprintf..." on line 608)
    # We need to replace lines 622-676 with socket-based code
    # Line 622 starts with "socketPath := filepath.Join..."
    # Line 676 ends with "hostConfig := &container.HostConfig{"

    # Keep lines 1-607, replace 608-775, keep rest
    before = lines[:607]  # 0-indexed, so up to line 608 (index 607)

    # New code for socket-based injection (replaces lines 608-775)
    replacement = [
        "",
        "\t// P0-CRIT-3: Use socket-based secret injection (memory-only, no files)",
        "\tsecretSocketPath, err := s.secretInjector.InjectSecrets(containerName, cred)",
        "\tif err != nil {",
        "\t\treturn &Response{",
        "\t\t\tJSONRPC: \"2.0\",",
        "\t\t\tID:      req.ID,",
        "\t\t\tError: &ErrorObj{",
        "\t\t\t\tCode:    InternalError,",
        "\t\t\t\tMessage: fmt.Sprintf(\"failed to create secret socket: %v\", err),",
        "\t\t\t},",
        "\t\t}",
        "\t}",
        "",
        "\t// Create control socket path for container communication",
        "\tsocketPath := filepath.Join(s.containerDir, containerName+\".sock\")",
        "",
        "\t// 4. Create container config with secret socket mount and proxy support",
        "\t// Check for HTTP_PROXY environment variable for SDTW adapter egress support",
        "\thttpProxy := os.Getenv(\"HTTP_PROXY\")",
        "\tenvVars := []string{",
        "\t\tfmt.Sprintf(\"ARMORCLAW_KEY_ID=%s\", params.KeyID),",
        "\t\tfmt.Sprintf(\"ARMORCLAW_ENDPOINT=%s\", socketPath),",
        "\t\tfmt.Sprintf(\"ARMORCLAW_SECRET_SOCKET=%s\", secretSocketPath), // P0-CRIT-3: Socket path",
        "\t}",
        "",
        "\t// Add HTTP_PROXY to container environment if configured (for SDTW adapter egress)",
        "\tif httpProxy != \"\" {",
        "\t\tenvVars = append(envVars, fmt.Sprintf(\"HTTP_PROXY=%s\", httpProxy))",
        "\t\t// Log proxy configuration",
        "\t\ts.securityLog.LogContainerStart(s.ctx, containerName, \"\", params.Image,",
        "\t\t\tslog.String(\"proxy\", httpProxy),",
        "\t\t)",
        "\t}",
        "",
        "\tcontainerConfig := &container.Config{",
        "\t\tImage: params.Image,",
        "\t\tEnv:   envVars,",
        "\t}",
        "",
        "\t// Mount secret socket into container (read-only, no file exposure)",
        "\thostConfig := &container.HostConfig{",
        "\t\tBinds: []string{",
        "\t\t\tfmt.Sprintf(\"%s:/run/secrets/socket:ro\", secretSocketPath),",
        "\t\t},",
        "\t\tAutoRemove: true, // Auto-remove on exit",
        "\t}",
    ]

    # Keep the rest (lines 776 onwards, after hostConfig)
    # Skip lines 608-775 (old file-based code)
    after = lines[775:]  # Skip the old code section

    # Combine and write
    new_content = '\n'.join(before + replacement + after)

    with open('server.go', 'w', encoding='utf-8', newline='\n') as f:
        f.write(new_content)

    print("P0-CRIT-3: Socket-based injection code applied to server.go")
    print("Modified lines 608-775 replaced with socket-based injection")
    print(f"Original file backed up to server.go.bak")

if __name__ == '__main__':
    modify_server()
