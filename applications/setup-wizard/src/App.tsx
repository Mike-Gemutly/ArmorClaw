import React, { useState, useEffect, useCallback } from 'react';
import {
  Shield,
  CheckCircle,
  AlertTriangle,
  Lock,
  Globe,
  Users,
  Key,
  MessageSquare,
  ArrowRight,
  ArrowLeft,
  RefreshCw,
  Copy,
  Eye,
  EyeOff,
  ChevronDown,
  ChevronUp,
  Terminal,
  Wifi,
  WifiOff,
  AlertCircle
} from 'lucide-react';

import { bridgeApi, hashPassphrase, generateDeviceFingerprint } from './services/bridgeApi';
import { mapSetupError, retryWithBackoff, SetupError, ErrorCodes } from './utils/errorHandling';

// Steps in the setup wizard
type Step = 'welcome' | 'claim' | 'security' | 'secrets' | 'adapters' | 'finalize';

// Data categories for security configuration
const DATA_CATEGORIES = [
  { id: 'banking', name: 'Banking Information', risk: 'high', examples: ['Account numbers', 'Balances', 'Credit cards'] },
  { id: 'pii', name: 'Personally Identifiable Information', risk: 'high', examples: ['SSN', 'Passport', "Driver's license"] },
  { id: 'medical', name: 'Medical Information', risk: 'high', examples: ['Health records', 'Prescriptions', 'Insurance'] },
  { id: 'residential', name: 'Residential Information', risk: 'medium', examples: ['Address', 'Phone', 'Email'] },
  { id: 'network', name: 'Network Information', risk: 'medium', examples: ['IP address', 'MAC address', 'Hostname'] },
  { id: 'identity', name: 'Identity Information', risk: 'medium', examples: ['Name', 'DOB', 'Photo'] },
  { id: 'location', name: 'Location Information', risk: 'low', examples: ['GPS', 'City', 'Country'] },
  { id: 'credentials', name: 'Credentials', risk: 'high', examples: ['Usernames', 'Passwords', 'API keys'] }
];

const ADAPTERS = [
  { id: 'slack', name: 'Slack', description: 'Team messaging' },
  { id: 'discord', name: 'Discord', description: 'Community chat' },
  { id: 'teams', name: 'Microsoft Teams', description: 'Enterprise communication' },
  { id: 'whatsapp', name: 'WhatsApp', description: 'Personal messaging' }
];

const API_PROVIDERS = [
  { id: 'openai', name: 'OpenAI', description: 'GPT-4, GPT-3.5' },
  { id: 'anthropic', name: 'Anthropic', description: 'Claude models' },
  { id: 'google', name: 'Google AI', description: 'Gemini models' },
  { id: 'mistral', name: 'Mistral', description: 'Open source models' }
];

