#!/usr/bin/env python3
"""
ArmorClaw v1 Container Entrypoint
Security boundary: loads secrets from FD passing, starts agent, never logs secret values
"""
import os
import sys
import subprocess
import json
import socket  # P0-CRIT-3: For socket-based secret loading

# ============================================================================
# Help and Usage
# ============================================================================

def show_help():
    """Display help information."""
    print("""
ArmorClaw v1 - Hardened Container Runtime for AI Agents

USAGE:
    docker run [OPTIONS] armorclaw/agent:v1 [COMMAND]

    By default, starts the ArmorClaw agent. To run a custom command,
    override the CMD: docker run armorclaw/agent:v1 <your-command>

EXAMPLES:
    # Start the agent (requires API key via bridge or -e)
    docker run -e OPENAI_API_KEY=sk-... armorclaw/agent:v1

    # Run a custom command
    docker run -e OPENAI_API_KEY=sk-... armorclaw/agent:v1 python -c "print('hello')"

    # Check container hardening (bypasses agent)
    docker run --rm -e OPENAI_API_KEY=sk-test armorclaw/agent:v1 id

SECRETS INJECTION:
    Production: Use bridge with file descriptor passing
    Testing: Use -e OPENAI_API_KEY=sk-... (or other provider keys)

SUPPORTED PROVIDERS:
    OpenAI: OPENAI_API_KEY
    Anthropic: ANTHROPIC_API_KEY
    OpenRouter: OPENROUTER_API_KEY
    Google/Gemini: GOOGLE_API_KEY or GEMINI_API_KEY
    xAI: XAI_API_KEY

SECURITY:
    - Runs as UID 10001 (non-root)
    - No shell access (/bin/sh has no execute bit)
    - No destructive tools (rm, mv, find, etc.)
    - LD_PRELOAD hooks block dangerous syscalls
    - Read-only root filesystem recommended

For more information: https://github.com/armorclaw/armorclaw
""")
    sys.exit(0)

# Check for --help flag before any validation
if '--help' in sys.argv or '-h' in sys.argv:
    show_help()

# Check for --version flag
if '--version' in sys.argv or '-v' in sys.argv:
    print("ArmorClaw v1.0.0")
    print("Hardened container runtime for AI agents")
    print("Build: debian:bookworm-slim")
    sys.exit(0)

# ============================================================================
# Secrets Loading (File Descriptor Passing)
# ============================================================================

def load_secrets_from_socket() -> dict:
    """
    Load secrets from Unix domain socket (P0-CRIT-3).

    The bridge creates a Unix socket at ARMORCLAW_SECRET_SOCKET for
    memory-only secret delivery. We connect to it and receive credentials.

    Returns:
        dict: Loaded secrets with keys: provider, token, display_name
    """
    secret_socket_path = os.getenv('ARMORCLAW_SECRET_SOCKET', '/run/secrets/socket')

    if not os.path.exists(secret_socket_path):
        # Socket doesn't exist, skip to file-based fallback
        return None

    try:
        print(f"[ArmorClaw] Connecting to secret socket: {secret_socket_path}")
        sock = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
        sock.settimeout(5)  # 5 second timeout
        sock.connect(secret_socket_path)

        # Read message length prefix (4 bytes, big-endian)
        length_data = b''
        while len(length_data) < 4:
            chunk = sock.recv(4 - len(length_data))
            if not chunk:
                print(f"[ArmorClaw] ✗ ERROR: Failed to read length prefix from socket", file=sys.stderr)
                sock.close()
                return None
            length_data += chunk

        # Parse length
        msg_length = (length_data[0] << 24) | (length_data[1] << 16) | (length_data[2] << 8) | length_data[3]

        # Read secrets data
        secrets_data = b''
        while len(secrets_data) < msg_length:
            chunk = sock.recv(min(4096, msg_length - len(secrets_data)))
            if not chunk:
                print(f"[ArmorClaw] ✗ ERROR: Failed to read secrets data from socket", file=sys.stderr)
                sock.close()
                return None
            secrets_data += chunk

        sock.close()

        # Parse JSON
        secrets = json.loads(secrets_data.decode('utf-8'))

        # Validate structure
        if not secrets.get('provider') or not secrets.get('token'):
            print(f"[ArmorClaw] ✗ ERROR: Invalid secrets structure from socket", file=sys.stderr)
            return None

        print(f"[ArmorClaw] ✓ Secrets loaded from socket (P0-CRIT-3: memory-only)")
        return secrets

    except FileNotFoundError:
        print(f"[ArmorClaw] ✗ ERROR: Secret socket not found: {secret_socket_path}", file=sys.stderr)
        return None
    except socket.timeout:
        print(f"[ArmorClaw] ✗ ERROR: Timeout connecting to secret socket", file=sys.stderr)
        return None
    except json.JSONDecodeError as e:
        print(f"[ArmorClaw] ✗ ERROR: Invalid JSON in socket data: {e}", file=sys.stderr)
        return None
    except Exception as e:
        print(f"[ArmorClaw] ⚠️ Failed to load secrets from socket: {e}", file=sys.stderr)
        return None

