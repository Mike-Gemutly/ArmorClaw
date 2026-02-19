#!/bin/bash
# ArmorClaw Matrix Homeserver Deployment Script
# Step 1: Deploy Standard Matrix Infrastructure
#
# Usage: ./deploy-matrix.sh [options]
#
# Options:
#   --homeserver=conduit|synapse   Choose homeserver (default: conduit)
#   --domain=example.com           Matrix server domain
#   --email=admin@example.com      Admin email for Let's Encrypt
#   --postgres-password=SECRET     PostgreSQL password
#   --turn-secret=SECRET           TURN server secret
#   --registration-token=TOKEN     Registration token (optional)
#   --dry-run                      Show commands without executing

set -euo pipefail

# ========================================
# CONFIGURATION
# ========================================
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

# Default values
HOMESERVER="conduit"
DOMAIN=""
EMAIL=""
POSTGRES_PASSWORD=""
TURN_SECRET=""
REGISTRATION_TOKEN=""
DRY_RUN=false

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# ========================================
# ARGUMENT PARSING
# ========================================
for arg in "$@"; do
    case $arg in
        --homeserver=*)
            HOMESERVER="${arg#*=}"
            shift
            ;;
        --domain=*)
            DOMAIN="${arg#*=}"
            shift
            ;;
        --email=*)
            EMAIL="${arg#*=}"
            shift
            ;;
        --postgres-password=*)
            POSTGRES_PASSWORD="${arg#*=}"
            shift
            ;;
        --turn-secret=*)
            TURN_SECRET="${arg#*=}"
            shift
            ;;
        --registration-token=*)
            REGISTRATION_TOKEN="${arg#*=}"
            shift
            ;;
        --dry-run)
            DRY_RUN=true
            shift
            ;;
        *)
            echo "Unknown argument: $arg"
            exit 1
            ;;
    esac
done

# ========================================
# HELPER FUNCTIONS
# ========================================
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

run_cmd() {
    if $DRY_RUN; then
        echo "[DRY-RUN] $*"
    else
        "$@"
    fi
}

check_command() {
    if ! command -v "$1" &> /dev/null; then
        log_error "$1 is not installed"
        exit 1
    fi
}

generate_secret() {
    openssl rand -hex 32
}

# ========================================
# PREREQUISITE CHECKS
# ========================================
check_prerequisites() {
    log_info "Checking prerequisites..."

    check_command docker
    check_command docker-compose
    check_command openssl

    # Check Docker is running
    if ! docker info &> /dev/null; then
        log_error "Docker daemon is not running"
        exit 1
    fi

    log_success "Prerequisites OK"
}

# ========================================
# VALIDATE INPUT
# ========================================
validate_input() {
    log_info "Validating input..."

    if [[ -z "$DOMAIN" ]]; then
        log_error "Domain is required. Use --domain=example.com"
        exit 1
    fi

    if [[ -z "$EMAIL" ]]; then
        log_error "Email is required for Let's Encrypt. Use --email=admin@example.com"
        exit 1
    fi

    if [[ "$HOMESERVER" != "conduit" && "$HOMESERVER" != "synapse" ]]; then
        log_error "Invalid homeserver: $HOMESERVER. Must be 'conduit' or 'synapse'"
        exit 1
    fi

    log_success "Input validation OK"
}

# ========================================
# GENERATE SECRETS
# ========================================
generate_secrets() {
    log_info "Generating secrets..."

    if [[ -z "$POSTGRES_PASSWORD" ]]; then
        POSTGRES_PASSWORD=$(generate_secret)
        log_warning "Generated PostgreSQL password (save this!)"
    fi

    if [[ -z "$TURN_SECRET" ]]; then
        TURN_SECRET=$(generate_secret)
        log_warning "Generated TURN secret (save this!)"
    fi

    log_success "Secrets generated"
}

