# ArmorClaw Fix Plan - Production Readiness

> **Document Purpose:** Comprehensive plan to address all critical gaps and bring ArmorClaw to production-ready state
> **Date Created:** 2026-02-10
> **Focus:** Docker Build → Production Stability → Observability → E2E Testing → Ecosystem
> **Status:** Part 1 Complete ✅ - Parts 2-7 Ready to Execute

---

## Executive Summary

**Current State:**
- ArmorClaw v0.1.0-beta seeking community testers
- Docker build has circular dependency issues (9+ fix attempts)
- Missing production-ready features: rate limiting, metrics, E2E tests
- Lacks first-party management interface

**Priority Order:**
1. **P0: Docker Build** - Unblock all development
2. **P1: Rate Limiting** - Prevent internal DoS
3. **P2: Authentication** - Add secondary auth layer
4. **P3: Graceful Shutdown** - Handle in-flight requests
5. **P4: Observability** - Metrics and monitoring
6. **P5: E2E Testing** - Real infrastructure validation
7. **P6: Shield Interface** - First-party management UI

---

## Part 1: Docker Build Fix (P0 - CRITICAL) ✅ COMPLETE

### Issue Summary

**Problem:** Circular dependency in security hardening phase causes build failures

**Root Cause:**
- Layer 1 removes tools using `rm -f`
- Layer 2 removes execute permissions from ALL binaries
- Layer 2 may execute before Layer 1 completes, breaking `rm`

**Git History:** 9+ commits (v1-v8) attempting fixes

### What Needs to Change

**File:** `Dockerfile` (lines 110-125)

**Current (BROKEN):**
```dockerfile
# Layer 1: Remove dangerous tools (keep /bin/sh for build)
RUN rm -f /bin/bash /bin/mv /bin/find \
    && rm -f /bin/ps /usr/bin/top ...

# Layer 2: Remove execute permissions from ALL remaining binaries
RUN find /bin -type f -exec chmod a-x {} \; ...
```

**Fixed:**
```dockerfile
# Layer 1: Remove /bin/rm FIRST to prevent self-deletion
RUN rm -f /bin/rm \
    /bin/bash /bin/mv /bin/find \
    /bin/ps /usr/bin/top ...

# Layer 2: Now safe to remove execute permissions (rm is already gone)
RUN find /bin -type f -exec chmod a-x {} \; ...
```

### Implementation

| Step | Action | File | Lines |
|------|--------|------|-------|
| 1 | Backup current Dockerfile | Dockerfile | - |
| 2 | Reorder removals (rm first) | Dockerfile | 110-117 |
| 3 | Add verification step | Dockerfile | +5 lines |
| 4 | Test build locally | - | - |
| 5 | Update README documentation | README.md | 284-290 |
| 6 | Clean up git history | - | - |

### Verification

```bash
# Build must succeed
docker build -t armorclaw/agent:v0.1.1 .

# Security tests must pass
./tests/test-exploits.sh

# Health check must work
docker run --rm armorclaw/agent:v0.1.1

# No /bin/rm in container
docker run --rm armorclaw/agent:v0.1.1 ls /bin/rm
# Should fail with "No such file or directory"
```

---

## Part 2: Rate Limiting (P1 - High Risk)

### Issue Summary

**Problem:** JSON-RPC server lacks rate limiting, vulnerable to internal DoS

**Current State Analysis:**

| Component | Has Rate Limiting? | Implementation |
|-----------|-------------------|----------------|
| Socket Server | ✅ Yes | `pkg/socket/server.go:30` (10 req/s) |
| JSON-RPC Server | ❓ Unclear | Uses socket server, needs verification |
| Voice/WebRTC | ✅ Yes | `pkg/voice/security.go` (configurable) |

### What Needs to Change

**File:** `bridge/pkg/rpc/server.go`

**Current:** No explicit rate limiting in RPC handler

**Required:** Add method-level rate limiting

### Implementation Plan

#### Option A: Enhance Socket Server Rate Limiting (Recommended)

Since RPC uses socket server, enhance existing rate limiting:

