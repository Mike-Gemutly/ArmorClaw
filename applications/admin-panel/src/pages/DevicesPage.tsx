import React, { useState } from 'react';
import {
  Smartphone,
  Monitor,
  Tablet,
  CheckCircle,
  XCircle,
  Clock,
  AlertTriangle,
  Shield,
  Trash2,
  Eye
} from 'lucide-react';

interface Device {
  id: string;
  name: string;
  type: 'phone' | 'tablet' | 'desktop';
  platform: string;
  trustState: 'verified' | 'unverified' | 'pending_approval' | 'rejected';
  lastSeen: Date;
  firstSeen: Date;
  ipAddress: string;
  userAgent: string;
  isCurrent: boolean;
}

const MOCK_DEVICES: Device[] = [
  {
    id: '1',
    name: 'Samsung Galaxy S24',
    type: 'phone',
    platform: 'Android 14',
    trustState: 'verified',
    lastSeen: new Date(),
    firstSeen: new Date('2026-01-10'),
    ipAddress: '192.168.1.100',
    userAgent: 'ArmorChat/1.0 (Android 14)',
    isCurrent: true
  },
  {
    id: '2',
    name: 'Desktop Chrome',
    type: 'desktop',
    platform: 'Windows 11',
    trustState: 'pending_approval',
    lastSeen: new Date(Date.now() - 5 * 60 * 1000),
    firstSeen: new Date(Date.now() - 5 * 60 * 1000),
    ipAddress: '192.168.1.105',
    userAgent: 'Element-X/1.0 (Windows)',
    isCurrent: false
  }
];

