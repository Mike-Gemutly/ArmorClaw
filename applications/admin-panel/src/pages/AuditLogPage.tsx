import React, { useState } from 'react';
import {
  Activity,
  Shield,
  Key,
  Users,
  AlertTriangle,
  CheckCircle,
  XCircle,
  Filter,
  Search,
  Download,
  Clock,
  Monitor
} from 'lucide-react';

interface AuditEvent {
  id: string;
  timestamp: Date;
  category: 'security' | 'auth' | 'data' | 'adapter' | 'system';
  action: string;
  actor: string;
  resource: string;
  outcome: 'success' | 'failure' | 'warning';
  details: string;
  ipAddress: string;
  deviceId?: string;
}

const MOCK_EVENTS: AuditEvent[] = [
  {
    id: '1',
    timestamp: new Date(Date.now() - 5 * 60 * 1000),
    category: 'auth',
    action: 'device.verified',
    actor: 'admin',
    resource: 'device:samsung-s24',
    outcome: 'success',
    details: 'Device Samsung Galaxy S24 verified',
    ipAddress: '192.168.1.100'
  },
  {
    id: '2',
    timestamp: new Date(Date.now() - 15 * 60 * 1000),
    category: 'security',
    action: 'config.updated',
    actor: 'admin',
    resource: 'security:banking',
    outcome: 'success',
    details: 'Banking data category changed from deny to allow',
    ipAddress: '192.168.1.100'
  },
  {
    id: '3',
    timestamp: new Date(Date.now() - 30 * 60 * 1000),
    category: 'data',
    action: 'keystore.set',
    actor: 'system',
    resource: 'secret:openai-key',
    outcome: 'success',
    details: 'API key stored securely',
    ipAddress: '127.0.0.1'
  },
  {
    id: '4',
    timestamp: new Date(Date.now() - 45 * 60 * 1000),
    category: 'auth',
    action: 'device.pending',
    actor: 'system',
    resource: 'device:desktop-chrome',
    outcome: 'warning',
    details: 'New device awaiting verification',
    ipAddress: '192.168.1.105'
  },
  {
    id: '5',
    timestamp: new Date(Date.now() - 60 * 60 * 1000),
    category: 'adapter',
    action: 'adapter.enabled',
    actor: 'admin',
    resource: 'adapter:slack',
    outcome: 'success',
    details: 'Slack adapter enabled',
    ipAddress: '192.168.1.100'
  },
  {
    id: '6',
    timestamp: new Date(Date.now() - 2 * 60 * 60 * 1000),
    category: 'security',
    action: 'auth.failed',
    actor: 'unknown',
    resource: 'auth:admin',
    outcome: 'failure',
    details: 'Invalid admin claim attempt',
    ipAddress: '10.0.0.55'
  },
  {
    id: '7',
    timestamp: new Date(Date.now() - 3 * 60 * 60 * 1000),
    category: 'system',
    action: 'lockdown.transition',
    actor: 'system',
    resource: 'state:lockdown',
    outcome: 'success',
    details: 'Transitioned from lockdown to bonding state',
    ipAddress: '127.0.0.1'
  }
];

const CATEGORY_COLORS = {
  security: { bg: 'bg-red-500/20', text: 'text-red-400', icon: Shield },
  auth: { bg: 'bg-blue-500/20', text: 'text-blue-400', icon: Users },
  data: { bg: 'bg-green-500/20', text: 'text-green-400', icon: Key },
  adapter: { bg: 'bg-purple-500/20', text: 'text-purple-400', icon: Activity },
  system: { bg: 'bg-gray-500/20', text: 'text-gray-400', icon: Monitor }
};

const OUTCOME_ICONS = {
  success: <CheckCircle className="w-4 h-4 text-green-400" />,
  failure: <XCircle className="w-4 h-4 text-red-400" />,
  warning: <AlertTriangle className="w-4 h-4 text-yellow-400" />
};

