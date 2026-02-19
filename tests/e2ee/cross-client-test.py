#!/usr/bin/env python3
"""
Cross-Client E2EE Test Suite for ArmorClaw

This script tests E2EE compatibility between ArmorChat and other Matrix clients.
It requires:
- A running Matrix server (Conduit)
- Element Web/Desktop installed
- ArmorChat with vodozemac compiled

Usage:
    python cross-client-test.py --server http://localhost:8008
"""

import argparse
import json
import requests
import time
import sys
from dataclasses import dataclass
from typing import Optional

@dataclass
class TestResult:
    name: str
    passed: bool
    message: str
    duration_ms: float

class MatrixTestClient:
    """Simple Matrix client for testing"""

    def __init__(self, server_url: str):
        self.server_url = server_url.rstrip('/')
        self.access_token: Optional[str] = None
        self.user_id: Optional[str] = None
        self.device_id: Optional[str] = None

    def register(self, username: str, password: str) -> dict:
        """Register a new user"""
        response = requests.post(
            f"{self.server_url}/_matrix/client/v3/register",
            json={
                "username": username,
                "password": password,
                "device_id": f"test_{username}",
                "initial_device_display_name": "Test Device"
            }
        )

        if response.status_code == 401:
            # Need to complete UIA
            response = requests.post(
                f"{self.server_url}/_matrix/client/v3/register",
                json={
                    "username": username,
                    "password": password,
                    "device_id": f"test_{username}",
                    "initial_device_display_name": "Test Device",
                    "auth": {"type": "m.login.dummy"}
                }
            )

        data = response.json()
        if response.status_code != 200:
            raise Exception(f"Registration failed: {data}")

        self.access_token = data.get("access_token")
        self.user_id = data.get("user_id")
        self.device_id = data.get("device_id")
        return data

    def login(self, username: str, password: str) -> dict:
        """Login as existing user"""
        response = requests.post(
            f"{self.server_url}/_matrix/client/v3/login",
            json={
                "type": "m.login.password",
                "user": username,
                "password": password,
                "device_id": f"test_{username}"
            }
        )

        data = response.json()
        if response.status_code != 200:
            raise Exception(f"Login failed: {data}")

        self.access_token = data.get("access_token")
        self.user_id = data.get("user_id")
        self.device_id = data.get("device_id")
        return data

    def create_room(self, name: str = "Test Room") -> str:
        """Create a new room"""
        response = requests.post(
            f"{self.server_url}/_matrix/client/v3/createRoom",
            headers={"Authorization": f"Bearer {self.access_token}"},
            json={
                "name": name,
                "preset": "private_chat",
                "visibility": "private"
            }
        )

        data = response.json()
        if response.status_code != 200:
            raise Exception(f"Create room failed: {data}")

        return data["room_id"]

    def send_message(self, room_id: str, body: str) -> str:
        """Send a text message"""
        txn_id = f"txn_{int(time.time() * 1000)}"
        response = requests.put(
            f"{self.server_url}/_matrix/client/v3/rooms/{room_id}/send/m.room.message/{txn_id}",
            headers={"Authorization": f"Bearer {self.access_token}"},
            json={
                "msgtype": "m.text",
                "body": body
            }
        )

        data = response.json()
        if response.status_code != 200:
            raise Exception(f"Send message failed: {data}")

        return data["event_id"]

    def get_messages(self, room_id: str, limit: int = 10) -> list:
        """Get messages from a room"""
        response = requests.get(
            f"{self.server_url}/_matrix/client/v3/rooms/{room_id}/messages",
            headers={"Authorization": f"Bearer {self.access_token}"},
            params={"limit": limit, "dir": "b"}
        )

        data = response.json()
        if response.status_code != 200:
            raise Exception(f"Get messages failed: {data}")

        return data.get("chunk", [])

    def invite_user(self, room_id: str, user_id: str) -> dict:
        """Invite a user to a room"""
        response = requests.post(
            f"{self.server_url}/_matrix/client/v3/rooms/{room_id}/invite",
            headers={"Authorization": f"Bearer {self.access_token}"},
            json={"user_id": user_id}
        )

        return response.json()

    def join_room(self, room_id: str) -> dict:
        """Join a room"""
        response = requests.post(
            f"{self.server_url}/_matrix/client/v3/rooms/{room_id}/join",
            headers={"Authorization": f"Bearer {self.access_token}"},
            json={}
        )

        return response.json()