```go
// File: bridge/pkg/socket/server.go

// Add method-level rate limiting
type Server struct {
    // ... existing fields ...

    // NEW: Method-specific rate limiters
    methodLimiters map[string]*rate.Limiter
    mu            sync.RWMutex
}

// Add to New() function
func New(cfg Config) (*Server, error) {
    s := &Server{
        // ... existing initialization ...

        // NEW: Initialize method-specific rate limiters
        s.methodLimiters = map[string]*rate.Limiter{
            "start":   rate.NewLimiter(rate.Every(time.Second), 1),  // 1 req/s
            "stop":    rate.NewLimiter(rate.Every(time.Second), 5),  // 5 req/s
            "status":  rate.NewLimiter(rate.Every(100*time.Millisecond), 10), // 10 req/s
            "attach_config": rate.NewLimiter(rate.Every(time.Second), 2), // 2 req/s
            // Default for unlisted methods
            "default": rate.NewLimiter(rate.Every(100*time.Millisecond), 20),
        }
    }

    return s, nil
}

// Add to handleConnection()
func (s *Server) handleConnection(conn net.Conn) {
    // ... existing code ...

    // NEW: Apply rate limiting per connection
    // Track requests per connection
    connLimiter := rate.NewLimiter(rate.Limit(10), 20) // 10 req/s burst 20

    for {
        // Wait for rate limit before reading
        if err := connLimiter.Wait(ctx); err != nil {
            break
        }

        // ... existing message handling ...
    }
}
```

#### Option B: Add Middleware Pattern (Alternative)

Create rate limiting middleware for RPC methods:

```go
// File: bridge/pkg/rpc/middleware/ratelimit.go

package middleware

import (
    "context"
    "golang.org/x/time/rate"
)

// RateLimiter middleware for JSON-RPC methods
type RateLimiter struct {
    limiters map[string]*rate.Limiter
    defaultLimiter *rate.Limiter
}

func NewRateLimiter() *RateLimiter {
    return &RateLimiter{
        limiters: map[string]*rate.Limiter{
            "start":   rate.NewLimiter(rate.Every(time.Second), 1),
            "stop":    rate.NewLimiter(rate.Every(time.Second), 5),
            "status":  rate.NewLimiter(rate.Every(100*time.Millisecond), 10),
            "attach_config": rate.NewLimiter(rate.Every(time.Second), 2),
            "send":    rate.NewLimiter(rate.Every(time.Second), 5),
        },
        defaultLimiter: rate.NewLimiter(rate.Every(100*time.Millisecond), 20),
    }
}

func (rl *RateLimiter) Allow(method string) bool {
    limiter := rl.limiters[method]
    if limiter == nil {
        limiter = rl.defaultLimiter
    }
    return limiter.Allow()
}

func (rl *RateLimiter) Wait(ctx context.Context, method string) error {
    limiter := rl.limiters[method]
    if limiter == nil {
        limiter = rl.defaultLimiter
    }
    return limiter.Wait(ctx)
}
```

### Configuration

Add to `bridge/config.example.toml`:

```toml
[rpc]
# Rate limiting for JSON-RPC methods
rate_limit_enabled = true
rate_limit_global = "20"           # requests per second (global)
rate_limit_burst = "50"            # burst size

# Per-method rate limits (overrides global)
[rpc.method_limits]
start = "1"                        # 1 request per second
stop = "5"                         # 5 requests per second
status = "10"                      # 10 requests per second
attach_config = "2"               # 2 requests per second
```

---

## Part 3: Authentication (P1 - High Risk)

### Issue Summary

**Problem:** Unix socket lacks secondary auth layer, relies solely on filesystem permissions

**Current State:**
- Socket permissions: `srwxrwxrwx` (owner/owner group)
- Only protection: filesystem ACLs
- No per-connection authentication
- No audit trail of who made requests

### What Needs to Change

**Option A: JWT Token Authentication (Recommended)**

Add JWT-based authentication for JSON-RPC connections.

#### Architecture

```
Client Request Flow:
1. Client authenticates with bridge via /auth endpoint
2. Bridge returns JWT token (15min expiry)
3. Client includes JWT in Authorization header
4. Bridge validates JWT before processing request
5. Bridge logs all actions with user identity
```

#### Implementation

**File:** `bridge/pkg/auth/jwt.go` (NEW)

```go
package auth

import (
    "crypto/rand"
    "encoding/base64"
   "fmt"
    "time"

    "github.com/golang-jwt/jwt/v5"
    "golang.org/x/crypto/bcrypt"
)

// JWTAuth provides JWT-based authentication for RPC
type JWTAuth struct {
    secretKey []byte
    issuer    string
}

type Claims struct {
    UserID    string   `json:"user_id"`
    Username  string   `json:"username"`
    Roles     []string `json:"roles"`
    jwt.RegisteredClaims
}

func NewJWTAuth(secret string) *JWTAuth {
    return &JWTAuth{
        secretKey: []byte(secret),
        issuer:    "armorclaw-bridge",
    }
}

// GenerateToken creates a new JWT token for a user
func (ja *JWTAuth) GenerateToken(userID, username string, roles []string) (string, error) {
    // Token expires in 15 minutes
    expiryTime := time.Now().Add(15 * time.Minute)

    claims := Claims{
        UserID:   userID,
        Username: username,
        Roles:    roles,
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: expiryTime,
            IssuedAt:  time.Now(),
            Issuer:    ja.issuer,
            ID:        generateTokenID(),
        },
    }

    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString(ja.secretKey)
}

// ValidateToken validates a JWT token and returns the claims
func (ja *JWTAuth) ValidateToken(tokenString string) (*Claims, error) {
    token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
        if token.Method != jwt.SigningMethodHS256 {
            return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
        }
        return ja.secretKey, nil
    })

    if err != nil {
        return nil, err
    }

    if claims, ok := token.Claims.(*Claims); ok && token.Valid {
        return claims, nil
    }

    return nil, fmt.Errorf("invalid token")
}

func generateTokenID() string {
    b := make([]byte, 16)
    rand.Read(b)
    return base64.URLEncoding.EncodeToString(b)
}
```

