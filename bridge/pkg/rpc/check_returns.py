#!/usr/bin/env python3

import re

files = ['bridge_handlers.go', 'browser.go', 'pii.go', 'studio.go']

for filename in files:
    print(f"\n{filename}:")
    with open(filename, 'r') as f:
        lines = f.readlines()

    error_returns = 0
    success_returns = 0
    old_returns = 0

    for line in lines:
        if 'return &Response{' in line:
            if 'Error: &ErrorObj{' in line:
                error_returns += 1
            elif 'Result:' in line:
                success_returns += 1
            else:
                old_returns += 1

    print(f"  Error returns: {error_returns}")
    print(f"  Success returns: {success_returns}")
    print(f"  Old returns (no Result/Error): {old_returns}")
    print(f"  Total Response returns: {error_returns + success_returns + old_returns}")
