import { useState, useEffect } from 'react';
import { X, RefreshCw } from 'lucide-react';
import type { Application } from '../../api/model';
import { useListDeployments } from '../../api/generated/deployments/deployments';
import { getDeploymentStatusBadge } from '../ApplicationDetailPage';

export interface BuildHistoryModalProps {
  app: Application;
  onClose: () => void;
}

export function BuildHistoryModal({ app, onClose }: BuildHistoryModalProps) {
  const { data: deploymentsData } = useListDeployments(app.id);
  const deployments = Array.isArray(deploymentsData) ? deploymentsData : [];
  const [selectedDepId, setSelectedDepId] = useState<string | null>(null);
  const [depLogs, setDepLogs] = useState<Array<{ timestamp: string; step: string; message: string; level: string }>>([]);
  const [isLoadingLogs, setIsLoadingLogs] = useState(false);

  useEffect(() => {
    if (!selectedDepId && deployments.length > 0) {
      setSelectedDepId(deployments[0].id);
    }
  }, [deployments, selectedDepId]);

  useEffect(() => {
    if (!selectedDepId) return;
    setIsLoadingLogs(true);
    fetch(`/api/v1/deployments/${selectedDepId}/logs`)
      .then(r => r.json())
      .then((data: any[]) => {
        if (Array.isArray(data)) {
          setDepLogs(data.map(entry => ({
            timestamp: entry.timestamp || entry.Timestamp || new Date().toISOString(),
            step: entry.step || entry.Step || 'info',
            message: entry.message || entry.Message || '',
            level: entry.level || entry.Level || 'info',
          })));
        } else {
          setDepLogs([]);
        }
      })
      .catch(() => setDepLogs([]))
      .finally(() => setIsLoadingLogs(false));
  }, [selectedDepId]);

  return (
    <div className="fixed inset-0 bg-black/60 backdrop-blur-sm flex items-center justify-center p-4 z-50">
      <div className="bg-zinc-900 border border-zinc-800 rounded-2xl max-w-3xl w-full p-6 space-y-5 shadow-2xl flex flex-col max-h-[90vh]">
        {/* Header */}
        <div className="flex items-center justify-between border-b border-zinc-800/80 pb-4">
          <div>
            <div className="flex items-center gap-2.5">
              <h3 className="text-lg font-bold">Build History & Logs</h3>
              <span className="text-xs px-2.5 py-0.5 rounded-full font-mono bg-emerald-950/80 text-emerald-400 border border-emerald-800/80">
                {app.name}
              </span>
            </div>
            <p className="text-xs text-zinc-400 mt-0.5">
              Build version tags and deployment build logs for this project
            </p>
          </div>
          <button
            type="button"
            onClick={onClose}
            className="text-zinc-500 hover:text-zinc-300 transition"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* Content */}
        <div className="grid grid-cols-12 gap-4 flex-1 min-h-0">
          {/* Versions list */}
          <div className="col-span-4 bg-zinc-950/60 border border-zinc-800 rounded-xl p-3 space-y-2 overflow-y-auto max-h-[420px]">
            <div className="text-xs font-semibold uppercase tracking-wider text-zinc-400 mb-2">
              Deployments ({deployments.length})
            </div>
            {deployments.length === 0 ? (
              <div className="text-xs text-zinc-500 italic">No deployments found.</div>
            ) : (
              deployments.map(dep => {
                const isSelected = selectedDepId === dep.id;
                const d = new Date(dep.createdAt);
                const dateStr = isNaN(d.getTime()) ? '' : d.toLocaleDateString() + ' ' + d.toLocaleTimeString();
                return (
                  <div
                    key={dep.id}
                    onClick={() => setSelectedDepId(dep.id)}
                    className={`p-3 rounded-lg border cursor-pointer transition space-y-1 ${
                      isSelected
                        ? 'bg-zinc-800 border-emerald-500 text-zinc-100'
                        : 'bg-zinc-900/60 border-zinc-800 hover:border-zinc-700 text-zinc-300'
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
                    {dateStr && (
                      <div className="text-[10px] text-zinc-500">
                        {dateStr}
                      </div>
                    )}
                  </div>
                );
              })
            )}
          </div>

          {/* Logs panel */}
          <div className="col-span-8 flex flex-col bg-black/80 rounded-xl border border-zinc-800 p-4 min-h-0 shadow-inner">
            <div className="flex items-center justify-between text-xs text-zinc-400 border-b border-zinc-800 pb-2 mb-2">
              <span className="font-mono">
                {selectedDepId ? `Deployment Logs: ${selectedDepId.slice(0, 8)}` : 'Select a deployment'}
              </span>
              {isLoadingLogs && <RefreshCw className="w-3.5 h-3.5 animate-spin text-emerald-400" />}
            </div>

            <div className="flex-1 overflow-y-auto font-mono text-xs text-zinc-300 space-y-1 max-h-[380px]">
              {depLogs.length === 0 ? (
                <div className="text-zinc-600 italic">No logs recorded for this deployment version.</div>
              ) : (
                depLogs.map((l, idx) => (
                  <div key={idx} className="flex items-start gap-2">
                    <span className="text-emerald-500 font-bold uppercase text-[10px]">[{l.step}]</span>
                    <span className={l.level === 'error' ? 'text-rose-400 font-semibold' : 'text-zinc-300'}>{l.message}</span>
                  </div>
                ))
              )}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
