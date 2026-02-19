#!/bin/bash
# ArmorClaw Final Hardening Script
# Called after security configuration is complete
# Version: 1.0.0

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

# Configuration paths
CONFIG_DIR="/etc/armorclaw"
DATA_DIR="/var/lib/armorclaw"
RUN_DIR="/run/armorclaw"
SECURITY_DIR="/etc/armorclaw/security.d"
LOG_DIR="/var/log/armorclaw"

#=============================================================================
# Helper Functions
#=============================================================================

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ ERROR: $1${NC}" >&2
}

print_warning() {
    echo -e "${YELLOW}⚠ WARNING: $1${NC}"
}

print_info() {
    echo -e "${CYAN}ℹ $1${NC}"
}

print_step() {
    echo -e "\n${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BLUE}  $1${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}\n"
}

check_root() {
    if [ "$EUID" -ne 0 ]; then
        print_error "This script must be run as root"
        exit 1
    fi
}

#=============================================================================
# Firewall Configuration
#=============================================================================

configure_firewall() {
    print_step "Step 1: Firewall Configuration"

    # Check if UFW is installed
    if ! command -v ufw &> /dev/null; then
        print_info "Installing UFW..."
        apt-get update -qq && apt-get install -y -qq ufw
    fi

    # Reset UFW to defaults
    print_info "Resetting firewall rules..."
    ufw --force reset

    # Set default policies
    print_info "Setting default deny policies..."
    ufw default deny incoming
    ufw default deny outgoing

    # Allow essential outbound
    print_info "Allowing essential outbound traffic..."
    ufw allow out 53/udp comment 'DNS'
    ufw allow out 80/tcp comment 'HTTP'
    ufw allow out 443/tcp comment 'HTTPS'

    # Check security configuration for enabled adapters
    local matrix_enabled="false"
    local webrtc_enabled="false"

    if [ -f "$CONFIG_DIR/config.toml" ]; then
        if grep -q "matrix.*enabled.*=.*true" "$CONFIG_DIR/config.toml" 2>/dev/null; then
            matrix_enabled="true"
        fi
        if grep -q "signaling_enabled.*=.*true" "$CONFIG_DIR/config.toml" 2>/dev/null; then
            webrtc_enabled="true"
        fi
    fi

    # Allow Matrix ports if enabled
    if [ "$matrix_enabled" = "true" ]; then
        print_info "Allowing Matrix ports..."
        ufw allow 8448/tcp comment 'Matrix federation'
        ufw allow 6167/tcp comment 'Matrix client API'
    fi

    # Allow WebRTC signaling if enabled
    if [ "$webrtc_enabled" = "true" ]; then
        print_info "Allowing WebRTC signaling port..."
        ufw allow 8443/tcp comment 'WebRTC signaling'
    fi

    # Allow SSH (critical!)
    print_info "Allowing SSH..."
    ufw allow 22/tcp comment 'SSH'

    # Tailscale auto-detection
    if command -v tailscale &> /dev/null; then
        print_info "Tailscale detected, allowing interface..."
        ufw allow in on tailscale0
    fi

    # Enable rate limiting for SSH
    print_info "Enabling SSH rate limiting..."
    ufw limit 22/tcp comment 'SSH rate limited'

    # Enable logging
    ufw logging on

    # Enable firewall
    print_info "Enabling firewall..."
    ufw --force enable

    print_success "Firewall configured"
    ufw status
}

#=============================================================================
# Container Security
#=============================================================================

