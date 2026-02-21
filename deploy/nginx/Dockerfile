# ArmorClaw Nginx Reverse Proxy
# Includes environment variable substitution for dynamic configuration
FROM nginx:alpine

# Install gettext for envsubst
RUN apk add --no-cache gettext

# Copy entrypoint script
COPY entrypoint.sh /docker-entrypoint.sh
RUN chmod +x /docker-entrypoint.sh

# Create templates directory
RUN mkdir -p /etc/nginx/templates

# Default environment values (can be overridden)
ENV MATRIX_SERVER_NAME=matrix.armorclaw.com

# Use our entrypoint
ENTRYPOINT ["/docker-entrypoint.sh"]
