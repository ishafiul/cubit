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
import { AccessManagementView } from './components/views/AccessManagementView';
import { LoginPage } from './components/auth/LoginPage';
import { SetupPage } from './components/auth/SetupPage';
import { customInstance } from './api/custom-instance';
import { useAuthStore } from './shared/stores/useAuthStore';
import type { DetailTab } from './components/ApplicationDetailPage';

export interface ClusterStatus {
  initialized: boolean;
  version: string;
}

let cachedClusterStatus: ClusterStatus | null = null;

export function setCachedClusterStatus(status: ClusterStatus | null) {
  cachedClusterStatus = status;
}

export async function fetchClusterStatus(): Promise<ClusterStatus> {
  try {
    return await customInstance<ClusterStatus>({ url: '/auth/status' });
  } catch {
    return { initialized: true, version: '1.3.0' };
  }
}

export async function getClusterStatus(): Promise<ClusterStatus> {
  if (cachedClusterStatus !== null) {
    return cachedClusterStatus;
  }
  const status = await fetchClusterStatus();
  cachedClusterStatus = status;
  return status;
}

// Route guards
export async function requireAuthGuard() {
  const status = await getClusterStatus();
  if (!status.initialized) {
    throw redirect({ to: '/setup' });
  }
  if (!useAuthStore.getState().isAuthenticated) {
    throw redirect({ to: '/login' });
  }
}

export async function loginGuard() {
  const status = await getClusterStatus();
  if (!status.initialized) {
    throw redirect({ to: '/setup' });
  }
  if (useAuthStore.getState().isAuthenticated) {
    throw redirect({ to: '/apps' });
  }
}

export async function setupGuard() {
  const status = await getClusterStatus();
  if (status.initialized) {
    throw redirect({ to: '/login' });
  }
}

export async function indexGuard() {
  const status = await getClusterStatus();
  if (!status.initialized) {
    throw redirect({ to: '/setup' });
  }
  if (!useAuthStore.getState().isAuthenticated) {
    throw redirect({ to: '/login' });
  }
  throw redirect({ to: '/apps' });
}

// 1. Root Route
export const rootRoute = createRootRoute({
  component: App,
});

// 2. Authentication & Onboarding Routes
export const loginRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/login',
  beforeLoad: loginGuard,
  component: LoginPage,
});

export const setupRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/setup',
  beforeLoad: setupGuard,
  component: SetupPage,
});

// 3. Index Route (redirects to /apps if authenticated, or /login / /setup)
export const indexRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/',
  beforeLoad: indexGuard,
  component: () => <Navigate to="/apps" replace />,
});

// 4. Workers (Apps) Route
export const appsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/apps',
  beforeLoad: requireAuthGuard,
  component: WorkersView,
});

// 5. Worker Detail Route
export const appDetailRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/apps/$appId',
  beforeLoad: requireAuthGuard,
  validateSearch: (search: Record<string, unknown>): { tab?: DetailTab } => {
    const rawTab = typeof search.tab === 'string' ? search.tab : undefined;
    if (rawTab === 'domains') {
      return { tab: 'triggers' };
    }
    const validTabs: DetailTab[] = ['overview', 'code', 'builds', 'triggers', 'bindings', 'settings'];
    if (rawTab && validTabs.includes(rawTab as DetailTab)) {
      return { tab: rawTab as DetailTab };
    }
    return { tab: undefined };
  },
  component: AppDetailRouteView,
});

// 6. Service Routes
export const dynamicWorkersRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/dynamic-workers',
  beforeLoad: requireAuthGuard,
  component: DynamicWorkersView,
});

function DurableObjectsRouteComponent() {
  const search = useSearch({ from: '/durable-objects' });
  return <DurableObjectsView initialSelectedId={search.id} />;
}

export const durableObjectsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/durable-objects',
  beforeLoad: requireAuthGuard,
  validateSearch: (search: Record<string, unknown>): { id?: string } => ({
    id: typeof search.id === 'string' ? search.id : undefined,
  }),
  component: DurableObjectsRouteComponent,
});

export const containersRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/containers',
  beforeLoad: requireAuthGuard,
  component: ContainersView,
});

function KVRouteComponent() {
  const search = useSearch({ from: '/kv' });
  return <KVView initialSelectedId={search.id} />;
}

export const kvRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/kv',
  beforeLoad: requireAuthGuard,
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
  beforeLoad: requireAuthGuard,
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
  beforeLoad: requireAuthGuard,
  validateSearch: (search: Record<string, unknown>): { id?: string } => ({
    id: typeof search.id === 'string' ? search.id : undefined,
  }),
  component: R2RouteComponent,
});

export const staticAssetsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/static-assets',
  beforeLoad: requireAuthGuard,
  component: StaticAssetsView,
});

function CronRouteComponent() {
  const search = useSearch({ from: '/cron' });
  return <CronView initialSelectedId={search.id} />;
}

export const cronRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/cron',
  beforeLoad: requireAuthGuard,
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
  beforeLoad: requireAuthGuard,
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
  beforeLoad: requireAuthGuard,
  validateSearch: (search: Record<string, unknown>): { id?: string } => ({
    id: typeof search.id === 'string' ? search.id : undefined,
  }),
  component: WorkflowsRouteComponent,
});

// 7. Fleet & Routing
export const nodesRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/nodes',
  beforeLoad: requireAuthGuard,
  component: NodesView,
});

export const domainsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/domains',
  beforeLoad: requireAuthGuard,
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
  beforeLoad: requireAuthGuard,
  validateSearch: (search: Record<string, unknown>): { appId?: string } => ({
    appId: typeof search.appId === 'string' ? search.appId : undefined,
  }),
  component: LogsRouteComponent,
});

export const githubRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/github',
  beforeLoad: requireAuthGuard,
  component: GitHubSettingsView,
});

export const accessManagementRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/access',
  beforeLoad: requireAuthGuard,
  component: AccessManagementView,
});

// 8. Route Tree & Router
export const routeTree = rootRoute.addChildren([
  indexRoute,
  loginRoute,
  setupRoute,
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
  accessManagementRoute,
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
