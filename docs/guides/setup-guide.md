# ArmorClaw Setup Guide

> **Last Updated:** 2026-02-06
> **Time to Complete:** 5-15 minutes

---

## Quick Start Options

ArmorClaw provides multiple ways to get started, depending on your experience level:

| Method | Time Required | Experience Level | Recommended For |
|--------|---------------|------------------|-----------------|
| **Shell Script Wizard** ⭐ | 10-15 min | Beginner | First-time users, complete automation |
| **CLI Wizard** | 5-10 min | Intermediate | Users who prefer to build first |
| **Manual Installation** | 15-20 min | Advanced | Full control over each step |
| **Docker Compose Stack** | 5 min | Expert | Quick testing with Matrix |

---

## Method 1: Interactive Setup Wizard ⭐ (Recommended)

The interactive setup wizard (`deploy/setup-wizard.sh`) guides you through the entire installation and configuration process with helpful prompts, validation, and security best practices.

### Prerequisites

- Ubuntu 22.04/24.04 or Debian 12
- Root/sudo access
- 2 GB RAM minimum

### Step-by-Step

#### 2. Run the Setup Wizard

```bash
cd bridge
./deploy/setup-wizard.sh
```

#### 2. Follow the Interactive Prompts

The wizard will guide you through:

1. **System Requirements Check**
    - OS detection (Ubuntu/Debian)
    - Memory and disk space verification
    - Permission check

2. **Docker Verification**
    - Check if Docker is installed and running
    - Manual installation required if Docker not found

3. **Container Image Setup**
   - Build ArmorClaw agent container
   - Or use existing image

4. **Bridge Installation**
   - Build bridge from source (Go 1.21+)
   - Install to /opt/armorclaw/
   - Create system user

5. **Configuration File Creation**
   - Socket path configuration
   - Logging level
   - Optional Matrix integration
   - Save to /etc/armorclaw/config.toml

6. **Keystore Initialization**
   - Initialize encrypted keystore
   - Hardware-derived master key
   - Zero-touch reboot setup
   - Salt file creation

7. **First API Key Setup (Optional)**
   - Choose provider (openai, anthropic, etc.)
   - Enter API key
   - Store in encrypted keystore

8. **Systemd Service Setup**
   - Create service file
   - Configure auto-start
   - Set resource limits

9. **Verification**
   - Check all directories
   - Verify binary installation
   - Test configuration

### Features

- ✅ **Colored output** for better readability
- ✅ **Input validation** with helpful error messages
- ✅ **Cancel-safe** - can press Ctrl+C anytime
- ✅ **Logging** - saves setup log to `/var/log/armorclaw-setup.log`
- ✅ **Security defaults** - pre-configured secure options
- ✅ **Beginner-friendly** - explains each step

### After Setup

```bash
# Start the bridge
sudo systemctl start armorclaw-bridge

# Check status
sudo systemctl status armorclaw-bridge

# View logs
sudo journalctl -u armorclaw-bridge -f

# Test health
echo '{"jsonrpc":"2.0","method":"health","id":1}' | sudo socat - UNIX-CONNECT:/run/armorclaw/bridge.sock
```

---

## Method 2: CLI Interactive Wizard

For users who prefer to build the bridge first, then use an interactive wizard:

### Step 1: Build the Bridge

```bash
cd bridge
go build -o build/armorclaw-bridge ./cmd/bridge
```

### Step 2: Run the CLI Wizard

```bash
./build/armorclaw-bridge setup
```

The wizard will guide you through:

1. **Docker Availability Check**
   ```
   🔍 Checking Docker availability... ✓
   ```

2. **Choose Configuration Location**
   ```
   📁 Configuration Setup
   Where would you like to store your ArmorClaw configuration?
     [1] Default (~/.armorclaw)
     [2] Custom location

   Choose an option (1-2) [1]:
   ```

3. **Select AI Provider**
   ```
   🤖 AI Provider Selection
   Which AI provider do you use?
     [1] OpenAI (GPT-4, GPT-3.5)
     [2] Anthropic (Claude)
     [3] OpenRouter
     [4] Google (Gemini)
     [5] xAI (Grok)
     [6] Skip (add keys later)

   Choose an option (1-6) [1]:
   ```

4. **Enter API Key**
   ```
   🔑 OpenAI API Key
   Enter your API key (input will be hidden):

   API Key: •••••••••••••••••••••
   ✓ API key stored as 'openai-default'
   ```

5. **Configure Matrix (Optional)**
   ```
   💬 Matrix Configuration (Optional)
   Enable Matrix? [y/N]:
   ```

