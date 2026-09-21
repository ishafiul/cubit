import React, { useState, useEffect } from 'react';
import {
  Server,
  Layers,
  Globe,
  RefreshCw,
  Terminal,
  Plus,
  ShieldCheck,
  Radio,
  Play,
  HardDrive,
  Zap,
  GitBranch,
  Code,
  X,
  ExternalLink,
  Copy,
  Check,
  History,
  Send,
  Database,
  Box,
  Cpu,
  Clock,
  GitMerge,
  FileCode,
} from 'lucide-react';

import {
  useListNodes,
} from './api/generated/nodes/nodes';
import {
  useListApplications,
  useCreateApplication,
  useTestApplication,
} from './api/generated/applications/applications';
import {
  useDeployApplication,
  useListDeployments,
} from './api/generated/deployments/deployments';
import {
  useListDomains,
  useCreateDomain,
} from './api/generated/domains/domains';
import {
  useGetRuntimeStatus,
  useUpgradeCelldDaemon,
} from './api/generated/runtime/runtime';
import type { Application } from './api/model';
import { ApplicationDetailPage, getDeploymentStatusBadge } from './components/ApplicationDetailPage';
import { KVView } from './components/services/KVView';
import { D1View } from './components/services/D1View';
import { R2View } from './components/services/R2View';
import { DynamicWorkersView } from './components/services/DynamicWorkersView';
import { CronView } from './components/services/CronView';
import { QueuesView } from './components/services/QueuesView';
import { WorkflowsView } from './components/services/WorkflowsView';
import { DurableObjectsView } from './components/services/DurableObjectsView';
import { ContainersView } from './components/services/ContainersView';
import { StaticAssetsView } from './components/services/StaticAssetsView';

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

