#!/bin/bash
#
# ArmorClaw Bridge Repository Initialization
# Creates the Go project structure for the Local Bridge
#

set -e

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}ArmorClaw Bridge Repository Init${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""

# Configuration
BRIDGE_DIR=${BRIDGE_DIR:-./bridge}
MODULE_PATH=${MODULE_PATH:-github.com/armorclaw/bridge}

echo -e "${YELLOW}Creating Go project structure...${NC}"

# Create directory structure
mkdir -p "$BRIDGE_DIR/cmd/armorclaw-bridge"
mkdir -p "$BRIDGE_DIR/internal/bridge"
mkdir -p "$BRIDGE_DIR/internal/keystore"
mkdir -p "$BRIDGE_DIR/internal/adapter"
mkdir -p "$BRIDGE_DIR/internal/config"
mkdir -p "$BRIDGE_DIR/pkg/api"
mkdir -p "$BRIDGE_DIR/scripts"
mkdir -p "$BRIDGE_DIR/configs"
mkdir -p "$BRIDGE_DIR/agent-skill"

echo -e "${GREEN}✓ Directory structure created${NC}"
echo ""

# Initialize Go module
echo -e "${YELLOW}Initializing Go module...${NC}"
cd "$BRIDGE_DIR"
go mod init "$MODULE_PATH"

echo -e "${GREEN}✓ Go module initialized: $MODULE_PATH${NC}"
echo ""

# Install core dependencies
echo -e "${YELLOW}Installing Go dependencies...${NC}"
go get \
    github.com/matrix-org/gomatrix \
    github.com/mitchellh/mapstructure \
    github.com/BurntSushi/toml \
    gopkg.in/natefinch/lumberjack.v2

echo -e "${GREEN}✓ Dependencies installed${NC}"
echo ""

# Create Makefile
echo -e "${YELLOW}Creating build files...${NC}"
cat > "$BRIDGE_DIR/Makefile" << 'EOF'
# ArmorClaw Bridge Makefile

.PHONY: build test clean install run

# Build variables
BINARY_NAME=armorclaw-bridge
BUILD_DIR=build
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-X main.Version=$(VERSION)"

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/armorclaw-bridge
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

test:
	@echo "Running tests..."
	$(GOTEST) -v -race -cover ./...

test-coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -v -race -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html
	@echo "Clean complete"

install: build
	@echo "Installing $(BINARY_NAME)..."
	@sudo cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/
	@sudo chmod +x /usr/local/bin/$(BINARY_NAME)
	@echo "Installed to /usr/local/bin/$(BINARY_NAME)"

run:
	@echo "Running bridge..."
	$(GOBUILD) $(LDFLAGS) -o /tmp/$(BINARY_NAME) ./cmd/armorclaw-bridge
	/tmp/$(BINARY_NAME)

deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

fmt:
	@echo "Formatting code..."
	$(GOCMD) fmt ./...

vet:
	@echo "Vetting code..."
	$(GOCMD) vet ./...

lint: fmt vet
	@echo "Linting complete"

.DEFAULT_GOAL := build
EOF

echo -e "${GREEN}✓ Makefile created${NC}"
echo ""

# Create .gitignore
cat > "$BRIDGE_DIR/.gitignore" << 'EOF'
# Binaries
*.exe
*.exe~
*.dll
*.so
*.dylib
/build/
/bin/

# Test files
*.test
coverage.out
coverage.html

# Go workspace file
go.work

# IDE
.vscode/
.idea/
*.swp
*.swo
*~

# OS
.DS_Store
Thumbs.db

# Environment
.env
.env.local

# Logs
*.log

# Temporary files
/tmp/
EOF

echo -e "${GREEN}✓ .gitignore created${NC}"
echo ""

# Create main.go stub
cat > "$BRIDGE_DIR/cmd/armorclaw-bridge/main.go" << 'EOF'
package main

import (
    "context"
    "fmt"
    "log"
    "os"
    "os/signal"
    "syscall"

    "github.com/armorclaw/bridge/internal/config"
    "github.com/armorclaw/bridge/internal/bridge"
)

var (
    Version   = "dev"
    BuildTime = "unknown"
)

func main() {
    log.Printf("ArmorClaw Bridge v%s starting...", Version)

    // Load configuration
    cfg, err := config.Load("/etc/armorclaw/bridge.toml")
    if err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }

    // Create bridge server
    server, err := bridge.NewServer(cfg)
    if err != nil {
        log.Fatalf("Failed to create server: %v", err)
    }

    // Start server in background
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    if err := server.Start(ctx); err != nil {
        log.Fatalf("Failed to start server: %v", err)
    }
    defer server.Stop()

    log.Printf("Bridge started on %s", cfg.Bridge.SocketPath)

    // Wait for shutdown signal
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
    <-sigChan

    log.Println("Shutting down...")
}
EOF

echo -e "${GREEN}✓ main.go stub created${NC}"
echo ""

# Create README
cat > "$BRIDGE_DIR/README.md" << 'EOF'
# ArmorClaw Bridge

The Local Bridge provides secure credential isolation and protocol adapters for ArmorClaw agents.

## Features

- **Encrypted Keystore:** SQLCipher-based credential storage
- **Matrix Integration:** Full Matrix client protocol support
- **JSON-RPC API:** Simple Unix socket communication
- **Offline Queueing:** Message buffering during outages

## Quick Start

```bash
# Build
make build

# Install
sudo make install

# Configure
sudo cp configs/bridge.toml /etc/armorclaw/

# Run
armorclaw-bridge
```

## Development

```bash
# Install dependencies
make deps

# Run tests
make test

# Format code
make fmt

# Build
make build
```

## Configuration

Edit `/etc/armorclaw/bridge.toml`:

```toml
[bridge]
socket_path = "/run/armorclaw/bridge.sock"

[matrix]
homeserver_url = "http://localhost:6167"
username = "agent"
password = "your-password"
```

## API

Send a message:

```json
{
  "jsonrpc": "2.0",
  "method": "bridge.send",
  "params": {
    "room": "!room:id:server.com",
    "message": "Hello from agent"
  },
  "id": 1
}
```

Receive messages:

```json
{
  "jsonrpc": "2.0",
  "method": "bridge.receive",
  "params": {"timeout": 30},
  "id": 2
}
```

## License

MIT License - see LICENSE file.
EOF

echo -e "${GREEN}✓ README.md created${NC}"
echo ""

# Create config stub
cat > "$BRIDGE_DIR/configs/bridge.toml" << 'EOF'
# ArmorClaw Bridge Configuration

[bridge]
socket_path = "/run/armorclaw/bridge.sock"
user = "armorclaw"
log_level = "info"

[matrix]
homeserver_url = "http://localhost:6167"
username = "agent"
password = "CHANGE_ME"
device_id = "ARMORCLAW_BRIDGE"

[keystore]
path = "/var/lib/armorclaw/keystore.db"
EOF

echo -e "${GREEN}✓ Config template created${NC}"
echo ""

# Summary
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}Repository Initialization Complete!${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo "Location: $BRIDGE_DIR"
echo "Module:   $MODULE_PATH"
echo ""
echo "Next steps:"
echo "  1. cd $BRIDGE_DIR"
echo "  2. make deps          # Download dependencies"
echo "  3. make build         # Build bridge binary"
echo "  4. make test          # Run tests"
echo ""
echo "Task 1.1 (Project Setup) complete!"
echo "Ready for Task 1.2 (Configuration System)"
