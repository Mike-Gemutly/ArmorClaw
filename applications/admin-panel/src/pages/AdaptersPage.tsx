import React, { useState } from 'react';
import {
  MessageSquare,
  Slack,
  Hash,
  Users,
  Phone,
  ToggleLeft,
  ToggleRight,
  Settings,
  ExternalLink,
  AlertTriangle,
  CheckCircle
} from 'lucide-react';

interface Adapter {
  id: string;
  name: string;
  icon: React.ReactNode;
  description: string;
  status: 'connected' | 'disconnected' | 'error' | 'pending';
  enabled: boolean;
  permissions: string[];
  lastSync?: Date;
  error?: string;
}

const ADAPTERS: Adapter[] = [
  {
    id: 'slack',
    name: 'Slack',
    icon: <Slack className="w-6 h-6" />,
    description: 'Connect to Slack workspaces for team messaging',
    status: 'disconnected',
    enabled: false,
    permissions: ['read_messages', 'send_messages', 'read_files']
  },
  {
    id: 'discord',
    name: 'Discord',
    icon: <Hash className="w-6 h-6" />,
    description: 'Connect to Discord servers and channels',
    status: 'disconnected',
    enabled: false,
    permissions: ['read_messages', 'send_messages', 'manage_channels']
  },
  {
    id: 'teams',
    name: 'Microsoft Teams',
    icon: <Users className="w-6 h-6" />,
    description: 'Connect to Microsoft Teams for enterprise communication',
    status: 'disconnected',
    enabled: false,
    permissions: ['read_messages', 'send_messages', 'read_files', 'join_meetings']
  },
  {
    id: 'whatsapp',
    name: 'WhatsApp',
    icon: <Phone className="w-6 h-6" />,
    description: 'Connect to WhatsApp for personal messaging',
    status: 'disconnected',
    enabled: false,
    permissions: ['read_messages', 'send_messages']
  }
];

