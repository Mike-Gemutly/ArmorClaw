import React, { useState, useEffect, createContext, useContext } from 'react';
import { BrowserRouter, Routes, Route, Link, useLocation, Navigate } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import {
  Shield,
  Key,
  Users,
  Globe,
  MessageSquare,
  Activity,
  Menu,
  X,
  AlertTriangle,
  CheckCircle,
  Lock,
  LogOut,
  Settings
} from 'lucide-react';

import { LoginPage } from './pages/LoginPage';
import { AdminDashboard } from './pages/AdminDashboard';
import { SecurityConfig } from './pages/SecurityConfig';
import { AdaptersPage } from './pages/AdaptersPage';
import { APIKeysPage } from './pages/APIKeysPage';
import { DevicesPage } from './pages/DevicesPage';
import { InvitationsPage } from './pages/InvitationsPage';
import { AuditLogPage } from './pages/AuditLogPage';
import { useLockdownStatus } from './services/bridgeApi';

// Query client for React Query
const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: 1,
      refetchOnWindowFocus: false,
    },
  },
});

// Auth Context
interface AuthContextType {
  isAuthenticated: boolean;
  token: string | null;
  login: (token: string) => void;
  logout: () => void;
}

const AuthContext = createContext<AuthContextType | null>(null);

function useAuth() {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
}

function AuthProvider({ children }: { children: React.ReactNode }) {
  const [token, setToken] = useState<string | null>(() => {
    return localStorage.getItem('admin_token');
  });

  const login = (newToken: string) => {
    setToken(newToken);
    localStorage.setItem('admin_token', newToken);
  };

  const logout = () => {
    setToken(null);
    localStorage.removeItem('admin_token');
  };

  // Check token expiry (24 hours)
  useEffect(() => {
    if (token) {
      const loginTime = localStorage.getItem('admin_login_time');
      if (loginTime) {
        const elapsed = Date.now() - parseInt(loginTime);
        if (elapsed > 24 * 60 * 60 * 1000) {
          logout();
        }
      } else {
        localStorage.setItem('admin_login_time', Date.now().toString());
      }
    }
  }, [token]);

  return (
    <AuthContext.Provider value={{ isAuthenticated: !!token, token, login, logout }}>
      {children}
    </AuthContext.Provider>
  );
}

// Protected Route
function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const { isAuthenticated } = useAuth();
  const location = useLocation();

  if (!isAuthenticated) {
    return <Navigate to="/login" state={{ from: location }} replace />;
  }

  return <>{children}</>;
}

// Mock stats - would come from bridge API in production
const MOCK_STATS = {
  lockdownMode: false,
  setupComplete: true,
  totalDevices: 2,
  pendingVerifications: 1,
  activeInvites: 1,
  apiKeysCount: 1,
  enabledAdapters: ['Slack']
};

