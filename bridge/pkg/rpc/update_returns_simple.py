#!/usr/bin/env python3

import re

# Update pii.go return statements
with open('pii.go', 'r') as f:
    lines = f.readlines()

output = []
i = 0
while i < len(lines):
    line = lines[i]

    # Replace error returns
    if 'return &Response{' in line and 'Error: &ErrorObj{' in line:
        match = re.search(r'Code: ([^,]+),\s*Message: "([^"]+)"', line)
        if match:
            error_code = match.group(1)
            error_msg = match.group(2)
            output.append(f'            return nil, &ErrorObj{{\n\t\t\t\tCode: {error_code},\n\t\t\t\tMessage: "{error_msg}",\n\t\t\t}}\n')
            i += 1
            continue

    # Replace success returns
    elif 'return &Response{' in line and 'Result:' in line:
        match = re.search(r'Result: ([^}]+)', line)
        if match:
            result = match.group(1)
            output.append(f'            return {result}, nil\n')
            i += 1
            continue

    output.append(line)
    i += 1

with open('pii.go', 'w') as f:
    f.writelines(output)

print("Updated pii.go return statements")

# Update studio.go return statements
with open('studio.go', 'r') as f:
    lines = f.readlines()

output = []
for line in lines:
    if 'return &Response{' in line and 'Error: &ErrorObj{' in line:
        match = re.search(r'Code: ([^,]+),\s*Message: "([^"]+)"', line)
        if match:
            error_code = match.group(1)
            error_msg = match.group(2)
            output.append(f'            return nil, &ErrorObj{{\n\t\t\t\tCode: {error_code},\n\t\t\t\tMessage: "{error_msg}",\n\t\t\t}}\n')
            continue

    elif 'return &Response{' in line and 'Result:' in line:
        match = re.search(r'Result: ([^}]+)', line)
        if match:
            result = match.group(1)
            output.append(f'            return {result}, nil\n')
            continue

    output.append(line)

with open('studio.go', 'w') as f:
    f.writelines(output)

print("Updated studio.go return statements")