export function AdaptersPage() {
  const [adapters, setAdapters] = useState<Adapter[]>(ADAPTERS);
  const [selectedAdapter, setSelectedAdapter] = useState<string | null>(null);

  const toggleAdapter = (id: string) => {
    setAdapters(prev => prev.map(adapter => {
      if (adapter.id === id) {
        return {
          ...adapter,
          enabled: !adapter.enabled,
          status: !adapter.enabled ? 'pending' : 'disconnected'
        };
      }
      return adapter;
    }));
  };

  const getStatusColor = (status: Adapter['status']) => {
    switch (status) {
      case 'connected': return 'text-green-400';
      case 'disconnected': return 'text-gray-400';
      case 'error': return 'text-red-400';
      case 'pending': return 'text-yellow-400';
    }
  };

  const getStatusIcon = (status: Adapter['status']) => {
    switch (status) {
      case 'connected': return <CheckCircle className="w-4 h-4 text-green-400" />;
      case 'disconnected': return <div className="w-4 h-4 rounded-full border-2 border-gray-500" />;
      case 'error': return <AlertTriangle className="w-4 h-4 text-red-400" />;
      case 'pending': return <div className="w-4 h-4 rounded-full bg-yellow-400 animate-pulse" />;
    }
  };

  const enabledCount = adapters.filter(a => a.enabled).length;

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Adapters</h1>
          <p className="text-gray-400">
            Connect ArmorClaw to external messaging platforms
          </p>
        </div>
        <div className="text-right">
          <p className="text-sm text-gray-400">Enabled</p>
          <p className="text-2xl font-bold">{enabledCount}/{adapters.length}</p>
        </div>
      </div>

      {/* Warning Banner */}
      {enabledCount === 0 && (
        <div className="bg-yellow-900/20 border border-yellow-500/50 rounded-lg p-4 flex items-center gap-3">
          <AlertTriangle className="w-5 h-5 text-yellow-400" />
          <div>
            <h3 className="font-semibold text-yellow-400">No Adapters Enabled</h3>
            <p className="text-sm text-yellow-300/80">
              Enable at least one adapter to allow ArmorClaw to communicate externally
            </p>
          </div>
        </div>
      )}

      {/* Adapter Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        {adapters.map(adapter => (
          <div
            key={adapter.id}
            className={`bg-gray-800/50 rounded-lg p-4 border-2 transition-colors ${
              adapter.enabled
                ? 'border-blue-500/50'
                : 'border-transparent hover:border-gray-700'
            }`}
          >
            <div className="flex items-start justify-between mb-3">
              <div className="flex items-center gap-3">
                <div className={`p-2 rounded-lg ${
                  adapter.enabled ? 'bg-blue-500/20 text-blue-400' : 'bg-gray-700 text-gray-400'
                }`}>
                  {adapter.icon}
                </div>
                <div>
                  <h3 className="font-semibold">{adapter.name}</h3>
                  <p className="text-sm text-gray-400">{adapter.description}</p>
                </div>
              </div>
              <button
                onClick={() => toggleAdapter(adapter.id)}
                className="p-1"
              >
                {adapter.enabled ? (
                  <ToggleRight className="w-8 h-8 text-blue-400" />
                ) : (
                  <ToggleLeft className="w-8 h-8 text-gray-500" />
                )}
              </button>
            </div>

            {/* Status */}
            <div className="flex items-center gap-2 mb-3">
              {getStatusIcon(adapter.status)}
              <span className={`text-sm capitalize ${getStatusColor(adapter.status)}`}>
                {adapter.status}
              </span>
              {adapter.lastSync && (
                <span className="text-xs text-gray-500 ml-auto">
                  Last sync: {adapter.lastSync.toLocaleString()}
                </span>
              )}
            </div>

            {/* Permissions Preview */}
            {adapter.enabled && (
              <div className="border-t border-gray-700 pt-3 mt-3">
                <p className="text-xs text-gray-400 mb-2">Permissions:</p>
                <div className="flex flex-wrap gap-1">
                  {adapter.permissions.map(perm => (
                    <span
                      key={perm}
                      className="px-2 py-0.5 bg-gray-700 text-gray-300 rounded text-xs"
                    >
                      {perm.replace('_', ' ')}
                    </span>
                  ))}
                </div>
              </div>
            )}

            {/* Error Message */}
            {adapter.status === 'error' && adapter.error && (
              <div className="mt-3 p-2 bg-red-900/20 border border-red-500/30 rounded text-sm text-red-300">
                {adapter.error}
              </div>
            )}

            {/* Configure Button */}
            {adapter.enabled && adapter.status === 'connected' && (
              <button
                onClick={() => setSelectedAdapter(adapter.id)}
                className="mt-3 w-full py-2 bg-gray-700 hover:bg-gray-600 rounded-lg text-sm font-medium transition-colors flex items-center justify-center gap-2"
              >
                <Settings className="w-4 h-4" />
                Configure
              </button>
            )}

            {/* Connect Button */}
            {adapter.enabled && adapter.status === 'disconnected' && (
              <button
                className="mt-3 w-full py-2 bg-blue-500 hover:bg-blue-600 rounded-lg text-sm font-medium transition-colors flex items-center justify-center gap-2"
              >
                <ExternalLink className="w-4 h-4" />
                Connect
              </button>
            )}
          </div>
        ))}
      </div>

      {/* Data Flow Info */}
      <div className="bg-gray-800/50 rounded-lg p-4">
        <h3 className="font-semibold mb-2">Data Flow Security</h3>
        <div className="text-sm text-gray-400 space-y-2">
          <p>
            All adapter communications are subject to your security configuration.
            Data categories marked as "deny" will never be shared through adapters.
          </p>
          <p>
            Website allowlists are enforced for any data marked as "allow with restrictions".
          </p>
        </div>
      </div>
    </div>
  );
}

export default AdaptersPage;
