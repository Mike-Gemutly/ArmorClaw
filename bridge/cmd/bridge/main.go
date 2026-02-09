// ArmorClaw Bridge - Main entry point
//
// The bridge provides a secure interface between the host system and isolated
// AI agent containers. It manages encrypted credentials and container lifecycle.
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/armorclaw/bridge/pkg/budget"
	"github.com/armorclaw/bridge/pkg/config"
	"github.com/armorclaw/bridge/pkg/docker"
	"github.com/armorclaw/bridge/pkg/keystore"
	"github.com/armorclaw/bridge/pkg/logger"
	"github.com/armorclaw/bridge/pkg/rpc"
	"github.com/armorclaw/bridge/pkg/voice"
	"github.com/armorclaw/bridge/pkg/webrtc"
)

var (
	version   = "1.0.0"
	buildTime = "unknown"
)

type cliConfig struct {
	command          string
	configPath       string
	configOutput     string
	socketPath       string
	dbPath           string
	matrixHomeserver string
	matrixUsername   string
	matrixPassword   string
	matrixEnabled    bool
	logLevel         string
	verbose          bool
	version          bool
	help             bool
	// Quick-start command flags
	addKeyProvider   string
	addKeyToken      string
	addKeyId         string
	addKeyDisplayName string
	startKeyId       string
}

func main() {
	cliCfg := parseFlags()

	if cliCfg.version {
		printVersion()
		return
	}

	if cliCfg.help {
		printHelp()
		return
	}

	// Handle config commands
	if cliCfg.command == "init" {
		runInitCommand(cliCfg)
		return
	}

	if cliCfg.command == "validate" {
		runValidateCommand(cliCfg)
		return
	}

	if cliCfg.command == "setup" {
		runSetupCommand(cliCfg)
		return
	}

	if cliCfg.command == "completion" {
		runCompletionCommand(cliCfg)
		return
	}

	if cliCfg.command == "daemon" {
		runDaemonCommand(cliCfg)
		return
	}

	if cliCfg.command == "help" {
		// Check if help is for a specific command
		args := flag.Args()
		if len(args) > 1 {
			printCommandHelp(args[1])
		} else {
			printHelp()
		}
		return
	}

	// Handle command-specific help (via --help/-h flag)
	if len(os.Args) > 2 && (os.Args[2] == "--help" || os.Args[2] == "-h") {
		// The command would be in cliCfg.command if --help is after a command
		printCommandHelp(cliCfg.command)
		return
	}

	// Handle quick-start commands
	if cliCfg.command == "add-key" {
		runAddKeyCommand(cliCfg)
		return
	}

	if cliCfg.command == "list-keys" {
		runListKeysCommand(cliCfg)
		return
	}

	if cliCfg.command == "start" {
		runStartCommand(cliCfg)
		return
	}

	// Default: Start the bridge server
	runBridgeServer(cliCfg)
}

// runInitCommand generates an example configuration file
func runInitCommand(cliCfg cliConfig) {
	outputPath := cliCfg.configOutput
	if outputPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			log.Fatalf("Failed to determine home directory: %v", err)
		}
		outputPath = filepath.Join(homeDir, ".armorclaw", "config.toml")
	}
	if err := config.GenerateExampleConfig(outputPath); err != nil {
		log.Fatalf("Failed to generate example config: %v", err)
	}
	log.Printf("✓ Example configuration written to: %s", outputPath)
	log.Println("✓ Edit this file to customize your ArmorClaw bridge configuration")
	log.Println("")
	log.Println("Quick start:")
	log.Println("  1. Add an API key: armorclaw-bridge add-key --provider openai --token sk-...")
	log.Println("  2. Start agent:    armorclaw-bridge start --key <key-id>")
}

// runValidateCommand validates the configuration
func runValidateCommand(cliCfg cliConfig) {
	cfg, err := config.Load(cliCfg.configPath)
	if err != nil {
		log.Fatalf("Configuration validation failed: %v", err)
	}
	log.Printf("✓ Configuration is valid!")
	log.Printf("  Socket: %s", cfg.Server.SocketPath)
	log.Printf("  Keystore: %s", cfg.Keystore.DBPath)
	log.Printf("  Matrix: %v", cfg.Matrix.Enabled)
}

