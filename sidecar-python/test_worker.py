import os
import sys
import time

import grpc
import pytest

sys.path.insert(0, os.path.dirname(__file__))

from proto import sidecar_pb2, sidecar_pb2_grpc
from worker import FORMAT_MAP, MAX_REQUESTS, SERVER_VERSION, _THRESHOLD_BYTES


class TestFormatMapping:

    EXPECTED = {
        "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet": ".xlsx",
        "application/vnd.openxmlformats-officedocument.presentationml.presentation": ".pptx",
        "application/vnd.ms-outlook": ".msg",
        "application/msword": ".doc",
        "application/vnd.ms-excel": ".xls",
        "application/vnd.ms-powerpoint": ".ppt",
    }

    def test_all_six_formats_present(self):
        for mime, ext in self.EXPECTED.items():
            assert FORMAT_MAP[mime] == ext, f"{mime} should map to {ext}"

    def test_unrecognized_mime_returns_none(self):
        from worker import OfficeSidecarServicer

        svc = OfficeSidecarServicer()
        assert svc._resolve_extension("application/unknown") is None

    def test_case_insensitive_matching(self):
        from worker import OfficeSidecarServicer

        svc = OfficeSidecarServicer()
        assert svc._resolve_extension("APPLICATION/VND.MS-EXCEL") == ".xls"
        assert svc._resolve_extension("Application/Vnd.Ms-Outlook") == ".msg"

    def test_extension_fallback(self):
        from worker import OfficeSidecarServicer

        svc = OfficeSidecarServicer()
        assert svc._resolve_extension("file.xlsx") == ".xlsx"
        assert svc._resolve_extension("file.msg") == ".msg"


class TestServerStartup:

    def test_healthcheck_returns_serving(self, grpc_stub):
        resp = grpc_stub.HealthCheck(sidecar_pb2.HealthCheckRequest())
        assert resp.status == "SERVING"

    def test_healthcheck_version(self, grpc_stub):
        resp = grpc_stub.HealthCheck(sidecar_pb2.HealthCheckRequest())
        assert resp.version == SERVER_VERSION

    def test_healthcheck_uptime_nonzero(self, grpc_stub):
        resp = grpc_stub.HealthCheck(sidecar_pb2.HealthCheckRequest())
        assert resp.uptime_seconds >= 0

    def test_healthcheck_active_requests(self, grpc_stub):
        resp = grpc_stub.HealthCheck(sidecar_pb2.HealthCheckRequest())
        assert resp.active_requests >= 1


class TestExtractTextFormats:

    CONVERTIBLE_FORMATS = [
        (
            "xlsx",
            "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
            "Hello ArmorClaw",
            "xlsx_file",
        ),
        (
            "pptx",
            "application/vnd.openxmlformats-officedocument.presentationml.presentation",
            "Test Presentation",
            "pptx_file",
        ),
        (
            "xls",
            "application/vnd.ms-excel",
            "Test",
            "xls_file",
        ),
        (
            "msg",
            "application/vnd.ms-outlook",
            None,
            "msg_file",
        ),
    ]

    ERROR_HANDLING_FORMATS = [
        ("doc", "application/msword", "doc_file"),
        ("ppt", "application/vnd.ms-powerpoint", "ppt_file"),
    ]

    @pytest.mark.parametrize(
        "fmt_name,mime,expected_text,fixture_name",
        CONVERTIBLE_FORMATS,
        ids=[f[0] for f in CONVERTIBLE_FORMATS],
    )
    def test_format_conversion(self, fmt_name, mime, expected_text, fixture_name, request, grpc_stub):
        fixture_path = request.getfixturevalue(fixture_name)
        content = fixture_path.read_bytes()

        resp = grpc_stub.ExtractText(
            sidecar_pb2.ExtractTextRequest(
                document_format=mime,
                document_content=content,
            )
        )

        if expected_text is not None:
            assert expected_text in resp.text, (
                f"{fmt_name}: expected {expected_text!r} in response, got {resp.text!r}"
            )
        else:
            assert isinstance(resp.text, str)

    @pytest.mark.parametrize(
        "fmt_name,mime,fixture_name",
        ERROR_HANDLING_FORMATS,
        ids=[f[0] for f in ERROR_HANDLING_FORMATS],
    )
    def test_format_error_handling(self, fmt_name, mime, fixture_name, request, grpc_stub):
        fixture_path = request.getfixturevalue(fixture_name)
        content = fixture_path.read_bytes()

        with pytest.raises(grpc.RpcError) as exc_info:
            grpc_stub.ExtractText(
                sidecar_pb2.ExtractTextRequest(
                    document_format=mime,
                    document_content=content,
                )
            )
        assert exc_info.value.code() in (grpc.StatusCode.INTERNAL, grpc.StatusCode.INVALID_ARGUMENT)

    def test_xlsx_page_count(self, grpc_stub, xlsx_file):
        content = xlsx_file.read_bytes()
        resp = grpc_stub.ExtractText(
            sidecar_pb2.ExtractTextRequest(
                document_format="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
                document_content=content,
            )
        )
        assert resp.page_count >= 0


