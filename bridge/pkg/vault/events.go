package vault

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/armorclaw/bridge/pkg/eventbus"
	pb "github.com/armorclaw/bridge/pkg/vault/proto"
)

const (
	initialBackoff = 1 * time.Second
	maxBackoff     = 30 * time.Second
	backoffFactor  = 2.0
)

// VaultEventBridge subscribes to vault governance events via gRPC streaming
// and publishes them as BridgeEvents on the EventBus.
type VaultEventBridge struct {
	client        *VaultGovernanceClient
	bus           *eventbus.EventBus
	logger        *slog.Logger
	sessionFilter string

	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewVaultEventBridge creates a new VaultEventBridge.
func NewVaultEventBridge(client *VaultGovernanceClient, bus *eventbus.EventBus) *VaultEventBridge {
	return &VaultEventBridge{
		client: client,
		bus:    bus,
		logger: slog.Default(),
	}
}

// WithSessionFilter sets an optional session filter for the subscription.
func (b *VaultEventBridge) WithSessionFilter(sessionID string) *VaultEventBridge {
	b.sessionFilter = sessionID
	return b
}

// WithLogger sets a custom logger.
func (b *VaultEventBridge) WithLogger(logger *slog.Logger) *VaultEventBridge {
	b.logger = logger
	return b
}

// StartSyncLoop starts a background goroutine that subscribes to vault events
// and publishes them to the EventBus. It reconnects automatically with
// exponential backoff on stream errors.
func (b *VaultEventBridge) StartSyncLoop(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	b.cancel = cancel

	b.wg.Add(1)
	go b.syncLoop(ctx)
}

// Stop signals the sync loop to stop and waits for it to finish.
func (b *VaultEventBridge) Stop() {
	if b.cancel != nil {
		b.cancel()
	}
	b.wg.Wait()
}

func (b *VaultEventBridge) syncLoop(ctx context.Context) {
	defer b.wg.Done()

	backoff := initialBackoff

	for {
		select {
		case <-ctx.Done():
			b.logger.Debug("vault event bridge stopped: context cancelled")
			return
		default:
		}

		err := b.subscribeAndPublish(ctx)
		if err != nil {
			if ctx.Err() != nil {
				b.logger.Debug("vault event bridge stopped: context cancelled after error")
				return
			}

			b.logger.Warn("vault event stream error, reconnecting",
				"error", err,
				"backoff", backoff,
			)

			timer := time.NewTimer(backoff)
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-timer.C:
			}

			backoff = time.Duration(float64(backoff) * backoffFactor)
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
		} else {
			backoff = initialBackoff
		}
	}
}

func (b *VaultEventBridge) subscribeAndPublish(ctx context.Context) error {
	req := &pb.SubscribeRequest{
		SessionId: b.sessionFilter,
	}

	stream, err := b.client.client.SubscribeEvents(ctx, req)
	if err != nil {
		return err
	}

	for {
		event, err := stream.Recv()
		if err != nil {
			return err
		}

		bridgeEvent := b.mapVaultEvent(event)
		if bridgeEvent == nil {
			b.logger.Debug("unmapped vault event type, skipping",
				"event_type", event.GetEventType(),
			)
			continue
		}

		if publishErr := b.bus.PublishBridgeEvent(bridgeEvent); publishErr != nil {
			b.logger.Warn("failed to publish vault event to eventbus",
				"error", publishErr,
				"event_type", event.GetEventType(),
			)
		}
	}
}

// mapVaultEvent converts a VaultEventStream proto message into a BridgeEvent.
func (b *VaultEventBridge) mapVaultEvent(ve *pb.VaultEventStream) eventbus.BridgeEvent {
	switch ve.GetEventType() {
	case "token_issued", "token_consumed", "secrets_zeroized":
		return eventbus.NewPlatformConnectedEvent("vault", ve.GetEventType())
	case "skill_gate_denied", "pii_detected_in_output":
		return &eventbus.BaseEvent{
			Type: eventbus.EventTypeAgentError,
			Ts:   time.Now(),
		}
	default:
		return nil
	}
}
