#!/usr/bin/env bash
set -euo pipefail

SOCKET_PATH="${ARMORCLAW_SOCKET_PATH:-/run/armorclaw/bridge.sock}"
QUIET=false

while [[ $# -gt 0 ]]; do
    case "$1" in
        --socket)
            SOCKET_PATH="$2"
            shift 2
            ;;
        --quiet|-q)
            QUIET=true
            shift
            ;;
        --help|-h)
            echo "Usage: $0 [--socket PATH] [--quiet]"
            echo ""
            echo "Health check script for ArmorClaw Bridge"
            echo ""
            echo "Options:"
            echo "  --socket PATH   Socket path (default: /run/armorclaw/bridge.sock)"
            echo "  --quiet, -q      Minimal output"
            echo "  --help, -h       Show this help"
            echo ""
            echo "Exit codes:"
            echo "  0  Health check passed (status: healthy)"
            echo "  1  Health check failed (status: degraded or unhealthy)"
            echo "  2  Unable to connect to bridge"
            exit 0
            ;;
        *)
            echo "Unknown option: $1" >&2
            echo "Use --help for usage information" >&2
            exit 2
            ;;
    esac
done

if [[ ! -S "$SOCKET_PATH" ]]; then
    if [[ "$QUIET" != true ]]; then
        echo "ERROR: Socket not found: $SOCKET_PATH" >&2
    fi
    exit 2
fi

REQUEST=$(cat <<EOF
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "health.check"
}
EOF
)

RESPONSE=$(curl --silent --show-error --unix-socket "$SOCKET_PATH" \
    -H "Content-Type: application/json" \
    -d "$REQUEST" \
    http://localhost/ 2>&1) || {
    if [[ "$QUIET" != true ]]; then
        echo "ERROR: Failed to connect to bridge at $SOCKET_PATH" >&2
    fi
    exit 2
}

if [[ "$QUIET" != true ]]; then
    echo "$RESPONSE" | jq . 2>/dev/null || echo "$RESPONSE"
fi

STATUS=$(echo "$RESPONSE" | jq -r '.result.status // empty' 2>/dev/null || true)

if [[ -z "$STATUS" ]]; then
    if [[ "$QUIET" != true ]]; then
        echo "ERROR: Unable to parse health status from response" >&2
    fi
    exit 2
fi

case "$STATUS" in
    healthy)
        if [[ "$QUIET" != true ]]; then
            echo "Health check: PASS"
        fi
        exit 0
        ;;
    degraded|unhealthy)
        if [[ "$QUIET" != true ]]; then
            echo "Health check: FAIL ($STATUS)"
        fi
        exit 1
        ;;
    *)
        if [[ "$QUIET" != true ]]; then
            echo "ERROR: Unknown status: $STATUS" >&2
        fi
        exit 2
        ;;
esac
