---
name: armorclaw_ops
version: 1.0.0
description: Consolidated ArmorClaw lifecycle management — deploy, monitor, remediate, backup, and restore
---

# ArmorClaw Ops Skill

Manage the full lifecycle of an ArmorClaw deployment from any AI assistant. This skill gives you a single entry point for deploying, checking health, restarting services, reading logs, and creating or restoring backups. It works with Claude Code, OpenCode, Cursor, or any tool that discovers skills from `.skills/`.

Six actions are available: **deploy**, **health**, **redeploy**, **logs**, **backup**, and **restore**. Each one connects to your VPS over SSH, runs the right commands, and reports back. Destructive actions (deploy, redeploy, backup, restore) ask for confirmation first. Read-only actions (health, logs) run immediately.

This is for developers who'd rather type a natural language command than SSH into a box and remember docker-compose flags. If you can tell your AI assistant "check health on my VPS," this skill handles the rest.

## Quick Start

```bash
# Check everything is healthy
/ops action=health vps_ip=192.168.1.100

# View recent bridge logs
/ops action=logs vps_ip=192.168.1.100 service=bridge

# Restart a misbehaving service
/ops action=redeploy vps_ip=192.168.1.100 service=bridge

# Deploy ArmorClaw to a new VPS
/ops action=deploy vps_ip=192.168.1.100 mode=sentinel domain=armorclaw.example.com

# Create a backup before making changes
/ops action=backup vps_ip=192.168.1.100

# Restore from a previous backup
/ops action=restore vps_ip=192.168.1.100 backup_path=/opt/armorclaw/backups/armorclaw-20260419.tar.gz
```

## Actions Reference

### deploy

Deploys ArmorClaw to a new VPS or updates an existing one. Delegates to `install.sh` on the remote machine. For sentinel mode (production with HTTPS), you need a domain pointing to the VPS.

**Required:** `vps_ip`

**Optional:** `mode` (native or sentinel), `domain`, `ssh_user`, `ssh_key`

**What to expect:** The skill detects the target platform, validates SSH connectivity, detects the existing topology, then runs the installer. You'll see progress output and a summary when it finishes. Deployment takes 2 to 5 minutes depending on mode.

**Safety:** Asks for confirmation before running the installer on the remote host.

```bash
# Local dev deployment (no TLS)
/ops action=deploy vps_ip=192.168.1.100

# Production deployment with Let's Encrypt
/ops action=deploy vps_ip=5.183.11.149 mode=sentinel domain=armorclaw.example.com

# Custom SSH user and key
/ops action=deploy vps_ip=5.183.11.149 ssh_user=admin ssh_key=~/.ssh/my_key mode=sentinel domain=armorclaw.example.com
```

### health

Checks all services in dependency order and shows a per-service status with a summary at the end. This is the fastest way to answer "is my deployment okay?"

**Required:** `vps_ip`

**Optional:** `verbose` (show detailed diagnostics), `ssh_user`, `ssh_key`

**What to expect:** Each service shows ✓ or ✗. If `verbose=true`, you'll also get version info, uptime, and resource usage. The summary counts healthy vs unhealthy services.

**Safety:** Read-only. Runs immediately without confirmation.

```bash
# Quick health check
/ops action=health vps_ip=192.168.1.100

# Detailed diagnostics
/ops action=health vps_ip=192.168.1.100 verbose=true
```

### redeploy

Restarts services in the correct dependency order so dependent services come up after their dependencies. Supports restarting a single service or the entire stack.

**Required:** `vps_ip`

**Optional:** `service` (name or `all`), `ssh_user`, `ssh_key`

**What to expect:** Services are stopped in reverse dependency order, then started in dependency order. You'll see each stop and start as it happens.

**Safety:** Asks for confirmation before restarting anything. Restarting `all` affects the entire stack, so expect brief downtime.

```bash
# Restart just the bridge
/ops action=redeploy vps_ip=192.168.1.100 service=bridge

# Restart everything (full stack)
/ops action=redeploy vps_ip=192.168.1.100 service=all
```

### logs

Shows docker logs for a specified service. Defaults to the last 100 lines. Use `service=all` for a condensed summary of core services.

**Required:** `vps_ip`

**Optional:** `service` (name or `all`), `tail` (number of lines), `ssh_user`, `ssh_key`

**What to expect:** Raw docker log output. For `service=all`, shows the last 20 lines per core service.

**Safety:** Read-only. Runs immediately without confirmation.

```bash
# Last 100 lines of bridge logs (default)
/ops action=logs vps_ip=192.168.1.100 service=bridge

# More context
/ops action=logs vps_ip=192.168.1.100 service=bridge tail=500

# Summary of all core services
/ops action=logs vps_ip=192.168.1.100 service=all
```

### backup

Creates a tar.gz archive of configs, secrets, and data volumes. The backup includes everything needed to restore the deployment to its current state.

**Required:** `vps_ip`

**Optional:** `backup_path` (auto-generated timestamp path if omitted), `ssh_user`, `ssh_key`

**What to expect:** Creates the archive on the VPS and reports the file path and size. Secrets are masked in the skill output but preserved in the archive itself.

**Safety:** Asks for confirmation before creating the backup. The backup is stored on the VPS, so make sure there's enough disk space.

```bash
# Auto-generated backup with timestamp
/ops action=backup vps_ip=192.168.1.100

# Custom backup location
/ops action=backup vps_ip=192.168.1.100 backup_path=/mnt/backups/armorclaw-20260419.tar.gz
```

### restore

Restores from a previously created backup file. Stops all services, restores configs and data volumes, then starts services in dependency order.

**Required:** `vps_ip`, `backup_path`

