# OpenClaw Agent - ArmorClaw v1 Integration
"""
ArmorClaw-integrated OpenClaw agent for hardened containers.

This package provides:
- ArmorClawAgent: Main agent class with Matrix integration
- BridgeClient: Client for communicating with ArmorClaw Local Bridge
- AsyncBridgeClient: Async client for async agent operations
"""

from .agent import ArmorClawAgent, main
from .bridge_client import BridgeClient, AsyncBridgeClient, get_default_client

__version__ = "1.0.0-sc"

__all__ = [
    "ArmorClawAgent",
    "BridgeClient",
    "AsyncBridgeClient",
    "get_default_client",
    "main",
]