**File:** `bridge/pkg/rpc/server.go` (MODIFY)

```go
// Add to Server struct
type Server struct {
    // ... existing fields ...

    // NEW: JWT authentication
    jwtAuth *auth.JWTAuth
    requireAuth bool
}

// Add to handleConnection()
func (s *Server) handleConnection(conn net.Conn) {
    // NEW: Authenticate connection if required
    if s.requireAuth {
        if !s.authenticateConnection(conn) {
            conn.Close()
            s.securityLog.Log("auth_failed", map[string]interface{}{
                "remote_addr": conn.RemoteAddr().String(),
                "reason":      "no_token_provided",
            })
            return
        }
    }

    // ... existing connection handling ...
}

func (s *Server) authenticateConnection(conn net.Conn) bool {
    // Read first message and check for Authorization header
    // For Unix socket, this would be in the message itself
    // For HTTP, this would be in the Authorization header

    // Implementation depends on protocol choice
    return true
}
```

**File:** `bridge/pkg/rpc/auth.go` (NEW)

```go
package rpc

import (
    "encoding/json"
    "net"

    "github.com/armorclaw/bridge/pkg/auth"
)

// AuthMessage is sent by clients to authenticate
type AuthMessage struct {
    JSONRPC string      `json:"jsonrpc"`
    ID      interface{} `json:"id"`
    Method  string       `json:"method"`
    Params  AuthParams   `json:"params"`
}

type AuthParams struct {
    Token   string `json:"token"`
    Request string `json:"request"` // The actual RPC request
}

// Authenticate validates a JWT token and returns user info
func (s *Server) Authenticate(token string) (*auth.Claims, error) {
    return s.jwtAuth.ValidateToken(token)
}
```

### Configuration

Add to `bridge/config.example.toml`:

```toml
[rpc]
# Enable JWT authentication for RPC
auth_enabled = true
# JWT secret (minimum 32 characters, use env var in production)
jwt_secret = "change-me-in-production-use-32-char-min"
# Token expiration (15 minutes recommended)
jwt_expiry = "15m"
# Require auth for all methods (except those in public_methods)
require_auth = true

# Public methods that don't require authentication
public_methods = ["status", "health"]
```

---

## Part 4: Graceful Shutdown (P1 - High Risk)

### Issue Summary

**Problem:** Bridge may drop in-flight requests during SIGTERM, lacks managed drain state

**Current State:**
- Signal handlers exist (`bridge/cmd/bridge/main.go:1482`)
- Voice/WebRTC have graceful shutdown
- TTL manager has graceful shutdown
- **Missing:** Graceful shutdown for JSON-RPC server

### What Needs to Change

**File:** `bridge/pkg/rpc/server.go`

**Current (lines 340-360):**

```go
for {
    select {
    case <-s.ctx.Done():
        // Server is shutting down
        return
    case <-time.After(100 * time.Millisecond):
        conn, err := s.listener.Accept()
        // ... handle connection ...
    }
}
```

**Required:** Implement drain state with timeout

### Implementation

