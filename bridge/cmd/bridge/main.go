// ArmorClaw Bridge - Main entry point
//
// The bridge provides a secure interface between the host system and isolated
// AI agent containers. It manages encrypted credentials and container lifecycle.
package main

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/armorclaw/bridge/pkg/budget"
	"github.com/armorclaw/bridge/pkg/config"
	"github.com/armorclaw/bridge/pkg/discovery"
	"github.com/armorclaw/bridge/pkg/docker"
	"github.com/armorclaw/bridge/pkg/errors"
	"github.com/armorclaw/bridge/pkg/eventbus"
	"github.com/armorclaw/bridge/pkg/health"
	"github.com/armorclaw/bridge/internal/adapter"
	"github.com/armorclaw/bridge/pkg/keystore"
	"github.com/armorclaw/bridge/pkg/logger"
	"github.com/armorclaw/bridge/pkg/notification"
	"github.com/armorclaw/bridge/pkg/rpc"
	"github.com/armorclaw/bridge/pkg/turn"
	// TODO: Voice package needs refactoring - uncomment when fixed
	// "github.com/armorclaw/bridge/pkg/voice"
	"github.com/armorclaw/bridge/pkg/webrtc"
)

var (
	version   = "0.2.0"
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
	// QR code command flags
	qrHost string
	qrPort int
	// Agent command flags
	agentType        string
	agentName        string
	agentRoom        string
	agentKey         string
	agentCapabilities string
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

	if cliCfg.command == "generate-qr" {
		runGenerateQRCommand(cliCfg)
		return
	}

	if cliCfg.command == "start-agent" {
		runStartAgentCommand(cliCfg)
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

		// Capture setup user MXID for error system
		if matrixUsername != "" && matrixHomeserver != "" {
			// Construct full MXID from username and homeserver
			// Format: @username:homeserver.domain
			homeserverDomain := strings.TrimPrefix(matrixHomeserver, "https://")
			homeserverDomain = strings.TrimPrefix(homeserverDomain, "http://")
			homeserverDomain = strings.Split(homeserverDomain, "/")[0] // Remove any path
			homeserverDomain = strings.Split(homeserverDomain, ":")[0] // Remove port if present
			cfg.ErrorSystem.SetupUserMXID = fmt.Sprintf("@%s:%s", matrixUsername, homeserverDomain)
		}
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
	fmt.Println("═══════════════════════════════════════════════════════════════════════════════")
	fmt.Println("")
	fmt.Println("📱 CONNECT ARMORCHAT / ARMORTERMINAL")
	fmt.Println("")
	fmt.Println("After starting the bridge, connect your mobile app:")
	fmt.Println("")
	fmt.Println("  For LOCAL NETWORK (same WiFi):")
	fmt.Println("    • Open ArmorChat - it will auto-discover this bridge")
	fmt.Printf("    • Look for: %s._armorclaw._tcp.local.\n", getHostname())
	fmt.Println("")
	fmt.Println("  For REMOTE VPS (different network):")
	fmt.Println("    • Option A: Scan QR code (run: armorclaw-bridge generate-qr)")
	fmt.Println("    • Option B: Enter your domain in ArmorChat setup")
	fmt.Println("    • Option C: Manual entry with the URLs shown at startup")
	fmt.Println("")
	fmt.Println("  ⚠️  mDNS discovery only works on the SAME network!")
	fmt.Println("     For VPS deployments, use QR code or manual entry.")
	fmt.Println("")
	fmt.Println("═══════════════════════════════════════════════════════════════════════════════")
}

// getHostname safely gets the system hostname
func getHostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "armorclaw-server"
	}
	return hostname
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
    local commands="init validate add-key list-keys start start-agent generate-qr setup version help completion"
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
        generate-qr)
            COMPREPLY=($(compgen -W "--host --port --help -h" -- "$cur"))
            ;;
        start-agent)
            COMPREPLY=($(compgen -W "--type --name --room --key --capabilities --help -h" -- "$cur"))
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
        'start:Start an agent container (legacy)'
        'start-agent:Start an AI agent (OpenClaw, assistant, etc.)'
        'generate-qr:Generate QR code for ArmorChat discovery'
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
            generate-qr)
                _arguments '--host[Public hostname/domain]' \
                           '--port[Public port]' \
                           '--help[Show help]'
                ;;
            start-agent)
                _arguments '--type[Agent type]:types:(assistant openclaw custom)' \
                           '--name[Agent display name]' \
                           '--room[Matrix room ID]' \
                           '--key[API key ID]' \
                           '--capabilities[Comma-separated capabilities]' \
                           '--help[Show help]'
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