export function SetupWizard() {
  const [currentStep, setCurrentStep] = useState<Step>('welcome');
  const [isClaimed, setIsClaimed] = useState(false);
  const [claimToken, setClaimToken] = useState('');
  const [challengeCode, setChallengeCode] = useState('');
  const [securityConfig, setSecurityConfig] = useState<Record<string, 'deny' | 'allow' | 'allow_all'>>(() => {
    const config: Record<string, string> = {};
    DATA_CATEGORIES.forEach(cat => config[cat.id] = 'deny');
    return config as Record<string, 'deny' | 'allow' | 'allow_all'>;
  });
  const [adapters, setAdapters] = useState<Record<string, boolean>>({});
  const [apiKeys, setApiKeys] = useState<Record<string, string>>({});
  const [showApiKey, setShowApiKey] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [setupError, setSetupError] = useState<SetupError | null>(null);
  const [isConnected, setIsConnected] = useState<boolean | null>(null);
  const [isRetrying, setIsRetrying] = useState(false);
  const [retryAttempt, setRetryAttempt] = useState(0);
  const [lockdownMode, setLockdownMode] = useState<string>('lockdown');

  // Check bridge connection on mount
  useEffect(() => {
    checkConnection();
  }, []);

  const checkConnection = useCallback(async () => {
    setIsRetrying(true);
    setSetupError(null);
    setRetryAttempt(0);

    try {
      await retryWithBackoff(
        async () => {
          const status = await bridgeApi.getLockdownStatus();
          setIsConnected(true);
          setLockdownMode(status.mode);
          if (status.admin_established) {
            setIsClaimed(true);
          }
          return status;
        },
        {
          maxAttempts: 3,
          baseDelay: 2000,
          onRetry: (attempt) => setRetryAttempt(attempt),
        }
      );
    } catch (error) {
      const mappedError = mapSetupError(error);
      setSetupError(mappedError);
      setIsConnected(false);
    } finally {
      setIsRetrying(false);
    }
  }, []);

  const clearError = useCallback(() => {
    setSetupError(null);
  }, []);

  const nextStep = () => {
    const steps: Step[] = ['welcome', 'claim', 'security', 'secrets', 'adapters', 'finalize'];
    const currentIndex = steps.indexOf(currentStep);
    if (currentIndex < steps.length - 1) {
      setCurrentStep(steps[currentIndex + 1]);
      clearError();
    }
  };

  const prevStep = () => {
    const steps: Step[] = ['welcome', 'claim', 'security', 'secrets', 'adapters', 'finalize'];
    const currentIndex = steps.indexOf(currentStep);
    if (currentIndex > 0) {
      setCurrentStep(steps[currentIndex - 1]);
      clearError();
    }
  };

  const handleClaim = async () => {
    setIsSubmitting(true);
    setSetupError(null);

    try {
      await retryWithBackoff(
        async () => {
          // Get challenge
          const challenge = await bridgeApi.getChallenge();
          setChallengeCode(challenge.nonce);

          // Generate device fingerprint
          const fingerprint = await generateDeviceFingerprint('Web Setup Wizard');

          // Hash passphrase (in production use argon2id)
          const passphraseCommitment = await hashPassphrase('setup-wizard-passphrase');

          // Claim ownership
          const result = await bridgeApi.claimOwnership({
            display_name: 'Admin',
            device_name: 'Web Setup Wizard',
            device_fingerprint: fingerprint,
            passphrase_commitment: passphraseCommitment,
            challenge_response: challenge.nonce
          });

          if (result.status === 'claimed') {
            setIsClaimed(true);
            setLockdownMode('configuring');
            nextStep();
          }
          return result;
        },
        {
          maxAttempts: 2,
          baseDelay: 1500,
        }
      );
    } catch (error) {
      const mappedError = mapSetupError(error);
      setSetupError(mappedError);
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleSubmit = async () => {
    setIsSubmitting(true);
    setSetupError(null);

    try {
      // Save security configuration with retry
      for (const [category, permission] of Object.entries(securityConfig)) {
        await retryWithBackoff(
          () => bridgeApi.setSecurityCategory(category, permission),
          { maxAttempts: 2 }
        );
      }

      // Enable adapters with retry
      for (const [adapterId, enabled] of Object.entries(adapters)) {
        if (enabled) {
          await retryWithBackoff(
            () => bridgeApi.enableAdapter(adapterId),
            { maxAttempts: 2 }
          );
        }
      }

      // Transition to operational
      await retryWithBackoff(
        () => bridgeApi.transitionMode('operational'),
        { maxAttempts: 2 }
      );
      setCurrentStep('finalize');
    } catch (error) {
      const mappedError = mapSetupError(error);
      setSetupError(mappedError);
    } finally {
      setIsSubmitting(false);
    }
  };

  const canProceed = () => {
    switch (currentStep) {
      case 'welcome':
        return isConnected !== false;
      case 'claim':
        return isClaimed;
      case 'security':
        return true;
      case 'secrets':
        return Object.values(apiKeys).some(k => k.length > 0);
      case 'adapters':
        return true;
      case 'finalize':
        return false;
      default:
        return false;
    }
  };

  const getStepProgress = () => {
    const steps: Step[] = ['welcome', 'claim', 'security', 'secrets', 'adapters', 'finalize'];
    return ((steps.indexOf(currentStep) + 1) / steps.length) * 100;
  };

  const renderStep = () => {
    switch (currentStep) {
      case 'welcome':
        return (
          <WelcomeStep
            isConnected={isConnected}
            isRetrying={isRetrying}
            retryAttempt={retryAttempt}
            error={setupError}
            onRetry={checkConnection}
          />
        );
      case 'claim':
        return (
          <ClaimStep
            claimToken={claimToken}
            setClaimToken={setClaimToken}
            challengeCode={challengeCode}
            isSubmitting={isSubmitting}
            error={setupError}
            onClaim={handleClaim}
            isConnected={isConnected}
          />
        );
      case 'security':
        return (
          <SecurityStep
            config={securityConfig}
            setConfig={setSecurityConfig}
          />
        );
      case 'secrets':
        return (
          <SecretsStep
            apiKeys={apiKeys}
            setApiKeys={setApiKeys}
            showApiKey={showApiKey}
            setShowApiKey={setShowApiKey}
          />
        );
      case 'adapters':
        return (
          <AdaptersStep
            adapters={adapters}
            setAdapters={setAdapters}
          />
        );
      case 'finalize':
        return <FinalizeStep />;
    }
  };

  return (
    <div className="min-h-screen bg-gray-900 text-white">
      {/* Header */}
      <header className="border-b border-gray-800 bg-gray-900/80 sticky top-0 z-10">
        <div className="max-w-4xl mx-auto px-6 py-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              <Shield className="w-8 h-8 text-blue-400" />
              <div>
                <h1 className="font-bold text-lg">ArmorClaw Setup</h1>
                <p className="text-sm text-gray-400">First-time configuration</p>
              </div>
            </div>
            <div className="flex items-center gap-4">
              {/* Connection Status */}
              <div className={`flex items-center gap-1 text-sm ${isConnected ? 'text-green-400' : isConnected === false ? 'text-red-400' : 'text-gray-400'}`}>
                {isConnected ? <Wifi className="w-4 h-4" /> : <WifiOff className="w-4 h-4" />}
                <span>{isConnected ? 'Connected' : isConnected === false ? 'Disconnected' : 'Checking...'}</span>
              </div>
              <div className="text-sm text-gray-400">
                Step {['welcome', 'claim', 'security', 'secrets', 'adapters', 'finalize'].indexOf(currentStep) + 1} of 6
              </div>
            </div>
          </div>

          {/* Progress bar */}
          <div className="mt-4 h-1 bg-gray-800 rounded-full overflow-hidden">
            <div
              className="h-full bg-blue-500 transition-all duration-300"
              style={{ width: `${getStepProgress()}%` }}
            />
          </div>
        </div>
      </header>

      {/* Main content */}
      <main className="max-w-4xl mx-auto px-6 py-8">
        {renderStep()}

        {setupError && (
          <div className="mt-4 p-4 bg-red-900/20 border border-red-500/50 rounded-lg">
            <div className="flex items-start gap-3">
              <AlertCircle className="w-5 h-5 text-red-400 flex-shrink-0 mt-0.5" />
              <div className="flex-1">
                <h4 className="font-medium text-red-300">{setupError.title}</h4>
                <p className="text-sm text-red-200/80 mt-1">{setupError.message}</p>
                {setupError.code !== ErrorCodes.UNKNOWN && (
                  <p className="text-xs text-red-400/60 mt-2 font-mono">
                    Error: {setupError.code}
                  </p>
                )}
              </div>
              {setupError.recoverable && (
                <button
                  onClick={clearError}
                  className="text-red-400 hover:text-red-300"
                >
                  <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                  </svg>
                </button>
              )}
            </div>
          </div>
        )}
      </main>

      {/* Footer */}
      {currentStep !== 'finalize' && (
        <footer className="border-t border-gray-800 bg-gray-900/80 sticky bottom-0">
          <div className="max-w-4xl mx-auto px-6 py-4">
            <div className="flex justify-between">
              <button
                onClick={prevStep}
                disabled={currentStep === 'welcome'}
                className="px-4 py-2 text-gray-400 hover:text-white disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
              >
                <ArrowLeft className="w-4 h-4" />
                Back
              </button>

              {currentStep === 'adapters' ? (
                <button
                  onClick={handleSubmit}
                  disabled={isSubmitting}
                  className="px-6 py-2 bg-green-500 hover:bg-green-600 rounded-lg font-medium flex items-center gap-2 disabled:opacity-50"
                >
                  {isSubmitting ? (
                    <>
                      <RefreshCw className="w-4 h-4 animate-spin" />
                      Saving...
                    </>
                  ) : (
                    <>
                      Complete Setup
                      <CheckCircle className="w-4 h-4" />
                    </>
                  )}
                </button>
              ) : (
                <button
                  onClick={currentStep === 'claim' ? handleClaim : nextStep}
                  disabled={!canProceed() || isSubmitting}
                  className="px-6 py-2 bg-blue-500 hover:bg-blue-600 rounded-lg font-medium flex items-center gap-2 disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  {isSubmitting ? (
                    <>
                      <RefreshCw className="w-4 h-4 animate-spin" />
                      Processing...
                    </>
                  ) : (
                    <>
                      Continue
                      <ArrowRight className="w-4 h-4" />
                    </>
                  )}
                </button>
              )}
            </div>
          </div>
        </footer>
      )}
    </div>
  );
}

