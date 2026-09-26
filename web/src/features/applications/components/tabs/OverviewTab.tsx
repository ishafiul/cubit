import {
  GitBranch,
  Calendar,
  Radio,
  Globe,
  Code,
  Zap,
  Link2,
  Settings,
  Send,
  Database,
  Clock,
  Inbox,
  ExternalLink,
} from 'lucide-react';
import type { Application } from '../../../../api/model';
import { useListDomains } from '../../../../api/generated/domains/domains';
import { useListDeployments } from '../../../../api/generated/deployments/deployments';
import { useApplicationActions } from '../../stores/applicationStore';
import { useFleetServices } from '../../hooks/useFleetServices';
import { MetricsPanel } from './MetricsPanel';
import { RecentRequestsTable } from './RecentRequestsTable';

export interface OverviewTabProps {
  app: Application;
  onTestApp?: (app: Application) => void;
  onNavigateToService?: (targetTab: string, resourceId?: string) => void;
}

export function OverviewTab({ app, onTestApp }: OverviewTabProps) {
  const { setActiveTab, setBuildsViewMode } = useApplicationActions();
  const { data: domains } = useListDomains();
  const { data: deployments } = useListDeployments(app.id);
  const { attachedCron, attachedQueues } = useFleetServices(app);

  const appDomains = Array.isArray(domains)
    ? domains.filter((d) => d.applicationId === app.id)
    : [];

  const deploymentsList = Array.isArray(deployments) ? deployments : [];
  const activeDep = deploymentsList.find((d) => d.id === app.activeDeploymentId);
  const activeBuildVersion = activeDep?.buildVersion || deploymentsList[0]?.buildVersion || 1;

  const totalBindingsCount = (app.bindings || []).length;
  const totalTriggersCount = appDomains.length + 1 + attachedCron.length + attachedQueues.length;

  const host = typeof window !== 'undefined' ? window.location.hostname || 'localhost' : 'localhost';
  const port = typeof window !== 'undefined' && window.location.port ? `:${window.location.port}` : '';
  const testUrl = app.subdomain
    ? `http://${app.subdomain}.${host}${port}`
    : `http://${host}${port}/api/v1/applications/${app.id}/test`;

  return (
    <div data-testid="tab-content-overview" className="p-6 space-y-6 max-w-6xl mx-auto w-full">
      {/* Top 3 High-Level Metadata Cards */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-4">
          <div className="flex items-center gap-2 text-zinc-400 text-xs font-medium">
            <Radio className="w-4 h-4 text-emerald-400" />
            Runtime Status
          </div>
          <p className="mt-2 text-xl font-bold text-zinc-100 capitalize">
            {app.status || 'Active'}
          </p>
          <p className="text-xs text-zinc-500 mt-1">Celld V8 isolate active</p>
        </div>

        <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-4">
          <div className="flex items-center gap-2 text-zinc-400 text-xs font-medium">
            <GitBranch className="w-4 h-4 text-blue-400" />
            Source Type
          </div>
          <p className="mt-2 text-xl font-bold text-zinc-100 capitalize">
            {app.sourceType || 'Inline'}
          </p>
          <p className="text-xs text-zinc-500 mt-1">
            {app.sourceType === 'git' ? app.gitRepo || 'Git repository' : 'Editor code bundle'}
          </p>
        </div>

        <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-4">
          <div className="flex items-center gap-2 text-zinc-400 text-xs font-medium">
            <Calendar className="w-4 h-4 text-purple-400" />
            Created At
          </div>
          <p className="mt-2 text-sm font-semibold text-zinc-200">
            {app.createdAt ? new Date(app.createdAt).toLocaleDateString() : 'Recently'}
          </p>
          <p className="text-xs text-zinc-500 mt-1 font-mono">
            Updated: {app.updatedAt ? new Date(app.updatedAt).toLocaleTimeString() : 'Recently'}
          </p>
        </div>
      </div>

      {/* Telemetry Metrics Panel */}
      <MetricsPanel appId={app.id} />

      {/* Architecture & Quotas + Connected Services Section */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
        {/* Architecture & Configuration Card */}
        <div className="lg:col-span-2 bg-zinc-900/30 border border-zinc-800/80 rounded-xl p-5 space-y-4">
          <div>
            <h3 className="text-sm font-bold text-zinc-200">Worker Architecture & Configuration</h3>
            <p className="text-xs text-zinc-400 mt-0.5">
              Underlying isolate specifications and deployment target
            </p>
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 text-xs">
            <div className="p-3 bg-zinc-950/60 rounded-lg border border-zinc-800/60 space-y-1">
              <span className="text-[11px] text-zinc-500 uppercase tracking-wider block">Ingress Route</span>
              <a
                href={testUrl}
                target="_blank"
                rel="noopener noreferrer"
                className="font-mono text-emerald-400 hover:underline flex items-center gap-1 font-semibold truncate"
                title={testUrl}
              >
                <Globe className="w-3.5 h-3.5 shrink-0" />
                <span className="truncate">{app.subdomain ? `${app.subdomain}.localhost` : 'Cluster Ingress'}</span>
                <ExternalLink className="w-3 h-3 shrink-0 opacity-60" />
              </a>
              <span className="text-[10px] text-zinc-500 block truncate">Direct Traefik Mesh endpoint</span>
            </div>

            <div className="p-3 bg-zinc-950/60 rounded-lg border border-zinc-800/60 space-y-1">
              <span className="text-[11px] text-zinc-500 uppercase tracking-wider block">Source Deployment</span>
              <span className="font-mono text-zinc-200 font-semibold block truncate">
                {app.sourceType === 'inline' ? 'Inline Worker (Web IDE)' : `Git Repo (${app.branch || 'main'})`}
              </span>
              <span className="text-[10px] text-zinc-500">Active version: v{activeBuildVersion}</span>
            </div>

            <div className="p-3 bg-zinc-950/60 rounded-lg border border-zinc-800/60 space-y-1">
              <span className="text-[11px] text-zinc-500 uppercase tracking-wider block">Compatibility Date</span>
              <span className="font-mono text-zinc-200 font-semibold block">{app.compatibilityDate || '2024-04-03'}</span>
              <span className="text-[10px] text-zinc-500">
                {app.compatibilityFlags?.includes('nodejs_compat') ? 'nodejs_compat enabled' : 'standard V8 runtime'}
              </span>
            </div>

            <div className="p-3 bg-zinc-950/60 rounded-lg border border-zinc-800/60 space-y-1">
              <span className="text-[11px] text-zinc-500 uppercase tracking-wider block">Resource Quotas</span>
              <span className="font-mono text-zinc-200 font-semibold block">
                {app.memoryLimitMb || 128} MB RAM / {app.maxDurationMs || 50} ms CPU
              </span>
              <span className="text-[10px] text-zinc-500">Isolate execution limits</span>
            </div>
          </div>

          {/* Quick actions strip */}
          <div className="pt-2 flex flex-wrap gap-2.5">
            <button
              type="button"
              data-testid="quick-action-code"
              onClick={() => setActiveTab('code')}
              className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-zinc-800 hover:bg-zinc-700 text-zinc-200 text-xs font-semibold transition"
            >
              <Code className="w-3.5 h-3.5 text-emerald-400" />
              Edit Code
            </button>
            <button
              type="button"
              data-testid="quick-action-tail"
              onClick={() => {
                setActiveTab('builds');
                setBuildsViewMode('tail');
              }}
              className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-zinc-800 hover:bg-zinc-700 text-zinc-200 text-xs font-semibold transition"
            >
              <Radio className="w-3.5 h-3.5 text-sky-400" />
              Live Tail
            </button>
            <button
              type="button"
              data-testid="quick-action-triggers"
              onClick={() => setActiveTab('triggers')}
              className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-zinc-800 hover:bg-zinc-700 text-zinc-200 text-xs font-semibold transition"
            >
              <Zap className="w-3.5 h-3.5 text-amber-400" />
              Triggers ({totalTriggersCount})
            </button>
            <button
              type="button"
              data-testid="quick-action-bindings"
              onClick={() => setActiveTab('bindings')}
              className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-zinc-800 hover:bg-zinc-700 text-zinc-200 text-xs font-semibold transition"
            >
              <Link2 className="w-3.5 h-3.5 text-indigo-400" />
              Bindings ({totalBindingsCount})
            </button>
            <button
              type="button"
              data-testid="quick-action-settings"
              onClick={() => setActiveTab('settings')}
              className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-zinc-800 hover:bg-zinc-700 text-zinc-200 text-xs font-semibold transition"
            >
              <Settings className="w-3.5 h-3.5 text-zinc-400" />
              Settings
            </button>
          </div>
        </div>

        {/* Connected Services Card */}
        <div className="bg-zinc-900/30 border border-zinc-800/80 rounded-xl p-5 space-y-3 flex flex-col justify-between">
          <div>
            <div className="flex items-center justify-between mb-2">
              <h3 className="text-sm font-bold text-zinc-200">Connected Fleet</h3>
              <span className="text-[10px] font-bold px-2 py-0.5 rounded-full bg-indigo-950 text-indigo-300 border border-indigo-800">
                {totalBindingsCount} Bound
              </span>
            </div>
            <p className="text-xs text-zinc-400 leading-relaxed">
              Resources injected via <code className="font-mono text-emerald-400">env.*</code> or triggering this worker.
            </p>

            <div className="mt-4 space-y-2 text-xs">
              <div className="flex items-center justify-between p-2 rounded-lg bg-zinc-950/60 border border-zinc-800/60">
                <span className="text-zinc-400 flex items-center gap-1.5">
                  <Database className="w-3.5 h-3.5 text-amber-400" /> Injected Resources
                </span>
                <span className="font-mono font-bold text-zinc-200">{totalBindingsCount}</span>
              </div>
              <div className="flex items-center justify-between p-2 rounded-lg bg-zinc-950/60 border border-zinc-800/60">
                <span className="text-zinc-400 flex items-center gap-1.5">
                  <Clock className="w-3.5 h-3.5 text-sky-400" /> Cron Triggers
                </span>
                <span className="font-mono font-bold text-zinc-200">{attachedCron.length}</span>
              </div>
              <div className="flex items-center justify-between p-2 rounded-lg bg-zinc-950/60 border border-zinc-800/60">
                <span className="text-zinc-400 flex items-center gap-1.5">
                  <Inbox className="w-3.5 h-3.5 text-emerald-400" /> Queue Consumers
                </span>
                <span className="font-mono font-bold text-zinc-200">{attachedQueues.length}</span>
              </div>
              <div className="flex items-center justify-between p-2 rounded-lg bg-zinc-950/60 border border-zinc-800/60">
                <span className="text-zinc-400 flex items-center gap-1.5">
                  <Globe className="w-3.5 h-3.5 text-purple-400" /> Custom Domains
                </span>
                <span className="font-mono font-bold text-zinc-200">{appDomains.length}</span>
              </div>
            </div>
          </div>

          <button
            type="button"
            data-testid="overview-test-worker-btn"
            onClick={() => onTestApp?.(app)}
            className="w-full mt-3 py-2 px-3 rounded-lg border border-emerald-900/60 bg-emerald-950/40 hover:bg-emerald-950/70 text-emerald-400 text-xs font-semibold transition flex items-center justify-center gap-1.5 cursor-pointer"
          >
            <Send className="w-3.5 h-3.5" />
            Test Worker Invocation
          </button>
        </div>
      </div>

      {/* Recent Requests Table */}
      <RecentRequestsTable app={app} onTestApp={onTestApp} />
    </div>
  );
}