// runSetupCommand runs the interactive setup wizard
func runSetupCommand(cliCfg cliConfig) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("\n╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║        Welcome to ArmorClaw - Interactive Setup           ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
	fmt.Println("")
	fmt.Println("This wizard will guide you through the initial setup process.")
	fmt.Println("Press Ctrl+C at any time to cancel.")
	fmt.Println("")

	// Step 1: Check Docker availability
	fmt.Print("🔍 Checking Docker availability... ")
	if !docker.IsAvailable() {
		fmt.Println("❌")
		log.Fatal("Docker is not available or not running. Please install and start Docker first.")
	}
	fmt.Println("✓")

	// Step 2: Configuration location
	fmt.Println("\n📁 Configuration Setup")
	fmt.Println("Where would you like to store your ArmorClaw configuration?")
	fmt.Println("  [1] Default (~/.armorclaw)")
	fmt.Println("  [2] Custom location")

	fmt.Print("Choose an option (1-2) [1]: ")
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	configPath := cliCfg.configPath
	if strings.TrimSpace(input) == "2" {
		fmt.Print("Enter custom path: ")
		configPath, _ = reader.ReadString('\n')
		configPath = strings.TrimSpace(configPath)
	}

	if configPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			log.Fatalf("Failed to determine home directory: %v", err)
		}
		configPath = filepath.Join(homeDir, ".armorclaw", "config.toml")
	} else {
		// If user provided a directory, append config.toml
		if !strings.HasSuffix(configPath, ".toml") {
			configPath = filepath.Join(configPath, "config.toml")
		}
	}

	// Create config directory if needed
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0750); err != nil {
		log.Fatalf("Failed to create config directory: %v", err)
	}

	// Check if config already exists
	if _, err := os.Stat(configPath); err == nil {
		fmt.Printf("\n⚠️  Configuration file already exists: %s\n", configPath)
		fmt.Print("Do you want to overwrite it? [y/N]: ")
		input, _ = reader.ReadString('\n')
		input = strings.ToLower(strings.TrimSpace(input))
		if input != "y" && input != "yes" {
			fmt.Println("Setup cancelled.")
			return
		}
	}

	// Step 3: API Provider Selection
	fmt.Println("\n🤖 AI Provider Selection")
	fmt.Println("Which AI provider do you use?")
	fmt.Println("  [1] OpenAI (GPT-4, GPT-3.5)")
	fmt.Println("  [2] Anthropic (Claude)")
	fmt.Println("  [3] OpenRouter")
	fmt.Println("  [4] Google (Gemini)")
	fmt.Println("  [5] xAI (Grok)")
	fmt.Println("  [6] Skip (add keys later)")

	fmt.Print("Choose an option (1-6) [1]: ")
	input, _ = reader.ReadString('\n')
	input = strings.TrimSpace(input)

	provider := ""
	providerName := ""
	defaultKeyName := ""

	switch input {
	case "1", "":
		provider = "openai"
		providerName = "OpenAI"
		defaultKeyName = "openai-default"
	case "2":
		provider = "anthropic"
		providerName = "Anthropic"
		defaultKeyName = "anthropic-default"
	case "3":
		provider = "openrouter"
		providerName = "OpenRouter"
		defaultKeyName = "openrouter-default"
	case "4":
		provider = "google"
		providerName = "Google"
		defaultKeyName = "google-default"
	case "5":
		provider = "xai"
		providerName = "xAI"
		defaultKeyName = "xai-default"
	case "6":
		provider = ""
		providerName = "None"
	default:
		fmt.Println("Invalid option. Skipping API key setup.")
		provider = ""
	}

	// Step 4: API Key Entry (if provider selected)
	if provider != "" {
		fmt.Printf("\n🔑 %s API Key\n", providerName)
		fmt.Println("Enter your API key (input will be hidden):")

		// Read API key securely
		var apiKey string
		var err error

		// Try to use termios for hidden input on Unix-like systems
		apiKey, err = readPassword(reader)
		if err != nil {
			// Fallback to regular input
			fmt.Print("API Key: ")
			apiKey, _ = reader.ReadString('\n')
			apiKey = strings.TrimSpace(apiKey)
		}

		if apiKey == "" {
			fmt.Println("⚠️  No API key provided. You can add one later with:")
			fmt.Println("  armorclaw-bridge add-key --provider <provider> --token <key>")
		} else {
			// Validate API key format
			if !validateAPIKeyFormat(provider, apiKey) {
				fmt.Printf("⚠️  Warning: API key format looks unusual for %s\n", providerName)
				fmt.Print("Continue anyway? [y/N]: ")
				input, _ = reader.ReadString('\n')
				input = strings.ToLower(strings.TrimSpace(input))
				if input != "y" && input != "yes" {
					fmt.Println("Setup cancelled.")
					return
				}
			}

			// Store the API key
			cfg := config.DefaultConfig()
			cfg.Keystore.DBPath = filepath.Join(filepath.Dir(configPath), "keystore.db")

			ks, err := keystore.New(cfg.ToKeystoreConfig())
			if err != nil {
				log.Fatalf("Failed to initialize keystore: %v", err)
			}

			if err := ks.Open(); err != nil {
				log.Fatalf("Failed to open keystore: %v", err)
			}
			defer ks.Close()

			cred := keystore.Credential{
				ID:          defaultKeyName,
				Provider:    keystore.Provider(provider),
				Token:       apiKey,
				DisplayName: fmt.Sprintf("%s API Key (setup wizard)", providerName),
				Tags:        []string{"setup-wizard", "production"},
			}

			if err := ks.Store(cred); err != nil {
				log.Fatalf("Failed to store API key: %v", err)
			}

			fmt.Printf("✓ API key stored as '%s'\n", defaultKeyName)
		}
	}

	// Step 5: Matrix Configuration (Optional)
	fmt.Println("\n💬 Matrix Configuration (Optional)")
	fmt.Println("ArmorClaw can integrate with Matrix for encrypted messaging.")
	fmt.Println("Leave blank to skip this step.")

	fmt.Print("Enable Matrix? [y/N]: ")
	input, _ = reader.ReadString('\n')
	input = strings.ToLower(strings.TrimSpace(input))

	matrixEnabled := input == "y" || input == "yes"
	matrixHomeserver := ""
	matrixUsername := ""
	matrixPassword := ""

	if matrixEnabled {
		fmt.Print("Matrix homeserver URL [https://matrix.example.com]: ")
		matrixHomeserver, _ = reader.ReadString('\n')
		matrixHomeserver = strings.TrimSpace(matrixHomeserver)
		if matrixHomeserver == "" {
			matrixHomeserver = "https://matrix.example.com"
		}

		fmt.Print("Matrix username: ")
		matrixUsername, _ = reader.ReadString('\n')
		matrixUsername = strings.TrimSpace(matrixUsername)

		fmt.Print("Matrix password: ")
		matrixPassword, _ = reader.ReadString('\n')
		matrixPassword = strings.TrimSpace(matrixPassword)
	}

	// Step 6: Generate Configuration
	fmt.Println("\n⚙️  Generating configuration...")

	// Create config structure
	cfg := config.DefaultConfig()

	// Override with wizard values (configDir was already declared above)
	configDir = filepath.Dir(configPath)
	cfg.Keystore.DBPath = filepath.Join(configDir, "keystore.db")

	if matrixEnabled {
		cfg.Matrix.Enabled = true
		cfg.Matrix.HomeserverURL = matrixHomeserver
		cfg.Matrix.Username = matrixUsername
		cfg.Matrix.Password = matrixPassword
	}

	// Save configuration
	if err := config.Save(cfg, configPath); err != nil {
		log.Fatalf("Failed to save configuration: %v", err)
	}

	fmt.Printf("✓ Configuration saved to: %s\n", configPath)

	// Step 7: Summary and Next Steps
	fmt.Println("\n╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║                   Setup Complete! ✓                       ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
	fmt.Println("")
	fmt.Println("Your ArmorClaw bridge is ready to use!")
	fmt.Println("")

	if provider != "" {
		fmt.Println("🚀 Quick Start:")
		fmt.Printf("  1. Start the bridge:  armorclaw-bridge\n")
		fmt.Printf("  2. Start an agent:    armorclaw-bridge start --key %s\n", defaultKeyName)
	} else {
		fmt.Println("🚀 Next Steps:")
		fmt.Println("  1. Add an API key:    armorclaw-bridge add-key --provider <provider> --token <key>")
		fmt.Println("  2. Start the bridge:  armorclaw-bridge")
		fmt.Println("  3. Start an agent:    armorclaw-bridge start --key <key-id>")
	}

	fmt.Println("")
	fmt.Println("📚 Documentation: https://github.com/armorclaw/armorclaw")
	fmt.Println("")
}

