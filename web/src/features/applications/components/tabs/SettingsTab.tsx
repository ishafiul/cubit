import React, { useState, useMemo, useEffect } from 'react';
import {
  Settings,
  Plus,
  Eye,
  EyeOff,
  Trash2,
  AlertTriangle,
  AlertCircle,
  ShieldCheck,
  FileCode,
  Upload,
  Cpu,
  Search,
  X,
  Check,
  RefreshCw,
  Save,
} from 'lucide-react';
import type { Application, EnvironmentVariable } from '../../../../api/model';
import {
  useUpdateApplication,
  useDeleteApplication,
  useImportWranglerConfig,
} from '../../../../api/generated/applications/applications';
import {
  useSecretVisibility,
  useApplicationActions,
} from '../../stores/applicationStore';
import { useToastActions } from '../../../../shared/stores/useToastStore';
import { environmentVariableSchema, validateSchema } from '../../../../shared/utils/validation';
import { isOk } from '../../../../shared/utils/result';
import {
  analyzeCompatibility,
  analyzeApplicationCompatibility,
} from '../../utils/compatibility-linter';
import { CompatibilityReportModal } from '../modals/CompatibilityReportModal';

export interface SettingsTabProps {
  app: Application;
  onDeleteApp?: (appId: string) => Promise<void>;
  onRefreshApps?: () => void;
}

