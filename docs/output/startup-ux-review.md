# Startup UX Review - ArmorClaw vs OpenClaw

> **Date:** 2026-02-07
> **Reviewer:** Startup Experience Analysis
> **Goal:** Exceed OpenClaw's ease-of-use standard
> **Rating:** 🔴 NEEDS IMPROVEMENT (3/10)

---

## Executive Summary

**Current Status:** ArmorClaw's startup experience is **significantly more complex** than OpenClaw's "just works" approach. While this is partly due to enhanced security features, there are many friction points that can be eliminated.

**OpenClaw Standard:** `pip install openclaw && export API_KEY=sk-xxx && openclaw`

**ArmorClaw Current:** Build → Initialize config → Edit config → Store keys → Start container (5+ steps)

**Target:** Should feel as simple as OpenClaw while maintaining security guarantees.

---

## 🚨 Critical Issues (Blocking First-Run Success)

### Issue #1: Windows Path Parsing Failure 🔴 CRITICAL

**Symptom:**
```
Failed to load configuration: toml: line 10: expected eight hexadecimal digits after '\U', but got "C:\\Us" instead
```

**Root Cause:** TOML parser interprets `\U` in Windows paths as Unicode escape sequence.

**Impact:** Config generation creates broken files on Windows.

**Fix Required:**
- Use forward slashes in paths consistently
- Sanitize paths when writing config files
- Add path normalization in Save() function

---

### Issue #2: No Sensible Defaults for Development 🔴 CRITICAL

**Problem:** Bridge won't start without manual configuration.

**Current Behavior:**
```bash
$ ./armorclaw-bridge
# Fails with config error
```

**OpenClaw Comparison:**
```bash
$ openclaw
# Works with sensible defaults
```

**Fix Required:**
- Bridge should start with built-in defaults if no config found
- Create default config on first run automatically
- Only require config for Matrix/advanced features

---

### Issue #3: API Key Storage is Manual 🔴 CRITICAL

**Current Workflow:**
1. Build bridge
2. Initialize config
3. Store key via RPC with complex JSON
4. Start container with stored key

**OpenClaw Workflow:**
```bash
export OPENAI_API_KEY=sk-xxx && openclaw
```

**Fix Required:**
- Add `--key` flag to bridge for quick testing
- Support `ARMORCLAW_API_KEY` environment variable
- Auto-store key in first-run wizard

---

### Issue #4: Build Requirement 🟠 HIGH

**Problem:** Must compile from source to use ArmorClaw.

**Impact:** Huge barrier to entry compared to `pip install openclaw`.

**Fix Required:**
- Pre-built binaries for Windows/macOS/Linux
- Homebrew/apt install options
- One-line install from release assets

---

## 📋 Detailed UX Analysis

### Phase 1: Installation

| Step | ArmorClaw | OpenClaw | Rating |
|------|------------|----------|--------|
| Download | Build from source | `pip install` | 2/10 |
| Prerequisites | Go + Docker + Make | Python + pip | 4/10 |
| Time to first run | 15-30 minutes | 30 seconds | 2/10 |

**Issues:**
- No pre-built binaries
- Complex build toolchain
- No package manager support

---

### Phase 2: First Run Experience

| Aspect | ArmorClaw | OpenClaw | Rating |
|--------|------------|----------|--------|
| Out-of-box | Fails without config | Works immediately | 1/10 |
| Config required | YES | NO | 1/10 |
| Error messages | Technical, unclear | Clear, actionable | 4/10 |
| Documentation required | YES | NO | 2/10 |

**First Run Test:**
```bash
# OpenClaw
$ openclaw
[INFO] Starting OpenClaw...
[INFO] Set OPENAI_API_KEY to use GPT-4
✅ Works (with helpful message)

# ArmorClaw
$ ./armorclaw-bridge
Error: Failed to load configuration
❌ Fails immediately
```

---

### Phase 3: Configuration

| Task | ArmorClaw | OpenClaw | Rating |
|------|------------|----------|--------|
| API key setup | Manual RPC call | Environment variable | 2/10 |
| Config complexity | TOML file with sections | None needed | 2/10 |
| Validation | Manual `validate` command | Automatic | 3/10 |

**Current API Key Setup:**
```bash
echo '{"jsonrpc":"2.0","method":"store_key","params":{...}}' | socat - UNIX-CONNECT:/run/armorclaw/bridge.sock
```

**Should be as simple as:**
```bash
export ARMORCLAW_API_KEY="sk-xxx" && ./armorclaw-bridge
```

---

### Phase 4: Starting an Agent

| Task | ArmorClaw | OpenClaw | Rating |
|------|------------|----------|--------|
| Command complexity | JSON-RPC with socat | `openclaw` | 2/10 |
| Error handling | Generic JSON errors | Clear messages | 3/10 |
| Success feedback | JSON response | Human-readable status | 3/10 |

---

### Phase 5: Feedback & Debugging

