import { useNavigate } from '@tanstack/react-router';
import { Layers, Plus, ShieldCheck, X, RefreshCw } from 'lucide-react';
import { useQueryClient } from '@tanstack/react-query';
import {
  useListApplications,
  useDeleteApplication,
  getListApplicationsQueryKey,
} from '../../api/generated/applications/applications';
import { useModalActions } from '../../shared/stores/useModalStore';
import { useDeploymentManager } from '../../shared/hooks/useDeploymentManager';
import { useToastActions } from '../../shared/stores/useToastStore';
import { ApplicationCard } from '../modals/ApplicationCard';
import type { Application } from '../../api/model';

export function WorkersView() {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { data: appsData } = useListApplications();
  const apps = Array.isArray(appsData) ? appsData : [];

  const {
    deployApp: handleDeployApp,
    isDeploying,
    activeDeploymentId,
    deployStatus,
    logs,
  } = useDeploymentManager();

  const {
    setShowNewAppModal,
    setViewingCodeApp,
    setTestingApp,
    setHistoryApp,
  } = useModalActions();
  const { addToast } = useToastActions();

  const deleteAppMutation = useDeleteApplication();

  const handleDeleteApp = async (appId: string) => {
    if (!window.confirm('Are you sure you want to delete this application?')) {
      return;
    }
    try {
      await deleteAppMutation.mutateAsync({ id: appId });
      queryClient.setQueryData(
        getListApplicationsQueryKey(),
        (old: Application[] | undefined) => (old ? old.filter((a) => a.id !== appId) : [])
      );
      await queryClient.invalidateQueries({ queryKey: getListApplicationsQueryKey() });
      addToast({
        title: 'Worker Deleted',
        description: 'Worker deleted successfully.',
        variant: 'info',
      });
    } catch (err: unknown) {
      console.error('Failed to delete application', err);
      addToast({
        title: 'Delete Failed',
        description: 'Failed to delete worker.',
        variant: 'error',
      });
    }
  };

  return (
    <div className="space-y-6">
      {/* Header action if empty */}
      {apps.length === 0 ? (
        <div className="p-12 rounded-2xl border border-dashed border-zinc-800 bg-zinc-950/40 text-center space-y-4">
          <div className="w-12 h-12 rounded-full bg-zinc-900 border border-zinc-800 mx-auto flex items-center justify-center text-zinc-500">
            <Layers className="w-6 h-6" />
          </div>
          <div className="space-y-1">
            <h3 className="text-base font-bold text-zinc-200">No Cloudflare Workers Created</h3>
            <p className="text-xs text-zinc-500 max-w-md mx-auto">
              Deploy your first Cloudflare Worker onto your bare-metal celld fleet using inline JavaScript or a GitHub repository.
            </p>
          </div>
          <button
            type="button"
            onClick={() => setShowNewAppModal(true)}
            className="inline-flex items-center gap-2 px-4 py-2 rounded-lg bg-emerald-500 hover:bg-emerald-600 text-zinc-950 text-xs font-bold transition shadow-sm"
          >
            <Plus className="w-4 h-4" />
            Create Worker
          </button>
        </div>
      ) : (
        <div className="space-y-4">
          <div className="grid grid-cols-1 gap-4">
            {apps.map(app => (
              <ApplicationCard
                key={app.id}
                app={app}
                onOpenApp={(selectedApp: Application) => {
                  navigate({
                    to: '/apps/$appId',
                    params: { appId: selectedApp.id },
                  });
                }}
                onDeploy={handleDeployApp}
                isDeploying={isDeploying === app.id}
                onViewCode={setViewingCodeApp}
                onTestApp={setTestingApp}
                onViewHistory={setHistoryApp}
                onDelete={handleDeleteApp}
              />
            ))}
          </div>
        </div>
      )}

      {/* Active Deployment Stream (Bottom card if active or logs exist) */}
      {(activeDeploymentId || logs.length > 0) && (
        <div className="p-5 rounded-2xl border border-zinc-800/80 bg-zinc-900/30 space-y-3">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2.5">
              <span className="text-xs font-semibold text-zinc-400 uppercase tracking-wider">
                Live Build & Deployment Stream
              </span>
              {activeDeploymentId && (
                <span className="text-xs font-mono text-zinc-500">ID: {activeDeploymentId.slice(0, 8)}</span>
              )}
            </div>
            {deployStatus && (
              <span className={`text-[10px] font-mono px-2 py-0.5 rounded-full border uppercase font-bold flex items-center gap-1.5 ${
                deployStatus === 'active'
                  ? 'bg-emerald-950/60 border-emerald-700/80 text-emerald-400'
                  : deployStatus === 'failed'
                  ? 'bg-rose-950/60 border-rose-700/80 text-rose-400'
                  : 'bg-amber-950/60 border-amber-700/80 text-amber-400'
              }`}>
                {deployStatus === 'active' && <span className="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse" />}
                {deployStatus === 'building' && <RefreshCw className="w-3 h-3 animate-spin text-amber-400" />}
                {deployStatus}
              </span>
            )}
          </div>

          {deployStatus === 'active' && (
            <div className="p-3 bg-emerald-950/40 border border-emerald-800/80 rounded-xl flex items-center justify-between text-xs text-emerald-300">
              <div className="flex items-center gap-2">
                <ShieldCheck className="w-4 h-4 text-emerald-400" />
                <span className="font-semibold">Worker active and routing traffic live!</span>
              </div>
              <span className="text-[11px] text-emerald-500/80">Traefik synchronized</span>
            </div>
          )}

          {deployStatus === 'failed' && (
            <div className="p-3 bg-rose-950/40 border border-rose-800/80 rounded-xl flex items-center justify-between text-xs text-rose-300">
              <div className="flex items-center gap-2">
                <X className="w-4 h-4 text-rose-400" />
                <span className="font-semibold">Build/Deploy failed.</span>
              </div>
              <span className="text-[11px] text-rose-500/80">Inspect step details below</span>
            </div>
          )}

          <div className="bg-black/80 rounded-xl p-4 font-mono text-xs text-zinc-300 max-h-48 overflow-y-auto border border-zinc-900 space-y-1 shadow-inner">
            {logs.length === 0 ? (
              <div className="text-zinc-600 italic">Waiting for deployment output...</div>
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
                    <span className={l.level === 'error' ? 'text-red-400 font-semibold' : 'text-zinc-200'}>{l.message}</span>
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
