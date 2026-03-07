package main

import (
	"bufio"
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
)

const (
	conduitConfigPath = "/etc/armorclaw/conduit.toml"
	initFlagPath      = "/var/lib/armorclaw/.bootstrapped"
	logFilePath       = "/var/log/armorclaw/bootstrap.log"
	conduitURL        = "http://localhost:6167"
	defaultTimeout    = 120 * time.Second
)

var (
	adminUsername string
	serverName    string
	logFile       *os.File
)

type logWriter struct{}

func (l logWriter) Write(p []byte) (n int, err error) {
	fmt.Print(string(p))
	if logFile != nil {
		logFile.Write(p)
	}
	return len(p), nil
}

var logger = logWriter{}

func logInfo(format string, args ...interface{}) {
	fmt.Fprintf(logger, "[%s] [INFO] %s\n", time.Now().Format("2006-01-02 15:04:05"), fmt.Sprintf(format, args...))
}

func logWarn(format string, args ...interface{}) {
	fmt.Fprintf(logger, "[%s] [WARN] %s\n", time.Now().Format("2006-01-02 15:04:05"), fmt.Sprintf(format, args...))
}

func logError(format string, args ...interface{}) {
	fmt.Fprintf(logger, "[%s] [ERROR] %s\n", time.Now().Format("2006-01-02 15:04:05"), fmt.Sprintf(format, args...))
}

func logDebug(format string, args ...interface{}) {
	fmt.Fprintf(logger, "[%s] [DEBUG] %s\n", time.Now().Format("2006-01-02 15:04:05"), fmt.Sprintf(format, args...))
}

type NonceResponse struct {
	Nonce string `json:"nonce"`
}

type RegisterResponse struct {
	UserID  string `json:"user_id"`
	ErrCode string `json:"errcode"`
	Error   string `json:"error"`
}

func generateSecureSecret(length int) (string, error) {
	bytes := make([]byte, length/2)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func generateSecurePassword() (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	const length = 24
	result := make([]byte, length)
	charsetLen := big.NewInt(int64(len(charset)))

	for i := range result {
		n, err := rand.Int(rand.Reader, charsetLen)
		if err != nil {
			return "", err
		}
		result[i] = charset[n.Int64()]
	}
	return string(result), nil
}

func waitForConduit(timeout time.Duration, maxAttempts int) bool {
	client := &http.Client{Timeout: 5 * time.Second}
	start := time.Now()
	attempt := 0
	backoff := 1 * time.Second

	logInfo("Waiting for Conduit to become ready (max %ds)...", int(timeout.Seconds()))

	for time.Since(start) < timeout && attempt < maxAttempts {
		attempt++
		resp, err := client.Get(conduitURL + "/_matrix/client/versions")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == 200 {
				logInfo("Conduit is ready after %d attempts", attempt)
				return true
			} else if resp.StatusCode == 503 {
				logWarn("Conduit temporarily unavailable (attempt %d/%d)", attempt, maxAttempts)
				time.Sleep(backoff)
				if backoff < 10*time.Second {
					backoff *= 2
				}
				continue
			}
			logDebug("Conduit not ready yet (HTTP %d, attempt %d)", resp.StatusCode, attempt)
		} else {
			logDebug("Conduit not ready yet: %v", err)
		}
		time.Sleep(2 * time.Second)
	}

	logError("Conduit failed to become ready after %ds", int(timeout.Seconds()))
	return false
}

func updateConduitConfigLine(key, value string) error {
	data, err := os.ReadFile(conduitConfigPath)
	if err != nil {
		return fmt.Errorf("failed to read config: %w", err)
	}

	var buf bytes.Buffer
	scanner := bufio.NewScanner(bytes.NewReader(data))
	keyFound := false

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, key+" ") || strings.HasPrefix(trimmed, key+"=") {
			if strings.Contains(key, "allow_registration") {
				buf.WriteString(fmt.Sprintf("allow_registration = %s\n", value))
			} else {
				buf.WriteString(fmt.Sprintf("%s = \"%s\"\n", key, value))
			}
			keyFound = true
		} else {
			buf.WriteString(line + "\n")
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	if !keyFound {
		if strings.Contains(key, "allow_registration") {
			buf.WriteString(fmt.Sprintf("allow_registration = %s\n", value))
		} else {
			buf.WriteString(fmt.Sprintf("%s = \"%s\"\n", key, value))
		}
	}

	if err := os.WriteFile(conduitConfigPath, buf.Bytes(), 0600); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	logInfo("Conduit config updated: %s", key)
	return nil
}

