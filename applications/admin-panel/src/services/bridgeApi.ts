/**
 * Bridge API Client
 *
 * Communicates with the ArmorClaw bridge via HTTP proxy to Unix socket.
 * All requests go through /api/* which is proxied to the bridge.
 */

// Types
export interface LockdownStatus {
  mode: 'lockdown' | 'bonding' | 'configuring' | 'hardening' | 'operational';
  admin_established: boolean;
  single_device_mode: boolean;
  allowed_communication: string[];
  setup_complete: boolean;
  security_configured: boolean;
  keystore_initialized: boolean;
  secrets_injected: boolean;
  hardening_complete: boolean;
}

export interface DataCategory {
  id: string;
  name: string;
  description: string;
  examples: string[];
  risk_level: 'high' | 'medium' | 'low';
  permission: 'deny' | 'allow' | 'allow_all';
  allowed_websites: string[];
  requires_approval: boolean;
}

export interface SecurityTier {
  id: string;
  name: string;
  description: string;
  restrictions: string[];
}

export interface Device {
  id: string;
  name: string;
  type: 'phone' | 'tablet' | 'desktop';
  platform: string;
  trust_state: 'verified' | 'unverified' | 'pending_approval' | 'rejected';
  last_seen: string;
  first_seen: string;
  ip_address: string;
  user_agent: string;
  is_current: boolean;
}

export interface Adapter {
  id: string;
  name: string;
  description: string;
  status: 'connected' | 'disconnected' | 'error' | 'pending';
  enabled: boolean;
  permissions: string[];
  last_sync?: string;
  error?: string;
}

export interface Invite {
  id: string;
  code: string;
  role: 'admin' | 'moderator' | 'user';
  created_by: string;
  created_at: string;
  expires_at?: string;
  max_uses: number;
  use_count: number;
  status: 'active' | 'used' | 'expired' | 'revoked' | 'exhausted';
}

export interface APIKey {
  id: string;
  provider: string;
  name: string;
  prefix: string;
  created_at: string;
  last_used?: string;
  status: 'active' | 'revoked' | 'expired';
}

export interface AuditEvent {
  id: string;
  timestamp: string;
  category: 'security' | 'auth' | 'data' | 'adapter' | 'system';
  action: string;
  actor: string;
  resource: string;
  outcome: 'success' | 'failure' | 'warning';
  details: string;
  ip_address: string;
  device_id?: string;
}

export interface ClaimRequest {
  matrix_user_id: string;
  device_id: string;
  user_agent: string;
  ip_address: string;
  device_name: string;
}

export interface ClaimResponse {
  id: string;
  token: string;
  challenge: string;
  challenge_type: string;
  status: string;
  created_at: string;
  expires_at: string;
}

export interface QRResult {
  token: {
    id: string;
    token: string;
    type: string;
    state: string;
    expires_at: string;
  };
  url: string;
  deep_link: string;
  qr_image: string; // Base64 encoded
  expires_at: string;
}

// RPC Request/Response types
interface RPCRequest {
  jsonrpc: '2.0';
  id: number;
  method: string;
  params?: Record<string, unknown>;
}

interface RPCResponse<T> {
  jsonrpc: '2.0';
  id: number;
  result?: T;
  error?: {
    code: number;
    message: string;
    data?: unknown;
  };
}

// API Client
class BridgeAPIClient {
  private requestId = 0;
  private baseUrl = '/api';

  async rpc<T>(method: string, params?: Record<string, unknown>): Promise<T> {
    this.requestId++;

    const request: RPCRequest = {
      jsonrpc: '2.0',
      id: this.requestId,
      method,
      params,
    };

    const response = await fetch(this.baseUrl, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(request),
    });

    if (!response.ok) {
      throw new Error(`HTTP error: ${response.status}`);
    }

    const data: RPCResponse<T> = await response.json();

    if (data.error) {
      throw new Error(data.error.message);
    }

