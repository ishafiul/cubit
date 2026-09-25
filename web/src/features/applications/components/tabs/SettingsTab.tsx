import React, { useState, useMemo } from 'react';
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
}

export function SettingsTab({ app, onDeleteApp }: SettingsTabProps) {
  const updateAppMutation = useUpdateApplication();
  const deleteAppMutation = useDeleteApplication();
  const importWranglerMutation = useImportWranglerConfig();
  const { addToast } = useToastActions();
  const { toggleSecretVisibility } = useApplicationActions();

  const [envKey, setEnvKey] = useState('');
  const [envValue, setEnvValue] = useState('');
  const [isSecret, setIsSecret] = useState(false);
  const [validationError, setValidationError] = useState<string | null>(null);
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
    } catch (err: any) {
      addToast({
        title: 'Failed to Delete Variable',
        description: err?.message || 'Could not delete variable.',
        variant: 'error',
      });
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
      }
      addToast({
        title: 'Application Deleted',
        description: `Successfully deleted ${app.name}`,
        variant: 'success',
      });
    } catch (err: any) {
      addToast({
        title: 'Deletion Failed',
        description: err?.message || 'Could not delete application.',
        variant: 'error',
      });
      setIsDeleting(false);
    }
  };

  const handleImportWrangler = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!rawWranglerConfig.trim()) return;
    setIsImportingWrangler(true);
    try {
      const res = await importWranglerMutation.mutateAsync({
        id: app.id,
        data: {
          rawConfig: rawWranglerConfig,
          format: 'auto',
        },
      });
      addToast({
        title: 'Wrangler Config Synced',
        description: res.message || 'Updated configuration and extracted routes.',
        variant: 'success',
      });
      setRawWranglerConfig('');
      setShowImportWranglerModal(false);
    } catch (err: any) {
      addToast({
        title: 'Import Failed',
        description: err?.message || 'Could not parse or apply wrangler configuration.',
        variant: 'error',
      });
    } finally {
      setIsImportingWrangler(false);
    }
  };

  return (
    <div data-testid="tab-content-settings" className="p-6 max-w-6xl mx-auto w-full space-y-8">
      {/* Environment Variables */}
      <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-6">
        <h3 className="text-sm font-bold text-zinc-100 flex items-center gap-2 mb-1">
          <Settings className="w-4 h-4 text-emerald-400" />
          Environment Variables & Secrets
        </h3>
        <p className="text-xs text-zinc-400 mb-6">
          Injected into the worker isolate runtime context (<code className="font-mono text-emerald-400">env.KEY</code>)
        </p>

        <form onSubmit={handleAddEnvVar} className="grid grid-cols-1 sm:grid-cols-12 gap-2 mb-6">
          <div className="sm:col-span-4">
            <input
              type="text"
              placeholder="VARIABLE_NAME"
              value={envKey}
              onChange={(e) => setEnvKey(e.target.value)}
              className="w-full px-3 py-1.5 rounded-lg bg-zinc-950 border border-zinc-800 text-xs text-zinc-200 focus:outline-none focus:border-emerald-500 font-mono"
            />
          </div>
          <div className="sm:col-span-5">
            <input
              type={isSecret ? 'password' : 'text'}
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
                checked={isSecret}
                onChange={(e) => setIsSecret(e.target.checked)}
                className="rounded bg-zinc-950 border-zinc-800 text-emerald-500 focus:ring-0"
              />
              Secret
            </label>
            <button
              type="submit"
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

        {envVars.length === 0 ? (
          <div className="p-4 rounded-lg bg-zinc-950/60 border border-zinc-800/60 text-xs text-zinc-500">
            No environment variables configured.
          </div>
        ) : (
          <div className="space-y-2">
            {envVars.map((env) => (
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

      {/* Wrangler Configuration Spec */}
      <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-6">
        <div className="flex items-center justify-between mb-4">
          <div>
            <h3 className="text-sm font-bold text-zinc-100 flex items-center gap-2 mb-1">
              <FileCode className="w-4 h-4 text-blue-400" />
              Worker Specification Preview
            </h3>
            <p className="text-xs text-zinc-400">
              Generated Celld runtime manifest derived from active configuration
            </p>
          </div>
          <div className="flex items-center gap-2">
            <button
              type="button"
              onClick={() => setShowCompatModal(true)}
              className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-zinc-800 hover:bg-zinc-700 text-zinc-300 border border-zinc-700 text-xs font-medium transition"
              title="Inspect Celld runtime compatibility report"
            >
              <Cpu className="w-3.5 h-3.5 text-purple-400" />
              Celld Compatibility
            </button>
            <button
              type="button"
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
        <pre className="p-4 rounded-lg bg-zinc-950 border border-zinc-800 font-mono text-xs text-zinc-300 overflow-x-auto">
{JSON.stringify(
  {
    name: app.name,
    compatibility_date: app.compatibilityDate || '2024-04-03',
    compatibility_flags: app.compatibilityFlags && app.compatibilityFlags.length > 0 ? app.compatibilityFlags : ['nodejs_compat'],
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
                  disabled={isImportingWrangler}
                  className="px-4 py-2 rounded-lg text-sm text-zinc-400 hover:text-white transition disabled:opacity-50"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  disabled={isImportingWrangler || !rawWranglerConfig.trim()}
                  className="flex items-center gap-2 px-4 py-2 rounded-lg bg-emerald-500 hover:bg-emerald-600 disabled:opacity-50 text-zinc-950 text-sm font-semibold transition"
                >
                  <Upload className={`w-4 h-4 ${isImportingWrangler ? 'animate-spin' : ''}`} />
                  {isImportingWrangler ? 'Syncing...' : 'Sync Configuration'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Celld Compatibility Report Modal */}
      <CompatibilityReportModal
        isOpen={showCompatModal}
        onClose={() => setShowCompatModal(false)}
        report={appCompatReport}
        title={`Celld Compatibility Report • ${app.name}`}
      />
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
  const displayValue = env.isSecret && !isVisible ? '••••••••••••••••' : env.value;

  return (
    <div className="flex items-center justify-between p-3 rounded-lg bg-zinc-950/60 border border-zinc-800/60">
      <div className="flex items-center gap-3">
        <span className="text-xs font-mono font-bold text-zinc-200">{env.key}</span>
        <span className="text-xs font-mono text-zinc-400">= {displayValue}</span>
        {env.isSecret && (
          <span className="text-[10px] font-bold px-1.5 py-0.5 rounded bg-zinc-800 text-amber-400 border border-zinc-700">
            SECRET
          </span>
        )}
      </div>

      <div className="flex items-center gap-2">
        {env.isSecret && (
          <button
            onClick={() => onToggleVisibility(env.key)}
            className="p-1 rounded text-zinc-500 hover:text-zinc-300 transition"
          >
            {isVisible ? <EyeOff className="w-3.5 h-3.5" /> : <Eye className="w-3.5 h-3.5" />}
          </button>
        )}
        <button
          onClick={() => onDelete(env.key)}
          className="p-1 rounded text-zinc-500 hover:text-rose-400 transition"
        >
          <Trash2 className="w-3.5 h-3.5" />
        </button>
      </div>
    </div>
  );
}
