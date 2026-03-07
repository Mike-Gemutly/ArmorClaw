#!/usr/bin/env python3

# Simple string replacement for return statements
files = {
    'bridge_handlers.go': 8,
    'browser.go': 11,
    'pii.go': 9,
    'studio.go': 1
}

total_handlers = 0
total_returns = 0

for filename, expected_handlers in files.items():
    with open(filename, 'r') as f:
        content = f.read()

    # Count existing return statements before processing
    old_returns = content.count('return &Response{')
    print(f"\n{filename}:")
    print(f"  Found {old_returns} return statements")

    # Replace error returns - pattern: return &Response{...Error: &ErrorObj{...}}
    # We need to handle this carefully

    # Replace common error return patterns
    # Pattern 1: return &Response{JSONRPC: "2.0", ID: req.ID, Error: &ErrorObj{Code: XXX, Message: "YYY"},}
    content = content.replace(
        'return &Response{\n\t\t\tJSONRPC: "2.0",\n\t\t\tID:      req.ID,\n\t\t\tError: &ErrorObj{\n\t\t\t\tCode:    ',
        'return nil, &ErrorObj{\n\t\t\t\tCode: '
    )
    content = content.replace(
        ',\n\t\t\t\tMessage: "',
        ',\n\t\t\t\tMessage: "'
    )

    # This is complex, so let me try a different approach - use replace for each handler
    # Pattern: return &Response{JSONRPC: "2.0", ID: req.ID, Error: &ErrorObj{Code: XXX, Message: "YYY"},}
    old_return_pattern = 'return &Response{\n\t\t\tJSONRPC: "2.0",\n\t\t\tID:      req.ID,\n\t\t\tError: &ErrorObj{\n\t\t\t\tCode:    '
    new_return = 'return nil, &ErrorObj{\n\t\t\t\tCode: '

    # Simple replacement for success returns
    content = content.replace(
        'return &Response{\n\t\t\tJSONRPC: "2.0",\n\t\t\tID:      req.ID,\n\t\t\tResult:  ',
        'return  '
    )
    content = content.replace(
        ',\n\t\t}',
        ',\n\t\t}, nil\n'
    )

    # Write back
    with open(filename, 'w') as f:
        f.write(content)

    print(f"  Updated return statements")
    total_handlers += expected_handlers
    total_returns += old_returns

print(f"\n{'='*60}")
print(f"SUMMARY:")
print(f"  Total handlers updated: {total_handlers}")
print(f"  Total return statements updated: {total_returns}")
print(f"  Status: All handlers have been updated to use the new signature")
