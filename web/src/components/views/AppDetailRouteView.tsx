import { useParams, useNavigate, useSearch, Link } from '@tanstack/react-router';
import { ArrowLeft, AlertCircle } from 'lucide-react';
import { useDashboard } from '../../context/DashboardContext';
import { ApplicationDetailPage, DetailTab } from '../ApplicationDetailPage';

export function AppDetailRouteView() {
  const { appId } = useParams({ from: '/apps/$appId' });
  const search = useSearch({ from: '/apps/$appId' }) as { tab?: DetailTab };
  const navigate = useNavigate();

  const {
    apps,
    handleDeployApp,
    isDeploying,
    setTestingApp,
    refetchApps,
  } = useDashboard();

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
        initialTab={search.tab || 'builds'}
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
        onNavigateToService={(targetTab, resourceId) => {
          (navigate as any)({
            to: `/${targetTab}`,
            search: resourceId ? { id: resourceId } : undefined,
          });
        }}
      />
    </div>
  );
}
