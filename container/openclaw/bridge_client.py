#!/usr/bin/env python3
"""
ArmorClaw Bridge Client

Python client for communicating with the ArmorClaw Local Bridge
via Unix domain socket using JSON-RPC 2.0 protocol.
"""

import os
import socket
import json
import asyncio
from typing import Any, Dict, Optional, List
from pathlib import Path


class BridgeClient:
    """
    Client for communicating with the ArmorClaw Local Bridge.

    The bridge runs on the host machine and provides a JSON-RPC 2.0
    interface over a Unix domain socket. This client handles:
    - Socket connection management
    - JSON-RPC request/response formatting
    - Event reception from Matrix
    """

    def __init__(self, socket_path: str = "/run/armorclaw/bridge.sock"):
        """
        Initialize the bridge client.

        Args:
            socket_path: Path to the bridge Unix socket
        """
        self.socket_path = socket_path
        self._request_id = 0

    def _get_next_id(self) -> int:
        """Get the next request ID."""
        self._request_id += 1
        return self._request_id

    def _send_request(self, method: str, params: Optional[Dict] = None) -> Dict:
        """
        Send a JSON-RPC request to the bridge.

        Args:
            method: RPC method name
            params: Method parameters (optional)

        Returns:
            Response dictionary

        Raises:
            ConnectionError: If unable to connect to bridge
            RuntimeError: If RPC call fails
        """
        request = {
            "jsonrpc": "2.0",
            "id": self._get_next_id(),
            "method": method
        }

        if params is not None:
            request["params"] = params

        # Connect to bridge socket
        try:
            sock = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
            sock.connect(self.socket_path)

            # Send request
            request_json = json.dumps(request) + "\n"
            sock.sendall(request_json.encode('utf-8'))

            # Receive response
            response_data = b""
            while True:
                chunk = sock.recv(4096)
                if not chunk:
                    break
                response_data += chunk
                # Try to parse complete JSON
                try:
                    response = json.loads(response_data.decode('utf-8'))
                    break
                except json.JSONDecodeError:
                    # Need more data
                    continue

            sock.close()

        except FileNotFoundError:
            raise ConnectionError(
                f"Bridge socket not found at {self.socket_path}. "
                f"Is the ArmorClaw bridge running?"
            )
        except ConnectionRefusedError:
            raise ConnectionError(
                f"Connection refused to {self.socket_path}. "
                f"Is the ArmorClaw bridge running?"
            )
        except Exception as e:
            raise ConnectionError(f"Failed to connect to bridge: {e}")

        # Check for RPC error
        if "error" in response:
            error = response["error"]
            raise RuntimeError(
                f"RPC error {error.get('code', 'unknown')}: {error.get('message', 'Unknown error')}"
            )

        return response.get("result")

    # ========================================================================
    # Core Methods
    # ========================================================================

    def status(self) -> Dict:
        """
        Get bridge status and container information.

        Returns:
            Status dictionary with keys: version, state, socket, containers, container_ids
        """
        return self._send_request("status")

    def health(self) -> Dict:
        """
        Health check endpoint.

        Returns:
            Health check result
        """
        return self._send_request("health")

    # ========================================================================
    # Container Methods
    # ========================================================================

    def start_container(
        self,
        key_id: str,
        agent_type: str = "openclaw",
        image: str = "armorclaw/agent:v1"
    ) -> Dict:
        """
        Start a new container with injected credentials.

        Args:
            key_id: ID of stored credential to inject
            agent_type: Type of agent to run
            image: Container image to use

        Returns:
            Container info with keys: container_id, container_name, status, endpoint
        """
        params = {
            "key_id": key_id,
            "agent_type": agent_type,
            "image": image
        }
        return self._send_request("start", params)

    def stop_container(self, container_id: str) -> Dict:
        """
        Stop a running container.

        Args:
            container_id: Docker container ID to stop

        Returns:
            Stop result with keys: status, container_id, container_name
        """
        params = {
            "container_id": container_id
        }
        return self._send_request("stop", params)

    # ========================================================================
    # Keystore Methods
    # ========================================================================

    def list_keys(self, provider: Optional[str] = None) -> List[Dict]:
        """
        List stored credentials (optionally filtered by provider).

        Args:
            provider: Filter by provider (openai, anthropic, etc.)

        Returns:
            List of key info dictionaries
        """
        params = {}
        if provider is not None:
            params["provider"] = provider
        return self._send_request("list_keys", params)

    def get_key(self, key_id: str) -> Dict:
        """
        Retrieve a specific credential (decrypts the token).

        Args:
            key_id: Credential ID to retrieve

        Returns:
            Credential dictionary with decrypted token
        """
        params = {
            "id": key_id
        }
        return self._send_request("get_key", params)

    # ========================================================================
    # Matrix Methods
    # ========================================================================

    def matrix_status(self) -> Dict:
        """
        Get Matrix connection status.

        Returns:
            Matrix status with keys: enabled, status, user_id, logged_in
        """
        return self._send_request("matrix.status")

    def matrix_login(self, username: str, password: str) -> Dict:
        """
        Login to Matrix homeserver.

        Args:
            username: Matrix username
            password: Matrix password

        Returns:
            Login result with keys: status, user_id
        """
        params = {
            "username": username,
            "password": password
        }
        return self._send_request("matrix.login", params)

    def matrix_send(
        self,
        room_id: str,
        message: str,
        msgtype: str = "m.text"
    ) -> Dict:
        """
        Send a message to a Matrix room.

        Args:
            room_id: Matrix room ID
            message: Message content
            msgtype: Message type (m.text, m.notice)

        Returns:
            Send result with keys: event_id, room_id
        """
        params = {
            "room_id": room_id,
            "message": message,
            "msgtype": msgtype
        }
        return self._send_request("matrix.send", params)

    def matrix_receive(self, limit: int = 10) -> Dict:
        """
        Receive pending Matrix events.

        Args:
            limit: Maximum number of events to return

        Returns:
            Receive result with keys: events, count
        """
        return self._send_request("matrix.receive")

    # ========================================================================
    # Config Methods
    # ========================================================================

    def attach_config(
        self,
        name: str,
        content: str,
        encoding: str = "raw",
        config_type: str = "",
        metadata: Optional[Dict[str, str]] = None
    ) -> Dict:
        """
        Attach a configuration file for use in containers.

        This allows sending configs via Element X that can be injected into
        containers as environment variables or mounted files.

        Args:
            name: Config filename (e.g., "agent.env", "config.toml")
            content: File content (base64 or raw string)
            encoding: Content encoding ("base64" or "raw", default: "raw")
            config_type: Config type hint ("env", "toml", "yaml", "json", etc.)
            metadata: Additional metadata key-value pairs

        Returns:
            Attach result with keys: config_id, name, path, size, type, encoding

        Example:
            # Attach an environment file
            client.attach_config(
                name="agent.env",
                content="MODEL=gpt-4\\nTEMPERATURE=0.7",
                encoding="raw",
                config_type="env"
            )

            # Attach a base64-encoded config
            import base64
            encoded = base64.b64encode(b"secret=value").decode()
            client.attach_config(
                name="secret.env",
                content=encoded,
                encoding="base64",
                config_type="env"
            )
        """
        params = {
            "name": name,
            "content": content,
            "encoding": encoding,
        }

        if config_type:
            params["type"] = config_type

        if metadata:
            params["metadata"] = metadata

        return self._send_request("attach_config", params)


