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
  Monitor,
  Loader2
} from 'lucide-react';
import { useAuditLog } from '../services/bridgeApi';
import type { AuditEvent } from '../services/bridgeApi';

const CATEGORY_COLORS = {
  security: { bg: 'bg-red-500/20', text: 'text-red-400', icon: Shield },
  auth: { bg: 'bg-blue-500/20', text: 'text-blue-400', icon: Users },
  data: { bg: 'bg-green-500/20', text: 'text-green-400', icon: Key },
  adapter: { bg: 'bg-purple-500/20', text: 'text-purple-400', icon: Activity },
  system: { bg: 'bg-gray-500/20', text: 'text-gray-400', icon: Monitor }
};

function getOutcomeIcon(outcome: AuditEvent['outcome']) {
  switch (outcome) {
    case 'success': return <CheckCircle className="w-4 h-4 text-green-400" />;
    case 'failure': return <XCircle className="w-4 h-4 text-red-400" />;
    case 'warning': return <AlertTriangle className="w-4 h-4 text-yellow-400" />;
  }
}

export function AuditLogPage() {
  const [filter, setFilter] = useState<string>('all');
  const [search, setSearch] = useState('');
  const { data: auditData, isLoading, error } = useAuditLog({
    category: filter !== 'all' ? filter : undefined,
    search: search || undefined,
  });

  if (isLoading) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <Loader2 className="w-8 h-8 text-blue-400 animate-spin" />
        <span className="ml-3 text-gray-400">Loading audit log...</span>
      </div>
    );
  }

  if (error) {
    return (
      <div className="bg-red-900/20 border border-red-500/50 rounded-lg p-6 text-center">
        <AlertTriangle className="w-8 h-8 text-red-400 mx-auto mb-3" />
        <h3 className="font-semibold text-red-400 mb-1">Failed to load audit log</h3>
        <p className="text-sm text-red-300/80">{error instanceof Error ? error.message : 'Unknown error'}</p>
      </div>
    );
  }

  const events = auditData?.events ?? [];
  const total = auditData?.total ?? 0;

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
    total,
    today: events.filter(e =>
      new Date(e.timestamp) > new Date(Date.now() - 24 * 60 * 60 * 1000)
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
                        {getOutcomeIcon(event.outcome)}
                      </div>
                      <p className="text-sm text-gray-300 mb-1">{event.details}</p>
                      <div className="flex items-center gap-4 text-xs text-gray-400">
                        <span className="flex items-center gap-1">
                          <Clock className="w-3 h-3" />
                          {new Date(event.timestamp).toLocaleString()}
                        </span>
                        <span>Actor: {event.actor}</span>
                        <span>IP: {event.ip_address}</span>
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
