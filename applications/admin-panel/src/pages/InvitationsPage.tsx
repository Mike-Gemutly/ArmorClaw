import React, { useState } from 'react';
import {
  Mail,
  Plus,
  Copy,
  Link,
  Trash2,
  Clock,
  Users,
  Shield,
  CheckCircle,
  XCircle,
  AlertTriangle,
  ExternalLink,
  Loader2
} from 'lucide-react';
import { useInvites, useCreateInvite, useRevokeInvite } from '../services/bridgeApi';
import type { Invite } from '../services/bridgeApi';

const ROLE_INFO = {
  admin: {
    color: 'text-red-400 bg-red-500/20',
    power: 100,
    permissions: ['Full administrative access', 'Security configuration', 'User management']
  },
  moderator: {
    color: 'text-yellow-400 bg-yellow-500/20',
    power: 50,
    permissions: ['Agent management', 'View audit logs', 'Create invites']
  },
  user: {
    color: 'text-blue-400 bg-blue-500/20',
    power: 0,
    permissions: ['Use agents', 'View own data']
  }
};

const EXPIRATION_OPTIONS = [
  { value: '1h', label: '1 Hour' },
  { value: '6h', label: '6 Hours' },
  { value: '1d', label: '1 Day' },
  { value: '7d', label: '7 Days' },
  { value: '30d', label: '30 Days' },
  { value: 'never', label: 'Never' }
];

function getStatusBadge(status: Invite['status']) {
  switch (status) {
    case 'active':
      return (
        <span className="flex items-center gap-1 px-2 py-0.5 bg-green-500/20 text-green-400 rounded text-xs">
          <CheckCircle className="w-3 h-3" />
          Active
        </span>
      );
    case 'used':
      return (
        <span className="flex items-center gap-1 px-2 py-0.5 bg-blue-500/20 text-blue-400 rounded text-xs">
          Used
        </span>
      );
    case 'expired':
      return (
        <span className="flex items-center gap-1 px-2 py-0.5 bg-gray-500/20 text-gray-400 rounded text-xs">
          <Clock className="w-3 h-3" />
          Expired
        </span>
      );
    case 'revoked':
      return (
        <span className="flex items-center gap-1 px-2 py-0.5 bg-red-500/20 text-red-400 rounded text-xs">
          <XCircle className="w-3 h-3" />
          Revoked
        </span>
      );
    case 'exhausted':
      return (
        <span className="flex items-center gap-1 px-2 py-0.5 bg-yellow-500/20 text-yellow-400 rounded text-xs">
          <AlertTriangle className="w-3 h-3" />
          Exhausted
        </span>
      );
  }
}

