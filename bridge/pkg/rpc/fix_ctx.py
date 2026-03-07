#!/usr/bin/env python3

def fix_ctx_references():
    with open('browser.go', 'r') as f:
        content = f.read()

    # Replace all s.ctx with ctx
    content = content.replace('context.WithTimeout(s.ctx, ', 'context.WithTimeout(ctx, ')
    content = content.replace('ctx := s.ctx', 'ctx := ctx')
    content = content.replace('s.ctx', 'ctx')

    with open('browser.go', 'w') as f:
        f.write(content)

    print("Fixed all s.ctx references to ctx")

fix_ctx_references()
