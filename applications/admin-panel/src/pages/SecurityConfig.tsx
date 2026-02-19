import React, { useState } from 'react';
import {
  Shield,
  AlertTriangle,
  Check,
  X,
  Plus,
  Trash2,
  ChevronDown,
  ChevronUp,
  Lock,
  Globe
} from 'lucide-react';

// Data categories with metadata
const DATA_CATEGORIES = [
  {
    id: 'banking',
    name: 'Banking Information',
    description: 'Account numbers, routing numbers, balances',
    examples: ['Account numbers', 'Routing numbers', 'Balances', 'Credit card numbers'],
    riskLevel: 'high'
  },
  {
    id: 'pii',
    name: 'Personally Identifiable Information',
    description: 'Government-issued identifiers and personal documents',
    examples: ['SSN', "Driver's license", 'Passport', 'Tax ID'],
    riskLevel: 'high'
  },
  {
    id: 'medical',
    name: 'Medical Information',
    description: 'Health records and medical history',
    examples: ['Diagnoses', 'Prescriptions', 'Lab results', 'Insurance info'],
    riskLevel: 'high'
  },
  {
    id: 'residential',
    name: 'Residential Information',
    description: 'Physical address and contact details',
    examples: ['Home address', 'Phone number', 'Personal email'],
    riskLevel: 'medium'
  },
  {
    id: 'network',
    name: 'Network Information',
    description: 'Network identifiers and infrastructure details',
    examples: ['IP address', 'MAC address', 'Hostname', 'DNS records'],
    riskLevel: 'medium'
  },
  {
    id: 'identity',
    name: 'Identity Information',
    description: 'Personal identity attributes',
    examples: ['Full name', 'Date of birth', 'Photo', 'Signature'],
    riskLevel: 'medium'
  },
  {
    id: 'location',
    name: 'Location Information',
    description: 'Geographic location data',
    examples: ['GPS coordinates', 'City', 'Country', 'Timezone'],
    riskLevel: 'low'
  },
  {
    id: 'credentials',
    name: 'Credentials',
    description: 'Authentication and access credentials',
    examples: ['Usernames', 'Passwords', 'API keys', 'Tokens'],
    riskLevel: 'high'
  }
];

type PermissionLevel = 'deny' | 'allow' | 'allow_all';

interface CategoryConfig {
  permission: PermissionLevel;
  allowedWebsites: string[];
  requiresApproval: boolean;
}

export function SecurityConfig() {
  const [configs, setConfigs] = useState<Record<string, CategoryConfig>>(() => {
    const initial: Record<string, CategoryConfig> = {};
    DATA_CATEGORIES.forEach(cat => {
      initial[cat.id] = {
        permission: 'deny',
        allowedWebsites: [],
        requiresApproval: true
      };
    });
    return initial;
  });

  const [expandedCategory, setExpandedCategory] = useState<string | null>(null);
  const [newWebsite, setNewWebsite] = useState('');

  const updateConfig = (categoryId: string, updates: Partial<CategoryConfig>) => {
    setConfigs(prev => ({
      ...prev,
      [categoryId]: { ...prev[categoryId], ...updates }
    }));
  };

  const addWebsite = (categoryId: string, website: string) => {
    if (!website.trim()) return;
    const config = configs[categoryId];
    if (!config.allowedWebsites.includes(website)) {
      updateConfig(categoryId, {
        allowedWebsites: [...config.allowedWebsites, website]
      });
    }
    setNewWebsite('');
  };

  const removeWebsite = (categoryId: string, website: string) => {
    updateConfig(categoryId, {
      allowedWebsites: configs[categoryId].allowedWebsites.filter(w => w !== website)
    });
  };

  const configuredCount = Object.values(configs).filter(c => c.permission !== 'deny').length;

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Security Configuration</h1>
          <p className="text-gray-400">
            Configure how ArmorClaw handles sensitive information
          </p>
        </div>
        <div className="text-right">
          <p className="text-sm text-gray-400">Configured</p>
          <p className="text-2xl font-bold">{configuredCount}/{DATA_CATEGORIES.length}</p>
        </div>
      </div>

      {/* Summary Card */}
      <div className="bg-gray-800/50 rounded-lg p-4">
        <div className="flex items-center gap-3">
          <Shield className="w-8 h-8 text-blue-400" />
          <div>
            <h3 className="font-semibold">Data Category Permissions</h3>
            <p className="text-sm text-gray-400">
              Each category controls what information ArmorClaw can access and where it can be used
            </p>
          </div>
        </div>
      </div>

      {/* Category Cards */}
      <div className="space-y-3">
        {DATA_CATEGORIES.map(category => (
          <CategoryCard
            key={category.id}
            category={category}
            config={configs[category.id]}
            isExpanded={expandedCategory === category.id}
            onToggle={() => setExpandedCategory(
              expandedCategory === category.id ? null : category.id
            )}
            onPermissionChange={(permission) => updateConfig(category.id, { permission })}
            onAddWebsite={(website) => addWebsite(category.id, website)}
            onRemoveWebsite={(website) => removeWebsite(category.id, website)}
            onRequiresApprovalChange={(requiresApproval) =>
              updateConfig(category.id, { requiresApproval })
            }
            newWebsite={newWebsite}
            onNewWebsiteChange={setNewWebsite}
          />
        ))}
      </div>

      {/* Save Button */}
      <div className="flex justify-end gap-3">
        <button className="px-4 py-2 text-gray-400 hover:text-white transition-colors">
          Reset to Defaults
        </button>
        <button className="px-6 py-2 bg-blue-500 hover:bg-blue-600 rounded-lg font-semibold transition-colors">
          Save Configuration
        </button>
      </div>
    </div>
  );
}

