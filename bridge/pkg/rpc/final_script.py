#!/usr/bin/env python3
"""
Comprehensive handler update script.
"""

import re

def update_handler_file(filepath):
    with open(filepath, 'r') as f:
        content = f.read()

    # Detect all handlers in the file
    handler_pattern = r'func \(s \*Server\) (handle\w+)\(req \*Request\) \*Response'
    matches = re.findall(handler_pattern, content)

    print(f"\n{filepath}: Found {len(matches)} handlers")
    handlers = list(matches)

    # Create a map of old to new signatures
    signature_replacements = {}
    for handler in handlers:
        old_sig = 'func (s *Server) ' + handler + '(req *Request) *Response'
        new_sig = 'func (s *Server) ' + handler + '(ctx context.Context, req *Request) (interface{}, *ErrorObj)'
        signature_replacements[old_sig] = new_sig

    # Apply signature replacements
    for old_sig, new_sig in signature_replacements.items():
        content = content.replace(old_sig, new_sig)

    # Now replace return statements line by line
    lines = content.split('\n')
    new_lines = []
    i = 0

    while i < len(lines):
        line = lines[i]

        stripped = line.lstrip()
        if stripped.startswith('return &Response{'):
            indent = line[:len(line) - len(stripped)]

            # Check if it's an error return
            if 'Error: &ErrorObj{' in line:
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
                result_start = line.find('Result:')
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