// Welcome Step Component
function WelcomeStep({
  isConnected,
  isRetrying,
  retryAttempt,
  error,
  onRetry
}: {
  isConnected: boolean | null;
  isRetrying: boolean;
  retryAttempt: number;
  error: SetupError | null;
  onRetry: () => void;
}) {
  return (
    <div className="text-center py-12">
      <Shield className="w-16 h-16 text-blue-400 mx-auto mb-6" />
      <h2 className="text-3xl font-bold mb-4">Welcome to ArmorClaw</h2>
      <p className="text-gray-400 max-w-lg mx-auto mb-8">
        Let's set up your secure AI agent environment. This wizard will guide you through:
      </p>

      {/* Connection Status - Loading */}
      {isRetrying && (
        <div className="max-w-md mx-auto mb-8 p-4 bg-blue-900/20 border border-blue-500/50 rounded-lg">
          <div className="flex items-center gap-3">
            <RefreshCw className="w-5 h-5 text-blue-400 animate-spin" />
            <span className="text-blue-300">
              Connecting to ArmorClaw Bridge...
              {retryAttempt > 0 && ` (attempt ${retryAttempt + 1})`}
            </span>
          </div>
        </div>
      )}

      {/* Connection Status - Error */}
      {isConnected === false && error && !isRetrying && (
        <div className="max-w-md mx-auto mb-8 p-4 bg-red-900/20 border border-red-500/50 rounded-lg">
          <div className="flex items-center gap-3 mb-3">
            <WifiOff className="w-5 h-5 text-red-400" />
            <span className="text-red-300">{error.title}</span>
          </div>
          <p className="text-sm text-gray-400 mb-3">{error.message}</p>
          <p className="text-sm text-gray-400 mb-3">
            Make sure the bridge is running on your server. Run:
          </p>
          <code className="block bg-gray-800 p-2 rounded text-sm mb-3">
            sudo armorclaw-bridge
          </code>
          <button
            onClick={onRetry}
            className="px-4 py-2 bg-gray-700 hover:bg-gray-600 rounded-lg text-sm flex items-center gap-2 mx-auto"
          >
            <RefreshCw className="w-4 h-4" />
            Retry Connection
          </button>
        </div>
      )}

      {/* Connection Status - Success */}
      {isConnected === true && (
        <div className="max-w-md mx-auto mb-8 p-4 bg-green-900/20 border border-green-500/50 rounded-lg">
          <div className="flex items-center gap-3">
            <CheckCircle className="w-5 h-5 text-green-400" />
            <span className="text-green-300">Connected to ArmorClaw Bridge</span>
          </div>
        </div>
      )}

      <div className="grid grid-cols-1 md:grid-cols-2 gap-4 max-w-lg mx-auto text-left">
        {[
          { icon: Lock, title: 'Admin Claiming', desc: 'Securely verify your identity' },
          { icon: Globe, title: 'Security Config', desc: 'Control data access permissions' },
          { icon: Key, title: 'API Keys', desc: 'Add your AI provider keys' },
          { icon: MessageSquare, title: 'Adapters', desc: 'Enable communication channels' }
        ].map((item, i) => (
          <div key={i} className="flex items-start gap-3 p-3 bg-gray-800/50 rounded-lg">
            <item.icon className="w-5 h-5 text-blue-400 mt-0.5" />
            <div>
              <div className="font-medium">{item.title}</div>
              <div className="text-sm text-gray-400">{item.desc}</div>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

// Claim Step Component
function ClaimStep({
  claimToken,
  setClaimToken,
  challengeCode,
  isSubmitting,
  error,
  onClaim,
  isConnected
}: {
  claimToken: string;
  setClaimToken: (t: string) => void;
  challengeCode: string;
  isSubmitting: boolean;
  error: SetupError | null;
  onClaim: () => void;
  isConnected: boolean | null;
}) {
  return (
    <div className="py-8">
      <h2 className="text-2xl font-bold mb-4">Claim Admin Access</h2>
      <p className="text-gray-400 mb-6">
        To secure your ArmorClaw instance, you need to claim admin access.
      </p>

      {isConnected ? (
        <div className="p-4 bg-green-900/20 border border-green-500/50 rounded-lg mb-6">
          <div className="flex items-center gap-2">
            <CheckCircle className="w-5 h-5 text-green-400" />
            <span className="text-green-300">Connected to ArmorClaw Bridge</span>
          </div>
          <p className="text-sm text-gray-400 mt-2">
            Click "Continue" to automatically claim admin access.
          </p>
        </div>
      ) : (
        <div className="grid md:grid-cols-2 gap-6">
          <div>
            <label className="block text-sm font-medium mb-2">Setup Token (Optional)</label>
            <input
              type="text"
              value={claimToken}
              onChange={(e) => setClaimToken(e.target.value)}
              placeholder="Enter token from terminal..."
              className="w-full px-4 py-3 bg-gray-800 border border-gray-700 rounded-lg focus:border-blue-500 focus:outline-none font-mono"
            />

            {challengeCode && (
              <div className="mt-4 p-4 bg-blue-900/20 border border-blue-500/50 rounded-lg">
                <p className="text-sm text-blue-300 mb-2">Verification Code</p>
                <code className="text-2xl font-mono font-bold">{challengeCode}</code>
              </div>
            )}
          </div>

          <div className="bg-gray-800/50 rounded-lg p-6 flex flex-col items-center justify-center">
            <div className="w-48 h-48 bg-white rounded-lg flex items-center justify-center mb-4">
              <span className="text-gray-400 text-sm">QR Code</span>
            </div>
            <p className="text-sm text-gray-400 text-center">
              Scan with ArmorChat app to auto-fill token
            </p>
          </div>
        </div>
      )}

      <div className="mt-6 p-4 bg-gray-800/50 rounded-lg">
        <div className="flex items-start gap-3">
          <Terminal className="w-5 h-5 text-gray-400 mt-0.5" />
          <div>
            <p className="font-medium">Looking for manual setup?</p>
            <p className="text-sm text-gray-400">
              Run <code className="bg-gray-700 px-1 rounded">sudo armorclaw-bridge</code> on your
              server terminal for advanced options.
            </p>
          </div>
        </div>
      </div>
    </div>
  );
}

// Security Step Component
function SecurityStep({
  config,
  setConfig
}: {
  config: Record<string, 'deny' | 'allow' | 'allow_all'>;
  setConfig: (c: Record<string, 'deny' | 'allow' | 'allow_all'>) => void;
}) {
  const [expanded, setExpanded] = useState<string | null>(null);

  const getRiskColor = (risk: string) => {
    switch (risk) {
      case 'high': return 'bg-red-500/20 text-red-400';
      case 'medium': return 'bg-yellow-500/20 text-yellow-400';
      case 'low': return 'bg-green-500/20 text-green-400';
      default: return 'bg-gray-500/20 text-gray-400';
    }
  };

  const updateCategory = (id: string, value: 'deny' | 'allow' | 'allow_all') => {
    setConfig({ ...config, [id]: value });
  };

  return (
    <div className="py-8">
      <h2 className="text-2xl font-bold mb-4">Security Configuration</h2>
      <p className="text-gray-400 mb-6">
        Control what information ArmorClaw can access and where it can be used.
      </p>

      <div className="space-y-3">
        {DATA_CATEGORIES.map(cat => (
          <div key={cat.id} className="bg-gray-800/50 rounded-lg overflow-hidden">
            <button
              onClick={() => setExpanded(expanded === cat.id ? null : cat.id)}
              className="w-full p-4 flex items-center justify-between hover:bg-gray-700/30"
            >
              <div className="flex items-center gap-3">
                <span className={`px-2 py-0.5 rounded text-xs uppercase font-semibold ${getRiskColor(cat.risk)}`}>
                  {cat.risk}
                </span>
                <span className="font-medium">{cat.name}</span>
              </div>
              <div className="flex items-center gap-3">
                <span className="text-sm text-gray-400 capitalize">{config[cat.id].replace('_', ' ')}</span>
                {expanded === cat.id ? <ChevronUp className="w-4 h-4" /> : <ChevronDown className="w-4 h-4" />}
              </div>
            </button>

            {expanded === cat.id && (
              <div className="border-t border-gray-700 p-4">
                <p className="text-sm text-gray-400 mb-3">
                  Examples: {cat.examples.join(', ')}
                </p>
                <div className="grid grid-cols-3 gap-2">
                  {['deny', 'allow', 'allow_all'].map((option) => (
                    <button
                      key={option}
                      onClick={() => updateCategory(cat.id, option as any)}
                      className={`py-2 px-3 rounded-lg text-sm font-medium transition-colors ${
                        config[cat.id] === option
                          ? option === 'deny'
                            ? 'bg-red-500/20 text-red-400 border border-red-500/50'
                            : option === 'allow_all'
                            ? 'bg-yellow-500/20 text-yellow-400 border border-yellow-500/50'
                            : 'bg-blue-500/20 text-blue-400 border border-blue-500/50'
                          : 'bg-gray-700 hover:bg-gray-600'
                      }`}
                    >
                      {option === 'allow_all' ? 'Allow All' : option.charAt(0).toUpperCase() + option.slice(1)}
                    </button>
                  ))}
                </div>
              </div>
            )}
          </div>
        ))}
      </div>
    </div>
  );
}

// Secrets Step Component
function SecretsStep({
  apiKeys,
  setApiKeys,
  showApiKey,
  setShowApiKey
}: {
  apiKeys: Record<string, string>;
  setApiKeys: (k: Record<string, string>) => void;
  showApiKey: string | null;
  setShowApiKey: (k: string | null) => void;
}) {
  return (
    <div className="py-8">
      <h2 className="text-2xl font-bold mb-4">API Keys</h2>
      <p className="text-gray-400 mb-6">
        Add your AI provider API keys. These are encrypted and stored securely.
      </p>

      <div className="space-y-4">
        {API_PROVIDERS.map(provider => (
          <div key={provider.id} className="bg-gray-800/50 rounded-lg p-4">
            <div className="flex items-center justify-between mb-3">
              <div>
                <h3 className="font-medium">{provider.name}</h3>
                <p className="text-sm text-gray-400">{provider.description}</p>
              </div>
              {apiKeys[provider.id] && (
                <CheckCircle className="w-5 h-5 text-green-400" />
              )}
            </div>
            <div className="relative">
              <input
                type={showApiKey === provider.id ? 'text' : 'password'}
                value={apiKeys[provider.id] || ''}
                onChange={(e) => setApiKeys({ ...apiKeys, [provider.id]: e.target.value })}
                placeholder={`Enter ${provider.name} API key...`}
                className="w-full px-4 py-2 pr-20 bg-gray-700 border border-gray-600 rounded-lg focus:border-blue-500 focus:outline-none font-mono text-sm"
              />
              <button
                onClick={() => setShowApiKey(showApiKey === provider.id ? null : provider.id)}
                className="absolute right-2 top-1/2 -translate-y-1/2 p-1 text-gray-400 hover:text-white"
              >
                {showApiKey === provider.id ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
              </button>
            </div>
          </div>
        ))}
      </div>

      <p className="mt-4 text-sm text-gray-400 flex items-center gap-2">
        <Lock className="w-4 h-4" />
        Keys are encrypted with hardware-bound encryption and never stored in plain text.
      </p>
    </div>
  );
}

// Adapters Step Component
function AdaptersStep({
  adapters,
  setAdapters
}: {
  adapters: Record<string, boolean>;
  setAdapters: (a: Record<string, boolean>) => void;
}) {
  return (
    <div className="py-8">
      <h2 className="text-2xl font-bold mb-4">Communication Adapters</h2>
      <p className="text-gray-400 mb-6">
        Choose which platforms ArmorClaw can use to communicate. You can change this later.
      </p>

      <div className="grid md:grid-cols-2 gap-4">
        {ADAPTERS.map(adapter => (
          <div
            key={adapter.id}
            onClick={() => setAdapters({ ...adapters, [adapter.id]: !adapters[adapter.id] })}
            className={`p-4 rounded-lg border-2 cursor-pointer transition-colors ${
              adapters[adapter.id]
                ? 'border-blue-500 bg-blue-500/10'
                : 'border-gray-700 hover:border-gray-600'
            }`}
          >
            <div className="flex items-center justify-between">
              <div>
                <h3 className="font-medium">{adapter.name}</h3>
                <p className="text-sm text-gray-400">{adapter.description}</p>
              </div>
              <div className={`w-6 h-6 rounded-full border-2 flex items-center justify-center ${
                adapters[adapter.id]
                  ? 'border-blue-500 bg-blue-500'
                  : 'border-gray-600'
              }`}>
                {adapters[adapter.id] && <CheckCircle className="w-4 h-4 text-white" />}
              </div>
            </div>
          </div>
        ))}
      </div>

      <p className="mt-4 text-sm text-gray-400">
        Adapters can be configured with more detail in the admin panel after setup.
      </p>
    </div>
  );
}

// Finalize Step Component
function FinalizeStep() {
  return (
    <div className="py-12 text-center">
      <div className="w-20 h-20 bg-green-500/20 rounded-full flex items-center justify-center mx-auto mb-6">
        <CheckCircle className="w-10 h-10 text-green-400" />
      </div>
      <h2 className="text-3xl font-bold mb-4">Setup Complete!</h2>
      <p className="text-gray-400 max-w-lg mx-auto mb-8">
        Your ArmorClaw instance is now configured and hardened. You can access the admin
        panel to make further changes.
      </p>

      <div className="flex flex-col items-center gap-4">
        <a
          href="/admin"
          className="px-6 py-3 bg-blue-500 hover:bg-blue-600 rounded-lg font-medium flex items-center gap-2"
        >
          Open Admin Panel
          <ArrowRight className="w-4 h-4" />
        </a>
        <a
          href="armorclaw://home"
          className="px-6 py-3 bg-gray-700 hover:bg-gray-600 rounded-lg font-medium flex items-center gap-2"
        >
          Open in ArmorChat
          <MessageSquare className="w-4 h-4" />
        </a>
      </div>

      <div className="mt-8 p-4 bg-gray-800/50 rounded-lg max-w-lg mx-auto">
        <h3 className="font-medium mb-2">Quick Start Guide</h3>
        <ol className="text-sm text-gray-400 text-left space-y-2">
          <li>1. Open ArmorChat on your device</li>
          <li>2. Connect to your Matrix server</li>
          <li>3. Start chatting with your AI agent!</li>
        </ol>
      </div>
    </div>
  );
}

export default SetupWizard;
