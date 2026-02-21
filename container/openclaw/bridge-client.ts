/**
 * ArmorClaw Bridge Client for OpenClaw
 *
 * TypeScript client for communicating with the ArmorClaw Local Bridge
 * via Unix domain socket using JSON-RPC 2.0 protocol.
 *
 * This allows OpenClaw to run inside ArmorClaw's hardened container
 * while communicating with Matrix through the secure bridge.
 */

import { createConnection } from 'net';
import { EventEmitter } from 'events';

export interface BridgeConfig {
  socketPath: string;
  timeout?: number;
}

export interface RPCRequest {
  jsonrpc: '2.0';
  id: number;
  method: string;
  params?: Record<string, unknown>;
}

export interface RPCResponse<T = unknown> {
  jsonrpc: '2.0';
  id: number;
  result?: T;
  error?: {
    code: number;
    message: string;
    data?: unknown;
  };
}

export interface MatrixEvent {
  event_id: string;
  room_id: string;
  sender: string;
  type: string;
  content: {
    msgtype?: string;
    body?: string;
    [key: string]: unknown;
  };
  origin_server_ts: number;
}

export interface BridgeStatus {
  version: string;
  state: string;
  socket: string;
  containers: number;
  container_ids: string[];
}

export interface MatrixStatus {
  enabled: boolean;
  connected: boolean;
  user_id?: string;
  device_id?: string;
}

export interface SendResult {
  event_id: string;
  room_id: string;
}

/**
 * ArmorClaw Bridge Client
 *
 * Communicates with the ArmorClaw Local Bridge over Unix sockets.
 */
export class ArmorClawBridgeClient extends EventEmitter {
  private socketPath: string;
  private timeout: number;
  private requestId = 0;
  private pendingRequests = new Map<number, {
    resolve: (value: unknown) => void;
    reject: (error: Error) => void;
    timer: NodeJS.Timeout;
  }>();

  constructor(config: Partial<BridgeConfig> = {}) {
    super();
    this.socketPath = config.socketPath || process.env.ARMORCLAW_BRIDGE_SOCKET || '/run/armorclaw/bridge.sock';
    this.timeout = config.timeout || 30000;
  }

  /**
   * Send a JSON-RPC request to the bridge
   */
  private async sendRequest<T>(method: string, params?: Record<string, unknown>): Promise<T> {
    const id = ++this.requestId;
    const request: RPCRequest = {
      jsonrpc: '2.0',
      id,
      method,
      ...(params && { params }),
    };

    return new Promise((resolve, reject) => {
      const timer = setTimeout(() => {
        this.pendingRequests.delete(id);
        reject(new Error(`Request timeout for method ${method}`));
      }, this.timeout);

      this.pendingRequests.set(id, {
        resolve: resolve as (value: unknown) => void,
        reject,
        timer,
      });

      this._sendRaw(request).catch((error) => {
        clearTimeout(timer);
        this.pendingRequests.delete(id);
        reject(error);
      });
    });
  }

  /**
   * Send raw request over Unix socket
   */
  private async _sendRaw(request: RPCRequest): Promise<void> {
    const socket = createConnection(this.socketPath);
    const requestData = JSON.stringify(request) + '\n';

    socket.on('error', (error) => {
      const pending = this.pendingRequests.get(request.id);
      if (pending) {
        clearTimeout(pending.timer);
        this.pendingRequests.delete(request.id);
        pending.reject(new Error(`Bridge connection failed: ${error.message}`));
      }
    });

    socket.on('data', (data) => {
      try {
        const response: RPCResponse = JSON.parse(data.toString());
        const pending = this.pendingRequests.get(response.id);
        if (pending) {
          clearTimeout(pending.timer);
          this.pendingRequests.delete(response.id);

          if (response.error) {
            pending.reject(new Error(`RPC error ${response.error.code}: ${response.error.message}`));
          } else {
            pending.resolve(response.result);
          }
        }
      } catch {
        // Incomplete response, wait for more data
      }
    });

    socket.write(requestData);
  }

  // =========================================================================
  // Core Methods
  // =========================================================================

  /**
   * Get bridge status and container information
   */
  async status(): Promise<BridgeStatus> {
    return this.sendRequest<BridgeStatus>('status');
  }

  /**
   * Health check endpoint
   */
  async health(): Promise<{ status: string }> {
    return this.sendRequest<{ status: string }>('health');
  }

  // =========================================================================
  // Matrix Methods
  // =========================================================================

  /**
   * Get Matrix connection status
   */
  async matrixStatus(): Promise<MatrixStatus> {
    return this.sendRequest<MatrixStatus>('matrix_status');
  }

  /**
   * Send a message to a Matrix room
   */
  async matrixSend(roomId: string, message: string, msgtype = 'm.text'): Promise<SendResult> {
    return this.sendRequest<SendResult>('matrix_send', {
      room_id: roomId,
      message,
      msgtype,
    });
  }

  /**
   * Receive pending Matrix events
   */
  async matrixReceive(roomId?: string, limit = 10): Promise<MatrixEvent[]> {
    return this.sendRequest<MatrixEvent[]>('matrix_receive', {
      room_id: roomId,
      limit,
    });
  }

  /**
   * Send a reaction to a Matrix event
   */
  async matrixReact(roomId: string, eventId: string, emoji: string): Promise<SendResult> {
    return this.sendRequest<SendResult>('matrix_react', {
      room_id: roomId,
      event_id: eventId,
      emoji,
    });
  }

  /**
   * Send a reply to a Matrix event
   */
  async matrixReply(roomId: string, eventId: string, message: string): Promise<SendResult> {
    return this.sendRequest<SendResult>('matrix_reply', {
      room_id: roomId,
      event_id: eventId,
      message,
    });
  }

  // =========================================================================
  // Config Attachment Methods
  // =========================================================================

  /**
   * Attach a config file to a Matrix message
   */
  async attachConfig(
    roomId: string,
    filename: string,
    content: string,
    mimeType: string
  ): Promise<SendResult> {
    return this.sendRequest<SendResult>('attach_config', {
      room_id: roomId,
      filename,
      content,
      mime_type: mimeType,
    });
  }

  // =========================================================================
  // Secret Management
  // =========================================================================

  /**
   * Get an injected secret value
   */
  async getSecret(key: string): Promise<{ key: string; value?: string }> {
    return this.sendRequest<{ key: string; value?: string }>('get_secret', { key });
  }

  /**
   * List available secrets (keys only, not values)
   */
  async listSecrets(): Promise<{ keys: string[] }> {
    return this.sendRequest<{ keys: string[] }>('list_secrets');
  }

  // =========================================================================
  // Cleanup
  // =========================================================================

  /**
   * Close all pending requests
   */
  close(): void {
    for (const [id, pending] of this.pendingRequests) {
      clearTimeout(pending.timer);
      pending.reject(new Error('Client closed'));
    }
    this.pendingRequests.clear();
  }
}

// Default export
export default ArmorClawBridgeClient;

// Singleton instance for convenience
let defaultClient: ArmorClawBridgeClient | null = null;

export function getDefaultClient(): ArmorClawBridgeClient {
  if (!defaultClient) {
    defaultClient = new ArmorClawBridgeClient();
  }
  return defaultClient;
}
