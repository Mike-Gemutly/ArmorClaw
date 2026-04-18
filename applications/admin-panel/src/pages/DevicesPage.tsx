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
  Eye,
  Loader2
} from 'lucide-react';
import { useDevices, useApproveDevice, useRejectDevice } from '../services/bridgeApi';
import type { Device } from '../services/bridgeApi';

const DEVICE_ICONS = {
  phone: <Smartphone className="w-5 h-5" />,
  tablet: <Tablet className="w-5 h-5" />,
  desktop: <Monitor className="w-5 h-5" />,
};

const TRUST_BADGES: Record<string, React.ReactNode> = {};

function getDeviceIcon(type: Device['type']) {
  return DEVICE_ICONS[type] || <Smartphone className="w-5 h-5" />;
}

function getTrustBadge(state: Device['trust_state']) {
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
    default:
      return null;
  }
}

export function DevicesPage() {
  const { data: devices, isLoading, error } = useDevices();
  const approveMutation = useApproveDevice();
  const rejectMutation = useRejectDevice();
  const [selectedDevice, setSelectedDevice] = useState<Device | null>(null);

  if (isLoading) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <Loader2 className="w-8 h-8 text-blue-400 animate-spin" />
        <span className="ml-3 text-gray-400">Loading devices...</span>
      </div>
    );
  }

  if (error) {
    return (
      <div className="bg-red-900/20 border border-red-500/50 rounded-lg p-6 text-center">
        <AlertTriangle className="w-8 h-8 text-red-400 mx-auto mb-3" />
        <h3 className="font-semibold text-red-400 mb-1">Failed to load devices</h3>
        <p className="text-sm text-red-300/80">{error instanceof Error ? error.message : 'Unknown error'}</p>
      </div>
    );
  }

  const deviceList = devices ?? [];
  const pendingCount = deviceList.filter(d => d.trust_state === 'pending_approval').length;
  const verifiedCount = deviceList.filter(d => d.trust_state === 'verified').length;

  const approveDevice = (id: string) => {
    approveMutation.mutate({ deviceId: id, approvedBy: 'admin' });
    setSelectedDevice(null);
  };

  const rejectDevice = (id: string) => {
    rejectMutation.mutate({ deviceId: id, rejectedBy: 'admin', reason: 'Rejected by admin' });
    setSelectedDevice(null);
  };

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
          <p className="text-2xl font-bold">{verifiedCount}/{deviceList.length}</p>
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
        {deviceList.map(device => (
          <div
            key={device.id}
            className={`bg-gray-800/50 rounded-lg p-4 ${
              device.trust_state === 'pending_approval' ? 'border border-yellow-500/30' : ''
            }`}
          >
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-3">
                <div className={`p-2 rounded-lg ${
                  device.trust_state === 'verified'
                    ? 'bg-green-500/20 text-green-400'
                    : device.trust_state === 'pending_approval'
                    ? 'bg-yellow-500/20 text-yellow-400'
                    : 'bg-gray-700 text-gray-400'
                }`}>
                  {getDeviceIcon(device.type)}
                </div>
                <div>
                  <div className="flex items-center gap-2">
                    <h3 className="font-semibold">{device.name}</h3>
                    {device.is_current && (
                      <span className="px-1.5 py-0.5 bg-blue-500/20 text-blue-400 text-xs rounded">
                        Current
                      </span>
                    )}
                    {getTrustBadge(device.trust_state)}
                  </div>
                  <p className="text-sm text-gray-400">
                    {device.platform} • {device.ip_address}
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
                {!device.is_current && device.trust_state !== 'pending_approval' && (
                  <button
                    onClick={() => rejectDevice(device.id)}
                    className="p-2 text-gray-400 hover:text-red-400 hover:bg-red-500/10 rounded-lg transition-colors"
                  >
                    <Trash2 className="w-4 h-4" />
                  </button>
                )}
              </div>
            </div>

            {/* Pending Actions */}
            {device.trust_state === 'pending_approval' && (
              <div className="flex gap-2 mt-3 pt-3 border-t border-gray-700">
                <button
                  onClick={() => approveDevice(device.id)}
                  disabled={approveMutation.isPending}
                  className="flex-1 py-2 bg-green-500 hover:bg-green-600 disabled:opacity-50 rounded-lg text-sm font-medium transition-colors flex items-center justify-center gap-2"
                >
                  <CheckCircle className="w-4 h-4" />
                  Approve
                </button>
                <button
                  onClick={() => rejectDevice(device.id)}
                  disabled={rejectMutation.isPending}
                  className="flex-1 py-2 bg-red-500 hover:bg-red-600 disabled:opacity-50 rounded-lg text-sm font-medium transition-colors flex items-center justify-center gap-2"
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
                selectedDevice.trust_state === 'verified'
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
                {getTrustBadge(selectedDevice.trust_state)}
              </div>
              <div className="flex justify-between text-sm">
                <span className="text-gray-400">IP Address</span>
                <span className="font-mono">{selectedDevice.ip_address}</span>
              </div>
              <div className="flex justify-between text-sm">
                <span className="text-gray-400">First Seen</span>
                <span>{new Date(selectedDevice.first_seen).toLocaleDateString()}</span>
              </div>
              <div className="flex justify-between text-sm">
                <span className="text-gray-400">Last Seen</span>
                <span>{new Date(selectedDevice.last_seen).toLocaleString()}</span>
              </div>
              <div className="text-sm">
                <span className="text-gray-400 block mb-1">User Agent</span>
                <code className="text-xs bg-gray-700 px-2 py-1 rounded block">
                  {selectedDevice.user_agent}
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
              {selectedDevice.trust_state === 'pending_approval' && (
                <>
                  <button
                    onClick={() => approveDevice(selectedDevice.id)}
                    disabled={approveMutation.isPending}
                    className="flex-1 px-4 py-2 bg-green-500 hover:bg-green-600 disabled:opacity-50 rounded-lg transition-colors"
                  >
                    Approve
                  </button>
                  <button
                    onClick={() => rejectDevice(selectedDevice.id)}
                    disabled={rejectMutation.isPending}
                    className="flex-1 px-4 py-2 bg-red-500 hover:bg-red-600 disabled:opacity-50 rounded-lg transition-colors"
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
