"""Tests for HMAC-SHA256 token validation interceptor."""

import hashlib
import hmac
import os
import tempfile
import time

import pytest

from interceptor import (
    TOKEN_TTL_SECONDS,
    MAX_TIMESTAMP_AGE_SECONDS,
    SERVER_VERSION,
    _calculate_hmac,
    validate_token,
    load_shared_secret,
)


TEST_SECRET = "test-shared-secret-for-unit-tests"


def _make_token(request_id: str, timestamp: int, operation: str, secret: str = TEST_SECRET) -> str:
    data = f"{request_id}{timestamp}{operation}"
    sig = _calculate_hmac(secret, data)
    return f"{request_id}:{timestamp}:{operation}:{sig}"


class TestCalculateHmac:
    def test_matches_manual_computation(self):
        data = "req123456781234extract_text"
        expected = hmac.new(TEST_SECRET.encode(), data.encode(), hashlib.sha256).hexdigest()
        assert _calculate_hmac(TEST_SECRET, data) == expected


class TestValidateToken:
    def test_valid_token_passes(self):
        now = int(time.time())
        token = _make_token("req-abc", now, "extract_text")
        validate_token(TEST_SECRET, token)

    def test_expired_token_rejected(self):
        old_ts = int(time.time()) - TOKEN_TTL_SECONDS - 60
        token = _make_token("req-exp", old_ts, "extract_text")
        with pytest.raises(ValueError, match="too old|expired"):
            validate_token(TEST_SECRET, token)

    def test_old_timestamp_rejected(self):
        old_ts = int(time.time()) - MAX_TIMESTAMP_AGE_SECONDS - 60
        token = _make_token("req-old", old_ts, "extract_text")
        with pytest.raises(ValueError, match="too old"):
            validate_token(TEST_SECRET, token)

    def test_invalid_signature_rejected(self):
        now = int(time.time())
        token = _make_token("req-sig", now, "extract_text")
        parts = token.split(":")
        parts[3] = "deadbeef" + parts[3][8:]
        tampered = ":".join(parts)
        with pytest.raises(ValueError, match="invalid.*signature"):
            validate_token(TEST_SECRET, tampered)

    def test_missing_token_rejected(self):
        with pytest.raises((ValueError, AttributeError)):
            validate_token(TEST_SECRET, None)

    def test_malformed_token_rejected(self):
        with pytest.raises(ValueError, match="malformed"):
            validate_token(TEST_SECRET, "only:two:parts")

    def test_malformed_timestamp_rejected(self):
        with pytest.raises(ValueError, match="malformed"):
            validate_token(TEST_SECRET, "req:notanumber:op:sig")


class TestLoadSharedSecret:
    def test_loads_from_file(self, tmp_path):
        secret_file = tmp_path / "secret"
        secret_file.write_text("my-secret-value\n")
        os.environ["SECRET_PATH"] = str(secret_file)
        try:
            secret = load_shared_secret()
            assert secret == "my-secret-value"
        finally:
            del os.environ["SECRET_PATH"]

    def test_raises_on_missing_file(self, tmp_path):
        os.environ["SECRET_PATH"] = str(tmp_path / "nonexistent")
        try:
            with pytest.raises(FileNotFoundError):
                load_shared_secret()
        finally:
            del os.environ["SECRET_PATH"]

    def test_raises_on_empty_file(self, tmp_path):
        secret_file = tmp_path / "empty"
        secret_file.write_text("")
        os.environ["SECRET_PATH"] = str(secret_file)
        try:
            with pytest.raises(RuntimeError, match="empty"):
                load_shared_secret()
        finally:
            del os.environ["SECRET_PATH"]


class TestServerVersion:
    def test_version_is_1_0_0(self):
        assert SERVER_VERSION == "1.0.0"