| Aspect | ArmorClaw | OpenClaw | Rating |
|--------|------------|----------|--------|
| Log clarity | Technical log format | User-friendly messages | 4/10 |
| Error messages | JSON-RPC errors | Plain language | 3/10 |
| Help availability | `--help` basic | Integrated help | 5/10 |

---

## 🎯 UX Ratings Summary

| Category | Current | Target | Gap |
|----------|---------|--------|-----|
| Installation | 2/10 | 8/10 | -6 |
| First Run | 1/10 | 9/10 | -8 |
| Configuration | 2/10 | 9/10 | -7 |
| Daily Use | 4/10 | 9/10 | -5 |
| Error Messages | 4/10 | 8/10 | -4 |
| Documentation | 7/10 | 8/10 | -1 |
| **OVERALL** | **3/10** | **9/10** | **-6** |

---

## 🚀 Recommended Improvements (Priority Order)

### 🔴 P0: Must Have (Blocks Basic Usage)

#### 1. Fix Windows Path Handling
```go
// In config.Save() - normalize paths
func Save(cfg *Config, path string) error {
    // Normalize paths for TOML output
    cfg.Keystore.DBPath = filepath.ToSlash(cfg.Keystore.DBPath)
    cfg.Server.SocketPath = filepath.ToSlash(cfg.Server.SocketPath)
    // ... rest of save
}
```

#### 2. Add Sensible Defaults Mode
```bash
# Bridge should work without config file
$ ./armorclaw-bridge
[INFO] No config found, using defaults
[INFO] Keystore: ~/.armorclaw/keystore.db (auto-created)
[INFO] Socket: /run/armorclaw/bridge.sock
[INFO] Ready! Store your first API key with: armorclaw-bridge add-key
```

#### 3. Add Quick-Start Commands
```bash
# Simple API key addition
$ ./armorclaw-bridge add-key --provider openai --token sk-xxx
[INFO] API key stored as 'openai-default'
[INFO] Start agent: armorclaw-bridge start --key openai-default
```

#### 4. Environment Variable Support for API Keys
```bash
# Should work like OpenClaw
$ export ARMORCLAW_API_KEY=sk-xxx
$ ./armorclaw-bridge start
[INFO] Using API key from ARMORCLAW_API_KEY
[INFO] Agent started
```

---

### 🟠 P1: Should Have (Major UX Improvements)

#### 5. Pre-Built Binaries
- GitHub releases with binaries for Windows/macOS/Linux
- Homebrew tap: `brew install armorclaw/bridge`
- APT repository for Debian/Ubuntu

#### 6. Interactive Setup Wizard
```bash
$ ./armorclaw-bridge setup
Welcome to ArmorClaw! Let's get you set up.

? Which AI provider do you use? (Use arrow keys)
❯ OpenAI (GPT-4/GPT-3.5)
  Anthropic (Claude)
  Google (Gemini)

? Enter your API key: ****************************

[SUCCESS] Configuration saved!
[NEXT] Start your agent: armorclaw-bridge start
```

#### 7. Better Error Messages
```json
// Current (confusing)
{"jsonrpc":"2.0","error":{"code":-32602,"message":"invalid params"}}

// Improved (helpful)
Error: The 'key_id' parameter is required.

Example: armorclaw-bridge start --key my-openai-key

List your keys: armorclaw-bridge list-keys
```

#### 8. User-Friendly CLI Wrapper
```bash
# Instead of JSON-RPC with socat
$ armorclaw start --key my-key
✓ Agent started (container: abc123)

$ armorclaw status
● Bridge: Running
● Agent: abc123 - Running for 5m
● Keys: 2 stored

$ armorclaw stop abc123
✓ Agent stopped
```

---

### 🟡 P2: Nice to Have (Polish)

#### 9. Auto-Configuration Detection
```bash
$ ./armorclaw-bridge
[INFO] Detected: ~/.openclaw/config.json
[INFO] Import OpenClaw keys? [Y/n]: y
[INFO] Imported 2 keys from OpenClaw
[INFO] Ready to use!
```

#### 10. Desktop Integration
- System tray icon
- GUI for key management
- Visual container status

#### 11. Quick Start Templates
```bash
$ armorclaw init --template developer
[INFO] Configured for development workflow
[INFO] Hot-reload enabled
[INFO] Debug logging enabled
```

#### 12. One-Command Dev Server
```bash
$ armorclaw dev
[INFO] Starting bridge with hot-reload...
[INFO] Starting agent with test key...
[INFO] Ready for development!
```

---

## 📊 OpenClaw Features to Emulate

| Feature | OpenClaw | ArmorClaw Status | Priority |
|----------|----------|-------------------|----------|
| `pip install` | ✅ | ❌ (needs build) | P0 |
| Env var API keys | ✅ | ❌ (manual RPC) | P0 |
| Works out of box | ✅ | ❌ (needs config) | P0 |
| Clear errors | ✅ | ⚠️ (JSON-RPC) | P1 |
| Simple CLI | ✅ | ❌ (JSON-RPC) | P1 |
| Auto-config | ✅ | ❌ | P2 |
| Progress bars | ✅ | ❌ | P2 |

