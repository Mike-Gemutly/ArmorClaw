package sidecar

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"testing"
	"time"
)

func findJava(t *testing.T) string {
	t.Helper()
	java, err := exec.LookPath("java")
	if err != nil {
		t.Skip("java not available on PATH — skipping Java E2E tests")
	}
	// Verify Java 21+ via --version output.
	out, err := exec.Command(java, "--version").CombinedOutput()
	if err != nil {
		t.Skipf("java --version failed: %v — skipping Java E2E tests", err)
	}
	firstLine := string(out)
	verRe := regexp.MustCompile(`(\d+)\.\d+\.\d+`)
	m := verRe.FindStringSubmatch(firstLine)
	if len(m) < 2 {
		// Single-number version like "openjdk 21 2024-04-16"
		singleRe := regexp.MustCompile(`(?:openjdk|java)\s+(\d+)`)
		sm := singleRe.FindStringSubmatch(firstLine)
		if len(sm) < 2 {
			t.Skipf("cannot parse java version from %q — skipping Java E2E tests", firstLine)
		}
		m = sm
	}
	major := m[1]
	if major < "21" {
		t.Skipf("java version %s is not 21+ — skipping Java E2E tests", major)
	}
	return java
}

func findJar(t *testing.T) string {
	t.Helper()
	jarPath := filepath.Join("..", "..", "..", "sidecar-java", "target", "sidecar.jar")
	abs, _ := filepath.Abs(jarPath)
	if _, err := os.Stat(jarPath); err != nil {
		t.Skipf("sidecar.jar not found at %s — skipping Java E2E tests", abs)
	}
	return jarPath
}

type javaServer struct {
	cmd    *exec.Cmd
	socket string
}

func startJavaSidecar(t *testing.T) *javaServer {
	t.Helper()
	java := findJava(t)
	jarPath := findJar(t)

	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "sidecar-java.sock")
	secretPath := filepath.Join(tmpDir, "secret")

	// Create empty secret file (dev mode — no HMAC validation).
	if err := os.WriteFile(secretPath, []byte{}, 0600); err != nil {
		t.Fatalf("write secret: %v", err)
	}

	cmd := exec.Command(java, "-jar", jarPath)
	cmd.Env = append(os.Environ(),
		"SIDECAR_JAVA_SOCKET_PATH="+socketPath,
		"SIDECAR_JAVA_MAX_REQUESTS=1000",
		"SIDECAR_SHARED_SECRET_PATH="+secretPath,
	)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		t.Fatalf("start java sidecar: %v", err)
	}

	// JVM cold start is slower than Python — use 20-second deadline.
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(socketPath); err == nil {
			return &javaServer{cmd: cmd, socket: socketPath}
		}
		time.Sleep(200 * time.Millisecond)
	}
	cmd.Process.Kill()
	cmd.Wait()
	t.Fatal("java sidecar socket did not appear within 20s")
	return nil
}

func (js *javaServer) stop() {
	if js.cmd != nil && js.cmd.Process != nil {
		js.cmd.Process.Kill()
		js.cmd.Wait()
	}
}

func (js *javaServer) makeClient(t *testing.T) *Client {
	t.Helper()
	return NewClient(&Config{
		SocketPath:  js.socket,
		Timeout:     15 * time.Second,
		MaxRetries:  1,
		DialTimeout: 5 * time.Second,
		MaxMsgSize:  JavaMaxMsgSize,
	})
}

func TestE2E_Java_HealthCheck(t *testing.T) {
	js := startJavaSidecar(t)
	defer js.stop()

	client := js.makeClient(t)
	defer client.Close()

	resp, err := client.HealthCheck(context.Background())
	if err != nil {
		t.Fatalf("HealthCheck: %v", err)
	}
	if resp.Status != "SERVING" {
		t.Errorf("expected status SERVING, got %q", resp.Status)
	}
	if resp.Version != "1.0.0" {
		t.Errorf("expected version 1.0.0, got %q", resp.Version)
	}
}

func TestE2E_Java_DocExtraction(t *testing.T) {
	js := startJavaSidecar(t)
	defer js.stop()

	client := js.makeClient(t)
	defer client.Close()

	docData, err := os.ReadFile(filepath.Join("..", "..", "..", "tests", "fixtures", "sample.doc"))
	if err != nil {
		t.Fatalf("read sample.doc fixture: %v", err)
	}

	req := &ExtractTextRequest{
		DocumentFormat:  "application/msword",
		DocumentContent: docData,
	}

	resp, err := client.ExtractText(context.Background(), req)
	if err != nil {
		t.Fatalf("ExtractText doc: %v", err)
	}
	if len(resp.Text) == 0 {
		t.Error("expected non-empty text from DOC extraction")
	}
}

func TestE2E_Java_PptExtraction(t *testing.T) {
	js := startJavaSidecar(t)
	defer js.stop()

	client := js.makeClient(t)
	defer client.Close()

	pptData, err := os.ReadFile(filepath.Join("..", "..", "..", "tests", "fixtures", "sample.ppt"))
	if err != nil {
		t.Fatalf("read sample.ppt fixture: %v", err)
	}

	req := &ExtractTextRequest{
		DocumentFormat:  "application/vnd.ms-powerpoint",
		DocumentContent: pptData,
	}

	resp, err := client.ExtractText(context.Background(), req)
	if err != nil {
		t.Fatalf("ExtractText ppt: %v", err)
	}
	if len(resp.Text) == 0 {
		t.Error("expected non-empty text from PPT extraction")
	}
}

func TestE2E_Java_UnsupportedFormat(t *testing.T) {
	js := startJavaSidecar(t)
	defer js.stop()

	client := js.makeClient(t)
	defer client.Close()

	req := &ExtractTextRequest{
		DocumentFormat:  "application/pdf",
		DocumentContent: []byte("%PDF-1.4 dummy"),
	}

	_, err := client.ExtractText(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for unsupported format PDF, got nil")
	}
}
