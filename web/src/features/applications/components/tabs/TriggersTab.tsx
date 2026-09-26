import React, { useState } from 'react';
import {
  Globe,
  Plus,
  Clock,
  ShieldCheck,
  Inbox,
  Play,
  RefreshCw,
  ArrowRight,
  X,
} from 'lucide-react';
import type { Application } from '../../../../api/model';
import { useListDomains, useCreateDomain } from '../../../../api/generated/domains/domains';
import { useToastActions } from '../../../../shared/stores/useToastStore';
import { domainSchema, validateSchema } from '../../../../shared/utils/validation';
import { isOk } from '../../../../shared/utils/result';
import { useFleetServices } from '../../hooks/useFleetServices';

export interface TriggersTabProps {
  app: Application;
  onNavigateToService?: (targetTab: string, resourceId?: string) => void;
}

export function TriggersTab({ app, onNavigateToService }: TriggersTabProps) {
  const { data: domains, refetch: refetchDomains, isLoading: isLoadingDomains } = useListDomains();
  const createDomainMutation = useCreateDomain();
  const { addToast } = useToastActions();

  const {
    attachedCron,
    attachedQueues,
    runCronNow,
    createCron,
    isCreatingCron,
  } = useFleetServices(app);

  const [hostname, setHostname] = useState('');
  const [validationError, setValidationError] = useState<string | null>(null);

  // Add Cron Modal State
  const [showAddCronModal, setShowAddCronModal] = useState(false);
  const [cronName, setCronName] = useState('');
  const [cronExpr, setCronExpr] = useState('*/5 * * * *');
  const [cronError, setCronError] = useState<string | null>(null);

  const [runningCronId, setRunningCronId] = useState<string | null>(null);

  const appDomains = Array.isArray(domains)
    ? domains.filter((d) => d.applicationId === app.id)
    : [];

  const subdomain = app.subdomain || app.name;

  const handleAddDomain = async (e: React.FormEvent) => {
    e.preventDefault();
    setValidationError(null);

    const validationResult = validateSchema(domainSchema, {
      hostname: hostname.trim(),
      pathPrefix: '/',
    });

    if (!isOk(validationResult)) {
      setValidationError(validationResult.error.issues[0]?.message || 'Invalid domain hostname');
      return;
    }

    try {
      await createDomainMutation.mutateAsync({
        data: {
          applicationId: app.id,
          hostname: validationResult.value.hostname,
          pathPrefix: '/',
        },
      });
      addToast({
        title: 'Domain Attached',
        description: `Successfully registered ${validationResult.value.hostname}`,
        variant: 'success',
      });
      setHostname('');
      refetchDomains();
    } catch (err: any) {
      addToast({
        title: 'Failed to Register Domain',
        description: err?.message || 'Could not register custom domain.',
        variant: 'error',
      });
    }
  };

  const handleRunCron = async (cronId: string) => {
    setRunningCronId(cronId);
    try {
      await runCronNow(cronId);
      addToast({
        title: 'Cron Invoked',
        description: 'Scheduled worker handler was triggered successfully.',
        variant: 'success',
      });
    } catch (err: any) {
      addToast({
        title: 'Cron Execution Failed',
        description: err?.message || 'Failed to trigger cron execution.',
        variant: 'error',
      });
    } finally {
      setRunningCronId(null);
    }
  };

  const handleCreateCronSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setCronError(null);
    if (!cronName.trim()) {
      setCronError('Please provide a descriptive name for this schedule');
      return;
    }
    if (!cronExpr.trim()) {
      setCronError('Please enter a valid 5-part cron expression');
      return;
    }

    try {
      await createCron({
        name: cronName.trim(),
        cron: cronExpr.trim(),
        targetAppId: app.id,
      });
      addToast({
        title: 'Cron Trigger Created',
        description: `Attached ${cronExpr.trim()} schedule to worker`,
        variant: 'success',
      });
      setShowAddCronModal(false);
      setCronName('');
      setCronExpr('*/5 * * * *');
    } catch (err: any) {
      setCronError(err?.message || 'Failed to attach cron trigger');
    }
  };

  return (
    <div data-testid="tab-content-triggers" className="p-6 max-w-6xl mx-auto w-full space-y-8">
      {/* Section 1: HTTP & Custom Domains */}
      <div className="space-y-4">
        <div>
          <h3 className="text-sm font-bold text-zinc-100 flex items-center gap-2">
            <Globe className="w-4 h-4 text-purple-400" />
            HTTP & Custom Domains
          </h3>
          <p className="text-xs text-zinc-400 mt-0.5">
            Hostnames routed directly to this worker through Traefik and edge proxies.
          </p>
        </div>

        {/* Default Traefik Subdomain Route */}
        <div className="p-5 rounded-xl border border-zinc-800 bg-zinc-900/20 space-y-3">
          <div className="flex items-start justify-between">
            <div>
              <h4 className="text-xs font-semibold text-zinc-200">Default Subdomain Ingress</h4>
              <p className="text-[11px] text-zinc-500 mt-0.5">
                Automatically assigned and synchronized with the Traefik fleet proxy.
              </p>
            </div>
            <span className="text-xs font-semibold text-emerald-400 bg-emerald-950/60 border border-emerald-800/80 px-2.5 py-0.5 rounded-full flex items-center gap-1">
              <ShieldCheck className="w-3.5 h-3.5" /> Active Ingress
            </span>
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 pt-1">
            <div className="p-3 rounded-lg bg-zinc-950 border border-zinc-800/80 font-mono text-xs flex items-center justify-between">
              <span className="text-emerald-400">{subdomain}.localhost</span>
              <span className="text-[10px] text-zinc-500">RFC 6761 Wildcard</span>
            </div>
            <div className="p-3 rounded-lg bg-zinc-950 border border-zinc-800/80 font-mono text-xs flex items-center justify-between">
              <span className="text-emerald-400">{subdomain}.cubit.local</span>
              <span className="text-[10px] text-zinc-500">Local Traefik Mesh</span>
            </div>
          </div>
        </div>

        {/* Custom Domains list */}
        {isLoadingDomains ? (
          <div className="p-4 text-xs text-zinc-500">Loading domains...</div>
        ) : appDomains.length > 0 && (
          <div className="space-y-2">
            {appDomains.map((d) => (
              <div
                key={d.id}
                className="p-4 rounded-xl border border-zinc-800 bg-zinc-900/30 flex items-center justify-between"
              >
                <div className="flex items-center gap-3">
                  <Globe className="w-4 h-4 text-emerald-400" />
                  <div>
                    <div className="font-mono text-xs font-semibold text-zinc-200">{d.hostname}</div>
                    <div className="text-[10px] text-zinc-500">Path prefix: {d.pathPrefix || '/'}</div>
                  </div>
                </div>

                <div className="flex items-center gap-2 text-xs text-emerald-400 bg-emerald-950/40 border border-emerald-900 px-3 py-1 rounded-full">
                  <ShieldCheck className="w-3.5 h-3.5" />
                  <span>Let's Encrypt Active</span>
                </div>
              </div>
            ))}
          </div>
        )}

        {/* Add Custom Domain Form */}
        <form onSubmit={handleAddDomain} className="p-4 rounded-xl border border-zinc-800 bg-zinc-900/20 space-y-3">
          <span className="text-xs font-semibold text-zinc-300 block">Add New Custom Domain</span>
          <div className="flex gap-2">
            <input
              type="text"
              required
              placeholder="api.yourdomain.com"
              value={hostname}
              onChange={(e) => setHostname(e.target.value)}
              className="flex-1 bg-zinc-950 border border-zinc-800 rounded-lg px-3 py-2 text-xs font-mono text-zinc-100 outline-none focus:border-emerald-500"
            />
            <button
              type="submit"
              disabled={createDomainMutation.isPending}
              className="px-4 py-2 rounded-lg bg-emerald-500 hover:bg-emerald-400 disabled:opacity-50 text-zinc-950 text-xs font-semibold transition flex items-center gap-1.5 shadow"
            >
              <Plus className="w-3.5 h-3.5" />
              {createDomainMutation.isPending ? 'Adding...' : 'Add Domain'}
            </button>
          </div>
          {validationError && (
            <p className="text-xs text-rose-400">{validationError}</p>
          )}
        </form>
      </div>

      {/* Section 2: Cron Triggers (scheduled()) */}
      <div className="space-y-4 pt-4 border-t border-zinc-800/80">
        <div className="flex items-center justify-between">
          <div>
            <h3 className="text-sm font-bold text-zinc-100 flex items-center gap-2">
              <Clock className="w-4 h-4 text-sky-400" />
              Cron Triggers (scheduled())
            </h3>
            <p className="text-xs text-zinc-400 mt-0.5">
              Execute your worker's exported <code className="font-mono text-emerald-400">scheduled(event, env, ctx)</code> handler on an automated schedule.
            </p>
          </div>
          <button
            type="button"
            data-testid="add-cron-trigger-btn"
            onClick={() => setShowAddCronModal(true)}
            className="flex items-center gap-1.5 px-3.5 py-1.5 rounded-lg bg-sky-600 hover:bg-sky-500 text-white text-xs font-semibold transition shadow-sm"
          >
            <Plus className="w-3.5 h-3.5" />
            Add Cron Trigger
          </button>
        </div>

        {attachedCron.length === 0 ? (
          <div className="p-6 rounded-xl border border-dashed border-zinc-800 bg-zinc-950/40 text-center space-y-2">
            <p className="text-xs text-zinc-400">No cron triggers scheduled for this worker.</p>
            <p className="text-[11px] text-zinc-600 max-w-sm mx-auto">
              Add a cron schedule to periodically invoke maintenance tasks, cache warming, or scheduled jobs.
            </p>
          </div>
        ) : (
          <div className="space-y-2.5">
            {attachedCron.map((cron) => (
              <div
                key={cron.id}
                className="p-4 rounded-xl border border-zinc-800 bg-zinc-900/40 flex items-center justify-between gap-4 flex-wrap"
              >
                <div className="flex items-center gap-3">
                  <div className="p-2 rounded-lg bg-sky-950/80 border border-sky-800/80 text-sky-400">
                    <Clock className="w-4 h-4" />
                  </div>
                  <div>
                    <div className="flex items-center gap-2">
                      <span className="text-xs font-bold text-zinc-200">{cron.name}</span>
                      <span className="font-mono text-[11px] px-2 py-0.5 rounded bg-zinc-800 text-emerald-400 border border-zinc-700">
                        {cron.cronExpression || cron.cron}
                      </span>
                      <span className="text-[10px] px-2 py-0.5 rounded-full font-bold uppercase bg-emerald-950 text-emerald-400 border border-emerald-800">
                        {cron.status || 'Active'}
                      </span>
                    </div>
                    <div className="text-[10px] text-zinc-500 font-mono mt-1">
                      Last run: {cron.lastRunAt || cron.last_run_at ? new Date(cron.lastRunAt || cron.last_run_at || '').toLocaleString() : 'Never'}
                    </div>
                  </div>
                </div>

                <div className="flex items-center gap-2">
                  <button
                    type="button"
                    data-testid={`run-cron-${cron.id}`}
                    onClick={() => handleRunCron(cron.id)}
                    disabled={runningCronId === cron.id}
                    className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg border border-sky-800 bg-sky-950/60 hover:bg-sky-900/80 text-sky-300 text-xs font-semibold transition"
                    title="Trigger immediate execution"
                  >
                    {runningCronId === cron.id ? (
                      <RefreshCw className="w-3.5 h-3.5 animate-spin" />
                    ) : (
                      <Play className="w-3.5 h-3.5" />
                    )}
                    Run Now
                  </button>
                  <button
                    type="button"
                    onClick={() => onNavigateToService?.('cron', cron.id)}
                    className="flex items-center gap-1 px-3 py-1.5 rounded-lg bg-zinc-800 hover:bg-zinc-700 text-zinc-300 text-xs font-semibold transition"
                  >
                    <span>Open in Cron</span>
                    <ArrowRight className="w-3.5 h-3.5 text-zinc-400" />
                  </button>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Section 3: Queue Consumers (queue()) */}
      <div className="space-y-4 pt-4 border-t border-zinc-800/80">
        <div className="flex items-center justify-between">
          <div>
            <h3 className="text-sm font-bold text-zinc-100 flex items-center gap-2">
              <Inbox className="w-4 h-4 text-emerald-400" />
              Queue Consumers (queue())
            </h3>
            <p className="text-xs text-zinc-400 mt-0.5">
              Process batches of messages via your worker's exported <code className="font-mono text-emerald-400">queue(batch, env, ctx)</code> handler.
            </p>
          </div>
          <button
            type="button"
            onClick={() => onNavigateToService?.('queues')}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-zinc-800 hover:bg-zinc-700 text-zinc-300 text-xs font-semibold transition"
          >
            <span>Manage Queues</span>
            <ArrowRight className="w-3.5 h-3.5 text-emerald-400" />
          </button>
        </div>

        {attachedQueues.length === 0 ? (
          <div className="p-6 rounded-xl border border-dashed border-zinc-800 bg-zinc-950/40 text-center space-y-2">
            <p className="text-xs text-zinc-400">No queues configured to send messages to this worker.</p>
            <p className="text-[11px] text-zinc-600 max-w-sm mx-auto">
              To process messages, bind this worker as a consumer in the Queues control panel.
            </p>
          </div>
        ) : (
          <div className="space-y-2.5">
            {attachedQueues.map((q) => (
              <div
                key={q.id}
                className="p-4 rounded-xl border border-zinc-800 bg-zinc-900/40 flex items-center justify-between"
              >
                <div className="flex items-center gap-3">
                  <div className="p-2 rounded-lg bg-emerald-950/80 border border-emerald-800/80 text-emerald-400">
                    <Inbox className="w-4 h-4" />
                  </div>
                  <div>
                    <div className="flex items-center gap-2">
                      <span className="text-xs font-bold text-zinc-200">{q.name}</span>
                      <span className="text-[10px] px-2 py-0.5 rounded-full font-bold uppercase bg-emerald-950 text-emerald-400 border border-emerald-800">
                        Consumer Active
                      </span>
                    </div>
                    <div className="text-[11px] text-zinc-500 font-mono mt-0.5">
                      Batch size: {q.maxBatchSize || q.max_batch_size || 10} • Retries: {q.maxRetries || q.max_retries || 3}
                    </div>
                  </div>
                </div>

                <button
                  type="button"
                  onClick={() => onNavigateToService?.('queues', q.id)}
                  className="flex items-center gap-1 px-3 py-1.5 rounded-lg bg-zinc-800 hover:bg-zinc-700 text-zinc-300 text-xs font-semibold transition"
                >
                  <span>Open Queue</span>
                  <ArrowRight className="w-3.5 h-3.5 text-emerald-400" />
                </button>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Add Cron Trigger Modal */}
      {showAddCronModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/75 backdrop-blur-sm p-4">
          <div className="w-full max-w-md bg-zinc-900 border border-zinc-800 rounded-2xl p-6 shadow-2xl space-y-4">
            <div className="flex items-center justify-between">
              <h3 className="text-base font-bold text-zinc-100 flex items-center gap-2">
                <Clock className="w-4 h-4 text-sky-400" />
                Add Cron Trigger
              </h3>
              <button
                type="button"
                onClick={() => setShowAddCronModal(false)}
                className="p-1 rounded-lg text-zinc-400 hover:text-zinc-200 hover:bg-zinc-800 transition"
              >
                <X className="w-4 h-4" />
              </button>
            </div>

            <p className="text-xs text-zinc-400">
              Schedule automated execution of <code className="font-mono text-emerald-400">scheduled()</code> on worker <code className="font-mono text-zinc-200">{app.name}</code>.
            </p>

            <form onSubmit={handleCreateCronSubmit} className="space-y-4">
              <div>
                <label className="text-xs font-semibold text-zinc-400 block mb-1">
                  Trigger Name
                </label>
                <input
                  type="text"
                  required
                  placeholder="e.g. nightly-cache-purge"
                  value={cronName}
                  onChange={(e) => setCronName(e.target.value)}
                  className="w-full bg-zinc-950 border border-zinc-800 rounded-lg p-2.5 text-xs text-zinc-200 focus:outline-none focus:border-sky-500 font-mono"
                />
              </div>

              <div>
                <label className="text-xs font-semibold text-zinc-400 block mb-1">
                  Cron Expression (5-part format)
                </label>
                <input
                  type="text"
                  required
                  placeholder="*/5 * * * *"
                  value={cronExpr}
                  onChange={(e) => setCronExpr(e.target.value)}
                  className="w-full bg-zinc-950 border border-zinc-800 rounded-lg p-2.5 text-xs text-emerald-400 focus:outline-none focus:border-sky-500 font-mono"
                />

                <div className="flex gap-2 mt-2">
                  <button
                    type="button"
                    onClick={() => setCronExpr('*/5 * * * *')}
                    className="text-[10px] px-2 py-0.5 rounded bg-zinc-800 hover:bg-zinc-700 text-zinc-300 font-mono transition"
                  >
                    Every 5m
                  </button>
                  <button
                    type="button"
                    onClick={() => setCronExpr('0 * * * *')}
                    className="text-[10px] px-2 py-0.5 rounded bg-zinc-800 hover:bg-zinc-700 text-zinc-300 font-mono transition"
                  >
                    Hourly
                  </button>
                  <button
                    type="button"
                    onClick={() => setCronExpr('0 0 * * *')}
                    className="text-[10px] px-2 py-0.5 rounded bg-zinc-800 hover:bg-zinc-700 text-zinc-300 font-mono transition"
                  >
                    Daily at midnight
                  </button>
                </div>
              </div>

              {cronError && (
                <p className="text-xs text-rose-400">{cronError}</p>
              )}

              <div className="flex justify-end gap-2 pt-2">
                <button
                  type="button"
                  onClick={() => setShowAddCronModal(false)}
                  className="px-3.5 py-1.5 rounded-lg border border-zinc-700 bg-zinc-800 hover:bg-zinc-700 text-xs font-semibold text-zinc-300 transition"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  disabled={isCreatingCron}
                  className="px-4 py-1.5 rounded-lg bg-sky-600 hover:bg-sky-500 disabled:opacity-50 text-xs font-semibold text-white transition shadow"
                >
                  {isCreatingCron ? 'Attaching...' : 'Attach Trigger'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