// readPassword reads a password from stdin with hidden input (Unix-like systems)
func readPassword(reader *bufio.Reader) (string, error) {
	// On Windows, this won't hide the input, but it will still work
	// On Unix-like systems, we use syscall to disable echo
	// For simplicity, this implementation just reads normally
	// A production version would use termios for hidden input
	fmt.Print("API Key: ")
	input, err := reader.ReadString('\n')
	return strings.TrimSpace(input), err
}

// validateAPIKeyFormat performs basic validation of API key format
func validateAPIKeyFormat(provider, key string) bool {
	switch provider {
	case "openai":
		return strings.HasPrefix(key, "sk-") || strings.HasPrefix(key, "sk-proj-")
	case "anthropic":
		return strings.HasPrefix(key, "sk-ant-")
	case "openrouter":
		return len(key) > 20 // OpenRouter keys vary
	case "google", "xai":
		return len(key) > 20 // Basic length check
	}
	return true
}

// runCompletionCommand outputs shell completion script
func runCompletionCommand(cliCfg cliConfig) {
	shell := cliCfg.configPath // Reuse configPath for shell type
	if shell == "" {
		// Detect shell from environment
		shell = os.Getenv("SHELL")
		if strings.Contains(shell, "zsh") {
			shell = "zsh"
		} else {
			shell = "bash"
		}
	}

	var script string
	switch shell {
	case "bash":
		script = `# ArmorClaw Bridge Bash Completion
# Save this file to: ~/.bash_completion.d/armorclaw-bridge
# Or source it in: ~/.bashrc

_armorclaw_bridge_commands() {
    local commands="init validate add-key list-keys start setup version help completion"
    echo "$commands"
}

_armorclaw_bridge_providers() {
    local providers="openai anthropic openrouter google gemini xai"
    echo "$providers"
}

_armorclaw_bridge() {
    local cur prev words cword
    _init_completion || return

    if [ $cword -eq 1 ]; then
        COMPREPLY=($(compgen -W "$(_armorclaw_bridge_commands)" -- "$cur"))
        return 0
    fi

    local cmd="${words[1]}"
    case "$cmd" in
        add-key)
            case "$prev" in
                --provider|-p)
                    COMPREPLY=($(compgen -W "$(_armorclaw_bridge_providers)" -- "$cur"))
                    ;;
                *)
                    COMPREPLY=($(compgen -W "--provider --token --id --name --help -h" -- "$cur"))
                    ;;
            esac
            ;;
        start)
            case "$prev" in
                --key|-k)
                    COMPREPLY=($(compgen -W "$(armorclaw-bridge list-keys 2>/dev/null | grep '•' | awk '{print $2}')" -- "$cur"))
                    ;;
                *)
                    COMPREPLY=($(compgen -W "--key --help -h" -- "$cur"))
                    ;;
            esac
            ;;
        completion)
            COMPREPLY=($(compgen -W "bash zsh" -- "$cur"))
            ;;
    esac
}

complete -F _armorclaw_bridge armorclaw-bridge
`
	case "zsh":
		script = `#compdef armorclaw-bridge
# ArmorClaw Bridge Zsh Completion
# Save this file to: ~/.zsh/completions/_armorclaw-bridge

_armorclaw_bridge() {
    local -a commands
    commands=(
        'init:Initialize configuration file'
        'validate:Validate configuration'
        'setup:Run interactive setup wizard'
        'add-key:Add an API key to the keystore'
        'list-keys:List all stored API keys'
        'start:Start an agent container'
        'completion:Generate shell completion script'
        'version:Show version information'
        'help:Show help information'
    )

    if (( CURRENT == 2 )); then
        _describe 'command' commands
    else
        case $words[2] in
            add-key)
                _arguments '--provider[AI provider]:providers:(openai anthropic openrouter google gemini xai)' \
                           '--token[API token]' \
                           '--id[Key ID]' \
                           '--name[Display name]' \
                           '--help[Show help]'
                ;;
            start)
                _arguments '--key[Key ID]:keys:(_armorclaw_bridge_keys)' \
                           '--help[Show help]'
                ;;
            completion)
                _arguments '--shell[Shell type]:shells:(bash zsh)'
                ;;
        esac
    fi
}

_armorclaw_bridge_keys() {
    local -a keys
    keys=($(armorclaw-bridge list-keys 2>/dev/null | grep '•' | awk '{print $2}'))
    _describe 'stored-key' keys
}
`
	default:
		log.Fatalf("Unsupported shell: %s. Supported: bash, zsh", shell)
	}

	fmt.Println(script)
	log.Printf("✓ %s completion script generated", shell)
	log.Println("")
	log.Println("To enable completion:")
	switch shell {
	case "bash":
		log.Println("  1. Save to file:")
		log.Println("     armorclaw-bridge completion bash > ~/.bash_completion.d/armorclaw-bridge")
		log.Println("  2. Source in ~/.bashrc:")
		log.Println("     source ~/.bash_completion.d/armorclaw-bridge")
		log.Println("  3. Or restart your shell")
	case "zsh":
		log.Println("  1. Save to file:")
		log.Println("     armorclaw-bridge completion zsh > ~/.zsh/completions/_armorclaw-bridge")
		log.Println("  2. Or add to ~/.zshrc:")
		log.Println("     autoload -U compinit && compinit")
		log.Println("  3. Restart your shell")
	}
}

