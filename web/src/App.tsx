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
} from 'lucide-react';

import {
  useListNodes,
} from './api/generated/nodes/nodes';
import {
  useListApplications,
  useCreateApplication,
} from './api/generated/applications/applications';
import {
  useDeployApplication,
} from './api/generated/deployments/deployments';
import {
  useListDomains,
  useCreateDomain,
} from './api/generated/domains/domains';
import {
  useGetRuntimeStatus,
  useUpgradeCelldDaemon,
} from './api/generated/runtime/runtime';

export default function App() {
  const [tab, setTab] = useState<'nodes' | 'apps' | 'domains' | 'logs'>('nodes');

  // Orval TanStack Query hooks (100% type-safe from OpenAPI)
  const { data: nodesData, refetch: refetchNodes } = useListNodes();
  const { data: appsData, refetch: refetchApps } = useListApplications();
  const { data: domainsData, refetch: refetchDomains } = useListDomains();
  const { data: runtimeData, refetch: refetchRuntime } = useGetRuntimeStatus();

  const createApplicationMutation = useCreateApplication();
  const deployApplicationMutation = useDeployApplication();
  const upgradeCelldMutation = useUpgradeCelldDaemon();
  const createDomainMutation = useCreateDomain();

  const nodes = Array.isArray(nodesData) ? nodesData : [];
  const apps = Array.isArray(appsData) ? appsData : [];
  const domains = Array.isArray(domainsData) ? domainsData : [];
  const runtimeStatus = runtimeData;

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
  const [targetVersion, setTargetVersion] = useState('0.3.0');
  const [selectedAppId, setSelectedAppId] = useState('');
  const [domainHost, setDomainHost] = useState('');

  // Live log streaming
  const [activeDeploymentId, setActiveDeploymentId] = useState<string | null>(null);
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

  // EventSource live log streaming
  useEffect(() => {
    if (!activeDeploymentId) return;

    setLogs([]);
    const es = new EventSource(`/api/v1/deployments/${activeDeploymentId}/logs/stream`);

    es.onmessage = (event) => {
      try {
        const entry = JSON.parse(event.data);
        setLogs(prev => [...prev, entry]);
      } catch (e) {
        console.error("Failed parsing log entry", e);
      }
    };

    es.addEventListener('complete', () => {
      es.close();
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

    if (autoDeploy && res && res.id) {
      handleDeployApp(res.id);
    }
  };

  const handleDeployApp = async (appId: string) => {
    const res = await deployApplicationMutation.mutateAsync({
      id: appId,
      data: { commitHash: 'HEAD' }
    });
    if (res && res.id) {
      setActiveDeploymentId(res.id);
      setTab('logs');
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

          <nav className="space-y-1">
            <button
              onClick={() => setTab('nodes')}
              className={`w-full flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium transition ${
                tab === 'nodes' ? 'bg-zinc-800 text-emerald-400 shadow-sm' : 'text-zinc-400 hover:bg-zinc-900 hover:text-zinc-200'
              }`}
            >
              <Server className="w-4 h-4" />
              Fleet Nodes
              <span className="ml-auto text-xs px-2 py-0.5 rounded-full bg-zinc-800 text-zinc-400">
                {nodes.length}
              </span>
            </button>

            <button
              onClick={() => setTab('apps')}
              className={`w-full flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium transition ${
                tab === 'apps' ? 'bg-zinc-800 text-emerald-400 shadow-sm' : 'text-zinc-400 hover:bg-zinc-900 hover:text-zinc-200'
              }`}
            >
              <Layers className="w-4 h-4" />
              Applications
              <span className="ml-auto text-xs px-2 py-0.5 rounded-full bg-zinc-800 text-zinc-400">
                {apps.length}
              </span>
            </button>

            <button
              onClick={() => setTab('domains')}
              className={`w-full flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium transition ${
                tab === 'domains' ? 'bg-zinc-800 text-emerald-400 shadow-sm' : 'text-zinc-400 hover:bg-zinc-900 hover:text-zinc-200'
              }`}
            >
              <Globe className="w-4 h-4" />
              Traefik Routing
              <span className="ml-auto text-xs px-2 py-0.5 rounded-full bg-zinc-800 text-zinc-400">
                {domains.length}
              </span>
            </button>

            <button
              onClick={() => setTab('logs')}
              className={`w-full flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium transition ${
                tab === 'logs' ? 'bg-zinc-800 text-emerald-400 shadow-sm' : 'text-zinc-400 hover:bg-zinc-900 hover:text-zinc-200'
              }`}
            >
              <Terminal className="w-4 h-4" />
              Build Logs
            </button>
          </nav>
        </div>

        {/* Fleet Status Badge */}
        <div className="p-4 rounded-xl border border-zinc-800 bg-zinc-900/60 text-xs space-y-2">
          <div className="flex items-center justify-between text-zinc-400">
            <span>celld Runtime</span>
            <span className="text-emerald-400 font-mono">v{runtimeStatus?.currentCelldVersion || '0.2.0'}</span>
          </div>
          <div className="flex items-center justify-between text-zinc-400">
            <span>Storage Driver</span>
            <span className="text-zinc-200 uppercase font-semibold">{runtimeStatus?.storageBackend || 'Garage S3'}</span>
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
        {/* Top Navbar */}
        <header className="h-16 border-b border-zinc-800/80 px-8 flex items-center justify-between bg-zinc-950/60 backdrop-blur">
          <div className="flex items-center gap-4">
            <h2 className="text-xl font-semibold capitalize tracking-tight">{tab}</h2>
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

        {/* View Content */}
        <div className="flex-1 overflow-y-auto p-8">
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

          {tab === 'apps' && (
            <div className="space-y-4">
              <div className="grid grid-cols-1 gap-3">
                {apps.map(app => (
                  <div key={app.id} className="p-5 rounded-xl border border-zinc-800/80 bg-zinc-900/20 flex items-center justify-between hover:border-zinc-700 transition">
                    <div className="space-y-1.5">
                      <div className="flex items-center gap-2.5">
                        <h4 className="font-semibold text-base">{app.name}</h4>
                        <span className={`text-[10px] px-2 py-0.5 rounded-full font-bold uppercase ${
                          app.status === 'running' ? 'bg-emerald-950 text-emerald-400 border border-emerald-800' : 'bg-zinc-800 text-zinc-400'
                        }`}>
                          {app.status}
                        </span>
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
                    </div>

                    <div className="flex items-center gap-2">
                      {app.sourceType === 'inline' && (
                        <button
                          type="button"
                          onClick={() => setViewingCodeApp(app)}
                          className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg border border-zinc-800 bg-zinc-900 hover:bg-zinc-800 text-zinc-300 text-xs font-medium transition"
                        >
                          <Code className="w-3.5 h-3.5 text-zinc-400" />
                          View Code
                        </button>
                      )}
                      <button
                        onClick={() => handleDeployApp(app.id)}
                        className="flex items-center gap-2 px-3.5 py-1.5 rounded-lg bg-emerald-500 hover:bg-emerald-600 text-zinc-950 text-xs font-semibold transition"
                      >
                        <Play className="w-3.5 h-3.5 fill-current" />
                        Deploy Now
                      </button>
                    </div>
                  </div>
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
            <div className="h-full flex flex-col space-y-3">
              <div className="flex items-center justify-between">
                <span className="text-sm font-semibold text-zinc-400 uppercase tracking-wider">Live Build & Deployment Stream</span>
                {activeDeploymentId && <span className="text-xs font-mono text-zinc-500">ID: {activeDeploymentId}</span>}
              </div>

              <div className="flex-1 bg-black/80 rounded-xl p-5 font-mono text-xs text-zinc-300 overflow-y-auto border border-zinc-900 space-y-1.5 shadow-inner">
                {logs.length === 0 ? (
                  <div className="text-zinc-600 italic">Waiting for deployment stream...</div>
                ) : (
                  logs.map((l, idx) => (
                    <div key={idx} className="flex items-start gap-3">
                      <span className="text-zinc-600">{new Date(l.timestamp).toLocaleTimeString()}</span>
                      <span className="text-emerald-500 font-bold uppercase text-[10px]">[{l.step}]</span>
                      <span className={l.level === 'error' ? 'text-red-400' : 'text-zinc-200'}>{l.message}</span>
                    </div>
                  ))
                )}
              </div>
            </div>
          )}
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
    </div>
  );
}
