#!/usr/bin/env python3

files = {
    'bridge_handlers.go': 8,
    'browser.go': 11,
    'pii.go': 9,
    'studio.go': 1
}

total = 0
for filename, expected_count in files.items():
    with open(filename, 'r') as f:
        content = f.read()

    # Count handlers with new signature
    count = content.count('func (s *Server) handle.*ctx context.Context')
    print(f"{filename}: {count} handlers updated (expected {expected_count})")
    total += count

print(f"\nTotal: {total} handlers updated (expected {sum(files.values())})")
