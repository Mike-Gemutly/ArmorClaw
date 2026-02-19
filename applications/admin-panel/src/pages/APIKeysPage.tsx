import React, { useState } from 'react';
import {
  Key,
  Plus,
  Eye,
  EyeOff,
  Trash2,
  Copy,
  AlertTriangle,
  CheckCircle,
  Clock,
  Shield
} from 'lucide-react';

interface APIKey {
  id: string;
  provider: string;
  name: string;
  prefix: string;
  createdAt: Date;
  lastUsed?: Date;
  status: 'active' | 'revoked' | 'expired';
}

interface PendingToken {
  id: string;
  token: string;
  createdAt: Date;
  expiresAt: Date;
  used: boolean;
}

const PROVIDERS = [
  { id: 'openai', name: 'OpenAI', description: 'GPT-4, GPT-3.5, DALL-E' },
  { id: 'anthropic', name: 'Anthropic', description: 'Claude models' },
  { id: 'google', name: 'Google AI', description: 'Gemini models' },
  { id: 'mistral', name: 'Mistral', description: 'Mistral AI models' },
  { id: 'groq', name: 'Groq', description: 'Fast inference' },
  { id: 'custom', name: 'Custom', description: 'Custom API endpoint' }
];

export function APIKeysPage() {
  const [keys, setKeys] = useState<APIKey[]>([
    {
      id: '1',
      provider: 'openai',
      name: 'OpenAI Production',
      prefix: 'sk-proj-****',
      createdAt: new Date('2026-01-15'),
      lastUsed: new Date('2026-02-14'),
      status: 'active'
    }
  ]);
  const [pendingTokens, setPendingTokens] = useState<PendingToken[]>([]);
  const [showAddModal, setShowAddModal] = useState(false);
  const [selectedProvider, setSelectedProvider] = useState<string>('');
  const [copiedToken, setCopiedToken] = useState<string | null>(null);

  const generateToken = () => {
    const token = `ac_${Math.random().toString(36).substring(2, 10)}_${Date.now().toString(36)}`;
    const newToken: PendingToken = {
      id: Math.random().toString(36).substring(7),
      token,
      createdAt: new Date(),
      expiresAt: new Date(Date.now() + 10 * 60 * 1000), // 10 minutes
      used: false
    };
    setPendingTokens(prev => [...prev, newToken]);
    return newToken;
  };

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text);
    setCopiedToken(text);
    setTimeout(() => setCopiedToken(null), 2000);
  };

  const revokeKey = (id: string) => {
    setKeys(prev => prev.map(key =>
      key.id === id ? { ...key, status: 'revoked' as const } : key
    ));
  };

  const activeKeys = keys.filter(k => k.status === 'active').length;
  const pendingCount = pendingTokens.filter(t => !t.used && t.expiresAt > new Date()).length;

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">API Keys</h1>
          <p className="text-gray-400">
            Manage API keys for AI providers securely
          </p>
        </div>
        <div className="text-right">
          <p className="text-sm text-gray-400">Active Keys</p>
          <p className="text-2xl font-bold">{activeKeys}</p>
        </div>
      </div>

      {/* Security Notice */}
      <div className="bg-blue-900/20 border border-blue-500/50 rounded-lg p-4 flex items-center gap-3">
        <Shield className="w-5 h-5 text-blue-400 flex-shrink-0" />
        <div>
          <h3 className="font-semibold text-blue-400">Secure Key Storage</h3>
          <p className="text-sm text-blue-300/80">
            API keys are encrypted at rest using hardware-bound encryption (XChaCha20-Poly1305).
            Keys are never logged or exposed in plain text after injection.
          </p>
        </div>
      </div>

      {/* Add Key Section */}
      <div className="bg-gray-800/50 rounded-lg p-4">
        <div className="flex items-center justify-between mb-4">
          <h3 className="font-semibold">Add New API Key</h3>
          <button
            onClick={() => setShowAddModal(true)}
            className="px-4 py-2 bg-blue-500 hover:bg-blue-600 rounded-lg text-sm font-medium transition-colors flex items-center gap-2"
          >
            <Plus className="w-4 h-4" />
            Generate Token
          </button>
        </div>

        <p className="text-sm text-gray-400">
          Generate a one-time token to securely inject your API key through ArmorChat.
          The token expires in 10 minutes and can only be used once.
        </p>
      </div>

      {/* Pending Tokens */}
      {pendingTokens.length > 0 && (
        <div className="bg-gray-800/50 rounded-lg p-4">
          <h3 className="font-semibold mb-3 flex items-center gap-2">
            <Clock className="w-4 h-4 text-yellow-400" />
            Pending Tokens ({pendingCount} active)
          </h3>
          <div className="space-y-2">
            {pendingTokens.filter(t => !t.used && t.expiresAt > new Date()).map(token => (
              <div
                key={token.id}
                className="flex items-center justify-between p-3 bg-gray-700/50 rounded-lg"
              >
                <div className="flex items-center gap-3">
                  <code className="text-sm font-mono text-blue-300">{token.token}</code>
                  <button
                    onClick={() => copyToClipboard(token.token)}
                    className="p-1 hover:bg-gray-600 rounded transition-colors"
                  >
                    {copiedToken === token.token ? (
                      <CheckCircle className="w-4 h-4 text-green-400" />
                    ) : (
                      <Copy className="w-4 h-4 text-gray-400" />
                    )}
                  </button>
                </div>
                <span className="text-xs text-gray-400">
                  Expires: {token.expiresAt.toLocaleTimeString()}
                </span>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Existing Keys */}
      <div className="bg-gray-800/50 rounded-lg p-4">
        <h3 className="font-semibold mb-3">Stored Keys</h3>
        {keys.length === 0 ? (
          <p className="text-gray-400 text-sm">No API keys configured yet</p>
        ) : (
          <div className="space-y-2">
            {keys.map(key => (
              <div
                key={key.id}
                className={`flex items-center justify-between p-3 rounded-lg ${
                  key.status === 'revoked'
                    ? 'bg-red-900/20 border border-red-500/30'
                    : 'bg-gray-700/50'
                }`}
              >
                <div className="flex items-center gap-3">
                  <div className={`p-2 rounded-lg ${
                    key.status === 'active' ? 'bg-green-500/20' : 'bg-red-500/20'
                  }`}>
                    <Key className={`w-4 h-4 ${
                      key.status === 'active' ? 'text-green-400' : 'text-red-400'
                    }`} />
                  </div>
                  <div>
                    <div className="flex items-center gap-2">
                      <span className="font-medium">{key.name}</span>
                      <span className="text-xs text-gray-400">({PROVIDERS.find(p => p.id === key.provider)?.name})</span>
                    </div>
                    <div className="flex items-center gap-2 text-sm text-gray-400">
                      <code>{key.prefix}</code>
                      {key.lastUsed && (
                        <span>• Last used: {key.lastUsed.toLocaleDateString()}</span>
                      )}
                    </div>
                  </div>
                </div>

                <div className="flex items-center gap-2">
                  {key.status === 'revoked' ? (
                    <span className="text-sm text-red-400">Revoked</span>
                  ) : (
                    <button
                      onClick={() => revokeKey(key.id)}
                      className="p-2 text-gray-400 hover:text-red-400 hover:bg-red-500/10 rounded-lg transition-colors"
                    >
                      <Trash2 className="w-4 h-4" />
                    </button>
                  )}
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Provider Info */}
      <div className="bg-gray-800/50 rounded-lg p-4">
        <h3 className="font-semibold mb-3">Supported Providers</h3>
        <div className="grid grid-cols-2 md:grid-cols-3 gap-2">
          {PROVIDERS.map(provider => (
            <div
              key={provider.id}
              className="p-3 bg-gray-700/50 rounded-lg"
            >
              <div className="font-medium text-sm">{provider.name}</div>
              <div className="text-xs text-gray-400">{provider.description}</div>
            </div>
          ))}
        </div>
      </div>

      {/* Add Modal */}
      {showAddModal && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-gray-800 rounded-lg p-6 max-w-md w-full mx-4">
            <h3 className="text-lg font-semibold mb-4">Generate One-Time Token</h3>

            <div className="mb-4">
              <label className="block text-sm font-medium mb-2">Select Provider</label>
              <select
                value={selectedProvider}
                onChange={(e) => setSelectedProvider(e.target.value)}
                className="w-full px-3 py-2 bg-gray-700 border border-gray-600 rounded-lg focus:outline-none focus:border-blue-500"
              >
                <option value="">Choose a provider...</option>
                {PROVIDERS.map(p => (
                  <option key={p.id} value={p.id}>{p.name}</option>
                ))}
              </select>
            </div>

            <div className="flex gap-3">
              <button
                onClick={() => setShowAddModal(false)}
                className="flex-1 px-4 py-2 bg-gray-700 hover:bg-gray-600 rounded-lg transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={() => {
                  const token = generateToken();
                  setShowAddModal(false);
                  setSelectedProvider('');
                }}
                disabled={!selectedProvider}
                className="flex-1 px-4 py-2 bg-blue-500 hover:bg-blue-600 disabled:opacity-50 disabled:cursor-not-allowed rounded-lg transition-colors"
              >
                Generate
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

export default APIKeysPage;
