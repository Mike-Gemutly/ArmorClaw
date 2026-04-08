// Package vault provides a Go client wrapper around the Vault Governance gRPC service.
//
// It connects to the Rust vault keystore over a Unix domain socket and exposes
// high-level methods for ephemeral token lifecycle (issue, consume, zeroize).
package vault

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"time"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/armorclaw/bridge/pkg/vault/proto"
)

// DefaultVaultSocketPath is the default Unix domain socket path for the vault keystore.
const DefaultVaultSocketPath = "/run/armorclaw/keystore.sock"

var (
	vaultIssueDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "armorclaw_vault_grpc_issue_duration_seconds",
		Help:    "Duration of IssueBlindFillToken gRPC calls",
		Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
	})
	vaultConsumeDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "armorclaw_vault_grpc_consume_duration_seconds",
		Help:    "Duration of ConsumeTokenForSidecar gRPC calls",
		Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
	})
	vaultZeroizeDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "armorclaw_vault_grpc_zeroize_duration_seconds",
		Help:    "Duration of ZeroizeToolSecrets gRPC calls",
		Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
	})
	vaultErrorsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "armorclaw_vault_grpc_errors_total",
		Help: "Total gRPC errors from vault governance client",
	}, []string{"method", "code"})
)

func init() {
	prometheus.MustRegister(vaultIssueDuration)
	prometheus.MustRegister(vaultConsumeDuration)
	prometheus.MustRegister(vaultZeroizeDuration)
	prometheus.MustRegister(vaultErrorsTotal)
}

// VaultGovernanceClient wraps the generated Governance gRPC client with
// connection lifecycle management and high-level convenience methods.
type VaultGovernanceClient struct {
	conn   *grpc.ClientConn
	client pb.GovernanceClient
	logger *slog.Logger
}

// NewGovernanceClient creates a new VaultGovernanceClient and immediately
// establishes a gRPC connection over the given Unix domain socket.
func NewGovernanceClient(socketPath string) (*VaultGovernanceClient, error) {
	if socketPath == "" {
		socketPath = DefaultVaultSocketPath
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dialer := func(ctx context.Context, _ string) (net.Conn, error) {
		d := net.Dialer{}
		return d.DialContext(ctx, "unix", socketPath)
	}

	backoffConfig := backoff.Config{
		MaxDelay: 5 * time.Second,
	}

	conn, err := grpc.DialContext(ctx,
		"unix://"+socketPath,
		grpc.WithContextDialer(dialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithConnectParams(grpc.ConnectParams{
			Backoff:           backoffConfig,
			MinConnectTimeout: 5 * time.Second,
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("dial vault governance: %w", err)
	}

	return &VaultGovernanceClient{
		conn:   conn,
		client: pb.NewGovernanceClient(conn),
		logger: slog.Default(),
	}, nil
}

// IssueBlindFillToken generates a UUID token_id and delegates to IssueEphemeralToken.
// Returns the token_id on success so the caller can later pass it to ConsumeTokenForSidecar.
func (c *VaultGovernanceClient) IssueBlindFillToken(ctx context.Context, sessionID, toolName, secret string, ttl time.Duration) (string, error) {
	start := time.Now()
	tokenID := uuid.New().String()

	ttlMs := ttl.Milliseconds()
	_, err := c.client.IssueEphemeralToken(ctx, &pb.IssueTokenRequest{
		TokenId:   tokenID,
		Plaintext: secret,
		SessionId: sessionID,
		ToolName:  toolName,
		TtlMs:     ttlMs,
	})
	vaultIssueDuration.Observe(time.Since(start).Seconds())
	if err != nil {
		vaultErrorsTotal.WithLabelValues("IssueBlindFillToken", grpc.Code(err).String()).Inc()
		return "", fmt.Errorf("issue blind fill token: %w", err)
	}

	return tokenID, nil
}

// ConsumeTokenForSidecar redeems an ephemeral token and returns the plaintext secret.
// Maps gRPC status codes to idiomatic Go errors:
//   - NOT_FOUND → wrapped "token not found or already consumed"
//   - PERMISSION_DENIED → wrapped "session does not own this token"
//   - DEADLINE_EXCEEDED → wrapped "token TTL expired"
func (c *VaultGovernanceClient) ConsumeTokenForSidecar(ctx context.Context, tokenID, sessionID, toolName string) (string, error) {
	start := time.Now()
	resp, err := c.client.ConsumeEphemeralToken(ctx, &pb.ConsumeTokenRequest{
		TokenId:   tokenID,
		SessionId: sessionID,
		ToolName:  toolName,
	})
	vaultConsumeDuration.Observe(time.Since(start).Seconds())
	if err != nil {
		vaultErrorsTotal.WithLabelValues("ConsumeTokenForSidecar", grpc.Code(err).String()).Inc()
		return "", fmt.Errorf("consume token: %w", err)
	}

	return resp.Plaintext, nil
}

// ZeroizeToolSecrets securely erases all in-memory secrets for the given tool/session pair.
// Returns the number of secrets destroyed.
func (c *VaultGovernanceClient) ZeroizeToolSecrets(ctx context.Context, toolName, sessionID string) (uint32, error) {
	start := time.Now()
	resp, err := c.client.ZeroizeToolSecrets(ctx, &pb.ZeroizeRequest{
		ToolName:  toolName,
		SessionId: sessionID,
	})
	vaultZeroizeDuration.Observe(time.Since(start).Seconds())
	if err != nil {
		vaultErrorsTotal.WithLabelValues("ZeroizeToolSecrets", grpc.Code(err).String()).Inc()
		return 0, fmt.Errorf("zeroize tool secrets: %w", err)
	}

	return uint32(resp.SecretsDestroyed), nil
}

// Close closes the underlying gRPC connection.
func (c *VaultGovernanceClient) Close() error {
	if c.conn == nil {
		return nil
	}
	err := c.conn.Close()
	if err != nil {
		return fmt.Errorf("close vault governance connection: %w", err)
	}
	return nil
}
