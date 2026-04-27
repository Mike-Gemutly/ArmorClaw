package com.armorclaw.sidecar;

import armorclaw.sidecar.v1.Sidecar;
import io.grpc.stub.StreamObserver;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.ArgumentCaptor;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;

import java.util.concurrent.atomic.AtomicInteger;

import static org.junit.jupiter.api.Assertions.*;
import static org.mockito.Mockito.*;

@ExtendWith(MockitoExtension.class)
class ExtractorServiceTest {

    @Mock
    private StreamObserver<Sidecar.ExtractTextResponse> extractResponseObserver;

    @Mock
    private StreamObserver<Sidecar.HealthCheckResponse> healthResponseObserver;

    private ExtractorService service;

    @BeforeEach
    void setUp() {
        service = new ExtractorService(new AtomicInteger(0), 100, () -> {});
    }

    @Test
    void extractDocText_minimalDoc_returnsExpectedText() throws Exception {
        byte[] docBytes = TestFixtures.createMinimalDoc();
        Sidecar.ExtractTextRequest request = Sidecar.ExtractTextRequest.newBuilder()
                .setDocumentFormat("application/msword")
                .setDocumentContent(com.google.protobuf.ByteString.copyFrom(docBytes))
                .build();

        service.extractText(request, extractResponseObserver);

        ArgumentCaptor<Sidecar.ExtractTextResponse> captor =
                ArgumentCaptor.forClass(Sidecar.ExtractTextResponse.class);
        verify(extractResponseObserver).onNext(captor.capture());
        verify(extractResponseObserver).onCompleted();

        String text = captor.getValue().getText();
        assertTrue(text.contains("Test document content"),
                "Expected 'Test document content' in extracted text, got: " + text);
    }

    @Test
    void extractDocText_emptyDoc_returnsEmptyString() throws Exception {
        byte[] docBytes = TestFixtures.createEmptyDoc();
        Sidecar.ExtractTextRequest request = Sidecar.ExtractTextRequest.newBuilder()
                .setDocumentFormat("application/msword")
                .setDocumentContent(com.google.protobuf.ByteString.copyFrom(docBytes))
                .build();

        service.extractText(request, extractResponseObserver);

        ArgumentCaptor<Sidecar.ExtractTextResponse> captor =
                ArgumentCaptor.forClass(Sidecar.ExtractTextResponse.class);
        verify(extractResponseObserver).onNext(captor.capture());
        verify(extractResponseObserver).onCompleted();

        assertTrue(captor.getValue().getText().trim().isEmpty(),
                "Expected empty text for empty doc");
    }

    @Test
    void extractDocText_corruptDoc_propagatesException() throws Exception {
        byte[] corruptBytes = TestFixtures.createCorruptDoc();
        Sidecar.ExtractTextRequest request = Sidecar.ExtractTextRequest.newBuilder()
                .setDocumentFormat("application/msword")
                .setDocumentContent(com.google.protobuf.ByteString.copyFrom(corruptBytes))
                .build();

        assertThrows(IllegalArgumentException.class, () -> {
            service.extractText(request, extractResponseObserver);
        });
    }

    @Test
    void extractPptText_minimalPpt_returnsExpectedText() throws Exception {
        byte[] pptBytes = TestFixtures.createMinimalPpt();
        Sidecar.ExtractTextRequest request = Sidecar.ExtractTextRequest.newBuilder()
                .setDocumentFormat("application/vnd.ms-powerpoint")
                .setDocumentContent(com.google.protobuf.ByteString.copyFrom(pptBytes))
                .build();

        service.extractText(request, extractResponseObserver);

        ArgumentCaptor<Sidecar.ExtractTextResponse> captor =
                ArgumentCaptor.forClass(Sidecar.ExtractTextResponse.class);
        verify(extractResponseObserver).onNext(captor.capture());
        verify(extractResponseObserver).onCompleted();

        String text = captor.getValue().getText();
        assertTrue(text.contains("Test slide content"),
                "Expected 'Test slide content' in extracted text, got: " + text);
    }

    @Test
    void extractPptText_emptyPpt_returnsNonEmptyText() throws Exception {
        byte[] pptBytes = TestFixtures.createEmptyPpt();
        Sidecar.ExtractTextRequest request = Sidecar.ExtractTextRequest.newBuilder()
                .setDocumentFormat("application/vnd.ms-powerpoint")
                .setDocumentContent(com.google.protobuf.ByteString.copyFrom(pptBytes))
                .build();

        service.extractText(request, extractResponseObserver);

        ArgumentCaptor<Sidecar.ExtractTextResponse> captor =
                ArgumentCaptor.forClass(Sidecar.ExtractTextResponse.class);
        verify(extractResponseObserver).onNext(captor.capture());
        verify(extractResponseObserver).onCompleted();

        assertNotNull(captor.getValue().getText());
    }

    @Test
    void extractPptText_corruptPpt_returnsError() throws Exception {
        byte[] corruptBytes = TestFixtures.createCorruptPpt();
        Sidecar.ExtractTextRequest request = Sidecar.ExtractTextRequest.newBuilder()
                .setDocumentFormat("application/vnd.ms-powerpoint")
                .setDocumentContent(com.google.protobuf.ByteString.copyFrom(corruptBytes))
                .build();

        service.extractText(request, extractResponseObserver);

        verify(extractResponseObserver).onError(any());
        verify(extractResponseObserver, never()).onNext(any());
    }

    @Test
    void extractText_unsupportedFormat_returnsInvalidArgument() {
        Sidecar.ExtractTextRequest request = Sidecar.ExtractTextRequest.newBuilder()
                .setDocumentFormat("application/pdf")
                .setDocumentContent(com.google.protobuf.ByteString.copyFromUtf8("test"))
                .build();

        service.extractText(request, extractResponseObserver);

        verify(extractResponseObserver).onError(any());
        verify(extractResponseObserver, never()).onNext(any());
    }

    @Test
    void healthCheck_returnsServingStatus() {
        Sidecar.HealthCheckRequest request = Sidecar.HealthCheckRequest.newBuilder().build();

        service.healthCheck(request, healthResponseObserver);

        ArgumentCaptor<Sidecar.HealthCheckResponse> captor =
                ArgumentCaptor.forClass(Sidecar.HealthCheckResponse.class);
        verify(healthResponseObserver).onNext(captor.capture());
        verify(healthResponseObserver).onCompleted();

        assertEquals("SERVING", captor.getValue().getStatus());
        assertEquals("1.0.0", captor.getValue().getVersion());
        assertTrue(captor.getValue().getUptimeSeconds() >= 0);
    }
}
