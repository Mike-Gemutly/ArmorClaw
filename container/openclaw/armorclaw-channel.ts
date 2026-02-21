/**
 * ArmorClaw Channel Provider for OpenClaw
 *
 * This channel provider integrates OpenClaw with ArmorClaw's bridge,
 * allowing messages to flow between Matrix (via ArmorClaw) and OpenClaw.
 *
 * Location: Copy to openclaw/src/channels/plugins/armorclaw/
 */

import { ArmorClawBridgeClient, type MatrixEvent, type BridgeStatus } from './bridge-client.js';
import type {
  ChannelMeta,
  ChannelPlugin,
  ChannelAccountSnapshot,
  ChannelInboundMessage,
  ChannelOutboundMessage,
} from './types.core.js';

// Channel metadata
export const ARMORCLAW_CHANNEL_META: ChannelMeta = {
  id: 'armorclaw',
  label: 'ArmorClaw',
  selectionLabel: 'ArmorClaw (Matrix Bridge)',
  docsPath: '/channels/armorclaw',
  blurb: 'Secure Matrix communication through ArmorClaw bridge',
  order: 100,
  aliases: ['matrix-bridge', 'armor'],
  selectionExtras: ['Zero-trust security', 'E2EE support', 'Container isolation'],
};

export interface ArmorClawConfig {
  socketPath?: string;
  roomId?: string;
  autoReply?: boolean;
  verbose?: boolean;
}

export interface ArmorClawAccountSnapshot extends ChannelAccountSnapshot {
  roomId: string;
  connected: boolean;
}

/**
 * ArmorClaw Channel Plugin
 *
 * Bridges OpenClaw to Matrix through the ArmorClaw security layer.
 */
export class ArmorClawChannelPlugin implements ChannelPlugin {
  readonly id = 'armorclaw';
  readonly meta = ARMORCLAW_CHANNEL_META;

  private client: ArmorClawBridgeClient;
  private config: ArmorClawConfig;
  private pollInterval?: NodeJS.Timeout;
  private running = false;

  constructor(config: ArmorClawConfig = {}) {
    this.config = config;
    this.client = new ArmorClawBridgeClient({
      socketPath: config.socketPath,
    });
  }

  // ==========================================================================
  // Channel Plugin Interface
  // ==========================================================================

  /**
   * Initialize the channel
   */
  async setup(input: { config?: ArmorClawConfig }): Promise<void> {
    if (input.config) {
      this.config = { ...this.config, ...input.config };
    }

    // Verify bridge connection
    try {
      const status = await this.client.status();
      console.log(`[ArmorClaw] Connected to bridge v${status.version}`);
    } catch (error) {
      throw new Error(`Failed to connect to ArmorClaw bridge: ${error}`);
    }
  }

  /**
   * Start listening for messages
   */
  async start(onMessage: (msg: ChannelInboundMessage) => Promise<void>): Promise<void> {
    if (this.running) return;
    this.running = true;

    // Poll for Matrix events
    this.pollInterval = setInterval(async () => {
      try {
        const events = await this.client.matrixReceive(this.config.roomId, 10);
        for (const event of events) {
          const message = this.mapEventToMessage(event);
          if (message) {
            await onMessage(message);
          }
        }
      } catch (error) {
        console.error('[ArmorClaw] Error polling for messages:', error);
      }
    }, 1000); // Poll every second

    console.log('[ArmorClaw] Started listening for messages');
  }

  /**
   * Stop listening for messages
   */
  async stop(): Promise<void> {
    this.running = false;
    if (this.pollInterval) {
      clearInterval(this.pollInterval);
      this.pollInterval = undefined;
    }
    console.log('[ArmorClaw] Stopped listening');
  }

  /**
   * Send a message through ArmorClaw
   */
  async send(message: ChannelOutboundMessage): Promise<void> {
    const roomId = message.roomId || this.config.roomId;
    if (!roomId) {
      throw new Error('No room ID configured for ArmorClaw');
    }

    await this.client.matrixSend(roomId, message.body, message.msgtype || 'm.text');
  }

  /**
   * Get account status
   */
  async getStatus(): Promise<ArmorClawAccountSnapshot | null> {
    try {
      const status = await this.client.status();
      const matrixStatus = await this.client.matrixStatus();

      return {
        accountId: 'armorclaw-bridge',
        name: 'ArmorClaw Bridge',
        enabled: true,
        roomId: this.config.roomId || '',
        connected: matrixStatus.connected,
      };
    } catch {
      return null;
    }
  }

  /**
   * Health check
   */
  async healthCheck(): Promise<{ healthy: boolean; message?: string }> {
    try {
      const health = await this.client.health();
      return {
        healthy: health.status === 'ok',
        message: health.status,
      };
    } catch (error) {
      return {
        healthy: false,
        message: String(error),
      };
    }
  }

  // ==========================================================================
  // Helper Methods
  // ==========================================================================

  /**
   * Map Matrix event to channel inbound message
   */
  private mapEventToMessage(event: MatrixEvent): ChannelInboundMessage | null {
    // Only process text messages
    if (event.type !== 'm.room.message') return null;
    if (event.content.msgtype !== 'm.text' && event.content.msgtype !== 'm.emote') return null;

    return {
      channelId: 'armorclaw',
      accountId: 'armorclaw-bridge',
      from: event.sender,
      to: event.room_id,
      body: event.content.body || '',
      msgtype: event.content.msgtype,
      timestamp: event.origin_server_ts,
      metadata: {
        eventId: event.event_id,
        roomId: event.room_id,
      },
    };
  }

  /**
   * Get channel actions (for OpenClaw integration)
   */
  getActions(): Array<{ name: string; handler: (...args: unknown[]) => Promise<void> }> {
    return [
      {
        name: 'sendReaction',
        handler: async (roomId: string, eventId: string, emoji: string) => {
          await this.client.matrixReact(roomId, eventId, emoji);
        },
      },
      {
        name: 'sendReply',
        handler: async (roomId: string, eventId: string, message: string) => {
          await this.client.matrixReply(roomId, eventId, message);
        },
      },
      {
        name: 'attachConfig',
        handler: async (roomId: string, filename: string, content: string, mimeType: string) => {
          await this.client.attachConfig(roomId, filename, content, mimeType);
        },
      },
    ];
  }
}

// Factory function for OpenClaw
export function createArmorClawChannel(config?: ArmorClawConfig): ChannelPlugin {
  return new ArmorClawChannelPlugin(config);
}

export default ArmorClawChannelPlugin;