function CategoryCard({
  category,
  config,
  isExpanded,
  onToggle,
  onPermissionChange,
  onAddWebsite,
  onRemoveWebsite,
  onRequiresApprovalChange,
  newWebsite,
  onNewWebsiteChange
}: {
  category: typeof DATA_CATEGORIES[0];
  config: CategoryConfig;
  isExpanded: boolean;
  onToggle: () => void;
  onPermissionChange: (permission: PermissionLevel) => void;
  onAddWebsite: (website: string) => void;
  onRemoveWebsite: (website: string) => void;
  onRequiresApprovalChange: (requiresApproval: boolean) => void;
  newWebsite: string;
  onNewWebsiteChange: (value: string) => void;
}) {
  const riskColors = {
    high: 'bg-red-500/20 text-red-400',
    medium: 'bg-yellow-500/20 text-yellow-400',
    low: 'bg-green-500/20 text-green-400'
  };

  const permissionIcons = {
    deny: <X className="w-5 h-5 text-red-400" />,
    allow: <Check className="w-5 h-5 text-blue-400" />,
    allow_all: <AlertTriangle className="w-5 h-5 text-yellow-400" />
  };

  return (
    <div className="bg-gray-800/50 rounded-lg overflow-hidden">
      {/* Header - Always visible */}
      <button
        onClick={onToggle}
        className="w-full p-4 flex items-center justify-between hover:bg-gray-700/30 transition-colors"
      >
        <div className="flex items-center gap-3">
          <span className={`px-2 py-0.5 rounded text-xs uppercase font-semibold ${riskColors[category.riskLevel]}`}>
            {category.riskLevel}
          </span>
          <div className="text-left">
            <h3 className="font-semibold">{category.name}</h3>
            <p className="text-sm text-gray-400">{category.description}</p>
          </div>
        </div>

        <div className="flex items-center gap-3">
          {permissionIcons[config.permission]}
          <span className="text-sm uppercase">
            {config.permission === 'allow_all' ? 'All' : config.permission}
          </span>
          {isExpanded ? (
            <ChevronUp className="w-5 h-5 text-gray-400" />
          ) : (
            <ChevronDown className="w-5 h-5 text-gray-400" />
          )}
        </div>
      </button>

      {/* Expanded Content */}
      {isExpanded && (
        <div className="border-t border-gray-700 p-4 space-y-4">
          {/* Permission Selection */}
          <div>
            <h4 className="text-sm font-semibold mb-2">Overall Permission</h4>
            <div className="space-y-2">
              <PermissionOption
                selected={config.permission === 'deny'}
                onClick={() => onPermissionChange('deny')}
                title="Deny All"
                description="Never use this type of information"
                icon={<Lock className="w-4 h-4" />}
              />
              <PermissionOption
                selected={config.permission === 'allow'}
                onClick={() => onPermissionChange('allow')}
                title="Allow with Restrictions"
                description="Use only on specified websites (recommended)"
                icon={<Globe className="w-4 h-4" />}
                recommended
              />
              <PermissionOption
                selected={config.permission === 'allow_all'}
                onClick={() => onPermissionChange('allow_all')}
                title="Allow All"
                description="No restrictions (not recommended)"
                icon={<AlertTriangle className="w-4 h-4" />}
                dangerous
              />
            </div>
          </div>

          {/* Website Allowlist */}
          {config.permission === 'allow' && (
            <div>
              <h4 className="text-sm font-semibold mb-2">Allowed Websites</h4>
              <p className="text-xs text-gray-400 mb-2">
                This data can ONLY be used on these websites:
              </p>

              {/* Website Chips */}
              {config.allowedWebsites.length > 0 && (
                <div className="flex flex-wrap gap-2 mb-2">
                  {config.allowedWebsites.map(website => (
                    <span
                      key={website}
                      className="px-2 py-1 bg-blue-500/20 text-blue-300 rounded text-sm flex items-center gap-1"
                    >
                      {website}
                      <button
                        onClick={() => onRemoveWebsite(website)}
                        className="hover:text-red-400"
                      >
                        <X className="w-3 h-3" />
                      </button>
                    </span>
                  ))}
                </div>
              )}

              {/* Add Website Input */}
              <div className="flex gap-2">
                <input
                  type="text"
                  value={newWebsite}
                  onChange={(e) => onNewWebsiteChange(e.target.value)}
                  placeholder="example.com"
                  className="flex-1 px-3 py-2 bg-gray-700 border border-gray-600 rounded-lg text-sm focus:outline-none focus:border-blue-500"
                  onKeyDown={(e) => {
                    if (e.key === 'Enter') {
                      onAddWebsite(newWebsite);
                    }
                  }}
                />
                <button
                  onClick={() => onAddWebsite(newWebsite)}
                  className="px-4 py-2 bg-gray-700 hover:bg-gray-600 rounded-lg transition-colors"
                >
                  <Plus className="w-4 h-4" />
                </button>
              </div>
            </div>
          )}

          {/* Requires Approval Toggle */}
          {config.permission !== 'deny' && (
            <div className="flex items-center justify-between">
              <span className="text-sm">Require approval for this category</span>
              <button
                onClick={() => onRequiresApprovalChange(!config.requiresApproval)}
                className={`w-12 h-6 rounded-full transition-colors ${
                  config.requiresApproval ? 'bg-blue-500' : 'bg-gray-600'
                }`}
              >
                <div
                  className={`w-5 h-5 bg-white rounded-full transition-transform ${
                    config.requiresApproval ? 'translate-x-6' : 'translate-x-0.5'
                  }`}
                />
              </button>
            </div>
          )}
        </div>
      )}
    </div>
  );
}

