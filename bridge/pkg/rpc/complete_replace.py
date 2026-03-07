#!/usr/bin/env python3

import re

def complete_replace(filename):
    with open(filename, 'r') as f:
        content = f.read()

    # Pattern to match complete return &Response{...} statements
    # We'll use a regex that matches the entire block
    pattern = r'(return\s+&Response\{[^}]*\})'

    def replace_complete(match):
        original = match.group(1)
        # Check if it's an error return
        if 'Error: &ErrorObj{' in original:
            # Extract error code and message
            match_err = re.search(r'Code:\s*([^\n,]+),\s*Message:\s*"([^"]+)"', original)
            if match_err:
                code = match_err.group(1).strip()
                msg = match_err.group(2).strip()
                return f'return nil, &ErrorObj{{\n\t\t\t\tCode: {code},\n\t\t\t\tMessage: "{msg}",\n\t\t\t}}'
        # Check if it's a success return
        elif 'Result:' in original:
            # Extract result
            match_res = re.search(r'Result:\s*([^\n,]+)', original)
            if match_res:
                result = match_res.group(1).strip()
                return f'return {result}, nil'
        # Fallback: just remove the return statement
        return ''

    content = re.sub(pattern, replace_complete, content)

    with open(filename, 'w') as f:
        f.write(content)

    print(f"Updated {filename}")

# Update all files
complete_replace('bridge_handlers.go')
complete_replace('browser.go')
complete_replace('pii.go')
complete_replace('studio.go')
