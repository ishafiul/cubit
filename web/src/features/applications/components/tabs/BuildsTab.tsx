import { useState } from 'react';
import { Terminal, RefreshCw, RotateCcw, Clock, AlertCircle } from 'lucide-react';
import type { Application } from '../../../../api/model';
import {
  useListDeployments,
  useRollbackApplication,
} from '../../../../api/generated/deployments/deployments';
import {
  useDeploymentLogs,
  useDeployStatus,
  useActiveDeploymentId,
} from '../../../../shared/stores/useDeploymentTrackerStore';
import { useToastActions } from '../../../../shared/stores/useToastStore';
import { getDeploymentStatusBadge } from '../WorkbenchHeader';

export function BuildsTab({ app }: { app: Application }) {
  const { data: deployments, refetch: refetchDeployments, isLoading } = useListDeployments(app.id);

  const rollbackMutation = useRollbackApplication();
  const { addToast } = useToastActions();

  const activeDeploymentId = useActiveDeploymentId();
  const deployStatus = useDeployStatus();
  const logs = useDeploymentLogs();

  const [rollingBackId, setRollingBackId] = useState<string | null>(null);

  const handleRollback = async (deploymentId: string) => {
    setRollingBackId(deploymentId);
    try {
      await rollbackMutation.mutateAsync({
        id: app.id,
        data: { deploymentId },
      });
      addToast({
        title: 'Rollback Triggered',
        description: `Successfully rolled back to deployment ${deploymentId.slice(0, 8)}`,
        variant: 'success',
      });
      refetchDeployments();
    } catch (err: any) {
      addToast({
        title: 'Rollback Failed',
        description: err?.message || 'Could not complete rollback.',
        variant: 'error',
      });
    } finally {
      setRollingBackId(null);
    }
  };

  return (
    <div data-testid="tab-content-builds" className="p-6 max-w-6xl mx-auto w-full space-y-6">
      {/* Live Log Stream if currently building or active logs exist */}
      {(activeDeploymentId || logs.length > 0) && (
        <div className="bg-zinc-950 border border-zinc-800 rounded-xl overflow-hidden shadow-2xl">
          <div className="p-3 bg-zinc-900 border-b border-zinc-800 flex items-center justify-between">
            <div className="flex items-center gap-2">
              <Terminal className="w-4 h-4 text-emerald-400" />
              <span className="text-xs font-bold text-zinc-200">
                Live Deployment Stream: <span className="font-mono text-zinc-400">{activeDeploymentId}</span>
              </span>
              <span className={`text-[10px] font-bold px-2 py-0.5 rounded-full ${getDeploymentStatusBadge(deployStatus)}`}>
                {deployStatus || 'Active'}
              </span>
            </div>
          </div>
          <div className="p-4 font-mono text-xs text-zinc-300 max-h-64 overflow-y-auto space-y-1">
            {logs.map((log, idx) => (
              <div key={idx} className="flex items-start gap-2">
                <span className="text-zinc-600 shrink-0">
                  {new Date(log.timestamp).toLocaleTimeString()}
                </span>
                <span className="text-emerald-500 font-semibold shrink-0">[{log.step}]</span>
                <span className={log.level === 'error' ? 'text-rose-400' : 'text-zinc-300'}>
                  {log.message}
                </span>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Deployment History */}
      <div className="bg-zinc-900 border border-zinc-800 rounded-xl overflow-hidden">
        <div className="p-4 border-b border-zinc-800 flex items-center justify-between">
          <div>
            <h3 className="text-sm font-bold text-zinc-100 flex items-center gap-2">
              <Clock className="w-4 h-4 text-purple-400" />
              Deployment History
            </h3>
            <p className="text-xs text-zinc-400 mt-0.5">
              Previous releases and isolate snapshots stored in Garage S3
            </p>
          </div>
          <button
            onClick={() => refetchDeployments()}
            className="p-1.5 rounded-lg border border-zinc-800 hover:bg-zinc-800 text-zinc-400 hover:text-zinc-200 transition"
          >
            <RefreshCw className="w-3.5 h-3.5" />
          </button>
        </div>

        {isLoading ? (
          <div className="p-8 text-center text-xs text-zinc-500">Loading deployments...</div>
        ) : !deployments || deployments.length === 0 ? (
          <div className="p-8 text-center text-xs text-zinc-500 flex flex-col items-center gap-2">
            <AlertCircle className="w-6 h-6 text-zinc-600" />
            No deployments recorded yet. Click &quot;Deploy Worker&quot; to build.
          </div>
        ) : (
          <div className="divide-y divide-zinc-800/60">
            {deployments.map((dep) => (
              <div
                key={dep.id}
                className="p-4 flex flex-col sm:flex-row sm:items-center justify-between gap-3 hover:bg-zinc-800/30 transition"
              >
                <div>
                  <div className="flex items-center gap-2">
                    <span className="font-mono text-xs font-bold text-zinc-200">
                      {dep.id.slice(0, 10)}
                    </span>
                    <span
                      className={`text-[10px] font-bold px-2 py-0.5 rounded-full ${getDeploymentStatusBadge(
                        dep.status || 'pending'
                      )}`}
                    >
                      {dep.status || 'pending'}
                    </span>
                  </div>
                  <p className="text-xs text-zinc-500 mt-1 font-mono">
                    Commit: {dep.commitHash ? dep.commitHash.slice(0, 7) : 'HEAD'} • Created:{' '}
                    {new Date(dep.createdAt || '').toLocaleString()}
                  </p>
                </div>

                <div className="flex items-center gap-2">
                  {dep.status === 'superseded' && (
                    <button
                      onClick={() => handleRollback(dep.id)}
                      disabled={rollingBackId === dep.id}
                      className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-zinc-800 hover:bg-zinc-700 text-xs font-semibold text-zinc-300 transition"
                    >
                      <RotateCcw className="w-3 h-3 text-amber-400" />
                      {rollingBackId === dep.id ? 'Rolling back...' : 'Rollback'}
                    </button>
                  )}
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