6. **Complete Setup**
   ```
   ╔════════════════════════════════════════════════════════════╗
   ║                   Setup Complete! ✓                       ║
   ╚════════════════════════════════════════════════════════════╝

   🚀 Quick Start:
   ╚════════════════════════════════════════════════════════════╝

   🚀 Quick Start:
     1. Start the bridge:  armorclaw-bridge
     2. Start an agent:    armorclaw-bridge start --key openai-default
   ```

#### 4. Start the Bridge

```bash
./build/armorclaw-bridge
```

#### 5. Start Your Agent

```bash
./build/armorclaw-bridge start --key openai-default
```

---

## Method 4: Docker Compose Stack (Quick Test)

For quick testing with Matrix integration without manual setup:

```bash
cd bridge
./deploy/launch-element-x.sh
```

This deploys:
- Matrix Conduit server
- Caddy reverse proxy (with SSL)
- ArmorClaw Bridge
- Auto-provisioning

---

## Method 5: VPS Deployment via Tarball 🆓

Deploy ArmorClaw to a remote VPS (like Hostinger) by creating a tarball locally and transferring it.

### Overview

This method is perfect for:
- Deploying to a remote VPS
- Air-gapped environments
- Production deployments
- Multiple server deployments

**Time:** 10-15 minutes | **Difficulty:** Easy (with automated script)

---

### Quick Start: Automated Deployment ⭐

**Use the automated deployment script for the easiest VPS deployment:**

```bash
# From local machine
scp deploy/vps-deploy.sh armorclaw-deploy.tar.gz user@your-vps-ip:/tmp/

# SSH into VPS
ssh user@your-vps-ip

# Run deployment script
chmod +x /tmp/vps-deploy.sh
sudo bash /tmp/vps-deploy.sh
```

**The vps-deploy.sh script handles:**
- ✅ Pre-flight checks (disk, memory, ports)
- ✅ Tarball verification
- ✅ Docker installation (if needed)
- ✅ File extraction and line ending fixes
- ✅ Interactive configuration
- ✅ Automated deployment

**See:** [deploy/vps-deploy.sh](../deploy/vps-deploy.sh) for the script source

---

### Manual Deployment Steps

#### Step 1: Create Tarball on Local Machine

Navigate to your ArmorClaw directory and create the deployment tarball:

```bash
cd armorclaw

# Create deployment tarball (excludes build artifacts and sensitive data)
tar -czf armorclaw-deploy.tar.gz \
    --exclude='.git' \
    --exclude='node_modules' \
    --exclude='*.log' \
    --exclude='.env' \
    --exclude='bridge/build' \
    --exclude='container/venv' \
    .

# Verify tarball was created
ls -lh armorclaw-deploy.tar.gz
```

**Expected output:**
```
-rw-r--r-- 1 user user 2.5M Feb  7 12:00 armorclaw-deploy.tar.gz
```

**What's included:**
- ✅ Complete source code
- ✅ Docker configurations
- ✅ Deployment scripts
- ✅ Documentation
- ✅ Configuration templates

**What's excluded:**
- ❌ Build artifacts (rebuild on server)
- ❌ Git history (smaller size)
- ❌ Sensitive data (.env files)
- ❌ Large dependencies

---

### Step 2: Transfer Tarball to VPS

Choose a transfer method based on your operating system:

#### Option A: SCP (Linux/macOS/WSL) ⭐

```bash
# Transfer to VPS
scp armorclaw-deploy.tar.gz user@your-vps-ip:/tmp/

# Example with custom SSH port:
scp -P 2222 armorclaw-deploy.tar.gz user@your-vps-ip:/tmp/

# With progress bar:
scp -v armorclaw-deploy.tar.gz user@your-vps-ip:/tmp/
```

#### Option B: SCP (Windows PowerShell)

```powershell
# Transfer to VPS
scp armorclaw-deploy.tar.gz user@your-vps-ip:/tmp/
```

#### Option C: WinSCP (Windows GUI)