def validate_secrets(secrets: dict) -> bool:
    """
    Validate the structure of loaded secrets.

    Args:
        secrets: Dict to validate

    Returns:
        bool: True if secrets structure is valid
    """
    if not secrets:
        return False

    required_fields = ['provider', 'token']
    for field in required_fields:
        if field not in secrets:
            print(f"[ArmorClaw] ✗ ERROR: Missing required field in secrets: {field}", file=sys.stderr)
            return False

    # Validate token is not empty
    if not secrets.get('token'):
        print(f"[ArmorClaw] ✗ ERROR: Token is empty", file=sys.stderr)
        return False

    return True

def load_secrets_from_bridge() -> dict:
    """
    Load secrets from the bridge-provided file descriptor or pipe.

    The bridge creates a file at /run/secrets containing the
    secrets JSON. We read from it and set environment variables.

    Returns:
        dict: Loaded secrets with keys: provider, token, display_name
    """
    # Check for SECRETS_PATH environment variable (bridge-mounted file)
    secrets_path = os.getenv('ARMORCLAW_SECRETS_PATH', '/run/secrets')

    secrets = None

    # Try reading from pipe/file first
    # Need to check if it's a file (not just a directory)
    if os.path.isfile(secrets_path):
        try:
            print(f"[ArmorClaw] Loading secrets from {secrets_path}")
            with open(secrets_path, 'r') as f:
                secrets = json.load(f)

            # Validate secrets structure
            if not validate_secrets(secrets):
                print("[ArmorClaw] ✗ ERROR: Invalid secrets structure", file=sys.stderr)
                print("[ArmorClaw] Secrets must contain 'provider' and 'token' fields", file=sys.stderr)
                return None

            print("[ArmorClaw] ✓ Secrets loaded from bridge")
        except json.JSONDecodeError as e:
            print(f"[ArmorClaw] ✗ ERROR: Invalid JSON in secrets file: {e}", file=sys.stderr)
            return None
        except Exception as e:
            print(f"[ArmorClaw] ⚠ Failed to load secrets from file: {e}", file=sys.stderr)
    elif os.path.isdir(secrets_path):
        # Directory exists but no file - bridge might not have created it yet
        print(f"[ArmorClaw] ⚠ Secrets directory exists but no file found at {secrets_path}", file=sys.stderr)

    # If no secrets from file, try environment variables (for testing)
    if not secrets:
        env_vars = ['OPENAI_API_KEY', 'ANTHROPIC_API_KEY', 'OPENROUTER_API_KEY',
                    'GOOGLE_API_KEY', 'GEMINI_API_KEY', 'XAI_API_KEY']
        for var in env_vars:
            if os.getenv(var):
                print(f"[ArmorClaw] ⚠ Using environment variable {var} (testing mode)")
                # Don't return secrets from env vars - just note they exist
                break

    return secrets