func removeConfigLine(key string) error {
	data, err := os.ReadFile(conduitConfigPath)
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	scanner := bufio.NewScanner(bytes.NewReader(data))

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, key+" ") || strings.HasPrefix(trimmed, key+"=") {
			continue
		}
		buf.WriteString(line + "\n")
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	if err := os.WriteFile(conduitConfigPath, buf.Bytes(), 0600); err != nil {
		return err
	}

	logInfo("Config key removed: %s", key)
	return nil
}

func computeHMAC(nonce, username, password, sharedSecret string) string {
	data := fmt.Sprintf("%s\x00%s\x00%s\x00admin", nonce, username, password)
	mac := hmac.New(sha1.New, []byte(sharedSecret))
	mac.Write([]byte(data))
	return hex.EncodeToString(mac.Sum(nil))
}

func registerAdmin(sharedSecret, password string) (string, error) {
	client := &http.Client{Timeout: 10 * time.Second}

	nonceResp, err := getNonce(client)
	if err != nil {
		return "", err
	}

	attemptUsername := adminUsername
	maxRetries := 3

	for retry := 0; retry < maxRetries; retry++ {
		if retry > 0 {
			attemptUsername = fmt.Sprintf("armor-admin-%s", uuid.New().String()[:6])
			logInfo("Trying alternative username: %s", attemptUsername)
		}

		mac := computeHMAC(nonceResp.Nonce, attemptUsername, password, sharedSecret)

		payload := map[string]interface{}{
			"nonce":    nonceResp.Nonce,
			"username": attemptUsername,
			"password": password,
			"admin":    true,
			"mac":      mac,
		}

		body, err := json.Marshal(payload)
		if err != nil {
			return "", err
		}

		resp, err := client.Post(
			conduitURL+"/_synapse/admin/v1/register",
			"application/json",
			bytes.NewReader(body),
		)
		if err != nil {
			return "", err
		}

		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return "", err
		}

		var regResp RegisterResponse
		if err := json.Unmarshal(respBody, &regResp); err != nil {
			return "", err
		}

		if resp.StatusCode == 200 {
			userID := regResp.UserID
			if userID == "" {
				userID = fmt.Sprintf("@%s:%s", attemptUsername, serverName)
			}
			logInfo("Admin user registered successfully: %s", userID)

			usernameFile := "/var/lib/armorclaw/.admin_username"
			if err := os.WriteFile(usernameFile, []byte(attemptUsername), 0644); err != nil {
				logWarn("Failed to save username file: %v", err)
			}

			adminUsername = attemptUsername
			return userID, nil
		}

		if resp.StatusCode == 400 || resp.StatusCode == 409 {
			if strings.Contains(strings.ToLower(regResp.ErrCode), "user_in_use") ||
				strings.Contains(strings.ToLower(regResp.Error), "already in use") {
				logWarn("Username '%s' already exists", attemptUsername)
				continue
			}
			return "", fmt.Errorf("registration failed: %s", regResp.Error)
		}

		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return "", fmt.Errorf("failed to register admin after %d attempts", maxRetries)
}

func getNonce(client *http.Client) (*NonceResponse, error) {
	resp, err := client.Get(conduitURL + "/_synapse/admin/v1/register")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var nonceResp NonceResponse
	if err := json.Unmarshal(body, &nonceResp); err != nil {
		return nil, err
	}

	if nonceResp.Nonce == "" {
		return nil, fmt.Errorf("no nonce returned")
	}

	return &nonceResp, nil
}

func sendSIGHUP() error {
	cmd := exec.Command("pidof", "conduit")
	output, err := cmd.Output()
	if err != nil {
		return err
	}

	pids := strings.Fields(strings.TrimSpace(string(output)))
	if len(pids) == 0 {
		return fmt.Errorf("no conduit process found")
	}

	var pid int
	if _, err := fmt.Sscanf(pids[0], "%d", &pid); err != nil {
		return err
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		return err
	}

	if err := proc.Signal(syscall.SIGHUP); err != nil {
		return err
	}

	logInfo("Sent SIGHUP to Conduit (PID %d)", pid)
	return nil
}

func validateUsername(username string) bool {
	if len(username) == 0 || len(username) > 255 {
		return false
	}
	for _, r := range username {
		if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') ||
			r == '-' || r == '_' || r == '.') {
			return false
		}
	}
	return true
}