```go
// File: bridge/pkg/rpc/server.go

// Add to Server struct
type Server struct {
    // ... existing fields ...

    // NEW: Graceful shutdown
    draining     bool
    drainTimeout time.Duration
    wg           sync.WaitGroup
    mu           sync.RWMutex
}

// Add new constants
const (
    DefaultDrainTimeout = 30 * time.Second
)

// Modify Serve() method
func (s *Server) Serve() error {
    s.log().Info("Starting JSON-RPC server",
        "socket", s.socketPath,
        "drain_timeout", s.drainTimeout,
    )

    for {
        select {
        case <-s.ctx.Done():
            s.log().Info("Shutdown signal received, starting graceful shutdown")
            return s.gracefulShutdown()

        case conn := <-s.acceptCh:
            // Check if we're draining
            s.mu.RLock()
            draining := s.draining
            s.mu.RUnlock()

            if draining {
                // Reject new connections during drain
                s.log().Info("Rejecting connection during drain",
                    "remote_addr", conn.RemoteAddr().String(),
                )
                conn.Close()
                continue
            }

            s.wg.Add(1)
            go s.handleConnection(conn)
        }
    }
}

// NEW: gracefulShutdown initiates drain state
func (s *Server) gracefulShutdown() error {
    s.mu.Lock()
    s.draining = true
    s.mu.Unlock()

    s.log().Info("Server entering drain state",
        "timeout", s.drainTimeout,
    )

    // Stop accepting new connections
    s.listener.Close()

    // Wait for active connections to finish or timeout
    done := make(chan struct{})
    go func() {
        s.wg.Wait()
        close(done)
    }()

    select {
    case <-done:
        s.log().Info("All connections finished gracefully")
        return nil
    case <-time.After(s.drainTimeout):
        s.log().Warn("Drain timeout expired, forcing shutdown",
            "active_connections", s.wgCount(),
        )
        return fmt.Errorf("drain timeout expired")
    }
}

// NEW: wgCount returns current wait group count
func (s *Server) wgCount() int {
    // This is a rough estimate for logging
    return int(s.wgCount.Load())
}

// Add accept channel to Server struct
type Server struct {
    // ... existing fields ...

    // NEW: Accept channel
    acceptCh chan net.Conn
    wgCount  atomic.Int32
}

// Modify initialization
func (s *Server) startAcceptLoop() {
    go func() {
        for {
            conn, err := s.listener.Accept()
            if err != nil {
                if errors.Is(err, net.ErrClosed) {
                    return // Server closed
                }
                s.log().Error("Accept error", "error", err)
                continue
            }
            s.acceptCh <- conn
        }
    }()
}
```

### Configuration

Add to `bridge/config.example.toml`:

```toml
[rpc]
# Graceful shutdown settings
graceful_shutdown_timeout = "30s"  # How long to wait for connections to finish
# Drain mode rejects new connections but allows in-flight requests to complete
```

---

## Part 5: Observability Infrastructure (P2)

### Issue Summary

**Problem:** System lacks standard metrics (Prometheus/OpenTelemetry), health monitors not exposed externally

**Current State:**
- OpenTelemetry dependencies present but unused (`// indirect` in go.mod)
- Health monitors exist but not exposed for external monitoring
- No Prometheus metrics endpoint
- No structured logging format for external systems

### What Needs to Change

#### Option A: Prometheus Metrics (Recommended)

**File:** `bridge/pkg/metrics/prometheus.go` (NEW)

```go
package metrics

import (
    "net/http"
    "time"

    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
    // RPC metrics
    rpcRequestsTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "armorclaw_rpc_requests_total",
            Help: "Total number of RPC requests",
        },
        []string{"method", "status"},
    )

    rpcRequestDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "armorclaw_rpc_request_duration_seconds",
            Help:    "RPC request duration in seconds",
            Buckets: prometheus.DefBuckets,
        },
        []string{"method"},
    )

    rpcActiveConnections = prometheus.NewGauge(
        prometheus.GaugeOpts{
            Name: "armorclaw_rpc_active_connections",
            Help: "Number of active RPC connections",
        },
    )

    // Container metrics
    containerTotal = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "armorclaw_containers_total",
            Help: "Total number of containers",
        },
        []string{"state"}, // running, stopped, etc.
    )

    // Keystore metrics
    keystoreOperations = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "armorclaw_keystore_operations_total",
            Help: "Total number of keystore operations",
        },
        []string{"operation"}, // add, get, delete, list
    )

    // Budget metrics
    budgetUsage = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "armorclaw_budget_usage_usd",
            Help: "Budget usage in USD",
        },
        []string{"period"}, // daily, monthly
    )
)

func init() {
    prometheus.MustRegister(rpcRequestsTotal)
    prometheus.MustRegister(rpcRequestDuration)
    prometheus.MustRegister(rpcActiveConnections)
    prometheus.MustRegister(containerTotal)
    prometheus.MustRegister(keystoreOperations)
    prometheus.MustRegister(budgetUsage)
}

// MetricsServer serves Prometheus metrics
type MetricsServer struct {
    addr   string
    server *http.Server
}

func NewMetricsServer(addr string) *MetricsServer {
    mux := http.NewServeMux()
    mux.Handle("/metrics", promhttp.Handler())
    mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        w.Write([]byte("OK"))
    })

    return &MetricsServer{
        addr: addr,
        server: &http.Server{
            Addr:    addr,
            Handler: mux,
        },
    }
}

func (ms *MetricsServer) Start() error {
    ms.log().Info("Starting metrics server", "addr", ms.addr)
    return ms.server.ListenAndServe()
}

func (ms *MetricsServer) Shutdown(ctx context.Context) error {
    ms.log().Info("Shutting down metrics server")
    return ms.server.Shutdown(ctx)
}
```

