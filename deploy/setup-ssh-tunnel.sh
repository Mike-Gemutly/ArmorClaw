#!/bin/bash
# ArmorClaw SSH Tunnel Setup - Automates Lesson 3 from DEPLOYMENT_LESSONS.md
# Version: 1.0
#
# This script automatically establishes an SSH tunnel for VPS access,
# solving the WSL/Windows PowerShell tunnel visibility issue.
#
# Usage:
#   ./deploy/setup-ssh-tunnel.sh [--check] [--stop] [--local-port PORT] [--remote-port PORT]
#
# Environment Variables:
#   CONNECT_VPS    - SSH command to connect to VPS (e.g., "ssh root@1.2.3.4")
#   VPS_IP         - VPS IP address (alternative to CONNECT_VPS)
#   SSH_USER       - SSH username (default: root)
#   SSH_KEY        - Path to SSH private key

set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

# Default ports
LOCAL_PORT="${ARMORCLAW_LOCAL_PORT:-9000}"
REMOTE_PORT="${ARMORCLAW_REMOTE_PORT:-8081}"
TUNNEL_PID_FILE="/tmp/armorclaw-ssh-tunnel.pid"

# Detect WSL environment
is_wsl() {
    if [[ -f /proc/version ]]; then
        grep -qi microsoft /proc/version
        return $?
    fi
    return 1
}

# Detect if running in WSL but tunnel was opened in Windows
is_windows_tunnel() {
    # Check if port is open but no tunnel process found
    if nc -z 127.0.0.1 "$LOCAL_PORT" 2>/dev/null; then
        if ! pgrep -f "ssh.*-L.*${LOCAL_PORT}" >/dev/null 2>&1; then
            return 0
        fi
    fi
    return 1
}

# Get SSH connection command
get_ssh_cmd() {
    if [[ -n "${CONNECT_VPS:-}" ]]; then
        echo "$CONNECT_VPS"
        return 0
    fi

    if [[ -n "${VPS_IP:-}" ]]; then
        local user="${SSH_USER:-root}"
        local key="${SSH_KEY:+-i $SSH_KEY}"
        echo "ssh $key -o StrictHostKeyChecking=no ${user}@${VPS_IP}"
        return 0
    fi

    echo ""
    return 1
}

# Check if tunnel is active
check_tunnel() {
    if [[ -f "$TUNNEL_PID_FILE" ]]; then
        local pid=$(cat "$TUNNEL_PID_FILE" 2>/dev/null)
        if kill -0 "$pid" 2>/dev/null; then
            if nc -z 127.0.0.1 "$LOCAL_PORT" 2>/dev/null; then
                return 0
            fi
        fi
        rm -f "$TUNNEL_PID_FILE"
    fi
    return 1
}

# Start SSH tunnel
start_tunnel() {
    local ssh_cmd=$(get_ssh_cmd)

    if [[ -z "$ssh_cmd" ]]; then
        echo -e "${RED}ERROR:${NC} No SSH connection configured."
        echo "Set CONNECT_VPS or VPS_IP environment variable."
        echo ""
        echo "Examples:"
        echo "  export CONNECT_VPS='ssh root@5.183.11.149'"
        echo "  export VPS_IP=5.183.11.149"
        return 1
    fi

    if check_tunnel; then
        echo -e "${GREEN}✓${NC} SSH tunnel already active (PID: $(cat $TUNNEL_PID_FILE))"
        echo -e "  Local port: ${LOCAL_PORT} → Remote port: ${REMOTE_PORT}"
        return 0
    fi

    echo -e "${CYAN}Setting up SSH tunnel...${NC}"
    echo -e "  Local:  127.0.0.1:${LOCAL_PORT}"
    echo -e "  Remote: localhost:${REMOTE_PORT}"

    # Extract SSH options from command
    local ssh_opts=""
    if [[ "$ssh_cmd" == *"ssh"* ]]; then
        ssh_opts=$(echo "$ssh_cmd" | sed 's/^ssh//')
    fi

    # Start tunnel in background
    ssh -f -N -L "${LOCAL_PORT}:localhost:${REMOTE_PORT}" \
        -o StrictHostKeyChecking=no \
        -o ExitOnForwardFailure=yes \
        -o ServerAliveInterval=60 \
        -o ServerAliveCountMax=3 \
        $ssh_opts

    local pid=$!
    echo $pid > "$TUNNEL_PID_FILE"

    # Wait for tunnel to be ready
    local wait_count=0
    while [[ $wait_count -lt 10 ]]; do
        if nc -z 127.0.0.1 "$LOCAL_PORT" 2>/dev/null; then
            echo -e "${GREEN}✓${NC} SSH tunnel established (PID: $pid)"
            echo -e "  Bridge RPC available at: http://127.0.0.1:${LOCAL_PORT}"
            return 0
        fi
        sleep 0.5
        ((wait_count++)) || true
    done

    echo -e "${RED}ERROR:${NC} Tunnel failed to establish"
    return 1
}