1. Download [WinSCP](https://winscp.net/)
2. Connect to your VPS
3. Navigate to local `armorclaw-deploy.tar.gz`
4. Drag to remote `/tmp/` folder

#### Option D: rsync (Large Files)

```bash
# Better for large files with resume support
rsync -avzP --partial \
    armorclaw-deploy.tar.gz \
    user@your-vps-ip:/tmp/
```

---

### Step 3: SSH into VPS

```bash
# SSH into your VPS
ssh user@your-vps-ip

# Or with custom port:
ssh -p 2222 user@your-vps-ip
```

---

### Step 4: Verify Tarball on VPS

After SSH'ing into the VPS, verify the tarball transferred correctly:

```bash
# Check file exists and size
ls -lh /tmp/armorclaw-deploy.tar.gz

# Should show something like:
# -rw-r--r-- 1 user user 2.5M Feb  7 12:05 armorclaw-deploy.tar.gz

# Verify tar integrity (list contents without extracting)
tar -tvf /tmp/armorclaw-deploy.tar.gz | head -20

# Should show file listing:
# -rw-r--r-- user/user 1234 2026-02-07 12:00 bridge/
# -rw-r--r-- user/user 5678 2026-02-07 12:00 bridge/cmd/bridge/main.go
# ...
```

**If tar shows errors:**
```bash
# File corrupted - re-transfer
exit  # Exit VPS
# On local machine, re-run scp command from Step 2
```

---

### Step 5: Extract Tarball

Create deployment directory and extract:

```bash
# Create deployment directory
sudo mkdir -p /opt/armorclaw

# Navigate to deployment directory
cd /opt/armorclaw

# Extract tarball
sudo tar -xzf /tmp/armorclaw-deploy.tar.gz

# Verify extraction
ls -la
```

**Expected output:**
```
total 52
drwxr-xr-x 6 root root 4096 Feb  7 12:00 .
drwxr-xr-x 3 root root 4096 Feb  7 12:00 ..
drwxr-xr-x 2 root root 4096 Feb  7 12:00 bridge
drwxr-xr-x 2 root root 4096 Feb  7 12:00 configs
drwxr-xr-x 2 root root 4096 Feb  7 12:00 deploy
drwxr-xr-x 2 root root 4096 Feb  7 12:00 docs
-rw-r--r-- 1 root root 1045 Feb  7 12:00 README.md
```

---

### Step 6: What to Do After Extracting

Now you have ArmorClaw on your VPS. Choose your next step:

#### Option A: Run Setup Wizard (Recommended) ⭐

```bash
cd /opt/armorclaw

# Run the interactive setup wizard
sudo ./deploy/setup-wizard.sh
```

The wizard will guide you through:
- Docker installation (if needed)
- Bridge compilation
- Keystore initialization
- API key setup
- Systemd service configuration

#### Option B: Deploy Docker Compose Stack (Quick Start)

```bash
cd /opt/armorclaw

# Create .env file
cat > .env <<EOF
MATRIX_DOMAIN=srv1313371.hstgr.cloud
MATRIX_ADMIN_USER=admin
MATRIX_ADMIN_PASSWORD=$(openssl rand -base64 16 | tr -d '/+=')
ROOM_NAME=ArmorClaw Agents
ROOM_ALIAS=agents
EOF

# Note the admin password
echo "Admin Password: $(grep MATRIX_ADMIN_PASSWORD .env | cut -d= -f2)"

# Deploy the stack
sudo docker compose -f docker-compose-stack.yml up -d

# Check status
sudo docker compose -f docker-compose-stack.yml ps
```

#### Option C: Manual Bridge Installation

```bash
cd /opt/armorclaw/bridge

# Install Go (if not installed)
wget https://go.dev/dl/go1.21.0.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.0.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin

# Build bridge
CGO_ENABLED=1 go build -o build/armorclaw-bridge ./cmd/bridge

# Run setup
sudo ./build/armorclaw-bridge setup
```

---

### Step 7: Verify Deployment

After deployment, verify everything is working:

```bash
# Check containers are running
sudo docker ps

# Check bridge logs
sudo docker logs armorclaw-bridge

# Check bridge health
echo '{"jsonrpc":"2.0","method":"health","id":1}' | sudo nc -U /run/armorclaw/bridge.sock

# Check Matrix is accessible
curl -I http://localhost:8448
```

---

### Step 8: Connect via Element X

1. **Download Element X app** on your device

2. **Login with:**
   - **Homeserver:** `http://your-vps-ip:8448` or `https://your-domain.com`
   - **Username:** `admin`
   - **Password:** (from .env file or setup wizard)

3. **Join or create room:** `#agents:your-domain.com`

---

### Troubleshooting

#### Issue: "Permission denied extracting tarball"

**Solution:**
```bash
# Use sudo for extraction
sudo tar -xzf /tmp/armorclaw-deploy.tar.gz
```

#### Issue: "Tarball corrupted"

**Solution:**
```bash
# On local machine, verify tar integrity
tar -tvf armorclaw-deploy.tar.gz

# Re-transfer if corrupted
scp armorclaw-deploy.tar.gz user@your-vps-ip:/tmp/
```

#### Issue: "Docker not installed"

**Solution:**
```bash
# Quick install Docker
curl -fsSL https://get.docker.com | sudo sh

# Enable and start Docker
sudo systemctl enable docker
sudo systemctl start docker

# Verify installation
docker --version
```

#### Issue: "Go not installed"

**Solution:**
```bash
# Install Go 1.21
wget https://go.dev/dl/go1.21.0.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.0.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc

# Verify installation
go version
```

---

### Quick Reference Commands

**All-in-one (Local to VPS):**
```bash
# On local machine
cd armorclaw
tar -czf armorclaw-deploy.tar.gz --exclude='.git' --exclude='bridge/build' .
scp armorclaw-deploy.tar.gz user@your-vps-ip:/tmp/
ssh user@your-vps-ip
```

**All-in-one (On VPS):**
```bash
# On VPS after SSH
sudo mkdir -p /opt/armorclaw && \
cd /opt/armorclaw && \
sudo tar -xzf /tmp/armorclaw-deploy.tar.gz && \
sudo ./deploy/setup-wizard.sh
```

---

### Next Steps

- **[Hostinger Docker Deployment Guide](hostinger-docker-deployment.md)** - Complete Docker deployment for Hostinger VPS
- **[Hostinger VPS Deployment](hostinger-deployment.md)** - Detailed tarball deployment guide
- **[Element X Quick Start](element-x-quickstart.md)** - Connect to agents via Element X
- **[Troubleshooting Guide](troubleshooting.md)** - Common issues and solutions

---

## Additional Resources

- **Full Manual Installation:** See [deployment-quickref.md](2026-02-05-deployment-quickref.md) for detailed manual steps
- **Docker Compose Stack:** Use for quick testing with Matrix integration
- **Troubleshooting:** See [troubleshooting.md](troubleshooting.md) for common issues

---

## Post-Setup Verification

### Check Bridge Status

```bash
./build/armorclaw-bridge status
```

### List Stored Keys

```bash
./build/armorclaw-bridge list-keys
```

Expected output:

```
✓ Found 1 API key(s):

  • openai-default
    Provider: openai
    Name: OpenAI API Key
```

### Validate Configuration

```bash
./build/armorclaw-bridge validate
```

---

## Troubleshooting

> **🔍 Quick Error Lookup:** Search for any error message in our [Error Catalog](error-catalog.md)

### Issue: "Docker is not available"

**Solution:** Start Docker Desktop or the Docker daemon:

```bash
# macOS/Windows
open -a Docker

# Linux
sudo systemctl start docker
```

### Issue: "Failed to initialize keystore"

**Solution:** Ensure the keystore directory exists:

```bash
mkdir -p ~/.armorclaw
chmod 700 ~/.armorclaw
```

### Issue: "Key not found"

**Solution:** List your available keys:

```bash
./build/armorclaw-bridge list-keys
```

If no keys are found, add one:

```bash
./build/armorclaw-bridge add-key --provider openai --token sk-xxx
```

### Issue: Permission denied on socket

**Solution:** The bridge requires proper permissions on the socket directory:

```bash
sudo mkdir -p /run/armorclaw
sudo chown $USER /run/armorclaw
```

### Issue: Windows path parsing error

**Solution:** ArmorClaw automatically handles Windows paths. If you see this error:

```
expected eight hexadecimal digits after '\U', but got "C:\\Us" instead
```

It's been fixed in the latest version. Update to the latest binary.

---

## Configuration File Reference

After setup, your configuration will be at `~/.armorclaw/config.toml`:

```toml
[server]
  socket_path = "/run/armorclaw/bridge.sock"  # Unix socket for RPC
  pid_file = "/run/armorclaw/bridge.pid"       # PID file for daemon mode
  daemonize = false                              # Run as daemon (background)

[keystore]
  db_path = "/home/user/.armorclaw/keystore.db" # Encrypted keystore
  master_key = ""                                # Auto-generated (don't set)

[matrix]
  enabled = false                                 # Enable Matrix integration
  homeserver_url = "https://matrix.example.com"   # Matrix homeserver
  username = "bridge-bot"                         # Matrix username
  password = "change-me"                          # Matrix password

[logging]
  level = "info"                                  # Log level: debug, info, warn, error
  format = "text"                                 # Log format: text, json
```

---

## Advanced Configuration

### Environment Variable Overrides

You can override any configuration setting with environment variables:

```bash
export ARMORCLAW_SOCKET_PATH="/tmp/custom.sock"
export ARMORCLAW_LOG_LEVEL="debug"
export ARMORCLAW_MATRIX_ENABLED="true"
./build/armorclaw-bridge
```

### CLI Flag Overrides

```bash
./build/armorclaw-bridge \
  --socket /tmp/custom.sock \
  --log-level debug \
  --matrix-enabled \
  --matrix-homeserver https://matrix.example.com
```

**Precedence:** CLI Flags > Environment Variables > Config File > Defaults

### Shell Completion

ArmorClaw provides shell completion for bash and zsh to make daily usage more efficient:

#### Bash Completion

```bash
# Generate and load completion
./build/armorclaw-bridge completion bash > ~/.bash_completion.d/armorclaw-bridge
source ~/.bash_completion.d/armorclaw-bridge

# Or add to ~/.bashrc for persistent completion
echo 'source ~/.bash_completion.d/armorclaw-bridge' >> ~/.bashrc
```

#### Zsh Completion

```bash
# Generate completion
./build/armorclaw-bridge completion zsh > ~/.zsh/completions/_armorclaw-bridge

# Ensure completion is enabled in ~/.zshrc
autoload -U compinit && compinit
```

#### Completion Features

- **Command completion:** `armorclaw-bridge <TAB>` shows all commands
- **Flag completion:** `armorclaw-bridge add-key --<TAB>` shows available flags
- **Value completion:** `armorclaw-bridge add-key --provider <TAB>` shows providers (openai, anthropic, etc.)
- **Key ID completion:** `armorclaw-bridge start --key <TAB>` shows your stored keys

### Daemon Mode

Run the bridge as a background daemon for long-running operations:

#### Daemon Commands

```bash
# Start daemon
./build/armorclaw-bridge daemon start

# Check status
./build/armorclaw-bridge daemon status

# View logs
./build/armorclaw-bridge daemon logs

# Restart daemon
./build/armorclaw-bridge daemon restart

# Stop daemon
./build/armorclaw-bridge daemon stop
```

#### Daemon Features

- **Background execution:** Bridge runs in background, doesn't block terminal
- **PID file:** Tracks process ID at `/run/armorclaw/bridge.pid`
- **Graceful shutdown:** Responds to SIGTERM for clean shutdown
- **Log file:** Optional logging to file (configured in `config.toml`)

#### Daemon Configuration

Add to your `config.toml`:

```toml
[server]
  daemonize = true
  pid_file = "/run/armorclaw/bridge.pid"
  log_file = "/var/log/armorclaw/bridge.log"
```

---

## Security Best Practices

### 1. Protect Your Keystore

The keystore is encrypted with a hardware-derived master key, but you should still protect the file:

```bash
chmod 600 ~/.armorclaw/keystore.db
chmod 700 ~/.armorclaw
```

### 2. Use Strong API Keys

- Rotate API keys regularly
- Use keys with usage restrictions (IP, rate limits)
- Never commit API keys to version control

### 3. Run with Least Privilege

The bridge runs as your user, not as root. The container runs as UID 10001 (claw).

### 4. Audit Your Setup

```bash
# Check running containers
docker ps

# Inspect bridge logs
./build/armorclaw-bridge status

# Verify no secrets in environment
docker inspect <container-id> | grep -i env
```

### 5. Docker Build Process

If you encounter issues during container build, particularly related to circular dependencies in the security hardening phase:
- The build process removes dangerous tools using `/bin/rm` commands
- If you see an error with exit code 125, verify that the Dockerfile doesn't have self-deletion issues
- Previously, line 88 of the Dockerfile was removing `/bin/rm` itself which caused build failures
- This has been fixed by removing `/bin/rm` from that deletion command 

---

## Next Steps

- **🔍 Having errors?** Search the [Error Catalog](error-catalog.md) by error text
- Read the [Architecture Guide](../plans/2026-02-05-armorclaw-v1-design.md)
- Explore the [RPC API Reference](../reference/rpc-api.md)
- Check out the [Troubleshooting Guide](troubleshooting.md)
- Review [Security Best Practices](../plans/2026-02-05-security-hardening.md)

---

## Getting Help

- **Documentation:** [docs/](../index.md)
- **Issues:** https://github.com/armorclaw/armorclaw/issues
- **Discussions:** https://github.com/armorclaw/armorclaw/discussions

---

**Setup Guide Last Updated:** 2026-02-07
