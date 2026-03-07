#!/usr/bin/env python3
import re

def update_handlers(filepath):
    with open(filepath, 'r') as f:
        content = f.read()

    # List of handlers to update
    handlers = [
        'handleBridgeStop',
        'handleBridgeStatus',
        'handleBridgeChannel',
        'handleUnbridgeChannel',
        'handleListBridgedChannels',
        'handleGhostUserList',
        'handleAppServiceStatus'
    ]

    for handler in handlers:
        # Update signature
        old_sig = f'func (s *Server) {handler}(req *Request) *Response'
        new_sig = f'func (s *Server) {handler}(ctx context.Context, req *Request) (interface{}, *ErrorObj)'
        content = content.replace(old_sig, new_sig)

    # Replace all error returns
    error_pattern = r'return &Response\{\s*JSONRPC: "2\.0",\s*ID: req\.ID,\s*Error: &ErrorObj\{\s*Code: ([^,]+),\s*Message: "([^"]+)",\s*\},\s*\}'
    content = re.sub(error_pattern, r'return nil, &ErrorObj{\n\t\t\tCode: \1,\n\t\t\tMessage: "\2",\n\t\t}', content)

    # Replace success returns
    success_pattern = r'return &Response\{\s*JSONRPC: "2\.0",\s*ID: req\.ID,\s*Result: ([^}]+),\s*\}'
    content = re.sub(success_pattern, r'return \1, nil', content)

    with open(filepath, 'w') as f:
        f.write(content)

    print(f"Updated {filepath}")

# Update all files
update_handlers('bridge_handlers.go')
update_handlers('pii.go')
update_handlers('studio.go')
