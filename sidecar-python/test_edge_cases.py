import os
import sys
import time
from concurrent.futures import ThreadPoolExecutor

import grpc
import pytest

sys.path.insert(0, os.path.dirname(__file__))

from conftest import TEST_SECRET, make_token_metadata
from proto import sidecar_pb2, sidecar_pb2_grpc
from worker import _THRESHOLD_BYTES


class TestEmptyPayload:
    def test_zero_bytes(self, grpc_stub):
        with pytest.raises(grpc.RpcError) as exc:
            grpc_stub.ExtractText(
                sidecar_pb2.ExtractTextRequest(
                    document_format="application/vnd.ms-excel",
                    document_content=b"",
                )
            )
        assert exc.value.code() == grpc.StatusCode.INVALID_ARGUMENT

    def test_one_byte(self, grpc_stub):
        resp = grpc_stub.ExtractText(
            sidecar_pb2.ExtractTextRequest(
                document_format="application/vnd.ms-excel",
                document_content=b"\x00",
            )
        )
        assert isinstance(resp.text, str)

    def test_seven_bytes(self, grpc_stub):
        resp = grpc_stub.ExtractText(
            sidecar_pb2.ExtractTextRequest(
                document_format="application/vnd.ms-excel",
                document_content=b"\x00" * 7,
            )
        )
        assert isinstance(resp.text, str)


class TestCorruptFiles:
    def _corrupt_and_verify_server_alive(self, grpc_stub, magic, mime):
        corrupt = magic + os.urandom(1024)
        with pytest.raises(grpc.RpcError):
            grpc_stub.ExtractText(
                sidecar_pb2.ExtractTextRequest(
                    document_format=mime,
                    document_content=corrupt,
                )
            )
        resp = grpc_stub.HealthCheck(sidecar_pb2.HealthCheckRequest())
        assert resp.status == "SERVING"

    def test_corrupt_zip(self, grpc_stub):
        self._corrupt_and_verify_server_alive(
            grpc_stub, b"PK\x03\x04",
            "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
        )

    def test_corrupt_ole(self, grpc_stub):
        self._corrupt_and_verify_server_alive(
            grpc_stub, b"\xd0\xcf\x11\xe0\xa1\xb1\x1a\xe1",
            "application/vnd.ms-excel",
        )

    def test_valid_zip_non_office(self, grpc_stub):
        jar_content = b"PK\x03\x04" + os.urandom(512)
        with pytest.raises(grpc.RpcError):
            grpc_stub.ExtractText(
                sidecar_pb2.ExtractTextRequest(
                    document_format="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
                    document_content=jar_content,
                )
            )


class TestThresholdBoundary:
    def test_exactly_threshold_bytes(self, grpc_stub):
        content = b"\x00" * _THRESHOLD_BYTES
        with pytest.raises(grpc.RpcError):
            grpc_stub.ExtractText(
                sidecar_pb2.ExtractTextRequest(
                    document_format="application/vnd.ms-excel",
                    document_content=content,
                )
            )

    def test_threshold_minus_one(self, grpc_stub):
        content = b"\x00" * (_THRESHOLD_BYTES - 1)
        with pytest.raises(grpc.RpcError):
            grpc_stub.ExtractText(
                sidecar_pb2.ExtractTextRequest(
                    document_format="application/vnd.ms-excel",
                    document_content=content,
                )
            )

    def test_threshold_plus_one(self, grpc_stub):
        content = b"\x00" * (_THRESHOLD_BYTES + 1)
        with pytest.raises(grpc.RpcError):
            grpc_stub.ExtractText(
                sidecar_pb2.ExtractTextRequest(
                    document_format="application/vnd.ms-excel",
                    document_content=content,
                )
            )


class TestConcurrentRequests:
    def test_ten_concurrent_succeed(self, grpc_stub, xlsx_file):
        content = xlsx_file.read_bytes()

        def extract():
            return grpc_stub.ExtractText(
                sidecar_pb2.ExtractTextRequest(
                    document_format="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
                    document_content=content,
                )
            )

        with ThreadPoolExecutor(max_workers=10) as pool:
            futures = [pool.submit(extract) for _ in range(10)]
            results = [f.result() for f in futures]

        for resp in results:
            assert "Hello ArmorClaw" in resp.text


class TestFormatStringVariants:
    def test_xls_macro_enabled_variant(self, grpc_stub, xls_file):
        content = xls_file.read_bytes()
        with pytest.raises(grpc.RpcError) as exc:
            grpc_stub.ExtractText(
                sidecar_pb2.ExtractTextRequest(
                    document_format="application/vnd.ms-excel.sheet.macroenabled.12",
                    document_content=content,
                )
            )
        assert exc.value.code() == grpc.StatusCode.INVALID_ARGUMENT

    def test_msg_variant(self, grpc_stub, msg_file):
        content = msg_file.read_bytes()
        resp = grpc_stub.ExtractText(
            sidecar_pb2.ExtractTextRequest(
                document_format="application/vnd.ms-outlook",
                document_content=content,
            )
        )
        assert isinstance(resp.text, str)


class TestTokenIntegration:
    def test_valid_token_success(self, grpc_stub_auth, xlsx_file):
        content = xlsx_file.read_bytes()
        metadata = make_token_metadata()
        resp = grpc_stub_auth.ExtractText(
            sidecar_pb2.ExtractTextRequest(
                document_format="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
                document_content=content,
            ),
            metadata=metadata,
        )
        assert "Hello ArmorClaw" in resp.text

    def test_expired_token_rejected(self, grpc_stub_auth, xlsx_file):
        content = xlsx_file.read_bytes()
        old_ts = int(time.time()) - 3600
        metadata = make_token_metadata(timestamp=old_ts)
        with pytest.raises(grpc.RpcError) as exc:
            grpc_stub_auth.ExtractText(
                sidecar_pb2.ExtractTextRequest(
                    document_format="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
                    document_content=content,
                ),
                metadata=metadata,
            )
        assert exc.value.code() == grpc.StatusCode.UNAUTHENTICATED

    def test_missing_token_rejected(self, grpc_stub_auth, xlsx_file):
        content = xlsx_file.read_bytes()
        with pytest.raises(grpc.RpcError) as exc:
            grpc_stub_auth.ExtractText(
                sidecar_pb2.ExtractTextRequest(
                    document_format="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
                    document_content=content,
                ),
            )
        assert exc.value.code() == grpc.StatusCode.UNAUTHENTICATED

    def test_tampered_token_rejected(self, grpc_stub_auth, xlsx_file):
        content = xlsx_file.read_bytes()
        metadata = make_token_metadata()
        metadata = [(k, v + "tampered") for k, v in metadata]
        with pytest.raises(grpc.RpcError) as exc:
            grpc_stub_auth.ExtractText(
                sidecar_pb2.ExtractTextRequest(
                    document_format="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
                    document_content=content,
                ),
                metadata=metadata,
            )
        assert exc.value.code() == grpc.StatusCode.UNAUTHENTICATED
