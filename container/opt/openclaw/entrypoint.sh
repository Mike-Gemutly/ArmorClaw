#!/bin/bash
# ArmorClaw v1 Container Entrypoint
# Security boundary: verifies secrets, starts agent, never logs secret values
set -euo pipefail

# ============================================================================
# Secrets Verification (Fail-Fast)
# ============================================================================
# Container must have at least one API key to function
# Logs presence only, NEVER logs values

SECRETS_PRESENT="false"

# Check for common API key environment variables (presence check only)
if [ -n "${OPENAI_API_KEY:-}" ]; then
    echo "[ArmorClaw] ✓ OpenAI API key present"
    SECRETS_PRESENT="true"
fi

if [ -n "${ANTHROPIC_API_KEY:-}" ]; then
    echo "[ArmorClaw] ✓ Anthropic API key present"
    SECRETS_PRESENT="true"
fi

if [ -n "${OPENROUTER_API_KEY:-}" ]; then
    echo "[ArmorClaw] ✓ OpenRouter API key present"
    SECRETS_PRESENT="true"
fi

if [ -n "${GOOGLE_API_KEY:-}" ] || [ -n "${GEMINI_API_KEY:-}" ]; then
    echo "[ArmorClaw] ✓ Google/Gemini API key present"
    SECRETS_PRESENT="true"
fi

if [ -n "${XAI_API_KEY:-}" ]; then
    echo "[ArmorClaw] ✓ xAI API key present"
    SECRETS_PRESENT="true"
fi

# Fail if no secrets detected
if [ "$SECRETS_PRESENT" = "false" ]; then
    echo "[ArmorClaw] ✗ ERROR: No API keys detected" >&2
    echo "[ArmorClaw] Container cannot start without credentials" >&2
    echo "[ArmorClaw] Use: docker run -e OPENAI_API_KEY=sk-... armorclaw/agent:v1" >&2
    exit 1
fi

# ============================================================================
# Security: Verify Hardening (Self-Check)
# ============================================================================

# Verify we're running as non-root (UID 10001)
CURRENT_UID=$(id -u)
if [ "$CURRENT_UID" != "10001" ]; then
    echo "[ArmorClaw] ✗ WARNING: Not running as UID 10001 (current: $CURRENT_UID)" >&2
fi

# Verify no shell is available (security check)
if [ -x "/bin/sh" ] || [ -x "/bin/bash" ]; then
    echo "[ArmorClaw] ✗ WARNING: Shell detected in container (security issue)" >&2
fi

# ============================================================================
# Secrets Hygiene (Unset After Verification)
# ============================================================================
# While we can't fully clear them from /proc/self/environ,
# we explicitly unset them from our shell environment after recording presence.
# The agent process will inherit them before this unsetting.

OPENAI_API_KEY=""
ANTHROPIC_API_KEY=""
OPENROUTER_API_KEY=""
GOOGLE_API_KEY=""
GEMINI_API_KEY=""
XAI_API_KEY=""

# ============================================================================
# Start OpenClaw Agent
# ============================================================================

echo "[ArmorClaw] Starting OpenClaw agent..."

# Use exec to replace shell with agent process (PID 1)
# This ensures signals are handled correctly
exec "$@"
