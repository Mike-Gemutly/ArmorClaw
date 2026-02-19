/**
 * Bridge Discovery Component
 *
 * UI for discovering and selecting ArmorClaw bridges on the local network.
 */

import React, { useState } from 'react';
import {
  Wifi,
  RefreshCw,
  CheckCircle,
  AlertCircle,
  ChevronRight,
  Plus,
  X,
  Loader
} from 'lucide-react';
import {
  DiscoveredBridge,
  DiscoveryClient,
  useDiscovery
} from '../services/bridgeDiscovery';

interface BridgeDiscoveryProps {
  onBridgeSelected: (bridge: DiscoveredBridge) => void;
  onManualEntry: (host: string, port: number) => void;
  allowManual?: boolean;
}

export function BridgeDiscovery({
  onBridgeSelected,
  onManualEntry,
  allowManual = true
}: BridgeDiscoveryProps) {
  const {
    bridges,
    isScanning,
    error,
    scan,
    testConnection,
    selectedBridge,
    setSelectedBridge
  } = useDiscovery(true);

  const [showManual, setShowManual] = useState(false);
  const [manualHost, setManualHost] = useState('');
  const [manualPort, setManualPort] = useState('8080');
  const [manualError, setManualError] = useState<string | null>(null);
  const [isTesting, setIsTesting] = useState(false);
  const [testingBridge, setTestingBridge] = useState<DiscoveredBridge | null>(null);

  const handleSelectBridge = async (bridge: DiscoveredBridge) => {
    setTestingBridge(bridge);
    setIsTesting(true);

    const connected = await testConnection(bridge);

    setIsTesting(false);
    setTestingBridge(null);

    if (connected) {
      setSelectedBridge(bridge);
      onBridgeSelected(bridge);
    }
  };

  const handleManualSubmit = async () => {
    const client = new DiscoveryClient();
    const validation = client.validateManualConnection(manualHost, manualPort);

    if (!validation.valid) {
      setManualError(validation.error || 'Invalid connection details');
      return;
    }

    const bridge: DiscoveredBridge = {
      name: `Manual: ${manualHost}`,
      host: manualHost,
      port: parseInt(manualPort),
    };

    setIsTesting(true);
    setManualError(null);

    const connected = await testConnection(bridge);

    setIsTesting(false);

    if (connected) {
      setSelectedBridge(bridge);
      onManualEntry(manualHost, parseInt(manualPort));
    } else {
      setManualError('Could not connect to bridge at this address');
    }
  };

  return (
    <div className="space-y-4">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h3 className="font-medium">Discover Bridges</h3>
          <p className="text-sm text-gray-400">
            Scanning your local network for ArmorClaw bridges
          </p>
        </div>
        <button
          onClick={scan}
          disabled={isScanning}
          className="p-2 rounded-lg hover:bg-gray-800 disabled:opacity-50"
          title="Rescan"
        >
          <RefreshCw className={`w-5 h-5 ${isScanning ? 'animate-spin' : ''}`} />
        </button>
      </div>

      {/* Scanning indicator */}
      {isScanning && (
        <div className="flex items-center gap-3 p-4 bg-blue-900/20 border border-blue-500/50 rounded-lg">
          <RefreshCw className="w-5 h-5 text-blue-400 animate-spin" />
          <span className="text-blue-300">Scanning for bridges...</span>
        </div>
      )}

      {/* Error */}
      {error && !isScanning && bridges.length === 0 && (
        <div className="flex items-center gap-3 p-4 bg-yellow-900/20 border border-yellow-500/50 rounded-lg">
          <AlertCircle className="w-5 h-5 text-yellow-400" />
          <div className="flex-1">
            <span className="text-yellow-300">{error}</span>
            <p className="text-sm text-yellow-400/80 mt-1">
              Try manual connection below
            </p>
          </div>
        </div>
      )}

      {/* Discovered bridges */}
      {bridges.length > 0 && (
        <div className="space-y-2">
          {bridges.map((bridge, index) => (
            <button
              key={index}
              onClick={() => handleSelectBridge(bridge)}
              disabled={isTesting}
              className={`
                w-full p-4 rounded-lg border-2 text-left transition-colors
                ${selectedBridge?.host === bridge.host
                  ? 'border-blue-500 bg-blue-500/10'
                  : 'border-gray-700 hover:border-gray-600'
                }
                ${isTesting && testingBridge?.host === bridge.host ? 'opacity-75' : ''}
              `}
            >
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-3">
                  <Wifi className="w-5 h-5 text-green-400" />
                  <div>
                    <div className="font-medium">{bridge.name || bridge.host}</div>
                    <div className="text-sm text-gray-400">
                      {bridge.host}:{bridge.port}
                      {bridge.version && ` • v${bridge.version}`}
                      {bridge.mode && ` • ${bridge.mode}`}
                    </div>
                  </div>
                </div>
                <div className="flex items-center gap-2">
                  {isTesting && testingBridge?.host === bridge.host ? (
                    <Loader className="w-5 h-5 animate-spin text-blue-400" />
                  ) : selectedBridge?.host === bridge.host ? (
                    <CheckCircle className="w-5 h-5 text-green-400" />
                  ) : (
                    <ChevronRight className="w-5 h-5 text-gray-500" />
                  )}
                </div>
              </div>
            </button>
          ))}
        </div>
      )}

      {/* Manual entry */}
      {allowManual && (
        <div className="mt-4">
          {!showManual ? (
            <button
              onClick={() => setShowManual(true)}
              className="w-full p-3 border-2 border-dashed border-gray-700 rounded-lg hover:border-gray-600 flex items-center justify-center gap-2 text-gray-400 hover:text-gray-300"
            >
              <Plus className="w-5 h-5" />
              Enter address manually
            </button>
          ) : (
            <div className="p-4 bg-gray-800/50 rounded-lg border border-gray-700">
              <div className="flex items-center justify-between mb-3">
                <span className="font-medium">Manual Connection</span>
                <button
                  onClick={() => {
                    setShowManual(false);
                    setManualError(null);
                  }}
                  className="text-gray-400 hover:text-gray-300"
                >
                  <X className="w-5 h-5" />
                </button>
              </div>

              <div className="grid grid-cols-3 gap-3">
                <div className="col-span-2">
                  <label className="block text-sm text-gray-400 mb-1">Host</label>
                  <input
                    type="text"
                    value={manualHost}
                    onChange={(e) => setManualHost(e.target.value)}
                    placeholder="192.168.1.100"
                    className="w-full px-3 py-2 bg-gray-700 border border-gray-600 rounded-lg focus:border-blue-500 focus:outline-none"
                  />
                </div>
                <div>
                  <label className="block text-sm text-gray-400 mb-1">Port</label>
                  <input
                    type="text"
                    value={manualPort}
                    onChange={(e) => setManualPort(e.target.value)}
                    placeholder="8080"
                    className="w-full px-3 py-2 bg-gray-700 border border-gray-600 rounded-lg focus:border-blue-500 focus:outline-none"
                  />
                </div>
              </div>

              {manualError && (
                <p className="mt-2 text-sm text-red-400">{manualError}</p>
              )}

              <button
                onClick={handleManualSubmit}
                disabled={isTesting || !manualHost}
                className="mt-3 w-full py-2 bg-blue-500 hover:bg-blue-600 rounded-lg font-medium disabled:opacity-50 disabled:cursor-not-allowed flex items-center justify-center gap-2"
              >
                {isTesting ? (
                  <>
                    <Loader className="w-4 h-4 animate-spin" />
                    Connecting...
                  </>
                ) : (
                  <>
                    Connect
                    <ChevronRight className="w-4 h-4" />
                  </>
                )}
              </button>
            </div>
          )}
        </div>
      )}
    </div>
  );
}

export default BridgeDiscovery;
