import { Outlet, Link, useLocation } from '@tanstack/react-router';
import {
  Server,
  Layers,
  Globe,
  RefreshCw,
  Terminal,
  Plus,
  HardDrive,
  Code,
  Send,
  Database,
  Box,
  Cpu,
  Clock,
  GitMerge,
  FileCode,
  GitBranch,
} from 'lucide-react';
import { DashboardProvider, useDashboard } from './context/DashboardContext';
import { NewAppModal } from './components/modals/NewAppModal';
import { AddNodeModal } from './components/modals/AddNodeModal';
import { UpgradeModal } from './components/modals/UpgradeModal';
import { NewDomainModal } from './components/modals/NewDomainModal';
import { ViewCodeModal } from './components/modals/ViewCodeModal';
import { TestAppModal } from './components/modals/TestAppModal';
import { BuildHistoryModal } from './components/modals/BuildHistoryModal';

export type ActiveTab =
  | 'nodes'
  | 'apps'
  | 'dynamic-workers'
  | 'durable-objects'
  | 'containers'
  | 'kv'
  | 'd1'
  | 'r2'
  | 'static-assets'
  | 'cron'
  | 'queues'
  | 'workflows'
  | 'domains'
  | 'logs';

function getHeaderTitle(pathname: string): string {
  if (pathname.startsWith('/apps')) return 'Workers';
  if (pathname.startsWith('/dynamic-workers')) return 'Dynamic Workers';
  if (pathname.startsWith('/durable-objects')) return 'Durable Objects & Facets';
  if (pathname.startsWith('/containers')) return 'Containers (Experimental)';
  if (pathname.startsWith('/static-assets')) return 'Static Assets';
  if (pathname.startsWith('/cron')) return 'Cron Triggers';
  if (pathname.startsWith('/queues')) return 'Queues';
  if (pathname.startsWith('/workflows')) return 'Workflows';
  if (pathname.startsWith('/kv')) return 'KV Namespaces';
  if (pathname.startsWith('/d1')) return 'D1 SQL Databases';
  if (pathname.startsWith('/r2')) return 'R2 Object Storage';
  if (pathname.startsWith('/nodes')) return 'Fleet Nodes';
  if (pathname.startsWith('/domains')) return 'Traefik Routing';
  if (pathname.startsWith('/logs')) return 'Build Logs';
  if (pathname.startsWith('/github')) return 'GitHub Integration';
  return 'Workers';
}