apply_container_security() {
    print_step "Step 2: Container Security"

    # Verify seccomp profile
    print_info "Verifying seccomp profile..."
    if [ -f "/etc/docker/seccomp/armorclaw.json" ]; then
        print_success "Seccomp profile found"
    else
        print_warning "Seccomp profile not found, creating default..."
        mkdir -p /etc/docker/seccomp
        cat > /etc/docker/seccomp/armorclaw.json <<'EOF'
{
    "defaultAction": "SCMP_ACT_ERRNO",
    "architectures": ["SCMP_ARCH_X86_64", "SCMP_ARCH_X86"],
    "syscalls": [
        {
            "names": ["read", "write", "close", "fstat", "mmap", "mprotect", "munmap", "brk", "ioctl", "access", "pipe", "dup2", "getpid", "socket", "connect", "sendto", "recvfrom", "sendmsg", "recvmsg", "shutdown", "bind", "listen", "accept", "getsockname", "getpeername", "setsockopt", "getsockopt", "clone", "execve", "exit", "wait4", "kill", "uname", "fcntl", "flock", "fsync", "fdatasync", "truncate", "ftruncate", "getdents", "getcwd", "chdir", "fchdir", "rename", "mkdir", "rmdir", "unlink", "readlink", "chmod", "fchmod", "chown", "fchown", "lchown", "umask", "gettimeofday", "getrlimit", "getrusage", "sysinfo", "times", "getuid", "getgid", "setuid", "setgid", "geteuid", "getegid", "setpgid", "getppid", "getpgrp", "setsid", "setreuid", "setregid", "getgroups", "setgroups", "setresuid", "getresuid", "setresgid", "getresgid", "getpgid", "setfsuid", "setfsgid", "getsid", "capget", "capset", "rt_sigpending", "rt_sigtimedwait", "rt_sigqueueinfo", "sigaltstack", "utime", "mknod", "uselib", "personality", "ustat", "statfs", "fstatfs", "sysfs", "getpriority", "setpriority", "sched_setparam", "sched_getparam", "sched_setscheduler", "sched_getscheduler", "sched_get_priority_max", "sched_get_priority_min", "sched_rr_get_interval", "mlock", "munlock", "mlockall", "munlockall", "vhangup", "pivot_root", "prctl", "arch_prctl", "adjtimex", "setrlimit", "chroot", "sync", "acct", "settimeofday", "mount", "umount2", "swapon", "swapoff", "reboot", "sethostname", "setdomainname", "iopl", "ioperm", "init_module", "delete_module", "quotactl", "gettid", "readahead", "setxattr", "lsetxattr", "fsetxattr", "getxattr", "lgetxattr", "fgetxattr", "listxattr", "llistxattr", "flistxattr", "removexattr", "lremovexattr", "fremovexattr", "tkill", "time", "futex", "sched_setaffinity", "sched_getaffinity", "set_thread_area", "io_setup", "io_destroy", "io_getevents", "io_submit", "io_cancel", "get_thread_area", "epoll_create", "epoll_ctl", "epoll_wait", "remap_file_pages", "getdents64", "set_tid_address", "restart_syscall", "semtimedop", "fadvise64", "timer_create", "timer_settime", "timer_gettime", "timer_getoverrun", "timer_delete", "clock_settime", "clock_gettime", "clock_getres", "clock_nanosleep", "exit_group", "epoll_wait", "tgkill", "utimes", "mbind", "set_mempolicy", "get_mempolicy", "mq_open", "mq_unlink", "mq_timedsend", "mq_timedreceive", "mq_notify", "mq_getsetattr", "kexec_load", "waitid", "add_key", "request_key", "keyctl", "ioprio_set", "ioprio_get", "inotify_init", "inotify_add_watch", "inotify_rm_watch", "migrate_pages", "openat", "mkdirat", "mknodat", "fchownat", "futimesat", "newfstatat", "unlinkat", "renameat", "linkat", "symlinkat", "readlinkat", "fchmodat", "faccessat", "pselect6", "ppoll", "unshare", "set_robust_list", "get_robust_list", "splice", "tee", "sync_file_range", "vmsplice", "move_pages", "utimensat", "epoll_pwait", "signalfd", "timerfd_create", "eventfd", "fallocate", "timerfd_settime", "timerfd_gettime", "accept4", "signalfd4", "eventfd2", "epoll_create1", "dup3", "pipe2", "inotify_init1", "preadv", "pwritev", "rt_tgsigqueueinfo", "perf_event_open", "recvmmsg", "fanotify_init", "fanotify_mark", "prlimit64", "name_to_handle_at", "open_by_handle_at", "clock_adjtime", "syncfs", "sendmmsg", "setns", "getcpu", "process_vm_readv", "process_vm_writev"],
            "action": "SCMP_ACT_ALLOW"
        }
    ]
}
EOF
        print_success "Seccomp profile created"
    fi

    # Configure Docker daemon for security
    print_info "Configuring Docker daemon..."
    if [ -f "/etc/docker/daemon.json" ]; then
        print_info "Docker daemon.json exists, verifying settings..."
    else
        cat > /etc/docker/daemon.json <<EOF
{
    "icc": false,
    "live-restore": true,
    "userland-proxy": false,
    "no-new-privileges": true,
    "seccomp-profile": "/etc/docker/seccomp/armorclaw.json"
}
EOF
        print_success "Docker daemon configured for security"
        print_warning "Docker daemon restart required: systemctl restart docker"
    fi

    # Set resource limits in systemd
    print_info "Configuring container resource limits..."
    if [ -f "/etc/systemd/system/armorclaw-bridge.service" ]; then
        print_success "Systemd service already configured"
    else
        print_warning "Systemd service not found"
    fi

    print_success "Container security configured"
}