export function InvitationsPage() {
  const { data: invites, isLoading, error } = useInvites();
  const createMutation = useCreateInvite();
  const revokeMutation = useRevokeInvite();
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [newRole, setNewRole] = useState<Invite['role']>('user');
  const [newExpiration, setNewExpiration] = useState('7d');
  const [newMaxUses, setNewMaxUses] = useState('1');
  const [copiedCode, setCopiedCode] = useState<string | null>(null);

  if (isLoading) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <Loader2 className="w-8 h-8 text-blue-400 animate-spin" />
        <span className="ml-3 text-gray-400">Loading invitations...</span>
      </div>
    );
  }

  if (error) {
    return (
      <div className="bg-red-900/20 border border-red-500/50 rounded-lg p-6 text-center">
        <AlertTriangle className="w-8 h-8 text-red-400 mx-auto mb-3" />
        <h3 className="font-semibold text-red-400 mb-1">Failed to load invitations</h3>
        <p className="text-sm text-red-300/80">{error instanceof Error ? error.message : 'Unknown error'}</p>
      </div>
    );
  }

  const inviteList = invites ?? [];
  const activeCount = inviteList.filter(i => i.status === 'active').length;

  const parseExpiration = (exp: string): number => {
    const units: Record<string, number> = { h: 3600000, d: 86400000 };
    const match = exp.match(/^(\d+)([hd])$/);
    return match ? parseInt(match[1]) * units[match[2]] : 0;
  };

  const createInvite = () => {
    createMutation.mutate({
      role: newRole,
      expiration: newExpiration,
      max_uses: parseInt(newMaxUses),
      created_by: 'admin',
    }, {
      onSuccess: () => {
        setShowCreateModal(false);
        setNewRole('user');
        setNewExpiration('7d');
        setNewMaxUses('1');
      },
    });
  };

  const copyInviteLink = (code: string) => {
    const link = `https://armorclaw.app/invite/${code}`;
    navigator.clipboard.writeText(link);
    setCopiedCode(code);
    setTimeout(() => setCopiedCode(null), 2000);
  };

  const handleRevoke = (id: string) => {
    revokeMutation.mutate({ inviteId: id, revokedBy: 'admin' });
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Invitations</h1>
          <p className="text-gray-400">
            Create and manage team member invitations
          </p>
        </div>
        <div className="flex items-center gap-4">
          <div className="text-right">
            <p className="text-sm text-gray-400">Active Invites</p>
            <p className="text-2xl font-bold">{activeCount}</p>
          </div>
          <button
            onClick={() => setShowCreateModal(true)}
            className="px-4 py-2 bg-blue-500 hover:bg-blue-600 rounded-lg text-sm font-medium transition-colors flex items-center gap-2"
          >
            <Plus className="w-4 h-4" />
            Create Invite
          </button>
        </div>
      </div>

      {/* Role Info */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        {(['admin', 'moderator', 'user'] as const).map(role => (
          <div key={role} className={`bg-gray-800/50 rounded-lg p-4 border-2 ${
            role === 'admin' ? 'border-red-500/30' : role === 'moderator' ? 'border-yellow-500/30' : 'border-blue-500/30'
          }`}>
            <div className="flex items-center gap-2 mb-2">
              <span className={`px-2 py-0.5 rounded text-xs uppercase font-semibold ${ROLE_INFO[role].color}`}>
                {role}
              </span>
              <span className="text-xs text-gray-400">Power: {ROLE_INFO[role].power}</span>
            </div>
            <ul className="text-sm text-gray-400 space-y-1">
              {ROLE_INFO[role].permissions.map(perm => (
                <li key={perm} className="flex items-center gap-1">
                  <CheckCircle className="w-3 h-3 text-green-400" />
                  {perm}
                </li>
              ))}
            </ul>
          </div>
        ))}
      </div>

      {/* Invitations List */}
      <div className="bg-gray-800/50 rounded-lg overflow-hidden">
        <div className="p-4 border-b border-gray-700">
          <h3 className="font-semibold">All Invitations</h3>
        </div>
        {inviteList.length === 0 ? (
          <div className="p-8 text-center text-gray-400">
            <Mail className="w-12 h-12 mx-auto mb-3 opacity-50" />
            <p>No invitations created yet</p>
          </div>
        ) : (
          <div className="divide-y divide-gray-700">
            {inviteList.map(invite => (
              <div key={invite.id} className="p-4">
                <div className="flex items-center justify-between mb-2">
                  <div className="flex items-center gap-3">
                    <code className="px-3 py-1 bg-gray-700 rounded font-mono text-sm">
                      {invite.code}
                    </code>
                    {getStatusBadge(invite.status)}
                    <span className={`px-2 py-0.5 rounded text-xs uppercase font-semibold ${ROLE_INFO[invite.role].color}`}>
                      {invite.role}
                    </span>
                  </div>
                  <div className="flex items-center gap-2">
                    {invite.status === 'active' && (
                      <>
                        <button
                          onClick={() => copyInviteLink(invite.code)}
                          className="p-2 text-gray-400 hover:text-white hover:bg-gray-700 rounded-lg transition-colors"
                          title="Copy invite link"
                        >
                          {copiedCode === invite.code ? (
                            <CheckCircle className="w-4 h-4 text-green-400" />
                          ) : (
                            <Copy className="w-4 h-4" />
                          )}
                        </button>
                        <button
                          onClick={() => handleRevoke(invite.id)}
                          disabled={revokeMutation.isPending}
                          className="p-2 text-gray-400 hover:text-red-400 hover:bg-red-500/10 rounded-lg transition-colors"
                          title="Revoke invite"
                        >
                          <Trash2 className="w-4 h-4" />
                        </button>
                      </>
                    )}
                  </div>
                </div>
                <div className="flex items-center gap-4 text-sm text-gray-400">
                  <span className="flex items-center gap-1">
                    <Users className="w-3 h-3" />
                    {invite.use_count}/{invite.max_uses === 0 ? '∞' : invite.max_uses} uses
                  </span>
                  {invite.expires_at && (
                    <span className="flex items-center gap-1">
                      <Clock className="w-3 h-3" />
                      Expires: {new Date(invite.expires_at).toLocaleDateString()}
                    </span>
                  )}
                  <span>Created: {new Date(invite.created_at).toLocaleDateString()}</span>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Create Modal */}
      {showCreateModal && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-gray-800 rounded-lg p-6 max-w-md w-full mx-4">
            <h3 className="text-lg font-semibold mb-4">Create New Invitation</h3>

            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium mb-2">Role</label>
                <div className="grid grid-cols-3 gap-2">
                  {(['admin', 'moderator', 'user'] as const).map(role => (
                    <button
                      key={role}
                      onClick={() => setNewRole(role)}
                      className={`py-2 px-3 rounded-lg border-2 text-sm font-medium transition-colors ${
                        newRole === role
                          ? role === 'admin'
                            ? 'border-red-500 bg-red-500/20 text-red-400'
                            : role === 'moderator'
                            ? 'border-yellow-500 bg-yellow-500/20 text-yellow-400'
                            : 'border-blue-500 bg-blue-500/20 text-blue-400'
                          : 'border-gray-700 hover:border-gray-600'
                      }`}
                    >
                      {role.charAt(0).toUpperCase() + role.slice(1)}
                    </button>
                  ))}
                </div>
              </div>

              <div>
                <label className="block text-sm font-medium mb-2">Expiration</label>
                <select
                  value={newExpiration}
                  onChange={(e) => setNewExpiration(e.target.value)}
                  className="w-full px-3 py-2 bg-gray-700 border border-gray-600 rounded-lg focus:outline-none focus:border-blue-500"
                >
                  {EXPIRATION_OPTIONS.map(opt => (
                    <option key={opt.value} value={opt.value}>{opt.label}</option>
                  ))}
                </select>
              </div>

              <div>
                <label className="block text-sm font-medium mb-2">Max Uses</label>
                <input
                  type="number"
                  min="1"
                  value={newMaxUses}
                  onChange={(e) => setNewMaxUses(e.target.value)}
                  className="w-full px-3 py-2 bg-gray-700 border border-gray-600 rounded-lg focus:outline-none focus:border-blue-500"
                />
                <p className="text-xs text-gray-400 mt-1">Set to 0 for unlimited uses</p>
              </div>
            </div>

            <div className="flex gap-3 mt-6">
              <button
                onClick={() => setShowCreateModal(false)}
                className="flex-1 px-4 py-2 bg-gray-700 hover:bg-gray-600 rounded-lg transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={createInvite}
                disabled={createMutation.isPending}
                className="flex-1 px-4 py-2 bg-blue-500 hover:bg-blue-600 disabled:opacity-50 rounded-lg transition-colors"
              >
                Create
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

export default InvitationsPage;
