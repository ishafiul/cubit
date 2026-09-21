import { useState, useEffect } from 'react';
import { ShieldCheck, X, RefreshCw } from 'lucide-react';
import { useDashboard } from '../../context/DashboardContext';
import { useListDeployments } from '../../api/generated/deployments/deployments';
import type { Application } from '../../api/model';
import { getDeploymentStatusBadge } from '../ApplicationDetailPage';

export function BuildLogsView({
  initialAppId,
  onAppChange,
}: {
  initialAppId?: string;
  onAppChange?: (appId: string) => void;
} = {}) {
  const { apps, activeDeploymentId, deployStatus, logs } = useDashboard();
  const [selectedLogsAppId, setSelectedLogsAppId] = useState<string>(initialAppId || '');

  useEffect(() => {
    if (initialAppId !== undefined) {
      setSelectedLogsAppId(initialAppId);
    }
  }, [initialAppId]);

  return (
    <div className="h-full flex flex-col space-y-4 min-h-0">
      <div className="flex items-center justify-between flex-wrap gap-3">
        <div className="flex items-center gap-3">
          <span className="text-sm font-semibold text-zinc-400 uppercase tracking-wider">
            {selectedLogsAppId ? 'Project Build Logs' : 'Live Build & Deployment Stream'}
          </span>
          {!selectedLogsAppId && deployStatus && (
            <span
              className={`text-[11px] font-mono px-2.5 py-0.5 rounded-full border uppercase font-bold flex items-center gap-1.5 ${
                deployStatus === 'active'
                  ? 'bg-emerald-950/60 border-emerald-700/80 text-emerald-400'
                  : deployStatus === 'failed'
                  ? 'bg-rose-950/60 border-rose-700/80 text-rose-400'
                  : 'bg-amber-950/60 border-amber-700/80 text-amber-400'
              }`}
            >
              {deployStatus === 'active' && <span className="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse" />}
              {deployStatus === 'building' && <RefreshCw className="w-3 h-3 animate-spin text-amber-400" />}
              Status: {deployStatus}
            </span>
          )}
        </div>

        <div className="flex items-center gap-3">
          <div className="flex items-center gap-2">
            <label className="text-xs text-zinc-400 font-medium">Filter Project:</label>
            <select
              value={selectedLogsAppId}
              onChange={e => {
                const val = e.target.value;
                setSelectedLogsAppId(val);
                onAppChange?.(val);
              }}
              className="bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-1.5 text-xs text-zinc-200 focus:outline-none focus:border-emerald-500 font-mono"
            >
              <option value="">-- Active Live Stream --</option>
              {apps.map(a => (
                <option key={a.id} value={a.id}>
                  {a.name} ({a.subdomain || a.name})
                </option>
              ))}
            </select>
          </div>
          {!selectedLogsAppId && activeDeploymentId && (
            <span className="text-xs font-mono text-zinc-500">ID: {activeDeploymentId.slice(0, 8)}</span>
          )}
        </div>
      </div>

      {selectedLogsAppId ? (
        <ProjectBuildLogsView appId={selectedLogsAppId} apps={apps} />
      ) : (
        <div className="flex-1 flex flex-col space-y-3 min-h-0">
          {deployStatus === 'active' && (
            <div className="p-3 bg-emerald-950/40 border border-emerald-800/80 rounded-xl flex items-center justify-between text-xs text-emerald-300">
              <div className="flex items-center gap-2">
                <ShieldCheck className="w-4 h-4 text-emerald-400" />
                <span className="font-semibold">Deployment active and traffic routing live!</span>
              </div>
              <span className="text-[11px] text-emerald-500/80">Traefik route synchronized</span>
            </div>
          )}

          {deployStatus === 'failed' && (
            <div className="p-3 bg-rose-950/40 border border-rose-800/80 rounded-xl flex items-center justify-between text-xs text-rose-300">
              <div className="flex items-center gap-2">
                <X className="w-4 h-4 text-rose-400" />
                <span className="font-semibold">Deployment failed during build or route sync.</span>
              </div>
              <span className="text-[11px] text-rose-500/80">Check logs below for details</span>
            </div>
          )}

          <div className="flex-1 bg-black/80 rounded-xl p-5 font-mono text-xs text-zinc-300 overflow-y-auto border border-zinc-900 space-y-1.5 shadow-inner min-h-[300px]">
            {logs.length === 0 ? (
              <div className="text-zinc-600 italic">Waiting for deployment stream...</div>
            ) : (
              logs.map((l, idx) => {
                const timeStr = (() => {
                  try {
                    const d = new Date(l.timestamp);
                    return isNaN(d.getTime()) ? new Date().toLocaleTimeString() : d.toLocaleTimeString();
                  } catch (_) {
                    return new Date().toLocaleTimeString();
                  }
                })();
                return (
                  <div key={idx} className="flex items-start gap-3">
                    <span className="text-zinc-600">{timeStr}</span>
                    <span className="text-emerald-500 font-bold uppercase text-[10px]">[{l.step}]</span>
                    <span className={l.level === 'error' ? 'text-red-400 font-semibold' : 'text-zinc-200'}>
                      {l.message}
                    </span>
                  </div>
                );
              })
            )}
          </div>
        </div>
      )}
    </div>
  );
}