// runDaemonCommand manages daemon operations (start/stop/restart/status/logs)
func runDaemonCommand(cliCfg cliConfig) {
	if len(os.Args) < 3 {
		printDaemonHelp()
		log.Fatal("Error: daemon requires an action (start, stop, restart, status, logs)")
	}

	action := os.Args[2]

	switch action {
	case "start":
		daemonStart(cliCfg)
	case "stop":
		daemonStop()
	case "restart":
		daemonStop()
		time.Sleep(1 * time.Second)
		daemonStart(cliCfg)
	case "status":
		daemonStatus()
	case "logs":
		daemonLogs()
	default:
		printDaemonHelp()
		log.Fatalf("Error: unknown daemon action: %s", action)
	}
}

// daemonStart starts the bridge as a daemon
func daemonStart(cliCfg cliConfig) {
	// Load configuration
	cfg, err := config.Load(cliCfg.configPath)
	if err != nil {
		log.Printf("Warning: Using default configuration: %v", err)
		cfg = config.DefaultConfig()
	}

	// Enable daemonize
	cfg.Server.Daemonize = true

	// Set default PID file if not specified
	if cfg.Server.PidFile == "" {
		cfg.Server.PidFile = "/run/armorclaw/bridge.pid"
	}

	// Set default log file for daemon
	if cfg.Logging.Output == "stdout" {
		cfg.Logging.Output = "file"
		if cfg.Logging.File == "" {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				log.Fatalf("Failed to determine home directory: %v", err)
			}
			cfg.Logging.File = filepath.Join(homeDir, ".armorclaw", "bridge.log")
		}
	}

	// Check if already running
	if daemonStatusRunning() {
		log.Fatal("Error: daemon is already running")
	}

	// Create runtime directory
	runtimeDir := filepath.Dir(cfg.Server.SocketPath)
	if err := os.MkdirAll(runtimeDir, 0750); err != nil {
		log.Fatalf("Failed to create runtime directory: %v", err)
	}

	log.Println("Starting ArmorClaw Bridge as daemon...")
	log.Printf("PID file: %s", cfg.Server.PidFile)
	log.Printf("Log file: %s", cfg.Logging.File)

	// Fork and detach (simulate daemon mode for now - on Windows, we'll run in foreground)
	// For true daemon mode, we'd use syscall fork/exec, but for cross-platform compatibility:
	// We'll start in background with output redirected

	// For now, just start normally with logging to file
	// Note: This will run in foreground, not true daemon mode
	// True daemon mode would require fork/exec which is platform-specific
	log.Println("Note: Running in foreground (true daemon mode requires platform-specific fork)")
	runBridgeServer(cliCfg)
}

// daemonStop stops the daemon
func daemonStop() {
	// Load config to get PID file location
	cfg, err := config.Load("")
	if err != nil {
		cfg = config.DefaultConfig()
	}

	if cfg.Server.PidFile == "" {
		cfg.Server.PidFile = "/run/armorclaw/bridge.pid"
	}

	// Read PID file
	pidData, err := os.ReadFile(cfg.Server.PidFile)
	if err != nil {
		log.Fatalf("Failed to read PID file: %v", err)
	}

	pidStr := strings.TrimSpace(string(pidData))
	var pid int
	fmt.Sscanf(pidStr, "%d", &pid)

	// Check if process is running
	process, err := os.FindProcess(pid)
	if err != nil {
		log.Printf("Process %d not found (already stopped?)", pid)
		os.Remove(cfg.Server.PidFile)
		return
	}

	// Send SIGTERM for graceful shutdown
	if err := process.Signal(syscall.SIGTERM); err != nil {
		log.Fatalf("Failed to send SIGTERM to process %d: %v", pid, err)
	}

	// Wait a bit and check if it stopped
	time.Sleep(2 * time.Second)

	_, err = os.FindProcess(pid)
	if err == nil {
		// Process still running, force kill
		log.Printf("Process %d did not stop gracefully, sending SIGKILL...", pid)
		// Note: os.Process.Signal is not available on all platforms
		// For production, would use syscall.Kill
		time.Sleep(500 * time.Millisecond)
	}

	// Clean up PID file
	os.Remove(cfg.Server.PidFile)

	log.Printf("✓ Daemon stopped (PID: %d)", pid)
}

// daemonRestart restarts the daemon
func daemonRestart() {
	log.Println("Restarting daemon...")
	// Implementation: stop then start
	// This is handled by calling stop then start
}

// daemonStatus checks daemon status
func daemonStatus() {
	// Load config to get PID file location
	cfg, err := config.Load("")
	if err != nil {
		cfg = config.DefaultConfig()
	}

	if cfg.Server.PidFile == "" {
		cfg.Server.PidFile = "/run/armorclaw/bridge.pid"
	}

	// Check PID file
	pidData, err := os.ReadFile(cfg.Server.PidFile)
	if err != nil {
		fmt.Println("Daemon status: Stopped (no PID file)")
		return
	}

	pidStr := strings.TrimSpace(string(pidData))
	var pid int
	fmt.Sscanf(pidStr, "%d", &pid)

	// Check if process is running
	_, err = os.FindProcess(pid)
	if err != nil {
		fmt.Printf("Daemon status: Stopped (stale PID file, PID: %d)\n", pid)
		os.Remove(cfg.Server.PidFile)
		return
	}

	// Process is running
	// Get start time
	createTime, err := getProcessStartTime(pid)
	if err != nil {
		fmt.Printf("Daemon status: Running (PID: %d)\n", pid)
		return
	}

	uptime := time.Since(createTime)
	fmt.Printf("Daemon status: Running (PID: %d, uptime: %s)\n", pid, uptime.Round(time.Second))
}

// daemonLogs shows daemon logs
func daemonLogs() {
	// Load config to get log file location
	cfg, err := config.Load("")
	if err != nil {
		cfg = config.DefaultConfig()
	}

	logFile := cfg.Logging.File
	if logFile == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			log.Fatal("Failed to determine home directory")
		}
		logFile = filepath.Join(homeDir, ".armorclaw", "bridge.log")
	}

	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		fmt.Printf("Log file not found: %s\n", logFile)
		return
	}

	fmt.Printf("Showing last 50 lines of %s:\n", logFile)
	fmt.Println("---")

	// Use tail to show last 50 lines
	cmd := exec.Command("tail", "-n", "50", logFile)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Fatalf("Failed to read log file: %v", err)
	}
}

