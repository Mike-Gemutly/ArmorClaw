package com.armorclaw.sidecar;

import io.grpc.ForwardingServerCall;
import io.grpc.Metadata;
import io.grpc.ServerCall;
import io.grpc.ServerCallHandler;
import io.grpc.ServerInterceptor;
import io.grpc.Status;

/**
 * gRPC server interceptor that adds {@code x-sidecar-server-version} to trailing metadata
 * on every RPC response. Matches the version protocol in bridge/pkg/sidecar/version.go.
 */
public class VersionInterceptor implements ServerInterceptor {

    private static final String SERVER_VERSION = "1.0.0";
    private static final Metadata.Key<String> VERSION_KEY =
            Metadata.Key.of("x-sidecar-server-version", Metadata.ASCII_STRING_MARSHALLER);

    @Override
    public <ReqT, RespT> ServerCall.Listener<ReqT> interceptCall(
            ServerCall<ReqT, RespT> call,
            Metadata headers,
            ServerCallHandler<ReqT, RespT> next) {

        ServerCall<ReqT, RespT> versioningCall =
                new ForwardingServerCall.SimpleForwardingServerCall<>(call) {
                    @Override
                    public void close(Status status, Metadata trailers) {
                        trailers.put(VERSION_KEY, SERVER_VERSION);
                        super.close(status, trailers);
                    }
                };

        return next.startCall(versioningCall, headers);
    }
}
