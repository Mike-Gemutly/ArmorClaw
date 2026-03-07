#!/usr/bin/env python3

import re

def smart_replace(filename):
    with open(filename, 'r') as f:
        lines = f.readlines()

    output = []
    i = 0

    while i < len(lines):
        line = lines[i]

        # Check if this is a handler function signature
        is_handler = 'func (s *Server) handle' in line and 'req *Request) *Response' in line

        if is_handler:
            # Found a handler - look for return statements inside
            found_handler = True
            i += 1

            while i < len(lines) and found_handler:
                current_line = lines[i]

                # Check for return &Response{ at the start of the line (after indentation)
                if current_line.strip().startswith('return &Response{') or current_line.strip().startswith('\treturn &Response{'):
                    # Extract indentation
                    indent_match = re.match(r'^(\s*)', current_line)
                    indent = indent_match.group(1) if indent_match else ''

                    # Check if it's an error return or success return
                    if 'Error: &ErrorObj{' in current_line:
                        # Extract error code and message
                        match = re.search(r'Code: ([^,]+),\s*Message: "([^"]+)"', current_line)
                        if match:
                            code = match.group(1)
                            msg = match.group(2)
                            new_line = indent + 'return nil, &ErrorObj{\n'
                            new_line += indent + '\t\t\t\tCode: ' + code + ',\n'
                            new_line += indent + '\t\t\t\tMessage: "' + msg + '",\n'
                            new_line += indent + '\t\t\t}\n'
                            output.append(new_line)
                        else:
                            # Fallback: just remove the return &Response{...}
                            match = re.search(r'(return &Response\{[^}]+\})', current_line)
                            if match:
                                # Remove everything from return &Response{ to the closing }
                                text = match.group(1)
                                # Try to extract result
                                result_match = re.search(r'Result: ([^}]+)', text)
                                if result_match:
                                    result = result_match.group(1).strip()
                                    if result:
                                        new_line = indent + 'return ' + result + ', nil\n'
                                    else:
                                        new_line = ''
                                else:
                                    # Extract error code and message
                                    error_match = re.search(r'Code: ([^,]+),\s*Message: "([^"]+)"', text)
                                    if error_match:
                                        code = error_match.group(1)
                                        msg = error_match.group(2)
                                        new_line = indent + 'return nil, &ErrorObj{\n'
                                        new_line += indent + '\t\t\t\tCode: ' + code + ',\n'
                                        new_line += indent + '\t\t\t\tMessage: "' + msg + '",\n'
                                        new_line += indent + '\t\t\t}\n'
                                    else:
                                        new_line = ''
                                output.append(new_line)
                        i += 1
                        continue
                    elif 'Result:' in current_line:
                        # Extract result
                        match = re.search(r'Result: ([^}]+)', current_line)
                        if match:
                            result = match.group(1).strip()
                            new_line = indent + 'return ' + result + ', nil\n'
                            output.append(new_line)
                        else:
                            # Try to find the closing }
                            closing_idx = current_line.find('}')
                            if closing_idx != -1:
                                result = current_line[8:closing_idx].strip()
                                new_line = indent + 'return ' + result + ', nil\n'
                                output.append(new_line)
                            else:
                                output.append(current_line)
                        i += 1
                        continue
                    else:
                        # Just remove return &Response{
                        closing_idx = current_line.find('}')
                        if closing_idx != -1:
                            result = current_line[8:closing_idx].strip()
                            new_line = indent + 'return ' + result + ', nil\n'
                            output.append(new_line)
                        else:
                            output.append(current_line)
                        i += 1
                        continue
                else:
                    output.append(current_line)
                    i += 1
                    # Check for end of function
                    if current_line.strip() == '}' or (current_line.strip().startswith('}') and current_line[0] not in '\t '):
                        found_handler = False
                        break
                break

        else:
            output.append(line)
            i += 1

    with open(filename, 'w') as f:
        f.writelines(output)

    print(f"Updated {filename}")

# Update all files
smart_replace('bridge_handlers.go')
smart_replace('browser.go')
smart_replace('pii.go')
smart_replace('studio.go')