**Optional:** `ssh_user`, `ssh_key`

**What to expect:** Services stop, files are restored from the archive, services start again. The whole process takes 1 to 3 minutes depending on data volume size.

**Safety:** Asks for confirmation before proceeding. This stops all services, so expect full downtime during the restore. Make sure the backup file exists at the specified path.

```bash
# Restore from a specific backup
/ops action=restore vps_ip=192.168.1.100 backup_path=/opt/armorclaw/backups/armorclaw-20260419.tar.gz
```

## Parameters Reference

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| action | string | yes | health | Action: deploy\|health\|redeploy\|logs\|backup\|restore |
| vps_ip | string | yes | - | Target VPS IP address or hostname |
| ssh_user | string | no | root | SSH username |
| ssh_key | string | no | ~/.ssh/id_ed25519 | Path to SSH private key |
| service | string | no | all | Target service name (see Service Names table) |
| mode | string | no | native | Deployment mode: native\|sentinel |
| verbose | boolean | no | false | Show detailed diagnostics |
| domain | string | no | - | Public domain for sentinel mode |
| tail | string | no | 100 | Number of log lines to show |
| backup_path | string | no | auto-generated | Backup file path on VPS |

## Service Names

| Friendly Name | Container Name | Health Check |
|--------------|----------------|--------------|
| bridge | armorclaw-sentinel | HTTP :8443/health + socket |
| matrix | armorclaw-conduit | HTTP :6167 |
| vault | armorclaw-vault | Socket :keystore.sock |
| sygnal | armorclaw-sygnal | HTTP :5000/health |
| qdrant | armorclaw-qdrant | HTTP :6333/healthz |
| caddy | armorclaw-caddy | Config validation |
| jetski | jetski | HTTP :9222/health |
| browser | armorclaw-browser | HTTP :3000/health |
| sidecar-office | armorclaw-sidecar-office | Socket :sidecar-office.sock |
| coturn | armorclaw-coturn | UDP :3478 |
| catwalk | armorclaw-catwalk | TCP :8080 |

## Dependency Order

```
Layer 0: Docker → Vault (socket)
Layer 1: Matrix Conduit (HTTP :6167)
Layer 2: Bridge (socket + HTTP :8443)
Layer 3: Sygnal (HTTP :5000) + Qdrant (HTTP :6333)
Layer 4: Caddy (sentinel only)
Layer 5: Jetski + Browser + Sidecar-Office
```

Services start Layer 0 through Layer 5 (dependency order).
Services stop Layer 5 through Layer 0 (reverse order).

This means Vault must be healthy before Matrix tries to start, and Bridge must be healthy before Sygnal or Qdrant. If you're troubleshooting a cascade failure, start at Layer 0 and work up.

## Troubleshooting

### All services unhealthy

Check the Docker daemon itself. If Docker isn't running, nothing will be healthy.

```bash
ssh root@your-vps-ip 'docker info'
```

### Bridge unhealthy, Matrix healthy

The bridge depends on both Vault and Matrix. If Matrix is up but Bridge is down, restart the bridge.

```bash
/ops action=redeploy vps_ip=... service=bridge
```

### Vault unhealthy

Bridge depends on Vault. Fix Vault first, then check if Bridge recovers on its own. If not, redeploy Bridge after Vault is healthy.

```bash
/ops action=redeploy vps_ip=... service=vault
```

### Cannot connect via SSH

Check your SSH key path and VPS IP. Try connecting manually first.

```bash
ssh -i ~/.ssh/id_ed25519 -o ConnectTimeout=10 root@your-vps-ip
```

### Services keep restarting

This usually points to a config error or missing dependency. Pull logs for the affected service.

```bash
/ops action=logs vps_ip=... service=all
```

## Advanced Usage

```bash
# Verbose health check with full diagnostics
/ops action=health vps_ip=192.168.1.100 verbose=true

# Backup to a mounted external drive
/ops action=backup vps_ip=192.168.1.100 backup_path=/mnt/backups/armorclaw.tar.gz

# Deep log dive (last 500 lines)
/ops action=logs vps_ip=192.168.1.100 service=bridge tail=500

# Full stack restart (maintenance window)
/ops action=redeploy vps_ip=192.168.1.100 service=all

# Non-root SSH with custom key
/ops action=health vps_ip=192.168.1.100 ssh_user=admin ssh_key=~/.ssh/my_key

# Check health, then redeploy if bridge is down (two invocations)
/ops action=health vps_ip=192.168.1.100
/ops action=redeploy vps_ip=192.168.1.100 service=bridge
```

## Automation Levels

| Step | Level | Description |
|------|-------|-------------|
| detect_platform | auto | Detect OS and shell environment locally |
| validate_ssh | auto | Test SSH connectivity to VPS |
| detect_topology | auto | Discover running services and mode |
| deploy | confirm | Ask before running installer on VPS |
| check_health | auto | Poll services and report status |
| redeploy | confirm | Ask before restarting services |
| view_logs | auto | Fetch docker logs immediately |
| create_backup | confirm | Ask before creating archive |
| restore_backup | confirm | Ask before overwriting data |
| print_summary | auto | Display results |

## Related Skills

- **[Deploy](../deploy/SKILL.md)** - Standalone deployment skill (more granular control)
- **[Status](../status/SKILL.md)** - Standalone health check skill
- **[Provision](../provision/SKILL.md)** - Connect mobile devices to your deployment
- **[Cloudflare](../cloudflare/SKILL.md)** - Configure HTTPS via Cloudflare

## References

- Main documentation: `doc/armorclaw.md`
- Installer script: `deploy/install.sh`
- Health check script: `deploy/health-check.sh`
- Ops skill definition: `.skills/ops.yaml`
