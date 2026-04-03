# Cross-Platform Utility Documentation

This document provides cross-platform patterns for AI CLI skills to use when deploying and managing ArmorClaw.

## Platform Detection

### Detection Logic

Skills should detect the user's platform using the following logic:

```bash
# Platform detection order
if [ -n "$MSYSTEM" ] || [ -n "$MINGW" ]; then
    # Git Bash / MSYS2 on Windows
    PLATFORM="windows-gitbash"
elif [ -n "$WSL_DISTRO_NAME" ] || [ -d /mnt/c ]; then
    # WSL (Windows Subsystem for Linux)
    PLATFORM="wsl"
elif [[ "$OSTYPE" == "darwin"* ]]; then
    # macOS
    PLATFORM="macos"
elif [[ "$OSTYPE" == "linux-gnu"* ]]; then
    # Linux
    PLATFORM="linux"
elif [ -n "$COMSPEC" ] || [ -n "$ComSpec" ]; then
    # PowerShell or cmd.exe
    PLATFORM="windows-powershell"
else
    PLATFORM="unknown"
fi
```

### Quick Platform Checks

| Platform | Quick Check | Characteristic |
|----------|-------------|----------------|
| **Linux** | `[ -f /proc/version ]` | Standard Linux |
| **macOS** | `[[ "$OSTYPE" == "darwin"* ]]` | macOS |
| **Windows (PowerShell)** | `$env:COMSPEC -or $ComSpec` | PowerShell / cmd.exe |
| **Windows (Git Bash)** | `[ -n "$MSYSTEM" ]` | Git Bash / MSYS2 |
| **WSL** | `[ -n "$WSL_DISTRO_NAME" ]` | Windows Subsystem for Linux |

## SSH Connections

### Linux / macOS

```bash
# SSH connection
ssh -i ~/.ssh/id_ed25519 user@192.168.1.100

# SSH with custom port
ssh -i ~/.ssh/id_ed25519 -p 2222 user@192.168.1.100

# SSH command execution
ssh -i ~/.ssh/id_ed25519 user@192.168.1.100 "docker ps"

# SCP file transfer
scp -i ~/.ssh/id_ed25519 local-file.txt user@192.168.1.100:/remote/path/

# SCP directory transfer
scp -i ~/.ssh/id_ed25519 -r local-dir/ user@192.168.1.100:/remote/path/
```

### Windows (Git Bash)

```bash
# Git Bash uses standard OpenSSH (same as Linux/macOS)
ssh -i ~/.ssh/id_ed25519 user@192.168.1.100

# SSH command execution
ssh -i ~/.ssh/id_ed25519 user@192.168.1.100 "docker ps"

# SCP file transfer
scp -i ~/.ssh/id_ed25519 local-file.txt user@192.168.1.100:/remote/path/
```

**Note:** Git Bash paths use forward slashes `/` and `~` expands correctly.

### WSL (Windows Subsystem for Linux)

```bash
# WSL uses standard OpenSSH
ssh -i ~/.ssh/id_ed25519 user@192.168.1.100

# Access Windows paths from WSL
ls /mnt/c/Users/YourName/.ssh/

# Copy SSH key from Windows to WSL
cp /mnt/c/Users/YourName/.ssh/id_ed25519 ~/.ssh/id_ed25519
chmod 600 ~/.ssh/id_ed25519
```

**Note:** Windows `C:\Users\Name\` maps to `/mnt/c/Users/Name/` in WSL.

### Windows (PowerShell)

```powershell
# PowerShell SSH requires OpenSSH client (installed on Windows 10+)
ssh -i "C:\Users\YourName\.ssh\id_ed25519" user@192.168.1.100

# SSH command execution
ssh -i "C:\Users\YourName\.ssh\id_ed25519" user@192.168.1.100 "docker ps"

# PSCP (PuTTY's SCP alternative - requires PuTTY installation)
pscp -i "C:\Users\YourName\.ssh\id_ed25519.ppk" local-file.txt user@192.168.1.100:/remote/path/
```

**Git Bash Alternative:** Recommend users install Git Bash and use standard OpenSSH commands instead of PowerShell.

## Path Handling

### Linux / macOS

```bash
# Home directory
~/

# SSH key location
~/.ssh/id_ed25519

# Absolute path
/home/username/.ssh/id_ed25519

# Environment variable
$HOME/.ssh/id_ed25519
```

### Windows (Git Bash)

```bash
# Git Bash uses Unix-style paths with forward slashes
~/

# SSH key location
~/.ssh/id_ed25519

# Access Windows home
~/.ssh/  # Maps to C:\Users\YourName\.ssh\

# Absolute path (still uses forward slashes)
/c/Users/YourName/.ssh/id_ed25519
```

### WSL (Windows Subsystem for Linux)

```bash
# WSL home directory
~/

# SSH key location
~/.ssh/id_ed25519

# Access Windows paths
/mnt/c/Users/YourName/.ssh/id_ed25519

# Copy from Windows to WSL
cp /mnt/c/Users/YourName/.ssh/id_ed25519 ~/.ssh/id_ed25519
```

### Windows (PowerShell)

```powershell
# Windows home directory
$env:USERPROFILE

# SSH key location
$env:USERPROFILE\.ssh\id_ed25519

# Absolute path
C:\Users\YourName\.ssh\id_ed25519