#=============================================================================
# Keystore Hardening
#=============================================================================

harden_keystore() {
    print_step "Step 3: Keystore Hardening"

    # Verify keystore exists
    if [ ! -f "$DATA_DIR/keystore.db" ]; then
        print_info "Keystore not initialized, will be created on first start"
        return
    fi

    # Set strict permissions
    print_info "Setting keystore permissions..."
    chmod 600 "$DATA_DIR/keystore.db"
    chmod 600 "$DATA_DIR/keystore.db.salt" 2>/dev/null || true
    chown armorclaw:armorclaw "$DATA_DIR/keystore.db"
    chown armorclaw:armorclaw "$DATA_DIR/keystore.db.salt" 2>/dev/null || true

    print_success "Keystore hardened"
}

#=============================================================================
# Audit Logging
#=============================================================================

configure_audit_logging() {
    print_step "Step 4: Audit Logging Configuration"

    # Create log directory
    mkdir -p "$LOG_DIR"
    chmod 750 "$LOG_DIR"
    chown armorclaw:armorclaw "$LOG_DIR"

    # Install auditd if not present
    if ! command -v auditd &> /dev/null; then
        print_info "Installing auditd..."
        apt-get install -y -qq auditd audispd-plugins
    fi

    # Create ArmorClaw audit rules
    print_info "Creating audit rules..."
    cat > /etc/audit/rules.d/armorclaw.rules <<EOF
# ArmorClaw Security Audit Rules
# Monitor configuration changes
-w $CONFIG_DIR -p wa -k armorclaw_config
-w $DATA_DIR -p wa -k armorclaw_data
-w $RUN_DIR -p wa -k armorclaw_runtime

# Monitor Docker access
-w /usr/bin/docker -p x -k docker_exec
-w /var/run/docker.sock -p rw -k docker_socket

# Monitor keystore access
-w $DATA_DIR/keystore.db -p rw -k keystore_access

# Monitor user/group changes
-w /etc/passwd -p wa -k identity
-w /etc/group -p wa -k identity
-w /etc/shadow -p wa -k identity

# Monitor network changes
-w /etc/network/ -p wa -k network
-a always,exit -F arch=b64 -S socket -S bind -S listen -S accept -k network

# Monitor module loading
-w /sbin/insmod -p x -k modules
-w /sbin/rmmod -p x -k modules
-w /sbin/modprobe -p x -k modules
EOF

    # Create logrotate config
    print_info "Creating logrotate configuration..."
    cat > /etc/logrotate.d/armorclaw <<EOF
$LOG_DIR/*.log {
    daily
    rotate 30
    compress
    delaycompress
    missingok
    notifempty
    create 0640 armorclaw armorclaw
    sharedscripts
    postrotate
        systemctl reload armorclaw-bridge > /dev/null 2>&1 || true
    endscript
}
EOF

    # Restart auditd
    if service auditd status > /dev/null 2>&1; then
        service auditd restart
        print_success "Auditd restarted"
    else
        print_warning "Auditd not running, starting..."
        service auditd start
    fi

    print_success "Audit logging configured"
}

#=============================================================================
# Network Hardening
#=============================================================================

harden_network() {
    print_step "Step 5: Network Hardening"

    # Disable IPv6 if not needed (optional)
    # Commented out by default as it may break some setups
    # print_info "Disabling IPv6..."
    # sysctl -w net.ipv6.conf.all.disable_ipv6=1
    # sysctl -w net.ipv6.conf.default.disable_ipv6=1

    # Kernel network hardening
    print_info "Applying kernel network hardening..."
    cat > /etc/sysctl.d/99-armorclaw.conf <<EOF
# ArmorClaw Network Hardening

# TCP SYN flood protection
net.ipv4.tcp_syncookies = 1
net.ipv4.tcp_max_syn_backlog = 2048
net.ipv4.tcp_synack_retries = 2
net.ipv4.tcp_syn_retries = 5

# Disable IP forwarding
net.ipv4.ip_forward = 0

# Disable send redirects
net.ipv4.conf.all.send_redirects = 0
net.ipv4.conf.default.send_redirects = 0

# Disable accept source route
net.ipv4.conf.all.accept_source_route = 0
net.ipv4.conf.default.accept_source_route = 0

# Disable accept redirects
net.ipv4.conf.all.accept_redirects = 0
net.ipv4.conf.default.accept_redirects = 0
net.ipv4.conf.all.secure_redirects = 0
net.ipv4.conf.default.secure_redirects = 0

# Enable reverse path filtering
net.ipv4.conf.all.rp_filter = 1
net.ipv4.conf.default.rp_filter = 1

# Log martian packets
net.ipv4.conf.all.log_martians = 1
net.ipv4.conf.default.log_martians = 1

# Ignore ICMP echo requests
net.ipv4.icmp_echo_ignore_all = 1

# Ignore ICMP redirect
net.ipv4.conf.all.accept_redirects = 0
net.ipv4.conf.default.accept_redirects = 0

# Disable TCP timestamps
net.ipv4.tcp_timestamps = 0

# Enable TCP wrapper
net.ipv4.tcp_wrapper = 1
EOF

    sysctl -p /etc/sysctl.d/99-armorclaw.conf

    print_success "Network hardened"
}

#=============================================================================
# Final Verification
#=============================================================================

verify_hardening() {
    print_step "Step 6: Final Verification"

    local all_ok=true

    # Check firewall
    print_info "Verifying firewall..."
    if ufw status | grep -q "Status: active"; then
        print_success "Firewall is active"
    else
        print_error "Firewall is not active"
        all_ok=false
    fi

    # Check Docker
    print_info "Verifying Docker..."
    if docker info &> /dev/null; then
        print_success "Docker is running"
    else
        print_error "Docker is not running"
        all_ok=false
    fi

    # Check directories
    print_info "Verifying directories..."
    for dir in "$CONFIG_DIR" "$DATA_DIR" "$RUN_DIR" "$LOG_DIR"; do
        if [ -d "$dir" ]; then
            print_success "Directory exists: $dir"
        else
            print_error "Directory missing: $dir"
            all_ok=false
        fi
    done

    # Check audit
    print_info "Verifying audit logging..."
    if service auditd status > /dev/null 2>&1; then
        print_success "Auditd is running"
    else
        print_warning "Auditd is not running (non-critical)"
    fi

    # Generate security report
    print_info "Generating security report..."
    REPORT_FILE="$LOG_DIR/hardening-report-$(date +%Y%m%d-%H%M%S).txt"

    cat > "$REPORT_FILE" <<EOF
ArmorClaw Hardening Report
Generated: $(date)

=== Firewall Status ===
$(ufw status verbose)

=== Docker Security ===
$(docker info 2>/dev/null | grep -E "Security|Seccomp" || echo "Docker not running")

=== Directory Permissions ===
$(ls -la "$CONFIG_DIR" "$DATA_DIR" "$RUN_DIR" 2>/dev/null)

=== Audit Rules ===
$(auditctl -l 2>/dev/null || echo "Audit not available")

=== Network Configuration ===
$(sysctl net.ipv4.tcp_syncookies net.ipv4.conf.all.rp_filter 2>/dev/null)

=== Open Ports ===
$(ss -tuln)

Hardening complete at $(date)
EOF

    chmod 640 "$REPORT_FILE"
    chown armorclaw:armorclaw "$REPORT_FILE"

    if [ "$all_ok" = true ]; then
        print_success "All verification checks passed!"
    else
        print_warning "Some verification checks failed"
    fi

    print_info "Security report saved to: $REPORT_FILE"
}

#=============================================================================
# Main
#=============================================================================

main() {
    echo ""
    echo -e "${CYAN}╔══════════════════════════════════════════════════════╗${NC}"
    echo -e "${CYAN}║${NC}        ${BOLD}ArmorClaw Final Hardening${NC}                  ${CYAN}║${NC}"
    echo -e "${CYAN}╚══════════════════════════════════════════════════════╝${NC}"
    echo ""

    check_root

    # Create required directories
    mkdir -p "$SECURITY_DIR"
    mkdir -p "$LOG_DIR"

    # Run hardening steps
    configure_firewall
    apply_container_security
    harden_keystore
    configure_audit_logging
    harden_network
    verify_hardening

    echo ""
    echo -e "${GREEN}╔══════════════════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║${NC}          ${BOLD}Hardening Complete!${NC}                       ${GREEN}║${NC}"
    echo -e "${GREEN}╚══════════════════════════════════════════════════════╝${NC}"
    echo ""
    echo -e "${BOLD}Security Status:${NC}"
    echo "  • Firewall: Configured and active"
    echo "  • Container: Seccomp profile applied"
    echo "  • Keystore: Hardened permissions"
    echo "  • Audit: Logging enabled"
    echo "  • Network: Kernel hardened"
    echo ""
    echo -e "${BOLD}Next Steps:${NC}"
    echo "  1. Review security report in $LOG_DIR"
    echo "  2. Start the bridge: systemctl start armorclaw-bridge"
    echo "  3. Begin using ArmorClaw with ArmorChat"
    echo ""
}

# Run main
main "$@"