def apply_secrets(secrets: dict) -> bool:
    """
    Apply loaded secrets to the environment.

    Args:
        secrets: Dict with provider, token, and optional display_name

    Returns:
        bool: True if secrets were applied successfully
    """
    if not secrets or 'token' not in secrets:
        return False

    provider = secrets.get('provider', 'unknown').lower()
    token = secrets['token']

    # Map provider to environment variable name
    provider_env_map = {
        'openai': 'OPENAI_API_KEY',
        'anthropic': 'ANTHROPIC_API_KEY',
        'openrouter': 'OPENROUTER_API_KEY',
        'google': 'GOOGLE_API_KEY',
        'gemini': 'GEMINI_API_KEY',
        'xai': 'XAI_API_KEY',
        'slack': 'SLACK_BOT_TOKEN',
        'discord': 'DISCORD_BOT_TOKEN',
        'teams': 'MICROSOFT_API_KEY',
        'whatsapp': 'WHATSAPP_API_KEY',
    }

    env_var = provider_env_map.get(provider)
    if env_var:
        os.environ[env_var] = token
        os.environ['ARMORCLAW_PROVIDER'] = provider  # Set provider for agent
        print(f"[ArmorClaw] ✓ {provider} API key loaded from bridge")
        return True
    else:
        print(f"[ArmorClaw] ⚠ Unknown provider: {provider}", file=sys.stderr)
        return False

# ============================================================================
# Secrets Verification (Fail-Fast)
# ============================================================================

# Try to load secrets from bridge first
bridge_secrets = load_secrets_from_bridge()
if bridge_secrets:
    apply_secrets(bridge_secrets)

# Check for API keys (either from bridge or environment)
secrets_present = False

if os.getenv('OPENAI_API_KEY'):
    print("[ArmorClaw] ✓ OpenAI API key present")
    secrets_present = True

if os.getenv('ANTHROPIC_API_KEY'):
    print("[ArmorClaw] ✓ Anthropic API key present")
    secrets_present = True

if os.getenv('OPENROUTER_API_KEY'):
    print("[ArmorClaw] ✓ OpenRouter API key present")
    secrets_present = True

if os.getenv('GOOGLE_API_KEY') or os.getenv('GEMINI_API_KEY'):
    print("[ArmorClaw] ✓ Google/Gemini API key present")
    secrets_present = True

if os.getenv('XAI_API_KEY'):
    print("[ArmorClaw] ✓ xAI API key present")
    secrets_present = True

# Fail if no secrets detected
if not secrets_present:
    print("[ArmorClaw] ✗ ERROR: No API keys detected", file=sys.stderr)
    print("[ArmorClaw] Container cannot start without credentials", file=sys.stderr)
    print("[ArmorClaw]", file=sys.stderr)
    print("[ArmorClaw] To inject secrets, start container via bridge:", file=sys.stderr)
    print('[ArmorClaw]   echo \'{"method":"start","params":{"key_id":"..."}}\' | socat - UNIX-CONNECT:/run/armorclaw/bridge.sock', file=sys.stderr)
    print("[ArmorClaw]", file=sys.stderr)
    print("[ArmorClaw] For testing only, use: docker run -e OPENAI_API_KEY=sk-... armorclaw/agent:v1", file=sys.stderr)
    sys.exit(1)

# ============================================================================
# Egress Proxy Configuration (HTTP_PROXY)
# ============================================================================