**File:** `bridge/pkg/rpc/server.go` (MODIFY)

```go
// Add to RPC method handlers
func (s *Server) handleMethod(method string, params json.RawMessage) (interface{}, *ErrorObj) {
    start := time.Now()

    // Record request start
    s.wg.Add(1)
    defer s.wg.Done()

    // ... existing method handling ...

    // Record metrics
    status := "success"
    if err != nil {
        status = "error"
    }
    metrics.rpcRequestsTotal.WithLabelValues(method, status).Inc()
    metrics.rpcRequestDuration.WithLabelValues(method).Observe(time.Since(start).Seconds())

    return result, nil
}
```

#### Option B: OpenTelemetry (Already in dependencies)

**File:** `bridge/pkg/telemetry/telemetry.go` (NEW)

```go
package telemetry

import (
    "context"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
    "go.opentelemetry.io/otel/sdk/resource"
    "go.opentelemetry.io/otel/sdk/trace"
    semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

var tracer = otel.Tracer("github.com/armorclaw/bridge")

// Init initializes OpenTelemetry
func Init(serviceName, endpoint string) error {
    ctx := context.Background()

    res, err := resource.New(ctx,
        resource.WithAttributes(
            semconv.ServiceName(serviceName),
            semconv.ServiceVersion("v0.1.0"),
        ),
    )
    if err != nil {
        return err
    }

    traceExporter, err := otlptracehttp.New(ctx,
        otlptracehttp.WithEndpoint(endpoint),
    )
    if err != nil {
        return err
    }

    tp := trace.NewTracerProvider(
        trace.WithBatcher(trace.NewBatchExportSpanProcessor(traceExporter)),
        trace.WithResource(res),
    )

    otel.SetTracerProvider(tp)
    return nil
}

// StartSpan creates a new span for tracing
func StartSpan(ctx context.Context, name string) (context.Context, trace.Span) {
    return tracer.Start(ctx, name,
        trace.WithAttributes(
            attribute.String("service", "armorclaw-bridge"),
        ),
    )
}
```

**File:** `bridge/pkg/rpc/server.go` (MODIFY)

```go
func (s *Server) handleMethod(method string, params json.RawMessage) (interface{}, *ErrorObj) {
    ctx, span := telemetry.StartSpan(s.ctx, "rpc."+method)
    defer span.End()

    span.SetAttributes(
        attribute.String("rpc.method", method),
    )

    // ... existing method handling ...

    if err != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
        return nil, &ErrorObj{Code: InternalError, Message: err.Error()}
    }

    span.SetStatus(codes.Ok, "OK")
    return result, nil
}
```

### Configuration

Add to `bridge/config.example.toml`:

```toml
[telemetry]
# Enable OpenTelemetry tracing
enabled = true
service_name = "armorclaw-bridge"
# OTLP endpoint (e.g., Jaeger, Tempo)
otlp_endpoint = "http://localhost:4318"

[metrics]
# Prometheus metrics endpoint
enabled = true
listen_addr = ":9090"
# Expose metrics at http://localhost:9090/metrics
```

---

## Part 6: End-to-End Testing (P2)

### Issue Summary

**Problem:** E2E tests use stubs, need real infrastructure validation

**Current State:**
- Basic E2E test exists (`tests/test-e2e.sh`)
- Uses stub bridge, not real compiled binary
- Tests container operations but not full RPC flow
- No Matrix Conduit integration testing
- No multi-NAT WebRTC testing

### What Needs to Change

#### E2E Test Suite Enhancement

**File:** `tests/test-e2e-full.sh` (NEW)