export function DevicesPage() {
  const [devices, setDevices] = useState<Device[]>(MOCK_DEVICES);
  const [selectedDevice, setSelectedDevice] = useState<Device | null>(null);

  const getDeviceIcon = (type: Device['type']) => {
    switch (type) {
      case 'phone': return <Smartphone className="w-5 h-5" />;
      case 'tablet': return <Tablet className="w-5 h-5" />;
      case 'desktop': return <Monitor className="w-5 h-5" />;
    }
  };

  const getTrustBadge = (state: Device['trustState']) => {
    switch (state) {
      case 'verified':
        return (
          <span className="flex items-center gap-1 px-2 py-0.5 bg-green-500/20 text-green-400 rounded text-xs">
            <CheckCircle className="w-3 h-3" />
            Verified
          </span>
        );
      case 'unverified':
        return (
          <span className="flex items-center gap-1 px-2 py-0.5 bg-gray-500/20 text-gray-400 rounded text-xs">
            Unverified
          </span>
        );
      case 'pending_approval':
        return (
          <span className="flex items-center gap-1 px-2 py-0.5 bg-yellow-500/20 text-yellow-400 rounded text-xs">
            <Clock className="w-3 h-3" />
            Pending
          </span>
        );
      case 'rejected':
        return (
          <span className="flex items-center gap-1 px-2 py-0.5 bg-red-500/20 text-red-400 rounded text-xs">
            <XCircle className="w-3 h-3" />
            Rejected
          </span>
        );
    }
  };

  const approveDevice = (id: string) => {
    setDevices(prev => prev.map(d =>
      d.id === id ? { ...d, trustState: 'verified' as const } : d
    ));
    setSelectedDevice(null);
  };

  const rejectDevice = (id: string) => {
    setDevices(prev => prev.map(d =>
      d.id === id ? { ...d, trustState: 'rejected' as const } : d
    ));
    setSelectedDevice(null);
  };

  const removeDevice = (id: string) => {
    setDevices(prev => prev.filter(d => d.id !== id));
    setSelectedDevice(null);
  };

  const pendingCount = devices.filter(d => d.trustState === 'pending_approval').length;
  const verifiedCount = devices.filter(d => d.trustState === 'verified').length;

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Device Management</h1>
          <p className="text-gray-400">
            Verify and manage trusted devices
          </p>
        </div>
        <div className="text-right">
          <p className="text-sm text-gray-400">Verified Devices</p>
          <p className="text-2xl font-bold">{verifiedCount}/{devices.length}</p>
        </div>
      </div>

      {/* Pending Approval Banner */}
      {pendingCount > 0 && (
        <div className="bg-yellow-900/20 border border-yellow-500/50 rounded-lg p-4 flex items-center gap-3">
          <AlertTriangle className="w-5 h-5 text-yellow-400" />
          <div className="flex-1">
            <h3 className="font-semibold text-yellow-400">
              {pendingCount} Device{pendingCount > 1 ? 's' : ''} Pending Approval
            </h3>
            <p className="text-sm text-yellow-300/80">
              Review and approve or reject new device connections
            </p>
          </div>
        </div>
      )}

      {/* Device List */}
      <div className="space-y-3">
        {devices.map(device => (
          <div
            key={device.id}
            className={`bg-gray-800/50 rounded-lg p-4 ${
              device.trustState === 'pending_approval' ? 'border border-yellow-500/30' : ''
            }`}
          >
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-3">
                <div className={`p-2 rounded-lg ${
                  device.trustState === 'verified'
                    ? 'bg-green-500/20 text-green-400'
                    : device.trustState === 'pending_approval'
                    ? 'bg-yellow-500/20 text-yellow-400'
                    : 'bg-gray-700 text-gray-400'
                }`}>
                  {getDeviceIcon(device.type)}
                </div>
                <div>
                  <div className="flex items-center gap-2">
                    <h3 className="font-semibold">{device.name}</h3>
                    {device.isCurrent && (
                      <span className="px-1.5 py-0.5 bg-blue-500/20 text-blue-400 text-xs rounded">
                        Current
                      </span>
                    )}
                    {getTrustBadge(device.trustState)}
                  </div>
                  <p className="text-sm text-gray-400">
                    {device.platform} • {device.ipAddress}
                  </p>
                </div>
              </div>

              <div className="flex items-center gap-2">
                <button
                  onClick={() => setSelectedDevice(device)}
                  className="p-2 text-gray-400 hover:text-white hover:bg-gray-700 rounded-lg transition-colors"
                >
                  <Eye className="w-4 h-4" />
                </button>
                {!device.isCurrent && device.trustState !== 'pending_approval' && (
                  <button
                    onClick={() => removeDevice(device.id)}
                    className="p-2 text-gray-400 hover:text-red-400 hover:bg-red-500/10 rounded-lg transition-colors"
                  >
                    <Trash2 className="w-4 h-4" />
                  </button>
                )}
              </div>
            </div>

            {/* Pending Actions */}
            {device.trustState === 'pending_approval' && (
              <div className="flex gap-2 mt-3 pt-3 border-t border-gray-700">
                <button
                  onClick={() => approveDevice(device.id)}
                  className="flex-1 py-2 bg-green-500 hover:bg-green-600 rounded-lg text-sm font-medium transition-colors flex items-center justify-center gap-2"
                >
                  <CheckCircle className="w-4 h-4" />
                  Approve
                </button>
                <button
                  onClick={() => rejectDevice(device.id)}
                  className="flex-1 py-2 bg-red-500 hover:bg-red-600 rounded-lg text-sm font-medium transition-colors flex items-center justify-center gap-2"
                >
                  <XCircle className="w-4 h-4" />
                  Reject
                </button>
              </div>
            )}
          </div>
        ))}
      </div>

      {/* Trust Information */}
      <div className="bg-gray-800/50 rounded-lg p-4">
        <h3 className="font-semibold mb-3 flex items-center gap-2">
          <Shield className="w-4 h-4 text-blue-400" />
          Device Trust Levels
        </h3>
        <div className="space-y-2 text-sm">
          <div className="flex items-start gap-2">
            <CheckCircle className="w-4 h-4 text-green-400 mt-0.5" />
            <div>
              <span className="font-medium text-green-400">Verified</span>
              <p className="text-gray-400">Device has been approved and can interact with ArmorClaw</p>
            </div>
          </div>
          <div className="flex items-start gap-2">
            <Clock className="w-4 h-4 text-yellow-400 mt-0.5" />
            <div>
              <span className="font-medium text-yellow-400">Pending</span>
              <p className="text-gray-400">New device awaiting admin approval</p>
            </div>
          </div>
          <div className="flex items-start gap-2">
            <XCircle className="w-4 h-4 text-red-400 mt-0.5" />
            <div>
              <span className="font-medium text-red-400">Rejected</span>
              <p className="text-gray-400">Device was denied access and cannot connect</p>
            </div>
          </div>
        </div>
      </div>

      {/* Device Detail Modal */}
      {selectedDevice && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-gray-800 rounded-lg p-6 max-w-md w-full mx-4">
            <div className="flex items-center gap-3 mb-4">
              <div className={`p-2 rounded-lg ${
                selectedDevice.trustState === 'verified'
                  ? 'bg-green-500/20 text-green-400'
                  : 'bg-gray-700 text-gray-400'
              }`}>
                {getDeviceIcon(selectedDevice.type)}
              </div>
              <div>
                <h3 className="text-lg font-semibold">{selectedDevice.name}</h3>
                <p className="text-sm text-gray-400">{selectedDevice.platform}</p>
              </div>
            </div>

            <div className="space-y-3 mb-6">
              <div className="flex justify-between text-sm">
                <span className="text-gray-400">Status</span>
                {getTrustBadge(selectedDevice.trustState)}
              </div>
              <div className="flex justify-between text-sm">
                <span className="text-gray-400">IP Address</span>
                <span className="font-mono">{selectedDevice.ipAddress}</span>
              </div>
              <div className="flex justify-between text-sm">
                <span className="text-gray-400">First Seen</span>
                <span>{selectedDevice.firstSeen.toLocaleDateString()}</span>
              </div>
              <div className="flex justify-between text-sm">
                <span className="text-gray-400">Last Seen</span>
                <span>{selectedDevice.lastSeen.toLocaleString()}</span>
              </div>
              <div className="text-sm">
                <span className="text-gray-400 block mb-1">User Agent</span>
                <code className="text-xs bg-gray-700 px-2 py-1 rounded block">
                  {selectedDevice.userAgent}
                </code>
              </div>
            </div>

            <div className="flex gap-3">
              <button
                onClick={() => setSelectedDevice(null)}
                className="flex-1 px-4 py-2 bg-gray-700 hover:bg-gray-600 rounded-lg transition-colors"
              >
                Close
              </button>
              {selectedDevice.trustState === 'pending_approval' && (
                <>
                  <button
                    onClick={() => approveDevice(selectedDevice.id)}
                    className="flex-1 px-4 py-2 bg-green-500 hover:bg-green-600 rounded-lg transition-colors"
                  >
                    Approve
                  </button>
                  <button
                    onClick={() => rejectDevice(selectedDevice.id)}
                    className="flex-1 px-4 py-2 bg-red-500 hover:bg-red-600 rounded-lg transition-colors"
                  >
                    Reject
                  </button>
                </>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

export default DevicesPage;