    return data.result as T;
  }

  // Lockdown methods
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
    certificate: string;
    session_token: string;
    expires_at: string;
    next_step: string;
  }> {
    return this.rpc('lockdown.claim_ownership', params);
  }

  async transitionMode(target: string): Promise<{ success: boolean; mode: string }> {
    return this.rpc('lockdown.transition', { target });
  }

  // Security methods
  async getSecurityCategories(): Promise<DataCategory[]> {
    return this.rpc<DataCategory[]>('security.get_categories');
  }

  async setSecurityCategory(category: string, permission: string): Promise<{ success: boolean }> {
    return this.rpc('security.set_category', { category, permission });
  }

  async getSecurityTiers(): Promise<SecurityTier[]> {
    return this.rpc<SecurityTier[]>('security.get_tiers');
  }

  async setSecurityTier(tier: string): Promise<{ success: boolean }> {
    return this.rpc('security.set_tier', { tier });
  }

  // Device methods
  async listDevices(): Promise<Device[]> {
    return this.rpc<Device[]>('device.list');
  }

  async getDevice(deviceId: string): Promise<Device> {
    return this.rpc('device.get', { device_id: deviceId });
  }

  async approveDevice(deviceId: string, approvedBy: string): Promise<{ success: boolean }> {
    return this.rpc('device.approve', { device_id: deviceId, approved_by: approvedBy });
  }

  async rejectDevice(deviceId: string, rejectedBy: string, reason: string): Promise<{ success: boolean }> {
    return this.rpc('device.reject', { device_id: deviceId, rejected_by: rejectedBy, reason });
  }

  // Adapter methods
  async listAdapters(): Promise<Adapter[]> {
    return this.rpc<Adapter[]>('adapter.list');
  }

  async enableAdapter(adapterId: string): Promise<{ success: boolean }> {
    return this.rpc('adapter.enable', { adapter_id: adapterId });
  }

  async disableAdapter(adapterId: string): Promise<{ success: boolean }> {
    return this.rpc('adapter.disable', { adapter_id: adapterId });
  }

  async configureAdapter(adapterId: string, config: Record<string, unknown>): Promise<{ success: boolean }> {
    return this.rpc('adapter.configure', { adapter_id: adapterId, config });
  }

  // Invite methods
  async createInvite(params: {
    role: string;
    expiration: string;
    max_uses: number;
    welcome_message?: string;
    created_by: string;
  }): Promise<Invite> {
    return this.rpc('invite.create', params);
  }

  async listInvites(): Promise<Invite[]> {
    return this.rpc<Invite[]>('invite.list');
  }

  async revokeInvite(inviteId: string, revokedBy: string): Promise<{ success: boolean }> {
    return this.rpc('invite.revoke', { invite_id: inviteId, revoked_by: revokedBy });
  }

  async validateInvite(code: string): Promise<Invite> {
    return this.rpc('invite.validate', { code });
  }

  // QR methods
  async generateSetupQR(): Promise<QRResult> {
    return this.rpc('qr.generate_setup');
  }

  async generateBondingQR(challenge: string): Promise<QRResult> {
    return this.rpc('qr.generate_bonding', { challenge });
  }

  async generateSecretQR(provider: string): Promise<QRResult> {
    return this.rpc('qr.generate_secret', { provider });
  }

  async generateInviteQR(inviteCode: string, role: string): Promise<QRResult> {
    return this.rpc('qr.generate_invite', { invite_code: inviteCode, role });
  }

  // Admin claim methods (for Element X integration)
  async initiateAdminClaim(params: ClaimRequest): Promise<ClaimResponse> {
    return this.rpc('admin.initiate_claim', params);
  }

  async validateAdminToken(token: string): Promise<ClaimResponse> {
    return this.rpc('admin.validate_token', { token });
  }

  async respondChallenge(token: string, response: string): Promise<{ success: boolean }> {
    return this.rpc('admin.respond_challenge', { token, response });
  }

  async approveClaim(claimId: string, approvedBy: string): Promise<{ success: boolean }> {
    return this.rpc('admin.approve_claim', { claim_id: claimId, approved_by: approvedBy });
  }

  async rejectClaim(claimId: string, rejectedBy: string, reason: string): Promise<{ success: boolean }> {
    return this.rpc('admin.reject_claim', { claim_id: claimId, rejected_by: rejectedBy, reason });
  }

  // Audit methods
  async getAuditLog(params?: {
    category?: string;
    search?: string;
    limit?: number;
    offset?: number;
  }): Promise<{ events: AuditEvent[]; total: number }> {
    return this.rpc('audit.get_log', params);
  }
}

// Export singleton instance
export const bridgeApi = new BridgeAPIClient();

// React Query hooks
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';

// Lockdown hooks
export function useLockdownStatus() {
  return useQuery({
    queryKey: ['lockdown', 'status'],
    queryFn: () => bridgeApi.getLockdownStatus(),
    refetchInterval: 5000, // Refresh every 5 seconds during setup
  });
}

// Security hooks
export function useSecurityCategories() {
  return useQuery({
    queryKey: ['security', 'categories'],
    queryFn: () => bridgeApi.getSecurityCategories(),
  });
}

export function useSetSecurityCategory() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ category, permission }: { category: string; permission: string }) =>
      bridgeApi.setSecurityCategory(category, permission),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['security', 'categories'] });
    },
  });
}

// Device hooks
export function useDevices() {
  return useQuery({
    queryKey: ['devices'],
    queryFn: () => bridgeApi.listDevices(),
  });
}

export function useApproveDevice() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ deviceId, approvedBy }: { deviceId: string; approvedBy: string }) =>
      bridgeApi.approveDevice(deviceId, approvedBy),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['devices'] });
    },
  });
}

// Adapter hooks
export function useAdapters() {
  return useQuery({
    queryKey: ['adapters'],
    queryFn: () => bridgeApi.listAdapters(),
  });
}

export function useToggleAdapter() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ adapterId, enabled }: { adapterId: string; enabled: boolean }) =>
      enabled
        ? bridgeApi.enableAdapter(adapterId)
        : bridgeApi.disableAdapter(adapterId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['adapters'] });
    },
  });
}

// Invite hooks
export function useInvites() {
  return useQuery({
    queryKey: ['invites'],
    queryFn: () => bridgeApi.listInvites(),
  });
}

export function useCreateInvite() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (params: Parameters<typeof bridgeApi.createInvite>[0]) =>
      bridgeApi.createInvite(params),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['invites'] });
    },
  });
}

// QR hooks
export function useSetupQR() {
  return useQuery({
    queryKey: ['qr', 'setup'],
    queryFn: () => bridgeApi.generateSetupQR(),
    staleTime: 60000, // QR valid for 1 minute
  });
}

// Audit hooks
export function useAuditLog(params?: { category?: string; search?: string }) {
  return useQuery({
    queryKey: ['audit', params],
    queryFn: () => bridgeApi.getAuditLog(params),
  });
}