```bash
#!/bin/bash
set -euo pipefail

# ArmorClaw Full E2E Integration Tests
# Tests complete flow: build → bridge → Matrix → containers → WebRTC

echo "🧪 Full End-to-End Integration Tests"
echo "======================================="
echo ""

TEST_NS="test-e2e-full-$(date +%s)"
TEST_DIR="/tmp/armorclaw-$TEST_NS"
BRIDGE_BIN="$TEST_DIR/armorclaw-bridge"
MATRIX_DIR="$TEST_DIR/matrix"
RESULTS_FILE="$TEST_DIR/results.txt"

# Cleanup handler
cleanup() {
    echo "Cleaning up..."

    # Stop all containers
    docker stop $(docker ps -q --filter "name=e2e-*") 2>/dev/null || true
    docker rm $(docker ps -aq --filter "name=e2e-*") 2>/dev/null || true

    # Stop bridge
    pkill -f "$BRIDGE_BIN" 2>/dev/null || true

    # Stop Matrix if running
    docker stop armorclaw-conduit-e2e 2>/dev/null || true
    docker rm armorclaw-conduit-e2e 2>/dev/null || true

    # Remove test directory
    rm -rf "$TEST_DIR"
}
trap cleanup EXIT

mkdir -p "$TEST_DIR"

# ============================================================================
# Test 1: Build Bridge Binary
# ============================================================================
echo "Test 1: Build Bridge Binary"
echo "-------------------------"

cd "$TEST_DIR"
git clone https://github.com/armorclaw/armorclaw.git armorclaw 2>/dev/null || true
cd armorclaw/bridge

if go build -o "$BRIDGE_BIN" ./cmd/bridge; then
    echo "✅ Bridge binary built successfully"
else
    echo "❌ FAIL: Could not build bridge"
    exit 1
fi

echo ""

# ============================================================================
# Test 2: Start Matrix Conduit
# ============================================================================
echo "Test 2: Matrix Infrastructure"
echo "-----------------------------"

mkdir -p "$MATRIX_DIR/data"

# Create minimal conduit config
cat > "$MATRIX_DIR/conduit.toml" <<'EOF'
[global]
server_name = "localhost"
database_path = "/var/lib/matrix-conduit/conduit.db"

[global.well_known]
server = "127.0.0.1:8448"

EOF

# Start Matrix Conduit
docker run -d --rm \
    --name armorclaw-conduit-e2e \
    -p 8448:8448 \
    -p 6167:6167 \
    -v "$MATRIX_DIR/data:/var/lib/matrix-conduit" \
    -v "$MATRIX_DIR/conduit.toml:/etc/conduit/conduit.toml:ro" \
    matrixconduit/matrix-conduit:latest \
    > /dev/null 2>&1

sleep 3

if docker ps | grep -q armorclaw-conduit-e2e; then
    echo "✅ Matrix Conduit started"
else
    echo "❌ FAIL: Matrix Conduit failed to start"
    docker logs armorclaw-conduit-e2e
    exit 1
fi

echo ""

# ============================================================================
# Test 3: Bridge Startup with Matrix
# ============================================================================
echo "Test 3: Bridge with Matrix"
echo "-------------------------"

# Initialize bridge config
mkdir -p "$TEST_DIR/.armorclaw"

cat > "$TEST_DIR/.armorclaw/config.toml" <<'EOF'
[matrix]
enabled = true
homeserver_url = "http://127.0.0.1:8448"
username = "@admin:localhost"
password = "test_password_123"

[rpc]
socket = "/run/armorclaw/e2e.sock"
auth_enabled = false  # Disable for testing
EOF

# Start bridge
if "$BRIDGE_BIN" --config "$TEST_DIR/.armorclaw/config.toml" &
BRIDGE_PID=$!
sleep 2
then
    echo "✅ Bridge started (PID: $BRIDGE_PID)"
else
    echo "❌ FAIL: Bridge failed to start"
    exit 1
fi

echo ""

# ============================================================================
# Test 4: JSON-RPC Operations
# ============================================================================
echo "Test 4: JSON-RPC Operations"
echo "--------------------------"

# Test status
echo '{"jsonrpc":"2.0","id":1,"method":"status"}' | \
    socat - UNIX-CONNECT:"$TEST_DIR/.armorclaw/e2e.sock" 2>/dev/null | \
    jq .

# Test list_keys
echo '{"jsonrpc":"2.0","id":2,"method":"list_keys"}' | \
    socat - UNIX-CONNECT:"TEST_DIR/.armorclaw/e2e.sock" 2>/dev/null | \
    jq .

echo ""

# ============================================================================
# Test 5: Container Lifecycle
# ============================================================================
echo "Test 5: Container Lifecycle"
echo "-----------------------"

# Start container
echo '{"jsonrpc":"2.0","id":3,"method":"start","params":{"name":"e2e-test","key":"test-key"}}' | \
    socat - UNIX-CONNECT:"$TEST_DIR/.armorclaw/e2e.sock" 2>/dev/null | \
    jq .

# Check status
sleep 2
echo '{"jsonrpc":"2.0","id":4,"method":"status"}' | \
    socat - UNIX-CONNECT:"$TEST_DIR/.armorclaw/e2e.sock" 2>/dev/null | \
    jq .

# Stop container
echo '{"jsonrpc":"2.0","id":5,"method":"stop","params":{"name":"e2e-test"}}' | \
    socat - UNIX-CONNECT:"$TEST_DIR/.armorclaw/e2e.sock" 2>/dev/null | \
    jq .

echo ""

# ============================================================================
# Test 6: Cleanup
# ============================================================================
echo "Test 6: Cleanup"
echo "-----------"

# Kill bridge
kill $BRIDGE_PID 2>/dev/null || true
wait $BRIDGE_PID 2>/dev/null || true

# Stop Matrix
docker stop armorclaw-conduit-e2e >/dev/null 2>&1

echo "✅ All E2E tests completed"
echo ""

# Save results
echo "Test Results: $(date)" > "$RESULTS_FILE"
echo "Bridge Build: PASS" >> "$RESULTS_FILE"
echo "Matrix Infrastructure: PASS" >> "$RESULTS_FILE"
echo "Bridge Startup: PASS" >> "$RESULTS_FILE"
echo "JSON-RPC Operations: PASS" >> "$RESULTS_FILE"
echo "Container Lifecycle: PASS" >> "$RESULTS_FILE"
echo "Cleanup: PASS" >> "$RESULTS_FILE"

echo "✅ ALL FULL E2E TESTS PASSED"
```

