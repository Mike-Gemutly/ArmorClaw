# Lessons Learned

## Windows Cross-Platform Support

### Daemon / Background Process
- `sevlyar/go-daemon` uses POSIX `fork()` and does not work on Windows. Split into platform-specific files using build tags (`daemon_unix.go` / `daemon_windows.go`).
- On Windows, use `exec.Command` with `syscall.CREATE_NEW_PROCESS_GROUP` and `Process.Release()` to spawn a detached background process.

### Signal Handling
- `syscall.SIGTERM` compiles on Windows but is never delivered by the OS. Use `os.Interrupt` for portable Ctrl+C handling.
- `os.Interrupt` is equivalent to `syscall.SIGINT` and works on all platforms.

### Android SDK Paths
- The Android code already handles Windows SDK paths (`LOCALAPPDATA\Android\Sdk`) and appends `.exe` to `adb` and `emulator` binaries.
- Always use `runtime.GOOS == "windows"` checks when constructing platform-specific executable paths.

### iOS Simulator
- `GetSimulators()` already returns empty on non-darwin. iOS simulator and `/bin/ps` usage are guarded by `runtime.GOOS` checks.
- No changes needed for iOS-only code paths when targeting Windows + Android.

## Agent Skills (Cross-Platform)

### Skill Directory Standard
- Agent skills use the `.agents/skills/<skill-name>/SKILL.md` format with YAML frontmatter (`name`, `description`).
- This structure is compatible across multiple agent platforms (Claude, OpenCode, Charm/Crush CLI, Droid Factory, Warp).
- Place project-specific skills in the repo root `.agents/skills/` directory; personal/global skills go in `~/.agents/skills/`.

### Platform Targeting
- Skills scoped to Windows/Linux/Android must explicitly exclude iOS artifacts (`.swift`, `.storyboard`, `.xib`, `Podfile`).
- Use `mobilecli` CLI commands as the preferred tool for device interaction (screenshots, navigation); fall back to `adb` when the binary is unavailable.
- Screenshot capture relies on `mobilecli screenshot` or `adb exec-out screencap -p` — both work on Windows and Linux with a connected Android device or emulator.