export default function App() {
  const [tab, setTab] = useState<ActiveTab>('apps');
  const [currentAppId, setCurrentAppId] = useState<string | null>(null);
  const [selectedServiceId, setSelectedServiceId] = useState<string | undefined>(undefined);

  const handleNavigateToService = (targetTab: ActiveTab, resourceId?: string) => {
    setCurrentAppId(null);
    setSelectedServiceId(resourceId);
    setTab(targetTab);
  };

  // Orval TanStack Query hooks (100% type-safe from OpenAPI)
  const { data: nodesData, refetch: refetchNodes } = useListNodes();
  const { data: appsData, refetch: refetchApps } = useListApplications();
  const { data: domainsData, refetch: refetchDomains } = useListDomains();
  const { data: runtimeData, refetch: refetchRuntime } = useGetRuntimeStatus();

  const nodes = Array.isArray(nodesData) ? nodesData : [];
  const apps = Array.isArray(appsData) ? appsData : [];
  const domains = Array.isArray(domainsData) ? domainsData : [];
  const runtimeStatus = runtimeData;
  const currentApp = apps.find(a => a.id === currentAppId) || null;

  const createApplicationMutation = useCreateApplication();
  const deployApplicationMutation = useDeployApplication();
  const upgradeCelldMutation = useUpgradeCelldDaemon();
  const createDomainMutation = useCreateDomain();

  // Modals state
  const [showNewAppModal, setShowNewAppModal] = useState(false);
  const [showUpgradeModal, setShowUpgradeModal] = useState(false);
  const [showNewDomainModal, setShowNewDomainModal] = useState(false);

  // Form states
  const [appName, setAppName] = useState('');
  const [sourceType, setSourceType] = useState<'inline' | 'git'>('inline');
  const [gitRepo, setGitRepo] = useState('');
  const [gitBranch, setGitBranch] = useState('main');
  const [autoDeploy, setAutoDeploy] = useState(true);
  const [inlineCode, setInlineCode] = useState(
`export default {
  async fetch(request, env, ctx) {
    return new Response("Hello World from Cubit Worker!", {
      headers: { "content-type": "text/plain; charset=utf-8" },
    });
  },
};`
  );
  const [viewingCodeApp, setViewingCodeApp] = useState<any>(null);
  const [testingApp, setTestingApp] = useState<Application | null>(null);
  const [historyApp, setHistoryApp] = useState<Application | null>(null);
  const [selectedLogsAppId, setSelectedLogsAppId] = useState<string>('');
  const [targetVersion, setTargetVersion] = useState('0.3.0');
  const [selectedAppId, setSelectedAppId] = useState('');
  const [domainHost, setDomainHost] = useState('');

  // Live log streaming
  const [activeDeploymentId, setActiveDeploymentId] = useState<string | null>(null);
  const [deployStatus, setDeployStatus] = useState<string>('');
  const [isDeploying, setIsDeploying] = useState<string | null>(null);
  const [logs, setLogs] = useState<Array<{ timestamp: string; step: string; message: string; level: string }>>([]);

  const refreshAll = () => {
    refetchNodes();
    refetchApps();
    refetchDomains();
    refetchRuntime();
  };

  // Poll runtime status every 5 seconds
  useEffect(() => {
    const interval = setInterval(refreshAll, 5000);
    return () => clearInterval(interval);
  }, []);

  // EventSource live log streaming with initial REST fetch fallback
  useEffect(() => {
    if (!activeDeploymentId) return;

    setLogs([]);
    setDeployStatus('building');

    // Immediate REST fetch for existing logs
    fetch(`/api/v1/deployments/${activeDeploymentId}/logs`)
      .then(r => r.json())
      .then((data: any[]) => {
        if (Array.isArray(data) && data.length > 0) {
          const formatted = data.map(entry => ({
            timestamp: entry.timestamp || entry.Timestamp || new Date().toISOString(),
            step: entry.step || entry.Step || 'info',
            message: entry.message || entry.Message || '',
            level: entry.level || entry.Level || 'info',
          }));
          setLogs(formatted);
        }
      })
      .catch(err => console.error("Failed fetching initial logs", err));

    // Fetch deployment details to update status
    fetch(`/api/v1/deployments/${activeDeploymentId}`)
      .then(r => r.json())
      .then((data: any) => {
        if (data && data.status) {
          setDeployStatus(data.status);
        }
      })
      .catch(err => console.error("Failed fetching deployment info", err));

    const es = new EventSource(`/api/v1/deployments/${activeDeploymentId}/logs/stream`);

    es.onmessage = (event) => {
      try {
        const entry = JSON.parse(event.data);
        const normalized = {
          timestamp: entry.timestamp || entry.Timestamp || new Date().toISOString(),
          step: entry.step || entry.Step || 'info',
          message: entry.message || entry.Message || '',
          level: entry.level || entry.Level || 'info',
        };
        setLogs(prev => {
          const exists = prev.some(p => p.step === normalized.step && p.message === normalized.message);
          return exists ? prev : [...prev, normalized];
        });
      } catch (e) {
        console.error("Failed parsing log entry", e);
      }
    };

    es.addEventListener('complete', (event: any) => {
      es.close();
      if (event.data) {
        try {
          const parsed = JSON.parse(event.data);
          if (parsed.status) setDeployStatus(parsed.status);
        } catch (_) {}
      } else {
        setDeployStatus('active');
      }
      refreshAll();
    });

    es.onerror = () => {
      es.close();
    };

    return () => {
      es.close();
    };
  }, [activeDeploymentId]);

  const handleCreateApp = async (e: React.FormEvent) => {
    e.preventDefault();
    const payload = sourceType === 'git'
      ? { name: appName, sourceType: 'git' as const, gitRepo, branch: gitBranch || 'main' }
      : { name: appName, sourceType: 'inline' as const, inlineCode: inlineCode || undefined };

    const res = await createApplicationMutation.mutateAsync({
      data: payload,
    });
    setAppName('');
    setGitRepo('');
    setShowNewAppModal(false);
    refetchApps();

    if (res && res.id) {
      setCurrentAppId(res.id);
      if (autoDeploy) {
        handleDeployApp(res.id);
      }
    }
  };

  const handleDeployApp = async (appId: string) => {
    setIsDeploying(appId);
    try {
      const res = await deployApplicationMutation.mutateAsync({
        id: appId,
        data: { commitHash: 'HEAD' }
      });
      if (res && res.id) {
        setActiveDeploymentId(res.id);
        refetchApps();
      }
    } finally {
      setIsDeploying(null);
    }
  };

  const handleUpgradeCelld = async (e: React.FormEvent) => {
    e.preventDefault();
    await upgradeCelldMutation.mutateAsync({
      data: { targetVersion }
    });
    setShowUpgradeModal(false);
    refetchRuntime();
  };

  const handleCreateDomain = async (e: React.FormEvent) => {
    e.preventDefault();
    await createDomainMutation.mutateAsync({
      data: { applicationId: selectedAppId, hostname: domainHost, pathPrefix: '/' }
    });
    setDomainHost('');
    setShowNewDomainModal(false);
    refetchDomains();
  };

  return (
    <div className="flex h-screen bg-zinc-950 text-zinc-100 antialiased overflow-hidden">
      {/* Sidebar */}
      <aside className="w-64 border-r border-zinc-800/80 bg-zinc-900/40 p-5 flex flex-col justify-between">
        <div>
          <div className="flex items-center gap-3 px-2 py-3 mb-8">
            <div className="w-9 h-9 rounded-xl bg-gradient-to-tr from-emerald-600 to-teal-500 flex items-center justify-center font-bold text-white shadow-lg shadow-emerald-950">
              C
            </div>
            <div>
              <h1 className="font-bold text-lg leading-tight tracking-tight">Cubit</h1>
              <p className="text-xs text-zinc-500">celld Bare-Metal PaaS</p>
            </div>
          </div>

          <nav className="space-y-4 overflow-y-auto max-h-[calc(100vh-250px)] pr-1">
            {/* 1. Compute Section */}
            <div className="space-y-0.5">
              <span className="px-3 text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">
                Compute
              </span>
              <button
                onClick={() => {
                  setTab('apps');
                  setCurrentAppId(null);
                }}
                className={`w-full flex items-center gap-2.5 px-3 py-1.5 rounded-lg text-xs font-medium transition ${
                  tab === 'apps' ? 'bg-zinc-800 text-emerald-400 shadow-sm' : 'text-zinc-400 hover:bg-zinc-900 hover:text-zinc-200'
                }`}
              >
                <Layers className="w-3.5 h-3.5" />
                <span>Workers</span>
                <span className="ml-auto text-[10px] px-1.5 py-0.5 rounded-full bg-zinc-800 text-zinc-400">
                  {apps.length}
                </span>
              </button>

              <button
                onClick={() => setTab('dynamic-workers')}
                className={`w-full flex items-center gap-2.5 px-3 py-1.5 rounded-lg text-xs font-medium transition ${
                  tab === 'dynamic-workers' ? 'bg-zinc-800 text-emerald-400 shadow-sm' : 'text-zinc-400 hover:bg-zinc-900 hover:text-zinc-200'
                }`}
              >
                <Code className="w-3.5 h-3.5" />
                <span>Dynamic Workers</span>
              </button>

              <button
                onClick={() => setTab('durable-objects')}
                className={`w-full flex items-center gap-2.5 px-3 py-1.5 rounded-lg text-xs font-medium transition ${
                  tab === 'durable-objects' ? 'bg-zinc-800 text-emerald-400 shadow-sm' : 'text-zinc-400 hover:bg-zinc-900 hover:text-zinc-200'
                }`}
              >
                <Box className="w-3.5 h-3.5" />
                <span>Durable Objects</span>
              </button>

              <button
                onClick={() => setTab('containers')}
                className={`w-full flex items-center gap-2.5 px-3 py-1.5 rounded-lg text-xs font-medium transition ${
                  tab === 'containers' ? 'bg-zinc-800 text-emerald-400 shadow-sm' : 'text-zinc-400 hover:bg-zinc-900 hover:text-zinc-200'
                }`}
              >
                <Cpu className="w-3.5 h-3.5" />
                <span>Containers</span>
                <span className="ml-auto text-[9px] px-1.5 py-0.2 rounded-full bg-amber-950 text-amber-400 border border-amber-800/80">
                  Exp
                </span>
              </button>
            </div>

            {/* 2. Storage & Databases */}
            <div className="space-y-0.5">
              <span className="px-3 text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">
                Storage & Data
              </span>
              <button
                onClick={() => setTab('kv')}
                className={`w-full flex items-center gap-2.5 px-3 py-1.5 rounded-lg text-xs font-medium transition ${
                  tab === 'kv' ? 'bg-zinc-800 text-emerald-400 shadow-sm' : 'text-zinc-400 hover:bg-zinc-900 hover:text-zinc-200'
                }`}
              >
                <Database className="w-3.5 h-3.5" />
                <span>KV Namespaces</span>
              </button>

              <button
                onClick={() => setTab('d1')}
                className={`w-full flex items-center gap-2.5 px-3 py-1.5 rounded-lg text-xs font-medium transition ${
                  tab === 'd1' ? 'bg-zinc-800 text-emerald-400 shadow-sm' : 'text-zinc-400 hover:bg-zinc-900 hover:text-zinc-200'
                }`}
              >
                <Database className="w-3.5 h-3.5 text-emerald-400" />
                <span>D1 SQL</span>
              </button>

              <button
                onClick={() => setTab('r2')}
                className={`w-full flex items-center gap-2.5 px-3 py-1.5 rounded-lg text-xs font-medium transition ${
                  tab === 'r2' ? 'bg-zinc-800 text-emerald-400 shadow-sm' : 'text-zinc-400 hover:bg-zinc-900 hover:text-zinc-200'
                }`}
              >
                <HardDrive className="w-3.5 h-3.5" />
                <span>R2 Storage</span>
              </button>

              <button
                onClick={() => setTab('static-assets')}
                className={`w-full flex items-center gap-2.5 px-3 py-1.5 rounded-lg text-xs font-medium transition ${
                  tab === 'static-assets' ? 'bg-zinc-800 text-emerald-400 shadow-sm' : 'text-zinc-400 hover:bg-zinc-900 hover:text-zinc-200'
                }`}
              >
                <FileCode className="w-3.5 h-3.5" />
                <span>Static Assets</span>
              </button>
            </div>

            {/* 3. Automation & Messaging */}
            <div className="space-y-0.5">
              <span className="px-3 text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">
                Automation & Events
              </span>
              <button
                onClick={() => setTab('cron')}
                className={`w-full flex items-center gap-2.5 px-3 py-1.5 rounded-lg text-xs font-medium transition ${
                  tab === 'cron' ? 'bg-zinc-800 text-emerald-400 shadow-sm' : 'text-zinc-400 hover:bg-zinc-900 hover:text-zinc-200'
                }`}
              >
                <Clock className="w-3.5 h-3.5" />
                <span>Cron Triggers</span>
              </button>

              <button
                onClick={() => setTab('queues')}
                className={`w-full flex items-center gap-2.5 px-3 py-1.5 rounded-lg text-xs font-medium transition ${
                  tab === 'queues' ? 'bg-zinc-800 text-emerald-400 shadow-sm' : 'text-zinc-400 hover:bg-zinc-900 hover:text-zinc-200'
                }`}
              >
                <Send className="w-3.5 h-3.5" />
                <span>Queues</span>
              </button>

              <button
                onClick={() => setTab('workflows')}
                className={`w-full flex items-center gap-2.5 px-3 py-1.5 rounded-lg text-xs font-medium transition ${
                  tab === 'workflows' ? 'bg-zinc-800 text-emerald-400 shadow-sm' : 'text-zinc-400 hover:bg-zinc-900 hover:text-zinc-200'
                }`}
              >
                <GitMerge className="w-3.5 h-3.5" />
                <span>Workflows</span>
              </button>
            </div>

            {/* 4. Infrastructure & Fleet */}
            <div className="space-y-0.5">
              <span className="px-3 text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">
                Fleet & Routing
              </span>
              <button
                onClick={() => setTab('nodes')}
                className={`w-full flex items-center gap-2.5 px-3 py-1.5 rounded-lg text-xs font-medium transition ${
                  tab === 'nodes' ? 'bg-zinc-800 text-emerald-400 shadow-sm' : 'text-zinc-400 hover:bg-zinc-900 hover:text-zinc-200'
                }`}
              >
                <Server className="w-3.5 h-3.5" />
                <span>Fleet Nodes</span>
                <span className="ml-auto text-[10px] px-1.5 py-0.5 rounded-full bg-zinc-800 text-zinc-400">
                  {nodes.length}
                </span>
              </button>

              <button
                onClick={() => setTab('domains')}
                className={`w-full flex items-center gap-2.5 px-3 py-1.5 rounded-lg text-xs font-medium transition ${
                  tab === 'domains' ? 'bg-zinc-800 text-emerald-400 shadow-sm' : 'text-zinc-400 hover:bg-zinc-900 hover:text-zinc-200'
                }`}
              >
                <Globe className="w-3.5 h-3.5" />
                <span>Traefik Routing</span>
                <span className="ml-auto text-[10px] px-1.5 py-0.5 rounded-full bg-zinc-800 text-zinc-400">
                  {domains.length}
                </span>
              </button>

              <button
                onClick={() => setTab('logs')}
                className={`w-full flex items-center gap-2.5 px-3 py-1.5 rounded-lg text-xs font-medium transition ${
                  tab === 'logs' ? 'bg-zinc-800 text-emerald-400 shadow-sm' : 'text-zinc-400 hover:bg-zinc-900 hover:text-zinc-200'
                }`}
              >
                <Terminal className="w-3.5 h-3.5" />
                <span>Build Logs</span>
              </button>
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
            <span className="text-zinc-200 uppercase font-semibold text-[11px]">{runtimeStatus?.storageBackend || 'Garage S3'}</span>
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
        {!(tab === 'apps' && currentApp) && (
          <header className="h-16 border-b border-zinc-800/80 px-8 flex items-center justify-between bg-zinc-950/60 backdrop-blur">
            <div className="flex items-center gap-4">
              <h2 className="text-xl font-semibold tracking-tight">
                {tab === 'apps' ? 'Workers' :
                 tab === 'dynamic-workers' ? 'Dynamic Workers' :
                 tab === 'durable-objects' ? 'Durable Objects & Facets' :
                 tab === 'containers' ? 'Containers (Experimental)' :
                 tab === 'static-assets' ? 'Static Assets' :
                 tab === 'cron' ? 'Cron Triggers' :
                 tab === 'queues' ? 'Queues' :
                 tab === 'workflows' ? 'Workflows' :
                 tab === 'kv' ? 'KV Namespaces' :
                 tab === 'd1' ? 'D1 SQL Databases' :
                 tab === 'r2' ? 'R2 Object Storage' :
                 tab === 'nodes' ? 'Fleet Nodes' :
                 tab === 'domains' ? 'Traefik Routing' :
                 tab === 'logs' ? 'Build Logs' :
                 tab}
              </h2>
            </div>

            <div className="flex items-center gap-3">
              <button
                onClick={() => setShowUpgradeModal(true)}
                className="flex items-center gap-2 px-3.5 py-1.5 rounded-lg border border-zinc-700 bg-zinc-900 hover:bg-zinc-800 text-sm font-medium transition"
              >
                <RefreshCw className="w-3.5 h-3.5 text-emerald-400" />
                Upgrade celld
              </button>

              {tab === 'apps' && (
                <button
                  onClick={() => setShowNewAppModal(true)}
                  className="flex items-center gap-2 px-3.5 py-1.5 rounded-lg bg-emerald-500 hover:bg-emerald-600 text-zinc-950 text-sm font-semibold transition"
                >
                  <Plus className="w-4 h-4" />
                  New Application
                </button>
              )}

              {tab === 'domains' && (
                <button
                  onClick={() => setShowNewDomainModal(true)}
                  className="flex items-center gap-2 px-3.5 py-1.5 rounded-lg bg-emerald-500 hover:bg-emerald-600 text-zinc-950 text-sm font-semibold transition"
                >
                  <Plus className="w-4 h-4" />
                  Add Domain
                </button>
              )}
            </div>
          </header>
        )}

        {/* View Content */}
        <div className={`flex-1 overflow-y-auto ${tab === 'apps' && currentApp ? 'p-0 flex flex-col' : 'p-8'}`}>
          {tab === 'nodes' && (
            <div className="space-y-6">
              <div className="grid grid-cols-3 gap-5">
                <div className="p-5 rounded-2xl border border-zinc-800/80 bg-zinc-900/30">
                  <div className="flex items-center justify-between text-zinc-400 text-sm mb-2">
                    <span>Fleet Nodes</span>
                    <Server className="w-4 h-4 text-emerald-400" />
                  </div>
                  <div className="text-3xl font-bold">{nodes.length}</div>
                </div>

                <div className="p-5 rounded-2xl border border-zinc-800/80 bg-zinc-900/30">
                  <div className="flex items-center justify-between text-zinc-400 text-sm mb-2">
                    <span>Active Workers</span>
                    <Radio className="w-4 h-4 text-emerald-400" />
                  </div>
                  <div className="text-3xl font-bold">{apps.filter(a => a.status === 'running').length}</div>
                </div>

                <div className="p-5 rounded-2xl border border-zinc-800/80 bg-zinc-900/30">
                  <div className="flex items-center justify-between text-zinc-400 text-sm mb-2">
                    <span>Traefik SSL Routes</span>
                    <ShieldCheck className="w-4 h-4 text-emerald-400" />
                  </div>
                  <div className="text-3xl font-bold">{domains.length}</div>
                </div>
              </div>

              {/* Node Cards */}
              <div className="space-y-3">
                <h3 className="text-sm font-semibold text-zinc-400 uppercase tracking-wider">Bare-Metal Node Instances</h3>
                <div className="grid grid-cols-2 gap-4">
                  {nodes.map(node => (
                    <div key={node.id} className="p-5 rounded-xl border border-zinc-800/80 bg-zinc-900/20 space-y-4 hover:border-zinc-700 transition">
                      <div className="flex items-start justify-between">
                        <div>
                          <div className="flex items-center gap-2">
                            <h4 className="font-semibold text-base">{node.name}</h4>
                            <span className={`text-[10px] px-2 py-0.5 rounded-full font-bold uppercase ${
                              node.status === 'active' ? 'bg-emerald-950 text-emerald-400 border border-emerald-800' : 'bg-amber-950 text-amber-400 border border-amber-800'
                            }`}>
                              {node.status}
                            </span>
                          </div>
                          <p className="text-xs text-zinc-500 font-mono mt-0.5">{node.ipAddress}</p>
                        </div>
                        <span className="text-xs font-mono text-zinc-400 bg-zinc-800 px-2 py-1 rounded">
                          celld v{node.celldVersion}
                        </span>
                      </div>

                      <div className="grid grid-cols-2 gap-2 text-xs text-zinc-400">
                        <div className="p-2.5 rounded bg-zinc-950/40 border border-zinc-900 flex items-center gap-2">
                          <Radio className="w-3.5 h-3.5 text-zinc-500" />
                          <span>Worker Port: {node.workerPort}</span>
                        </div>
                        <div className="p-2.5 rounded bg-zinc-950/40 border border-zinc-900 flex items-center gap-2">
                          <HardDrive className="w-3.5 h-3.5 text-zinc-500" />
                          <span>Peer Port: {node.internalPort}</span>
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            </div>
          )}

          {tab === 'apps' && currentApp && (
            <ApplicationDetailPage
              app={currentApp}
              onBack={() => setCurrentAppId(null)}
              onDeploy={handleDeployApp}
              isDeploying={isDeploying === currentApp.id}
              onTestApp={setTestingApp}
              onRefreshApps={refetchApps}
              onNavigateToService={handleNavigateToService}
            />
          )}

          {tab === 'apps' && !currentApp && (
            <div className="space-y-4">
              <div className="grid grid-cols-1 gap-4">
                {apps.map(app => (
                  <ApplicationCard
                    key={app.id}
                    app={app}
                    onOpenApp={() => setCurrentAppId(app.id)}
                    onDeploy={handleDeployApp}
                    isDeploying={isDeploying === app.id}
                    onViewCode={setViewingCodeApp}
                    onTestApp={setTestingApp}
                    onViewHistory={setHistoryApp}
                  />
                ))}
              </div>
            </div>
          )}

          {tab === 'domains' && (
            <div className="space-y-4">
              <div className="grid grid-cols-1 gap-3">
                {domains.map(dom => (
                  <div key={dom.id} className="p-5 rounded-xl border border-zinc-800/80 bg-zinc-900/20 flex items-center justify-between hover:border-zinc-700 transition">
                    <div className="space-y-1">
                      <div className="flex items-center gap-3">
                        <h4 className="font-mono text-base text-emerald-400 font-semibold">{dom.hostname}</h4>
                        <span className="text-xs bg-zinc-800 px-2 py-0.5 rounded text-zinc-400">{dom.pathPrefix}</span>
                      </div>
                      <p className="text-xs text-zinc-500">Traefik Ingress Route</p>
                    </div>

                    <div className="flex items-center gap-2 text-xs text-emerald-400 bg-emerald-950/40 border border-emerald-900 px-3 py-1 rounded-full">
                      <ShieldCheck className="w-4 h-4" />
                      Let's Encrypt Active
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}

          {tab === 'logs' && (
            <div className="h-full flex flex-col space-y-4 min-h-0">
              <div className="flex items-center justify-between flex-wrap gap-3">
                <div className="flex items-center gap-3">
                  <span className="text-sm font-semibold text-zinc-400 uppercase tracking-wider">
                    {selectedLogsAppId ? 'Project Build Logs' : 'Live Build & Deployment Stream'}
                  </span>
                  {!selectedLogsAppId && deployStatus && (
                    <span className={`text-[11px] font-mono px-2.5 py-0.5 rounded-full border uppercase font-bold flex items-center gap-1.5 ${
                      deployStatus === 'active'
                        ? 'bg-emerald-950/60 border-emerald-700/80 text-emerald-400'
                        : deployStatus === 'failed'
                        ? 'bg-rose-950/60 border-rose-700/80 text-rose-400'
                        : 'bg-amber-950/60 border-amber-700/80 text-amber-400'
                    }`}>
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
                      onChange={e => setSelectedLogsAppId(e.target.value)}
                      className="bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-1.5 text-xs text-zinc-200 focus:outline-none focus:border-emerald-500 font-mono"
                    >
                      <option value="">-- Active Live Stream --</option>
                      {apps.map(a => (
                        <option key={a.id} value={a.id}>{a.name} ({a.subdomain || a.name})</option>
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
                            <span className={l.level === 'error' ? 'text-red-400 font-semibold' : 'text-zinc-200'}>{l.message}</span>
                          </div>
                        );
                      })
                    )}
                  </div>
                </div>
              )}
            </div>
          )}

          {tab === 'dynamic-workers' && <DynamicWorkersView />}
          {tab === 'durable-objects' && <DurableObjectsView initialSelectedId={selectedServiceId} />}
          {tab === 'containers' && <ContainersView />}
          {tab === 'kv' && <KVView initialSelectedId={selectedServiceId} />}
          {tab === 'd1' && <D1View initialSelectedId={selectedServiceId} />}
          {tab === 'r2' && <R2View initialSelectedId={selectedServiceId} />}
          {tab === 'static-assets' && <StaticAssetsView />}
          {tab === 'cron' && <CronView initialSelectedId={selectedServiceId} />}
          {tab === 'queues' && <QueuesView initialSelectedId={selectedServiceId} />}
          {tab === 'workflows' && <WorkflowsView initialSelectedId={selectedServiceId} />}
        </div>
      </main>

      {/* New Application Modal */}
      {showNewAppModal && (
        <div className="fixed inset-0 bg-black/60 backdrop-blur-sm flex items-center justify-center p-4 z-50">
          <div className="bg-zinc-900 border border-zinc-800 rounded-2xl max-w-lg w-full p-6 space-y-5 shadow-2xl">
            <div className="flex items-center justify-between">
              <h3 className="text-lg font-bold">New Cloudflare Worker</h3>
              <button
                type="button"
                onClick={() => setShowNewAppModal(false)}
                className="text-zinc-500 hover:text-zinc-300 transition"
              >
                <X className="w-5 h-5" />
              </button>
            </div>

            {/* Source Type Selector */}
            <div className="flex rounded-lg bg-zinc-950 p-1 border border-zinc-800">
              <button
                type="button"
                onClick={() => setSourceType('inline')}
                className={`flex-1 py-2 px-3 rounded-md text-xs font-semibold flex items-center justify-center gap-2 transition ${
                  sourceType === 'inline' ? 'bg-zinc-800 text-emerald-400 shadow' : 'text-zinc-400 hover:text-zinc-200'
                }`}
              >
                <Zap className="w-3.5 h-3.5 text-emerald-400" />
                Hello World / Inline Code
              </button>
              <button
                type="button"
                onClick={() => setSourceType('git')}
                className={`flex-1 py-2 px-3 rounded-md text-xs font-semibold flex items-center justify-center gap-2 transition ${
                  sourceType === 'git' ? 'bg-zinc-800 text-purple-400 shadow' : 'text-zinc-400 hover:text-zinc-200'
                }`}
              >
                <GitBranch className="w-3.5 h-3.5 text-purple-400" />
                Git Repository
              </button>
            </div>

            <form onSubmit={handleCreateApp} className="space-y-4">
              <div>
                <label className="text-xs font-medium text-zinc-400 mb-1 block">Application Name</label>
                <input
                  type="text"
                  required
                  placeholder="my-worker-api"
                  value={appName}
                  onChange={e => setAppName(e.target.value)}
                  className="w-full bg-zinc-950 border border-zinc-800 rounded-lg px-3 py-2 text-sm text-zinc-100 focus:outline-none focus:border-emerald-500"
                />
              </div>

              {sourceType === 'inline' ? (
                <>
                  <div>
                    <div className="flex items-center justify-between mb-1">
                      <label className="text-xs font-medium text-zinc-400">Worker Source Code (ES Module)</label>
                      <span className="text-[10px] text-zinc-500 font-mono">Cloudflare Worker API</span>
                    </div>
                    <textarea
                      rows={6}
                      value={inlineCode}
                      onChange={e => setInlineCode(e.target.value)}
                      className="w-full bg-zinc-950 border border-zinc-800 rounded-lg p-3 text-xs font-mono text-zinc-200 focus:outline-none focus:border-emerald-500 resize-none leading-relaxed"
                    />
                  </div>

                  <label className="flex items-center gap-2.5 text-xs text-zinc-400 cursor-pointer select-none">
                    <input
                      type="checkbox"
                      checked={autoDeploy}
                      onChange={e => setAutoDeploy(e.target.checked)}
                      className="rounded bg-zinc-950 border-zinc-800 text-emerald-500 focus:ring-0 focus:ring-offset-0"
                    />
                    <span>Deploy immediately after creation</span>
                  </label>
                </>
              ) : (
                <>
                  <div>
                    <label className="text-xs font-medium text-zinc-400 mb-1 block">Git Repository URL</label>
                    <input
                      type="text"
                      required
                      placeholder="https://github.com/myorg/worker"
                      value={gitRepo}
                      onChange={e => setGitRepo(e.target.value)}
                      className="w-full bg-zinc-950 border border-zinc-800 rounded-lg px-3 py-2 text-sm text-zinc-100 focus:outline-none focus:border-emerald-500 font-mono"
                    />
                  </div>

                  <div>
                    <label className="text-xs font-medium text-zinc-400 mb-1 block">Branch</label>
                    <input
                      type="text"
                      placeholder="main"
                      value={gitBranch}
                      onChange={e => setGitBranch(e.target.value)}
                      className="w-full bg-zinc-950 border border-zinc-800 rounded-lg px-3 py-2 text-sm text-zinc-100 focus:outline-none focus:border-emerald-500 font-mono"
                    />
                  </div>
                </>
              )}

              <div className="flex justify-end gap-3 pt-2">
                <button
                  type="button"
                  onClick={() => setShowNewAppModal(false)}
                  className="px-4 py-2 rounded-lg border border-zinc-800 text-sm font-medium hover:bg-zinc-800 transition"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  className="px-4 py-2 rounded-lg bg-emerald-500 hover:bg-emerald-600 text-zinc-950 text-sm font-semibold transition"
                >
                  {sourceType === 'inline' && autoDeploy ? 'Create & Deploy' : 'Create Application'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* View Code Modal */}
      {viewingCodeApp && (
        <div className="fixed inset-0 bg-black/60 backdrop-blur-sm flex items-center justify-center p-4 z-50">
          <div className="bg-zinc-900 border border-zinc-800 rounded-2xl max-w-lg w-full p-6 space-y-4 shadow-2xl">
            <div className="flex items-center justify-between">
              <div>
                <h3 className="text-lg font-bold">{viewingCodeApp.name}</h3>
                <p className="text-xs text-zinc-400">Inline Worker Script</p>
              </div>
              <button
                type="button"
                onClick={() => setViewingCodeApp(null)}
                className="text-zinc-500 hover:text-zinc-300 transition"
              >
                <X className="w-5 h-5" />
              </button>
            </div>

            <pre className="p-4 rounded-xl bg-zinc-950 border border-zinc-800/80 text-xs font-mono text-zinc-200 overflow-x-auto max-h-80 leading-relaxed">
              <code>{viewingCodeApp.inlineCode || '// No inline code found'}</code>
            </pre>

            <div className="flex justify-end">
              <button
                type="button"
                onClick={() => setViewingCodeApp(null)}
                className="px-4 py-2 rounded-lg border border-zinc-800 text-sm font-medium hover:bg-zinc-800 transition"
              >
                Close
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Upgrade celld Modal */}
      {showUpgradeModal && (
        <div className="fixed inset-0 bg-black/60 backdrop-blur-sm flex items-center justify-center p-4 z-50">
          <div className="bg-zinc-900 border border-zinc-800 rounded-2xl max-w-md w-full p-6 space-y-5 shadow-2xl">
            <div>
              <h3 className="text-lg font-bold">Rolling Upgrade celld Daemon</h3>
              <p className="text-xs text-zinc-400 mt-1">
                Upgrades the celld runtime container one node at a time with zero dropped requests.
              </p>
            </div>

            <form onSubmit={handleUpgradeCelld} className="space-y-4">
              <div>
                <label className="text-xs font-medium text-zinc-400 mb-1 block">Target Version</label>
                <input
                  type="text"
                  required
                  placeholder="0.3.0"
                  value={targetVersion}
                  onChange={e => setTargetVersion(e.target.value)}
                  className="w-full bg-zinc-950 border border-zinc-800 rounded-lg px-3 py-2 text-sm text-zinc-100 focus:outline-none focus:border-emerald-500 font-mono"
                />
              </div>

              <div className="flex justify-end gap-3 pt-2">
                <button
                  type="button"
                  onClick={() => setShowUpgradeModal(false)}
                  className="px-4 py-2 rounded-lg border border-zinc-800 text-sm font-medium hover:bg-zinc-800 transition"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  className="px-4 py-2 rounded-lg bg-emerald-500 hover:bg-emerald-600 text-zinc-950 text-sm font-semibold transition"
                >
                  Start Rolling Upgrade
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* New Domain Modal */}
      {showNewDomainModal && (
        <div className="fixed inset-0 bg-black/60 backdrop-blur-sm flex items-center justify-center p-4 z-50">
          <div className="bg-zinc-900 border border-zinc-800 rounded-2xl max-w-md w-full p-6 space-y-5 shadow-2xl">
            <h3 className="text-lg font-bold">Add Custom Domain</h3>
            <form onSubmit={handleCreateDomain} className="space-y-4">
              <div>
                <label className="text-xs font-medium text-zinc-400 mb-1 block">Target Application</label>
                <select
                  required
                  value={selectedAppId}
                  onChange={e => setSelectedAppId(e.target.value)}
                  className="w-full bg-zinc-950 border border-zinc-800 rounded-lg px-3 py-2 text-sm text-zinc-100 focus:outline-none focus:border-emerald-500"
                >
                  <option value="">Select an application...</option>
                  {apps.map(a => (
                    <option key={a.id} value={a.id}>{a.name}</option>
                  ))}
                </select>
              </div>

              <div>
                <label className="text-xs font-medium text-zinc-400 mb-1 block">Hostname</label>
                <input
                  type="text"
                  required
                  placeholder="api.example.com"
                  value={domainHost}
                  onChange={e => setDomainHost(e.target.value)}
                  className="w-full bg-zinc-950 border border-zinc-800 rounded-lg px-3 py-2 text-sm text-zinc-100 focus:outline-none focus:border-emerald-500 font-mono"
                />
              </div>

              <div className="flex justify-end gap-3 pt-2">
                <button
                  type="button"
                  onClick={() => setShowNewDomainModal(false)}
                  className="px-4 py-2 rounded-lg border border-zinc-800 text-sm font-medium hover:bg-zinc-800 transition"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  className="px-4 py-2 rounded-lg bg-emerald-500 hover:bg-emerald-600 text-zinc-950 text-sm font-semibold transition"
                >
                  Create Domain Route
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Test Application Modal */}
      {testingApp && (
        <TestAppModal
          app={testingApp}
          onClose={() => setTestingApp(null)}
        />
      )}

      {/* Build History & Logs Modal */}
      {historyApp && (
        <BuildHistoryModal
          app={historyApp}
          onClose={() => setHistoryApp(null)}
        />
      )}
    </div>
  );
}

// -----------------------------------------------------------------------------
// Subcomponents
// -----------------------------------------------------------------------------

interface ApplicationCardProps {
  app: Application;
  onOpenApp: (app: Application) => void;
  onDeploy: (id: string) => void;
  isDeploying: boolean;
  onViewCode: (app: Application) => void;
  onTestApp: (app: Application) => void;
  onViewHistory: (app: Application) => void;
}

function ApplicationCard({
  app,
  onOpenApp,
  onDeploy,
  isDeploying,
  onViewCode,
  onTestApp,
  onViewHistory,
}: ApplicationCardProps) {
  const { data: deploymentsData } = useListDeployments(app.id);
  const deployments = Array.isArray(deploymentsData) ? deploymentsData : [];
  const activeDep = deployments.find(d => d.id === app.activeDeploymentId) || (deployments.length > 0 ? deployments[0] : null);
  const buildVersion = activeDep?.buildVersion;
  const subdomain = app.subdomain || app.name;
  const testUrl = app.testUrl || `http://${subdomain}.localhost:8000`;
  const [copied, setCopied] = useState(false);

  const copyUrl = (e: React.MouseEvent) => {
    e.stopPropagation();
    navigator.clipboard.writeText(testUrl);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div className="p-5 rounded-xl border border-zinc-800/80 bg-zinc-900/20 hover:border-zinc-700 transition space-y-3">
      <div className="flex items-start justify-between">
        <div className="space-y-1.5">
          <div className="flex items-center gap-2.5 flex-wrap">
            <h4
              onClick={() => onOpenApp(app)}
              className="font-semibold text-base hover:text-emerald-400 cursor-pointer transition"
            >
              {app.name}
            </h4>
            <span className={`text-[10px] px-2 py-0.5 rounded-full font-bold uppercase ${
              app.status === 'running' ? 'bg-emerald-950 text-emerald-400 border border-emerald-800' : 'bg-zinc-800 text-zinc-400'
            }`}>
              {app.status}
            </span>

            {/* Build Version Badge */}
            {buildVersion ? (
              <span className="text-[10px] px-2.5 py-0.5 rounded-full font-bold bg-emerald-950/80 text-emerald-300 border border-emerald-800/80">
                v{buildVersion}
              </span>
            ) : (
              <span className="text-[10px] px-2 py-0.5 rounded-full font-medium bg-zinc-800/80 text-zinc-500 border border-zinc-700/50">
                v0 (Draft)
              </span>
            )}

            {app.sourceType === 'inline' ? (
              <span className="text-[10px] px-2 py-0.5 rounded-full font-semibold bg-sky-950/80 text-sky-400 border border-sky-800/60 flex items-center gap-1">
                <Zap className="w-2.5 h-2.5" /> Inline Code
              </span>
            ) : (
              <span className="text-[10px] px-2 py-0.5 rounded-full font-semibold bg-purple-950/80 text-purple-400 border border-purple-800/60 flex items-center gap-1">
                <GitBranch className="w-2.5 h-2.5" /> Git
              </span>
            )}
          </div>

          {app.sourceType === 'inline' ? (
            <p className="text-xs text-zinc-500 font-mono">Standalone Cloudflare Worker template</p>
          ) : (
            <p className="text-xs text-zinc-400 font-mono">{app.gitRepo} ({app.branch || 'main'})</p>
          )}

          {/* Subdomain / Test URL Badge */}
          <div className="flex items-center gap-2 pt-1">
            <span className="text-xs font-mono text-emerald-400 bg-emerald-950/30 border border-emerald-900/60 px-2.5 py-1 rounded-md flex items-center gap-1.5">
              <Globe className="w-3.5 h-3.5 text-emerald-400" />
              {testUrl}
            </span>
            <button
              type="button"
              onClick={copyUrl}
              title="Copy URL"
              className="p-1.5 rounded-md hover:bg-zinc-800 text-zinc-400 hover:text-zinc-200 transition"
            >
              {copied ? <Check className="w-3.5 h-3.5 text-emerald-400" /> : <Copy className="w-3.5 h-3.5" />}
            </button>
            <a
              href={testUrl}
              target="_blank"
              rel="noopener noreferrer"
              title="Open in new browser tab"
              className="p-1.5 rounded-md hover:bg-zinc-800 text-zinc-400 hover:text-zinc-200 transition"
            >
              <ExternalLink className="w-3.5 h-3.5" />
            </a>
          </div>
        </div>

        {/* Action buttons */}
        <div className="flex items-center gap-2 flex-wrap justify-end">
          <button
            type="button"
            onClick={() => onOpenApp(app)}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg border border-zinc-700 bg-zinc-800/80 hover:bg-zinc-700 text-zinc-200 text-xs font-medium transition"
          >
            Manage &rarr;
          </button>
          {app.sourceType === 'inline' && (
            <button
              type="button"
              onClick={() => onViewCode(app)}
              className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg border border-zinc-800 bg-zinc-900 hover:bg-zinc-800 text-zinc-300 text-xs font-medium transition"
            >
              <Code className="w-3.5 h-3.5 text-zinc-400" />
              Code
            </button>
          )}

          <button
            type="button"
            onClick={() => onViewHistory(app)}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg border border-zinc-800 bg-zinc-900 hover:bg-zinc-800 text-zinc-300 text-xs font-medium transition"
          >
            <History className="w-3.5 h-3.5 text-zinc-400" />
            Build Logs ({deployments.length})
          </button>

          <button
            type="button"
            onClick={() => onTestApp(app)}
            disabled={!app.activeDeploymentId && deployments.length === 0}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg border border-emerald-900/60 bg-emerald-950/30 hover:bg-emerald-950/60 disabled:opacity-40 text-emerald-400 text-xs font-semibold transition"
          >
            <Send className="w-3.5 h-3.5" />
            Test App
          </button>

          <button
            onClick={() => onDeploy(app.id)}
            disabled={isDeploying}
            className="flex items-center gap-2 px-3.5 py-1.5 rounded-lg bg-emerald-500 hover:bg-emerald-600 disabled:opacity-50 text-zinc-950 text-xs font-semibold transition"
          >
            {isDeploying ? (
              <RefreshCw className="w-3.5 h-3.5 animate-spin" />
            ) : (
              <Play className="w-3.5 h-3.5 fill-current" />
            )}
            {isDeploying ? 'Deploying...' : 'Deploy'}
          </button>
        </div>
      </div>
    </div>
  );
}

interface TestAppModalProps {
  app: Application;
  onClose: () => void;
}

function TestAppModal({ app, onClose }: TestAppModalProps) {
  const [method, setMethod] = useState<'GET' | 'POST' | 'PUT' | 'DELETE'>('GET');
  const [path, setPath] = useState('/');
  const [body, setBody] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [testResult, setTestResult] = useState<{
    statusCode: number;
    headers?: Record<string, string>;
    body: string;
    durationMs?: number;
  } | null>(null);
  const [copiedCurl, setCopiedCurl] = useState(false);

  const testAppMutation = useTestApplication();
  const subdomain = app.subdomain || app.name;
  const testUrl = app.testUrl || `http://${subdomain}.localhost:8000`;
  const curlCommand = `curl -i -X ${method} -H "Host: ${subdomain}.localhost:8000" http://localhost:8000${path}${
    method !== 'GET' && body ? ` -d '${body}'` : ''
  }`;

  const handleRunTest = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsLoading(true);
    const start = performance.now();
    try {
      const res = await testAppMutation.mutateAsync({
        id: app.id,
        data: {
          method,
          path: path.startsWith('/') ? path : `/${path}`,
          body: method !== 'GET' && body ? body : undefined,
        },
      });
      const duration = Math.round(performance.now() - start);
      setTestResult({
        statusCode: res.statusCode,
        headers: res.headers as Record<string, string> | undefined,
        body: res.body,
        durationMs: duration,
      });
    } catch (err: any) {
      setTestResult({
        statusCode: 500,
        body: err?.message || 'Failed to invoke test runner',
        durationMs: Math.round(performance.now() - start),
      });
    } finally {
      setIsLoading(false);
    }
  };

  const copyCurl = () => {
    navigator.clipboard.writeText(curlCommand);
    setCopiedCurl(true);
    setTimeout(() => setCopiedCurl(false), 2000);
  };

  return (
    <div className="fixed inset-0 bg-black/60 backdrop-blur-sm flex items-center justify-center p-4 z-50">
      <div className="bg-zinc-900 border border-zinc-800 rounded-2xl max-w-2xl w-full p-6 space-y-5 shadow-2xl overflow-hidden flex flex-col max-h-[90vh]">
        {/* Header */}
        <div className="flex items-center justify-between border-b border-zinc-800/80 pb-4">
          <div>
            <div className="flex items-center gap-2.5">
              <h3 className="text-lg font-bold">Test Application</h3>
              <span className="text-xs px-2.5 py-0.5 rounded-full font-mono bg-emerald-950/80 text-emerald-400 border border-emerald-800/80">
                {app.name}
              </span>
            </div>
            <p className="text-xs text-zinc-400 mt-0.5">
              Live HTTP testing via subdomain route{' '}
              <code className="text-emerald-400 font-mono">{testUrl}</code>
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
        <div className="space-y-4 overflow-y-auto pr-1">
          <form onSubmit={handleRunTest} className="space-y-3">
            <div className="flex gap-2">
              <select
                value={method}
                onChange={e => setMethod(e.target.value as any)}
                className="bg-zinc-950 border border-zinc-800 rounded-lg px-3 py-2 text-xs font-semibold text-emerald-400 focus:outline-none focus:border-emerald-500"
              >
                <option value="GET">GET</option>
                <option value="POST">POST</option>
                <option value="PUT">PUT</option>
                <option value="DELETE">DELETE</option>
              </select>
              <input
                type="text"
                value={path}
                onChange={e => setPath(e.target.value)}
                placeholder="/"
                className="flex-1 bg-zinc-950 border border-zinc-800 rounded-lg px-3 py-2 text-xs font-mono text-zinc-100 focus:outline-none focus:border-emerald-500"
              />
              <button
                type="submit"
                disabled={isLoading}
                className="px-4 py-2 rounded-lg bg-emerald-500 hover:bg-emerald-600 disabled:opacity-50 text-zinc-950 text-xs font-semibold transition flex items-center gap-1.5"
              >
                {isLoading ? <RefreshCw className="w-3.5 h-3.5 animate-spin" /> : <Play className="w-3.5 h-3.5 fill-current" />}
                Send Request
              </button>
            </div>

            {(method === 'POST' || method === 'PUT') && (
              <div>
                <label className="text-[11px] font-medium text-zinc-400 mb-1 block">Request Body (JSON / Raw)</label>
                <textarea
                  rows={3}
                  value={body}
                  onChange={e => setBody(e.target.value)}
                  placeholder='{"key": "value"}'
                  className="w-full bg-zinc-950 border border-zinc-800 rounded-lg p-2.5 text-xs font-mono text-zinc-200 focus:outline-none focus:border-emerald-500 resize-none leading-relaxed"
                />
              </div>
            )}
          </form>

          {/* Test Response */}
          {testResult && (
            <div className="space-y-2 border-t border-zinc-800/80 pt-3">
              <div className="flex items-center justify-between">
                <span className="text-xs font-semibold uppercase tracking-wider text-zinc-400">Response</span>
                <div className="flex items-center gap-2">
                  {testResult.durationMs !== undefined && (
                    <span className="text-[11px] font-mono text-zinc-500">{testResult.durationMs}ms</span>
                  )}
                  <span className={`text-[11px] font-mono font-bold px-2 py-0.5 rounded ${
                    testResult.statusCode >= 200 && testResult.statusCode < 300
                      ? 'bg-emerald-950 text-emerald-400 border border-emerald-800'
                      : 'bg-rose-950 text-rose-400 border border-rose-800'
                  }`}>
                    {testResult.statusCode}
                  </span>
                </div>
              </div>

              <div className="bg-black/90 border border-zinc-800/80 rounded-xl p-3.5 space-y-2">
                <pre className="text-xs font-mono text-zinc-200 whitespace-pre-wrap break-all overflow-y-auto max-h-48 leading-relaxed">
                  {testResult.body || '<Empty Body>'}
                </pre>
              </div>
            </div>
          )}

          {/* cURL command snippet */}
          <div className="space-y-1.5 border-t border-zinc-800/80 pt-3">
            <div className="flex items-center justify-between">
              <span className="text-[11px] text-zinc-400 font-medium">Terminal cURL command</span>
              <button
                type="button"
                onClick={copyCurl}
                className="text-[11px] text-emerald-400 hover:text-emerald-300 flex items-center gap-1 transition"
              >
                {copiedCurl ? <Check className="w-3 h-3" /> : <Copy className="w-3 h-3" />}
                {copiedCurl ? 'Copied' : 'Copy cURL'}
              </button>
            </div>
            <pre className="bg-zinc-950 border border-zinc-800 rounded-lg p-2.5 text-[11px] font-mono text-zinc-400 overflow-x-auto select-all">
              {curlCommand}
            </pre>
          </div>
        </div>

        {/* Footer */}
        <div className="flex items-center justify-between border-t border-zinc-800/80 pt-3">
          <a
            href={`${testUrl}${path}`}
            target="_blank"
            rel="noopener noreferrer"
            className="text-xs text-zinc-400 hover:text-zinc-200 flex items-center gap-1.5 transition"
          >
            <ExternalLink className="w-3.5 h-3.5" />
            Open in Browser ({testUrl})
          </a>
          <button
            type="button"
            onClick={onClose}
            className="px-4 py-2 rounded-lg border border-zinc-800 text-sm font-medium hover:bg-zinc-800 transition"
          >
            Close
          </button>
        </div>
      </div>
    </div>
  );
}

interface BuildHistoryModalProps {
  app: Application;
  onClose: () => void;
}

function BuildHistoryModal({ app, onClose }: BuildHistoryModalProps) {
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

        <div className="grid grid-cols-12 gap-4 flex-1 overflow-hidden min-h-[350px]">
          {/* Deployments List (Left Column) */}
          <div className="col-span-5 border-r border-zinc-800/80 pr-3 space-y-2 overflow-y-auto max-h-[500px]">
            <div className="text-xs font-semibold uppercase tracking-wider text-zinc-400 mb-2">
              Build Versions ({deployments.length})
            </div>
            {deployments.length === 0 ? (
              <div className="text-xs text-zinc-500 italic p-3">No builds found for this application yet.</div>
            ) : (
              deployments.map(dep => {
                const isSelected = selectedDepId === dep.id;
                const d = new Date(dep.createdAt);
                const dateStr = isNaN(d.getTime()) ? '' : d.toLocaleString();
                return (
                  <div
                    key={dep.id}
                    onClick={() => setSelectedDepId(dep.id)}
                    className={`p-3 rounded-xl border cursor-pointer transition space-y-1 ${
                      isSelected
                        ? 'bg-zinc-800 border-emerald-500/80 text-zinc-100 shadow-sm'
                        : 'bg-zinc-950/60 border-zinc-800/80 hover:border-zinc-700 text-zinc-300'
                    }`}
                  >
                    <div className="flex items-center justify-between">
                      <span className="font-bold text-xs text-emerald-400 font-mono">
                        v{dep.buildVersion || 1}
                      </span>
                      <span className={`text-[10px] px-2 py-0.5 rounded-full font-bold uppercase ${getDeploymentStatusBadge(dep.status)}`}>
                        {dep.status}
                      </span>
                    </div>
                    <div className="text-[11px] text-zinc-400 truncate font-mono">
                      {dep.commitHash ? dep.commitHash.slice(0, 7) : 'HEAD'} - {dep.commitMessage || 'Manual deployment'}
                    </div>
                    <div className="text-[10px] text-zinc-500">{dateStr}</div>
                  </div>
                );
              })
            )}
          </div>

          {/* Logs View (Right Column) */}
          <div className="col-span-7 flex flex-col space-y-2 overflow-hidden">
            <div className="flex items-center justify-between text-xs text-zinc-400">
              <span className="font-semibold uppercase tracking-wider">Build Output</span>
              {selectedDepId && (
                <span className="font-mono text-[10px] text-zinc-500">ID: {selectedDepId.slice(0, 8)}</span>
              )}
            </div>

            <div className="flex-1 bg-black/90 rounded-xl p-4 font-mono text-xs text-zinc-300 overflow-y-auto border border-zinc-900 space-y-1.5 shadow-inner">
              {isLoadingLogs ? (
                <div className="flex items-center gap-2 text-zinc-500 py-4">
                  <RefreshCw className="w-3.5 h-3.5 animate-spin text-emerald-400" />
                  <span>Loading build logs...</span>
                </div>
              ) : depLogs.length === 0 ? (
                <div className="text-zinc-600 italic">No log entries available for this build.</div>
              ) : (
                depLogs.map((l, idx) => {
                  const timeStr = (() => {
                    try {
                      const d = new Date(l.timestamp);
                      return isNaN(d.getTime()) ? new Date().toLocaleTimeString() : d.toLocaleTimeString();
                    } catch (_) {
                      return new Date().toLocaleTimeString();
                    }
                  })();
                  return (
                    <div key={idx} className="flex items-start gap-2 text-[11px] leading-relaxed">
                      <span className="text-zinc-600 shrink-0">{timeStr}</span>
                      <span className="text-emerald-500 font-bold uppercase text-[9px] shrink-0">[{l.step}]</span>
                      <span className={l.level === 'error' ? 'text-red-400 font-semibold' : 'text-zinc-300'}>
                        {l.message}
                      </span>
                    </div>
                  );
                })
              )}
            </div>
          </div>
        </div>

        {/* Footer */}
        <div className="flex justify-end border-t border-zinc-800/80 pt-3">
          <button
            type="button"
            onClick={onClose}
            className="px-4 py-2 rounded-lg border border-zinc-800 text-sm font-medium hover:bg-zinc-800 transition"
          >
            Close
          </button>
        </div>
      </div>
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
          setLogs(data.map(entry => ({
            timestamp: entry.timestamp || entry.Timestamp || new Date().toISOString(),
            step: entry.step || entry.Step || 'info',
            message: entry.message || entry.Message || '',
            level: entry.level || entry.Level || 'info',
          })));
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
                    <span className="font-bold text-xs text-emerald-400 font-mono">
                      v{dep.buildVersion || 1}
                    </span>
                    <span className={`text-[10px] px-2 py-0.5 rounded-full font-bold uppercase ${getDeploymentStatusBadge(dep.status)}`}>
                      {dep.status}
                    </span>
                  </div>
                  <div className="text-[11px] text-zinc-400 truncate font-mono">
                    {dep.commitHash ? dep.commitHash.slice(0, 7) : 'HEAD'} - {dep.commitMessage || 'Manual deployment'}
                  </div>
                  <div className="text-[10px] text-zinc-500">{dateStr}</div>
                </div>
              );
            })
          )}
        </div>

        {/* Build Logs console */}
        <div className="col-span-8 flex flex-col space-y-2 min-h-0">
          <div className="flex items-center justify-between text-xs text-zinc-400">
            <span className="font-semibold uppercase tracking-wider">
              {selectedDepId ? `Log Console (Deployment ID: ${selectedDepId.slice(0, 8)})` : 'Log Console'}
            </span>
          </div>

          <div className="flex-1 bg-black/90 rounded-xl p-5 font-mono text-xs text-zinc-300 overflow-y-auto border border-zinc-900 space-y-1.5 shadow-inner">
            {isLoading ? (
              <div className="flex items-center gap-2 text-zinc-500">
                <RefreshCw className="w-3.5 h-3.5 animate-spin text-emerald-400" />
                <span>Loading build logs...</span>
              </div>
            ) : logs.length === 0 ? (
              <div className="text-zinc-600 italic">No log entries found for this build.</div>
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
      </div>
    </div>
  );
}
