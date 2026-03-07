#!/usr/bin/env python3
import re

# Read the file
with open('browser.go', 'r') as f:
    content = f.read()

# Replace error returns with nil, &ErrorObj{...}
content = re.sub(
    r'return &Response\{\s*JSONRPC: "2\.0",\s*ID: req\.ID,\s*Error: &ErrorObj\{\s*Code: ([^,]+),\s*Message: "([^"]+)",\s*\},\s*\}',
    r'return nil, &ErrorObj{\n\t\t\tCode: \1,\n\t\t\tMessage: "\2",\n\t\t}',
    content
)

# Replace success returns with result, nil
content = re.sub(
    r'return &Response\{\s*JSONRPC: "2\.0",\s*ID: req\.ID,\s*Result: ([^}]+),\s*\}',
    r'return \1, nil',
    content
)

# Write back
with open('browser.go', 'w') as f:
    f.write(content)

print("Updated browser.go return statements")
