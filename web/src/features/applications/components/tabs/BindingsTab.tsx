import React, { useState } from 'react';
import { Layers, Plus, Database, HardDrive, Inbox, Box } from 'lucide-react';
import type { Application, ResourceBinding } from '../../../../api/model';
import { useUpdateApplication } from '../../../../api/generated/applications/applications';
import { useToastActions } from '../../../../shared/stores/useToastStore';
import { resourceBindingSchema, validateSchema } from '../../../../shared/utils/validation';
import { isOk } from '../../../../shared/utils/result';

export function BindingsTab({ app }: { app: Application }) {
  const updateAppMutation = useUpdateApplication();
  const { addToast } = useToastActions();

  const [bindingName, setBindingName] = useState('');
  const [bindingType, setBindingType] = useState<string>('kv');
  const [resourceId, setResourceId] = useState('');
  const [validationError, setValidationError] = useState<string | null>(null);

  const bindings: ResourceBinding[] = app.bindings || [];

  const handleAddBinding = async (e: React.FormEvent) => {
    e.preventDefault();
    setValidationError(null);

    const validationResult = validateSchema(resourceBindingSchema, {
      name: bindingName.trim(),
      type: bindingType,
      resourceId: resourceId.trim(),
    });

    if (!isOk(validationResult)) {
      setValidationError(validationResult.error.issues[0]?.message || 'Invalid binding configuration');
      return;
    }

    const newBinding = validationResult.value;
    const updatedBindings = [
      ...bindings.filter((b) => b.name !== newBinding.name),
      newBinding,
    ];

    try {
      await updateAppMutation.mutateAsync({
        id: app.id,
        data: {
          bindings: updatedBindings as any,
        },
      });
      addToast({
        title: 'Binding Added',
        description: `Bound ${newBinding.name} (${newBinding.type.toUpperCase()}) to worker`,
        variant: 'success',
      });
      setBindingName('');
      setResourceId('');
    } catch (err: any) {
      addToast({
        title: 'Failed to Save Binding',
        description: err?.message || 'Could not update application bindings.',
        variant: 'error',
      });
    }
  };

  const handleDeleteBinding = async (name: string) => {
    const updatedBindings = bindings.filter((b) => b.name !== name);
    try {
      await updateAppMutation.mutateAsync({
        id: app.id,
        data: {
          bindings: updatedBindings as any,
        },
      });
      addToast({
        title: 'Binding Removed',
        description: `Deleted binding ${name}`,
        variant: 'info',
      });
    } catch (err: any) {
      addToast({
        title: 'Failed to Remove Binding',
        description: err?.message || 'Could not remove binding.',
        variant: 'error',
      });
    }
  };

  const getBindingIcon = (type: string) => {
    switch (type) {
      case 'd1':
        return Database;
      case 'r2':
        return HardDrive;
      case 'queue':
        return Inbox;
      default:
        return Box;
    }
  };

  return (
    <div data-testid="tab-content-bindings" className="p-6 max-w-6xl mx-auto w-full space-y-6">
      <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-6">
        <h3 className="text-sm font-bold text-zinc-100 flex items-center gap-2 mb-1">
          <Layers className="w-4 h-4 text-emerald-400" />
          Cloudflare Compatible Resource Bindings
        </h3>
        <p className="text-xs text-zinc-400 mb-6">
          Connect KV namespaces, D1 databases, R2 buckets, Queues, and Services to your worker environment
        </p>

        {/* Add Binding Form */}
        <form onSubmit={handleAddBinding} className="grid grid-cols-1 sm:grid-cols-4 gap-2 mb-6">
          <input
            type="text"
            placeholder="BINDING_NAME"
            value={bindingName}
            onChange={(e) => setBindingName(e.target.value)}
            className="px-3 py-1.5 rounded-lg bg-zinc-950 border border-zinc-800 text-xs text-zinc-200 focus:outline-none focus:border-emerald-500 font-mono"
          />
          <select
            value={bindingType}
            onChange={(e) => setBindingType(e.target.value)}
            className="px-3 py-1.5 rounded-lg bg-zinc-950 border border-zinc-800 text-xs text-zinc-300 focus:outline-none focus:border-emerald-500"
          >
            <option value="kv">KV Namespace</option>
            <option value="d1">D1 Database</option>
            <option value="r2">R2 Bucket</option>
            <option value="service">Service (Worker-to-Worker)</option>
            <option value="queue">Queue</option>
            <option value="durable_object">Durable Object</option>
            <option value="workflow">Workflow</option>
          </select>
          <input
            type="text"
            placeholder="Target Resource ID / Bucket"
            value={resourceId}
            onChange={(e) => setResourceId(e.target.value)}
            className="px-3 py-1.5 rounded-lg bg-zinc-950 border border-zinc-800 text-xs text-zinc-200 focus:outline-none focus:border-emerald-500 font-mono"
          />
          <button
            type="submit"
            className="flex items-center justify-center gap-1.5 px-3 py-1.5 rounded-lg bg-emerald-600 hover:bg-emerald-500 text-xs font-semibold text-white transition"
          >
            <Plus className="w-3.5 h-3.5" />
            Attach Binding
          </button>
        </form>

        {validationError && (
          <p className="text-xs text-rose-400 -mt-4 mb-4">{validationError}</p>
        )}

        {bindings.length === 0 ? (
          <div className="p-6 rounded-lg bg-zinc-950/60 border border-zinc-800/60 text-xs text-zinc-500 text-center">
            No resource bindings attached to this application yet.
          </div>
        ) : (
          <div className="space-y-2">
            {bindings.map((b) => {
              const Icon = getBindingIcon(b.type || 'kv');
              return (
                <div
                  key={b.name}
                  className="flex items-center justify-between p-3.5 rounded-lg bg-zinc-950/60 border border-zinc-800/60"
                >
                  <div className="flex items-center gap-3">
                    <div className="p-2 rounded bg-zinc-900 border border-zinc-800 text-zinc-400">
                      <Icon className="w-4 h-4 text-emerald-400" />
                    </div>
                    <div>
                      <p className="text-xs font-mono font-bold text-zinc-200">{b.name}</p>
                      <p className="text-[11px] text-zinc-400 font-mono">
                        Type: <span className="uppercase text-zinc-300">{b.type}</span> • Target:{' '}
                        {b.resourceId}
                      </p>
                    </div>
                  </div>

                  <button
                    onClick={() => handleDeleteBinding(b.name)}
                    className="text-xs text-zinc-500 hover:text-rose-400 px-2.5 py-1 rounded border border-zinc-800 hover:border-rose-900 transition"
                  >
                    Delete
                  </button>
                </div>
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
}