function PermissionOption({
  selected,
  onClick,
  title,
  description,
  icon,
  recommended,
  dangerous
}: {
  selected: boolean;
  onClick: () => void;
  title: string;
  description: string;
  icon: React.ReactNode;
  recommended?: boolean;
  dangerous?: boolean;
}) {
  return (
    <button
      onClick={onClick}
      className={`w-full p-3 rounded-lg border-2 text-left transition-colors ${
        selected
          ? dangerous
            ? 'border-red-500/50 bg-red-500/10'
            : 'border-blue-500/50 bg-blue-500/10'
          : 'border-gray-700 hover:border-gray-600'
      }`}
    >
      <div className="flex items-start gap-3">
        <div className={`mt-0.5 ${dangerous ? 'text-red-400' : selected ? 'text-blue-400' : 'text-gray-400'}`}>
          {icon}
        </div>
        <div className="flex-1">
          <div className="flex items-center gap-2">
            <span className="font-semibold">{title}</span>
            {recommended && selected && (
              <span className="px-1.5 py-0.5 bg-blue-500 text-white text-xs rounded">
                RECOMMENDED
              </span>
            )}
          </div>
          <p className="text-sm text-gray-400">{description}</p>
        </div>
        <div
          className={`w-4 h-4 rounded-full border-2 ${
            selected
              ? dangerous
                ? 'border-red-500 bg-red-500'
                : 'border-blue-500 bg-blue-500'
              : 'border-gray-600'
          }`}
        >
          {selected && <Check className="w-3 h-3 text-white m-0.5" />}
        </div>
      </div>
    </button>
  );
}

export default SecurityConfig;
