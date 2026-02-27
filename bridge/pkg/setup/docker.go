package setup

import (
	"context"
	"fmt"
	"net"
	"os"
	"time"
)

const (
	// DefaultDockerSocketPath is the default path to the Docker socket.
	DefaultDockerSocketPath = "/var/run/docker.sock"

	// DockerHealthCheckTimeout is the timeout for Docker daemon health checks.
	DockerHealthCheckTimeout = 5 * time.Second
)

// DockerCheckResult contains information about Docker availability.
type DockerCheckResult struct {
	SocketExists   bool
	SocketReadable bool
	SocketWritable bool
	DaemonRunning  bool
	Error          error
}

// ValidateDockerSocket checks if the Docker socket exists and is accessible.
func ValidateDockerSocket() error {
	return ValidateDockerSocketAtPath(DefaultDockerSocketPath)
}

// ValidateDockerSocketAtPath checks if a Docker socket at the given path is accessible.
func ValidateDockerSocketAtPath(socketPath string) error {
	// Check if socket exists
	info, err := os.Stat(socketPath)
	if os.IsNotExist(err) {
		return ErrDockerSocket
	}
	if err != nil {
		return WrapError(err, ErrDockerSocket)
	}

	// Verify it's a socket
	if info.Mode()&os.ModeSocket == 0 {
		return &SetupError{
			Code:     "INS-001",
			Title:    "Docker socket not a socket",
			Cause:    fmt.Sprintf("The path %s exists but is not a Unix socket.", socketPath),
			Fix: []string{
				"Verify Docker is properly installed",
				"Check if another process is using the socket path",
			},
			ExitCode: 1,
		}
	}

	// Check if readable
	file, err := os.OpenFile(socketPath, os.O_RDONLY, 0)
	if err != nil {
		return ErrDockerPermission
	}
	file.Close()

	return nil
}

// CheckDockerDaemon verifies the Docker daemon is responding.
// This connects to the socket and sends a simple HTTP request.
func CheckDockerDaemon() error {
	return CheckDockerDaemonAtPath(DefaultDockerSocketPath)
}

// CheckDockerDaemonAtPath verifies the Docker daemon at a specific socket path.
func CheckDockerDaemonAtPath(socketPath string) error {
	// First validate the socket
	if err := ValidateDockerSocketAtPath(socketPath); err != nil {
		return err
	}

	// Try to connect and send a simple request
	conn, err := net.DialTimeout("unix", socketPath, DockerHealthCheckTimeout)
	if err != nil {
		return WrapError(err, ErrDockerSocket)
	}
	defer conn.Close()

	// Send a simple GET /_ping request
	ctx, cancel := context.WithTimeout(context.Background(), DockerHealthCheckTimeout)
	defer cancel()

	// Set deadline from context
	deadline, _ := ctx.Deadline()
	conn.SetDeadline(deadline)

	// Docker API ping endpoint
	request := "GET /_ping HTTP/1.1\r\nHost: localhost\r\n\r\n"
	_, err = conn.Write([]byte(request))
	if err != nil {
		return &SetupError{
			Code:     "INS-001",
			Title:    "Docker daemon not responding",
			Cause:    "The Docker socket exists but the daemon is not responding to requests.",
			Fix: []string{
				"Check if Docker daemon is running: systemctl status docker",
				"Restart Docker: sudo systemctl restart docker",
				"Check Docker logs: journalctl -u docker",
			},
			ExitCode: 1,
		}
	}

	// Read response
	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil || n == 0 {
		return &SetupError{
			Code:     "INS-001",
			Title:    "Docker daemon not responding",
			Cause:    "The Docker daemon accepted the connection but did not respond.",
			Fix: []string{
				"Check if Docker daemon is running: systemctl status docker",
				"Restart Docker: sudo systemctl restart docker",
			},
			ExitCode: 1,
		}
	}

	// Check for HTTP 200 response
	response := string(buf[:n])
	if len(response) < 12 || response[:12] != "HTTP/1.1 200" {
		return &SetupError{
			Code:     "INS-001",
			Title:    "Docker daemon unhealthy",
			Cause:    fmt.Sprintf("Docker daemon returned unexpected response: %s", response[:min(50, len(response))]),
			Fix: []string{
				"Check Docker daemon logs for errors",
				"Restart Docker: sudo systemctl restart docker",
			},
			ExitCode: 1,
		}
	}

	return nil
}

// FullDockerCheck performs a comprehensive Docker availability check.
func FullDockerCheck() DockerCheckResult {
	result := DockerCheckResult{}
	socketPath := DefaultDockerSocketPath

	// Check socket exists
	info, err := os.Stat(socketPath)
	if os.IsNotExist(err) {
		result.Error = ErrDockerSocket
		return result
	}
	result.SocketExists = err == nil

	// Check if it's a socket
	if result.SocketExists && info.Mode()&os.ModeSocket != 0 {
		// Check readable
		file, err := os.OpenFile(socketPath, os.O_RDONLY, 0)
		if err == nil {
			result.SocketReadable = true
			file.Close()
		}

		// Check writable (by trying to open for write without actually writing)
		file, err = os.OpenFile(socketPath, os.O_WRONLY, 0)
		if err == nil {
			result.SocketWritable = true
			file.Close()
		}
	}

	// Check daemon
	if result.SocketReadable {
		conn, err := net.DialTimeout("unix", socketPath, DockerHealthCheckTimeout)
		if err == nil {
			conn.Write([]byte("GET /_ping HTTP/1.1\r\nHost: localhost\r\n\r\n"))
			buf := make([]byte, 1024)
			n, err := conn.Read(buf)
			conn.Close()
			result.DaemonRunning = err == nil && n > 0
		}
	}

	// Set appropriate error
	if !result.SocketExists {
		result.Error = ErrDockerSocket
	} else if !result.SocketReadable || !result.SocketWritable {
		result.Error = ErrDockerPermission
	} else if !result.DaemonRunning {
		result.Error = &SetupError{
			Code:     "INS-001",
			Title:    "Docker daemon not running",
			Cause:    "The Docker socket is accessible but the daemon is not responding.",
			Fix: []string{
				"Start Docker: sudo systemctl start docker",
				"Check Docker status: systemctl status docker",
			},
			ExitCode: 1,
		}
	}

	return result
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