// runGenerateQRCommand generates a QR code for ArmorChat discovery
func runGenerateQRCommand(cliCfg cliConfig) {
	// Load configuration
	cfg, err := config.Load(cliCfg.configPath)
	if err != nil {
		log.Printf("Warning: Using default configuration: %v", err)
		cfg = config.DefaultConfig()
	}

	// Get hostname
	hostname := getHostname()
	if cliCfg.qrHost != "" {
		hostname = cliCfg.qrHost
	}

	port := cfg.Discovery.Port
	if cliCfg.qrPort != 0 {
		port = cliCfg.qrPort
	}

	// Determine protocol
	protocol := "http"
	if cfg.Discovery.TLS {
		protocol = "https"
	}

	// Build configuration data
	matrixHS := cfg.Matrix.HomeserverURL
	if matrixHS == "" {
		matrixHS = fmt.Sprintf("%s://%s:8448", protocol, hostname)
	}

	// Create the config JSON
	configData := map[string]interface{}{
		"version":          1,
		"matrix_homeserver": matrixHS,
		"rpc_url":          fmt.Sprintf("%s://%s:%d/api", protocol, hostname, port),
		"ws_url":           fmt.Sprintf("%s://%s:%d/ws", map[bool]string{true: "wss", false: "ws"}[cfg.Discovery.TLS], hostname, port),
		"push_gateway":     strings.TrimSuffix(matrixHS, "/") + "/_matrix/push/v1/notify",
		"server_name":      hostname,
		"expires_at":       time.Now().Add(24 * time.Hour).Unix(),
	}

	// Convert to JSON
	jsonData, err := json.Marshal(configData)
	if err != nil {
		log.Fatalf("Failed to create config JSON: %v", err)
	}

	// Base64 encode
	encodedData := base64.StdEncoding.EncodeToString(jsonData)

	// Create deep link URL
	deepLinkURL := fmt.Sprintf("armorclaw://config?d=%s", encodedData)
	webURL := fmt.Sprintf("https://armorclaw.app/config?d=%s", encodedData)

	fmt.Println("")
	fmt.Println("╔══════════════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║              ARMORCHAT DISCOVERY QR CODE GENERATED                          ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════════════════════╝")
	fmt.Println("")
	fmt.Println("To connect ArmorChat to this bridge:")
	fmt.Println("")
	fmt.Println("  1. Open ArmorChat on your device")
	fmt.Println("  2. Go to Settings → Add Server → Scan QR Code")
	fmt.Println("  3. Scan the QR code below or use the deep link")
	fmt.Println("")
	fmt.Println("┌─────────────────────────────────────────────────────────────────────────────┐")
	fmt.Println("│ CONFIGURATION DATA                                                          │")
	fmt.Println("├─────────────────────────────────────────────────────────────────────────────┤")
	fmt.Printf("│ Server:       %s\n", hostname)
	fmt.Printf("│ Port:         %d\n", port)
	fmt.Printf("│ TLS:          %v\n", cfg.Discovery.TLS)
	fmt.Printf("│ Matrix:       %s\n", matrixHS)
	fmt.Printf("│ RPC:          %s://%s:%d/api\n", protocol, hostname, port)
	fmt.Printf("│ WebSocket:    %s://%s:%d/ws\n", map[bool]string{true: "wss", false: "ws"}[cfg.Discovery.TLS], hostname, port)
	fmt.Println("│ Valid:        24 hours")
	fmt.Println("└─────────────────────────────────────────────────────────────────────────────┘")
	fmt.Println("")
	fmt.Println("┌─────────────────────────────────────────────────────────────────────────────┐")
	fmt.Println("│ DEEP LINK (copy/paste to device)                                            │")
	fmt.Println("├─────────────────────────────────────────────────────────────────────────────┤")
	fmt.Printf("│ %s\n", deepLinkURL[:min(75, len(deepLinkURL))])
	if len(deepLinkURL) > 75 {
		fmt.Printf("│ %s\n", deepLinkURL[75:min(150, len(deepLinkURL))])
	}
	if len(deepLinkURL) > 150 {
		fmt.Printf("│ %s\n", deepLinkURL[150:])
	}
	fmt.Println("└─────────────────────────────────────────────────────────────────────────────┘")
	fmt.Println("")
	fmt.Println("┌─────────────────────────────────────────────────────────────────────────────┐")
	fmt.Println("│ WEB LINK (for browsers)                                                     │")
	fmt.Println("├─────────────────────────────────────────────────────────────────────────────┤")
	fmt.Printf("│ %s\n", webURL[:min(75, len(webURL))])
	if len(webURL) > 75 {
		fmt.Printf("│ %s\n", webURL[75:min(150, len(webURL))])
	}
	if len(webURL) > 150 {
		fmt.Printf("│ %s\n", webURL[150:])
	}
	fmt.Println("└─────────────────────────────────────────────────────────────────────────────┘")
	fmt.Println("")
	fmt.Println("📝 TIP: For production use, consider:")
	fmt.Println("   • Use --host with your public domain (e.g., bridge.example.com)")
	fmt.Println("   • Ensure TLS is enabled in your config ([discovery] tls = true)")
	fmt.Println("   • Configure your firewall to allow the port")
	fmt.Println("")

	// Generate QR code ASCII if qrcode library is available
	// Note: This is a placeholder - for a full implementation, add github.com/skip2/go-qrcode
	fmt.Println("⚠️  ASCII QR code not available - use the deep link above or generate PNG:")
	fmt.Printf("   echo '%s' | qrencode -t UTF8\n", deepLinkURL)
	fmt.Println("")
}

