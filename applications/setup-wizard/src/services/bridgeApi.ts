/**
 * Bridge API Client for Setup Wizard
 *
 * Communicates with the ArmorClaw bridge via HTTP proxy to Unix socket.
 */

// Types
export interface LockdownStatus {
  mode: 'lockdown' | 'bonding' | 'configuring' | 'hardening' | 'operational';
  admin_established: boolean;
  setup_complete: boolean;
  security_configured: boolean;
}

export interface DataCategory {
  id: string;
  name: string;
  description: string;
  risk_level: 'high' | 'medium' | 'low';
  permission: 'deny' | 'allow' | 'allow_all';
}

export interface Adapter {
  id: string;
  name: string;
  description: string;
  enabled: boolean;
  status: string;
}

// RPC Client
class BridgeAPIClient {
  private requestId = 0;
  private baseUrl = '/api';

  async rpc<T>(method: string, params?: Record<string, unknown>): Promise<T> {
    this.requestId++;

    const response = await fetch(this.baseUrl, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        jsonrpc: '2.0',
        id: this.requestId,
        method,
        params,
      }),
    });

    if (!response.ok) {
      throw new Error(`HTTP ${response.status}: ${response.statusText}`);
    }

    const data = await response.json();

    if (data.error) {
      throw new Error(data.error.message);
    }

    return data.result;
  }

  // Lockdown
  async getLockdownStatus(): Promise<LockdownStatus> {
    return this.rpc<LockdownStatus>('lockdown.status');
  }

  async getChallenge(): Promise<{ nonce: string; expires_at: string }> {
    return this.rpc('lockdown.get_challenge');
  }

  async claimOwnership(params: {
    display_name: string;
    device_name: string;
    device_fingerprint: string;
    passphrase_commitment: string;
    challenge_response?: string;
  }): Promise<{
    status: string;
    admin_id: string;
    device_id: string;
    session_token: string;
    next_step: string;
  }> {
    return this.rpc('lockdown.claim_ownership', params);
  }

  async transitionMode(target: string): Promise<{ success: boolean }> {
    return this.rpc('lockdown.transition', { target });
  }

  // Security
  async getSecurityCategories(): Promise<DataCategory[]> {
    return this.rpc<DataCategory[]>('security.get_categories');
  }

  async setSecurityCategory(category: string, permission: string): Promise<{ success: boolean }> {
    return this.rpc('security.set_category', { category, permission });
  }

  // Adapters
  async listAdapters(): Promise<Adapter[]> {
    return this.rpc<Adapter[]>('adapter.list');
  }

  async enableAdapter(adapterId: string): Promise<{ success: boolean }> {
    return this.rpc('adapter.enable', { adapter_id: adapterId });
  }

  async disableAdapter(adapterId: string): Promise<{ success: boolean }> {
    return this.rpc('adapter.disable', { adapter_id: adapterId });
  }

  // Secrets
  async prepareSecret(params: {
    provider: string;
    name: string;
  }): Promise<{ token: string; expires_at: string }> {
    return this.rpc('secrets.prepare_add', params);
  }

  async submitSecret(params: {
    token: string;
    secret_value: string;
  }): Promise<{ success: boolean; key_id: string }> {
    return this.rpc('secrets.submit', params);
  }
}

export const bridgeApi = new BridgeAPIClient();

// Utility functions
export async function hashPassphrase(passphrase: string): Promise<string> {
  const encoder = new TextEncoder();
  const data = encoder.encode(passphrase);
  const hashBuffer = await crypto.subtle.digest('SHA-256', data);
  const hashArray = Array.from(new Uint8Array(hashBuffer));
  return hashArray.map(b => b.toString(16).padStart(2, '0')).join('');
}

export async function generateDeviceFingerprint(deviceName: string): Promise<string> {
  const data = `${deviceName}:${navigator.userAgent}:${Date.now()}`;
  const encoder = new TextEncoder();
  const hashBuffer = await crypto.subtle.digest('SHA-256', encoder.encode(data));
  const hashArray = Array.from(new Uint8Array(hashBuffer));
  return hashArray.map(b => b.toString(16).padStart(2, '0')).join('');
}
