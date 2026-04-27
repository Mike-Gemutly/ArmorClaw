package com.armorclaw.sidecar;

import io.grpc.Metadata;
import io.grpc.ServerCall;
import io.grpc.ServerCallHandler;
import io.grpc.ServerInterceptor;
import io.grpc.Status;

import javax.crypto.Mac;
import javax.crypto.spec.SecretKeySpec;
import java.io.IOException;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.security.MessageDigest;
import java.time.Instant;

/**
 * gRPC server interceptor that validates HMAC-SHA256 tokens.
 *
 * <p>Token format: {@code {request_id}:{timestamp}:{operation}:{hmac_hex}}
 * <p>HMAC input: {@code {request_id}{timestamp}{operation}} (concatenated, no separators)
 *
 * <p>If the shared secret file is missing or empty, all requests are allowed (dev mode).
 * Tokens have a 30-minute TTL and 5-minute future tolerance for clock skew.
 */
public class TokenInterceptor implements ServerInterceptor {

    private static final String HMAC_ALGORITHM = "HmacSHA256";
    private static final String SECRET_PATH_ENV = "SIDECAR_SHARED_SECRET_PATH";
    private static final String DEFAULT_SECRET_PATH = "/run/secrets/shared_secret";
    private static final long TOKEN_TTL_SECONDS = 30 * 60L;
    private static final long FUTURE_TOLERANCE_SECONDS = 5 * 60L;

    private static final Metadata.Key<String> TOKEN_KEY =
            Metadata.Key.of("x-request-token", Metadata.ASCII_STRING_MARSHALLER);

    private final byte[] sharedSecret;

    public TokenInterceptor() {
        this.sharedSecret = loadSharedSecret();
    }

    private static byte[] loadSharedSecret() {
        String secretPath = System.getenv(SECRET_PATH_ENV);
        if (secretPath == null || secretPath.isBlank()) {
            secretPath = DEFAULT_SECRET_PATH;
        }

        Path path = Path.of(secretPath);
        if (!Files.exists(path)) {
            System.out.println("[TokenInterceptor] Secret file not found at " + secretPath
                    + " — dev mode (no auth)");
            return null;
        }

        try {
            String secret = Files.readString(path).trim();
            if (secret.isEmpty()) {
                System.out.println("[TokenInterceptor] Secret file is empty at " + secretPath
                        + " — dev mode (no auth)");
                return null;
            }
            System.out.println("[TokenInterceptor] Loaded shared secret from " + secretPath);
            return secret.getBytes(StandardCharsets.UTF_8);
        } catch (IOException e) {
            System.err.println("[TokenInterceptor] Failed to read secret file: " + e.getMessage()
                    + " — dev mode (no auth)");
            return null;
        }
    }

    @Override
    public <ReqT, RespT> ServerCall.Listener<ReqT> interceptCall(
            ServerCall<ReqT, RespT> call,
            Metadata headers,
            ServerCallHandler<ReqT, RespT> next) {

        if (sharedSecret == null) {
            return next.startCall(call, headers);
        }

        String token = headers.get(TOKEN_KEY);
        if (token == null || token.isEmpty()) {
            call.close(Status.UNAUTHENTICATED.withDescription("missing token"), new Metadata());
            return new ServerCall.Listener<>() {};
        }

        try {
            validateToken(token);
        } catch (TokenValidationException e) {
            call.close(Status.UNAUTHENTICATED.withDescription(e.getMessage()), new Metadata());
            return new ServerCall.Listener<>() {};
        }

        return next.startCall(call, headers);
    }

    private void validateToken(String token) throws TokenValidationException {
        String[] parts = token.split(":");
        if (parts.length != 4) {
            throw new TokenValidationException("malformed token");
        }

        String requestId = parts[0];
        String timestampStr = parts[1];
        String operation = parts[2];
        String signature = parts[3];

        long timestamp;
        try {
            timestamp = Long.parseLong(timestampStr);
        } catch (NumberFormatException e) {
            throw new TokenValidationException("malformed token");
        }

        long now = Instant.now().getEpochSecond();

        // Token must not be older than 30 minutes
        if ((now - timestamp) > TOKEN_TTL_SECONDS) {
            throw new TokenValidationException("token expired");
        }

        // Token must not be more than 5 minutes in the future
        if ((timestamp - now) > FUTURE_TOLERANCE_SECONDS) {
            throw new TokenValidationException("token timestamp in the future");
        }

        // Recompute HMAC of {request_id}{timestamp}{operation} per token protocol
        String dataToSign = requestId + timestampStr + operation;
        String expectedSignature = computeHmac(dataToSign);

        // Constant-time comparison prevents timing side-channel attacks
        byte[] expected = expectedSignature.getBytes(StandardCharsets.UTF_8);
        byte[] actual = signature.getBytes(StandardCharsets.UTF_8);
        if (!MessageDigest.isEqual(expected, actual)) {
            throw new TokenValidationException("invalid token signature");
        }
    }

    private String computeHmac(String data) {
        try {
            Mac mac = Mac.getInstance(HMAC_ALGORITHM);
            SecretKeySpec keySpec = new SecretKeySpec(sharedSecret, HMAC_ALGORITHM);
            mac.init(keySpec);
            byte[] hmacBytes = mac.doFinal(data.getBytes(StandardCharsets.UTF_8));
            return hexEncode(hmacBytes);
        } catch (Exception e) {
            throw new RuntimeException("HMAC computation failed", e);
        }
    }

    private static String hexEncode(byte[] bytes) {
        StringBuilder sb = new StringBuilder(bytes.length * 2);
        for (byte b : bytes) {
            sb.append(String.format("%02x", b));
        }
        return sb.toString();
    }

    private static class TokenValidationException extends Exception {
        TokenValidationException(String message) {
            super(message);
        }
    }
}