# Stop SSH tunnel
stop_tunnel() {
    if [[ -f "$TUNNEL_PID_FILE" ]]; then
        local pid=$(cat "$TUNNEL_PID_FILE" 2>/dev/null)
        if kill -0 "$pid" 2>/dev/null; then
            kill "$pid" 2>/dev/null || true
            echo -e "${GREEN}✓${NC} SSH tunnel stopped (PID: $pid)"
        fi
        rm -f "$TUNNEL_PID_FILE"
    else
        # Try to find and kill any armorclaw tunnel
        pkill -f "ssh.*-L.*${LOCAL_PORT}:localhost:${REMOTE_PORT}" 2>/dev/null || true
        echo -e "${YELLOW}No active tunnel found${NC}"
    fi
}

# Verify tunnel connectivity
verify_tunnel() {
    echo -e "${CYAN}Verifying SSH tunnel...${NC}"

    if ! check_tunnel; then
        echo -e "${RED}✗${NC} No active tunnel"
        return 1
    fi

    if ! nc -z 127.0.0.1 "$LOCAL_PORT" 2>/dev/null; then
        echo -e "${RED}✗${NC} Port ${LOCAL_PORT} not responding"
        return 1
    fi

    # Try to reach bridge RPC
    local response
    response=$(curl -s --connect-timeout 5 \
        --unix-socket /dev/stdin \
        -d '{"jsonrpc":"2.0","method":"bridge.ping","id":1}' \
        "http://127.0.0.1:${LOCAL_PORT}" 2>/dev/null) || true

    if [[ -n "$response" ]]; then
        echo -e "${GREEN}✓${NC} Bridge RPC responding"
        return 0
    fi

    echo -e "${YELLOW}⚠${NC} Port open but bridge not responding (may need different verification)"
    return 0
}

# WSL-specific check and warning
check_wsl_issue() {
    if is_wsl && is_windows_tunnel; then
        echo -e "${YELLOW}⚠ WARNING:${NC} Detected tunnel opened in Windows PowerShell"
        echo -e "  WSL cannot see Windows tunnels. Open tunnel inside WSL instead:"
        echo ""
        echo "  $ ssh -L ${LOCAL_PORT}:localhost:${REMOTE_PORT} root@\${VPS_IP}"
        echo ""
        return 1
    fi
    return 0
}

# Main
main() {
    local action="start"

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --check|-c)
                action="check"
                shift
                ;;
            --stop|-s)
                action="stop"
                shift
                ;;
            --verify|-v)
                action="verify"
                shift
                ;;
            --local-port|-l)
                LOCAL_PORT="$2"
                shift 2
                ;;
            --remote-port|-r)
                REMOTE_PORT="$2"
                shift 2
                ;;
            --help|-h)
                echo "ArmorClaw SSH Tunnel Setup"
                echo ""
                echo "Usage: $0 [OPTIONS]"
                echo ""
                echo "Options:"
                echo "  --check, -c        Check if tunnel is active"
                echo "  --stop, -s         Stop active tunnel"
                echo "  --verify, -v       Verify tunnel connectivity"
                echo "  --local-port PORT  Local port (default: 9000)"
                echo "  --remote-port PORT Remote port (default: 8081)"
                echo "  --help, -h         Show this help"
                echo ""
                echo "Environment Variables:"
                echo "  CONNECT_VPS         SSH command (e.g., 'ssh root@1.2.3.4')"
                echo "  VPS_IP              VPS IP address"
                echo "  ARMORCLAW_LOCAL_PORT  Local port (default: 9000)"
                echo "  ARMORCLAW_REMOTE_PORT Remote port (default: 8081)"
                exit 0
                ;;
            *)
                echo "Unknown option: $1"
                exit 1
                ;;
        esac
    done

    case "$action" in
        start)
            check_wsl_issue || exit 1
            start_tunnel
            ;;
        check)
            if check_tunnel; then
                echo -e "${GREEN}✓${NC} Tunnel active (PID: $(cat $TUNNEL_PID_FILE))"
                exit 0
            else
                echo -e "${RED}✗${NC} No active tunnel"
                exit 1
            fi
            ;;
        stop)
            stop_tunnel
            ;;
        verify)
            verify_tunnel
            ;;
    esac
}

main "$@"
