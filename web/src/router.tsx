import {
  createRootRoute,
  createRoute,
  createRouter,
  Navigate,
  useSearch,
  useNavigate,
  redirect,
} from '@tanstack/react-router';
import App from './App';
import { WorkersView } from './components/views/WorkersView';
import { AppDetailRouteView } from './components/views/AppDetailRouteView';
import { NodesView } from './components/views/NodesView';
import { DomainsView } from './components/views/DomainsView';
import { BuildLogsView } from './components/views/BuildLogsView';
import { KVView } from './components/services/KVView';
import { D1View } from './components/services/D1View';
import { R2View } from './components/services/R2View';
import { StaticAssetsView } from './components/services/StaticAssetsView';
import { DynamicWorkersView } from './components/services/DynamicWorkersView';
import { CronView } from './components/services/CronView';
import { QueuesView } from './components/services/QueuesView';
import { WorkflowsView } from './components/services/WorkflowsView';
import { DurableObjectsView } from './components/services/DurableObjectsView';
import { ContainersView } from './components/services/ContainersView';
import { GitHubSettingsView } from './components/views/GitHubSettingsView';
import type { DetailTab } from './components/ApplicationDetailPage';

// 1. Root Route
export const rootRoute = createRootRoute({
  component: App,
});

// 2. Index Route (redirects to /apps)
export const indexRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/',
  beforeLoad: () => {
    throw redirect({ to: '/apps' });
  },
  component: () => <Navigate to="/apps" replace />,
});

// 3. Workers (Apps) Route
export const appsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/apps',
  component: WorkersView,
});

// 4. Worker Detail Route
export const appDetailRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/apps/$appId',
  validateSearch: (search: Record<string, unknown>): { tab?: DetailTab } => ({
    tab: typeof search.tab === 'string' ? (search.tab as DetailTab) : undefined,
  }),
  component: AppDetailRouteView,
});

// 5. Service Routes
export const dynamicWorkersRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/dynamic-workers',
  component: DynamicWorkersView,
});

function DurableObjectsRouteComponent() {
  const search = useSearch({ from: '/durable-objects' });
  return <DurableObjectsView initialSelectedId={search.id} />;
}

export const durableObjectsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/durable-objects',
  validateSearch: (search: Record<string, unknown>): { id?: string } => ({
    id: typeof search.id === 'string' ? search.id : undefined,
  }),
  component: DurableObjectsRouteComponent,
});

export const containersRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/containers',
  component: ContainersView,
});

function KVRouteComponent() {
  const search = useSearch({ from: '/kv' });
  return <KVView initialSelectedId={search.id} />;
}

export const kvRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/kv',
  validateSearch: (search: Record<string, unknown>): { id?: string } => ({
    id: typeof search.id === 'string' ? search.id : undefined,
  }),
  component: KVRouteComponent,
});

function D1RouteComponent() {
  const search = useSearch({ from: '/d1' });
  return <D1View initialSelectedId={search.id} />;
}

export const d1Route = createRoute({
  getParentRoute: () => rootRoute,
  path: '/d1',
  validateSearch: (search: Record<string, unknown>): { id?: string } => ({
    id: typeof search.id === 'string' ? search.id : undefined,
  }),
  component: D1RouteComponent,
});

function R2RouteComponent() {
  const search = useSearch({ from: '/r2' });
  return <R2View initialSelectedId={search.id} />;
}

export const r2Route = createRoute({
  getParentRoute: () => rootRoute,
  path: '/r2',
  validateSearch: (search: Record<string, unknown>): { id?: string } => ({
    id: typeof search.id === 'string' ? search.id : undefined,
  }),
  component: R2RouteComponent,
});

export const staticAssetsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/static-assets',
  component: StaticAssetsView,
});

function CronRouteComponent() {
  const search = useSearch({ from: '/cron' });
  return <CronView initialSelectedId={search.id} />;
}

export const cronRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/cron',
  validateSearch: (search: Record<string, unknown>): { id?: string } => ({
    id: typeof search.id === 'string' ? search.id : undefined,
  }),
  component: CronRouteComponent,
});

function QueuesRouteComponent() {
  const search = useSearch({ from: '/queues' });
  return <QueuesView initialSelectedId={search.id} />;
}

export const queuesRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/queues',
  validateSearch: (search: Record<string, unknown>): { id?: string } => ({
    id: typeof search.id === 'string' ? search.id : undefined,
  }),
  component: QueuesRouteComponent,
});

function WorkflowsRouteComponent() {
  const search = useSearch({ from: '/workflows' });
  return <WorkflowsView initialSelectedId={search.id} />;
}

export const workflowsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/workflows',
  validateSearch: (search: Record<string, unknown>): { id?: string } => ({
    id: typeof search.id === 'string' ? search.id : undefined,
  }),
  component: WorkflowsRouteComponent,
});

// 6. Fleet & Routing
export const nodesRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/nodes',
  component: NodesView,
});

export const domainsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/domains',
  component: DomainsView,
});

function LogsRouteComponent() {
  const search = useSearch({ from: '/logs' });
  const navigate = useNavigate();
  return (
    <BuildLogsView
      initialAppId={search.appId}
      onAppChange={(newAppId) => {
        navigate({
          to: '/logs',
          search: newAppId ? { appId: newAppId } : {},
          replace: true,
        });
      }}
    />
  );
}

export const logsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/logs',
  validateSearch: (search: Record<string, unknown>): { appId?: string } => ({
    appId: typeof search.appId === 'string' ? search.appId : undefined,
  }),
  component: LogsRouteComponent,
});

export const githubRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/github',
  component: GitHubSettingsView,
});

// 7. Route Tree & Router
export const routeTree = rootRoute.addChildren([
  indexRoute,
  appsRoute,
  appDetailRoute,
  dynamicWorkersRoute,
  durableObjectsRoute,
  containersRoute,
  kvRoute,
  d1Route,
  r2Route,
  staticAssetsRoute,
  cronRoute,
  queuesRoute,
  workflowsRoute,
  nodesRoute,
  domainsRoute,
  logsRoute,
  githubRoute,
]);

export const router = createRouter({
  routeTree,
  defaultPreload: 'intent',
});

declare module '@tanstack/react-router' {
  interface Register {
    router: typeof router;
  }
}