export function SettingsTab({ app, onDeleteApp, onRefreshApps }: SettingsTabProps) {
  const updateAppMutation = useUpdateApplication();
  const deleteAppMutation = useDeleteApplication();
  const importWranglerMutation = useImportWranglerConfig();
  const { addToast } = useToastActions();
  const { toggleSecretVisibility } = useApplicationActions();

  // Environment Variables Form State
  const [envKey, setEnvKey] = useState('');
  const [envValue, setEnvValue] = useState('');
  const [isSecret, setIsSecret] = useState(false);
  const [validationError, setValidationError] = useState<string | null>(null);

  // Environment Variables Filter & Search State
  const [varSearchTerm, setVarSearchTerm] = useState('');
  const [varTypeFilter, setVarTypeFilter] = useState<'ALL' | 'PLAIN' | 'SECRET'>('ALL');

  // Runtime & Compatibility Settings State
  const [compatDate, setCompatDate] = useState(app.compatibilityDate || '2024-04-03');
  const [nodejsCompat, setNodejsCompat] = useState(
    (app.compatibilityFlags || []).includes('nodejs_compat')
  );
  const [memoryLimitMB, setMemoryLimitMB] = useState(app.memoryLimitMb || 128);
  const [maxDurationMs, setMaxDurationMs] = useState(app.maxDurationMs || 50);
  const [isSavingRuntime, setIsSavingRuntime] = useState(false);
  const [runtimeSaveStatus, setRuntimeSaveStatus] = useState<'idle' | 'saved' | 'error'>('idle');

  useEffect(() => {
    setCompatDate(app.compatibilityDate || '2024-04-03');
    setNodejsCompat((app.compatibilityFlags || []).includes('nodejs_compat'));
    setMemoryLimitMB(app.memoryLimitMb || 128);
    setMaxDurationMs(app.maxDurationMs || 50);
  }, [app.compatibilityDate, app.compatibilityFlags, app.memoryLimitMb, app.maxDurationMs]);

  const [isDeleting, setIsDeleting] = useState(false);

  // Wrangler Import State
  const [showImportWranglerModal, setShowImportWranglerModal] = useState(false);
  const [rawWranglerConfig, setRawWranglerConfig] = useState('');
  const [isImportingWrangler, setIsImportingWrangler] = useState(false);

  // Celld Compatibility Report State
  const [showCompatModal, setShowCompatModal] = useState(false);
  const appCompatReport = useMemo(() => analyzeApplicationCompatibility(app), [app]);
  const liveCompatReport = useMemo(() => {
    if (!rawWranglerConfig.trim()) return null;
    return analyzeCompatibility(rawWranglerConfig);
  }, [rawWranglerConfig]);

  const envVars: EnvironmentVariable[] = app.envVars || [];

  const filteredEnvVars = useMemo(() => {
    return envVars.filter((v) => {
      if (varTypeFilter === 'PLAIN' && v.isSecret) return false;
      if (varTypeFilter === 'SECRET' && !v.isSecret) return false;
      if (varSearchTerm.trim()) {
        const q = varSearchTerm.toLowerCase().trim();
        const keyMatch = v.key.toLowerCase().includes(q);
        const valMatch = !v.isSecret && (v.value || '').toLowerCase().includes(q);
        return keyMatch || valMatch;
      }
      return true;
    });
  }, [envVars, varTypeFilter, varSearchTerm]);

  const handleAddEnvVar = async (e: React.FormEvent) => {
    e.preventDefault();
    setValidationError(null);

    const validationResult = validateSchema(environmentVariableSchema, {
      key: envKey.trim(),
      value: envValue,
      isSecret,
    });

    if (!isOk(validationResult)) {
      setValidationError(validationResult.error.issues[0]?.message || 'Invalid environment variable');
      return;
    }

    const newVar = validationResult.value;
    const updatedVars = [
      ...envVars.filter((v) => v.key !== newVar.key),
      newVar,
    ];

    try {
      await updateAppMutation.mutateAsync({
        id: app.id,
        data: {
          envVars: updatedVars,
        },
      });
      addToast({
        title: 'Variable Saved',
        description: `Configured ${newVar.key}`,
        variant: 'success',
      });
      setEnvKey('');
      setEnvValue('');
      setIsSecret(false);
      onRefreshApps?.();
    } catch (err: any) {
      addToast({
        title: 'Failed to Save Variable',
        description: err?.message || 'Could not update environment variables.',
        variant: 'error',
      });
    }
  };

  const handleDeleteEnvVar = async (key: string) => {
    const updatedVars = envVars.filter((v) => v.key !== key);
    try {
      await updateAppMutation.mutateAsync({
        id: app.id,
        data: {
          envVars: updatedVars,
        },
      });
      addToast({
        title: 'Variable Deleted',
        description: `Removed ${key}`,
        variant: 'info',
      });
      onRefreshApps?.();
    } catch (err: any) {
      addToast({
        title: 'Failed to Delete Variable',
        description: err?.message || 'Could not delete variable.',
        variant: 'error',
      });
    }
  };

  const handleSaveRuntimeSettings = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsSavingRuntime(true);
    setRuntimeSaveStatus('idle');

    try {
      await updateAppMutation.mutateAsync({
        id: app.id,
        data: {
          compatibilityDate: compatDate.trim() || undefined,
          compatibilityFlags: nodejsCompat ? ['nodejs_compat'] : [],
          memoryLimitMb: Number(memoryLimitMB),
          maxDurationMs: Number(maxDurationMs),
        },
      });
      setRuntimeSaveStatus('saved');
      addToast({
        title: 'Runtime Settings Saved',
        description: 'V8 isolate execution configuration updated successfully.',
        variant: 'success',
      });
      onRefreshApps?.();
      setTimeout(() => setRuntimeSaveStatus('idle'), 3000);
    } catch (err: any) {
      setRuntimeSaveStatus('error');
      addToast({
        title: 'Failed to Save Settings',
        description: err?.message || 'Could not update runtime settings.',
        variant: 'error',
      });
    } finally {
      setIsSavingRuntime(false);
    }
  };

  const handleDeleteApplication = async () => {
    if (!window.confirm(`Are you sure you want to permanently delete "${app.name}"?`)) {
      return;
    }
    setIsDeleting(true);
    try {
      if (onDeleteApp) {
        await onDeleteApp(app.id);
      } else {
        await deleteAppMutation.mutateAsync({ id: app.id });
        onRefreshApps?.();
      }
      addToast({
        title: 'Application Deleted',
        description: `Permanently removed ${app.name}`,
        variant: 'info',
      });
    } catch (err: any) {
      addToast({
        title: 'Deletion Failed',
        description: err?.message || 'Could not delete application.',
        variant: 'error',
      });
    } finally {
      setIsDeleting(false);
    }
  };

  const handleImportWrangler = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!rawWranglerConfig.trim()) return;

    setIsImportingWrangler(true);
    try {
      await importWranglerMutation.mutateAsync({
        id: app.id,
        data: {
          rawConfig: rawWranglerConfig,
        },
      });
      addToast({
        title: 'Wrangler Config Synced',
        description: 'Sanitized and applied routes, variables, and resource bindings.',
        variant: 'success',
      });
      setShowImportWranglerModal(false);
      setRawWranglerConfig('');
      onRefreshApps?.();
    } catch (err: any) {
      addToast({
        title: 'Import Failed',
        description: err?.message || 'Failed to parse and sync wrangler configuration.',
        variant: 'error',
      });
    } finally {
      setIsImportingWrangler(false);
    }
  };

  return (
    <div data-testid="tab-content-settings" className="p-6 max-w-6xl mx-auto w-full space-y-8">
      {/* Application Identity */}
      <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-6">
        <h3 className="text-sm font-bold text-zinc-100 flex items-center gap-2 mb-1">
          <Settings className="w-4 h-4 text-zinc-400" />
          Application Identity
        </h3>
        <p className="text-xs text-zinc-400 mb-4">
          Core metadata and identifiers registered in the Cubit cluster
        </p>

        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 text-xs font-mono">
          <div>
            <span className="text-zinc-500 block mb-1 font-sans">Application ID</span>
            <div className="p-2.5 rounded-lg bg-zinc-950 border border-zinc-800 text-zinc-300 select-all">
              {app.id}
            </div>
          </div>
          <div>
            <span className="text-zinc-500 block mb-1 font-sans">Cluster Ingress Subdomain</span>
            <div className="p-2.5 rounded-lg bg-zinc-950 border border-zinc-800 text-emerald-400 font-semibold select-all">
              {app.subdomain || app.name}.localhost
            </div>
          </div>
        </div>
      </div>

      {/* Runtime & Compatibility Configuration Form */}
      <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-6">
        <div className="flex items-center justify-between mb-4">
          <div>
            <h3 className="text-sm font-bold text-zinc-100 flex items-center gap-2 mb-1">
              <Cpu className="w-4 h-4 text-purple-400" />
              Runtime & Compatibility Settings
            </h3>
            <p className="text-xs text-zinc-400">
              Configure V8 isolate execution limits, CPU thresholds, and Cloudflare compatibility flags
            </p>
          </div>
          <div className="flex items-center gap-2">
            <button
              type="button"
              data-testid="celld-compat-btn"
              onClick={() => setShowCompatModal(true)}
              className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-zinc-800 hover:bg-zinc-700 text-zinc-300 border border-zinc-700 text-xs font-medium transition"
              title="Inspect Celld runtime compatibility report"
            >
              <Cpu className="w-3.5 h-3.5 text-purple-400" />
              Celld Compatibility
            </button>
            <button
              type="button"
              data-testid="sync-wrangler-btn"
              onClick={() => {
                setRawWranglerConfig('');
                setShowImportWranglerModal(true);
              }}
              className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-zinc-800 hover:bg-zinc-700 text-zinc-300 border border-zinc-700 text-xs font-medium transition"
              title="Sync configuration from wrangler.jsonc or wrangler.toml"
            >
              <Upload className="w-3.5 h-3.5 text-blue-400" />
              Sync Wrangler Config
            </button>
          </div>
        </div>

        <form onSubmit={handleSaveRuntimeSettings} className="space-y-4">
          <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
            <div>
              <label className="text-xs font-semibold text-zinc-400 block mb-1 font-sans">
                Compatibility Date
              </label>
              <input
                type="text"
                data-testid="runtime-compat-date-input"
                placeholder="2024-09-23"
                value={compatDate}
                onChange={(e) => setCompatDate(e.target.value)}
                className="w-full px-3 py-2 rounded-lg bg-zinc-950 border border-zinc-800 text-xs text-zinc-200 font-mono focus:outline-none focus:border-emerald-500"
              />
              <p className="text-[10px] text-zinc-500 mt-1">Cloudflare runtime version flag</p>
            </div>

            <div>
              <label className="text-xs font-semibold text-zinc-400 block mb-1 font-sans">
                Memory Limit (Isolate)
              </label>
              <select
                data-testid="runtime-memory-select"
                value={memoryLimitMB}
                onChange={(e) => setMemoryLimitMB(Number(e.target.value))}
                className="w-full px-3 py-2 rounded-lg bg-zinc-950 border border-zinc-800 text-xs text-zinc-200 font-mono focus:outline-none focus:border-emerald-500 cursor-pointer"
              >
                <option value={64}>64 MB (Micro)</option>
                <option value={128}>128 MB (Standard Worker)</option>
                <option value={256}>256 MB (Medium)</option>
                <option value={512}>512 MB (High)</option>
                <option value={1024}>1024 MB (Max)</option>
              </select>
              <p className="text-[10px] text-zinc-500 mt-1">V8 isolate RAM quota</p>
            </div>

            <div>
              <label className="text-xs font-semibold text-zinc-400 block mb-1 font-sans">
                CPU Execution Timeout (ms)
              </label>
              <input
                type="number"
                data-testid="runtime-cpu-timeout-input"
                min={5}
                max={30000}
                value={maxDurationMs}
                onChange={(e) => setMaxDurationMs(Number(e.target.value))}
                className="w-full px-3 py-2 rounded-lg bg-zinc-950 border border-zinc-800 text-xs text-zinc-200 font-mono focus:outline-none focus:border-emerald-500"
              />
              <p className="text-[10px] text-zinc-500 mt-1">Allowed CPU execution time</p>
            </div>
          </div>

          <div className="pt-2 flex items-center justify-between border-t border-zinc-800/80">
            <label className="flex items-center gap-2 text-xs text-zinc-300 cursor-pointer select-none">
              <input
                type="checkbox"
                data-testid="runtime-nodejs-compat-checkbox"
                checked={nodejsCompat}
                onChange={(e) => setNodejsCompat(e.target.checked)}
                className="rounded bg-zinc-950 border-zinc-800 text-emerald-500 focus:ring-0 w-4 h-4 cursor-pointer"
              />
              <span>Enable Node.js Compatibility (<code className="font-mono text-emerald-400">nodejs_compat</code>)</span>
            </label>

            <div className="flex items-center gap-2">
              {runtimeSaveStatus === 'saved' && (
                <span className="text-xs text-emerald-400 font-semibold flex items-center gap-1">
                  <Check className="w-3.5 h-3.5" /> Saved!
                </span>
              )}
              {runtimeSaveStatus === 'error' && (
                <span className="text-xs text-rose-400 font-semibold flex items-center gap-1">
                  <AlertCircle className="w-3.5 h-3.5" /> Failed saving settings
                </span>
              )}
              <button
                type="submit"
                data-testid="save-runtime-settings-btn"
                disabled={isSavingRuntime}
                className="flex items-center gap-1.5 px-4 py-2 rounded-lg bg-emerald-600 hover:bg-emerald-500 disabled:opacity-50 text-xs font-semibold text-white transition shadow"
              >
                {isSavingRuntime ? (
                  <RefreshCw className="w-3.5 h-3.5 animate-spin" />
                ) : (
                  <Save className="w-3.5 h-3.5" />
                )}
                <span>{isSavingRuntime ? 'Saving...' : 'Save Runtime Settings'}</span>
              </button>
            </div>
          </div>
        </form>
      </div>

      {/* Environment Variables & Secrets */}
      <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-6">
        <div className="flex items-center justify-between mb-4">
          <div>
            <h3 className="text-sm font-bold text-zinc-100 flex items-center gap-2 mb-1">
              <ShieldCheck className="w-4 h-4 text-emerald-400" />
              Environment Variables & Secrets
            </h3>
            <p className="text-xs text-zinc-400">
              Injected into worker execution context via <code className="font-mono text-emerald-400">env.*</code>
            </p>
          </div>
          <span className="text-xs font-mono text-zinc-400">
            {envVars.length} total ({envVars.filter((v) => v.isSecret).length} encrypted)
          </span>
        </div>

        {/* Search & Type Filter Bar */}
        <div className="flex flex-col sm:flex-row items-stretch sm:items-center justify-between gap-3 mb-4">
          <div className="relative flex-1 max-w-sm">
            <Search className="w-3.5 h-3.5 text-zinc-500 absolute left-3 top-1/2 -translate-y-1/2" />
            <input
              type="text"
              data-testid="var-search-input"
              value={varSearchTerm}
              onChange={(e) => setVarSearchTerm(e.target.value)}
              placeholder="Filter by variable name..."
              className="w-full pl-8 pr-8 py-1.5 rounded-lg bg-zinc-950 border border-zinc-800 text-xs text-zinc-200 placeholder-zinc-500 focus:outline-none focus:border-emerald-500 font-mono"
            />
            {varSearchTerm && (
              <button
                type="button"
                onClick={() => setVarSearchTerm('')}
                className="absolute right-2.5 top-1/2 -translate-y-1/2 text-zinc-500 hover:text-zinc-300 p-0.5"
              >
                <X className="w-3 h-3" />
              </button>
            )}
          </div>

          <div className="flex items-center gap-1.5 text-xs">
            <button
              type="button"
              data-testid="filter-all-btn"
              onClick={() => setVarTypeFilter('ALL')}
              className={`px-3 py-1 rounded-lg font-medium transition ${
                varTypeFilter === 'ALL'
                  ? 'bg-zinc-800 text-zinc-100 border border-zinc-700'
                  : 'text-zinc-400 hover:text-zinc-200 hover:bg-zinc-800/50'
              }`}
            >
              All ({envVars.length})
            </button>
            <button
              type="button"
              data-testid="filter-plain-btn"
              onClick={() => setVarTypeFilter('PLAIN')}
              className={`px-3 py-1 rounded-lg font-medium transition ${
                varTypeFilter === 'PLAIN'
                  ? 'bg-zinc-800 text-zinc-100 border border-zinc-700'
                  : 'text-zinc-400 hover:text-zinc-200 hover:bg-zinc-800/50'
              }`}
            >
              Plain Text ({envVars.filter((v) => !v.isSecret).length})
            </button>
            <button
              type="button"
              data-testid="filter-secret-btn"
              onClick={() => setVarTypeFilter('SECRET')}
              className={`px-3 py-1 rounded-lg font-medium transition ${
                varTypeFilter === 'SECRET'
                  ? 'bg-zinc-800 text-zinc-100 border border-zinc-700'
                  : 'text-zinc-400 hover:text-zinc-200 hover:bg-zinc-800/50'
              }`}
            >
              Encrypted Secrets ({envVars.filter((v) => v.isSecret).length})
            </button>
          </div>
        </div>

        {/* Add Variable Form */}
        <form onSubmit={handleAddEnvVar} className="grid grid-cols-1 sm:grid-cols-12 gap-2 mb-6">
          <div className="sm:col-span-4">
            <input
              type="text"
              data-testid="env-key-input"
              placeholder="VARIABLE_NAME"
              value={envKey}
              onChange={(e) => setEnvKey(e.target.value)}
              className="w-full px-3 py-1.5 rounded-lg bg-zinc-950 border border-zinc-800 text-xs text-zinc-200 focus:outline-none focus:border-emerald-500 font-mono"
            />
          </div>
          <div className="sm:col-span-5">
            <input
              type={isSecret ? 'password' : 'text'}
              data-testid="env-value-input"
              placeholder="Value"
              value={envValue}
              onChange={(e) => setEnvValue(e.target.value)}
              className="w-full px-3 py-1.5 rounded-lg bg-zinc-950 border border-zinc-800 text-xs text-zinc-200 focus:outline-none focus:border-emerald-500 font-mono"
            />
          </div>
          <div className="sm:col-span-3 flex items-center gap-2">
            <label className="flex items-center gap-1.5 text-xs text-zinc-400 select-none cursor-pointer">
              <input
                type="checkbox"
                data-testid="env-secret-checkbox"
                checked={isSecret}
                onChange={(e) => setIsSecret(e.target.checked)}
                className="rounded bg-zinc-950 border-zinc-800 text-emerald-500 focus:ring-0"
              />
              Secret
            </label>
            <button
              type="submit"
              data-testid="add-env-btn"
              className="flex-1 flex items-center justify-center gap-1 px-3 py-1.5 rounded-lg bg-emerald-600 hover:bg-emerald-500 text-xs font-semibold text-white transition"
            >
              <Plus className="w-3.5 h-3.5" />
              Add
            </button>
          </div>
        </form>

        {validationError && (
          <p className="text-xs text-rose-400 -mt-4 mb-4">{validationError}</p>
        )}

        {filteredEnvVars.length === 0 ? (
          <div className="p-4 rounded-lg bg-zinc-950/60 border border-zinc-800/60 text-xs text-zinc-500">
            {envVars.length === 0
              ? 'No environment variables configured.'
              : 'No variables match current filter criteria.'}
          </div>
        ) : (
          <div className="space-y-2">
            {filteredEnvVars.map((env) => (
              <EnvVarRow
                key={env.key}
                env={env}
                onToggleVisibility={toggleSecretVisibility}
                onDelete={handleDeleteEnvVar}
              />
            ))}
          </div>
        )}
      </div>

      {/* Worker Specification Preview */}
      <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-6">
        <h3 className="text-sm font-bold text-zinc-100 flex items-center gap-2 mb-1">
          <FileCode className="w-4 h-4 text-blue-400" />
          Worker Specification Preview
        </h3>
        <p className="text-xs text-zinc-400 mb-4">
          Generated Celld runtime manifest derived from active configuration
        </p>

        <pre className="p-4 rounded-lg bg-zinc-950 border border-zinc-800 font-mono text-xs text-zinc-300 overflow-x-auto">
{JSON.stringify(
  {
    name: app.name,
    compatibility_date: app.compatibilityDate || '2024-04-03',
    compatibility_flags: app.compatibilityFlags && app.compatibilityFlags.length > 0 ? app.compatibilityFlags : ['nodejs_compat'],
    memory_limit_mb: app.memoryLimitMb || 128,
    max_duration_ms: app.maxDurationMs || 50,
    vars: (app.envVars || []).reduce<Record<string, string>>((acc, v) => {
      acc[v.key] = v.isSecret ? '[REDACTED SECRET]' : v.value;
      return acc;
    }, {}),
    bindings: app.bindings || [],
  },
  null,
  2
)}
        </pre>
      </div>

      {/* Danger Zone */}
      <div className="bg-rose-950/20 border border-rose-900/40 rounded-xl p-6">
        <h3 className="text-sm font-bold text-rose-400 flex items-center gap-2 mb-1">
          <AlertTriangle className="w-4 h-4 text-rose-500" />
          Danger Zone
        </h3>
        <p className="text-xs text-zinc-400 mb-4">
          Permanently delete this worker isolate, domain bindings, and all deployed revisions from Garage S3.
        </p>

        <button
          type="button"
          data-testid="delete-application-btn"
          onClick={handleDeleteApplication}
          disabled={isDeleting}
          className="flex items-center gap-1.5 px-4 py-2 rounded-lg bg-rose-600 hover:bg-rose-500 text-xs font-semibold text-white transition disabled:opacity-50"
        >
          <Trash2 className="w-3.5 h-3.5" />
          {isDeleting ? 'Deleting...' : 'Delete Application'}
        </button>
      </div>

      {/* Sync Wrangler Config Modal */}
      {showImportWranglerModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm p-4">
          <div className="w-full max-w-xl bg-zinc-900 border border-zinc-800 rounded-2xl p-6 shadow-2xl space-y-4">
            <div className="flex items-center justify-between">
              <div>
                <h3 className="text-lg font-bold">Sync Wrangler Configuration</h3>
                <p className="text-xs text-zinc-400 mt-0.5">
                  App: <span className="font-mono text-emerald-400">{app.name}</span>
                </p>
              </div>
            </div>

            <div className="p-3 rounded-xl bg-zinc-950 border border-zinc-800/80 text-xs text-zinc-400 space-y-1">
              <div className="font-semibold text-zinc-300">Automatic Sanitization & Ingress Sync:</div>
              <p className="text-[11px] text-zinc-500">
                Upload or paste your <span className="font-mono text-zinc-300">wrangler.json</span>, <span className="font-mono text-zinc-300">wrangler.jsonc</span>, or <span className="font-mono text-zinc-300">wrangler.toml</span>.
                Cubit will sanitize prohibited keys (<code className="text-zinc-400">routes</code>, <code className="text-zinc-400">account_id</code>), convert routes to Traefik domains, and update bindings.
              </p>
            </div>

            <form onSubmit={handleImportWrangler} className="space-y-4">
              <div>
                <label className="text-xs font-semibold text-zinc-400 uppercase tracking-wider block mb-1">
                  Upload Wrangler File (.json, .jsonc, .toml)
                </label>
                <input
                  type="file"
                  accept=".json,.jsonc,.toml,text/plain"
                  onChange={(e) => {
                    const file = e.target.files?.[0];
                    if (file) {
                      const reader = new FileReader();
                      reader.onload = (ev) => {
                        if (typeof ev.target?.result === 'string') {
                          setRawWranglerConfig(ev.target.result);
                        }
                      };
                      reader.readAsText(file);
                    }
                  }}
                  className="w-full text-xs text-zinc-400 file:mr-3 file:py-1.5 file:px-3 file:rounded-lg file:border-0 file:text-xs file:font-semibold file:bg-zinc-800 file:text-zinc-200 hover:file:bg-zinc-700 cursor-pointer"
                />
              </div>

              <div>
                <label className="text-xs font-semibold text-zinc-400 uppercase tracking-wider block mb-1">
                  Or Paste Configuration
                </label>
                <textarea
                  rows={8}
                  required
                  placeholder='{&#10;  "name": "my-worker",&#10;  "main": "src/index.ts",&#10;  "compatibility_date": "2024-09-23",&#10;  "kv_namespaces": [{ "binding": "KV", "id": "my-kv" }]&#10;}'
                  value={rawWranglerConfig}
                  onChange={(e) => setRawWranglerConfig(e.target.value)}
                  className="w-full p-3 rounded-xl bg-zinc-950 border border-zinc-800 font-mono text-xs text-zinc-200 focus:outline-none focus:border-emerald-500"
                />
              </div>

              {/* Pre-Flight Live Compatibility Banner */}
              {liveCompatReport && (
                <div
                  className={`p-3 rounded-xl border text-xs space-y-2 ${
                    liveCompatReport.level === 'compatible'
                      ? 'bg-emerald-950/20 border-emerald-800/50 text-emerald-300'
                      : liveCompatReport.level === 'warning'
                      ? 'bg-amber-950/20 border-amber-800/50 text-amber-300'
                      : 'bg-rose-950/20 border-rose-800/50 text-rose-300'
                  }`}
                >
                  <div className="flex items-center justify-between font-semibold">
                    <div className="flex items-center gap-1.5">
                      {liveCompatReport.level === 'compatible' ? (
                        <ShieldCheck className="w-4 h-4 text-emerald-400" />
                      ) : liveCompatReport.level === 'warning' ? (
                        <AlertTriangle className="w-4 h-4 text-amber-400" />
                      ) : (
                        <AlertCircle className="w-4 h-4 text-rose-400" />
                      )}
                      <span>
                        Pre-Flight Check:{' '}
                        {liveCompatReport.level === 'compatible'
                          ? 'Fully Compatible with Celld'
                          : liveCompatReport.level === 'warning'
                          ? 'Deployable with Warnings'
                          : 'Incompatible Features Detected'}
                      </span>
                    </div>
                    <span className="font-mono text-[10px] text-zinc-400">
                      {liveCompatReport.supportedCount} supported • {liveCompatReport.warningCount} warnings •{' '}
                      {liveCompatReport.unsupportedCount} incompatible
                    </span>
                  </div>
                  <p className="text-[11px] text-zinc-400 leading-normal font-mono">
                    {liveCompatReport.summary}
                  </p>
                  {liveCompatReport.unsupportedFeatures.length > 0 && (
                    <div className="pt-1 text-[11px] space-y-1">
                      {liveCompatReport.unsupportedFeatures.map((f, i) => (
                        <div key={i} className="p-2 rounded bg-zinc-950/60 border border-rose-900/40">
                          <span className="font-semibold text-rose-300">• {f.name}: </span>
                          <span className="text-zinc-400">{f.remediation || f.details}</span>
                        </div>
                      ))}
                    </div>
                  )}
                </div>
              )}

              <div className="flex justify-end gap-3 pt-2">
                <button
                  type="button"
                  onClick={() => setShowImportWranglerModal(false)}
                  className="px-4 py-2 rounded-lg border border-zinc-700 bg-zinc-800 hover:bg-zinc-700 text-xs font-semibold text-zinc-300 transition"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  disabled={isImportingWrangler}
                  className="px-4 py-2 rounded-lg bg-emerald-600 hover:bg-emerald-500 disabled:opacity-50 text-xs font-semibold text-white transition shadow"
                >
                  {isImportingWrangler ? 'Sanitizing & Syncing...' : 'Sanitize & Sync Config'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Compatibility Report Modal */}
      {showCompatModal && (
        <CompatibilityReportModal
          isOpen={showCompatModal}
          report={appCompatReport}
          onClose={() => setShowCompatModal(false)}
        />
      )}
    </div>
  );
}

