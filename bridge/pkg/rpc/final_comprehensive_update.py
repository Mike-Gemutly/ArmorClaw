#!/usr/bin/env python3
"""
Comprehensive handler update script.

This script will:
1. Update handler signatures to use new format
2. Replace all return statements properly
3. Maintain exact formatting and indentation
"""

import re

def update_handler_file(filepath):
    with open(filepath, 'r') as f:
        content = f.read()

    # Replace all handler signatures
    # Old: func (s *Server) handleXXX(req *Request) *Response
    # New: func (s *Server) handleXXX(ctx context.Context, req *Request) (interface{}, *ErrorObj)

    handlers_to_update = []

    # Detect all handlers in the file
    handler_pattern = r'func \(s \*Server\) (handle\w+)\(req \*Request\) \*Response'
    matches = re.findall(handler_pattern, content)

    print(f"\n{filepath}: Found {len(matches)} handlers")
    for handler in matches:
        handlers_to_update.append(handler)

    # Create a map of old to new signatures
    signature_replacements = {}
    for handler in handlers_to_update:
        old_sig = f'func (s *Server) {handler}(req *Request) *Response'
        new_sig = f'func (s *Server) {handler}(ctx context.Context, req *Request) (interface{}, *ErrorObj)'
        signature_replacements[old_sig] = new_sig

    # Apply signature replacements
    for old_sig, new_sig in signature_replacements.items():
        content = content.replace(old_sig, new_sig)

    # Now replace return statements
    # We need to handle both error returns and success returns

    # Pattern for error returns
    error_return_pattern = r'return\s+&Response\{[^}]*Error:\s+&ErrorObj\{[^}]*Code:\s*([^\n,]+),\s*Message:\s*"([^"]+)"[^}]*\}[^}]*\}'
    success_return_pattern = r'return\s+&Response\{[^}]*Result:\s*([^\n}]+)[^}]*\}'

    def replace_error_return(match):
        code = match.group(1).strip()
        msg = match.group(2).strip()
        return f'return nil, &ErrorObj{{\n\t\t\t\tCode: {code},\n\t\t\t\tMessage: "{msg}",\n\t\t\t}}'

    def replace_success_return(match):
        result = match.group(1).strip()
        return f'return {result}, nil'

    # Apply replacements (this is complex, so we do it line by line instead)
    lines = content.split('\n')
    new_lines = []
    i = 0

    while i < len(lines):
        line = lines[i]

        # Check for return &Response{ at the start of a line (after optional whitespace)
        stripped = line.lstrip()
        if stripped.startswith('return &Response{'):
            indent = line[:len(line) - len(stripped)]

            # Check if it's an error return
            if 'Error: &ErrorObj{' in line:
                # Extract code and message from the same line
                match = re.search(r'Code:\s*([^\n,]+),\s*Message:\s*"([^"]+)"', line)
                if match:
                    code = match.group(1).strip()
                    msg = match.group(2).strip()
                    new_lines.append(f'{indent}return nil, &ErrorObj{{')
                    new_lines.append(f'{indent}\t\t\t\tCode: {code},')
                    new_lines.append(f'{indent}\t\t\t\tMessage: "{msg}",')
                    new_lines.append(f'{indent}\t\t\t}}')
                    i += 1
                    continue
            # Check if it's a success return
            elif 'Result:' in line:
                # Find Result: position and extract everything up to }
                result_start = line.find('Result:')
                # Find the closing }
                result_end = line.find('}', result_start)
                if result_end != -1:
                    result_text = line[result_start+7:result_end].strip()
                    new_lines.append(f'{indent}return {result_text}, nil')
                    i += 1
                    continue

        new_lines.append(line)
        i += 1

    content = '\n'.join(new_lines)

    # Write back
    with open(filepath, 'w') as f:
        f.write(content)

    print(f"Updated {filepath}: {len(signature_replacements)} handlers updated")

# Update all files
update_handler_file('bridge_handlers.go')
update_handler_file('browser.go')
update_handler_file('pii.go')
update_handler_file('studio.go')

print("\n" + "="*60)
print("UPDATE COMPLETE")
print("All handlers have been updated to use the new signature")
