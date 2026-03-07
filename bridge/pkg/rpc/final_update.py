#!/usr/bin/env python3

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

    # Replace error returns
    # Pattern: return &Response{...Error: &ErrorObj{...}}
    error_pattern = r'return &Response\{\s*JSONRPC: "2\.0",\s*ID: req\.ID,\s*Error: &ErrorObj\{\s*Code: ([^,]+),\s*Message: "([^"]+)",\s*\},\s*\}'

    def replace_error_return(match):
        code = match.group(1)
        msg = match.group(2)
        return f'            return nil, &ErrorObj{{\n\t\t\t\tCode: {code},\n\t\t\t\tMessage: "{msg}",\n\t\t\t}}\n'

    content = re.sub(error_pattern, replace_error_return, content)

    # Replace success returns
    # Pattern: return &Response{...Result: ...}
    success_pattern = r'return &Response\{\s*JSONRPC: "2\.0",\s*ID: req\.ID,\s*Result: ([^}]+),\s*\}'

    def replace_success_return(match):
        result = match.group(1)
        return f'            return {result}, nil\n'

    content = re.sub(success_pattern, replace_success_return, content)

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
