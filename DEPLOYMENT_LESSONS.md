# ArmorClaw Deployment: Hard-Won Lessons

> **Automation Status**: All 4 lessons are now automated. See details below.

### 1. The Socket Trap ✅ AUTOMATED
**Challenge**: The bridge defaults to a Unix Domain Socket (`.sock` file) inside the container.
**Solution**: You MUST explicitly pass `--socket "tcp://0.0.0.0:8080"`. If the logs say `unix socket path=tcp://...`, the binary is misinterpreting the flag as a filename. Use the `ARMORCLAW_RPC_TYPE=tcp` environment variable to force network binding.
**Automation**: `docker-compose.yml` sets `ARMORCLAW_RPC_TYPE=tcp` and `ARMORCLAW_RPC_ADDR=0.0.0.0:8080`.

### 2. The Identity Firewall ✅ AUTOMATED
**Challenge**: `ARMORCLAW_CONTAINER_MODE=true` triggers aggressive IP filtering.
**Solution**: When accessing the bridge through an SSH tunnel (Localhost), set this to `false`. Otherwise, the bridge sees `127.0.0.1` and drops the connection with `Empty reply from server`.
**Automation**: `docker-compose.yml` sets `ARMORCLAW_CONTAINER_MODE=false`.

### 3. SSH Tunneling Logic ✅ AUTOMATED
**Challenge**: WSL cannot see tunnels opened in Windows PowerShell.
**Solution**: Open the tunnel *inside* the WSL terminal using `$CONNECT_VPS -L 9000:localhost:8081`. This ensures the `check_bridge.sh` script can find the RPC gateway.
**Automation**: Run `deploy/setup-ssh-tunnel.sh` to automatically establish the tunnel. Set `CONNECT_VPS` or `VPS_IP` environment variable.

### 4. Hardware Unsealing ✅ AUTOMATED
**Challenge**: Keystore initialization fails if the secret isn't exactly 32 bytes or base64 encoded.
**Solution**: Use a hardware-derived master key via `$ARMORCLAW_KEYSTORE_SECRET` to ensure the vault unseals on startup.
**Automation**: `Dockerfile.quickstart` generates keystore key via `generate-keystore-key.sh`. `docker-compose-full.yml` provides template.