export function AuditLogPage() {
  const [events] = useState<AuditEvent[]>(MOCK_EVENTS);
  const [filter, setFilter] = useState<string>('all');
  const [search, setSearch] = useState('');

  const filteredEvents = events.filter(event => {
    if (filter !== 'all' && event.category !== filter) return false;
    if (search && !event.action.toLowerCase().includes(search.toLowerCase()) &&
        !event.details.toLowerCase().includes(search.toLowerCase())) return false;
    return true;
  });

  const exportLogs = () => {
    const data = JSON.stringify(filteredEvents, null, 2);
    const blob = new Blob([data], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `audit-log-${new Date().toISOString().split('T')[0]}.json`;
    a.click();
    URL.revokeObjectURL(url);
  };

  const stats = {
    total: events.length,
    today: events.filter(e =>
      e.timestamp > new Date(Date.now() - 24 * 60 * 60 * 1000)
    ).length,
    failures: events.filter(e => e.outcome === 'failure').length,
    warnings: events.filter(e => e.outcome === 'warning').length
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Audit Log</h1>
          <p className="text-gray-400">
            Security events and access history
          </p>
        </div>
        <button
          onClick={exportLogs}
          className="px-4 py-2 bg-gray-700 hover:bg-gray-600 rounded-lg text-sm font-medium transition-colors flex items-center gap-2"
        >
          <Download className="w-4 h-4" />
          Export
        </button>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <div className="bg-gray-800/50 rounded-lg p-4">
          <div className="text-2xl font-bold">{stats.total}</div>
          <div className="text-sm text-gray-400">Total Events</div>
        </div>
        <div className="bg-gray-800/50 rounded-lg p-4">
          <div className="text-2xl font-bold">{stats.today}</div>
          <div className="text-sm text-gray-400">Last 24 Hours</div>
        </div>
        <div className="bg-gray-800/50 rounded-lg p-4">
          <div className="text-2xl font-bold text-red-400">{stats.failures}</div>
          <div className="text-sm text-gray-400">Failures</div>
        </div>
        <div className="bg-gray-800/50 rounded-lg p-4">
          <div className="text-2xl font-bold text-yellow-400">{stats.warnings}</div>
          <div className="text-sm text-gray-400">Warnings</div>
        </div>
      </div>

      {/* Filters */}
      <div className="flex flex-col md:flex-row gap-4">
        <div className="flex-1 relative">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
          <input
            type="text"
            placeholder="Search events..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="w-full pl-10 pr-4 py-2 bg-gray-800/50 border border-gray-700 rounded-lg focus:outline-none focus:border-blue-500"
          />
        </div>
        <div className="flex items-center gap-2">
          <Filter className="w-4 h-4 text-gray-400" />
          <select
            value={filter}
            onChange={(e) => setFilter(e.target.value)}
            className="px-3 py-2 bg-gray-800/50 border border-gray-700 rounded-lg focus:outline-none focus:border-blue-500"
          >
            <option value="all">All Categories</option>
            <option value="security">Security</option>
            <option value="auth">Authentication</option>
            <option value="data">Data Access</option>
            <option value="adapter">Adapters</option>
            <option value="system">System</option>
          </select>
        </div>
      </div>

      {/* Event List */}
      <div className="bg-gray-800/50 rounded-lg overflow-hidden">
        {filteredEvents.length === 0 ? (
          <div className="p-8 text-center text-gray-400">
            <Activity className="w-12 h-12 mx-auto mb-3 opacity-50" />
            <p>No events found</p>
          </div>
        ) : (
          <div className="divide-y divide-gray-700">
            {filteredEvents.map(event => {
              const CategoryIcon = CATEGORY_COLORS[event.category].icon;
              return (
                <div key={event.id} className="p-4 hover:bg-gray-700/30 transition-colors">
                  <div className="flex items-start gap-3">
                    <div className={`p-2 rounded-lg ${CATEGORY_COLORS[event.category].bg}`}>
                      <CategoryIcon className={`w-4 h-4 ${CATEGORY_COLORS[event.category].text}`} />
                    </div>
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2 mb-1">
                        <code className="text-sm font-mono text-blue-300">
                          {event.action}
                        </code>
                        {OUTCOME_ICONS[event.outcome]}
                      </div>
                      <p className="text-sm text-gray-300 mb-1">{event.details}</p>
                      <div className="flex items-center gap-4 text-xs text-gray-400">
                        <span className="flex items-center gap-1">
                          <Clock className="w-3 h-3" />
                          {event.timestamp.toLocaleString()}
                        </span>
                        <span>Actor: {event.actor}</span>
                        <span>IP: {event.ipAddress}</span>
                      </div>
                    </div>
                    <div className="text-right text-xs">
                      <div className={`px-2 py-0.5 rounded ${CATEGORY_COLORS[event.category].bg} ${CATEGORY_COLORS[event.category].text}`}>
                        {event.category}
                      </div>
                    </div>
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </div>

      {/* Retention Notice */}
      <div className="bg-gray-800/50 rounded-lg p-4">
        <div className="flex items-center gap-2 text-sm text-gray-400">
          <Clock className="w-4 h-4" />
          <p>
            Audit logs are retained for 90 days. For compliance requirements, use the Export
            function to archive logs externally.
          </p>
        </div>
      </div>
    </div>
  );
}

export default AuditLogPage;