# Environment variable
$env:USERPROFILE\.ssh\
```

**Git Bash Alternative:** For path simplicity, recommend Git Bash over PowerShell for deployment tasks.

## HTTP Requests

### Linux / macOS / Git Bash / WSL (Standard)

```bash
# GET request
curl https://api.example.com/data

# GET with headers
curl -H "Authorization: Bearer token" https://api.example.com/data

# POST request with JSON
curl -X POST -H "Content-Type: application/json" \
  -d '{"key":"value"}' https://api.example.com/create

# POST with data file
curl -X POST -H "Content-Type: application/json" \
  -d @data.json https://api.example.com/create

# Download file
curl -O https://example.com/file.tar.gz

# Follow redirects
curl -L https://example.com/redirect
```

### Windows (PowerShell)

```powershell
# GET request
Invoke-WebRequest -Uri "https://api.example.com/data" -Method Get

# GET with headers
$headers = @{ "Authorization" = "Bearer token" }
Invoke-WebRequest -Uri "https://api.example.com/data" -Method Get -Headers $headers

# POST request with JSON
$body = @{ key = "value" } | ConvertTo-Json
Invoke-WebRequest -Uri "https://api.example.com/create" -Method Post -Body $body -ContentType "application/json"

# POST with data file
$body = Get-Content data.json -Raw
Invoke-WebRequest -Uri "https://api.example.com/create" -Method Post -Body $body -ContentType "application/json"

# Download file
Invoke-WebRequest -Uri "https://example.com/file.tar.gz" -OutFile "file.tar.gz"

# Follow redirects (PowerShell 6+)
Invoke-WebRequest -Uri "https://example.com/redirect" -FollowRel
```

**Git Bash Alternative:** For consistency and simplicity, recommend using `curl` from Git Bash rather than PowerShell cmdlets.

## Platform-Specific Considerations

### Line Endings

- **Linux/macOS:** LF (`\n`)
- **Windows (Git Bash):** Git auto-conversion handles line endings correctly
- **Windows (PowerShell):** CRLF (`\r\n`) - may cause issues with shell scripts

**Solution:** Always use Git Bash for shell script execution on Windows.

### Permissions

- **Linux/macOS:** Standard Unix permissions (`chmod 600` for SSH keys)
- **WSL:** Requires `chmod` for SSH keys even if copied from Windows
- **Windows (PowerShell):** Uses Windows ACLs, not Unix permissions

**Solution:** Always set file permissions with `chmod` in Linux/macOS/WSL/Git Bash environments.

### Environment Variables

| Platform | Variable Syntax | Example |
|----------|----------------|---------|
| **Linux/macOS/WSL/Git Bash** | `$VAR` or `${VAR}` | `$HOME`, `${OPENROUTER_API_KEY}` |
| **PowerShell** | `$env:VAR` | `$env:USERPROFILE`, `$env:OPENROUTER_API_KEY` |

**Solution:** Skills should export environment variables and let the shell interpret them correctly.

## Recommended Approach for Skills

### Platform Support Table

| Skill | Linux | macOS | Windows (Git Bash) | Windows (PowerShell) | WSL |
|-------|-------|-------|-------------------|----------------------|-----|
| `deploy` | ✅ Full | ✅ Full | ✅ Full | ⚠️ Git Bash recommended | ✅ Full |
| `status` | ✅ Full | ✅ Full | ✅ Full | ⚠️ Git Bash recommended | ✅ Full |
| `cloudflare` | ✅ Full | ✅ Full | ✅ Full | ⚠️ Git Bash recommended | ✅ Full |
| `provision` | ✅ Full | ✅ Full | ✅ Full | ⚠️ Git Bash recommended | ✅ Full |

### Documentation Pattern

When documenting platform-specific commands, use this pattern:

```bash
# Primary: Linux/macOS/WSL/Git Bash (standard OpenSSH)
ssh -i ~/.ssh/id_ed25519 user@192.168.1.100

# Windows (PowerShell) - Git Bash recommended
# ssh -i "C:\Users\YourName\.ssh\id_ed25519" user@192.168.1.100
```

### Best Practices

1. **Prioritize Git Bash over PowerShell** on Windows for consistency
2. **Use forward slashes `/`** in all paths (works on all platforms except PowerShell)
3. **Use `~` for home directory** (expands correctly on all Unix-like systems)
4. **Always provide Git Bash examples** before PowerShell examples
5. **Document WSL path mappings** (`/mnt/c/...`) for clarity
6. **Use `curl` for HTTP requests** (consistent across Linux/macOS/Git Bash/WSL)

### Platform Detection in Skills

Skills should include platform detection logic:

```bash
# Platform detection
if [ -n "$MSYSTEM" ] || [ -n "$MINGW" ]; then
    PLATFORM="windows-gitbash"
elif [ -n "$WSL_DISTRO_NAME" ]; then
    PLATFORM="wsl"
elif [[ "$OSTYPE" == "darwin"* ]]; then
    PLATFORM="macos"
else
    PLATFORM="linux"
fi

echo "Detected platform: $PLATFORM"
```

## References

- **ARMORCLAW.md:** Main deployment documentation
- **deploy/install.sh:** Cross-platform installer
- **Git for Windows:** https://git-scm.com/download/win (recommended for Windows users)
- **WSL Installation:** https://docs.microsoft.com/en-us/windows/wsl/install
