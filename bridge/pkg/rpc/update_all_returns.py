#!/usr/bin/env python3

import re

def update_return_statements(filename):
    with open(filename, 'r') as f:
        lines = f.readlines()

    output = []
    i = 0
    while i < len(lines):
        line = lines[i]

        # Check if this is a handler function (by checking if it starts with 'func' and has handle)
        is_handler = 'func (s *Server) handle' in line

        if is_handler and not line.strip().startswith('func'):
            # This is inside a function - check for return statements
            if 'return &Response{' in line:
                # Check if it's an error return or success return
                if 'Error: &ErrorObj{' in line:
                    # Extract error code and message
                    match = re.search(r'Code: ([^,]+),\s*Message: "([^"]+)"', line)
                    if match:
                        error_code = match.group(1)
                        error_msg = match.group(2)
                        output.append(f'            return nil, &ErrorObj{{\n\t\t\t\tCode: {error_code},\n\t\t\t\tMessage: "{error_msg}",\n\t\t\t}}\n')
                    else:
                        # Fallback: just replace the whole return statement
                        match = re.search(r'return &Response\{[^}]+\}', line)
                        if match:
                            output.append(line)
                        else:
                            output.append(line)
                    i += 1
                    continue
                elif 'Result:' in line:
                    # Extract result
                    match = re.search(r'Result: ([^}]+)', line)
                    if match:
                        result = match.group(1)
                        output.append(f'            return {result}, nil\n')
                    else:
                        output.append(line)
                    i += 1
                    continue
                else:
                    # Just replace return &Response{ with return
                    if 'return &Response{' in line:
                        # Remove everything from return &Response{ to the closing }
                        match = re.search(r'return &Response\{([^}]*)\}', line)
                        if match:
                            result = match.group(1).strip()
                            if result:
                                output.append(f'            return {result}, nil\n')
                            else:
                                output.append(line)
                        else:
                            output.append(line)
                    else:
                        output.append(line)
                    i += 1
                    continue

        output.append(line)
        i += 1

    with open(filename, 'w') as f:
        f.writelines(output)

    print(f"Updated {filename}")

# Update all files
update_return_statements('bridge_handlers.go')
update_return_statements('browser.go')
update_return_statements('pii.go')
update_return_statements('studio.go')
