import React, { useState } from 'react';
import {
  Layers,
  Plus,
  Database,
  HardDrive,
  Inbox,
  Box,
  Folder,
  Server,
  Cpu,
  GitBranch,
  Clock,
  RefreshCw,
  ArrowRight,
  ExternalLink,
  Trash2,
} from 'lucide-react';
import type { Application, ResourceBinding } from '../../../../api/model';
import { useUpdateApplication } from '../../../../api/generated/applications/applications';
import { useToastActions } from '../../../../shared/stores/useToastStore';
import { resourceBindingSchema, validateSchema } from '../../../../shared/utils/validation';
import { isOk } from '../../../../shared/utils/result';
import {
  useFleetServices,
  getServiceTabForBindingType,
} from '../../hooks/useFleetServices';

export interface BindingsTabProps {
  app: Application;
  onNavigateToService?: (targetTab: string, resourceId?: string) => void;
}

export function BindingsTab({ app, onNavigateToService }: BindingsTabProps) {
  const updateAppMutation = useUpdateApplication();
  const { addToast } = useToastActions();

  const {
    attachedCron,
    attachedQueues,
    attachedDO,
    attachedWorkflows,
    isLoading: isLoadingFleet,
    refetch: refetchFleet,
  } = useFleetServices(app);

  const [bindingName, setBindingName] = useState('');
  const [bindingType, setBindingType] = useState<string>('kv_namespace');
  const [resourceId, setResourceId] = useState('');
  const [validationError, setValidationError] = useState<string | null>(null);

  const bindings: ResourceBinding[] = app.bindings || [];
  const totalAttachedFleet =
    attachedCron.length + attachedQueues.length + attachedDO.length + attachedWorkflows.length;

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
      case 'd1_database':
        return Database;
      case 'r2':
      case 'r2_bucket':
        return HardDrive;
      case 'queue':
        return Inbox;
      case 'workflow':
        return GitBranch;
      case 'durable_object':
        return Cpu;
      case 'container':
        return Server;
      case 'assets':
        return Folder;
      default:
        return Box;
    }
  };

  return (
    <div data-testid="tab-content-bindings" className="p-6 max-w-6xl mx-auto w-full space-y-8">
      {/* Section 1: Injected Cloudflare Resource Bindings (env.*) */}
      <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-6">
        <h3 className="text-sm font-bold text-zinc-100 flex items-center gap-2 mb-1">
          <Layers className="w-4 h-4 text-emerald-400" />
          Cloudflare Compatible Resource Bindings (env.*)
        </h3>
        <p className="text-xs text-zinc-400 mb-6">
          Connect KV namespaces, D1 databases, R2 buckets, Queues, Workflows, Containers, and Services directly to your worker isolate environment
        </p>

        <form onSubmit={handleAddBinding} className="grid grid-cols-1 sm:grid-cols-12 gap-2 mb-6">
          <div className="sm:col-span-4">
            <input
              type="text"
              data-testid="binding-name-input"
              placeholder="BINDING_NAME"
              value={bindingName}
              onChange={(e) => setBindingName(e.target.value)}
              className="w-full px-3 py-1.5 rounded-lg bg-zinc-950 border border-zinc-800 text-xs text-zinc-200 focus:outline-none focus:border-emerald-500 font-mono"
            />
          </div>
          <div className="sm:col-span-3">
            <select
              data-testid="binding-type-select"
              value={bindingType}
              onChange={(e) => setBindingType(e.target.value)}
              className="w-full px-3 py-1.5 rounded-lg bg-zinc-950 border border-zinc-800 text-xs text-zinc-200 focus:outline-none focus:border-emerald-500 font-mono"
            >
              <option value="kv">KV Namespace</option>
              <option value="d1">D1 Database</option>
              <option value="r2">R2 Bucket</option>
              <option value="queue">Queue Producer</option>
              <option value="workflow">Workflow</option>
              <option value="durable_object">Durable Object</option>
              <option value="container">Service Container</option>
              <option value="assets">Static Assets</option>
            </select>
          </div>
          <div className="sm:col-span-3">
            <input
              type="text"
              data-testid="binding-resource-id-input"
              placeholder="Resource Name / ID"
              value={resourceId}
              onChange={(e) => setResourceId(e.target.value)}
              className="w-full px-3 py-1.5 rounded-lg bg-zinc-950 border border-zinc-800 text-xs text-zinc-200 focus:outline-none focus:border-emerald-500 font-mono"
            />
          </div>
          <div className="sm:col-span-2">
            <button
              type="submit"
              data-testid="add-binding-btn"
              className="w-full flex items-center justify-center gap-1 px-3 py-1.5 rounded-lg bg-emerald-600 hover:bg-emerald-500 text-xs font-semibold text-white transition shadow"
            >
              <Plus className="w-3.5 h-3.5" />
              Bind
            </button>
          </div>
        </form>

        {validationError && (
          <p className="text-xs text-rose-400 -mt-4 mb-4">{validationError}</p>
        )}

        {bindings.length === 0 ? (
          <div className="p-4 rounded-lg bg-zinc-950/60 border border-zinc-800/60 text-xs text-zinc-500">
            No injected resource bindings configured.
          </div>
        ) : (
          <div className="space-y-2">
            {bindings.map((b) => {
              const Icon = getBindingIcon(b.type);
              const tabKey = getServiceTabForBindingType(b.type);

              return (
                <div
                  key={b.name}
                  data-testid={`binding-row-${b.name}`}
                  className="flex items-center justify-between p-3 rounded-lg bg-zinc-950/60 border border-zinc-800/60 font-mono text-xs"
                >
                  <div className="flex items-center gap-3 overflow-hidden">
                    <Icon className="w-4 h-4 text-emerald-400 shrink-0" />
                    <span className="font-semibold text-zinc-200">{b.name}</span>
                    <span className="text-zinc-600">→</span>
                    <span className="text-zinc-400 truncate">
                      {b.type.toUpperCase()}: {b.resourceId}
                    </span>
                  </div>

                  <div className="flex items-center gap-2 shrink-0 ml-4">
                    {onNavigateToService && (
                      <button
                        type="button"
                        onClick={() => onNavigateToService(tabKey, b.resourceId)}
                        className="flex items-center gap-1 text-[11px] text-zinc-400 hover:text-zinc-200 bg-zinc-800/80 hover:bg-zinc-700 px-2 py-1 rounded transition"
                        title="Open resource in service manager"
                      >
                        <span>Open</span>
                        <ExternalLink className="w-3 h-3 text-zinc-400" />
                      </button>
                    )}
                    <button
                      type="button"
                      data-testid={`delete-binding-${b.name}`}
                      onClick={() => handleDeleteBinding(b.name)}
                      className="p-1 rounded hover:bg-zinc-800 text-zinc-500 hover:text-rose-400 transition"
                      title="Remove binding"
                    >
                      <Trash2 className="w-3.5 h-3.5" />
                    </button>
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </div>

      {/* Section 2: Attached Fleet Services (Reverse Bindings) */}
      <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-6">
        <div className="flex items-center justify-between mb-4">
          <div>
            <h3 className="text-sm font-bold text-zinc-100 flex items-center gap-2 mb-1">
              <Server className="w-4 h-4 text-indigo-400" />
              Attached Fleet Services & Triggers
            </h3>
            <p className="text-xs text-zinc-400">
              External services configured across your fleet that invoke or target this worker.
            </p>
          </div>

          <button
            type="button"
            onClick={() => refetchFleet()}
            disabled={isLoadingFleet}
            className="p-1.5 rounded-lg border border-zinc-800 hover:bg-zinc-800 text-zinc-400 hover:text-zinc-200 transition"
            title="Refresh fleet services"
          >
            <RefreshCw className={`w-3.5 h-3.5 ${isLoadingFleet ? 'animate-spin text-emerald-400' : ''}`} />
          </button>
        </div>

        {totalAttachedFleet === 0 ? (
          <div className="p-6 rounded-xl border border-dashed border-zinc-800 bg-zinc-950/40 text-center space-y-2">
            <p className="text-xs text-zinc-400">No Fleet Services Attached</p>
            <p className="text-[11px] text-zinc-600 max-w-sm mx-auto">
              Configure Crons, Queues, Workflows, or Durable Objects to target this application to see them listed here.
            </p>
          </div>
        ) : (
          <div className="space-y-4">
            {/* Cron Triggers */}
            {attachedCron.length > 0 && (
              <div className="space-y-2">
                <h4 className="text-xs font-semibold text-zinc-400 uppercase tracking-wider flex items-center gap-1.5">
                  <Clock className="w-3.5 h-3.5 text-sky-400" /> Attached Cron Triggers ({attachedCron.length})
                </h4>
                <div className="space-y-1.5">
                  {attachedCron.map((cron) => (
                    <div
                      key={cron.id}
                      className="p-3 rounded-lg bg-zinc-950/60 border border-zinc-800/60 flex items-center justify-between text-xs"
                    >
                      <div className="flex items-center gap-2.5">
                        <span className="font-semibold text-zinc-200">{cron.name}</span>
                        <span className="font-mono text-[10px] px-2 py-0.5 rounded bg-zinc-800 text-emerald-400 border border-zinc-700">
                          {cron.cronExpression || cron.cron}
                        </span>
                        <span className="text-[10px] text-zinc-500 font-mono">
                          Last run: {cron.lastRunAt || cron.last_run_at ? new Date(cron.lastRunAt || cron.last_run_at || '').toLocaleTimeString() : 'Never'}
                        </span>
                      </div>
                      <button
                        type="button"
                        onClick={() => onNavigateToService?.('cron', cron.id)}
                        className="flex items-center gap-1 text-[11px] px-2.5 py-1 rounded bg-zinc-800 hover:bg-zinc-700 text-zinc-300 font-medium transition"
                      >
                        <span>Open in Cron</span>
                        <ArrowRight className="w-3 h-3 text-zinc-400" />
                      </button>
                    </div>
                  ))}
                </div>
              </div>
            )}

            {/* Queue Consumers */}
            {attachedQueues.length > 0 && (
              <div className="space-y-2">
                <h4 className="text-xs font-semibold text-zinc-400 uppercase tracking-wider flex items-center gap-1.5">
                  <Inbox className="w-3.5 h-3.5 text-emerald-400" /> Attached Queue Consumers ({attachedQueues.length})
                </h4>
                <div className="space-y-1.5">
                  {attachedQueues.map((q) => (
                    <div
                      key={q.id}
                      className="p-3 rounded-lg bg-zinc-950/60 border border-zinc-800/60 flex items-center justify-between text-xs"
                    >
                      <div className="flex items-center gap-2.5">
                        <span className="font-semibold text-zinc-200">{q.name}</span>
                        <span className="text-[10px] px-2 py-0.5 rounded-full font-bold uppercase bg-emerald-950 text-emerald-400 border border-emerald-800">
                          Consumer Active
                        </span>
                        <span className="text-[11px] text-zinc-500 font-mono">
                          Batch: {q.maxBatchSize || q.max_batch_size || 10} • Retries: {q.maxRetries || q.max_retries || 3}
                        </span>
                      </div>
                      <button
                        type="button"
                        onClick={() => onNavigateToService?.('queues', q.id)}
                        className="flex items-center gap-1 text-[11px] px-2.5 py-1 rounded bg-zinc-800 hover:bg-zinc-700 text-zinc-300 font-medium transition"
                      >
                        <span>Open Queue</span>
                        <ArrowRight className="w-3 h-3 text-emerald-400" />
                      </button>
                    </div>
                  ))}
                </div>
              </div>
            )}

            {/* Durable Objects */}
            {attachedDO.length > 0 && (
              <div className="space-y-2">
                <h4 className="text-xs font-semibold text-zinc-400 uppercase tracking-wider flex items-center gap-1.5">
                  <Cpu className="w-3.5 h-3.5 text-purple-400" /> Attached Durable Object Namespaces ({attachedDO.length})
                </h4>
                <div className="space-y-1.5">
                  {attachedDO.map((doCls) => (
                    <div
                      key={doCls.id}
                      className="p-3 rounded-lg bg-zinc-950/60 border border-zinc-800/60 flex items-center justify-between text-xs"
                    >
                      <div className="flex items-center gap-2.5">
                        <span className="font-semibold text-zinc-200">{doCls.name}</span>
                        <span className="text-[10px] px-2 py-0.5 rounded-full font-bold uppercase bg-purple-950 text-purple-300 border border-purple-800">
                          Durable Object
                        </span>
                        {doCls.facets && doCls.facets.length > 0 && (
                          <span className="text-[10px] text-zinc-500 font-mono">
                            {doCls.facets.length} Facets
                          </span>
                        )}
                      </div>
                      <button
                        type="button"
                        onClick={() => onNavigateToService?.('durable-objects', doCls.id)}
                        className="flex items-center gap-1 text-[11px] px-2.5 py-1 rounded bg-zinc-800 hover:bg-zinc-700 text-zinc-300 font-medium transition"
                      >
                        <span>Open DO</span>
                        <ArrowRight className="w-3 h-3 text-purple-400" />
                      </button>
                    </div>
                  ))}
                </div>
              </div>
            )}

            {/* Workflows */}
            {attachedWorkflows.length > 0 && (
              <div className="space-y-2">
                <h4 className="text-xs font-semibold text-zinc-400 uppercase tracking-wider flex items-center gap-1.5">
                  <GitBranch className="w-3.5 h-3.5 text-indigo-400" /> Attached Workflows ({attachedWorkflows.length})
                </h4>
                <div className="space-y-1.5">
                  {attachedWorkflows.map((wf) => (
                    <div
                      key={wf.id}
                      className="p-3 rounded-lg bg-zinc-950/60 border border-zinc-800/60 flex items-center justify-between text-xs"
                    >
                      <div className="flex items-center gap-2.5">
                        <span className="font-semibold text-zinc-200">{wf.name}</span>
                        <span className="text-[10px] px-2 py-0.5 rounded-full font-bold uppercase bg-indigo-950 text-indigo-300 border border-indigo-800">
                          Workflow Target
                        </span>
                        {wf.steps && (
                          <span className="text-[11px] text-zinc-500 font-mono">
                            {wf.steps.length} Steps
                          </span>
                        )}
                      </div>
                      <button
                        type="button"
                        onClick={() => onNavigateToService?.('workflows', wf.id)}
                        className="flex items-center gap-1 text-[11px] px-2.5 py-1 rounded bg-zinc-800 hover:bg-zinc-700 text-zinc-300 font-medium transition"
                      >
                        <span>Open Workflow</span>
                        <ArrowRight className="w-3 h-3 text-indigo-400" />
                      </button>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
}
