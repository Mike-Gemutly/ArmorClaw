import React from 'react';
import {
  Shield,
  Key,
  Users,
  Settings,
  Globe,
  MessageSquare,
  AlertTriangle,
  CheckCircle,
  Clock,
  Activity
} from 'lucide-react';

interface DashboardStats {
  lockdownMode: boolean;
  setupComplete: boolean;
  totalDevices: number;
  pendingVerifications: number;
  activeInvites: number;
  apiKeysCount: number;
  enabledAdapters: string[];
}

export function AdminDashboard({ stats }: { stats: DashboardStats }) {
  return (
    <div className="space-y-6">
      {/* Status Banner */}
      {stats.lockdownMode ? (
        <div className="bg-red-900/20 border border-red-500/50 rounded-lg p-4 flex items-center gap-3">
          <AlertTriangle className="w-6 h-6 text-red-400" />
          <div>
            <h3 className="font-semibold text-red-400">Lockdown Mode Active</h3>
            <p className="text-sm text-red-300/80">
              Complete setup to transition to operational mode
            </p>
          </div>
        </div>
      ) : (
        <div className="bg-green-900/20 border border-green-500/50 rounded-lg p-4 flex items-center gap-3">
          <CheckCircle className="w-6 h-6 text-green-400" />
          <div>
            <h3 className="font-semibold text-green-400">System Operational</h3>
            <p className="text-sm text-green-300/80">
              All security configurations applied
            </p>
          </div>
        </div>
      )}

      {/* Quick Stats */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        <StatCard
          icon={<Users className="w-5 h-5" />}
          label="Connected Devices"
          value={stats.totalDevices}
          color="blue"
        />
        <StatCard
          icon={<Clock className="w-5 h-5" />}
          label="Pending Verifications"
          value={stats.pendingVerifications}
          color={stats.pendingVerifications > 0 ? 'yellow' : 'gray'}
        />
        <StatCard
          icon={<MessageSquare className="w-5 h-5" />}
          label="Active Invites"
          value={stats.activeInvites}
          color="purple"
        />
        <StatCard
          icon={<Key className="w-5 h-5" />}
          label="API Keys"
          value={stats.apiKeysCount}
          color="green"
        />
      </div>

      {/* Setup Progress */}
      {!stats.setupComplete && <SetupProgress />}

      {/* Quick Actions */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        <QuickActionCard
          icon={<Shield className="w-6 h-6" />}
          title="Security Configuration"
          description="Configure data categories, website allowlists, and security tiers"
          href="/security"
        />
        <QuickActionCard
          icon={<Globe className="w-6 h-6" />}
          title="Adapters"
          description="Manage Slack, Discord, Teams, WhatsApp connections"
          href="/adapters"
        />
        <QuickActionCard
          icon={<Key className="w-6 h-6" />}
          title="API Keys"
          description="Add and manage API keys for AI providers"
          href="/secrets"
        />
        <QuickActionCard
          icon={<Users className="w-6 h-6" />}
          title="Device Management"
          description="Verify pending devices and manage trusted devices"
          href="/devices"
        />
        <QuickActionCard
          icon={<MessageSquare className="w-6 h-6" />}
          title="Invitations"
          description="Create and manage team member invitations"
          href="/invites"
        />
        <QuickActionCard
          icon={<Activity className="w-6 h-6" />}
          title="Audit Log"
          description="View security events and access history"
          href="/audit"
        />
      </div>

      {/* Enabled Adapters */}
      <div className="bg-gray-800/50 rounded-lg p-4">
        <h3 className="font-semibold mb-3">Enabled Adapters</h3>
        <div className="flex flex-wrap gap-2">
          {stats.enabledAdapters.length === 0 ? (
            <span className="text-gray-400 text-sm">No adapters enabled</span>
          ) : (
            stats.enabledAdapters.map(adapter => (
              <span
                key={adapter}
                className="px-3 py-1 bg-blue-500/20 text-blue-300 rounded-full text-sm"
              >
                {adapter}
              </span>
            ))
          )}
        </div>
      </div>
    </div>
  );
}

function StatCard({
  icon,
  label,
  value,
  color
}: {
  icon: React.ReactNode;
  label: string;
  value: number;
  color: 'blue' | 'yellow' | 'gray' | 'purple' | 'green';
}) {
  const colorClasses = {
    blue: 'bg-blue-500/10 text-blue-400',
    yellow: 'bg-yellow-500/10 text-yellow-400',
    gray: 'bg-gray-500/10 text-gray-400',
    purple: 'bg-purple-500/10 text-purple-400',
    green: 'bg-green-500/10 text-green-400'
  };

  return (
    <div className="bg-gray-800/50 rounded-lg p-4">
      <div className="flex items-center gap-3">
        <div className={`p-2 rounded-lg ${colorClasses[color]}`}>
          {icon}
        </div>
        <div>
          <p className="text-2xl font-bold">{value}</p>
          <p className="text-sm text-gray-400">{label}</p>
        </div>
      </div>
    </div>
  );
}

function QuickActionCard({
  icon,
  title,
  description,
  href
}: {
  icon: React.ReactNode;
  title: string;
  description: string;
  href: string;
}) {
  return (
    <a
      href={href}
      className="bg-gray-800/50 rounded-lg p-4 hover:bg-gray-700/50 transition-colors group"
    >
      <div className="flex items-start gap-3">
        <div className="p-2 rounded-lg bg-gray-700 text-gray-300 group-hover:bg-blue-500/20 group-hover:text-blue-400 transition-colors">
          {icon}
        </div>
        <div>
          <h3 className="font-semibold group-hover:text-blue-400 transition-colors">
            {title}
          </h3>
          <p className="text-sm text-gray-400">{description}</p>
        </div>
      </div>
    </a>
  );
}

function SetupProgress() {
  const steps = [
    { name: 'Admin Claimed', completed: true },
    { name: 'Security Config', completed: false },
    { name: 'Keystore Init', completed: false },
    { name: 'API Keys Added', completed: false },
    { name: 'Hardening', completed: false }
  ];

  const completedCount = steps.filter(s => s.completed).length;
  const progress = (completedCount / steps.length) * 100;

  return (
    <div className="bg-gray-800/50 rounded-lg p-4">
      <div className="flex items-center justify-between mb-3">
        <h3 className="font-semibold">Setup Progress</h3>
        <span className="text-sm text-gray-400">{completedCount}/{steps.length} complete</span>
      </div>

      {/* Progress bar */}
      <div className="h-2 bg-gray-700 rounded-full mb-4">
        <div
          className="h-full bg-blue-500 rounded-full transition-all duration-300"
          style={{ width: `${progress}%` }}
        />
      </div>

      {/* Steps */}
      <div className="space-y-2">
        {steps.map((step, index) => (
          <div key={step.name} className="flex items-center gap-2">
            {step.completed ? (
              <CheckCircle className="w-4 h-4 text-green-400" />
            ) : (
              <div className="w-4 h-4 rounded-full border-2 border-gray-600" />
            )}
            <span className={step.completed ? 'text-gray-300' : 'text-gray-500'}>
              {step.name}
            </span>
          </div>
        ))}
      </div>
    </div>
  );
}

export default AdminDashboard;
