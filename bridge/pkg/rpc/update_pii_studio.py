#!/usr/bin/env python3
import re

def update_pii(filepath):
    with open(filepath, 'r') as f:
        lines = f.readlines()

    output = []
    i = 0

    # Define all pii handlers
    handlers = [
        'handlePIIRequest',
        'handlePIIApprove',
        'handlePIIDeny',
        'handlePIIStatus',
        'handlePIIListPending',
        'handlePIIStats',
        'handlePIICancel',
        'handlePIIFulfill',
        'handlePIIWaitForApproval'
    ]

    current_handler = None
    in_function = False

    while i < len(lines):
        line = lines[i]

        # Check if this is a handler function
        is_handler = any(f'func (s *Server) {h}(req *Request) *Response' in line for h in handlers)

        if is_handler and not in_function:
            # Update signature
            for h in handlers:
                if f'func (s *Server) {h}(req *Request) *Response' in line:
                    output.append(f'func (s *Server) {h}(ctx context.Context, req *Request) (interface{}, *ErrorObj)\n')
                    current_handler = h
                    in_function = True
                    break
            i += 1
            continue

        # Process the function body
        if in_function:
            # Check for return statements
            if 'return &Response{' in line:
                # Extract the return statement
                match = re.search(r'Result: ([^}]+)', line)
                if match:
                    result = match.group(1)
                    output.append(f'            return {result}, nil\n')
                else:
                    output.append(line)
            elif 'Error: &ErrorObj{' in line:
                match = re.search(r'Code: ([^,]+),\s*Message: "([^"]+)"', line)
                if match:
                    error_code = match.group(1)
                    error_msg = match.group(2)
                    output.append(f'            return nil, &ErrorObj{{\n\t\t\t\tCode: {error_code},\n\t\t\t\tMessage: "{error_msg}",\n\t\t\t}}\n')
                else:
                    output.append(line)
            else:
                output.append(line)
            i += 1

            # Check if we're at the end of the function
            if line.strip() == '}' and current_handler:
                in_function = False
                current_handler = None
                output.append(line)
                i += 1
                continue
        else:
            output.append(line)
            i += 1

    with open(filepath, 'w') as f:
        f.writelines(output)
    print(f"Updated {filepath}")

def update_studio(filepath):
    with open(filepath, 'r') as f:
        lines = f.readlines()

    output = []
    in_function = False

    for i, line in enumerate(lines):
        # Check for handleStudio function
        if 'func (s *Server) handleStudio(req *Request) *Response' in line:
            output.append('func (s *Server) handleStudio(ctx context.Context, req *Request) (interface{}, *ErrorObj)\n')
            in_function = True
            continue

        if in_function:
            # Process return statements
            if 'return &Response{' in line:
                match = re.search(r'Error: &ErrorObj\{\s*Code: ([^,]+),\s*Message: "([^"]+)"', line)
                if match:
                    error_code = match.group(1)
                    error_msg = match.group(2)
                    output.append(f'            return nil, &ErrorObj{{\n\t\t\t\tCode: {error_code},\n\t\t\t\tMessage: "{error_msg}",\n\t\t\t}}\n')
                else:
                    match = re.search(r'Result: ([^}]+)', line)
                    if match:
                        result = match.group(1)
                        output.append(f'            return {result}, nil\n')
                    else:
                        output.append(line)
            else:
                output.append(line)

            # Check for end of function
            if line.strip() == '}':
                in_function = False
                output.append(line)
        else:
            output.append(line)

    with open(filepath, 'w') as f:
        f.writelines(output)
    print(f"Updated {filepath}")

# Update files
update_pii('pii.go')
update_studio('studio.go')
