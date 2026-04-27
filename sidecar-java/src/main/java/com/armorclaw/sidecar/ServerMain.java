package com.armorclaw.sidecar;

import io.grpc.Server;
import io.grpc.netty.NettyServerBuilder;
import io.netty.channel.epoll.EpollEventLoopGroup;
import io.netty.channel.unix.DomainSocketAddress;

import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.attribute.PosixFilePermission;
import java.util.Set;
import java.util.concurrent.TimeUnit;

/**
 * Entry point for the ArmorClaw Java Sidecar.
 *
 * <p>Binds a gRPC server to a Unix Domain Socket with 0600 permissions.
 * Graceful shutdown drains for up to 30 seconds.
 */
public class ServerMain {

    private static final String DEFAULT_SOCKET_PATH =
            "/run/armorclaw/sidecar-java/sidecar-java.sock";
    private static final int SHUTDOWN_TIMEOUT_SECONDS = 30;

    public static void main(String[] args) throws Exception {
        String socketPath = System.getenv("SIDECAR_JAVA_SOCKET_PATH");
        if (socketPath == null || socketPath.isBlank()) {
            socketPath = DEFAULT_SOCKET_PATH;
        }

        Path socketFile = Path.of(socketPath);
        Files.createDirectories(socketFile.getParent());

        EpollEventLoopGroup bossGroup = new EpollEventLoopGroup(1);
        EpollEventLoopGroup workerGroup = new EpollEventLoopGroup();

        Server server = NettyServerBuilder.forAddress(new DomainSocketAddress(socketPath))
                .bossEventLoopGroup(bossGroup)
                .workerEventLoopGroup(workerGroup)
                .channelType(io.netty.channel.epoll.EpollServerDomainSocketChannel.class)
                .intercept(new TokenInterceptor())
                .intercept(new VersionInterceptor())
                .addService(new ExtractorService())
                .build();

        Runtime.getRuntime().addShutdownHook(new Thread(() -> {
            System.out.println("Shutting down gRPC server...");
            server.shutdown();
            try {
                if (!server.awaitTermination(SHUTDOWN_TIMEOUT_SECONDS, TimeUnit.SECONDS)) {
                    System.out.println("Forcing shutdown after " + SHUTDOWN_TIMEOUT_SECONDS + "s drain");
                    server.shutdownNow();
                }
            } catch (InterruptedException e) {
                server.shutdownNow();
            }
            bossGroup.shutdownGracefully(0, SHUTDOWN_TIMEOUT_SECONDS, TimeUnit.SECONDS);
            workerGroup.shutdownGracefully(0, SHUTDOWN_TIMEOUT_SECONDS, TimeUnit.SECONDS);
        }));

        server.start();

        Files.setPosixFilePermissions(socketFile, Set.of(
                PosixFilePermission.OWNER_READ,
                PosixFilePermission.OWNER_WRITE
        ));

        System.out.println("ArmorClaw Java Sidecar started on " + socketPath);

        server.awaitTermination();
    }
}
