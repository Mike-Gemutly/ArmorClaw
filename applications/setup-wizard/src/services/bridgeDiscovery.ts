/**
 * Bridge Discovery Service
 *
 * Discovers ArmorClaw bridges on the local network via mDNS/Bonjour.
 * Falls back to manual IP entry if discovery fails.
 */

export interface DiscoveredBridge {
  name: string;
  host: string;
  port: number;
  ips?: string[];
  version?: string;
  mode?: string;
  hardware?: string;
  txt?: Record<string, string>;
}

export interface DiscoveryResult {
  bridges: DiscoveredBridge[];
  scanned: boolean;
  error?: string;
}

/**
 * DiscoveryClient handles bridge discovery on the local network
 */
export class DiscoveryClient {
  private apiBase: string;
  private timeout: number;

  constructor(apiBase: string = '/api', timeout: number = 5000) {
    this.apiBase = apiBase;
    this.timeout = timeout;
  }

  /**
   * Discover bridges via mDNS through the bridge's discovery endpoint
   *
   * Note: Browsers cannot do mDNS directly. This requires either:
   * 1. A bridge endpoint that returns discovered bridges
   * 2. A native companion app
   * 3. WebRTC-based local discovery
   *
   * For now, we use a bridge endpoint that does the mDNS discovery.
   */
  async discover(): Promise<DiscoveryResult> {
    try {
      const controller = new AbortController();
      const timeoutId = setTimeout(() => controller.abort(), this.timeout);

      const response = await fetch(`${this.apiBase}/discover`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ jsonrpc: '2.0', id: 1, method: 'bridge.discover' }),
        signal: controller.signal,
      });

      clearTimeout(timeoutId);

      if (!response.ok) {
        throw new Error(`HTTP ${response.status}`);
      }

      const data = await response.json();

      if (data.error) {
        throw new Error(data.error.message);
      }

      return {
        bridges: data.result?.bridges || [],
        scanned: true,
      };
    } catch (error) {
      return {
        bridges: [],
        scanned: true,
        error: error instanceof Error ? error.message : 'Discovery failed',
      };
    }
  }

  /**
   * Test connection to a specific bridge
   */
  async testConnection(bridge: DiscoveredBridge): Promise<boolean> {
    try {
      const url = bridge.port === 443
        ? `https://${bridge.host}/api`
        : bridge.port === 80
          ? `http://${bridge.host}/api`
          : `http://${bridge.host}:${bridge.port}/api`;

      const controller = new AbortController();
      const timeoutId = setTimeout(() => controller.abort(), 3000);

      const response = await fetch(url, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ jsonrpc: '2.0', id: 1, method: 'status' }),
        signal: controller.signal,
      });

      clearTimeout(timeoutId);

      return response.ok;
    } catch {
      return false;
    }
  }

  /**
   * Scan common IP ranges for bridges (fallback when mDNS fails)
   */
  async scanLocalNetwork(): Promise<DiscoveryResult> {
    // Get current page's host to determine local network
    const currentHost = window.location.hostname;
    const bridges: DiscoveredBridge[] = [];

    // If we're already connected to a bridge, return it
    if (currentHost && currentHost !== 'localhost' && currentHost !== '127.0.0.1') {
      bridges.push({
        name: 'Current Bridge',
        host: currentHost,
        port: parseInt(window.location.port) || 8080,
      });
    }

    // Try common local IPs
    const commonIPs = this.getCommonLocalIPs();

    for (const ip of commonIPs) {
      if (ip === currentHost) continue;

      const bridge: DiscoveredBridge = {
        name: `Bridge at ${ip}`,
        host: ip,
        port: 8080,
      };

      if (await this.testConnection(bridge)) {
        bridges.push(bridge);
      }
    }

    return {
      bridges,
      scanned: true,
      error: bridges.length === 0 ? 'No bridges found on local network' : undefined,
    };
  }

  /**
   * Get common local IP addresses to scan
   */
  private getCommonLocalIPs(): string[] {
    const ips: string[] = [];

    // Common router/DHCP ranges
    for (let i = 1; i < 255; i++) {
      // 192.168.1.x (most common)
      ips.push(`192.168.1.${i}`);
    }

    // Also try 192.168.0.x
    for (let i = 1; i < 10; i++) {
      ips.push(`192.168.0.${i}`);
    }

    // Try 10.0.0.x
    for (let i = 1; i < 10; i++) {
      ips.push(`10.0.0.${i}`);
    }

    return ips;
  }

  /**
   * Validate manual connection details
   */
  validateManualConnection(host: string, port: string): { valid: boolean; error?: string } {
    if (!host || host.trim() === '') {
      return { valid: false, error: 'Host is required' };
    }

    // Validate IP address
    const ipRegex = /^(\d{1,3}\.){3}\d{1,3}$/;
    if (ipRegex.test(host)) {
      const parts = host.split('.').map(Number);
      if (parts.some(p => p > 255)) {
        return { valid: false, error: 'Invalid IP address' };
      }
    } else {
      // Validate hostname
      const hostnameRegex = /^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$/;
      if (!hostnameRegex.test(host)) {
        return { valid: false, error: 'Invalid hostname' };
      }
    }

    // Validate port
    const portNum = parseInt(port);
    if (isNaN(portNum) || portNum < 1 || portNum > 65535) {
      return { valid: false, error: 'Port must be between 1 and 65535' };
    }

    return { valid: true };
  }
}

/**
 * React hook for bridge discovery
 */
import { useState, useEffect, useCallback } from 'react';

export interface UseDiscoveryResult {
  bridges: DiscoveredBridge[];
  isScanning: boolean;
  error: string | null;
  scan: () => Promise<void>;
  testConnection: (bridge: DiscoveredBridge) => Promise<boolean>;
  selectedBridge: DiscoveredBridge | null;
  setSelectedBridge: (bridge: DiscoveredBridge | null) => void;
}

export function useDiscovery(autoScan: boolean = true): UseDiscoveryResult {
  const [bridges, setBridges] = useState<DiscoveredBridge[]>([]);
  const [isScanning, setIsScanning] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [selectedBridge, setSelectedBridge] = useState<DiscoveredBridge | null>(null);

  const client = new DiscoveryClient();

  const scan = useCallback(async () => {
    setIsScanning(true);
    setError(null);

    // Try mDNS discovery first
    let result = await client.discover();

    // If mDNS fails, try local network scan
    if (result.bridges.length === 0 && result.error) {
      result = await client.scanLocalNetwork();
    }

    setBridges(result.bridges);

    if (result.error && result.bridges.length === 0) {
      setError(result.error);
    }

    // Auto-select first bridge if only one found
    if (result.bridges.length === 1) {
      setSelectedBridge(result.bridges[0]);
    }

    setIsScanning(false);
  }, []);

  const testConnection = useCallback(async (bridge: DiscoveredBridge) => {
    return client.testConnection(bridge);
  }, []);

  useEffect(() => {
    if (autoScan) {
      scan();
    }
  }, [autoScan, scan]);

  return {
    bridges,
    isScanning,
    error,
    scan,
    testConnection,
    selectedBridge,
    setSelectedBridge,
  };
}

// Export singleton client
export const discoveryClient = new DiscoveryClient();