function ProjectBuildLogsView({ appId, apps }: { appId: string; apps: Application[] }) {
  const app = apps.find(a => a.id === appId);
  const { data: deploymentsData } = useListDeployments(appId);
  const deployments = Array.isArray(deploymentsData) ? deploymentsData : [];
  const [selectedDepId, setSelectedDepId] = useState<string | null>(null);
  const [logs, setLogs] = useState<Array<{ timestamp: string; step: string; message: string; level: string }>>([]);
  const [isLoading, setIsLoading] = useState(false);

  useEffect(() => {
    if (deployments.length > 0 && !selectedDepId) {
      setSelectedDepId(deployments[0].id);
    }
  }, [deployments, selectedDepId]);

  useEffect(() => {
    if (!selectedDepId) return;
    setIsLoading(true);
    fetch(`/api/v1/deployments/${selectedDepId}/logs`)
      .then(r => r.json())
      .then((data: any[]) => {
        if (Array.isArray(data)) {
          setLogs(
            data.map(entry => ({
              timestamp: entry.timestamp || entry.Timestamp || new Date().toISOString(),
              step: entry.step || entry.Step || 'info',
              message: entry.message || entry.Message || '',
              level: entry.level || entry.Level || 'info',
            }))
          );
        } else {
          setLogs([]);
        }
      })
      .catch(() => setLogs([]))
      .finally(() => setIsLoading(false));
  }, [selectedDepId]);

  return (
    <div className="flex-1 flex flex-col space-y-3 min-h-0">
      <div className="grid grid-cols-12 gap-4 flex-1 min-h-0">
        {/* Builds list */}
        <div className="col-span-4 bg-zinc-900/30 border border-zinc-800/80 rounded-xl p-3 space-y-2 overflow-y-auto">
          <div className="text-xs font-semibold uppercase tracking-wider text-zinc-400 mb-2">
            Builds for {app?.name || 'Project'} ({deployments.length})
          </div>
          {deployments.length === 0 ? (
            <div className="text-xs text-zinc-500 italic">No deployments found for this project yet.</div>
          ) : (
            deployments.map(dep => {
              const isSelected = selectedDepId === dep.id;
              const d = new Date(dep.createdAt);
              const dateStr = isNaN(d.getTime()) ? '' : d.toLocaleString();
              return (
                <div
                  key={dep.id}
                  onClick={() => setSelectedDepId(dep.id)}
                  className={`p-3 rounded-lg border cursor-pointer transition space-y-1 ${
                    isSelected
                      ? 'bg-zinc-800 border-emerald-500 text-zinc-100'
                      : 'bg-zinc-950/60 border-zinc-800 hover:border-zinc-700 text-zinc-300'
                  }`}
                >
                  <div className="flex items-center justify-between">
                    <span className="text-xs font-bold font-mono">v{dep.buildVersion || 1}</span>
                    <span className={`text-[10px] px-2 py-0.5 rounded-full font-bold uppercase ${getDeploymentStatusBadge(dep.status)}`}>
                      {dep.status}
                    </span>
                  </div>
                  <div className="text-[10px] font-mono text-zinc-500 truncate">
                    Commit: {dep.commitHash.slice(0, 7)}
                  </div>
                  {dateStr && <div className="text-[10px] text-zinc-500">{dateStr}</div>}
                </div>
              );
            })
          )}
        </div>

        {/* Logs terminal */}
        <div className="col-span-8 flex flex-col bg-black/80 rounded-xl border border-zinc-800/80 p-4 min-h-0 shadow-inner">
          <div className="flex items-center justify-between text-xs text-zinc-400 border-b border-zinc-800 pb-2 mb-2">
            <span className="font-mono">
              {selectedDepId ? `Deployment Logs: ${selectedDepId.slice(0, 8)}` : 'Select a deployment version'}
            </span>
            {isLoading && <RefreshCw className="w-3.5 h-3.5 animate-spin text-emerald-400" />}
          </div>

          <div className="flex-1 overflow-y-auto font-mono text-xs text-zinc-300 space-y-1">
            {logs.length === 0 ? (
              <div className="text-zinc-600 italic">No build output for this deployment.</div>
            ) : (
              logs.map((l, idx) => (
                <div key={idx} className="flex items-start gap-2">
                  <span className="text-emerald-500 font-bold uppercase text-[10px]">[{l.step}]</span>
                  <span className={l.level === 'error' ? 'text-rose-400 font-semibold' : 'text-zinc-300'}>
                    {l.message}
                  </span>
                </div>
              ))
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