# ========================================
# CREATE ENVIRONMENT FILE
# ========================================
create_env_file() {
    log_info "Creating environment file..."

    ENV_FILE="${SCRIPT_DIR}/.env"

    cat > "$ENV_FILE" << EOF
# ArmorClaw Matrix Infrastructure
# Generated: $(date -u +"%Y-%m-%dT%H:%M:%SZ")

# Matrix Configuration
MATRIX_SERVER_NAME=${DOMAIN}
ALLOW_REGISTRATION=false
ALLOW_FEDERATION=true
ALLOW_PUBLIC_ROOMS=true
LOG_LEVEL=warn

# PostgreSQL Configuration
POSTGRES_USER=armorclaw
POSTGRES_PASSWORD=${POSTGRES_PASSWORD}
POSTGRES_DB=matrix

# TURN Server
TURN_SECRET=${TURN_SECRET}

# Synapse-specific (only used if --homeserver=synapse)
REGISTRATION_SHARED_SECRET=$(generate_secret)
MACAROON_SECRET=$(generate_secret)
FORM_SECRET=$(generate_secret)
EOF

    if [[ -n "$REGISTRATION_TOKEN" ]]; then
        echo "REGISTRATION_TOKEN=${REGISTRATION_TOKEN}" >> "$ENV_FILE"
    fi

    chmod 600 "$ENV_FILE"
    log_success "Environment file created: $ENV_FILE"
}

# ========================================
# SETUP DIRECTORIES
# ========================================
setup_directories() {
    log_info "Setting up directories..."

    mkdir -p "${SCRIPT_DIR}/configs/nginx/ssl"
    mkdir -p "${SCRIPT_DIR}/configs/nginx/conf.d"

    log_success "Directories created"
}

# ========================================
# GENERATE TLS CERTIFICATES
# ========================================
setup_tls() {
    log_info "Setting up TLS certificates..."

    # Create dummy certificates for initial startup
    # These will be replaced by Let's Encrypt
    SSL_DIR="${SCRIPT_DIR}/configs/nginx/ssl"

    if [[ ! -f "${SSL_DIR}/fullchain.pem" ]]; then
        log_info "Creating self-signed certificate for initial setup..."

        run_cmd openssl req -x509 -nodes -days 1 -newkey rsa:2048 \
            -keyout "${SSL_DIR}/privkey.pem" \
            -out "${SSL_DIR}/fullchain.pem" \
            -subj "/CN=${DOMAIN}"

        log_warning "Self-signed certificate created. Will be replaced by Let's Encrypt."
    fi

    log_success "TLS setup complete"
}

# ========================================
# INITIALIZE HOMESERVER
# ========================================
initialize_homeserver() {
    log_info "Initializing ${HOMESERVER}..."

    if [[ "$HOMESERVER" == "synapse" ]]; then
        # Generate Synapse signing key
        run_cmd docker run --rm \
            -v "${SCRIPT_DIR}/data/synapse:/data" \
            -e SYNAPSE_SERVER_NAME="${DOMAIN}" \
            -e SYNAPSE_REPORT_STATS=no \
            matrixorg/synapse:latest generate

        log_success "Synapse signing key generated"
    fi

    # Conduit doesn't need special initialization
    log_success "Homeserver initialization complete"
}

# ========================================
# START SERVICES
# ========================================
start_services() {
    log_info "Starting Matrix services..."

    cd "${SCRIPT_DIR}"

    # Pull images first
    run_cmd docker-compose -f docker-compose.matrix.yml --profile "${HOMESERVER}" pull

    # Start core services
    run_cmd docker-compose -f docker-compose.matrix.yml --profile "${HOMESERVER}" up -d postgres nginx certbot

    log_info "Waiting for PostgreSQL..."
    sleep 10

    # Start homeserver
    run_cmd docker-compose -f docker-compose.matrix.yml --profile "${HOMESERVER}" up -d "${HOMESERVER}"

    # Start TURN server
    run_cmd docker-compose -f docker-compose.matrix.yml --profile "${HOMESERVER}" up -d coturn

    log_success "Services started"
}

# ========================================
# REQUEST LET'S ENCRYPT CERTIFICATE
# ========================================
request_certificate() {
    log_info "Requesting Let's Encrypt certificate..."

    if $DRY_RUN; then
        log_info "[DRY-RUN] Would request certificate for ${DOMAIN}"
        return
    fi

    # Wait for nginx to be ready
    sleep 5

    # Request certificate
    run_cmd docker run --rm \
        -v "${SCRIPT_DIR}/certbot_etc:/etc/letsencrypt" \
        -v "${SCRIPT_DIR}/certbot_data:/var/www/certbot" \
        certbot/certbot certonly \
        --webroot \
        --webroot-path /var/www/certbot \
        --email "${EMAIL}" \
        --agree-tos \
        --no-eff-email \
        -d "${DOMAIN}"

    # Reload nginx with real certificates
    run_cmd docker-compose -f docker-compose.matrix.yml exec nginx nginx -s reload

    log_success "Certificate obtained"
}

