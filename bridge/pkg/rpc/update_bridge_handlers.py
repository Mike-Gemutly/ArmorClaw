#!/usr/bin/env python3
import re

# Read the file
with open('bridge_handlers.go', 'r') as f:
    lines = f.readlines()

output = []
i = 0
while i < len(lines):
    line = lines[i]

    # Update handleBridgeStop
    if 'func (s *Server) handleBridgeStop(req *Request) *Response' in line:
        output.append('func (s *Server) handleBridgeStop(ctx context.Context, req *Request) (interface{}, *ErrorObj)\n')
        i += 1
        # Find and replace all returns in this function
        while i < len(lines) and not lines[i].startswith('}'):
            old_return = lines[i]
            # Replace return &Response{...Error:...} with return nil, &ErrorObj{...}
            if 'return &Response{' in old_return and 'Error: &ErrorObj{' in old_return:
                # Extract error code and message
                match = re.search(r'Code: ([^,]+),\s*Message: "([^"]+)"', old_return)
                if match:
                    error_code = match.group(1)
                    error_msg = match.group(2)
                    new_return = f'            return nil, &ErrorObj{{\n\t\t\t\tCode: {error_code},\n\t\t\t\tMessage: "{error_msg}",\n\t\t\t}}\n'
                    output.append(new_return)
                else:
                    output.append(old_return)
            # Replace return &Response{...Result:...} with return result, nil
            elif 'return &Response{' in old_return and 'Result:' in old_return:
                # Extract result (everything after Result:)
                match = re.search(r'Result: ([^}]+)', old_return)
                if match:
                    result = match.group(1)
                    new_return = f'            return {result}, nil\n'
                    output.append(new_return)
                else:
                    output.append(old_return)
            else:
                output.append(old_return)
            i += 1
        continue

    # Update handleBridgeStatus
    elif 'func (s *Server) handleBridgeStatus(req *Request) *Response' in line:
        output.append('func (s *Server) handleBridgeStatus(ctx context.Context, req *Request) (interface{}, *ErrorObj)\n')
        i += 1
        while i < len(lines) and not lines[i].startswith('}'):
            old_return = lines[i]
            match = re.search(r'Result:\s+stats', old_return)
            if match:
                output.append(old_return)
            elif 'return &Response{' in old_return and 'Result:' in old_return:
                match = re.search(r'Result: ([^}]+)', old_return)
                if match:
                    result = match.group(1)
                    new_return = f'            return {result}, nil\n'
                    output.append(new_return)
                else:
                    output.append(old_return)
            else:
                output.append(old_return)
            i += 1
        continue

    # Update handleBridgeChannel
    elif 'func (s *Server) handleBridgeChannel(req *Request) *Response' in line:
        output.append('func (s *Server) handleBridgeChannel(ctx context.Context, req *Request) (interface{}, *ErrorObj)\n')
        i += 1
        while i < len(lines) and not lines[i].startswith('}'):
            old_return = lines[i]
            if 'return &Response{' in old_return and 'Error: &ErrorObj{' in old_return:
                match = re.search(r'Code: ([^,]+),\s*Message: "([^"]+)"', old_return)
                if match:
                    error_code = match.group(1)
                    error_msg = match.group(2)
                    new_return = f'            return nil, &ErrorObj{{\n\t\t\t\tCode: {error_code},\n\t\t\t\tMessage: "{error_msg}",\n\t\t\t}}\n'
                    output.append(new_return)
                else:
                    output.append(old_return)
            elif 'return &Response{' in old_return and 'Result:' in old_return:
                match = re.search(r'Result: ([^}]+)', old_return)
                if match:
                    result = match.group(1)
                    new_return = f'            return {result}, nil\n'
                    output.append(new_return)
                else:
                    output.append(old_return)
            else:
                output.append(old_return)
            i += 1
        continue

    # Update handleUnbridgeChannel
    elif 'func (s *Server) handleUnbridgeChannel(req *Request) *Response' in line:
        output.append('func (s *Server) handleUnbridgeChannel(ctx context.Context, req *Request) (interface{}, *ErrorObj)\n')
        i += 1
        while i < len(lines) and not lines[i].startswith('}'):
            old_return = lines[i]
            if 'return &Response{' in old_return and 'Error: &ErrorObj{' in old_return:
                match = re.search(r'Code: ([^,]+),\s*Message: "([^"]+)"', old_return)
                if match:
                    error_code = match.group(1)
                    error_msg = match.group(2)
                    new_return = f'            return nil, &ErrorObj{{\n\t\t\t\tCode: {error_code},\n\t\t\t\tMessage: "{error_msg}",\n\t\t\t}}\n'
                    output.append(new_return)
                else:
                    output.append(old_return)
            elif 'return &Response{' in old_return and 'Result:' in old_return:
                match = re.search(r'Result: ([^}]+)', old_return)
                if match:
                    result = match.group(1)
                    new_return = f'            return {result}, nil\n'
                    output.append(new_return)
                else:
                    output.append(old_return)
            else:
                output.append(old_return)
            i += 1
        continue

    # Update handleListBridgedChannels
    elif 'func (s *Server) handleListBridgedChannels(req *Request) *Response' in line:
        output.append('func (s *Server) handleListBridgedChannels(ctx context.Context, req *Request) (interface{}, *ErrorObj)\n')
        i += 1
        while i < len(lines) and not lines[i].startswith('}'):
            old_return = lines[i]
            if 'return &Response{' in old_return and 'Result:' in old_return:
                match = re.search(r'Result: ([^}]+)', old_return)
                if match:
                    result = match.group(1)
                    new_return = f'            return {result}, nil\n'
                    output.append(new_return)
                else:
                    output.append(old_return)
            else:
                output.append(old_return)
            i += 1
        continue

    # Update handleGhostUserList
    elif 'func (s *Server) handleGhostUserList(req *Request) *Response' in line:
        output.append('func (s *Server) handleGhostUserList(ctx context.Context, req *Request) (interface{}, *ErrorObj)\n')
        i += 1
        while i < len(lines) and not lines[i].startswith('}'):
            old_return = lines[i]
            if 'return &Response{' in old_return and 'Result:' in old_return:
                match = re.search(r'Result: ([^}]+)', old_return)
                if match:
                    result = match.group(1)
                    new_return = f'            return {result}, nil\n'
                    output.append(new_return)
                else:
                    output.append(old_return)
            else:
                output.append(old_return)
            i += 1
        continue

    # Update handleAppServiceStatus
    elif 'func (s *Server) handleAppServiceStatus(req *Request) *Response' in line:
        output.append('func (s *Server) handleAppServiceStatus(ctx context.Context, req *Request) (interface{}, *ErrorObj)\n')
        i += 1
        while i < len(lines) and not lines[i].startswith('}'):
            old_return = lines[i]
            if 'return &Response{' in old_return and 'Result:' in old_return:
                match = re.search(r'Result: ([^}]+)', old_return)
                if match:
                    result = match.group(1)
                    new_return = f'            return {result}, nil\n'
                    output.append(new_return)
                else:
                    output.append(old_return)
            else:
                output.append(old_return)
            i += 1
        continue

    output.append(line)
    i += 1

with open('bridge_handlers.go', 'w') as f:
    f.writelines(output)

print("Updated bridge_handlers.go")
