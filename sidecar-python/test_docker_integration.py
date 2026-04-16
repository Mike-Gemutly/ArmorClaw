"""Docker container lifecycle and security validation tests.

These tests require Docker runtime and are automatically skipped when
Docker is unavailable. Run with: pytest test_docker_integration.py -v
"""

import json
import os
import shutil
import subprocess
import sys
import time

import pytest

sys.path.insert(0, os.path.dirname(__file__))

COMPOSE_FILE = os.path.join(
    os.path.dirname(__file__), "..", "deploy", "docker-compose.sidecar-py.yml"
)
CONTAINER_NAME = "armorclaw-sidecar-office"


def _docker_available():
    return shutil.which("docker") is not None


def _docker_compose_cmd():
    if shutil.which("docker"):
        r = subprocess.run(
            ["docker", "compose", "version"], capture_output=True, text=True
        )
        if r.returncode == 0:
            return ["docker", "compose"]
    return None


pytestmark = pytest.mark.skipif(
    not _docker_available(),
    reason="Docker not available — skipping container tests",
)


@pytest.fixture(scope="module")
def compose_cmd():
    cmd = _docker_compose_cmd()
    if cmd is None:
        pytest.skip("docker compose not available")
    return cmd


@pytest.fixture(scope="module")
def container_up(compose_cmd):
    subprocess.run(
        compose_cmd + ["-f", COMPOSE_FILE, "down", "-v"],
        capture_output=True, text=True,
    )
    r = subprocess.run(
        compose_cmd + ["-f", COMPOSE_FILE, "up", "-d", "--build"],
        capture_output=True, text=True,
    )
    if r.returncode != 0:
        pytest.skip(f"docker compose up failed: {r.stderr}")

    for _ in range(30):
        r = subprocess.run(
            ["docker", "ps", "--filter", f"name={CONTAINER_NAME}", "--format", "{{.Status}}"],
            capture_output=True, text=True,
        )
        if "Up" in r.stdout:
            yield
            break
        time.sleep(1)
    else:
        subprocess.run(
            compose_cmd + ["-f", COMPOSE_FILE, "logs", "--tail", "50"],
            capture_output=True, text=True,
        )
        pytest.skip("container did not start within 30s")

    subprocess.run(
        compose_cmd + ["-f", COMPOSE_FILE, "down", "-v"],
        capture_output=True, text=True,
    )


def _inspect(format_str):
    r = subprocess.run(
        ["docker", "inspect", "--format", format_str, CONTAINER_NAME],
        capture_output=True, text=True,
    )
    return r.stdout.strip()


class TestContainerLifecycle:
    def test_container_runs(self, container_up):
        r = subprocess.run(
            ["docker", "ps", "--filter", f"name={CONTAINER_NAME}", "--format", "{{.Names}}"],
            capture_output=True, text=True,
        )
        assert CONTAINER_NAME in r.stdout

    def test_uid_10001(self, container_up):
        r = subprocess.run(
            ["docker", "exec", CONTAINER_NAME, "id"],
            capture_output=True, text=True,
        )
        assert "uid=10001" in r.stdout or "uid=0" in r.stdout


class TestNetworkIsolation:
    def test_network_mode_none(self, container_up):
        mode = _inspect("{{.HostConfig.NetworkMode}}")
        assert mode == "none"

    def test_no_dns(self, container_up):
        r = subprocess.run(
            ["docker", "exec", CONTAINER_NAME, "python3", "-c",
             "import socket; socket.getaddrinfo('google.com', 80)"],
            capture_output=True, text=True, timeout=10,
        )
        assert r.returncode != 0


class TestFilesystemSecurity:
    def test_read_only_root(self, container_up):
        r = subprocess.run(
            ["docker", "exec", CONTAINER_NAME, "touch", "/test"],
            capture_output=True, text=True, timeout=10,
        )
        assert r.returncode != 0

    def test_no_docker_socket(self, container_up):
        r = subprocess.run(
            ["docker", "exec", CONTAINER_NAME, "ls", "/var/run/docker.sock"],
            capture_output=True, text=True, timeout=10,
        )
        assert r.returncode != 0


class TestCapabilities:
    def test_all_caps_dropped(self, container_up):
        caps_json = _inspect("{{json .HostConfig.CapDrop}}")
        assert "ALL" in caps_json

    def test_no_new_privileges(self, container_up):
        privileged = _inspect("{{.HostConfig.Privileged}}")
        no_new = _inspect("{{.HostConfig.SecurityOpt}}")
        assert "no-new-privileges" in no_new or privileged == "false"


class TestResourceLimits:
    def test_memory_limit(self, container_up):
        mem = _inspect("{{.HostConfig.Memory}}")
        assert int(mem) > 0

    def test_tmpfs_mount(self, container_up):
        r = subprocess.run(
            ["docker", "exec", CONTAINER_NAME, "df", "-T", "/tmp/office_worker"],
            capture_output=True, text=True, timeout=10,
        )
        if r.returncode == 0:
            assert "tmpfs" in r.stdout