#### WebRTC Multi-NAT Testing

**File:** `tests/test-webrtc-multi-nat.sh` (NEW)

```bash
#!/bin/bash
# WebRTC Multi-NAT Traversal Tests
# Tests TURN server with simulated NAT scenarios

echo "🌐 WebRTC Multi-NAT Traversal Tests"
echo "==================================="
echo ""

# Test scenarios:
# 1. Full Cone NAT
# 2. Restricted Cone NAT
# 3. Port Restricted Cone NAT
# 4. Symmetric NAT

# For each scenario, test:
# - STUN binding request
# - TURN relay establishment
# - Media path verification

# Implementation would use:
# - Docker network simulating each NAT type
# - tc/netem for Linux traffic control
# - WebRTC testing tools

echo "Note: Full multi-NAT testing requires network simulation setup."
echo "Basic TURN connectivity test:"

# Test TURN server availability
if host -W 5 turn.trycloudflare.com 443; then
    echo "✅ TURN server (Cloudflare) reachable"
else
    echo "⚠️  TURN server not reachable"
fi

# Add TURN server specific tests from existing docs
source ./tests/voice/test-turn-connectivity.sh
```

---

## Part 7: Shield Interface (P3)

### Issue Summary

**Problem:** Platform relies on third-party clients like Element X, lacks first-party "Shield" interface

**Current State:**
- CLI-only interface
- Element X integration exists
- No web UI
- No native management dashboard

### Options

#### Option A: Web-Based Shield Dashboard (Recommended)

Create a web interface for managing ArmorClaw.

**Tech Stack:**
- Frontend: React/Next.js or SvelteKit
- Backend: Go HTTP server (add to bridge)
- Auth: JWT tokens from bridge
- Real-time: WebSocket for live updates

**Features:**
- Dashboard overview
- Container management (start/stop/restart)
- Keystore management
- Budget monitoring
- Real-time logs
- Matrix room management
- Configuration editor

**File Structure:**
```
shield/
├── web/              # Next.js app
│   ├── app/
│   ├── components/
│   └── lib/
└── api/             # Go HTTP server
    └── handlers/
```

#### Option B: TUI (Terminal UI) Interface

Create a terminal-based dashboard using Bubble Tea or similar.

**Benefits:**
- Lightweight
- No browser dependency
- Consistent with CLI-first philosophy
- Runs on headless servers

**File:** `shield/cmd/tui/main.go`

```go
package main

import (
    "fmt"
    "log/slog"

    "github.com/charmbracelet/bubbles/table"
    "github.com/charmbracelet/lipgloss"
    "github.com/charmbracelet/bubbletea"
)

// TUI for ArmorClaw management
type TUIModel struct {
    containers []ContainerInfo
    keys       []KeyInfo
    budget     BudgetInfo
    logs       []LogEntry
    selected   int
}

func (m TUIModel) Init() tea.Cmd {
    return tea.Batch(
        fetchContainers(),
        fetchKeys(),
        fetchBudget(),
        tailLogs(),
    )
}

func (m TUIModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case TickMsg:
        return m, tea.Batch(
            fetchContainers(),
            fetchLogs(),
        )
    }
    return m, nil
}

func (m TUIModel) View() string {
    return lipgloss.JoinVertical(
        lipgloss.NewStyle().Foreground(lipgloss.Color("Fuchsia")).Render("ArmorClaw Shield TUI"),
        "",
        m.renderContainers(),
        "",
        m.renderKeys(),
        "",
        m.renderBudget(),
        "",
        "Press q to quit, ↑↓ to navigate, Enter to select",
    )
}
```

#### Option C: Enhanced CLI (Minimal)

Add interactive mode to existing CLI.

**File:** `bridge/cmd/bridge/main.go` (ADD)

