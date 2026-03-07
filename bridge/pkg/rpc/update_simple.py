#!/usr/bin/env python3

# Simple replacements without f-strings
with open('pii.go', 'r') as f:
    content = f.read()

# Replace all handler signatures
content = content.replace(
    'func (s *Server) handlePIIRequest(req *Request) *Response',
    'func (s *Server) handlePIIRequest(ctx context.Context, req *Request) (interface{}, *ErrorObj)'
)
content = content.replace(
    'func (s *Server) handlePIIApprove(req *Request) *Response',
    'func (s *Server) handlePIIApprove(ctx context.Context, req *Request) (interface{}, *ErrorObj)'
)
content = content.replace(
    'func (s *Server) handlePIIDeny(req *Request) *Response',
    'func (s *Server) handlePIIDeny(ctx context.Context, req *Request) (interface{}, *ErrorObj)'
)
content = content.replace(
    'func (s *Server) handlePIIStatus(req *Request) *Response',
    'func (s *Server) handlePIIStatus(ctx context.Context, req *Request) (interface{}, *ErrorObj)'
)
content = content.replace(
    'func (s *Server) handlePIIListPending(req *Request) *Response',
    'func (s *Server) handlePIIListPending(ctx context.Context, req *Request) (interface{}, *ErrorObj)'
)
content = content.replace(
    'func (s *Server) handlePIIStats(req *Request) *Response',
    'func (s *Server) handlePIIStats(ctx context.Context, req *Request) (interface{}, *ErrorObj)'
)
content = content.replace(
    'func (s *Server) handlePIICancel(req *Request) *Response',
    'func (s *Server) handlePIICancel(ctx context.Context, req *Request) (interface{}, *ErrorObj)'
)
content = content.replace(
    'func (s *Server) handlePIIFulfill(req *Request) *Response',
    'func (s *Server) handlePIIFulfill(ctx context.Context, req *Request) (interface{}, *ErrorObj)'
)
content = content.replace(
    'func (s *Server) handlePIIWaitForApproval(req *Request) *Response',
    'func (s *Server) handlePIIWaitForApproval(ctx context.Context, req *Request) (interface{}, *ErrorObj)'
)

with open('pii.go', 'w') as f:
    f.write(content)

print("Updated pii.go signatures")

# Update studio.go
with open('studio.go', 'r') as f:
    content = f.read()

content = content.replace(
    'func (s *Server) handleStudio(req *Request) *Response',
    'func (s *Server) handleStudio(ctx context.Context, req *Request) (interface{}, *ErrorObj)'
)

with open('studio.go', 'w') as f:
    f.write(content)

print("Updated studio.go signatures")
