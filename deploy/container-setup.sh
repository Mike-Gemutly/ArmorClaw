#!/bin/bash
# ArmorClaw Container Setup Wizard
# Simplified setup for Docker container deployment
# Version: 1.0.0

# NOTE: Do NOT use set -e here. Transient failures (curl timeouts, docker pulls)
# must not kill the setup wizard. Critical commands check errors explicitly.

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

# Paths
CONFIG_DIR="/etc/armorclaw"
DATA_DIR="/var/lib/armorclaw"
RUN_DIR="/run/armorclaw"
CONFIG_FILE="$CONFIG_DIR/config.toml"
SETUP_FLAG="$CONFIG_DIR/.setup_complete"

#=============================================================================
# Helper Functions
#=============================================================================

print_header() {
    echo -e "${CYAN}"
    echo "╔══════════════════════════════════════════════════════╗"
    echo "║        ${BOLD}ArmorClaw Container Setup${NC}${CYAN}                     ║"
    echo "║        ${BOLD}Version 1.0.0${NC}${CYAN}                                  ║"
    echo "╚══════════════════════════════════════════════════════╝"
    echo -e "${NC}"
}

print_step() {
    local step="$1"
    local total="$2"
    echo -e "\n${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BLUE}  Step $step of $total: ${BOLD}$3${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}\n"
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ ERROR: $1${NC}" >&2
}

print_warning() {
    echo -e "${YELLOW}⚠ WARNING: $1${NC}"
}

print_info() {
    echo -e "${CYAN}ℹ $1${NC}"
}

prompt_input() {
    local prompt="$1"
    local default="$2"
    local result

    if [ -n "$default" ]; then
        echo -ne "${CYAN}$prompt [$default]: ${NC}"
    else
        echo -ne "${CYAN}$prompt: ${NC}"
    fi

    read -r result
    # Strip carriage returns and trailing whitespace (handles CRLF from terminals)
    result="${result%$'\r'}"
    result="${result%"${result##*[![:space:]]}"}"
    echo "${result:-$default}"
}

prompt_yes_no() {
    local prompt="$1"
    local default="${2:-n}"

    while true; do
        if [ "$default" = "y" ]; then
            echo -ne "${CYAN}$prompt [Y/n]: ${NC}"
        else
            echo -ne "${CYAN}$prompt [y/N]: ${NC}"
        fi

        read -r response
        response=${response:-$default}

        case "$response" in
            [Yy]|[Yy][Ee][Ss]) return 0 ;;
            [Nn]|[Nn][Oo]) return 1 ;;
        esac

        echo -e "${YELLOW}Please answer yes or no.${NC}"
    done
}

#=============================================================================
# Environment Variable Support
#=============================================================================

# Check for environment variables (non-interactive mode)
check_env_vars() {
    # Check for minimal required env vars for non-interactive mode
    # Only SERVER_NAME and API_KEY are required - everything else has defaults
    if [ -n "$ARMORCLAW_SERVER_NAME" ] || [ -n "$ARMORCLAW_API_KEY" ]; then
        print_info "Environment variables detected - using non-interactive mode"
        NON_INTERACTIVE=true

        # Server name - auto-detect if not provided
        SERVER_NAME="${ARMORCLAW_SERVER_NAME:-$(curl -s --connect-timeout 5 ifconfig.me 2>/dev/null || hostname -I | awk '{print $1}')}"

        # Matrix config - use internal by default
        MATRIX_SERVER="${ARMORCLAW_MATRIX_SERVER:-$SERVER_NAME:6167}"
        MATRIX_URL="${ARMORCLAW_MATRIX_URL:-http://localhost:6167}"

        # API config
        API_KEY="${ARMORCLAW_API_KEY:-}"
        API_BASE_URL="${ARMORCLAW_API_BASE_URL:-https://api.openai.com/v1}"
        # Detect provider from base URL
        case "$API_BASE_URL" in
            *anthropic*) API_PROVIDER="anthropic" ;;
            *) API_PROVIDER="openai" ;;
        esac

        # Bridge config
        BRIDGE_PASSWORD="${ARMORCLAW_BRIDGE_PASSWORD:-$(openssl rand -base64 16 2>/dev/null | tr -d '/+=' || echo 'bridge123')}"
        LOG_LEVEL="${ARMORCLAW_LOG_LEVEL:-info}"
        SECURITY_TIER="${ARMORCLAW_SECURITY_TIER:-enhanced}"
        SOCKET_PATH="${ARMORCLAW_SOCKET_PATH:-/run/armorclaw/bridge.sock}"
    else
        NON_INTERACTIVE=false
    fi
}

#=============================================================================
# Setup Functions
#=============================================================================

