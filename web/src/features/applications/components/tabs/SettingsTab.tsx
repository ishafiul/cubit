import React, { useState } from 'react';
import { Settings, Plus, Eye, EyeOff, Trash2, AlertTriangle, FileCode } from 'lucide-react';
import type { Application, EnvironmentVariable } from '../../../../api/model';
import {
  useUpdateApplication,
  useDeleteApplication,
} from '../../../../api/generated/applications/applications';
import {
  useSecretVisibility,
  useApplicationActions,
} from '../../stores/applicationStore';
import { useToastActions } from '../../../../shared/stores/useToastStore';
import { environmentVariableSchema, validateSchema } from '../../../../shared/utils/validation';
import { isOk } from '../../../../shared/utils/result';

export interface SettingsTabProps {
  app: Application;
  onDeleteApp?: (appId: string) => Promise<void>;
}

export function SettingsTab({ app, onDeleteApp }: SettingsTabProps) {
  const updateAppMutation = useUpdateApplication();
  const deleteAppMutation = useDeleteApplication();
  const { addToast } = useToastActions();
  const { toggleSecretVisibility } = useApplicationActions();

  const [envKey, setEnvKey] = useState('');
  const [envValue, setEnvValue] = useState('');
  const [isSecret, setIsSecret] = useState(false);
  const [validationError, setValidationError] = useState<string | null>(null);
  const [isDeleting, setIsDeleting] = useState(false);

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
    compatibility_date: '2024-04-03',
    compatibility_flags: ['nodejs_compat'],
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