function EnvVarRow({
  env,
  onToggleVisibility,
  onDelete,
}: {
  env: EnvironmentVariable;
  onToggleVisibility: (key: string) => void;
  onDelete: (key: string) => void;
}) {
  const isVisible = useSecretVisibility(env.key);

  return (
    <div
      data-testid={`env-row-${env.key}`}
      className="flex items-center justify-between p-3 rounded-lg bg-zinc-950/60 border border-zinc-800/60 font-mono text-xs"
    >
      <div className="flex items-center gap-3 overflow-hidden">
        <span className="font-semibold text-zinc-200 shrink-0">{env.key}</span>
        <span className="text-zinc-600 shrink-0">=</span>
        <span className="text-zinc-400 truncate">
          {env.isSecret && !isVisible ? '••••••••••••••••' : env.value}
        </span>
      </div>

      <div className="flex items-center gap-2 shrink-0 ml-4">
        {env.isSecret && (
          <button
            type="button"
            data-testid={`toggle-secret-${env.key}`}
            onClick={() => onToggleVisibility(env.key)}
            className="p-1 rounded hover:bg-zinc-800 text-zinc-400 hover:text-zinc-200 transition"
            title={isVisible ? 'Hide secret' : 'Reveal secret'}
          >
            {isVisible ? <EyeOff className="w-3.5 h-3.5" /> : <Eye className="w-3.5 h-3.5" />}
          </button>
        )}
        <button
          type="button"
          data-testid={`delete-env-${env.key}`}
          onClick={() => onDelete(env.key)}
          className="p-1 rounded hover:bg-zinc-800 text-zinc-500 hover:text-rose-400 transition"
          title="Delete variable"
        >
          <Trash2 className="w-3.5 h-3.5" />
        </button>
      </div>
    </div>
  );
}