create_directories() {
    print_info "Creating directory structure..."

    mkdir -p "$CONFIG_DIR"
    mkdir -p "$DATA_DIR"
    mkdir -p "$RUN_DIR"
    mkdir -p "/var/log/armorclaw"
    mkdir -p "$CONFIG_DIR/ssl"

    print_success "Directories created"
}

generate_self_signed_cert() {
    print_info "Generating self-signed SSL certificate..."

    local ssl_dir="$CONFIG_DIR/ssl"
    local ip_address=$(curl -s --connect-timeout 5 ifconfig.me 2>/dev/null || hostname -I | awk '{print $1}')

    openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
        -keyout "$ssl_dir/key.pem" \
        -out "$ssl_dir/cert.pem" \
        -subj "/CN=$ip_address" \
        -addext "subjectAltName=IP:$ip_address" 2>/dev/null

    chmod 600 "$ssl_dir/key.pem"
    chmod 644 "$ssl_dir/cert.pem"

    print_success "Self-signed certificate generated for $ip_address"
    print_warning "Note: Browsers will show security warnings. Ask the agent about ngrok or Cloudflare for trusted SSL."
}

check_docker_socket() {
    print_info "Checking Docker socket..."

    if [ ! -S /var/run/docker.sock ]; then
        print_error "Docker socket not found at /var/run/docker.sock"
        print_error "This container requires Docker socket access."
        echo ""
        echo "Run with: docker run -v /var/run/docker.sock:/var/run/docker.sock ..."
        exit 1
    fi

    # Test Docker access
    if ! docker info >/dev/null 2>&1; then
        print_error "Cannot connect to Docker daemon"
        print_error "Ensure your user has Docker permissions"
        exit 1
    fi

    print_success "Docker socket accessible"
}

configure_matrix() {
    if [ "$NON_INTERACTIVE" = true ]; then
        print_info "Using environment variables for Matrix configuration"
    else
        print_step 1 5 "Matrix Homeserver Configuration"

        echo "Enter your Matrix server details:"
        echo "  - For domain: matrix.example.com"
        echo "  - For IP-only: YOUR_VPS_IP (e.g., 192.168.1.100)"
        echo ""

        while true; do
            MATRIX_SERVER=$(prompt_input "Matrix server (domain or IP)" "")
            if [ -n "$MATRIX_SERVER" ]; then
                break
            fi
            print_warning "Matrix server is required. Please enter a domain or IP address."
        done

        # Auto-detect if IP address
        if echo "$MATRIX_SERVER" | grep -qE '^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$'; then
            print_info "Detected IP address mode - using HTTP"
            MATRIX_URL="http://localhost:6167"
            MATRIX_SERVER="$MATRIX_SERVER:8448"
        else
            MATRIX_URL=$(prompt_input "Matrix homeserver URL" "http://localhost:6167")
        fi

        echo ""
        if prompt_yes_no "Create bridge user on Matrix?" "y"; then
            BRIDGE_USER=$(prompt_input "Bridge username" "bridge")
            BRIDGE_PASSWORD=$(prompt_input "Bridge password" "")
            if [ -z "$BRIDGE_PASSWORD" ]; then
                # Generate random password
                BRIDGE_PASSWORD=$(openssl rand -base64 16 2>/dev/null || cat /dev/urandom | tr -dc 'a-zA-Z0-9' | fold -w 16 | head -n 1)
                print_info "Generated password: $BRIDGE_PASSWORD"
            fi
        else
            BRIDGE_USER="bridge"
            BRIDGE_PASSWORD="bridge123"
        fi
    fi

    print_success "Matrix configuration complete"
}