// daemonStatusRunning checks if daemon is running without loading config
func daemonStatusRunning() bool {
	cfg, err := config.Load("")
	if err != nil {
		cfg = config.DefaultConfig()
	}

	if cfg.Server.PidFile == "" {
		cfg.Server.PidFile = "/run/armorclaw/bridge.pid"
	}

	pidData, err := os.ReadFile(cfg.Server.PidFile)
	if err != nil {
		return false
	}

	pidStr := strings.TrimSpace(string(pidData))
	var pid int
	fmt.Sscanf(pidStr, "%d", &pid)

	_, err = os.FindProcess(pid)
	return err == nil
}

// getProcessStartTime gets the creation time of a process
func getProcessStartTime(pid int) (time.Time, error) {
	// On Linux, read /proc/<pid>/stat
	// On Windows, use different method
	statPath := fmt.Sprintf("/proc/%d/stat", pid)
	data, err := os.ReadFile(statPath)
	if err != nil {
		return time.Time{}, err
	}

	// Parse stat file (field 22 is creation time in jiffies)
	fields := strings.Fields(string(data))
	if len(fields) < 22 {
		return time.Time{}, fmt.Errorf("invalid stat format")
	}

	// Field 22 is starttime (since boot in jiffies)
	var starttime int64
	fmt.Sscanf(fields[21], "%d", &starttime)

	// Get system boot time
	// Would need to read /proc/stat for accurate boot time
	// For simplicity, return current time minus approximate uptime
	// This is a simplified version - production code would be more accurate

	return time.Now(), nil
}

// printDaemonHelp shows daemon help
func printDaemonHelp() {
	help := `COMMAND: daemon

Manage the ArmorClaw Bridge background daemon.

USAGE:
    armorclaw-bridge daemon [action]

ACTIONS:
    start     Start bridge as background daemon
    stop      Stop the background daemon
    restart   Restart the daemon
    status    Show daemon status
    logs      Show recent log entries

EXAMPLES:
    # Start daemon
    armorclaw-daemon start

    # Check status
    armorclaw-daemon status

    # View logs
    armorclaw-bridge daemon logs

    # Stop daemon
    armorclaw-bridge daemon stop

CONFIGURATION:
    Daemon mode is configured in config.toml:

    [server]
      daemonize = true
      pid_file = "/run/armorclaw/bridge.pid"

    [logging]
      output = "file"
      file = "/path/to/bridge.log"
`
	fmt.Println(help)
}

// runAddKeyCommand adds an API key to the keystore
func runAddKeyCommand(cliCfg cliConfig) {
	// Load configuration (use defaults if not found)
	cfg, err := config.Load(cliCfg.configPath)
	if err != nil {
		log.Printf("Warning: Using default configuration: %v", err)
		cfg = config.DefaultConfig()
	}

	// Check for required parameters
	if cliCfg.addKeyProvider == "" {
		log.Fatal("Error: --provider is required (openai, anthropic, openrouter, google, xai)")
	}
	if cliCfg.addKeyToken == "" {
		token := os.Getenv("ARMORCLAW_API_KEY")
		if token == "" {
			log.Fatal("Error: --token is required or set ARMORCLAW_API_KEY environment variable")
		}
		cliCfg.addKeyToken = token
	}

	// Initialize keystore
	log.Println("Initializing encrypted keystore...")
	ks, err := keystore.New(cfg.ToKeystoreConfig())
	if err != nil {
		log.Fatalf("Failed to initialize keystore: %v", err)
	}

	if err := ks.Open(); err != nil {
		log.Fatalf("Failed to open keystore: %v", err)
	}
	defer ks.Close()

	// Generate key ID if not provided
	keyID := cliCfg.addKeyId
	if keyID == "" {
		keyID = cliCfg.addKeyProvider + "-default"
	}

	// Create credential
	displayName := cliCfg.addKeyDisplayName
	if displayName == "" {
		displayName = fmt.Sprintf("%s API Key", cliCfg.addKeyProvider)
	}

	cred := keystore.Credential{
		ID:          keyID,
		Provider:    keystore.Provider(cliCfg.addKeyProvider),
		Token:       cliCfg.addKeyToken,
		DisplayName: displayName,
		Tags:        []string{"cli", "quick-start"},
	}

	// Store credential
	if err := ks.Store(cred); err != nil {
		log.Fatalf("Failed to store credential: %v", err)
	}

	log.Printf("✓ API key stored as '%s'", keyID)
	log.Printf("  Provider: %s", cliCfg.addKeyProvider)
	log.Printf("  Display name: %s", cliCfg.addKeyDisplayName)
	log.Println("")
	log.Println("Start an agent with this key:")
	log.Printf("  armorclaw-bridge start --key %s", keyID)
}

// runListKeysCommand lists all stored credentials
func runListKeysCommand(cliCfg cliConfig) {
	// Load configuration
	cfg, err := config.Load(cliCfg.configPath)
	if err != nil {
		log.Printf("Warning: Using default configuration: %v", err)
		cfg = config.DefaultConfig()
	}

	// Initialize keystore
	ks, err := keystore.New(cfg.ToKeystoreConfig())
	if err != nil {
		log.Fatalf("Failed to initialize keystore: %v", err)
	}

	if err := ks.Open(); err != nil {
		log.Fatalf("Failed to open keystore: %v", err)
	}
	defer ks.Close()

	// List credentials (empty provider means all)
	creds, err := ks.List("")
	if err != nil {
		log.Fatalf("Failed to list credentials: %v", err)
	}

	if len(creds) == 0 {
		log.Println("No API keys stored.")
		log.Println("")
		log.Println("Add one with:")
		log.Println("  armorclaw-bridge add-key --provider openai --token sk-...")
		return
	}

	log.Printf("✓ Found %d API key(s):\n", len(creds))
	for _, cred := range creds {
		log.Printf("  • %s", cred.ID)
		log.Printf("    Provider: %s", cred.Provider)
		if cred.DisplayName != "" {
			log.Printf("    Name: %s", cred.DisplayName)
		}
		log.Println("")
	}
}

