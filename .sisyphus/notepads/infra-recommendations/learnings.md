# Learnings from Infra Recommendations

## Creating Systemd Health Check Scripts

- Follow existing pattern in deploy/health-check.sh for colors and pass/fail functions
- Use systemctl is-active for timer checking
- Always make health check scripts executable with chmod +x
- Exit 0 for healthy, exit 1 for unhealthy states
- Support multiple timer names (certbot.timer, certbot-renew.timer)

## Docker Compose Healthchecks for Host-Networked Services

- For services using `network_mode: "host"`, healthcheck runs in container context but checks host network
- Use `nc -z -u` for UDP port checks (works with host networking since container accesses host's interfaces)
- Ensure healthcheck tools (nc) are available in the base image (coturn/coturn:latest includes nc)
- Healthcheck after `deploy` section, before next service definition
- UDP ports like 3478 (STUN/TURN) require `nc -z -u` flag, not just `-z`
- Consistent retry logic: 30s interval, 10s timeout, 3 retries

## IPv4 Preference in IP Detection

- **docker-compose.matrix.yml**: Modified coturn entrypoint to prefer IPv4 with fallback to IPv6
  - Uses `curl -s -4 https://api.ipify.org` for IPv4 preference
  - Falls back to `curl -s -6 https://api.ipify.org` with warning on failure
  - Additional check to detect and force IPv4 fallback if IPv6 is detected (starts with 0000)
- **scripts/deploy-infrastructure.sh**: Updated DNS validation to prefer IPv4
  - Same pattern: IPv4 first, then IPv6 fallback
  - Warning message uses terminal colors (YELLOW) for visibility
- **Implementation approach**:
  - Conditional operator: `curl -4 || { warning; curl -6; } || fallback`
  - Warning messages go to stderr (&2) for visibility
  - IPv6 address check: `[ "$IP%%.*" = "0000" ]` to detect non-IPv4 addresses
- **Testing**: curl -4 returns IPv4 format (N.N.N.N), curl -6 returns IPv6 format (no output in this environment)
- **Verification**: YAML syntax valid (Python yaml.safe_load), shell script syntax valid (bash -n)

## Docker Build Error Visibility Script

- **Error handling pattern**: Use `set -eo pipefail` to catch errors early in bash scripts (from deploy/installer-v4.sh line 9)
- **Build output**: Add `--progress=plain` to docker build/compose commands for full build output visibility
- **Error formatting**: Create clear error banners with color-coded output (RED for errors, YELLOW for warnings, GREEN for success)
- **Command detection**: Support multiple docker commands (auto-detect, docker-compose, docker compose, docker build)
- **Log capture**: Save build output to temporary file for detailed error analysis
- **Failure reporting**: Show last 50 lines of build log on failure with clear formatting
- **Troubleshooting tips**: Provide actionable tips when build fails (check syntax, verify dependencies, check docker info)
- **Script hygiene**: Always make bash scripts executable with `chmod +x`
- **Section organization**: Use clear section headers (e.g., "Section 1: Color Output and Logging") to improve navigation in complex scripts

## Script Templates from Existing Code

- **abort() function pattern** (deploy/installer-v4.sh lines 111-126): Creates formatted error message with error code and message, shows clear "Installation Aborted" banner, explains how to retry, and includes log file location
- **set -e pattern** (scripts/deploy-infrastructure.sh line 7): Minimal error catching, simple and effective
- **Color output**: Use 16-bit ANSI colors (e.g., RED='\033[0;31m') with proper formatting codes


## YAML/Shell Script Validation

- **Issue**: Complex shell variable expansion in command array caused YAML parsing issues
- **Problem**: `$$` and nested variable expansion in multi-line shell script broke YAML structure
- **Solution**: Use explicit if/then logic with clear conditional branching
- **YAML structure**: Command list with `['-c', '|', <multiline script>]` where script contains embedded newlines
- **Validation methods**:
  - `python3 -c "import yaml; yaml.safe_load(open('file'))"` - validates YAML syntax
  - Extract script from YAML, validate with `bash -n` - validates shell syntax
- **Better approach**: Keep logic simple and readable in shell, let YAML handle structure

## IPv4 Preference Fix (Syntax Correction)

- **File**: docker-compose.matrix.yml (lines 33-55)
- **Changed from**: Complex one-liner with nested variable expansion
  ```yaml
  EXTERNAL_IP=$${COTURN_EXTERNAL_IP:-$$(curl -s -4 ...) || ...}
  ```
- **Changed to**: Clear if/then/else structure
  ```bash
  if [ -z "${COTURN_EXTERNAL_IP}" ]; then
    echo "Detecting public IP..."
    EXTERNAL_IP=$(curl -s -4 https://api.ipify.org 2>/dev/null || echo '0.0.0.0')
    if echo "$EXTERNAL_IP" | grep -q ':'; then
      echo "Warning: Detected IPv6 address..." >&2
    fi
  else
    EXTERNAL_IP="${COTURN_EXTERNAL_IP}"
  fi
  ```
- **Benefits**:
  - No YAML parsing issues
  - More readable and maintainable
  - Easier to debug
  - Clearer logic flow
- **IPv6 fallback**: Still supported, detected via `grep -q ':'`
- **Validation**: Both YAML and shell syntax verified successfully

## Container Rollback Mechanism

- **File**: scripts/container-rollback.sh (326 lines)
- **Tag-based approach**: Uses :current and :prev Docker tags
- **Supported services**: bridge, matrix, coturn, sygnal, caddy
- **Commands**:
  - tag-current-as-prev: Tag current images as :prev before deploy
  - rollback: Restore previous version
  - status: Show current tag status
  - help: Comprehensive usage documentation
- **--dry-run mode**: Preview changes without executing
- **Logging**: All actions logged to /var/log/armorclaw/rollback.log
- **Error handling**: Checks docker daemon running, handles missing images gracefully
- **No config/volume rollback**: Only container images are rolled back
- **Independent per container**: No coordination across services
- **Manual trigger only**: No automatic rollback on failure
