import React, { useState } from 'react';
import {
  Shield,
  Lock,
  Key,
  AlertTriangle,
  Eye,
  EyeOff,
  Smartphone,
  QrCode,
  ArrowRight
} from 'lucide-react';

interface LoginPageProps {
  onLogin: (token: string) => void;
}

export function LoginPage({ onLogin }: LoginPageProps) {
  const [mode, setMode] = useState<'password' | 'token' | 'qr'>('password');
  const [password, setPassword] = useState('');
  const [token, setToken] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(false);

  const handlePasswordLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsLoading(true);
    setError(null);

    try {
      // In production, this would verify with the bridge
      await new Promise(resolve => setTimeout(resolve, 1000));

      if (password.length >= 8) {
        onLogin('session_' + Date.now());
      } else {
        setError('Invalid passphrase');
      }
    } catch (err) {
      setError('Login failed. Please try again.');
    } finally {
      setIsLoading(false);
    }
  };

  const handleTokenLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsLoading(true);
    setError(null);

    try {
      // In production, validate token with bridge
      await new Promise(resolve => setTimeout(resolve, 1000));

      if (token.length >= 10) {
        onLogin(token);
      } else {
        setError('Invalid token');
      }
    } catch (err) {
      setError('Token validation failed.');
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="min-h-screen bg-gray-900 flex items-center justify-center p-4">
      <div className="w-full max-w-md">
        {/* Header */}
        <div className="text-center mb-8">
          <div className="flex justify-center mb-4">
            <div className="p-3 bg-blue-500/20 rounded-full">
              <Shield className="w-10 h-10 text-blue-400" />
            </div>
          </div>
          <h1 className="text-2xl font-bold">ArmorClaw Admin</h1>
          <p className="text-gray-400 mt-2">Sign in to access the admin panel</p>
        </div>

        {/* Login Card */}
        <div className="bg-gray-800/50 rounded-lg p-6 border border-gray-700">
          {/* Mode Tabs */}
          <div className="flex mb-6 bg-gray-900/50 rounded-lg p-1">
            <button
              onClick={() => setMode('password')}
              className={`flex-1 py-2 px-3 rounded-md text-sm font-medium transition-colors ${
                mode === 'password'
                  ? 'bg-blue-500 text-white'
                  : 'text-gray-400 hover:text-white'
              }`}
            >
              <Lock className="w-4 h-4 inline mr-1" />
              Passphrase
            </button>
            <button
              onClick={() => setMode('token')}
              className={`flex-1 py-2 px-3 rounded-md text-sm font-medium transition-colors ${
                mode === 'token'
                  ? 'bg-blue-500 text-white'
                  : 'text-gray-400 hover:text-white'
              }`}
            >
              <Key className="w-4 h-4 inline mr-1" />
              Token
            </button>
            <button
              onClick={() => setMode('qr')}
              className={`flex-1 py-2 px-3 rounded-md text-sm font-medium transition-colors ${
                mode === 'qr'
                  ? 'bg-blue-500 text-white'
                  : 'text-gray-400 hover:text-white'
              }`}
            >
              <QrCode className="w-4 h-4 inline mr-1" />
              QR
            </button>
          </div>

          {/* Error */}
          {error && (
            <div className="mb-4 p-3 bg-red-900/20 border border-red-500/50 rounded-lg flex items-center gap-2">
              <AlertTriangle className="w-4 h-4 text-red-400" />
              <span className="text-sm text-red-300">{error}</span>
            </div>
          )}

          {/* Password Mode */}
          {mode === 'password' && (
            <form onSubmit={handlePasswordLogin}>
              <div className="mb-4">
                <label className="block text-sm font-medium mb-2">
                  Admin Passphrase
                </label>
                <div className="relative">
                  <input
                    type={showPassword ? 'text' : 'password'}
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    placeholder="Enter your passphrase"
                    className="w-full px-4 py-3 bg-gray-900/50 border border-gray-600 rounded-lg focus:border-blue-500 focus:outline-none pr-10"
                  />
                  <button
                    type="button"
                    onClick={() => setShowPassword(!showPassword)}
                    className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 hover:text-white"
                  >
                    {showPassword ? <EyeOff className="w-5 h-5" /> : <Eye className="w-5 h-5" />}
                  </button>
                </div>
                <p className="text-xs text-gray-500 mt-2">
                  This is the passphrase you set during initial setup
                </p>
              </div>

              <button
                type="submit"
                disabled={isLoading || password.length < 8}
                className="w-full py-3 bg-blue-500 hover:bg-blue-600 disabled:opacity-50 disabled:cursor-not-allowed rounded-lg font-medium flex items-center justify-center gap-2"
              >
                {isLoading ? (
                  <span>Signing in...</span>
                ) : (
                  <>
                    Sign In
                    <ArrowRight className="w-4 h-4" />
                  </>
                )}
              </button>
            </form>
          )}

          {/* Token Mode */}
          {mode === 'token' && (
            <form onSubmit={handleTokenLogin}>
              <div className="mb-4">
                <label className="block text-sm font-medium mb-2">
                  Session Token
                </label>
                <input
                  type="text"
                  value={token}
                  onChange={(e) => setToken(e.target.value)}
                  placeholder="Paste your session token"
                  className="w-full px-4 py-3 bg-gray-900/50 border border-gray-600 rounded-lg focus:border-blue-500 focus:outline-none font-mono text-sm"
                />
                <p className="text-xs text-gray-500 mt-2">
                  Get a session token from ArmorChat or the setup wizard
                </p>
              </div>

              <button
                type="submit"
                disabled={isLoading || token.length < 10}
                className="w-full py-3 bg-blue-500 hover:bg-blue-600 disabled:opacity-50 disabled:cursor-not-allowed rounded-lg font-medium flex items-center justify-center gap-2"
              >
                {isLoading ? (
                  <span>Validating...</span>
                ) : (
                  <>
                    Sign In with Token
                    <ArrowRight className="w-4 h-4" />
                  </>
                )}
              </button>
            </form>
          )}

          {/* QR Mode */}
          {mode === 'qr' && (
            <div className="text-center">
              <div className="bg-white rounded-lg p-4 inline-block mb-4">
                <div className="w-48 h-48 bg-gray-100 flex items-center justify-center">
                  <div className="text-gray-400 text-sm">
                    <Smartphone className="w-12 h-12 mx-auto mb-2" />
                    <p>Scan with ArmorChat</p>
                  </div>
                </div>
              </div>
              <p className="text-sm text-gray-400">
                Open ArmorChat on your verified device and scan this QR code
                to sign in to the admin panel.
              </p>
              <p className="text-xs text-gray-500 mt-2">
                QR code refreshes every 30 seconds
              </p>
            </div>
          )}
        </div>

        {/* Security Notice */}
        <div className="mt-6 p-4 bg-gray-800/30 rounded-lg">
          <div className="flex items-start gap-3">
            <Shield className="w-5 h-5 text-blue-400 mt-0.5" />
            <div>
              <p className="text-sm font-medium">Secure Access</p>
              <p className="text-xs text-gray-400 mt-1">
                This admin panel is only accessible from your verified devices.
                All sessions expire after 24 hours of inactivity.
              </p>
            </div>
          </div>
        </div>

        {/* Footer */}
        <div className="mt-6 text-center text-xs text-gray-500">
          <p>ArmorClaw v1.0.0</p>
          <p className="mt-1">Having trouble? Check the docs at armorclaw.app/docs</p>
        </div>
      </div>
    </div>
  );
}

export default LoginPage;
