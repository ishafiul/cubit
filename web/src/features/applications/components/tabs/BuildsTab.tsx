import { useState } from 'react';
import {
  Terminal,
  RefreshCw,
  RotateCcw,
  Clock,
  AlertCircle,
  Radio,
  Download,
  CheckCircle2,
  X,
  FileCode,
} from 'lucide-react';
import type { Application, Deployment } from '../../../../api/model';
import {
  useListDeployments,
  useRollbackApplication,
  useGetDeploymentLogs,
} from '../../../../api/generated/deployments/deployments';
import {
  useDeploymentLogs,
  useDeployStatus,
  useActiveDeploymentId,
} from '../../../../shared/stores/useDeploymentTrackerStore';
import { useToastActions } from '../../../../shared/stores/useToastStore';
import { getDeploymentStatusBadge } from '../WorkbenchHeader';
import {
  useBuildsViewMode,
  useSelectedDeploymentId,
  useApplicationActions,
} from '../../stores/applicationStore';
import { LiveTailView } from './LiveTailView';

export interface BuildsTabProps {
  app: Application;
  onTestApp?: (app: Application) => void;
}

export function BuildsTab({ app, onTestApp }: BuildsTabProps) {
  const { data: deployments, refetch: refetchDeployments, isLoading } = useListDeployments(app.id);

  const rollbackMutation = useRollbackApplication();
  const { addToast } = useToastActions();

  const activeDeploymentId = useActiveDeploymentId();
  const deployStatus = useDeployStatus();
  const liveLogs = useDeploymentLogs();

  const buildsViewMode = useBuildsViewMode();
  const selectedDeploymentId = useSelectedDeploymentId();
  const { setBuildsViewMode, setSelectedDeploymentId } = useApplicationActions();

  const [rollingBackId, setRollingBackId] = useState<string | null>(null);
  const [rollbackTargetDep, setRollbackTargetDep] = useState<Deployment | null>(null);
  const [isDownloadingBundle, setIsDownloadingBundle] = useState(false);

  const deploymentsList = Array.isArray(deployments) ? deployments : [];

  // Determine active/selected deployment
  const effectiveSelectedDepId = selectedDeploymentId || deploymentsList[0]?.id || null;
  const selectedDep = deploymentsList.find((d) => d.id === effectiveSelectedDepId) || deploymentsList[0] || null;

  // Query logs for selected deployment
  const { data: pastLogsData, isLoading: isLoadingPastLogs } = useGetDeploymentLogs(
    selectedDep?.id || '',
    {
      query: {
        enabled: Boolean(selectedDep?.id),
      },
    }
  );

  const pastLogs = Array.isArray(pastLogsData) ? pastLogsData : [];

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
      setRollbackTargetDep(null);
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

  const handleDownloadBundle = async (depId?: string, versionNum?: number) => {
    setIsDownloadingBundle(true);
    try {
      const targetId = depId || app.activeDeploymentId || '';
      const url = targetId
        ? `/api/v1/applications/${app.id}/bundle?deploymentId=${targetId}`
        : `/api/v1/applications/${app.id}/bundle`;
      const res = await fetch(url);
      if (!res.ok) throw new Error('Failed to fetch bundle');
      const text = await res.text();
      const blob = new Blob([text], { type: 'application/javascript' });
      const downloadUrl = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = downloadUrl;
      const v = versionNum ? `v${versionNum}` : 'bundle';
      a.download = `${app.name}-${v}.js`;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(downloadUrl);
      addToast({
        title: 'Bundle Downloaded',
        description: `Saved ${app.name}-${v}.js`,
        variant: 'success',
      });
    } catch (err: any) {
      addToast({
        title: 'Download Failed',
        description: err?.message || 'Could not download compiled bundle.',
        variant: 'error',
      });
    } finally {
      setIsDownloadingBundle(false);
    }
  };

  return (
    <div data-testid="tab-content-builds" className="p-6 max-w-6xl mx-auto w-full space-y-6">
      {/* Sub-View Switcher: Deployment History vs Live Request Tail */}
      <div className="flex items-center justify-between border-b border-zinc-800 pb-3">
        <div className="flex items-center gap-2">
          <button
            type="button"
            data-testid="mode-history-btn"
            onClick={() => setBuildsViewMode('history')}
            className={`px-3 py-1.5 rounded-lg text-xs font-semibold flex items-center gap-2 transition ${
              buildsViewMode === 'history'
                ? 'bg-zinc-800 text-zinc-100 border border-zinc-700'
                : 'text-zinc-400 hover:text-zinc-200 hover:bg-zinc-900'
            }`}
          >
            <Terminal className="w-3.5 h-3.5" />
            <span>Deployment History ({deploymentsList.length})</span>
          </button>
          <button
            type="button"
            data-testid="mode-tail-btn"
            onClick={() => setBuildsViewMode('tail')}
            className={`px-3 py-1.5 rounded-lg text-xs font-semibold flex items-center gap-2 transition ${
              buildsViewMode === 'tail'
                ? 'bg-emerald-950/80 text-emerald-400 border border-emerald-800'
                : 'text-zinc-400 hover:text-zinc-200 hover:bg-zinc-900'
            }`}
          >
            <Radio className="w-3.5 h-3.5 text-emerald-400 animate-pulse" />
            <span>Live Request Tail (Realtime SSE)</span>
          </button>
        </div>
      </div>

      {buildsViewMode === 'tail' ? (
        <LiveTailView app={app} onTestApp={onTestApp} />
      ) : (
        <>
          {/* Active Build Stream if currently building or active logs exist */}
          {(activeDeploymentId || liveLogs.length > 0) && (
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
                {liveLogs.map((log, idx) => (
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

          {/* Two-Column Split Layout for Deployment History */}
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
                type="button"
                data-testid="refresh-deployments-btn"
                onClick={() => refetchDeployments()}
                className="p-1.5 rounded-lg border border-zinc-800 hover:bg-zinc-800 text-zinc-400 hover:text-zinc-200 transition"
              >
                <RefreshCw className="w-3.5 h-3.5" />
              </button>
            </div>

            {isLoading ? (
              <div className="p-8 text-center text-xs text-zinc-500">Loading deployments...</div>
            ) : deploymentsList.length === 0 ? (
              <div className="p-8 text-center text-xs text-zinc-500 flex flex-col items-center gap-2">
                <AlertCircle className="w-6 h-6 text-zinc-600" />
                No deployments recorded yet. Click &quot;Deploy Worker&quot; to build.
              </div>
            ) : (
              <div className="grid grid-cols-1 lg:grid-cols-12 min-h-[480px]">
                {/* Left Column: Deployment List */}
                <div className="lg:col-span-5 border-r border-zinc-800/80 divide-y divide-zinc-800/60 max-h-[640px] overflow-y-auto">
                  {deploymentsList.map((dep) => {
                    const isSelected = selectedDep?.id === dep.id;
                    const isLive = app.activeDeploymentId === dep.id || dep.status === 'active';
                    const d = new Date(dep.createdAt || '');

                    return (
                      <div
                        key={dep.id}
                        data-testid={`deployment-item-${dep.id}`}
                        onClick={() => setSelectedDeploymentId(dep.id)}
                        className={`p-4 transition cursor-pointer flex flex-col gap-2 ${
                          isSelected
                            ? 'bg-zinc-800/60 border-l-4 border-l-emerald-500'
                            : 'hover:bg-zinc-800/30'
                        }`}
                      >
                        <div className="flex items-center justify-between">
                          <div className="flex items-center gap-2">
                            <span className="font-mono text-xs font-bold text-zinc-200">
                              v{dep.buildVersion || 1}
                            </span>
                            {isLive && (
                              <span className="text-[10px] px-2 py-0.5 rounded-full font-bold uppercase bg-emerald-950 text-emerald-400 border border-emerald-800 flex items-center gap-1">
                                <span className="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse" />
                                LIVE
                              </span>
                            )}
                            <span
                              className={`text-[10px] px-2 py-0.5 rounded-full font-bold uppercase ${getDeploymentStatusBadge(
                                dep.status || 'pending'
                              )}`}
                            >
                              {dep.status || 'pending'}
                            </span>
                          </div>

                          {dep.status === 'superseded' && (
                            <button
                              type="button"
                              data-testid={`rollback-btn-${dep.id}`}
                              onClick={(e) => {
                                e.stopPropagation();
                                setRollbackTargetDep(dep);
                              }}
                              disabled={rollingBackId === dep.id}
                              className="flex items-center gap-1 px-2.5 py-1 rounded bg-zinc-800 hover:bg-zinc-700 text-[11px] font-semibold text-zinc-300 transition"
                            >
                              <RotateCcw className="w-3 h-3 text-amber-400" />
                              Rollback
                            </button>
                          )}
                        </div>

                        <div className="text-xs text-zinc-400 truncate">
                          <span className="font-mono text-zinc-300">
                            {dep.commitHash ? dep.commitHash.slice(0, 7) : 'HEAD'}
                          </span>
                          {' — '}
                          <span>{dep.commitMessage || 'Manual deployment'}</span>
                        </div>

                        <div className="flex items-center justify-between text-[11px] text-zinc-500 font-mono">
                          <span>{d.toLocaleString()}</span>
                          {dep.bundleSize != null && dep.bundleSize > 0 && (
                            <span>{((dep.bundleSize) / 1024).toFixed(1)} KB</span>
                          )}
                        </div>
                      </div>
                    );
                  })}
                </div>

                {/* Right Column: Deployment Details & Build Console */}
                <div className="lg:col-span-7 flex flex-col bg-zinc-950">
                  {selectedDep ? (
                    <>
                      {/* Header */}
                      <div className="p-4 border-b border-zinc-800 flex items-center justify-between bg-zinc-900/60">
                        <div>
                          <div className="flex items-center gap-2">
                            <span className="font-mono text-sm font-bold text-zinc-200">
                              Release v{selectedDep.buildVersion || 1}
                            </span>
                            <span className="font-mono text-xs text-zinc-500">
                              ({selectedDep.id.slice(0, 8)})
                            </span>
                            <span
                              className={`text-[10px] font-bold px-2 py-0.5 rounded-full ${getDeploymentStatusBadge(
                                selectedDep.status || 'pending'
                              )}`}
                            >
                              {selectedDep.status || 'pending'}
                            </span>
                          </div>
                          <p className="text-xs text-zinc-400 mt-0.5 font-mono">
                            Commit: {selectedDep.commitHash ? selectedDep.commitHash.slice(0, 7) : 'HEAD'} •{' '}
                            {new Date(selectedDep.createdAt || '').toLocaleString()}
                          </p>
                        </div>

                        <button
                          type="button"
                          data-testid="download-bundle-btn"
                          onClick={() => handleDownloadBundle(selectedDep.id, selectedDep.buildVersion)}
                          disabled={isDownloadingBundle}
                          className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg border border-zinc-700 bg-zinc-800 hover:bg-zinc-700 text-xs font-semibold text-zinc-200 transition"
                          title="Download compiled worker bundle"
                        >
                          <Download className={`w-3.5 h-3.5 text-zinc-400 ${isDownloadingBundle ? 'animate-bounce' : ''}`} />
                          <span>{isDownloadingBundle ? 'Downloading...' : 'Download Bundle'}</span>
                        </button>
                      </div>

                      {/* Build Output Console */}
                      <div className="flex-1 p-4 font-mono text-xs overflow-y-auto max-h-[560px] space-y-1.5 bg-zinc-950">
                        <div className="text-zinc-600 text-[11px] pb-2 border-b border-zinc-900">
                          // Build logs recorded for release {selectedDep.id.slice(0, 8)}
                        </div>

                        {isLoadingPastLogs ? (
                          <div className="p-6 text-center text-zinc-600">Loading build logs...</div>
                        ) : pastLogs.length === 0 ? (
                          <div className="p-6 text-center text-zinc-600 flex flex-col items-center gap-1">
                            <FileCode className="w-5 h-5 text-zinc-700" />
                            <span>No build log entries recorded for this deployment.</span>
                          </div>
                        ) : (
                          pastLogs.map((l, idx) => (
                            <div key={idx} className="flex items-start gap-2">
                              <span className="text-zinc-600 shrink-0">
                                {new Date(l.timestamp).toLocaleTimeString()}
                              </span>
                              <span className="text-emerald-500 font-semibold shrink-0">
                                [{l.step}]
                              </span>
                              <span className={l.level === 'error' ? 'text-rose-400' : 'text-zinc-300'}>
                                {l.message}
                              </span>
                            </div>
                          ))
                        )}
                      </div>
                    </>
                  ) : (
                    <div className="p-8 text-center text-xs text-zinc-500">
                      Select a deployment to view details
                    </div>
                  )}
                </div>
              </div>
            )}
          </div>
        </>
      )}

      {/* Rollback Confirmation Modal */}
      {rollbackTargetDep && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/75 backdrop-blur-sm p-4">
          <div className="w-full max-w-md bg-zinc-900 border border-zinc-800 rounded-2xl p-6 shadow-2xl space-y-4">
            <div className="flex items-center justify-between">
              <h3 className="text-base font-bold text-zinc-100 flex items-center gap-2">
                <RotateCcw className="w-4 h-4 text-amber-400" />
                Confirm Zero-Downtime Rollback
              </h3>
              <button
                type="button"
                onClick={() => setRollbackTargetDep(null)}
                className="p-1 rounded-lg text-zinc-400 hover:text-zinc-200 hover:bg-zinc-800 transition"
              >
                <X className="w-4 h-4" />
              </button>
            </div>

            <div className="p-3.5 rounded-xl bg-amber-950/20 border border-amber-800/40 text-xs text-amber-300 space-y-1">
              <span className="font-semibold block">Target Release: v{rollbackTargetDep.buildVersion || 1}</span>
              <p className="text-[11px] text-zinc-400">
                Commit: <code className="font-mono text-zinc-200">{rollbackTargetDep.commitHash ? rollbackTargetDep.commitHash.slice(0, 7) : 'HEAD'}</code> ({rollbackTargetDep.commitMessage || 'Manual release'})
              </p>
              <p className="text-[11px] text-zinc-400 pt-1">
                Traefik ingress traffic will be instantaneously repointed to this isolate revision snapshot without downtime.
              </p>
            </div>

            <div className="flex justify-end gap-2 pt-2">
              <button
                type="button"
                onClick={() => setRollbackTargetDep(null)}
                className="px-3.5 py-1.5 rounded-lg border border-zinc-700 bg-zinc-800 hover:bg-zinc-700 text-xs font-semibold text-zinc-300 transition"
              >
                Cancel
              </button>
              <button
                type="button"
                data-testid="confirm-rollback-btn"
                onClick={() => handleRollback(rollbackTargetDep.id)}
                disabled={rollingBackId === rollbackTargetDep.id}
                className="px-4 py-1.5 rounded-lg bg-amber-600 hover:bg-amber-500 disabled:opacity-50 text-xs font-semibold text-white transition shadow flex items-center gap-1.5"
              >
                {rollingBackId === rollbackTargetDep.id ? (
                  <RefreshCw className="w-3.5 h-3.5 animate-spin" />
                ) : (
                  <CheckCircle2 className="w-3.5 h-3.5" />
                )}
                <span>{rollingBackId === rollbackTargetDep.id ? 'Rolling back...' : 'Confirm Rollback'}</span>
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
