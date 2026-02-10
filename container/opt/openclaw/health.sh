#!/usr/bin/env python3
"""
ArmorClaw Container Health Check
Uses Python since /bin/sh is intentionally removed for security hardening.
"""
import sys

try:
    from openclaw import agent
    print("OK: Agent module is importable")
    sys.exit(0)
except ImportError as e:
    print(f"ERROR: Cannot import agent module: {e}")
    sys.exit(1)
except Exception as e:
    print(f"ERROR: Unexpected error: {e}")
    sys.exit(1)