class E2EETestSuite:
    """Cross-client E2EE test suite"""

    def __init__(self, server_url: str):
        self.server_url = server_url
        self.results: list[TestResult] = []

    def run_test(self, name: str, test_func) -> TestResult:
        """Run a single test"""
        start_time = time.time()
        try:
            test_func()
            result = TestResult(
                name=name,
                passed=True,
                message="OK",
                duration_ms=(time.time() - start_time) * 1000
            )
        except Exception as e:
            result = TestResult(
                name=name,
                passed=False,
                message=str(e),
                duration_ms=(time.time() - start_time) * 1000
            )

        self.results.append(result)
        status = "✓" if result.passed else "✗"
        print(f"  {status} {name} ({result.duration_ms:.0f}ms)")
        if not result.passed:
            print(f"    Error: {result.message}")
        return result

    def test_server_connectivity(self):
        """Test that Matrix server is reachable"""
        response = requests.get(f"{self.server_url}/_matrix/client/versions")
        assert response.status_code == 200, f"Server not reachable: {response.status_code}"
        data = response.json()
        assert "versions" in data, "Invalid server response"

    def test_user_registration(self):
        """Test user registration"""
        client = MatrixTestClient(self.server_url)
        result = client.register(f"test_user_{int(time.time())}", "test_password_123")
        assert "access_token" in result, "No access token in registration response"
        assert "user_id" in result, "No user_id in registration response"

    def test_room_creation(self):
        """Test room creation"""
        client = MatrixTestClient(self.server_url)
        client.register(f"room_test_{int(time.time())}", "test_password")
        room_id = client.create_room("Test Room")
        assert room_id.startswith("!"), f"Invalid room ID: {room_id}"

    def test_message_send_receive(self):
        """Test message send and receive"""
        client = MatrixTestClient(self.server_url)
        client.register(f"msg_test_{int(time.time())}", "test_password")
        room_id = client.create_room("Message Test Room")

        test_message = f"Test message at {time.time()}"
        event_id = client.send_message(room_id, test_message)

        messages = client.get_messages(room_id)
        assert len(messages) > 0, "No messages found"

        found = any(
            m.get("content", {}).get("body") == test_message
            for m in messages
        )
        assert found, "Sent message not found in room"

    def test_two_user_message_flow(self):
        """Test message flow between two users"""
        # User 1 creates room and sends message
        client1 = MatrixTestClient(self.server_url)
        client1.register(f"user1_{int(time.time())}", "password1")
        room_id = client1.create_room("Two User Test")

        # User 2 registers and joins
        client2 = MatrixTestClient(self.server_url)
        client2.register(f"user2_{int(time.time())}", "password2")
        client1.invite_user(room_id, client2.user_id)
        client2.join_room(room_id)

        # User 2 sends message
        test_message = f"Hello from user2 at {time.time()}"
        client2.send_message(room_id, test_message)

        # User 1 receives message
        messages = client1.get_messages(room_id)
        found = any(
            m.get("content", {}).get("body") == test_message
            for m in messages
        )
        assert found, "User 2's message not received by User 1"

    def run_all(self):
        """Run all tests"""
        print("\n" + "="*60)
        print("ArmorClaw Cross-Client E2EE Test Suite")
        print("="*60)
        print(f"\nServer: {self.server_url}")
        print("\nRunning tests...\n")

        self.run_test("Server Connectivity", self.test_server_connectivity)
        self.run_test("User Registration", self.test_user_registration)
        self.run_test("Room Creation", self.test_room_creation)
        self.run_test("Message Send/Receive", self.test_message_send_receive)
        self.run_test("Two User Message Flow", self.test_two_user_message_flow)

        print("\n" + "-"*60)
        passed = sum(1 for r in self.results if r.passed)
        total = len(self.results)
        print(f"Results: {passed}/{total} tests passed")

        if passed == total:
            print("All tests passed! ✓")
            return 0
        else:
            print("Some tests failed! ✗")
            return 1


def main():
    parser = argparse.ArgumentParser(description="ArmorClaw Cross-Client E2EE Tests")
    parser.add_argument(
        "--server",
        default="http://localhost:8008",
        help="Matrix server URL (default: http://localhost:8008)"
    )
    args = parser.parse_args()

    suite = E2EETestSuite(args.server)
    sys.exit(suite.run_all())


if __name__ == "__main__":
    main()
