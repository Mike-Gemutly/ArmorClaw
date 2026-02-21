#!/bin/sh
# ArmorClaw Nginx Entrypoint
# Handles environment variable substitution in nginx configs

set -e

# Default values for required environment variables
: ${MATRIX_SERVER_NAME:="matrix.armorclaw.com"}
: ${TURN_SERVER_NAME:="turn.${MATRIX_SERVER_NAME}"}

echo "ArmorClaw Nginx Configuration"
echo "  MATRIX_SERVER_NAME: ${MATRIX_SERVER_NAME}"
echo "  TURN_SERVER_NAME: ${TURN_SERVER_NAME}"

# Process main nginx.conf template
if [ -f "/etc/nginx/nginx.conf.template" ]; then
    echo "Processing nginx.conf.template..."
    envsubst '${MATRIX_SERVER_NAME} ${TURN_SERVER_NAME}' < /etc/nginx/nginx.conf.template > /etc/nginx/nginx.conf
fi

# Process template directory (for additional configs)
TEMPLATE_DIR="/etc/nginx/templates"
if [ -d "$TEMPLATE_DIR" ]; then
    echo "Processing additional templates..."
    for template in "$TEMPLATE_DIR"/*.template; do
        if [ -f "$template" ]; then
            filename=$(basename "$template" .template)
            echo "  - $filename"
            envsubst '${MATRIX_SERVER_NAME} ${TURN_SERVER_NAME}' < "$template" > "/etc/nginx/conf.d/$filename"
        fi
    done
fi

# Verify required configs exist
if [ ! -f "/etc/nginx/nginx.conf" ]; then
    echo "ERROR: /etc/nginx/nginx.conf not found"
    exit 1
fi

# Test nginx configuration
echo "Testing nginx configuration..."
nginx -t

echo "Starting nginx..."
exec nginx -g 'daemon off;'