configure_api() {
    if [ "$NON_INTERACTIVE" = true ]; then
        print_info "Using environment variables for API configuration"
    else
        print_step 2 5 "AI Provider Configuration"

        echo "Select your AI provider:"
        echo "  1) OpenAI"
        echo "  2) Anthropic (Claude)"
        echo "  3) GLM-5 (Zhipu AI)"
        echo "  4) Custom (OpenAI-compatible)"
        echo ""

        while true; do
            local choice=$(prompt_input "Provider" "1")

            case "$choice" in
                1)
                    API_BASE_URL="https://api.openai.com/v1"
                    API_PROVIDER="openai"
                    print_info "Selected: OpenAI"
                    break
                    ;;
                2)
                    API_BASE_URL="https://api.anthropic.com/v1"
                    API_PROVIDER="anthropic"
                    print_info "Selected: Anthropic (Claude)"
                    break
                    ;;
                3)
                    API_BASE_URL="https://api.z.ai/api/coding/paas/v4"
                    API_PROVIDER="openai"
                    print_info "Selected: GLM-5 (Zhipu AI)"
                    break
                    ;;
                4)
                    while true; do
                        API_BASE_URL=$(prompt_input "API base URL" "")
                        if [ -n "$API_BASE_URL" ]; then
                            break
                        fi
                        print_warning "API base URL is required for custom providers."
                    done
                    API_PROVIDER="openai"
                    print_info "Selected: Custom ($API_BASE_URL)"
                    break
                    ;;
                *)
                    print_warning "Invalid choice '$choice'. Please enter 1, 2, 3, or 4."
                    ;;
            esac
        done

        while true; do
            API_KEY=$(prompt_input "API key" "")
            if [ -z "$API_KEY" ]; then
                print_warning "API key is required."
                continue
            fi
            if [ ${#API_KEY} -lt 20 ]; then
                print_warning "API key appears too short (minimum 20 characters)."
                print_info "OpenAI keys start with 'sk-', Anthropic keys start with 'sk-ant-'"
                continue
            fi
            break
        done
    fi

    print_success "API configuration complete"
}

configure_bridge() {
    if [ "$NON_INTERACTIVE" = true ]; then
        print_info "Using environment variables for bridge configuration"
    else
        print_step 3 5 "Bridge Configuration"

        echo "Configure the ArmorClaw bridge settings:"
        echo ""

        LOG_LEVEL=$(prompt_input "Log level (debug/info/warn)" "info")
        SOCKET_PATH=$(prompt_input "Bridge socket path" "/run/armorclaw/bridge.sock")

        echo ""
        print_info "Log level: $LOG_LEVEL"
        print_info "Socket path: $SOCKET_PATH"
    fi

    print_success "Bridge configuration complete"
}

write_config() {
    print_step 4 5 "Writing Configuration"

    print_info "Creating config.toml..."

    cat > "$CONFIG_FILE" << EOF
# ArmorClaw Bridge Configuration
# Generated by container-setup.sh on $(date -Iseconds)

[server]
socket_path = "${SOCKET_PATH:-/run/armorclaw/bridge.sock}"
pid_file = "/run/armorclaw/bridge.pid"
daemonize = false

[keystore]
db_path = "${DATA_DIR}/keystore.db"
master_key = ""
providers = []

[matrix]
enabled = true
homeserver_url = "${MATRIX_URL}"
username = "${BRIDGE_USER:-bridge}"
password = "${BRIDGE_PASSWORD}"
device_id = "armorclaw-bridge"
sync_interval = 5
auto_rooms = []

[matrix.retry]
max_retries = 3
retry_delay = 5
backoff_multiplier = 2.0

[matrix.zero_trust]
trusted_senders = []
trusted_rooms = []
reject_untrusted = false

[budget]
daily_limit_usd = 5.0
monthly_limit_usd = 100.0
alert_threshold = 80.0
hard_stop = true

[logging]
level = "${LOG_LEVEL:-info}"
format = "json"
output = "stdout"

[discovery]
enabled = true
port = 8080
tls = false
api_path = "/api"
ws_path = "/ws"

[notifications]
enabled = false
alert_threshold = 0.8

[eventbus]
websocket_enabled = false
websocket_addr = "0.0.0.0:8444"
websocket_path = "/events"
max_subscribers = 100
inactivity_timeout = "30m"

[errors]
enabled = true
store_enabled = true
notify_enabled = true
store_path = "${DATA_DIR}/errors.db"
retention_days = 30
rate_limit_window = "5m"
retention_period = "24h"

[compliance]
enabled = $([ "$SECURITY_TIER" = "maximum" ] && echo "true" || echo "false")
streaming_mode = $([ "$SECURITY_TIER" = "maximum" ] && echo "false" || echo "true")
quarantine_enabled = $([ "$SECURITY_TIER" = "maximum" ] && echo "true" || echo "false")
notify_on_quarantine = true
audit_enabled = $([ "$SECURITY_TIER" = "maximum" ] && echo "true" || echo "false")
audit_retention_days = 90
tier = "${SECURITY_TIER:-essential}"

[provisioning]
signing_secret = "$(openssl rand -hex 32)"
default_expiry_seconds = 60
max_expiry_seconds = 300
one_time_use = true
data_dir = "${DATA_DIR}"
EOF

    chmod 600 "$CONFIG_FILE"
    print_success "Configuration written to $CONFIG_FILE"
}

initialize_keystore() {
    print_step 5 5 "Initializing Keystore"

    print_info "Keystore will be initialized by bridge on first run..."

    # Create keystore directory
    mkdir -p "$DATA_DIR"

    # Store API key temporarily for later injection
    # The bridge will initialize its own SQLCipher keystore
    print_info "Storing API key for bridge injection..."

    # Create a temp file with API key for the bridge to read on startup
    # Use printf to avoid shell expansion of special chars in API keys
    printf 'api_key=%s\nbase_url=%s\nprovider=%s\n' "$API_KEY" "$API_BASE_URL" "${API_PROVIDER:-openai}" > "$DATA_DIR/.api_key_temp"
    chmod 600 "$DATA_DIR/.api_key_temp"

    print_success "Keystore prepared (bridge will initialize on startup)"
}

add_api_key_to_bridge() {
    print_info "Adding API key to bridge keystore..."

    # Wait for bridge socket
    local max_attempts=30
    local attempt=0
    while [ ! -S /run/armorclaw/bridge.sock ] && [ $attempt -lt $max_attempts ]; do
        sleep 1
        attempt=$((attempt + 1))
    done

    if [ -S /run/armorclaw/bridge.sock ]; then
        # Add API key via RPC (method: store_key, requires id + provider + token)
        local key_provider="${API_PROVIDER:-openai}"
        echo "{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"store_key\",\"params\":{\"id\":\"${key_provider}-default\",\"provider\":\"${key_provider}\",\"token\":\"$API_KEY\",\"display_name\":\"Default API\"}}" | \
            socat - UNIX-CONNECT:/run/armorclaw/bridge.sock 2>/dev/null || true

        # Clean up temp file
        rm -f "$DATA_DIR/.api_key_temp"

        print_success "API key added to keystore"
    else
        print_warning "Bridge socket not available - API key stored in $DATA_DIR/.api_key_temp"
        print_warning "Add manually with: store_key RPC method"
    fi
}

start_matrix_stack() {
    print_info "Starting Matrix stack..."

    cd /opt/armorclaw

    if [ ! -f "docker-compose.matrix.yml" ]; then
        print_warning "docker-compose.matrix.yml not found - skipping Matrix stack"
        return
    fi

    # Check if Matrix is already running
    if docker ps 2>/dev/null | grep -q "armorclaw-matrix\|matrix-conduit"; then
        print_success "Matrix stack already running"
        return
    fi

    # --- Prepare config files on the HOST filesystem ---
    # When running inside a container with the Docker socket mounted,
    # bind mount paths in compose files resolve on the HOST filesystem.
    local HOST_CONFIGS="/tmp/armorclaw-configs"
    print_info "Copying config files to host ($HOST_CONFIGS)..."
    docker run --rm -v /tmp:/tmp alpine mkdir -p "$HOST_CONFIGS" 2>/dev/null || true

    # Export server name without port for Conduit's federation identity
    export MATRIX_SERVER_NAME="${MATRIX_SERVER%%:*}"

    # Generate a registration shared secret for creating users without open registration.
    # This is written into conduit.toml, used to register bridge + admin users,
    # then removed after registration is complete.
    REGISTRATION_SHARED_SECRET="$(openssl rand -hex 32 2>/dev/null || head -c 64 /dev/urandom | od -An -tx1 | tr -d ' \n')"

    # Dynamically update conduit.toml with the correct server_name and shared secret
    local CONDUIT_TEMPLATE="/opt/armorclaw/configs/conduit.toml"
    local CONDUIT_STAGING="/tmp/conduit.toml.staging"
    if [ -f "$CONDUIT_TEMPLATE" ]; then
        cp "$CONDUIT_TEMPLATE" "$CONDUIT_STAGING"
        sed -i "s|^server_name = .*|server_name = \"${MATRIX_SERVER_NAME}\"|" "$CONDUIT_STAGING"
        sed -i "s|^client = .*|client = \"https://${MATRIX_SERVER_NAME}\"|" "$CONDUIT_STAGING"
        sed -i "s|^server = .*|server = \"https://${MATRIX_SERVER_NAME}:443\"|" "$CONDUIT_STAGING"
        # Append registration_shared_secret for user creation
        echo "" >> "$CONDUIT_STAGING"
        echo "# Temporary: shared secret for user registration (removed after setup)" >> "$CONDUIT_STAGING"
        echo "registration_shared_secret = \"${REGISTRATION_SHARED_SECRET}\"" >> "$CONDUIT_STAGING"
    fi

    # Copy configs to host-accessible path
    for f in conduit.toml turnserver.conf; do
        local src="/opt/armorclaw/configs/$f"
        # Use staged conduit.toml with dynamic values
        if [ "$f" = "conduit.toml" ] && [ -f "$CONDUIT_STAGING" ]; then
            src="$CONDUIT_STAGING"
        fi
        if [ -f "$src" ]; then
            cat "$src" | \
                docker run --rm -i -v /tmp:/tmp alpine sh -c "cat > $HOST_CONFIGS/$f" 2>/dev/null || true
        fi
    done
    rm -f "$CONDUIT_STAGING"

    # Set ARMORCLAW_CONFIGS so compose resolves bind mounts to the host path
    export ARMORCLAW_CONFIGS="$HOST_CONFIGS"

    # Generate a random TURN secret for Coturn authentication
    export TURN_SECRET="$(openssl rand -hex 32 2>/dev/null || head -c 64 /dev/urandom | od -An -tx1 | tr -d ' \n')"

    # --- Start containers ---
    print_info "Starting Matrix containers (this may take a few minutes on first run)..."
    local COMPOSE_OUTPUT
    COMPOSE_OUTPUT=$(docker compose -f docker-compose.matrix.yml up -d 2>&1)
    local COMPOSE_EXIT=$?
    if [ $COMPOSE_EXIT -ne 0 ]; then
        print_error "Docker Compose failed (exit $COMPOSE_EXIT):"
        echo "$COMPOSE_OUTPUT" | tail -20
        print_warning "Matrix stack may not be available. Check 'docker compose logs' for details."
        return
    fi
    print_success "Docker Compose started"

    # --- Wait for Conduit to be healthy ---
    wait_for_conduit
}

wait_for_conduit() {
    print_info "Waiting for Conduit homeserver to be ready..."
    local max_attempts=24  # 24 * 5s = 120s
    local attempt=0

    while [ $attempt -lt $max_attempts ]; do
        # Check if Conduit responds to the versions endpoint
        if curl -sf --connect-timeout 3 "http://localhost:6167/_matrix/client/versions" >/dev/null 2>&1; then
            print_success "Conduit homeserver is ready"
            return 0
        fi

        attempt=$((attempt + 1))
        if [ $((attempt % 4)) -eq 0 ]; then
            print_info "Still waiting for Conduit... (${attempt}/${max_attempts})"
        else
            echo -n "."
        fi
        sleep 5
    done

    echo ""
    print_error "Conduit did not become ready within 120 seconds"
    print_warning "Check Conduit logs: docker logs armorclaw-conduit"
    print_warning "Bridge user registration will be skipped - you may need to run create-matrix-admin.sh manually"
    return 1
}

register_matrix_user() {
    local username="$1"
    local password="$2"
    local is_admin="${3:-false}"
    local server_name="${MATRIX_SERVER_NAME:-${MATRIX_SERVER%%:*}}"

    if [ -z "$REGISTRATION_SHARED_SECRET" ]; then
        print_warning "No registration shared secret available - cannot register $username"
        return 1
    fi

    # Conduit supports Synapse-compatible shared-secret registration:
    # 1. GET /_synapse/admin/v1/register (get nonce)
    # 2. POST /_synapse/admin/v1/register (register with HMAC)

    # Step 1: Get nonce
    local NONCE_RESPONSE
    NONCE_RESPONSE=$(curl -sf --connect-timeout 10 "http://localhost:6167/_synapse/admin/v1/register" 2>/dev/null)
    if [ $? -ne 0 ] || [ -z "$NONCE_RESPONSE" ]; then
        print_warning "Failed to get registration nonce for $username"
        return 1
    fi

    local NONCE
    NONCE=$(echo "$NONCE_RESPONSE" | jq -r '.nonce // empty' 2>/dev/null)
    if [ -z "$NONCE" ]; then
        print_warning "Invalid nonce response for $username"
        return 1
    fi

    # Step 2: Compute HMAC
    # HMAC = HMAC-SHA1(shared_secret, nonce + "\0" + username + "\0" + password + "\0" + admin_flag)
    local admin_flag="notadmin"
    if [ "$is_admin" = "true" ]; then
        admin_flag="admin"
    fi

    local MAC
    MAC=$(printf '%s\0%s\0%s\0%s' "$NONCE" "$username" "$password" "$admin_flag" | \
        openssl dgst -sha1 -hmac "$REGISTRATION_SHARED_SECRET" | awk '{print $NF}')

    if [ -z "$MAC" ]; then
        print_warning "Failed to compute HMAC for $username"
        return 1
    fi

    # Step 3: Register
    local REG_RESPONSE
    REG_RESPONSE=$(curl -sf --connect-timeout 10 -X POST \
        "http://localhost:6167/_synapse/admin/v1/register" \
        -H "Content-Type: application/json" \
        -d "{\"nonce\":\"$NONCE\",\"username\":\"$username\",\"password\":\"$password\",\"admin\":$is_admin,\"mac\":\"$MAC\"}" 2>/dev/null)
    local REG_EXIT=$?

    if [ $REG_EXIT -ne 0 ]; then
        # Check if user already exists
        local ERROR_MSG
        ERROR_MSG=$(echo "$REG_RESPONSE" | jq -r '.error // empty' 2>/dev/null)
        if echo "$ERROR_MSG" | grep -qi "exists\|already"; then
            print_info "User @${username}:${server_name} already exists"
            return 0
        fi
        print_warning "Failed to register $username: $ERROR_MSG"
        return 1
    fi

    local USER_ID
    USER_ID=$(echo "$REG_RESPONSE" | jq -r '.user_id // empty' 2>/dev/null)
    if [ -n "$USER_ID" ]; then
        print_success "Registered Matrix user: $USER_ID"
        return 0
    fi

    print_warning "Registration returned unexpected response for $username"
    return 1
}

configure_admin_user() {
    if [ "$NON_INTERACTIVE" = true ]; then
        ADMIN_USER="${ARMORCLAW_ADMIN_USER:-admin}"
        ADMIN_PASSWORD="${ARMORCLAW_ADMIN_PASSWORD:-}"
        if [ -z "$ADMIN_PASSWORD" ]; then
            ADMIN_PASSWORD=$(openssl rand -base64 16 2>/dev/null | tr -d '/+=' || cat /dev/urandom | tr -dc 'a-zA-Z0-9' | fold -w 16 | head -n 1)
            print_info "Generated admin password: $ADMIN_PASSWORD"
        fi
        return
    fi

    echo ""
    echo -e "${CYAN}Admin User for Element X / ArmorChat${NC}"
    echo "This account is how YOU log in to chat with the bridge."
    echo ""

    ADMIN_USER=$(prompt_input "Admin username" "admin")

    while true; do
        ADMIN_PASSWORD=$(prompt_input "Admin password (min 8 chars)" "")
        if [ -z "$ADMIN_PASSWORD" ]; then
            # Generate a random password
            ADMIN_PASSWORD=$(openssl rand -base64 16 2>/dev/null | tr -d '/+=')
            print_info "Generated admin password: $ADMIN_PASSWORD"
            break
        fi
        if [ ${#ADMIN_PASSWORD} -lt 8 ]; then
            print_warning "Password must be at least 8 characters."
            continue
        fi
        break
    done

    echo ""
    print_info "Admin user: @${ADMIN_USER}:${MATRIX_SERVER_NAME:-${MATRIX_SERVER%%:*}}"
}

register_users() {
    local server_name="${MATRIX_SERVER_NAME:-${MATRIX_SERVER%%:*}}"

    print_info "Registering Matrix users..."

    # Register bridge bot user
    if register_matrix_user "${BRIDGE_USER:-bridge}" "$BRIDGE_PASSWORD" "false"; then
        print_success "Bridge user ready: @${BRIDGE_USER:-bridge}:${server_name}"
    else
        print_warning "Bridge user registration failed - bridge may not connect to Matrix"
        print_warning "Run 'create-matrix-admin.sh ${BRIDGE_USER:-bridge}' manually after setup"
    fi

    # Register admin user (for human to log in via Element X / ArmorChat)
    if register_matrix_user "$ADMIN_USER" "$ADMIN_PASSWORD" "true"; then
        print_success "Admin user ready: @${ADMIN_USER}:${server_name}"
    else
        print_warning "Admin user registration failed"
        print_warning "Run 'create-matrix-admin.sh $ADMIN_USER' manually after setup"
    fi

    # Clean up: Remove registration_shared_secret from Conduit config on host
    # This prevents anyone from registering users after setup
    print_info "Removing registration shared secret..."
    local HOST_CONFIGS="/tmp/armorclaw-configs"
    echo "$(cat /dev/null)" | docker run --rm -i -v /tmp:/tmp alpine sh -c "
        if [ -f '$HOST_CONFIGS/conduit.toml' ]; then
            sed -i '/registration_shared_secret/d' '$HOST_CONFIGS/conduit.toml'
            sed -i '/Temporary.*shared secret/d' '$HOST_CONFIGS/conduit.toml'
        fi
    " 2>/dev/null || true

    # Restart Conduit to pick up the config without shared secret
    docker restart armorclaw-conduit >/dev/null 2>&1 || true
    sleep 3
}

setup_bridge_room() {
    local server_name="${MATRIX_SERVER_NAME:-${MATRIX_SERVER%%:*}}"

    print_info "Setting up bridge management room..."

    # Wait for Conduit to be ready after restart
    local max_attempts=12
    local attempt=0
    while [ $attempt -lt $max_attempts ]; do
        if curl -sf --connect-timeout 3 "http://localhost:6167/_matrix/client/versions" >/dev/null 2>&1; then
            break
        fi
        sleep 2
        attempt=$((attempt + 1))
    done

    # Log in as admin to get access token
    local LOGIN_RESPONSE
    LOGIN_RESPONSE=$(curl -sf --connect-timeout 10 -X POST "http://localhost:6167/_matrix/client/v3/login" \
        -H "Content-Type: application/json" \
        -d "{\"type\":\"m.login.password\",\"user\":\"$ADMIN_USER\",\"password\":\"$ADMIN_PASSWORD\"}" 2>/dev/null)

    if [ -z "$LOGIN_RESPONSE" ]; then
        print_warning "Could not log in as admin — skipping room setup"
        return 1
    fi

    local ADMIN_TOKEN
    ADMIN_TOKEN=$(echo "$LOGIN_RESPONSE" | jq -r '.access_token // empty' 2>/dev/null)
    if [ -z "$ADMIN_TOKEN" ]; then
        print_warning "Admin login failed — skipping room setup"
        return 1
    fi

    # Create the bridge management room.
    # The admin is the creator and automatically gets power level 100.
    local BRIDGE_USER_ID="@${BRIDGE_USER:-bridge}:${server_name}"
    local CREATE_RESPONSE
    CREATE_RESPONSE=$(curl -sf --connect-timeout 10 -X POST "http://localhost:6167/_matrix/client/v3/createRoom" \
        -H "Authorization: Bearer $ADMIN_TOKEN" \
        -H "Content-Type: application/json" \
        -d "{
            \"preset\": \"trusted_private_chat\",
            \"name\": \"ArmorClaw Bridge\",
            \"topic\": \"Management room — send !status to check bridge health\",
            \"invite\": [\"$BRIDGE_USER_ID\"],
            \"power_level_content_override\": {
                \"users\": {
                    \"@${ADMIN_USER}:${server_name}\": 100,
                    \"$BRIDGE_USER_ID\": 50
                },
                \"events_default\": 0,
                \"state_default\": 50,
                \"ban\": 100,
                \"kick\": 100,
                \"redact\": 50,
                \"invite\": 50
            },
            \"is_direct\": false
        }" 2>/dev/null)

    local ROOM_ID
    ROOM_ID=$(echo "$CREATE_RESPONSE" | jq -r '.room_id // empty' 2>/dev/null)

    if [ -n "$ROOM_ID" ]; then
        print_success "Bridge room created: $ROOM_ID"
        print_info "Admin @${ADMIN_USER}:${server_name} has power level 100 (admin)"
        print_info "Bridge ${BRIDGE_USER_ID} has power level 50 (moderator)"

        # Update config.toml with the room so the bridge auto-joins
        if [ -f "$CONFIG_FILE" ]; then
            sed -i "s|^auto_rooms = .*|auto_rooms = [\"$ROOM_ID\"]|" "$CONFIG_FILE"
        fi

        BRIDGE_ROOM_ID="$ROOM_ID"
    else
        print_warning "Room creation failed — you can create one manually in Element X"
        BRIDGE_ROOM_ID=""
    fi

    # Log out to clean up the access token
    curl -sf --connect-timeout 5 -X POST "http://localhost:6167/_matrix/client/v3/logout" \
        -H "Authorization: Bearer $ADMIN_TOKEN" >/dev/null 2>&1 || true
}

final_summary() {
    local server_name="${MATRIX_SERVER_NAME:-${MATRIX_SERVER%%:*}}"

    echo ""
    echo -e "${GREEN}╔══════════════════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║${NC}        ${BOLD}Setup Complete!${NC}                                  ${GREEN}║${NC}"
    echo -e "${GREEN}╚══════════════════════════════════════════════════════╝${NC}"
    echo ""
    echo "Configuration:"
    echo "  - Matrix server: $MATRIX_SERVER"
    echo "  - Matrix URL: $MATRIX_URL"
    echo "  - API provider: $API_BASE_URL"
    echo "  - Log level: ${LOG_LEVEL:-info}"
    echo "  - Security tier: ${SECURITY_TIER:-essential}"
    echo ""
    echo "Files:"
    echo "  - Config: $CONFIG_FILE"
    echo "  - Keystore: $DATA_DIR/keystore.db"
    echo "  - SSL cert: $CONFIG_DIR/ssl/cert.pem"
    echo ""

    # Admin credentials (critical for Element X / ArmorChat login)
    echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BOLD}Admin Login (Element X / ArmorChat):${NC}"
    echo -e "  Username:   ${GREEN}@${ADMIN_USER:-admin}:${server_name}${NC}"
    echo -e "  Password:   ${GREEN}${ADMIN_PASSWORD}${NC}"
    echo -e "  Homeserver: ${GREEN}http://${server_name}:6167${NC}"
    echo ""
    echo -e "  ${YELLOW}⚠ Save these credentials now — the password is not stored.${NC}"
    echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""

    # Show bridge room info if created
    if [ -n "$BRIDGE_ROOM_ID" ]; then
        echo -e "${BOLD}Bridge Room:${NC}"
        echo -e "  Room:       ${GREEN}ArmorClaw Bridge${NC}  ($BRIDGE_ROOM_ID)"
        echo -e "  Your role:  ${GREEN}Admin (power level 100)${NC}"
        echo -e "  Bridge:     ${GREEN}Moderator (power level 50)${NC}"
        echo ""
    fi

    # Check if IP-based setup
    if echo "$MATRIX_SERVER" | grep -qE '^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+':; then
        echo "Next steps (IP-based setup):"
        echo "  1. Ensure ports are open: 6167, 8448, 5000, 8443"
        echo "  2. Open Element X or ArmorChat"
        echo "  3. Set homeserver to: http://$server_name:6167"
        echo "  4. Log in with admin credentials above"
        if [ -n "$BRIDGE_ROOM_ID" ]; then
            echo "  5. Open 'ArmorClaw Bridge' room (auto-created)"
            echo "  6. Send '!status' to verify connection"
        else
            echo "  5. Start DM with: @${BRIDGE_USER:-bridge}:$server_name"
            echo "  6. Send '!status' to verify connection"
        fi
        echo ""
        echo "For SSL (ask the agent in ArmorChat):"
        echo "  \"Set up a cloudflare tunnel\" - Free, trusted SSL"
        echo ""
        echo "To add more apps (ArmorTerminal):"
        echo "  \"Install armorterminal\" - Terminal access to agents"
    else
        echo "Next steps:"
        echo "  1. Open Element X or ArmorChat"
        echo "  2. Set homeserver to: https://$MATRIX_SERVER"
        echo "  3. Log in with admin credentials above"
        if [ -n "$BRIDGE_ROOM_ID" ]; then
            echo "  4. Open 'ArmorClaw Bridge' room (auto-created)"
            echo "  5. Send '!status' to verify connection"
        else
            echo "  4. Start DM with: @${BRIDGE_USER:-bridge}:$server_name"
            echo "  5. Send '!status' to verify connection"
        fi
    fi
    echo ""
}