# ========================================
# CREATE FIRST ADMIN USER
# ========================================
create_admin_user() {
    log_info "Creating admin user..."

    if [[ -z "$REGISTRATION_TOKEN" ]]; then
        log_warning "No registration token set. Skipping admin user creation."
        log_info "Set a registration token and restart to create users."
        return
    fi

    log_info "To create an admin user, run:"
    echo ""
    echo "  # Register user:"
    echo "  curl -X POST https://${DOMAIN}/_matrix/client/v3/register \\"
    echo "    -d '{\"username\":\"admin\",\"password\":\"YOUR_PASSWORD\",\"auth\":{\"type\":\"m.login.registration_token\",\"token\":\"${REGISTRATION_TOKEN}\"}}'"
    echo ""
    echo "  # Promote to admin (Synapse only):"
    echo "  docker-compose -f docker-compose.matrix.yml exec synapse register_new_matrix_user -u admin -a"
    echo ""
}

# ========================================
# VERIFY DEPLOYMENT
# ========================================
verify_deployment() {
    log_info "Verifying deployment..."

    # Check homeserver is responding
    log_info "Checking homeserver health..."

    if curl -sf "https://${DOMAIN}/_matrix/client/versions" > /dev/null 2>&1; then
        log_success "Homeserver is responding"
    else
        log_warning "Homeserver not yet responding (may need more time)"
    fi

    # Check well-known
    log_info "Checking well-known configuration..."

    if curl -sf "https://${DOMAIN}/.well-known/matrix/client" | grep -q "m.homeserver"; then
        log_success "Well-known client configuration OK"
    else
        log_warning "Well-known client configuration not found"
    fi

    if curl -sf "https://${DOMAIN}/.well-known/matrix/server" | grep -q "m.server"; then
        log_success "Well-known server configuration OK"
    else
        log_warning "Well-known server configuration not found"
    fi

    # Check federation port
    log_info "Checking federation port..."

    if curl -sf "https://${DOMAIN}:8448/_matrix/key/v2/server" > /dev/null 2>&1; then
        log_success "Federation port OK"
    else
        log_warning "Federation port not responding (may need firewall config)"
    fi

    log_success "Verification complete"
}

# ========================================
# PRINT SUMMARY
# ========================================
print_summary() {
    echo ""
    echo "========================================"
    echo "  ArmorClaw Matrix Deployment Complete"
    echo "========================================"
    echo ""
    echo "Homeserver: ${HOMESERVER}"
    echo "Domain: ${DOMAIN}"
    echo ""
    echo "Client URL: https://${DOMAIN}"
    echo "Federation URL: https://${DOMAIN}:8448"
    echo ""
    echo "Configuration saved to: ${SCRIPT_DIR}/.env"
    echo ""
    echo "Useful commands:"
    echo "  View logs:     docker-compose -f docker-compose.matrix.yml logs -f ${HOMESERVER}"
    echo "  Stop services: docker-compose -f docker-compose.matrix.yml down"
    echo "  Restart:       docker-compose -f docker-compose.matrix.yml restart"
    echo ""
    echo "Next steps:"
    echo "  1. Create admin user (see instructions above)"
    echo "  2. Configure ArmorChat to use https://${DOMAIN}"
    echo "  3. Test federation with another homeserver"
    echo ""
}

# ========================================
# MAIN
# ========================================
main() {
    echo ""
    echo "========================================"
    echo "  ArmorClaw Matrix Infrastructure"
    echo "  Step 1: Deploy Standard Matrix"
    echo "========================================"
    echo ""

    check_prerequisites
    validate_input
    generate_secrets
    create_env_file
    setup_directories
    setup_tls
    initialize_homeserver
    start_services
    request_certificate
    create_admin_user
    verify_deployment
    print_summary
}

main "$@"