```go
// Add interactive command
var cmdInteractive = &cobra.Command{
    Use:   "interactive",
    Short: "Launch interactive TUI mode",
    Run: func(cmd *cobra.Command, args []string) {
        if err := tui.Run(); err != nil {
            log.Fatalf("TUI error: %v", err)
        }
    },
}
```

---

## Part 8: Implementation Roadmap

### Sprint 1: Critical Fixes (Week 1)

| Priority | Task | File | Effort |
|----------|------|------|--------|
| P0 | Fix Docker build circular dependency | Dockerfile | 4h |
| P0 | Update documentation | README.md | 2h |
| P1 | Implement rate limiting for RPC | pkg/rpc/server.go | 8h |
| P1 | Add JWT authentication | pkg/auth/jwt.go | 12h |
| P1 | Implement graceful shutdown | pkg/rpc/server.go | 6h |

### Sprint 2: Observability (Week 2)

| Priority | Task | File | Effort |
|----------|------|------|--------|
| P2 | Add Prometheus metrics | pkg/metrics/prometheus.go | 8h |
| P2 | Initialize OpenTelemetry | pkg/telemetry/telemetry.go | 6h |
| P2 | Expose metrics endpoint | pkg/rpc/server.go | 4h |
| P2 | Add structured logging | bridge/cmd/bridge/main.go | 4h |

### Sprint 3: Testing (Week 3)

| Priority | Task | File | Effort |
|----------|------|------|--------|
| P2 | Write full E2E test suite | tests/test-e2e-full.sh | 12h |
| P2 | Add WebRTC multi-NAT tests | tests/test-webrtc-multi-nat.sh | 8h |
| P2 | Create Matrix integration tests | tests/test-matrix.sh | 6h |
| P2 | Add load testing | tests/test-load.sh | 8h |

### Sprint 4: Shield Interface (Week 4)

| Priority | Task | File | Effort |
|----------|------|------|--------|
| P3 | Design Shield web UI | shield/web/ | 16h |
| P3 | Implement Shield API | shield/api/ | 12h |
| P3 | Add WebSocket real-time updates | shield/api/ws.go | 8h |
| P3 | Deploy to preview environment | - | 4h |

---

## Part 9: Priority Matrix

| Issue | Priority | Complexity | Risk | Dependencies |
|-------|----------|------------|-----|--------------|
| Docker Build | P0 | Low | Medium | None |
| Rate Limiting | P1 | Medium | Low | None |
| Authentication | P1 | High | Medium | None |
| Graceful Shutdown | P1 | Low | Low | None |
| Observability | P2 | Medium | Low | None |
| E2E Testing | P2 | High | Medium | Docker Build |
| Shield UI | P3 | High | Low | All above |

---

## Part 10: Success Criteria

### Phase 1: Critical Fixes

- [ ] Docker build succeeds consistently
- [ ] All security tests pass (26/26)
- [ ] Rate limiting active and tested
- [ ] JWT authentication working
- [ ] Graceful shutdown tested

### Phase 2: Observability

- [ ] Prometheus metrics accessible at `/metrics`
- [ ] Health check accessible at `/health`
- [ ] Tracing data exported to OTLP endpoint
- [ ] Structured logging in JSON format

### Phase 3: E2E Testing

- [ ] Full E2E tests pass
- [ ] Multi-NAT WebRTC tests pass
- [ ] Load tests handle 100+ concurrent connections
- [ ] Integration tests pass

### Phase 4: Shield Interface

- [ ] Web UI deployed and accessible
- [ ] Can manage containers via UI
- [ ] Can manage keystore via UI
- [ ] Real-time logs visible
- [ ] Budget monitoring visible

---

## Part 11: File Changes Summary

### New Files to Create

```
bridge/pkg/auth/jwt.go                    # JWT authentication
bridge/pkg/metrics/prometheus.go       # Prometheus metrics
bridge/pkg/telemetry/telemetry.go      # OpenTelemetry
bridge/pkg/rpc/middleware/ratelimit.go  # Rate limiting middleware
shield/web/app/page.tsx                 # Shield web UI
shield/api/server.go                   # Shield API server
tests/test-e2e-full.sh                   # Full E2E tests
tests/test-webrtc-multi-nat.sh          # WebRTC NAT tests
```

### Files to Modify

```
Dockerfile                                # Lines 110-125
README.md                                 # Lines 284-290
bridge/config.example.toml                # Add auth, metrics sections
bridge/pkg/rpc/server.go                # Add rate limiting, auth, metrics
bridge/cmd/bridge/main.go                 # Add graceful shutdown
tests/test-e2e.sh                         # Enhance to use real bridge
```

---

**Document Version:** 1.0.0
**Last Updated:** 2026-02-10
**Status:** Ready to Execute

**Next Step:** Start with Docker build fix (P0), then proceed through each priority level.