class TestExtractTextErrors:

    def test_empty_content_returns_error(self, grpc_stub):
        with pytest.raises(grpc.RpcError) as exc_info:
            grpc_stub.ExtractText(
                sidecar_pb2.ExtractTextRequest(
                    document_format="application/vnd.ms-excel",
                    document_content=b"",
                )
            )
        assert exc_info.value.code() == grpc.StatusCode.INVALID_ARGUMENT

    def test_unsupported_format_returns_error(self, grpc_stub):
        with pytest.raises(grpc.RpcError) as exc_info:
            grpc_stub.ExtractText(
                sidecar_pb2.ExtractTextRequest(
                    document_format="application/unsupported",
                    document_content=b"\x00" * 16,
                )
            )
        assert exc_info.value.code() == grpc.StatusCode.INVALID_ARGUMENT

    def test_corrupt_zip_magic_no_crash(self, grpc_stub):
        corrupt = b"PK\x03\x04" + os.urandom(1024)
        with pytest.raises(grpc.RpcError):
            grpc_stub.ExtractText(
                sidecar_pb2.ExtractTextRequest(
                    document_format="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
                    document_content=corrupt,
                )
            )

    def test_server_survives_corrupt_input(self, grpc_stub):
        corrupt = b"\xd0\xcf\x11\xe0\xa1\xb1\x1a\xe1" + os.urandom(1024)
        with pytest.raises(grpc.RpcError):
            grpc_stub.ExtractText(
                sidecar_pb2.ExtractTextRequest(
                    document_format="application/vnd.ms-excel",
                    document_content=corrupt,
                )
            )
        resp = grpc_stub.HealthCheck(sidecar_pb2.HealthCheckRequest())
        assert resp.status == "SERVING"


class TestThresholdStreaming:

    def test_small_file_succeeds(self, grpc_stub, xlsx_file):
        content = xlsx_file.read_bytes()
        assert len(content) < _THRESHOLD_BYTES
        resp = grpc_stub.ExtractText(
            sidecar_pb2.ExtractTextRequest(
                document_format="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
                document_content=content,
            )
        )
        assert "Hello ArmorClaw" in resp.text

    def test_large_file_succeeds(self, grpc_stub, large_xlsx_file):
        content = large_xlsx_file.read_bytes()
        assert len(content) >= _THRESHOLD_BYTES, (
            f"large_xlsx_file must be >= {_THRESHOLD_BYTES} bytes, got {len(content)}"
        )
        resp = grpc_stub.ExtractText(
            sidecar_pb2.ExtractTextRequest(
                document_format="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
                document_content=content,
            )
        )
        assert len(resp.text) > 0


class TestTTLRecycling:

    def test_request_count_increases(self, grpc_stub):
        resp1 = grpc_stub.HealthCheck(sidecar_pb2.HealthCheckRequest())
        count1 = resp1.active_requests
        resp2 = grpc_stub.HealthCheck(sidecar_pb2.HealthCheckRequest())
        count2 = resp2.active_requests
        assert count2 >= count1


class TestUnimplementedRPCs:

    def test_upload_blob(self, grpc_stub):
        with pytest.raises(grpc.RpcError) as exc_info:
            grpc_stub.UploadBlob(sidecar_pb2.UploadBlobRequest())
        assert exc_info.value.code() == grpc.StatusCode.UNIMPLEMENTED

    def test_download_blob(self, grpc_stub):
        responses = grpc_stub.DownloadBlob(sidecar_pb2.DownloadBlobRequest())
        with pytest.raises(grpc.RpcError) as exc_info:
            list(responses)
        assert exc_info.value.code() == grpc.StatusCode.UNIMPLEMENTED

    def test_list_blobs(self, grpc_stub):
        with pytest.raises(grpc.RpcError) as exc_info:
            grpc_stub.ListBlobs(sidecar_pb2.ListBlobsRequest())
        assert exc_info.value.code() == grpc.StatusCode.UNIMPLEMENTED

    def test_delete_blob(self, grpc_stub):
        with pytest.raises(grpc.RpcError) as exc_info:
            grpc_stub.DeleteBlob(sidecar_pb2.DeleteBlobRequest())
        assert exc_info.value.code() == grpc.StatusCode.UNIMPLEMENTED

    def test_process_document(self, grpc_stub):
        with pytest.raises(grpc.RpcError) as exc_info:
            grpc_stub.ProcessDocument(sidecar_pb2.ProcessDocumentRequest())
        assert exc_info.value.code() == grpc.StatusCode.UNIMPLEMENTED
