# ---- Stage 1: Builder — compile Rust vault binary ----
FROM rust:1.77-slim AS builder

RUN apt-get update && apt-get install -y protobuf-compiler && rm -rf /var/lib/apt/lists/*

WORKDIR /build
COPY Cargo.toml Cargo.lock* build.rs ./
COPY proto/ proto/
COPY src/ src/

ENV PROTOC=/usr/bin/protoc
RUN cargo build --release && mv target/release/armorclaw-vault /armorclaw-vault

# ---- Stage 2: Runtime — minimal hardened image ----
FROM debian:bookworm-slim

RUN useradd --system --uid 10001 --no-create-home vault

COPY --from=builder /armorclaw-vault /usr/local/bin/armorclaw-vault

RUN mkdir -p /run/armorclaw && chown vault:vault /run/armorclaw

USER vault

HEALTHCHECK --interval=10s --timeout=3s --retries=3 --start-period=5s \
  CMD test -S /run/armorclaw/keystore.sock || exit 1

CMD ["armorclaw-vault"]