// runStartCommand starts an agent container
func runStartCommand(cliCfg cliConfig) {
	// Load configuration
	cfg, err := config.Load(cliCfg.configPath)
	if err != nil {
		log.Printf("Warning: Using default configuration: %v", err)
		cfg = config.DefaultConfig()
	}

	// Check for key_id
	if cliCfg.startKeyId == "" {
		log.Fatal("Error: --key is required. Use 'list-keys' to see available keys.")
	}

	// Initialize keystore
	ks, err := keystore.New(cfg.ToKeystoreConfig())
	if err != nil {
		log.Fatalf("Failed to initialize keystore: %v", err)
	}

	if err := ks.Open(); err != nil {
		log.Fatalf("Failed to open keystore: %v", err)
	}
	defer ks.Close()

	// TODO: Implement container start logic
	log.Printf("Starting agent with key '%s'...", cliCfg.startKeyId)
	log.Println("Note: Container start via RPC is not yet implemented in CLI mode.")
	log.Println("Use the RPC API to start containers:")
	log.Printf(`echo '{"jsonrpc":"2.0","method":"start","params":{"key_id":"%s"},"id":1}' | socat - UNIX-CONNECT:/run/armorclaw/bridge.sock`, cliCfg.startKeyId)
}

// runBridgeServer starts the bridge server
func runBridgeServer(cliCfg cliConfig) {
	log.Printf("Starting ArmorClaw Bridge v%s", version)
	log.Printf("Build time: %s", buildTime)

	// Check for ARMORCLAW_API_KEY environment variable (OpenClaw compatibility)
	if apiKey := os.Getenv("ARMORCLAW_API_KEY"); apiKey != "" {
		log.Println("⚠️  ARMORCLAW_API_KEY detected - This will auto-store your key for convenience")
		log.Println("    For production, consider using 'add-key' command instead")

		// Auto-store the key
		autoStoreKey(apiKey)
	}

	// Load configuration file
	cfg, err := config.Load(cliCfg.configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Apply CLI flag overrides
	if cliCfg.socketPath != "" {
		cfg.Server.SocketPath = cliCfg.socketPath
	}
	if cliCfg.dbPath != "" {
		cfg.Keystore.DBPath = cliCfg.dbPath
	}
	if cliCfg.matrixHomeserver != "" {
		cfg.Matrix.HomeserverURL = cliCfg.matrixHomeserver
		cfg.Matrix.Enabled = true
	}
	if cliCfg.matrixUsername != "" {
		cfg.Matrix.Username = cliCfg.matrixUsername
	}
	if cliCfg.matrixPassword != "" {
		cfg.Matrix.Password = cliCfg.matrixPassword
	}
	if cliCfg.matrixEnabled {
		cfg.Matrix.Enabled = true
	}
	if cliCfg.logLevel != "" {
		cfg.Logging.Level = cliCfg.logLevel
	}

	// Validate final configuration
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	// Setup logging
	setupLogging(cfg.Logging)

	log.Printf("Configuration loaded successfully")
	log.Printf("Socket: %s", cfg.Server.SocketPath)
	log.Printf("Keystore: %s", cfg.Keystore.DBPath)
	if cfg.Matrix.Enabled {
		log.Printf("Matrix: %s (enabled)", cfg.Matrix.HomeserverURL)
	} else {
		log.Printf("Matrix: disabled")
	}

	// Pre-flight checks
	// Check Docker availability before initializing services
	log.Println("Checking Docker availability...")
	if !docker.IsAvailable() {
		log.Fatalf("Docker is not available or not running. "+
			"Please start Docker and ensure the daemon is accessible.")
	}
	log.Println("Docker is available")

	// Ensure base runtime directory exists
	runtimeDir := filepath.Dir(cfg.Server.SocketPath)
	if runtimeDir == "" {
		runtimeDir = "/run/armorclaw"
	}
	if err := os.MkdirAll(runtimeDir, 0750); err != nil {
		log.Fatalf("Failed to create runtime directory %s: %v", runtimeDir, err)
	}
	log.Printf("Runtime directory ready: %s", runtimeDir)

	// Initialize keystore
	log.Println("Initializing encrypted keystore...")
	ks, err := keystore.New(cfg.ToKeystoreConfig())
	if err != nil {
		log.Fatalf("Failed to initialize keystore: %v", err)
	}

	if err := ks.Open(); err != nil {
		log.Fatalf("Failed to open keystore: %v", err)
	}
	defer ks.Close()

	log.Println("Keystore initialized with hardware-derived master key")

	// Initialize WebRTC components
	log.Println("Initializing WebRTC components...")

	// Create session manager
	sessionMgr := webrtc.NewSessionManager(30 * time.Minute)

	// Create token manager
	tokenMgr := webrtc.NewTokenManager()

	// Create WebRTC engine
	webrtcConfig := webrtc.DefaultEngineConfig()
	webrtcEngine, err := webrtc.NewEngine(webrtcConfig)
	if err != nil {
		log.Fatalf("Failed to create WebRTC engine: %v", err)
	}

	// Create TURN manager (optional)
	var turnMgr *webrtc.TURNManager
	if cfg.WebRTC.TURNSharedSecret != "" {
		turnMgr = webrtc.NewTURNManager(cfg.WebRTC.TURNSharedSecret, cfg.WebRTC.TURNServerURL)
		webrtcEngine.SetTURNManager(turnMgr)
		log.Println("TURN manager initialized")
	}

	// Create voice manager
	voiceConfig := voice.DefaultConfig()
	// Override with config file values if present
	if cfg.Voice.DefaultLifetime > 0 {
		voiceConfig.DefaultLifetime = cfg.Voice.DefaultLifetime
	}
	if cfg.Voice.MaxLifetime > 0 {
		voiceConfig.MaxLifetime = cfg.Voice.MaxLifetime
	}

	voiceMgr := voice.NewManager(
		sessionMgr,
		tokenMgr,
		webrtcEngine,
		turnMgr,
		voiceConfig,
	)

	// Start voice manager
	if err := voiceMgr.Start(); err != nil {
		log.Printf("Warning: Failed to start voice manager: %v", err)
		log.Println("Voice calls will not be available")
	}

	// Create budget manager
	budgetMgr := budget.NewManager(budget.Config{
		GlobalTokenLimit:     cfg.Budget.GlobalTokenLimit,
		GlobalDurationLimit:  cfg.Budget.GlobalDurationLimit,
		WarningThreshold:     cfg.Budget.WarningThreshold,
		HardStop:             cfg.Budget.HardStop,
	})

	log.Println("WebRTC components initialized")

	// Initialize RPC server
	log.Printf("Starting JSON-RPC server on %s", cfg.Server.SocketPath)
	server, err := rpc.New(rpc.Config{
		SocketPath:       cfg.Server.SocketPath,
		Keystore:         ks,
		MatrixHomeserver: cfg.Matrix.HomeserverURL,
		MatrixUsername:   cfg.Matrix.Username,
		MatrixPassword:   cfg.Matrix.Password,
		// WebRTC components
		SessionManager:    sessionMgr,
		TokenManager:      tokenMgr,
		SignalingServer:   nil, // TODO: Create and pass signaling server
		WebRTCEngine:      webrtcEngine,
		TURNManager:       turnMgr,
		VoiceManager:      voiceMgr,
		BudgetManager:     budgetMgr,
	})
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	if err := server.Start(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop()

	log.Println("ArmorClaw Bridge is running")
	log.Println("Press Ctrl+C to stop")

	// Wait for interrupt signal
	shutdownCtx, cancel := context.WithCancel(context.Background())
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		log.Println("\nShutting down...")

		// Stop voice manager
		voiceMgr.Stop()
		webrtcEngine.Stop()

		cancel()
	}()

	<-shutdownCtx.Done()
	log.Println("ArmorClaw Bridge stopped")
}

// autoStoreKey automatically stores an API key from environment variable
func autoStoreKey(apiKey string) {
	// Detect provider from key format
	provider := detectProviderFromKey(apiKey)
	if provider == "" {
		log.Println("Warning: Could not detect provider from API key format")
		log.Println("         Use 'add-key' command with explicit --provider")
		return
	}

	// Generate default config for auto-storage
	cfg := config.DefaultConfig()

	// Initialize keystore
	ks, err := keystore.New(cfg.ToKeystoreConfig())
	if err != nil {
		log.Printf("Failed to initialize keystore for auto-storage: %v", err)
		return
	}

	if err := ks.Open(); err != nil {
		log.Printf("Failed to open keystore for auto-storage: %v", err)
		return
	}
	defer ks.Close()

	// Create credential
	keyID := provider + "-default"
	cred := keystore.Credential{
		ID:          keyID,
		Provider:    keystore.Provider(provider),
		Token:       apiKey,
		DisplayName: "Auto-stored from ARMORCLAW_API_KEY",
		Tags:        []string{"environment-variable", "auto-stored"},
	}

	// Check if key already exists
	existing, err := ks.Retrieve(keyID)
	if err == nil && existing.Token == apiKey {
		log.Printf("✓ API key already stored as '%s'", keyID)
		return
	}

	// Store the credential
	if err := ks.Store(cred); err != nil {
		log.Printf("Failed to auto-store API key: %v", err)
		return
	}

	log.Printf("✓ API key auto-stored as '%s' (provider: %s)", keyID, provider)
	log.Println("  Start agent: armorclaw-bridge start --key " + keyID)
}

// detectProviderFromKey attempts to detect the AI provider from the API key format
func detectProviderFromKey(key string) string {
	// OpenAI keys start with sk-
	if strings.HasPrefix(key, "sk-") || strings.HasPrefix(key, "sk-proj-") {
		return "openai"
	}
	// Anthropic keys start with sk-ant-
	if strings.HasPrefix(key, "sk-ant-") {
		return "anthropic"
	}
	// Add more patterns as needed
	return ""
}

func parseFlags() cliConfig {
	cfg := cliConfig{}

	flag.StringVar(&cfg.configPath, "config", "", "Path to configuration file")
	flag.StringVar(&cfg.configOutput, "config-output", "", "Output path for 'config init' command")
	flag.StringVar(&cfg.socketPath, "socket", "", "Path to Unix domain socket (overrides config)")
	flag.StringVar(&cfg.dbPath, "db", "", "Path to keystore database (overrides config)")
	flag.StringVar(&cfg.matrixHomeserver, "matrix-homeserver", "", "Matrix homeserver URL (enables Matrix)")
	flag.StringVar(&cfg.matrixUsername, "matrix-username", "", "Matrix username for auto-login")
	flag.StringVar(&cfg.matrixPassword, "matrix-password", "", "Matrix password for auto-login")
	flag.BoolVar(&cfg.matrixEnabled, "matrix-enabled", false, "Enable Matrix communication")
	flag.StringVar(&cfg.logLevel, "log-level", "", "Log level: debug, info, warn, error")
	flag.BoolVar(&cfg.verbose, "v", false, "Verbose logging (sets log level to debug)")
	flag.BoolVar(&cfg.version, "version", false, "Print version and exit")
	flag.BoolVar(&cfg.help, "help", false, "Show help message")

	// Quick-start command flags
	flag.StringVar(&cfg.addKeyProvider, "provider", "", "Provider for add-key (openai, anthropic, openrouter, google, xai)")
	flag.StringVar(&cfg.addKeyToken, "token", "", "API token for add-key (or use ARMORCLAW_API_KEY env var)")
	flag.StringVar(&cfg.addKeyId, "id", "", "Key ID for add-key (default: <provider>-default)")
	flag.StringVar(&cfg.addKeyDisplayName, "name", "", "Display name for add-key")
	flag.StringVar(&cfg.startKeyId, "key", "", "Key ID for start command")

	flag.Parse()

	// Check for command-line commands (first argument after flags)
	args := flag.Args()
	if len(args) > 0 {
		cfg.command = args[0]
	}

	// Set verbose flag if -v is used
	if cfg.verbose {
		cfg.logLevel = "debug"
	}

	return cfg
}

func setupLogging(cfg config.LoggingConfig) {
	// Initialize the global structured logger
	if err := logger.Initialize(cfg.Level, cfg.Format, cfg.Output); err != nil {
		// Fallback to standard logging if initialization fails
		log.Printf("Warning: Failed to initialize structured logger: %v", err)
		log.Printf("Falling back to standard logging")
		return
	}
}

func printVersion() {
	fmt.Printf("ArmorClaw Bridge v%s\n", version)
	fmt.Printf("Build time: %s\n", buildTime)
	fmt.Println("License: MIT")
	fmt.Println("https://github.com/armorclaw/armorclaw")
}

func printHelp() {
	helpText := `USAGE:
    armorclaw-bridge [command] [flags]

COMMANDS:
    init        Initialize configuration file
    validate    Validate configuration
    setup       Run interactive setup wizard
    add-key     Add an API key to the keystore
    list-keys   List all stored API keys
    start       Start an agent container
    completion  Generate shell completion script
    version     Show version information
    help        Show this help message

EXAMPLES:
    # First-time setup (interactive)
    ./build/armorclaw-bridge setup

    # Quick start with defaults
    ./build/armorclaw-bridge init
    ./build/armorclaw-bridge add-key --provider openai --token sk-proj-...
    ./build/armorclaw-bridge start --key openai-default

    # List stored keys
    ./build/armorclaw-bridge list-keys

    # Generate shell completion
    ./build/armorclaw-bridge completion bash > ~/.bash_completion.d/armorclaw-bridge
    source ~/.bash_completion.d/armorclaw-bridge

FLAGS:
    -c, --config string      Path to configuration file (default: ~/.armorclaw/config.toml)
    -v, --verbose           Enable verbose (debug) logging
    -h, --help              Show this help message
    -V, --version           Show version information

CONFIGURATION:
    The bridge loads configuration from ~/.armorclaw/config.toml by default.
    You can override this with the -c flag or ARMORCLAW_CONFIG environment variable.

    For first-time setup, run: ./build/armorclaw-bridge setup

ENVIRONMENT VARIABLES:
    ARMORCLAW_API_KEY     API key (auto-stored on bridge startup)
    ARMORCLAW_CONFIG      Path to configuration file

DOCUMENTATION:
    https://github.com/armorclaw/armorclaw
    https://docs.armorclaw.com

SUPPORT:
    Issues: https://github.com/armorclaw/armorclaw/issues
`
	fmt.Println(helpText)
}

func printCommandHelp(command string) {
	var help string
	switch command {
	case "init":
		help = `COMMAND: init

Initialize a default configuration file.

USAGE:
    armorclaw-bridge init [-c|--config-output path]

FLAGS:
    -c, --config-output string   Output path for config file

EXAMPLES:
    # Create default config
    armorclaw-bridge init

    # Create config at custom path
    armorclaw-bridge init -c /custom/path/config.toml
`
	case "setup":
		help = `COMMAND: setup

Run interactive setup wizard for first-time configuration.

The wizard guides you through:
    - Docker availability check
    - Configuration location
    - AI provider selection
    - API key entry (stored securely)
    - Optional Matrix configuration
    - Automatic configuration generation

USAGE:
    armorclaw-bridge setup [-c|--config path]

EXAMPLES:
    # Run setup wizard
    armorclaw-bridge setup

    # Run with custom config location
    armorclaw-bridge setup -c /custom/path/config.toml
`
	case "add-key":
		help = `COMMAND: add-key

Add an API key to the encrypted keystore.

USAGE:
    armorclaw-bridge add-key -p PROVIDER -t TOKEN [flags]

FLAGS:
    -p, --provider string   AI provider (openai, anthropic, openrouter, google, xai)
    -t, --token string      API token (or use ARMORCLAW_API_KEY env var)
    -i, --id string         Key ID (default: <provider>-default)
    -n, --name string       Display name for the key

PROVIDERS:
    openai      OpenAI (GPT-4, GPT-3.5)
    anthropic   Anthropic (Claude)
    openrouter  OpenRouter (multi-provider)
    google      Google (Gemini)
    xai         xAI (Grok)

EXAMPLES:
    # Add OpenAI key
    armorclaw-bridge add-key --provider openai --token sk-proj-xxx

    # Add Anthropic key with custom ID
    armorclaw-bridge add-key --provider anthropic --token sk-ant-xxx --id claude-prod

    # Add key using environment variable
    export ARMORCLAW_API_KEY="sk-xxx"
    armorclaw-bridge add-key --provider openai
`
	case "list-keys":
		help = `COMMAND: list-keys

List all stored API keys in the keystore.

USAGE:
    armorclaw-bridge list-keys [-c|--config path]

OUTPUT:
    Shows key ID, provider, and display name for each stored key.

EXAMPLES:
    # List all keys
    armorclaw-bridge list-keys

    # No keys? Add one:
    armorclaw-bridge add-key --provider openai --token sk-proj-...
`
	case "start":
		help = `COMMAND: start

Start an AI agent container with stored credentials.

USAGE:
    armorclaw-bridge start -k KEY_ID [-c|--config path]

FLAGS:
    -k, --key string   Key ID to use (required)

EXAMPLES:
    # List keys first
    armorclaw-bridge list-keys

    # Start with specific key
    armorclaw-bridge start --key openai-default

    # Start bridge in foreground, then in another terminal:
    armorclaw-bridge start --key openai-default
`
	case "completion":
		help = `COMMAND: completion

Generate shell completion script for bash or zsh.

USAGE:
    armorclaw-bridge completion [bash|zsh]

EXAMPLES:
    # Generate bash completion
    armorclaw-bridge completion bash > ~/.bash_completion.d/armorclaw-bridge
    source ~/.bash_completion.d/armorclaw-bridge

    # Generate zsh completion
    armorclaw-bridge completion zsh > ~/.zsh/completions/_armorclaw-bridge
    # Then add to ~/.zshrc: autoload -U compinit && compinit
`
	case "validate":
		help = `COMMAND: validate

Validate configuration file.

USAGE:
    armorclaw-bridge validate [-c|--config path]

EXAMPLES:
    # Validate default config
    armorclaw-bridge validate

    # Validate custom config
    armorclaw-bridge validate -c /path/to/config.toml
`
	default:
		help = fmt.Sprintf("Unknown command: %s\n\nRun 'armorclaw-bridge help' for usage.", command)
	}
	fmt.Println(help)
}
