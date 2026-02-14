#!/usr/bin/env python3
"""
P0-CRIT-3: Socket-based secret loading for container entrypoint.
This module should be integrated into entrypoint.py to replace/extend load_secrets_from_bridge().
"""

import os
import socket
import json

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
        # Socket doesn't exist, try file-based fallback
        print(f"[ArmorClaw] ⚠ Secret socket not found: {secret_socket_path}", file=sys.stderr)
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


# Integration instructions for entrypoint.py:
# In the load_secrets_from_bridge() function, add at the beginning:
#     # Try P0-CRIT-3: Socket-based secret loading first
#     secrets = load_secrets_from_socket()
#     if secrets:
#         return secrets
#
#     # Fallback to file-based loading (P0-CRIT-1, for testing/development)
#     # ... existing file-based code ...
