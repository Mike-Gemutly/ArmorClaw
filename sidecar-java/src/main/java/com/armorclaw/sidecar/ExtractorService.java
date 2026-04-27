package com.armorclaw.sidecar;

import armorclaw.sidecar.v1.Sidecar;
import armorclaw.sidecar.v1.SidecarServiceGrpc;
import io.grpc.stub.StreamObserver;
import java.lang.management.ManagementFactory;
import java.lang.management.MemoryMXBean;

/**
 * gRPC service implementation for the ArmorClaw Java Sidecar.
 *
 * <p>HealthCheck is fully implemented. All other RPCs return UNIMPLEMENTED
 * until their respective tasks land (T4/T5 for ExtractText, etc.).
 */
public class ExtractorService extends SidecarServiceGrpc.SidecarServiceImplBase {

    private static final String VERSION = "1.0.0";
    private final long startTimeMillis;

    public ExtractorService() {
        this.startTimeMillis = System.currentTimeMillis();
    }

    @Override
    public void healthCheck(
            Sidecar.HealthCheckRequest request,
            StreamObserver<Sidecar.HealthCheckResponse> responseObserver) {

        long uptimeSeconds = (System.currentTimeMillis() - startTimeMillis) / 1000;

        MemoryMXBean memoryBean = ManagementFactory.getMemoryMXBean();
        long memoryUsedBytes = memoryBean.getHeapMemoryUsage().getUsed();

        Sidecar.HealthCheckResponse response = Sidecar.HealthCheckResponse.newBuilder()
                .setStatus("SERVING")
                .setVersion(VERSION)
                .setUptimeSeconds(uptimeSeconds)
                .setMemoryUsedBytes(memoryUsedBytes)
                .build();

        responseObserver.onNext(response);
        responseObserver.onCompleted();
    }

    @Override
    public void uploadBlob(
            Sidecar.UploadBlobRequest request,
            StreamObserver<Sidecar.UploadBlobResponse> responseObserver) {
        responseObserver.onError(io.grpc.Status.UNIMPLEMENTED
                .withDescription("UploadBlob not supported by Java sidecar")
                .asRuntimeException());
    }

    @Override
    public void downloadBlob(
            Sidecar.DownloadBlobRequest request,
            StreamObserver<Sidecar.BlobChunk> responseObserver) {
        responseObserver.onError(io.grpc.Status.UNIMPLEMENTED
                .withDescription("DownloadBlob not supported by Java sidecar")
                .asRuntimeException());
    }

    @Override
    public void listBlobs(
            Sidecar.ListBlobsRequest request,
            StreamObserver<Sidecar.ListBlobsResponse> responseObserver) {
        responseObserver.onError(io.grpc.Status.UNIMPLEMENTED
                .withDescription("ListBlobs not supported by Java sidecar")
                .asRuntimeException());
    }

    @Override
    public void deleteBlob(
            Sidecar.DeleteBlobRequest request,
            StreamObserver<Sidecar.DeleteBlobResponse> responseObserver) {
        responseObserver.onError(io.grpc.Status.UNIMPLEMENTED
                .withDescription("DeleteBlob not supported by Java sidecar")
                .asRuntimeException());
    }

    @Override
    public void extractText(
            Sidecar.ExtractTextRequest request,
            StreamObserver<Sidecar.ExtractTextResponse> responseObserver) {
        responseObserver.onError(io.grpc.Status.UNIMPLEMENTED
                .withDescription("ExtractText not yet implemented")
                .asRuntimeException());
    }

    @Override
    public void processDocument(
            Sidecar.ProcessDocumentRequest request,
            StreamObserver<Sidecar.ProcessDocumentResponse> responseObserver) {
        responseObserver.onError(io.grpc.Status.UNIMPLEMENTED
                .withDescription("ProcessDocument not supported by Java sidecar")
                .asRuntimeException());
    }

    @Override
    public void queryDocuments(
            Sidecar.QueryDocumentsRequest request,
            StreamObserver<Sidecar.QueryDocumentsResponse> responseObserver) {
        responseObserver.onError(io.grpc.Status.UNIMPLEMENTED
                .withDescription("QueryDocuments not supported by Java sidecar")
                .asRuntimeException());
    }
}