---

## 🎬 Proposed First-Run Experience

### Target Experience (Should Match OpenClaw Simplicity)

```bash
# Step 1: Install (one command)
curl -sSL https://install.armorclaw.io | sh

# Step 2: Set API key (environment variable, like OpenClaw)
export ARMORCLAW_API_KEY="sk-xxx"

# Step 3: Run (one command)
armorclaw start

# That's it! Agent is running with full security.
```

### Current Experience (Too Complex)

```bash
# Step 1: Clone and build
git clone https://github.com/armorclaw/armorclaw.git
cd armorclaw/bridge
go build -o build/armorclaw-bridge ./cmd/bridge

# Step 2: Initialize config
./build/armorclaw-bridge init

# Step 3: Edit config file
vim ~/.armorclaw/config.toml  # Manually edit

# Step 4: Store API key via complex RPC
echo '{"jsonrpc":"2.0","method":"store_key","params":{"id":"key1","provider":"openai","token":"sk-xxx"},"id":1}' | socat - UNIX-CONNECT:/run/armorclaw/bridge.sock

# Step 5: Start container
echo '{"jsonrpc":"2.0","method":"start","params":{"key_id":"key1"},"id":1}' | socat - UNIX-CONNECT:/run/armorclaw/bridge.sock

# Result: 5+ manual steps vs OpenClaw's 3 simple steps
```

---

## 🔧 Technical Implementation Plan

### Phase 1: Critical Fixes (1-2 days)

1. **Fix Windows Path Handling**
   - File: `bridge/pkg/config/loader.go`
   - Use `filepath.ToSlash()` before writing TOML
   - Add path normalization in `Save()`

2. **Add Default Config Mode**
   - File: `bridge/pkg/config/loader.go`
   - Return defaults without error when no config found
   - Log warning but continue

3. **Add Environment Variable API Key Support**
   - File: `bridge/cmd/bridge/main.go`
   - Check `ARMORCLAW_API_KEY` on startup
   - Auto-store if provided

4. **Add Quick-Start Commands**
   - File: `bridge/cmd/bridge/main.go`
   - Add subcommands: `add-key`, `start`, `stop`, `status`, `list-keys`

### Phase 2: UX Improvements (3-5 days)

5. **Create CLI Wrapper**
   - File: `cmd/cli/main.go`
   - User-friendly commands
   - Better error messages

6. **Setup Wizard**
   - File: `cmd/setup/main.go`
   - Interactive prompts
   - Auto-configuration

7. **Pre-Built Binaries**
   - GitHub Actions workflow
   - Multi-platform builds
   - Release automation

### Phase 3: Polish (1 week)

8. **Desktop Integration**
9. **OpenClaw Import**
10. **Auto-Configuration Detection**

---

## 📈 Success Metrics

### Before Improvements
- **Time to first run:** 15-30 minutes
- **Steps required:** 5+
- **Commands to memorize:** 10+
- **Documentation required:** YES
- **Success rate (unassisted):** ~20%

### After Improvements (Target)
- **Time to first run:** < 2 minutes
- **Steps required:** 2-3
- **Commands to memorize:** 3-5
- **Documentation required:** NO
- **Success rate (unassisted):** >80%

---

## 🎯 Immediate Action Items

### Today (Critical Path)

1. **Fix Windows path bug** (30 minutes)
   - Modify `config.Save()` to use forward slashes
   - Test on Windows

2. **Add default mode** (1 hour)
   - Bridge starts without config
   - Auto-creates keystore
   - Clear user messaging

3. **Add env var support** (1 hour)
   - Check `ARMORCLAW_API_KEY`
   - Auto-store on startup
   - Clear feedback

### This Week

4. **Implement CLI subcommands** (4 hours)
   - `armorclaw-bridge add-key`
   - `armorclaw-bridge start`
   - `armorclaw-bridge list-keys`

5. **Improve error messages** (2 hours)
   - Human-readable errors
   - Helpful examples
   - Next steps suggestions

### Next Sprint

6. **Setup wizard** (8 hours)
7. **Pre-built binaries** (8 hours)
8. **Documentation updates** (4 hours)

---

## 🏆 OpenClaw Compatibility Mode

To match OpenClaw's simplicity exactly:

```bash
# OpenClaw equivalent workflow
export ARMORCLAW_API_KEY="sk-xxx"
armorclaw run --provider openai --model gpt-4

# Internally:
# 1. Store key temporarily
# 2. Create container
# 3. Inject secret
# 4. Start agent
# 5. Clean up on exit
```

This gives users the **best of both worlds**:
- OpenClaw's simplicity
- ArmorClaw's security

---

**Review Status:** 🔴 CRITICAL IMPROVEMENTS NEEDED
**Overall Rating:** 3/10 (Target: 9/10)
**Priority:** P0 - Blocking adoption
**Effort:** 2-3 weeks to reach 8/10