func init() {
	if _, err := os.Stat(initFlagPath); err == nil {
		os.Exit(0)
	}

	serverName = os.Getenv("ARMORCLAW_SERVER_NAME")
	if serverName == "" {
		serverName = "localhost"
	}

	rawAdminUsername := os.Getenv("ARMORCLAW_ADMIN_USERNAME")
	if rawAdminUsername != "" && validateUsername(rawAdminUsername) {
		adminUsername = rawAdminUsername
	} else {
		adminUsername = fmt.Sprintf("armor-admin-%s", uuid.New().String()[:8])
	}

	if err := os.MkdirAll(filepath.Dir(logFilePath), 0750); err == nil {
		logFile, _ = os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	}
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--help" {
		fmt.Println("ArmorClaw Production Bootstrap - Secure Admin Creation")
		fmt.Println("\nEnvironment variables:")
		fmt.Println("  ARMORCLAW_ADMIN_USERNAME  Admin username (default: random)")
		fmt.Println("  ARMORCLAW_ADMIN_PASSWORD  Admin password (default: random)")
		fmt.Println("  ARMORCLAW_SERVER_NAME     Server name (default: localhost)")
		os.Exit(0)
	}

	logInfo("Starting secure first-run bootstrap...")

	password := os.Getenv("ARMORCLAW_ADMIN_PASSWORD")
	if password == "" {
		var err error
		password, err = generateSecurePassword()
		if err != nil {
			logError("Failed to generate password: %v", err)
			os.Exit(1)
		}
	}
	logInfo("Admin password generated (length: %d)", len(password))

	sharedSecret, err := generateSecureSecret(64)
	if err != nil {
		logError("Failed to generate shared secret: %v", err)
		os.Exit(1)
	}
	logDebug("Generated registration shared secret")

	if !waitForConduit(defaultTimeout, 60) {
		os.Exit(1)
	}

	if err := updateConduitConfigLine("allow_registration", "false"); err != nil {
		logError("Failed to set allow_registration: %v", err)
		os.Exit(1)
	}

	if err := updateConduitConfigLine("registration_shared_secret", sharedSecret); err != nil {
		logError("Failed to update Conduit config: %v", err)
		os.Exit(1)
	}

	if err := sendSIGHUP(); err != nil {
		logWarn("Could not reload Conduit: %v", err)
	}

	if !waitForConduitWithRetry(30 * time.Second) {
		os.Exit(1)
	}

	userID, err := registerAdmin(sharedSecret, password)
	if err != nil {
		logError("Admin registration failed: %v", err)
		os.Exit(1)
	}

	if err := removeConfigLine("registration_shared_secret"); err != nil {
		logWarn("Failed to remove shared secret: %v", err)
	}

	if err := os.MkdirAll(filepath.Dir(initFlagPath), 0750); err != nil {
		logWarn("Failed to create flag directory: %v", err)
	}

	if err := os.WriteFile(initFlagPath, []byte{}, 0644); err != nil {
		logWarn("Failed to create init flag: %v", err)
	}

	logInfo("Bootstrap complete. Admin user ready: %s", userID)

	fmt.Printf("\n%s\n", strings.Repeat("=", 60))
	fmt.Println("ArmorClaw Production Bootstrap - SUCCESS")
	fmt.Printf("%s\n", strings.Repeat("=", 60))
	fmt.Printf("Admin Username: @%s:%s\n", adminUsername, serverName)
	fmt.Printf("Admin Password: %s\n", password)
	fmt.Println("")
	fmt.Println("⚠️  SAVE CREDENTIALS NOW - They will NOT be stored!")
	fmt.Println("   Password is shown ONLY this once.")
	fmt.Println("")
	fmt.Printf("Next: Connect via Element X or ArmorChat using http://<your-ip>:6167\n")
	fmt.Printf("%s\n\n", strings.Repeat("=", 60))

	if logFile != nil {
		logFile.Close()
	}
}

func waitForConduitWithRetry(timeout time.Duration) bool {
	client := &http.Client{Timeout: 5 * time.Second}
	start := time.Now()
	interval := 500 * time.Millisecond

	for time.Since(start) < timeout {
		resp, err := client.Get(conduitURL + "/_matrix/client/versions")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == 200 {
				logInfo("Conduit reloaded and ready")
				return true
			}
		}
		time.Sleep(interval)
	}

	logError("Conduit failed to respond after reload")
	return false
}
