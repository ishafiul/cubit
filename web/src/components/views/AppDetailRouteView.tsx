import { useParams, useNavigate, useSearch, Link } from '@tanstack/react-router';
import { ArrowLeft, AlertCircle } from 'lucide-react';
import { useQueryClient } from '@tanstack/react-query';
import {
  useListApplications,
  useDeleteApplication,
  getListApplicationsQueryKey,
} from '../../api/generated/applications/applications';
import { useDeploymentManager } from '../../shared/hooks/useDeploymentManager';
import { useModalActions } from '../../shared/stores/useModalStore';
import { useToastActions } from '../../shared/stores/useToastStore';
import type { Application } from '../../api/model';
import { ApplicationDetailPage, DetailTab } from '../ApplicationDetailPage';

export function AppDetailRouteView() {
  const { appId } = useParams({ from: '/apps/$appId' });
  const search = useSearch({ from: '/apps/$appId' }) as { tab?: DetailTab };
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  const { data: appsData, refetch: refetchApps } = useListApplications();
  const apps = Array.isArray(appsData) ? appsData : [];

  const { deployApp: handleDeployApp, isDeploying } = useDeploymentManager();
  const { setTestingApp } = useModalActions();
  const { addToast } = useToastActions();
  const deleteAppMutation = useDeleteApplication();

  const handleDeleteApp = async (id: string) => {
    if (!window.confirm('Are you sure you want to delete this application?')) {
      return;
    }
    try {
      await deleteAppMutation.mutateAsync({ id });
      queryClient.setQueryData(
        getListApplicationsQueryKey(),
        (old: Application[] | undefined) => (old ? old.filter((a) => a.id !== id) : [])
      );
      await queryClient.invalidateQueries({ queryKey: getListApplicationsQueryKey() });
      addToast({
        title: 'Worker Deleted',
        description: 'Worker deleted successfully.',
        variant: 'info',
      });
      navigate({ to: '/apps' });
    } catch (err: unknown) {
      console.error('Failed to delete application', err);
      addToast({
        title: 'Delete Failed',
        description: 'Failed to delete worker.',
        variant: 'error',
      });
    }
  };

  const app = apps.find(a => a.id === appId);

  if (!app) {
    return (
      <div className="p-8 max-w-lg mx-auto text-center space-y-4">
        <div className="w-12 h-12 rounded-full bg-zinc-900 border border-zinc-800 mx-auto flex items-center justify-center text-amber-400">
          <AlertCircle className="w-6 h-6" />
        </div>
        <h3 className="text-lg font-bold text-zinc-100">Application Not Found</h3>
        <p className="text-xs text-zinc-400">
          The requested application ID <code className="font-mono text-emerald-400">{appId}</code> does not exist or may have been deleted.
        </p>
        <Link
          to="/apps"
          className="inline-flex items-center gap-2 px-4 py-2 rounded-lg bg-zinc-800 hover:bg-zinc-700 text-xs font-semibold text-zinc-200 transition"
        >
          <ArrowLeft className="w-4 h-4" />
          Back to Workers
        </Link>
      </div>
    );
  }

  return (
    <div className="h-full flex flex-col">
      <ApplicationDetailPage
        app={app}
        allApps={apps}
        initialTab={search.tab || 'overview'}
        onTabChange={(newTab) => {
          navigate({
            to: '/apps/$appId',
            params: { appId },
            search: { tab: newTab },
            replace: true,
          });
        }}
        onBack={() => {
          navigate({ to: '/apps' });
        }}
        onDeploy={handleDeployApp}
        isDeploying={isDeploying === app.id}
        onTestApp={(targetApp) => setTestingApp(targetApp)}
        onRefreshApps={refetchApps}
        onDeleteApp={handleDeleteApp}
        onNavigateToService={(targetTab, resourceId) => {
          if (targetTab === 'apps' && resourceId) {
            navigate({
              to: '/apps/$appId',
              params: { appId: resourceId },
            });
            return;
          }
          (navigate as any)({
            to: `/${targetTab}`,
            search: resourceId ? { id: resourceId } : undefined,
          });
        }}
      />
    </div>
  );
}