# ============================================================================
# Async Bridge Client (for async agent operations)
# ============================================================================

class AsyncBridgeClient:
    """
    Async client for communicating with the ArmorClaw Local Bridge.

    This provides the same functionality as BridgeClient but with
    async methods for use in async agent loops.
    """

    def __init__(self, socket_path: str = "/run/armorclaw/bridge.sock"):
        """
        Initialize the async bridge client.

        Args:
            socket_path: Path to the bridge Unix socket
        """
        self.socket_path = socket_path
        self._request_id = 0

    def _get_next_id(self) -> int:
        """Get the next request ID."""
        self._request_id += 1
        return self._request_id

    async def _send_request(self, method: str, params: Optional[Dict] = None) -> Dict:
        """
        Send an async JSON-RPC request to the bridge.

        Args:
            method: RPC method name
            params: Method parameters (optional)

        Returns:
            Response dictionary
        """
        request = {
            "jsonrpc": "2.0",
            "id": self._get_next_id(),
            "method": method
        }

        if params is not None:
            request["params"] = params

        # Connect to bridge socket
        try:
            reader, writer = await asyncio.open_unix_connection(
                self.socket_path
            )

            # Send request
            request_json = json.dumps(request) + "\n"
            writer.write(request_json.encode('utf-8'))
            await writer.drain()

            # Receive response
            response_data = b""
            while True:
                try:
                    chunk = await asyncio.wait_for(reader.read(4096), timeout=5.0)
                    if not chunk:
                        break
                    response_data += chunk
                    # Try to parse complete JSON
                    try:
                        response = json.loads(response_data.decode('utf-8'))
                        break
                    except json.JSONDecodeError:
                        # Need more data
                        continue
                except asyncio.TimeoutError:
                    break

            writer.close()
            await writer.wait_closed()

        except FileNotFoundError:
            raise ConnectionError(
                f"Bridge socket not found at {self.socket_path}. "
                f"Is the ArmorClaw bridge running?"
            )
        except ConnectionRefusedError:
            raise ConnectionError(
                f"Connection refused to {self.socket_path}. "
                f"Is the ArmorClaw bridge running?"
            )

        # Check for RPC error
        if "error" in response:
            error = response["error"]
            raise RuntimeError(
                f"RPC error {error.get('code', 'unknown')}: {error.get('message', 'Unknown error')}"
            )

        return response.get("result")

    # ========================================================================
    # Async Core Methods
    # ========================================================================

    async def status(self) -> Dict:
        """Get bridge status (async)."""
        return await self._send_request("status")

    async def health(self) -> Dict:
        """Health check endpoint (async)."""
        return await self._send_request("health")

    # ========================================================================
    # Async Container Methods
    # ========================================================================

    async def start_container(
        self,
        key_id: str,
        agent_type: str = "openclaw",
        image: str = "armorclaw/agent:v1"
    ) -> Dict:
        """Start a new container (async)."""
        params = {
            "key_id": key_id,
            "agent_type": agent_type,
            "image": image
        }
        return await self._send_request("start", params)

    async def stop_container(self, container_id: str) -> Dict:
        """Stop a running container (async)."""
        params = {
            "container_id": container_id
        }
        return await self._send_request("stop", params)

    # ========================================================================
    # Async Matrix Methods
    # ========================================================================

    async def matrix_status(self) -> Dict:
        """Get Matrix connection status (async)."""
        return await self._send_request("matrix.status")

    async def matrix_send(
        self,
        room_id: str,
        message: str,
        msgtype: str = "m.text"
    ) -> Dict:
        """Send a message to a Matrix room (async)."""
        params = {
            "room_id": room_id,
            "message": message,
            "msgtype": msgtype
        }
        return await self._send_request("matrix.send", params)

    async def matrix_receive(self) -> Dict:
        """Receive pending Matrix events (async)."""
        return await self._send_request("matrix.receive")

    # ========================================================================
    # Async Config Methods
    # ========================================================================

    async def attach_config(
        self,
        name: str,
        content: str,
        encoding: str = "raw",
        config_type: str = "",
        metadata: Optional[Dict[str, str]] = None
    ) -> Dict:
        """
        Attach a configuration file for use in containers (async).

        This allows sending configs via Element X that can be injected into
        containers as environment variables or mounted files.

        Args:
            name: Config filename (e.g., "agent.env", "config.toml")
            content: File content (base64 or raw string)
            encoding: Content encoding ("base64" or "raw", default: "raw")
            config_type: Config type hint ("env", "toml", "yaml", "json", etc.)
            metadata: Additional metadata key-value pairs

        Returns:
            Attach result with keys: config_id, name, path, size, type, encoding
        """
        params = {
            "name": name,
            "content": content,
            "encoding": encoding,
        }

        if config_type:
            params["type"] = config_type

        if metadata:
            params["metadata"] = metadata

        return await self._send_request("attach_config", params)


# ============================================================================
# Convenience Functions
# ============================================================================

def get_default_client() -> BridgeClient:
    """
    Get a bridge client with the default socket path.

    The socket path can be overridden via environment variable:
    ARMORCLAW_BRIDGE_SOCKET

    Returns:
        BridgeClient instance
    """
    socket_path = os.getenv("ARMORCLAW_BRIDGE_SOCKET", "/run/armorclaw/bridge.sock")
    return BridgeClient(socket_path)
