#!/usr/bin/env python3

import re

def robust_replace(filename):
    with open(filename, 'r') as f:
        lines = f.readlines()

    output = []
    i = 0

    while i < len(lines):
        line = lines[i]

        # Check if this is a handler function signature
        if 'func (s *Server) handle' in line and 'req *Request) *Response' in line:
            # Found a handler - look for return statements inside
            found_handler = True
            i += 1

            while i < len(lines) and found_handler:
                current_line = lines[i]

                # Check for return &Response{ at the start of the line
                if current_line.strip().startswith('return &Response{'):
                    # Extract indentation
                    indent = current_line[:len(current_line) - len(current_line.lstrip())]

                    # Check if it's an error return
                    if 'Error: &ErrorObj{' in current_line:
                        # Extract code and message from this line
                        match = re.search(r'Code: ([^,]+),\s*Message: "([^"]+)"', current_line)
                        if match:
                            code = match.group(1)
                            msg = match.group(2)
                            new_return = indent + 'return nil, &ErrorObj{\n'
                            new_return += indent + '\t\t\t\tCode: ' + code + ',\n'
                            new_return += indent + '\t\t\t\tMessage: "' + msg + '",\n'
                            new_return += indent + '\t\t\t}\n'
                            output.append(new_return)
                        else:
                            # Fallback
                            new_return = indent + 'return nil, &ErrorObj{\n'
                            new_return += indent + '\t\t\t\tCode: InternalError,\n'
                            new_return += indent + '\t\t\t\tMessage: "error",\n'
                            new_return += indent + '\t\t\t}\n'
                            output.append(new_return)
                        i += 1
                        continue

                    # Check if it's a success return
                    elif 'Result:' in current_line:
                        # Find where Result: is
                        result_start = current_line.find('Result:')
                        result_end = current_line.find('}', result_start)
                        if result_end != -1:
                            result_text = current_line[result_start+7:result_end].strip()
                            # Extract the Result line
                            new_return = indent + 'return ' + result_text + ', nil\n'
                            output.append(new_return)
                        else:
                            output.append(current_line)
                        i += 1
                        continue

                    # Fallback: just remove the return &Response{...}
                    else:
                        # Find the closing }
                        closing_idx = current_line.find('}')
                        if closing_idx != -1:
                            result_text = current_line[8:closing_idx].strip()
                            if result_text:
                                new_return = indent + 'return ' + result_text + ', nil\n'
                            else:
                                new_return = ''
                            output.append(new_return)
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

        else:
            output.append(line)
            i += 1

    with open(filename, 'w') as f:
        f.writelines(output)

    print(f"Updated {filename}")

# Update all files
robust_replace('bridge_handlers.go')
robust_replace('browser.go')
robust_replace('pii.go')
robust_replace('studio.go')
