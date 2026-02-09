#!/bin/sh
# ArmorClaw Container Health Check
# Checks if the ArmorClaw agent is running and responsive

set -e

# Check if Python is available
if ! command -v python >/dev/null 2>&1; then
    echo "ERROR: Python not found"
    exit 1
fi

# Try to import and check the agent module
# This will fail if:
# - The module is not installed
# - There are import errors
# - Dependencies are missing
python -c "
import sys
try:
    # Try importing the agent module
    from openclaw import agent
    print('OK: Agent module is importable')
    sys.exit(0)
except ImportError as e:
    print(f'ERROR: Cannot import agent module: {e}')
    sys.exit(1)
except Exception as e:
    print(f'ERROR: Unexpected error: {e}')
    sys.exit(1)
" 2>/dev/null || exit 1

# If we get here, health check passed
exit 0