function Layout({ children }: { children: React.ReactNode }) {
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const location = useLocation();
  const { logout } = useAuth();
  const { data: lockdownStatus } = useLockdownStatus();

  // Use real data if available, otherwise fall back to mock
  const status = lockdownStatus || {
    mode: 'operational',
    setup_complete: MOCK_STATS.setupComplete,
  };

  const navItems = [
    { path: '/', icon: Shield, label: 'Dashboard' },
    { path: '/security', icon: Lock, label: 'Security' },
    { path: '/adapters', icon: Globe, label: 'Adapters' },
    { path: '/secrets', icon: Key, label: 'API Keys' },
    { path: '/devices', icon: Users, label: 'Devices' },
    { path: '/invites', icon: MessageSquare, label: 'Invitations' },
    { path: '/audit', icon: Activity, label: 'Audit Log' },
  ];

  const isLockdown = status.mode === 'lockdown' || status.mode === 'bonding';

  return (
    <div className="min-h-screen bg-gray-900 text-white">
      {/* Mobile Header */}
      <div className="lg:hidden flex items-center justify-between p-4 border-b border-gray-800">
        <div className="flex items-center gap-2">
          <Shield className="w-6 h-6 text-blue-400" />
          <span className="font-bold text-lg">ArmorClaw</span>
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={logout}
            className="p-2 text-gray-400 hover:text-white"
          >
            <LogOut className="w-5 h-5" />
          </button>
          <button
            onClick={() => setSidebarOpen(!sidebarOpen)}
            className="p-2 hover:bg-gray-800 rounded-lg"
          >
            {sidebarOpen ? <X className="w-6 h-6" /> : <Menu className="w-6 h-6" />}
          </button>
        </div>
      </div>

      <div className="flex">
        {/* Sidebar */}
        <aside className={`
          fixed inset-y-0 left-0 z-50 w-64 bg-gray-800/50 border-r border-gray-700
          transform transition-transform duration-200 lg:translate-x-0 lg:static lg:inset-auto
          ${sidebarOpen ? 'translate-x-0' : '-translate-x-full'}
        `}>
          <div className="p-4 border-b border-gray-700 hidden lg:block">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <Shield className="w-8 h-8 text-blue-400" />
                <div>
                  <span className="font-bold text-lg">ArmorClaw</span>
                  <span className="block text-xs text-gray-400">Admin Panel</span>
                </div>
              </div>
              <button
                onClick={logout}
                className="p-2 text-gray-400 hover:text-white hover:bg-gray-700 rounded-lg"
                title="Sign out"
              >
                <LogOut className="w-4 h-4" />
              </button>
            </div>
          </div>

          {/* Status Indicator */}
          <div className="p-4 border-b border-gray-700">
            {isLockdown ? (
              <div className="flex items-center gap-2 text-red-400 text-sm">
                <AlertTriangle className="w-4 h-4" />
                <span className="capitalize">{status.mode} Mode</span>
              </div>
            ) : (
              <div className="flex items-center gap-2 text-green-400 text-sm">
                <CheckCircle className="w-4 h-4" />
                <span className="capitalize">{status.mode}</span>
              </div>
            )}
          </div>

          {/* Navigation */}
          <nav className="p-4 space-y-1">
            {navItems.map(item => {
              const Icon = item.icon;
              const isActive = location.pathname === item.path;
              return (
                <Link
                  key={item.path}
                  to={item.path}
                  onClick={() => setSidebarOpen(false)}
                  className={`
                    flex items-center gap-3 px-3 py-2 rounded-lg transition-colors
                    ${isActive
                      ? 'bg-blue-500/20 text-blue-400'
                      : 'text-gray-400 hover:text-white hover:bg-gray-700/50'
                    }
                  `}
                >
                  <Icon className="w-5 h-5" />
                  <span>{item.label}</span>
                </Link>
              );
            })}
          </nav>

          {/* Version */}
          <div className="absolute bottom-0 left-0 right-0 p-4 border-t border-gray-700">
            <p className="text-xs text-gray-500">ArmorClaw v1.0.0</p>
            <p className="text-xs text-gray-600">Admin Panel v1.0.0</p>
          </div>
        </aside>

        {/* Mobile overlay */}
        {sidebarOpen && (
          <div
            className="fixed inset-0 bg-black/50 z-40 lg:hidden"
            onClick={() => setSidebarOpen(false)}
          />
        )}

        {/* Main Content */}
        <main className="flex-1 p-4 lg:p-8 min-h-screen">
          {children}
        </main>
      </div>
    </div>
  );
}

function AppRoutes() {
  const { login } = useAuth();

  return (
    <Routes>
      <Route path="/login" element={<LoginPage onLogin={login} />} />
      <Route
        path="/*"
        element={
          <ProtectedRoute>
            <Layout>
              <Routes>
                <Route path="/" element={<AdminDashboard stats={MOCK_STATS} />} />
                <Route path="/security" element={<SecurityConfig />} />
                <Route path="/adapters" element={<AdaptersPage />} />
                <Route path="/secrets" element={<APIKeysPage />} />
                <Route path="/devices" element={<DevicesPage />} />
                <Route path="/invites" element={<InvitationsPage />} />
                <Route path="/audit" element={<AuditLogPage />} />
              </Routes>
            </Layout>
          </ProtectedRoute>
        }
      />
    </Routes>
  );
}

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <AuthProvider>
          <AppRoutes />
        </AuthProvider>
      </BrowserRouter>
    </QueryClientProvider>
  );
}

export default App;