#=============================================================================
# Post-Setup Options
#=============================================================================

configure_security_tier() {
    if [ "$NON_INTERACTIVE" = true ]; then
        SECURITY_TIER="${ARMORCLAW_SECURITY_TIER:-essential}"
        print_info "Security tier: $SECURITY_TIER"
        return
    fi

    echo -e "\n${YELLOW}Security Configuration${NC}"
    echo "Select security tier:"
    echo "  1) Essential - Basic isolation (development/testing)"
    echo "  2) Enhanced  - + Seccomp, network isolation (recommended)"
    echo "  3) Maximum   - + Audit logging, PII scrubbing (production)"
    echo ""

    local choice=$(prompt_input "Security tier" "2")

    case "$choice" in
        1) SECURITY_TIER="essential" ;;
        2) SECURITY_TIER="enhanced" ;;
        3) SECURITY_TIER="maximum" ;;
        *) SECURITY_TIER="enhanced" ;;
    esac

    print_info "Security tier set to: $SECURITY_TIER"
}

offer_post_setup_options() {
    if [ "$NON_INTERACTIVE" = true ]; then
        return
    fi

    echo ""
    echo -e "${CYAN}Post-Setup Options${NC}"
    echo ""

    # Offer to install additional apps
    if prompt_yes_no "Install ArmorTerminal for terminal access?" "n"; then
        print_info "ArmorTerminal will be available after bridge starts"
        print_info "Configure with: RPC URL = http://YOUR_IP:8443/rpc"
        INSTALL_ARMORTERMINAL=true
    fi

    # Offer hardening
    if [ "$SECURITY_TIER" != "essential" ]; then
        print_info "Security hardening ($SECURITY_TIER tier) will be applied on bridge start"
    fi
}

#=============================================================================
# Main
#=============================================================================

main() {
    print_header

    # Check for environment variables
    check_env_vars

    # Create directories
    create_directories

    # Generate self-signed SSL certificate (default)
    generate_self_signed_cert

    # Check Docker socket
    check_docker_socket

    # Run configuration steps
    configure_matrix
    configure_api
    configure_bridge
    configure_security_tier

    # Collect admin credentials (for Element X / ArmorChat login)
    configure_admin_user

    # Persist admin user info for quickstart.sh to auto-claim OWNER role
    echo "${ADMIN_USER}" > "$DATA_DIR/.admin_user"
    echo "${MATRIX_SERVER_NAME:-${MATRIX_SERVER%%:*}}" >> "$DATA_DIR/.admin_user"
    chmod 600 "$DATA_DIR/.admin_user"

    write_config
    initialize_keystore

    # Start Matrix if available
    start_matrix_stack

    # Register bridge + admin users on Conduit (requires Matrix stack running)
    register_users

    # Create bridge management room with admin at power level 100
    # This ensures Element X / ArmorChat users see the room on first login
    setup_bridge_room

    # Offer post-setup options
    offer_post_setup_options

    # Show summary
    final_summary
}

# Run main
main "$@"
