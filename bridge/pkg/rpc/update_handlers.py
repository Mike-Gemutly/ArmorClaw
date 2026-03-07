#!/usr/bin/env python3
import re
import sys

# Read the file
with open('browser.go', 'r') as f:
    content = f.read()

# Replace all handler signatures
content = re.sub(
    r'func \(s \*Server\) handleBrowserWaitForElement\(req \*Request\) \*Response',
    r'func (s *Server) handleBrowserWaitForElement(ctx context.Context, req *Request) (interface{}, *ErrorObj)',
    content
)
content = re.sub(
    r'func \(s \*Server\) handleBrowserWaitForCaptcha\(req \*Request\) \*Response',
    r'func (s *Server) handleBrowserWaitForCaptcha(ctx context.Context, req *Request) (interface{}, *ErrorObj)',
    content
)
content = re.sub(
    r'func \(s \*Server\) handleBrowserWaitFor2FA\(req \*Request\) \*Response',
    r'func (s *Server) handleBrowserWaitFor2FA(ctx context.Context, req *Request) (interface{}, *ErrorObj)',
    content
)
content = re.sub(
    r'func \(s \*Server\) handleBrowserComplete\(req \*Request\) \*Response',
    r'func (s *Server) handleBrowserComplete(ctx context.Context, req *Request) (interface{}, *ErrorObj)',
    content
)
content = re.sub(
    r'func \(s \*Server\) handleBrowserFail\(req \*Request\) \*Response',
    r'func (s *Server) handleBrowserFail(ctx context.Context, req *Request) (interface{}, *ErrorObj)',
    content
)
content = re.sub(
    r'func \(s \*Server\) handleBrowserList\(req \*Request\) \*Response',
    r'func (s *Server) handleBrowserList(ctx context.Context, req *Request) (interface{}, *ErrorObj)',
    content
)
content = re.sub(
    r'func \(s \*Server\) handleBrowserCancel\(req \*Request\) \*Response',
    r'func (s *Server) handleBrowserCancel(ctx context.Context, req *Request) (interface{}, *ErrorObj)',
    content
)

# Write back
with open('browser.go', 'w') as f:
    f.write(content)

print("Updated browser.go handler signatures")