function DashboardLayout() {
  const location = useLocation();
  const {
    nodes,
    apps,
    domains,
    runtimeStatus,
    setShowNewAppModal,
    setShowAddNodeModal,
    setShowUpgradeModal,
    setShowNewDomainModal,
    testingApp,
    setTestingApp,
    historyApp,
    setHistoryApp,
  } = useDashboard();

  const pathname = location.pathname;
  const isAppDetailPage = pathname.startsWith('/apps/') && pathname !== '/apps';
  const headerTitle = getHeaderTitle(pathname);

  const navItemClass = (isActive: boolean) =>
    `w-full flex items-center gap-2.5 px-3 py-1.5 rounded-lg text-xs font-medium transition ${
      isActive
        ? 'bg-zinc-800 text-emerald-400 shadow-sm'
        : 'text-zinc-400 hover:bg-zinc-900 hover:text-zinc-200'
    }`;

  return (
    <div className="flex h-screen bg-zinc-950 text-zinc-100 font-sans selection:bg-emerald-500/20 antialiased overflow-hidden">
      {/* Left Sidebar */}
      <aside className="w-64 border-r border-zinc-800/80 bg-zinc-950/40 p-4 flex flex-col justify-between shrink-0">
        <div className="space-y-6">
          {/* Logo & Platform Info */}
          <div className="flex items-center gap-3 px-2 py-1">
            <div className="w-8 h-8 rounded-lg bg-emerald-500/10 border border-emerald-500/20 flex items-center justify-center text-emerald-400">
              <Layers className="w-4 h-4" />
            </div>
            <div>
              <h1 className="text-sm font-bold tracking-tight text-zinc-100 flex items-center gap-1.5">
                CUBIT
                <span className="text-[10px] px-1.5 py-0.2 rounded bg-zinc-800 text-zinc-400 font-mono">
                  v0.2.0
                </span>
              </h1>
              <p className="text-[11px] text-zinc-500">celld Fleet Manager</p>
            </div>
          </div>

          <nav className="space-y-4 overflow-y-auto max-h-[calc(100vh-250px)] pr-1">
            {/* 1. Compute Section */}
            <div className="space-y-0.5">
              <span className="px-3 text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">
                Compute
              </span>

              <Link
                to="/apps"
                className={navItemClass(pathname.startsWith('/apps'))}
              >
                <Layers className="w-3.5 h-3.5" />
                <span>Workers</span>
                <span className="ml-auto text-[10px] px-1.5 py-0.5 rounded-full bg-zinc-800 text-zinc-400">
                  {apps.length}
                </span>
              </Link>

              <Link
                to="/dynamic-workers"
                className={navItemClass(pathname === '/dynamic-workers')}
              >
                <Code className="w-3.5 h-3.5" />
                <span>Dynamic Workers</span>
              </Link>

              <Link
                to="/durable-objects"
                className={navItemClass(pathname === '/durable-objects')}
              >
                <Box className="w-3.5 h-3.5" />
                <span>Durable Objects</span>
              </Link>

              <Link
                to="/containers"
                className={navItemClass(pathname === '/containers')}
              >
                <Cpu className="w-3.5 h-3.5" />
                <span>Containers</span>
                <span className="ml-auto text-[9px] px-1.5 py-0.2 rounded-full bg-amber-950 text-amber-400 border border-amber-800/80">
                  Exp
                </span>
              </Link>
            </div>

            {/* 2. Storage & Databases */}
            <div className="space-y-0.5">
              <span className="px-3 text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">
                Storage & Data
              </span>

              <Link
                to="/kv"
                className={navItemClass(pathname === '/kv')}
              >
                <Database className="w-3.5 h-3.5" />
                <span>KV Namespaces</span>
              </Link>

              <Link
                to="/d1"
                className={navItemClass(pathname === '/d1')}
              >
                <Database className="w-3.5 h-3.5 text-emerald-400" />
                <span>D1 SQL</span>
              </Link>

              <Link
                to="/r2"
                className={navItemClass(pathname === '/r2')}
              >
                <HardDrive className="w-3.5 h-3.5" />
                <span>R2 Storage</span>
              </Link>

              <Link
                to="/static-assets"
                className={navItemClass(pathname === '/static-assets')}
              >
                <FileCode className="w-3.5 h-3.5" />
                <span>Static Assets</span>
              </Link>
            </div>

            {/* 3. Automation & Messaging */}
            <div className="space-y-0.5">
              <span className="px-3 text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">
                Automation & Events
              </span>

              <Link
                to="/cron"
                className={navItemClass(pathname === '/cron')}
              >
                <Clock className="w-3.5 h-3.5" />
                <span>Cron Triggers</span>
              </Link>

              <Link
                to="/queues"
                className={navItemClass(pathname === '/queues')}
              >
                <Send className="w-3.5 h-3.5" />
                <span>Queues</span>
              </Link>

              <Link
                to="/workflows"
                className={navItemClass(pathname === '/workflows')}
              >
                <GitMerge className="w-3.5 h-3.5" />
                <span>Workflows</span>
              </Link>
            </div>

            {/* 4. Infrastructure & Fleet */}
            <div className="space-y-0.5">
              <span className="px-3 text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">
                Fleet & Routing
              </span>

              <Link
                to="/nodes"
                className={navItemClass(pathname === '/nodes')}
              >
                <Server className="w-3.5 h-3.5" />
                <span>Fleet Nodes</span>
                <span className="ml-auto text-[10px] px-1.5 py-0.5 rounded-full bg-zinc-800 text-zinc-400">
                  {nodes.length}
                </span>
              </Link>

              <Link
                to="/domains"
                className={navItemClass(pathname === '/domains')}
              >
                <Globe className="w-3.5 h-3.5" />
                <span>Traefik Routing</span>
                <span className="ml-auto text-[10px] px-1.5 py-0.5 rounded-full bg-zinc-800 text-zinc-400">
                  {domains.length}
                </span>
              </Link>

              <Link
                to="/logs"
                className={navItemClass(pathname === '/logs')}
              >
                <Terminal className="w-3.5 h-3.5" />
                <span>Build Logs</span>
              </Link>

              <Link
                to="/github"
                className={navItemClass(pathname === '/github')}
              >
                <GitBranch className="w-3.5 h-3.5 text-purple-400" />
                <span>GitHub App</span>
              </Link>
            </div>
          </nav>
        </div>

        {/* Fleet Status Badge */}
        <div className="p-3.5 rounded-xl border border-zinc-800 bg-zinc-900/60 text-xs space-y-2 mt-2">
          <div className="flex items-center justify-between text-zinc-400">
            <span className="flex items-center gap-1.5">
              <span className="w-2 h-2 rounded-full bg-emerald-400 animate-pulse" />
              celld Runtime
            </span>
            <span className="text-emerald-400 font-mono font-bold">
              v{runtimeStatus?.currentCelldVersion || '0.5.1'}
            </span>
          </div>
          <div className="flex items-center justify-between text-zinc-400">
            <span>Storage Driver</span>
            <span className="text-zinc-200 uppercase font-semibold text-[11px]">
              {runtimeStatus?.storageBackend || 'Garage S3'}
            </span>
          </div>
          {runtimeStatus?.isUpgrading && (
            <div className="pt-2 flex items-center gap-2 text-amber-400 animate-pulse">
              <RefreshCw className="w-3.5 h-3.5 animate-spin" />
              <span>Rolling upgrade in progress...</span>
            </div>
          )}
        </div>
      </aside>

      {/* Main Content Area */}
      <main className="flex-1 flex flex-col overflow-hidden">
        {/* Top Navbar - Hidden when viewing an application detail page */}
        {!isAppDetailPage && (
          <header className="h-16 border-b border-zinc-800/80 px-8 flex items-center justify-between bg-zinc-950/60 backdrop-blur">
            <div className="flex items-center gap-4">
              <h2 className="text-xl font-semibold tracking-tight">{headerTitle}</h2>
            </div>

            <div className="flex items-center gap-3">
              <button
                type="button"
                onClick={() => setShowUpgradeModal(true)}
                className="flex items-center gap-2 px-3.5 py-1.5 rounded-lg border border-zinc-700 bg-zinc-900 hover:bg-zinc-800 text-sm font-medium transition"
              >
                <RefreshCw className="w-3.5 h-3.5 text-emerald-400" />
                Upgrade celld
              </button>

              {pathname === '/apps' && (
                <button
                  type="button"
                  onClick={() => setShowNewAppModal(true)}
                  className="flex items-center gap-2 px-3.5 py-1.5 rounded-lg bg-emerald-500 hover:bg-emerald-600 text-zinc-950 text-sm font-semibold transition"
                >
                  <Plus className="w-4 h-4" />
                  New Application
                </button>
              )}

              {pathname === '/domains' && (
                <button
                  type="button"
                  onClick={() => setShowNewDomainModal(true)}
                  className="flex items-center gap-2 px-3.5 py-1.5 rounded-lg bg-emerald-500 hover:bg-emerald-600 text-zinc-950 text-sm font-semibold transition"
                >
                  <Plus className="w-4 h-4" />
                  Add Domain
                </button>
              )}

              {pathname === '/nodes' && (
                <button
                  type="button"
                  onClick={() => setShowAddNodeModal(true)}
                  className="flex items-center gap-2 px-3.5 py-1.5 rounded-lg bg-emerald-500 hover:bg-emerald-600 text-zinc-950 text-sm font-semibold transition"
                >
                  <Plus className="w-4 h-4" />
                  Add Fleet Node
                </button>
              )}
            </div>
          </header>
        )}

        {/* View Content */}
        <div className={`flex-1 overflow-y-auto ${isAppDetailPage ? 'p-0 flex flex-col' : 'p-8'}`}>
          <Outlet />
        </div>
      </main>

      {/* Global Modals */}
      <NewAppModal />
      <AddNodeModal />
      <UpgradeModal />
      <NewDomainModal />
      <ViewCodeModal />

      {testingApp && (
        <TestAppModal
          app={testingApp}
          onClose={() => setTestingApp(null)}
        />
      )}

      {historyApp && (
        <BuildHistoryModal
          app={historyApp}
          onClose={() => setHistoryApp(null)}
        />
      )}
    </div>
  );
}

export default function App() {
  return (
    <DashboardProvider>
      <DashboardLayout />
    </DashboardProvider>
  );
}
