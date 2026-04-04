#!/bin/bash

set -e

CERTS_DIR="$(dirname "$0")/certs"
CA_KEY="$CERTS_DIR/ca.key"
CA_CERT="$CERTS_DIR/ca.crt"
SERVER_KEY="$CERTS_DIR/server.key"
SERVER_CERT="$CERTS_DIR/server.crt"
CLIENT_KEY="$CERTS_DIR/client.key"
CLIENT_CERT="$CERTS_DIR/client.crt"
INVALID_CLIENT_KEY="$CERTS_DIR/invalid-client.key"
INVALID_CLIENT_CERT="$CERTS_DIR/invalid-client.crt"
EXPIRED_CLIENT_KEY="$CERTS_DIR/expired-client.key"
EXPIRED_CLIENT_CERT="$CERTS_DIR/expired-client.crt"

mkdir -p "$CERTS_DIR"

echo "Generating CA key and certificate..."
openssl genrsa -out "$CA_KEY" 4096
openssl req -new -x509 -days 365 -key "$CA_KEY" -out "$CA_CERT" \
  -subj "/C=US/ST=California/L=San Francisco/O=Test/CN=Test CA"

echo "Generating server key and certificate..."
openssl genrsa -out "$SERVER_KEY" 4096
openssl req -new -key "$SERVER_KEY" -out "$CERTS_DIR/server.csr" \
  -subj "/C=US/ST=California/L=San Francisco/O=Test/CN=localhost"
openssl x509 -req -days 365 -in "$CERTS_DIR/server.csr" -CA "$CA_CERT" -CAkey "$CA_KEY" -CAcreateserial -out "$SERVER_CERT"

echo "Generating valid client key and certificate..."
openssl genrsa -out "$CLIENT_KEY" 4096
openssl req -new -key "$CLIENT_KEY" -out "$CERTS_DIR/client.csr" \
  -subj "/C=US/ST=California/L=San Francisco/O=Test/CN=test-client"
openssl x509 -req -days 365 -in "$CERTS_DIR/client.csr" -CA "$CA_CERT" -CAkey "$CA_KEY" -CAcreateserial -out "$CLIENT_CERT"

echo "Generating invalid client key and certificate (CN not in allowlist)..."
openssl genrsa -out "$INVALID_CLIENT_KEY" 4096
openssl req -new -key "$INVALID_CLIENT_KEY" -out "$CERTS_DIR/invalid-client.csr" \
  -subj "/C=US/ST=California/L=San Francisco/O=Test/CN=malicious-client"
openssl x509 -req -days 365 -in "$CERTS_DIR/invalid-client.csr" -CA "$CA_CERT" -CAkey "$CA_KEY" -CAcreateserial -out "$INVALID_CLIENT_CERT"

echo "Generating expired client key and certificate..."
openssl genrsa -out "$EXPIRED_CLIENT_KEY" 4096
openssl req -new -key "$EXPIRED_CLIENT_KEY" -out "$CERTS_DIR/expired-client.csr" \
  -subj "/C=US/ST=California/L=San Francisco/O=Test/CN=test-client"
openssl x509 -req -days -365 -in "$CERTS_DIR/expired-client.csr" -CA "$CA_CERT" -CAkey "$CA_KEY" -CAcreateserial -out "$EXPIRED_CLIENT_CERT"

echo "Cleaning up temporary CSR files..."
rm -f "$CERTS_DIR"/*.csr
rm -f "$CERTS_DIR"/ca.srl

echo "Certificate generation complete!"
echo "Generated files:"
ls -la "$CERTS_DIR"