def configure_proxy() -> bool:
    """
    Configure HTTP proxy for SDTW adapter outbound requests.

    SDTW adapters (Slack, Discord, Teams, WhatsApp) need to make
    HTTPS requests to external platform APIs. This function configures
    the HTTP proxy to enable that outbound traffic.

    Returns:
        bool: True if proxy was configured successfully
    """
    # Check for HTTP_PROXY environment variable (set by bridge)
    http_proxy = os.getenv('HTTP_PROXY')

    if not http_proxy:
        # No proxy configured - direct access (will fail in isolated containers)
        print("[ArmorClaw] ℹ No HTTP_PROXY configured - SDTW adapters may fail")
        return False

    # Validate proxy URL format
    if not http_proxy.startswith('http://') and not http_proxy.startswith('https://'):
        print(f"[ArmorClaw] ⚠ WARNING: Invalid HTTP_PROXY format: {http_proxy}", file=sys.stderr)
        return False

    # Set proxy environment variables for Python HTTP clients
    # Most Python HTTP libraries (requests, urllib3, httpx) respect these
    os.environ['HTTP_PROXY'] = http_proxy
    os.environ['HTTPS_PROXY'] = http_proxy  # For HTTPS requests
    os.environ['http_proxy'] = http_proxy  # Lowercase version
    os.environ['https_proxy'] = http_proxy  # Lowercase version

    # Disable proxy for localhost connections (if any)
    os.environ['NO_PROXY'] = 'localhost,127.0.0.1'

    print(f"[ArmorClaw] ✓ Egress proxy configured: {http_proxy}")
    print("[ArmorClaw] ℹ SDTW adapters will use proxy for outbound HTTPS requests")

    return True

# Configure egress proxy before agent starts
configure_proxy()

# ============================================================================
# Security: Verify Hardening (Self-Check)
# ============================================================================

# Verify we're running as non-root (UID 10001)
try:
    current_uid = os.getuid()
    if current_uid != 10001:
        print(f"[ArmorClaw] ✗ WARNING: Not running as UID 10001 (current: {current_uid})", file=sys.stderr)
except AttributeError:
    # Windows doesn't have os.getuid, but container is Linux
    pass

# ============================================================================
# Secrets Hygiene (Cleanup After Agent Inherits)
# ============================================================================
# NOTE: We NO LONGER unset environment variables here.
# The agent process (started via os.execv below) will inherit them.
# Once the agent process is running, we can't clear these from /proc/self/environ
# but that's acceptable since the container is isolated.

# ============================================================================
# Health Check and Validation
# ============================================================================

def validate_agent_startup(cmd):
    """
    Validate that the agent command is executable before attempting exec.
    This provides early failure with clear error messages.

    Args:
        cmd: Command list to execute

    Returns:
        tuple: (is_valid, error_message)
    """
    import shutil

    if not cmd or not cmd[0]:
        return False, "Empty command specified"

    # Check if command exists
    cmd_path = shutil.which(cmd[0])
    if not cmd_path:
        # Provide helpful error message
        print(f"[ArmorClaw] ✗ ERROR: Command not found: {cmd[0]}", file=sys.stderr)
        print(f"[ArmorClaw] Searched in PATH: {os.environ.get('PATH', '')}", file=sys.stderr)
        return False, f"Command '{cmd[0]}' not found in PATH"

    # Check if command is executable
    if not os.access(cmd_path, os.X_OK):
        return False, f"Command '{cmd[0]}' is not executable"

    # For Python commands, verify Python is available
    # Note: Skip subprocess version check in hardened environment where /usr/bin/env
    # may not be executable. shutil.which() already confirmed Python exists.
    if cmd[0] in ['python', 'python3', 'python3.11']:
        # Verify we can access the Python binary directly
        if not os.access(cmd_path, os.R_OK):
            return False, f"{cmd[0]} is not readable"
        # Skip version check to avoid /usr/bin/env dependency

    return True, None