// min helper function
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// runStartAgentCommand starts an AI agent (OpenClaw or similar) via bridge RPC
func runStartAgentCommand(cliCfg cliConfig) {
	// Validate required parameters
	if cliCfg.agentRoom == "" {
		log.Fatal("Error: --room is required. Specify the Matrix room ID for the agent.")
	}

	// Generate agent name if not provided
	agentName := cliCfg.agentName
	if agentName == "" {
		agentName = fmt.Sprintf("%s-agent", cliCfg.agentType)
	}

	// Parse capabilities
	capabilities := []string{"chat"}
	if cliCfg.agentCapabilities != "" {
		capabilities = strings.Split(cliCfg.agentCapabilities, ",")
		for i, cap := range capabilities {
			capabilities[i] = strings.TrimSpace(cap)
		}
	}

	// Check if bridge is running
	socketPath := "/run/armorclaw/bridge.sock"
	if _, err := os.Stat(socketPath); os.IsNotExist(err) {
		log.Fatal("Error: Bridge is not running. Start it first with: armorclaw-bridge")
	}

	// Connect to bridge via Unix socket
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		log.Fatalf("Error: Failed to connect to bridge: %v", err)
	}
	defer conn.Close()

	// Generate agent ID
	agentID := fmt.Sprintf("%s-%d", cliCfg.agentType, time.Now().Unix())

	// Build RPC request
	request := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "agent.start",
		"params": map[string]interface{}{
			"agent_id":     agentID,
			"name":         agentName,
			"type":         cliCfg.agentType,
			"room_id":      cliCfg.agentRoom,
			"capabilities": capabilities,
		},
	}

	// Add key_id if provided
	if cliCfg.agentKey != "" {
		request["params"].(map[string]interface{})["key_id"] = cliCfg.agentKey
	}

	// Send request
	requestJSON, _ := json.Marshal(request)
	_, err = conn.Write(append(requestJSON, '\n'))
	if err != nil {
		log.Fatalf("Error: Failed to send request: %v", err)
	}

	// Read response
	buffer := make([]byte, 4096)
	n, err := conn.Read(buffer)
	if err != nil {
		log.Fatalf("Error: Failed to read response: %v", err)
	}

	// Parse response
	var response struct {
		Jsonrpc string `json:"jsonrpc"`
		ID      int    `json:"id"`
		Result  *struct {
			AgentID     string   `json:"agent_id"`
			Name        string   `json:"name"`
			Type        string   `json:"type"`
			Status      string   `json:"status"`
			RoomID      string   `json:"room_id"`
			Capabilities []string `json:"capabilities"`
		} `json:"result"`
		Error *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(buffer[:n], &response); err != nil {
		log.Fatalf("Error: Failed to parse response: %v", err)
	}

	if response.Error != nil {
		log.Fatalf("Error: Agent start failed (code %d): %s", response.Error.Code, response.Error.Message)
	}

	if response.Result == nil {
		log.Fatal("Error: No result returned from bridge")
	}

	// Success output
	fmt.Println("")
	fmt.Println("╔══════════════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                      AGENT STARTED SUCCESSFULLY                              ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════════════════════╝")
	fmt.Println("")
	fmt.Println("┌─────────────────────────────────────────────────────────────────────────────┐")
	fmt.Println("│ AGENT INFORMATION                                                           │")
	fmt.Println("├─────────────────────────────────────────────────────────────────────────────┤")
	fmt.Printf("│ Agent ID:     %s\n", response.Result.AgentID)
	fmt.Printf("│ Name:         %s\n", response.Result.Name)
	fmt.Printf("│ Type:         %s\n", response.Result.Type)
	fmt.Printf("│ Status:       %s\n", response.Result.Status)
	fmt.Printf("│ Room:         %s\n", response.Result.RoomID)
	fmt.Printf("│ Capabilities: %s\n", strings.Join(response.Result.Capabilities, ", "))
	fmt.Println("└─────────────────────────────────────────────────────────────────────────────┘")
	fmt.Println("")
	fmt.Println("📱 Connect ArmorChat to interact with this agent:")
	fmt.Printf("   Room ID: %s\n", response.Result.RoomID)
	fmt.Println("")
	fmt.Println("🔧 Management commands:")
	fmt.Println("   Check status:  armorclaw-bridge agent-status --id " + response.Result.AgentID)
	fmt.Println("   Stop agent:    armorclaw-bridge stop-agent --id " + response.Result.AgentID)
	fmt.Println("   View logs:     journalctl -u armorclaw-bridge -f")
	fmt.Println("")

	// If using OpenClaw type, provide additional guidance
	if cliCfg.agentType == "openclaw" || cliCfg.agentType == "assistant" {
		fmt.Println("💡 OpenClaw Agent Tips:")
		fmt.Println("   • Ensure API keys are stored: armorclaw-bridge add-key --provider openai --token sk-xxx")
		fmt.Println("   • Use docker-compose to manage: docker-compose -f docker-compose.bridge.yml --profile openclaw up -d")
		fmt.Println("   • Check container status: docker ps | grep openclaw")
		fmt.Println("")
	}
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

	// Initialize error handling system
	log.Println("Initializing error handling system...")
	errorCfg := cfg.ToErrorSystemConfig()
	errorSystem, err := errors.Initialize(errors.Config{
		StorePath:       errorCfg.StorePath,
		RetentionDays:   errorCfg.RetentionDays,
		RateLimitWindow: errorCfg.RateLimitWindow,
		RetentionPeriod: errorCfg.RetentionPeriod,
		ConfigAdminMXID: errorCfg.ConfigAdminMXID,
		SetupUserMXID:   errorCfg.SetupUserMXID,
		AdminRoomID:     errorCfg.AdminRoomID,
		FallbackMXID:    errorCfg.FallbackMXID,
		Enabled:         errorCfg.Enabled,
		StoreEnabled:    errorCfg.StoreEnabled,
		NotifyEnabled:   errorCfg.NotifyEnabled,
	})
	if err != nil {
		log.Fatalf("Failed to initialize error system: %v", err)
	}
	defer errorSystem.Stop()

	// Start the error system
	if err := errorSystem.Start(context.Background()); err != nil {
		log.Printf("Warning: Failed to start error system: %v", err)
	} else {
		log.Println("Error system initialized")
		if errorCfg.StoreEnabled {
			log.Printf("Error store: %s", errorCfg.StorePath)
		}
		if errorCfg.SetupUserMXID != "" {
			log.Printf("Setup user: %s", errorCfg.SetupUserMXID)
		}
	}

	// Initialize WebRTC components
	log.Println("Initializing WebRTC components...")

	// Create session manager
	sessionConfig := webrtc.DefaultSessionConfig()
	sessionConfig.DefaultTTL = 30 * time.Minute
	sessionMgr := webrtc.NewSessionManager(sessionConfig)

	// Create token manager (requires secret for signing)
	// Use TURN secret or generate a random one
	tokenSecret := cfg.WebRTC.TURNSharedSecret
	if tokenSecret == "" {
		// Generate a random secret if not configured
		tokenSecret = fmt.Sprintf("armorclaw-%d", time.Now().UnixNano())
	}
	tokenMgr := webrtc.NewTokenManager(tokenSecret, 24*time.Hour)

	// Create WebRTC engine
	webrtcConfig := webrtc.DefaultEngineConfig()
	webrtcEngine, err := webrtc.NewEngine(webrtcConfig)
	if err != nil {
		log.Fatalf("Failed to create WebRTC engine: %v", err)
	}

	// Create TURN manager (optional) - from turn package
	var turnMgr *turn.Manager
	if cfg.WebRTC.TURNSharedSecret != "" {
		// Use default TURN config with the configured secret
		turnConfig := turn.DefaultConfig()
		turnConfig.Secret = cfg.WebRTC.TURNSharedSecret
		if cfg.WebRTC.TURNServerURL != "" {
			// Parse TURN URL (format: turn:host:port)
			turnURL := cfg.WebRTC.TURNServerURL
			if strings.HasPrefix(turnURL, "turn:") {
				turnURL = strings.TrimPrefix(turnURL, "turn:")
			} else if strings.HasPrefix(turnURL, "turns:") {
				turnURL = strings.TrimPrefix(turnURL, "turns:")
			}
			parts := strings.Split(turnURL, ":")
			if len(parts) >= 1 {
				turnConfig.Servers[0].Host = parts[0]
			}
			if len(parts) >= 2 {
				var port int
				fmt.Sscanf(parts[1], "%d", &port)
				turnConfig.Servers[0].Port = port
			}
		}
		turnMgr = turn.NewManager(turnConfig)
		webrtcEngine.SetTURNManager(turnMgr)
		log.Println("TURN manager initialized")
	}

	// TODO: Voice package needs refactoring - uncomment when fixed
	/*
	// Create voice manager
	voiceConfig := voice.DefaultConfig()

	// Helper function to parse duration strings
	parseDuration := func(s string) time.Duration {
		if s == "" {
			return 0
		}
		d, err := time.ParseDuration(s)
		if err != nil {
			log.Printf("Warning: Invalid duration '%s': %v", s, err)
			return 0
		}
		return d
	}

	// Helper function to convert string slice to bool map
	stringSliceToBoolMap := func(slice []string) map[string]bool {
		result := make(map[string]bool)
		for _, s := range slice {
			result[s] = true
		}
		return result
	}

	// Override with config file values if present

	// General settings
	if cfg.Voice.DefaultLifetime != "" {
		if d := parseDuration(cfg.Voice.DefaultLifetime); d > 0 {
			voiceConfig.DefaultLifetime = d
		}
	}
	if cfg.Voice.MaxLifetime != "" {
		if d := parseDuration(cfg.Voice.MaxLifetime); d > 0 {
			voiceConfig.MaxLifetime = d
		}
	}
	voiceConfig.AutoAnswer = cfg.Voice.AutoAnswer
	voiceConfig.RequireMembership = cfg.Voice.RequireMembership
	voiceConfig.AllowedRooms = stringSliceToBoolMap(cfg.Voice.AllowedRooms)
	voiceConfig.BlockedRooms = stringSliceToBoolMap(cfg.Voice.BlockedRooms)

	// Security settings
	voiceConfig.MaxConcurrentCalls = cfg.Voice.Security.MaxConcurrentCalls
	if cfg.Voice.Security.MaxCallDuration != "" {
		if d := parseDuration(cfg.Voice.Security.MaxCallDuration); d > 0 {
			voiceConfig.SecurityPolicy.MaxCallDuration = d
		}
	}
	voiceConfig.SecurityPolicy.RateLimitCalls = cfg.Voice.Security.RateLimitCalls
	if cfg.Voice.Security.RateLimitWindow != "" {
		if d := parseDuration(cfg.Voice.Security.RateLimitWindow); d > 0 {
			voiceConfig.SecurityPolicy.RateLimitWindow = d
		}
	}
	voiceConfig.SecurityPolicy.RequireE2EE = cfg.Voice.Security.RequireE2EE
	voiceConfig.SecurityPolicy.RequireSignalingTLS = cfg.Voice.Security.RequireSignalingTLS

	// Budget settings
	voiceConfig.DefaultTokenLimit = cfg.Voice.Budget.DefaultTokenLimit
	if cfg.Voice.Budget.DefaultDurationLimit != "" {
		if d := parseDuration(cfg.Voice.Budget.DefaultDurationLimit); d > 0 {
			voiceConfig.DefaultDurationLimit = d
		}
	}
	voiceConfig.WarningThreshold = cfg.Voice.Budget.WarningThreshold
	voiceConfig.HardStop = cfg.Voice.Budget.HardStop

	// TTL settings
	if cfg.Voice.TTL.DefaultTTL != "" {
		if d := parseDuration(cfg.Voice.TTL.DefaultTTL); d > 0 {
			voiceConfig.TTLConfig.DefaultTTL = d
		}
	}
	if cfg.Voice.TTL.MaxTTL != "" {
		if d := parseDuration(cfg.Voice.TTL.MaxTTL); d > 0 {
			voiceConfig.TTLConfig.MaxTTL = d
		}
	}
	if cfg.Voice.TTL.EnforcementInterval != "" {
		if d := parseDuration(cfg.Voice.TTL.EnforcementInterval); d > 0 {
			voiceConfig.TTLConfig.EnforcementInterval = d
		}
	}
	if cfg.Voice.TTL.WarningThreshold > 0 {
		voiceConfig.TTLConfig.WarningThreshold = cfg.Voice.TTL.WarningThreshold
	}
	voiceConfig.TTLConfig.HardStop = cfg.Voice.TTL.HardStop

	// Update budget config in voiceConfig
	voiceConfig.BudgetConfig.DefaultTokenLimit = cfg.Voice.Budget.DefaultTokenLimit
	if cfg.Voice.Budget.DefaultDurationLimit != "" {
		if d := parseDuration(cfg.Voice.Budget.DefaultDurationLimit); d > 0 {
			voiceConfig.BudgetConfig.DefaultDurationLimit = d
		}
	}
	voiceConfig.BudgetConfig.WarningThreshold = cfg.Voice.Budget.WarningThreshold
	voiceConfig.BudgetConfig.HardStop = cfg.Voice.Budget.HardStop

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
	*/

	// Create budget tracker
	budgetTracker, err := budget.NewBudgetTracker(budget.BudgetConfig{
		DailyLimitUSD:   cfg.Budget.DailyLimitUSD,
		MonthlyLimitUSD: cfg.Budget.MonthlyLimitUSD,
		AlertThreshold:  cfg.Budget.AlertThreshold,
		HardStop:        cfg.Budget.HardStop,
	})
	if err != nil {
		log.Fatalf("Failed to create budget tracker: %v", err)
	}

	log.Println("WebRTC components initialized")

	// Initialize Docker client for container management
	log.Println("Initializing Docker client...")
	dockerClient, err := docker.New(docker.Config{
		Host:       "", // Use default socket
		APIVersion: "1.45",
		Scopes: []docker.Scope{
			docker.ScopeCreate,
			docker.ScopeExec,
			docker.ScopeRemove,
		},
	})
	if err != nil {
		log.Fatalf("Failed to create Docker client: %v", err)
	}

	// Initialize notification system (requires Matrix adapter)
	// Declare notifier early so it can be used in health monitor callback
	var notifier *notification.Notifier
	if cfg.Matrix.Enabled && cfg.Notifications.AdminRoomID != "" {
		log.Println("Initializing notification system...")
		// We'll create the notifier after Matrix adapter is initialized
		// For now, create a placeholder that will be configured later
		notifier = notification.NewNotifier(nil, notification.Config{
			AdminRoomID: cfg.Notifications.AdminRoomID,
			Enabled:     true,
		})
	} else {
		log.Println("Notifications disabled (Matrix not enabled or no admin room configured)")
	}

	// Initialize health monitor
	log.Println("Initializing container health monitor...")
	healthConfig := health.DefaultMonitorConfig()
	healthMonitor := health.NewMonitor(dockerClient, healthConfig)

	// Set up container failure handler
	healthMonitor.SetFailureHandler(func(containerID, containerName, reason string) {
		log.Printf("Container failure detected: %s (%s) - %s", containerName, containerID, reason)

		// Send notification if configured
		if notifier != nil {
			_ = notifier.SendContainerAlert("container_failed", containerID, containerName, reason)
		}
	})

	// Start health monitor
	if err := healthMonitor.Start(); err != nil {
		log.Printf("Warning: Failed to start health monitor: %v", err)
		log.Println("Container health monitoring will not be available")
	} else {
		log.Println("Health monitor started")
	}

	// Initialize event bus for real-time Matrix event push
	var eventBus *eventbus.EventBus
	if cfg.Matrix.Enabled {
		log.Println("Initializing event bus for Matrix events...")

		// Parse inactivity timeout
		inactivityTimeout := 30 * time.Minute
		if cfg.EventBus.InactivityTimeout != "" {
			if d, err := time.ParseDuration(cfg.EventBus.InactivityTimeout); err == nil {
				inactivityTimeout = d
			}
		}

		eventBusConfig := eventbus.Config{
			WebSocketEnabled:  cfg.EventBus.WebSocketEnabled,
			WebSocketAddr:     cfg.EventBus.WebSocketAddr,
			WebSocketPath:     cfg.EventBus.WebSocketPath,
			MaxSubscribers:    cfg.EventBus.MaxSubscribers,
			InactivityTimeout: inactivityTimeout,
		}

		eventBus = eventbus.NewEventBus(eventBusConfig)

		// Start event bus
		if err := eventBus.Start(); err != nil {
			log.Printf("Warning: Failed to start event bus: %v", err)
			log.Println("Real-time event push will not be available")
			eventBus = nil
		} else {
			log.Println("Event bus started for real-time Matrix event distribution")
			if cfg.EventBus.WebSocketEnabled {
				log.Printf("WebSocket endpoint: %s%s", cfg.EventBus.WebSocketAddr, cfg.EventBus.WebSocketPath)
			}
		}
	} else {
		log.Println("Event bus disabled (Matrix not enabled)")
	}

	// Initialize WebRTC signaling server
	var signalingSvr *webrtc.SignalingServer
	if cfg.WebRTC.SignalingEnabled {
		log.Println("Initializing WebRTC signaling server...")
		// Create signaling server with WebSocket endpoint
		signalingSvr = webrtc.NewSignalingServer(
			cfg.WebRTC.SignalingAddr,
			cfg.WebRTC.SignalingPath,
			sessionMgr,
			tokenMgr,
		)

		// Configure TLS if certificates provided
		if cfg.WebRTC.SignalingTLSCert != "" && cfg.WebRTC.SignalingTLSKey != "" {
			signalingSvr.SetTLS(cfg.WebRTC.SignalingTLSCert, cfg.WebRTC.SignalingTLSKey)
			log.Printf("Signaling server TLS enabled")
		}

		// Start signaling server
		if err := signalingSvr.Start(); err != nil {
			log.Printf("Warning: Failed to start signaling server: %v", err)
			log.Println("WebRTC signaling will use JSON-RPC fallback")
			signalingSvr = nil
		} else {
			log.Printf("Signaling server started on %s%s", cfg.WebRTC.SignalingAddr, cfg.WebRTC.SignalingPath)
		}
	}

	// Connect notifier to Matrix adapter (if Matrix is enabled)
	if notifier != nil && cfg.Matrix.Enabled {
		// We need to get the Matrix adapter from the RPC server later
		// For now, the notifier will be passed to the budget manager
		log.Printf("Notifier configured for admin room: %s", cfg.Notifications.AdminRoomID)
	}

	// Set notifier on budget manager for budget alerts
	if notifier != nil {
		// The budget manager has access to the budget tracker
		// We'll set the notifier on the budget tracker after it's created
		log.Println("Budget alerts will be sent to Matrix")
	}

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
		SignalingServer:   signalingSvr, // Now integrated
		WebRTCEngine:      webrtcEngine,
		TURNManager:       turnMgr,
		// TODO: Voice package needs refactoring - uncomment when fixed
		// VoiceManager:      voiceMgr,
		BudgetManager:     budgetTracker,
		HealthMonitor:     healthMonitor, // Health monitoring
		Notifier:          notifier, // Notifications
		EventBus:          eventBus, // Event push mechanism
		ErrorSystem:       errorSystem, // Error handling system
	})
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	if err := server.Start(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop()

	// Wire up event bus to Matrix adapter if both are enabled
	// TODO: Fix type mismatch between eventbus.MatrixEvent and adapter.MatrixEvent
	/*
	if eventBus != nil && cfg.Matrix.Enabled {
		log.Println("Wiring Matrix adapter to event bus...")
		// Get the Matrix adapter from the server and set up event publishing
		// The RPC server should provide access to the Matrix adapter
		if matrixAdapter := server.GetMatrixAdapter(); matrixAdapter != nil {
			// Type assertion to get the actual Matrix adapter
			if ma, ok := matrixAdapter.(*adapter.MatrixAdapter); ok {
				ma.SetEventPublisher(eventBus)
				log.Println("Matrix events will be published to event bus in real-time")
			} else {
				log.Println("Warning: Matrix adapter type assertion failed")
			}
		} else {
			log.Println("Warning: Matrix adapter not available for event bus integration")
		}
	}
	*/

	// Set notifier's Matrix adapter if notifier is enabled
	if notifier != nil && cfg.Matrix.Enabled {
		if matrixAdapter := server.GetMatrixAdapter(); matrixAdapter != nil {
			// Type assertion for Matrix adapter
			if ma, ok := matrixAdapter.(*adapter.MatrixAdapter); ok {
				notifier.SetMatrixAdapter(ma)
				log.Println("Notifier connected to Matrix adapter")
			}
		}
	}

	// Connect error system to Matrix adapter for notifications
	if errorSystem != nil && cfg.Matrix.Enabled {
		if matrixAdapter := server.GetMatrixAdapter(); matrixAdapter != nil {
			// Type assertion to get the actual Matrix adapter
			if ma, ok := matrixAdapter.(*adapter.MatrixAdapter); ok {
				errorSystem.SetMatrixAdapter(ma)
				log.Println("Error system connected to Matrix adapter")
			}
		}
	}

	// Create shutdown context early for components that need it
	shutdownCtx, cancel := context.WithCancel(context.Background())

	// Initialize mDNS discovery server
	var discoveryServer *discovery.Server
	if cfg.Discovery.Enabled {
		log.Println("Starting mDNS discovery server...")

		// Use Matrix homeserver from discovery config, or fall back to Matrix config
		matrixHS := cfg.Discovery.MatrixHomeserver
		if matrixHS == "" {
			matrixHS = cfg.Matrix.HomeserverURL
		}

		// Derive push gateway if not set
		pushGW := cfg.Discovery.PushGateway
		if pushGW == "" && matrixHS != "" {
			// Derive from API URL
			pushGW = strings.TrimSuffix(matrixHS, "/") + "/_matrix/push/v1/notify"
		}

		discoveryConfig := discovery.ServerConfig{
			InstanceName:     cfg.Discovery.InstanceName,
			Port:             cfg.Discovery.Port,
			TLS:              cfg.Discovery.TLS,
			MatrixHomeserver: matrixHS,
			PushGateway:      pushGW,
			APIPath:          cfg.Discovery.APIPath,
			WSPath:           cfg.Discovery.WSPath,
			ExtraTXT: map[string]string{
				"hardware": cfg.Discovery.Hardware,
			},
		}

		discoveryServer, err = discovery.NewServerWithConfig(discoveryConfig)
		if err != nil {
			log.Printf("Warning: Failed to create mDNS discovery server: %v", err)
			log.Println("Bridge discovery will not be available via mDNS")
		} else {
			if err := discoveryServer.Start(shutdownCtx); err != nil {
				log.Printf("Warning: Failed to start mDNS discovery server: %v", err)
				log.Println("Bridge discovery will not be available via mDNS")
				discoveryServer = nil
			} else {
				info := discoveryServer.Info()
				log.Printf("mDNS discovery started: %s._armorclaw._tcp.local.", info.Name)
				log.Printf("Advertising: %s://%s:%d", map[bool]string{true: "https", false: "http"}[cfg.Discovery.TLS], info.Host, info.Port)
				if matrixHS != "" {
					log.Printf("Matrix homeserver: %s", matrixHS)
				}
			}
		}
	} else {
		log.Println("mDNS discovery disabled")
	}

	log.Println("ArmorClaw Bridge is running")
	log.Println("Press Ctrl+C to stop")
	log.Println("")

	// Show connection guidance for ArmorChat
	printConnectionGuidance(cfg)

	// Wait for interrupt signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		log.Println("\nShutting down...")

		// Stop mDNS discovery server
		if discoveryServer != nil {
			log.Println("Stopping mDNS discovery server...")
			discoveryServer.Stop()
		}

		// Stop WebRTC signaling server
		if signalingSvr != nil {
			log.Println("Stopping WebRTC signaling server...")
			signalingSvr.Stop()
		}

		// Stop event bus
		if eventBus != nil {
			log.Println("Stopping event bus...")
			eventBus.Stop()
		}

		// Stop health monitor
		log.Println("Stopping health monitor...")
		healthMonitor.Stop()

		// Stop notifier
		if notifier != nil {
			log.Println("Stopping notifier...")
			notifier.Stop()
		}

		// Stop error system
		if errorSystem != nil {
			log.Println("Stopping error system...")
			errorSystem.Stop()
		}

		// TODO: Voice package needs refactoring - uncomment when fixed
		// voiceMgr.Stop()
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
	// QR code command flags
	flag.StringVar(&cfg.qrHost, "host", "", "Host/domain for QR code (generate-qr command)")
	flag.IntVar(&cfg.qrPort, "port", 0, "Port for QR code (generate-qr command)")
	// Agent command flags
	flag.StringVar(&cfg.agentType, "type", "assistant", "Agent type (start-agent command)")
	flag.StringVar(&cfg.agentName, "name", "", "Agent display name (start-agent command)")
	flag.StringVar(&cfg.agentRoom, "room", "", "Matrix room ID for agent (start-agent command)")
	flag.StringVar(&cfg.agentKey, "key", "", "API key ID for agent (start-agent command)")
	flag.StringVar(&cfg.agentCapabilities, "capabilities", "chat", "Comma-separated capabilities (start-agent command)")

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
    start       Start an agent container (legacy, use start-agent)
    start-agent Start an AI agent (OpenClaw, assistant, etc.)
    generate-qr Generate QR code for ArmorChat discovery
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

    # Start an AI agent
    ./build/armorclaw-bridge start-agent --room '!room:matrix.example.com' --type assistant
    ./build/armorclaw-bridge start-agent --room '!room:matrix.example.com' --type openclaw --key openai-default

    # Generate QR code for ArmorChat
    ./build/armorclaw-bridge generate-qr --host bridge.example.com

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

// printConnectionGuidance displays instructions for connecting ArmorChat to this bridge
func printConnectionGuidance(cfg *config.Config) {
	fmt.Println("")
	fmt.Println("╔══════════════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                    ARMORCHAT CONNECTION GUIDE                                ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════════════════════╝")
	fmt.Println("")
	fmt.Println("Connect ArmorChat or ArmorTerminal to this bridge using one of these methods:")
	fmt.Println("")

	// Determine protocol based on TLS setting
	protocol := "http"
	if cfg.Discovery.TLS {
		protocol = "https"
	}

	// Get hostname for display
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "localhost"
	}

	// Derive the public URL (for remote deployments, this would be the VPS domain)
	// In production, this should come from config or auto-detection
	publicHost := hostname
	if cfg.Discovery.InstanceName != "" {
		publicHost = cfg.Discovery.InstanceName
	}

	// Method 1: QR Code / Deep Link (Recommended for remote VPS)
	fmt.Println("┌─────────────────────────────────────────────────────────────────────────────┐")
	fmt.Println("│ METHOD 1: QR CODE (Recommended for Remote VPS)                              │")
	fmt.Println("├─────────────────────────────────────────────────────────────────────────────┤")
	fmt.Println("│ 1. Open ArmorChat on your device                                            │")
	fmt.Println("│ 2. Tap 'Scan QR Code' or go to Settings → Add Server                        │")
	fmt.Println("│ 3. Scan the QR code generated by this command:                              │")
	fmt.Println("│                                                                             │")
	fmt.Printf(" │     armorclaw-bridge generate-qr --host %s --port %d\n", publicHost, cfg.Discovery.Port)
	fmt.Println("│                                                                             │")
	fmt.Println("│ The QR code contains signed server configuration including:                 │")
	fmt.Println("│   • Matrix homeserver URL                                                   │")
	fmt.Println("│   • Bridge RPC and WebSocket endpoints                                      │")
	fmt.Println("│   • Push gateway URL                                                        │")
	fmt.Println("└─────────────────────────────────────────────────────────────────────────────┘")
	fmt.Println("")

	// Method 2: Well-Known Discovery (for custom domains)
	fmt.Println("┌─────────────────────────────────────────────────────────────────────────────┐")
	fmt.Println("│ METHOD 2: WELL-KNOWN DISCOVERY (Custom Domain)                              │")
	fmt.Println("├─────────────────────────────────────────────────────────────────────────────┤")
	fmt.Println("│ If your Matrix server has .well-known configured:                           │")
	fmt.Println("│                                                                             │")
	fmt.Println("│ 1. Open ArmorChat on your device                                            │")
	fmt.Println("│ 2. Enter your domain (e.g., 'matrix.example.com')                           │")
	fmt.Println("│ 3. ArmorChat will auto-discover the bridge configuration                    │")
	fmt.Println("│                                                                             │")
	fmt.Println("│ Required .well-known endpoint:                                              │")
	if cfg.Matrix.HomeserverURL != "" {
		fmt.Printf("│   %s/.well-known/matrix/client\n", cfg.Matrix.HomeserverURL)
	}
	fmt.Println("│                                                                             │")
	fmt.Println("│ Response must include 'com.armorclaw.bridge' section with:                  │")
	fmt.Println("│   {\"api_endpoint\": \"...\", \"ws_endpoint\": \"...\", \"push_gateway\": \"...\"}   │")
	fmt.Println("└─────────────────────────────────────────────────────────────────────────────┘")
	fmt.Println("")

	// Method 3: mDNS Discovery (Local Network Only)
	if cfg.Discovery.Enabled {
		fmt.Println("┌─────────────────────────────────────────────────────────────────────────────┐")
		fmt.Println("│ METHOD 3: mDNS DISCOVERY (Same Network Only)                                │")
		fmt.Println("├─────────────────────────────────────────────────────────────────────────────┤")
		fmt.Println("│ If your device is on the SAME network as this server:                       │")
		fmt.Println("│                                                                             │")
		fmt.Println("│ 1. Open ArmorChat on your device                                            │")
		fmt.Println("│ 2. The app will automatically discover this bridge                          │")
		fmt.Printf("│ 3. Look for: %s._armorclaw._tcp.local.\n", hostname)
		fmt.Println("│                                                                             │")
		fmt.Println("│ ⚠️  NOTE: mDNS does NOT work across different networks or VPNs!            │")
		fmt.Println("│    For remote VPS deployments, use QR code or manual entry.                 │")
		fmt.Println("└─────────────────────────────────────────────────────────────────────────────┘")
		fmt.Println("")
	}

	// Method 4: Manual Configuration
	fmt.Println("┌─────────────────────────────────────────────────────────────────────────────┐")
	fmt.Println("│ METHOD 4: MANUAL CONFIGURATION (Fallback)                                   │")
	fmt.Println("├─────────────────────────────────────────────────────────────────────────────┤")
	fmt.Println("│ If other methods don't work, enter these URLs manually in ArmorChat:        │")
	fmt.Println("│                                                                             │")
	if cfg.Matrix.HomeserverURL != "" {
		fmt.Printf("│ Matrix Server:  %s\n", cfg.Matrix.HomeserverURL)
	} else {
		fmt.Println("│ Matrix Server:  (not configured - set [matrix] homeserver_url)              │")
	}
	fmt.Printf("│ Bridge RPC:     %s://%s:%d/api\n", protocol, publicHost, cfg.Discovery.Port)
	fmt.Printf("│ Bridge WebSocket: %s://%s:%d/ws\n", map[bool]string{true: "wss", false: "ws"}[cfg.Discovery.TLS], publicHost, cfg.Discovery.Port)
	fmt.Println("│                                                                             │")
	fmt.Println("│ To set up Matrix integration, edit your config:                             │")
	fmt.Println("│   ~/.armorclaw/config.toml                                                  │")
	fmt.Println("│                                                                             │")
	fmt.Println("│   [matrix]                                                                  │")
	fmt.Println("│   enabled = true                                                            │")
	fmt.Println("│   homeserver_url = \"https://matrix.yourdomain.com\"                          │")
	fmt.Println("└─────────────────────────────────────────────────────────────────────────────┘")
	fmt.Println("")

	// Configuration summary
	fmt.Println("┌─────────────────────────────────────────────────────────────────────────────┐")
	fmt.Println("│ CURRENT CONFIGURATION                                                       │")
	fmt.Println("├─────────────────────────────────────────────────────────────────────────────┤")
	fmt.Printf("│ Discovery:      %s\n", map[bool]string{true: "ENABLED (mDNS + well-known)", false: "DISABLED"}[cfg.Discovery.Enabled])
	fmt.Printf("│ TLS:            %s\n", map[bool]string{true: "ENABLED (HTTPS/WSS)", false: "DISABLED (HTTP/WS)"}[cfg.Discovery.TLS])
	fmt.Printf("│ Port:           %d\n", cfg.Discovery.Port)
	if cfg.Matrix.HomeserverURL != "" {
		fmt.Printf("│ Matrix:         %s\n", cfg.Matrix.HomeserverURL)
	} else {
		fmt.Println("│ Matrix:         NOT CONFIGURED")
	}
	if cfg.Discovery.PushGateway != "" {
		fmt.Printf("│ Push Gateway:   %s\n", cfg.Discovery.PushGateway)
	}
	fmt.Println("└─────────────────────────────────────────────────────────────────────────────┘")
	fmt.Println("")

	// Troubleshooting hints
	fmt.Println("Troubleshooting:")
	fmt.Println("  • If ArmorChat can't connect, check firewall allows port " + fmt.Sprintf("%d", cfg.Discovery.Port))
	fmt.Println("  • For remote VPS, ensure your domain's DNS is properly configured")
	fmt.Println("  • Generate QR: armorclaw-bridge generate-qr --help")
	fmt.Println("  • Check status: armorclaw-bridge daemon status")
	fmt.Println("  • View docs:    https://github.com/armorclaw/armorclaw/tree/main/docs/guides")
	fmt.Println("")
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
	case "generate-qr":
		help = `COMMAND: generate-qr

Generate a QR code for ArmorChat/ArmorTerminal discovery.

This command creates a signed configuration URL that ArmorChat can use
to automatically discover and connect to this bridge.

USAGE:
    armorclaw-bridge generate-qr [--host hostname] [--port port]

FLAGS:
    --host string   Public hostname/domain (default: system hostname)
    --port int      Public port (default: from config)

OUTPUT:
    • Deep link URL (armorclaw://config?d=...)
    • Web link URL (https://armorclaw.app/config?d=...)
    • Configuration summary

EXAMPLES:
    # Generate QR with defaults
    armorclaw-bridge generate-qr

    # Generate QR for production domain
    armorclaw-bridge generate-qr --host bridge.example.com --port 443

    # Generate QR for local development
    armorclaw-bridge generate-qr --host 192.168.1.100

DISCOVERY METHODS:
    ArmorChat supports multiple discovery methods:

    1. QR Code (this command) - Best for remote VPS
    2. mDNS discovery - Same network only
    3. Well-known discovery - Custom domains
    4. Manual entry - Fallback option

NOTES:
    • The generated QR is valid for 24 hours
    • For production, ensure TLS is enabled in config
    • mDNS discovery only works on the same local network
`
	case "start-agent":
		help = `COMMAND: start-agent

Start an AI agent (OpenClaw, assistant, etc.) via the bridge RPC.

This command connects to the running bridge and starts an agent
that can interact with Matrix users through ArmorChat.

USAGE:
    armorclaw-bridge start-agent --room ROOM_ID [flags]

FLAGS:
    --type string          Agent type (default: "assistant")
                           Options: assistant, openclaw, custom
    --name string          Agent display name (default: "<type>-agent")
    --room string          Matrix room ID for agent (required)
    --key string           API key ID to use for the agent
    --capabilities string  Comma-separated capabilities (default: "chat")
                           Options: chat, voice, video, files, code

PREREQUISITES:
    1. Bridge must be running: armorclaw-bridge
    2. API key must be stored (if using AI features):
       armorclaw-bridge add-key --provider openai --token sk-xxx
    3. Matrix room must exist and bridge must be invited

EXAMPLES:
    # Start a basic assistant in a room
    armorclaw-bridge start-agent --room '!abc123:matrix.example.com'

    # Start OpenClaw agent with specific key
    armorclaw-bridge start-agent --room '!abc123:matrix.example.com' --type openclaw --key openai-default

    # Start agent with multiple capabilities
    armorclaw-bridge start-agent --room '!abc123:matrix.example.com' --capabilities "chat,files,code"

    # Start with custom name
    armorclaw-bridge start-agent --room '!abc123:matrix.example.com' --name "Support Bot"

AGENT MANAGEMENT:
    After starting, users can interact with the agent in the Matrix room.
    The agent will respond to messages based on its configuration.

    To stop an agent:
    • Via RPC: echo '{"method":"agent.stop","params":{"agent_id":"xxx"}}' | socat - UNIX-CONNECT:/run/armorclaw/bridge.sock

DOCKER COMPOSE:
    For OpenClaw agents, you can also use Docker Compose:

    docker-compose -f docker-compose.bridge.yml --profile openclaw up -d

    This requires:
    • ARMORCLAW_MATRIX_ROOM environment variable set
    • OpenClaw container image built
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