def check_agent_readiness():
    """
    Check if the agent environment is ready for startup.
    This is called before attempting to start the agent.

    Returns:
        bool: True if environment is ready
    """
    # Check for required environment variables
    required_vars = []
    for var in ['OPENAI_API_KEY', 'ANTHROPIC_API_KEY', 'OPENROUTER_API_KEY',
                'GOOGLE_API_KEY', 'GEMINI_API_KEY', 'XAI_API_KEY']:
        if os.getenv(var):
            required_vars.append(var)

    if not required_vars:
        print("[ArmorClaw] ⚠ WARNING: No API keys detected - agent may fail", file=sys.stderr)

    # Check available memory (basic sanity check)
    try:
        with open('/proc/meminfo', 'r') as f:
            meminfo = f.read()
            # Parse MemAvailable (or MemTotal if MemAvailable not present)
            for line in meminfo.split('\n'):
                if line.startswith('MemAvailable:') or line.startswith('MemTotal:'):
                    parts = line.split()
                    if len(parts) >= 2:
                        kb = int(parts[1])
                        mb = kb // 1024
                        if mb < 128:  # Less than 128MB available
                            print(f"[ArmorClaw] ⚠ WARNING: Low memory available ({mb}MB)", file=sys.stderr)
                        break
    except (FileNotFoundError, ValueError, IndexError):
        pass  # Meminfo not available (non-Linux or read error)

    return True

# ============================================================================
# Start OpenClaw Agent
# ============================================================================

print("[ArmorClaw] Starting OpenClaw agent...")

# Get the command to run from CMD or use default
if len(sys.argv) > 1:
    cmd = sys.argv[1:]
else:
    # Default: start ArmorClaw agent (direct Python import)
    cmd = ['python', '-c', 'from openclaw import main; main()']

# Pre-flight validation
print(f"[ArmorClaw] Validating agent startup: {' '.join(cmd[:2])}...")
is_valid, error_msg = validate_agent_startup(cmd)
if not is_valid:
    print(f"[ArmorClaw] ✗ ERROR: Agent validation failed: {error_msg}", file=sys.stderr)
    print(f"[ArmorClaw] Container cannot start without the agent", file=sys.stderr)
    print(f"[ArmorClaw]", file=sys.stderr)
    print(f"[ArmorClaw] This may indicate:", file=sys.stderr)
    print(f"[ArmorClaw]   1. The agent module is not installed", file=sys.stderr)
    print(f"[ArmorClaw]   2. The container image is incomplete", file=sys.stderr)
    print(f"[ArmorClaw]   3. A build or installation issue", file=sys.stderr)
    print(f"[ArmorClaw]", file=sys.stderr)
    print(f"[ArmorClaw] For testing, you can override the command:", file=sys.stderr)
    print(f'[ArmorClaw]   docker run --rm -e OPENAI_API_KEY=sk-... {cmd[0]} <your-command>', file=sys.stderr)
    sys.exit(127)  # 127 = command not found

# Check environment readiness
if not check_agent_readiness():
    print("[ArmorClaw] ⚠ WARNING: Environment checks failed", file=sys.stderr)
    print("[ArmorClaw] Agent may not function correctly", file=sys.stderr)

print("[ArmorClaw] ✓ Agent validation passed, starting...")

# Use exec to replace Python with agent process (PID 1)
# This ensures signals are handled correctly and environment is inherited
# Note: os.execv does not return on success - the current process is replaced
# Timeout handling is the responsibility of the container orchestrator (Docker)
try:
    # Try to find the command (already validated above)
    import shutil
    cmd_path = shutil.which(cmd[0])
    if cmd_path:
        # Set whitelist flag to allow security hook to pass this execve call
        os.environ['ARMORCLAW_ALLOW_EXEC'] = '1'
        os.execv(cmd_path, cmd)
    else:
        # Command not found, try execvp as fallback
        os.environ['ARMORCLAW_ALLOW_EXEC'] = '1'
        os.execvp(cmd[0], cmd)
except (FileNotFoundError, OSError) as e:
    # This should not happen after validation, but handle it anyway
    print(f"[ArmorClaw] ✗ ERROR: Unexpected error during exec: {e}", file=sys.stderr)
    print(f"[ArmorClaw] Command: {' '.join(cmd)}", file=sys.stderr)
    sys.exit(1)
except Exception as e:
    # Catch-all for any other errors
    print(f"[ArmorClaw] ✗ ERROR: Failed to start agent: {e}", file=sys.stderr)
    sys.exit(1)

# This line should never be reached due to exec
print("[ArmorClaw] ✗ ERROR: exec returned unexpectedly", file=sys.stderr)
sys.exit(1)
