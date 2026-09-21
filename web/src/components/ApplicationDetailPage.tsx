import React, { useState, useEffect } from 'react';
import {
  ArrowLeft,
  Play,
  RefreshCw,
  Send,
  Terminal,
  Code,
  Globe,
  Settings,
  ShieldCheck,
  Trash2,
  Plus,
  ExternalLink,
  Copy,
  Check,
  Zap,
  GitBranch,
  Save,
  Eye,
  EyeOff,
  Clock,
  Database,
  HardDrive,
  GitMerge,
  Box,
  Link2,
  ArrowRight,
  Inbox,
  X,
  Cpu,
} from 'lucide-react';
import type { Application, ResourceBinding, ResourceBindingType } from '../api/model';
import type { ActiveTab } from '../App';
import { useListDeployments } from '../api/generated/deployments/deployments';
import { useUpdateApplication } from '../api/generated/applications/applications';
import { useListDomains, useCreateDomain } from '../api/generated/domains/domains';
import { CodeEditor } from './CodeEditor';

export function getDeploymentStatusBadge(status: string) {
  switch (status) {
    case 'active':
      return 'bg-emerald-950 text-emerald-400 border border-emerald-800';
    case 'superseded':
      return 'bg-zinc-800/80 text-zinc-400 border border-zinc-700/60';
    case 'building':
    case 'deploying':
      return 'bg-amber-950 text-amber-400 border border-amber-800';
    case 'failed':
      return 'bg-rose-950 text-rose-400 border border-rose-800';
    case 'pending':
    default:
      return 'bg-zinc-900 text-zinc-500 border border-zinc-800';
  }
}

export interface ApplicationDetailPageProps {
  app: Application;
  onBack: () => void;
  onDeploy: (appId: string) => Promise<void> | void;
  isDeploying: boolean;
  onTestApp: (app: Application) => void;
  onRefreshApps: () => void;
  onNavigateToService?: (targetTab: ActiveTab, resourceId?: string) => void;
}

export function ApplicationDetailPage({
  app,
  onBack,
  onDeploy,
  isDeploying,
  onTestApp,
  onRefreshApps,
  onNavigateToService,
}: ApplicationDetailPageProps) {
  const [activeTab, setActiveTab] = useState<'builds' | 'code' | 'domains' | 'bindings' | 'settings'>('builds');

  // Subdomain & URLs
  const subdomain = app.subdomain || app.name;
  const testUrl = app.testUrl || `http://${subdomain}.localhost:8000`;
  const [copiedUrl, setCopiedUrl] = useState(false);

  // Deployments hook
  const { data: deploymentsData, refetch: refetchDeployments } = useListDeployments(app.id);
  const deployments = Array.isArray(deploymentsData) ? deploymentsData : [];
  const activeDep = deployments.find(d => d.id === app.activeDeploymentId) || (deployments.length > 0 ? deployments[0] : null);
  const activeBuildVersion = activeDep?.buildVersion || 1;
  const latestBuildVersion = deployments.reduce((max, d) => Math.max(max, d.buildVersion || 1), 0);
  const nextDeployVersion = latestBuildVersion + 1;

  // Selected deployment for logs
  const [selectedDepId, setSelectedDepId] = useState<string | null>(null);
  const selectedDep = deployments.find(d => d.id === selectedDepId) || null;
  const [depLogs, setDepLogs] = useState<Array<{ timestamp: string; step: string; message: string; level: string }>>([]);
  const [isLoadingLogs, setIsLoadingLogs] = useState(false);

  // Code editor state
  const [editedCode, setEditedCode] = useState(app.inlineCode || '');
  const [isSavingCode, setIsSavingCode] = useState(false);
  const [codeSaveStatus, setCodeSaveStatus] = useState<'idle' | 'saved' | 'error'>('idle');

  // Git branch state
  const [gitBranch, setGitBranch] = useState(app.branch || 'main');
  const [isSavingBranch, setIsSavingBranch] = useState(false);

  // Custom Domains state
  const { data: allDomainsData, refetch: refetchDomains } = useListDomains();
  const allDomains = Array.isArray(allDomainsData) ? allDomainsData : [];
  const appDomains = allDomains.filter(d => d.applicationId === app.id);
  const [newHostname, setNewHostname] = useState('');
  const [isCreatingDomain, setIsCreatingDomain] = useState(false);
  const createDomainMutation = useCreateDomain();

  // Environment variables state
  const [envVars, setEnvVars] = useState<Array<{ key: string; value: string; isSecret: boolean }>>(
    app.envVars ? app.envVars.map(e => ({ key: e.key || (e as any).Key || '', value: e.value || (e as any).Value || '', isSecret: !!(e.isSecret || (e as any).IsSecret) })) : []
  );
  const [showSecretValues, setShowSecretValues] = useState<Record<number, boolean>>({});
  const [isSavingEnv, setIsSavingEnv] = useState(false);
  const [envSaveStatus, setEnvSaveStatus] = useState<'idle' | 'saved' | 'error'>('idle');

  const updateAppMutation = useUpdateApplication();

  // Service Bindings State
  const [bindings, setBindings] = useState<ResourceBinding[]>(app.bindings || []);
  const [isSavingBindings, setIsSavingBindings] = useState(false);
  const [showAddBindingModal, setShowAddBindingModal] = useState(false);

  // New binding form state
  const [newBindingType, setNewBindingType] = useState<ResourceBindingType>('kv_namespace');
  const [newBindingName, setNewBindingName] = useState('MY_KV');
  const [newBindingResourceId, setNewBindingResourceId] = useState('');
  const [customResourceId, setCustomResourceId] = useState('');

  // Attached fleet services state (reverse bindings)
  const [attachedCron, setAttachedCron] = useState<any[]>([]);
  const [attachedQueues, setAttachedQueues] = useState<any[]>([]);
  const [attachedDO, setAttachedDO] = useState<any[]>([]);
  const [attachedWorkflows, setAttachedWorkflows] = useState<any[]>([]);
  const [isLoadingFleet, setIsLoadingFleet] = useState(false);

  // Available fleet resources for binding dropdown
  const [fleetKV, setFleetKV] = useState<any[]>([]);
  const [fleetD1, setFleetD1] = useState<any[]>([]);
  const [fleetR2, setFleetR2] = useState<any[]>([]);
  const [fleetQueues, setFleetQueues] = useState<any[]>([]);
  const [fleetWorkflows, setFleetWorkflows] = useState<any[]>([]);

  // Sync bindings when app updates
  useEffect(() => {
    setBindings(app.bindings || []);
  }, [app.bindings]);

  // Fetch fleet services and available resources
  const fetchFleetServices = async () => {
    setIsLoadingFleet(true);
    try {
      const [cronRes, queuesRes, doRes, wfRes, kvRes, d1Res, r2Res] = await Promise.all([
        fetch('/api/v1/cron').catch(() => null),
        fetch('/api/v1/queues').catch(() => null),
        fetch('/api/v1/durable-objects').catch(() => null),
        fetch('/api/v1/workflows').catch(() => null),
        fetch('/api/v1/kv/namespaces').catch(() => null),
        fetch('/api/v1/d1/databases').catch(() => null),
        fetch('/api/v1/r2/buckets').catch(() => null),
      ]);

      const appIdMatches = (target: string | undefined | null) => isServiceAttachedToApp(target, app);

      if (cronRes?.ok) {
        const data = await cronRes.json();
        const list = Array.isArray(data) ? data : [];
        setAttachedCron(list.filter((c: any) => appIdMatches(c.targetAppId)));
      }
      if (queuesRes?.ok) {
        const data = await queuesRes.json();
        const list = Array.isArray(data) ? data : [];
        setFleetQueues(list);
        setAttachedQueues(list.filter((q: any) => appIdMatches(q.consumerAppId)));
      }
      if (doRes?.ok) {
        const data = await doRes.json();
        const list = Array.isArray(data) ? data : [];
        setAttachedDO(list.filter((d: any) => appIdMatches(d.appId)));
      }
      if (wfRes?.ok) {
        const data = await wfRes.json();
        const list = Array.isArray(data) ? data : [];
        setFleetWorkflows(list);
        setAttachedWorkflows(list.filter((w: any) => appIdMatches(w.targetAppId)));
      }
      if (kvRes?.ok) {
        const data = await kvRes.json();
        setFleetKV(Array.isArray(data) ? data : []);
      }
      if (d1Res?.ok) {
        const data = await d1Res.json();
        setFleetD1(Array.isArray(data) ? data : []);
      }
      if (r2Res?.ok) {
        const data = await r2Res.json();
        setFleetR2(Array.isArray(data) ? data : []);
      }
    } catch (err) {
      console.error('Error fetching fleet services:', err);
    } finally {
      setIsLoadingFleet(false);
    }
  };

  useEffect(() => {
    fetchFleetServices();
  }, [app.id, app.name, app.subdomain]);

  const totalAttachedFleet = attachedCron.length + attachedQueues.length + attachedDO.length + attachedWorkflows.length;
  const totalBindingsCount = bindings.length + totalAttachedFleet;

  const handleTypeChange = (t: ResourceBindingType) => {
    setNewBindingType(t);
    if (t === 'kv_namespace') {
      setNewBindingName('MY_KV');
      setNewBindingResourceId(fleetKV[0]?.name || fleetKV[0]?.id || '');
    } else if (t === 'd1_database') {
      setNewBindingName('DB');
      setNewBindingResourceId(fleetD1[0]?.name || fleetD1[0]?.id || '');
    } else if (t === 'r2_bucket') {
      setNewBindingName('MY_BUCKET');
      setNewBindingResourceId(fleetR2[0]?.name || '');
    } else if (t === 'queue') {
      setNewBindingName('MY_QUEUE');
      setNewBindingResourceId(fleetQueues[0]?.name || fleetQueues[0]?.id || '');
    } else if (t === 'workflow') {
      setNewBindingName('MY_WORKFLOW');
      setNewBindingResourceId(fleetWorkflows[0]?.name || fleetWorkflows[0]?.id || '');
    }
  };

  const handleAddBinding = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newBindingName.trim()) return;
    const finalResourceId = newBindingResourceId === '__custom__' ? customResourceId.trim() : (newBindingResourceId.trim() || undefined);
    const newBinding: ResourceBinding = {
      type: newBindingType,
      name: newBindingName.trim().replace(/^env\./, ''),
      resourceId: finalResourceId,
    };
    const updated = [...bindings, newBinding];
    setBindings(updated);
    setIsSavingBindings(true);
    try {
      await updateAppMutation.mutateAsync({
        id: app.id,
        data: { bindings: updated },
      });
      onRefreshApps();
      setShowAddBindingModal(false);
      setCustomResourceId('');
    } catch (err) {
      console.error('Failed to add binding:', err);
    } finally {
      setIsSavingBindings(false);
    }
  };

  const handleDeleteBinding = async (index: number) => {
    const updated = bindings.filter((_, i) => i !== index);
    setBindings(updated);
    setIsSavingBindings(true);
    try {
      await updateAppMutation.mutateAsync({
        id: app.id,
        data: { bindings: updated },
      });
      onRefreshApps();
    } catch (err) {
      console.error('Failed to delete binding:', err);
    } finally {
      setIsSavingBindings(false);
    }
  };

  // Initialize selected deployment for logs
  useEffect(() => {
    if (!selectedDepId && deployments.length > 0) {
      setSelectedDepId(deployments[0].id);
    }
  }, [deployments, selectedDepId]);

  // Sync edited code when app changes
  useEffect(() => {
    setEditedCode(app.inlineCode || '');
  }, [app.inlineCode]);

  // Fetch logs for selected deployment
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

  const copyUrl = () => {
    navigator.clipboard.writeText(testUrl);
    setCopiedUrl(true);
    setTimeout(() => setCopiedUrl(false), 2000);
  };

  // Save Code
  const handleSaveCode = async (triggerDeploy = false) => {
    setIsSavingCode(true);
    setCodeSaveStatus('idle');
    try {
      await updateAppMutation.mutateAsync({
        id: app.id,
        data: { inlineCode: editedCode },
      });
      setCodeSaveStatus('saved');
      onRefreshApps();
      setTimeout(() => setCodeSaveStatus('idle'), 3000);

      if (triggerDeploy) {
        await onDeploy(app.id);
        refetchDeployments();
      }
    } catch (err) {
      console.error('Failed saving code:', err);
      setCodeSaveStatus('error');
    } finally {
      setIsSavingCode(false);
    }
  };

  // Save Git branch
  const handleSaveBranch = async () => {
    setIsSavingBranch(true);
    try {
      await updateAppMutation.mutateAsync({
        id: app.id,
        data: { branch: gitBranch },
      });
      onRefreshApps();
    } finally {
      setIsSavingBranch(false);
    }
  };

  // Add Domain
  const handleAddDomain = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newHostname) return;
    setIsCreatingDomain(true);
    try {
      await createDomainMutation.mutateAsync({
        data: {
          applicationId: app.id,
          hostname: newHostname.trim(),
          pathPrefix: '/',
        },
      });
      setNewHostname('');
      refetchDomains();
    } finally {
      setIsCreatingDomain(false);
    }
  };

  // Save Environment Variables
  const handleSaveEnvVars = async () => {
    setIsSavingEnv(true);
    setEnvSaveStatus('idle');
    try {
      const validEnvs = envVars.filter(e => e.key.trim() !== '');
      await updateAppMutation.mutateAsync({
        id: app.id,
        data: {
          envVars: validEnvs.map(e => ({
            key: e.key.trim(),
            value: e.value,
            isSecret: e.isSecret,
          })),
        },
      });
      setEnvSaveStatus('saved');
      onRefreshApps();
      setTimeout(() => setEnvSaveStatus('idle'), 3000);
    } catch (err) {
      console.error('Failed saving env vars:', err);
      setEnvSaveStatus('error');
    } finally {
      setIsSavingEnv(false);
    }
  };

  const addEnvVarRow = () => {
    setEnvVars([...envVars, { key: '', value: '', isSecret: false }]);
  };

  const removeEnvVarRow = (index: number) => {
    setEnvVars(envVars.filter((_, i) => i !== index));
  };

  const updateEnvVarRow = (index: number, field: 'key' | 'value' | 'isSecret', val: any) => {
    setEnvVars(envVars.map((row, i) => (i === index ? { ...row, [field]: val } : row)));
  };

  return (
    <div className="flex-1 flex flex-col overflow-hidden">
      {/* Application Top Bar */}
      <div className="border-b border-zinc-800/80 bg-zinc-950/80 backdrop-blur px-8 py-4 space-y-4">
        {/* Navigation Breadcrumb */}
        <div className="flex items-center justify-between">
          <button
            type="button"
            onClick={onBack}
            className="flex items-center gap-2 text-xs font-medium text-zinc-400 hover:text-zinc-200 transition"
          >
            <ArrowLeft className="w-4 h-4" />
            <span>Applications</span>
            <span className="text-zinc-600">/</span>
            <span className="text-zinc-200 font-semibold">{app.name}</span>
          </button>

          {/* Persistent Action Buttons */}
          <div className="flex items-center gap-3">
            <button
              type="button"
              onClick={() => onTestApp(app)}
              className="flex items-center gap-1.5 px-3.5 py-1.5 rounded-lg border border-emerald-900/60 bg-emerald-950/40 hover:bg-emerald-950/70 text-emerald-400 text-xs font-semibold transition"
            >
              <Send className="w-3.5 h-3.5" />
              Test App
            </button>

            <button
              type="button"
              onClick={() => onDeploy(app.id)}
              disabled={isDeploying}
              className="flex items-center gap-2 px-4 py-1.5 rounded-lg bg-emerald-500 hover:bg-emerald-600 disabled:opacity-50 text-zinc-950 text-xs font-bold transition shadow-sm"
            >
              {isDeploying ? (
                <RefreshCw className="w-3.5 h-3.5 animate-spin" />
              ) : (
                <Play className="w-3.5 h-3.5 fill-current" />
              )}
              {isDeploying ? 'Deploying...' : `Deploy v${nextDeployVersion}`}
            </button>
          </div>
        </div>

        {/* Project Header Info */}
        <div className="flex items-center justify-between flex-wrap gap-4">
          <div className="space-y-1.5">
            <div className="flex items-center gap-3 flex-wrap">
              <h1 className="text-2xl font-bold tracking-tight text-zinc-100">{app.name}</h1>
              <span className={`text-[11px] px-2.5 py-0.5 rounded-full font-bold uppercase ${
                app.status === 'running' ? 'bg-emerald-950 text-emerald-400 border border-emerald-800' : 'bg-zinc-800 text-zinc-400'
              }`}>
                {app.status}
              </span>
              <span className="text-[11px] px-2.5 py-0.5 rounded-full font-mono font-bold bg-emerald-950/80 text-emerald-300 border border-emerald-800/80">
                v{activeBuildVersion}
              </span>
              {app.sourceType === 'inline' ? (
                <span className="text-[10px] px-2 py-0.5 rounded-full font-semibold bg-sky-950/80 text-sky-400 border border-sky-800/60 flex items-center gap-1">
                  <Zap className="w-2.5 h-2.5" /> Inline Worker
                </span>
              ) : (
                <span className="text-[10px] px-2 py-0.5 rounded-full font-semibold bg-purple-950/80 text-purple-400 border border-purple-800/60 flex items-center gap-1">
                  <GitBranch className="w-2.5 h-2.5" /> Git ({app.branch || 'main'})
                </span>
              )}

              {/* Service Bindings Status Badge */}
              <button
                type="button"
                onClick={() => setActiveTab('bindings')}
                className={`text-[10px] px-2.5 py-0.5 rounded-full font-semibold flex items-center gap-1.5 transition cursor-pointer ${
                  totalBindingsCount > 0
                    ? 'bg-indigo-950/80 text-indigo-300 border border-indigo-800/80 hover:bg-indigo-900/60'
                    : 'bg-zinc-900 text-zinc-500 border border-zinc-800 hover:text-zinc-300'
                }`}
                title={totalBindingsCount > 0 ? `${totalBindingsCount} bound services. Click to view bindings.` : 'No services bound. Click to configure bindings.'}
              >
                <Link2 className="w-2.5 h-2.5 text-indigo-400" />
                {totalBindingsCount > 0 ? `${totalBindingsCount} Bound Services` : 'No Bound Services'}
              </button>
            </div>

            {/* Testable Subdomain URL */}
            <div className="flex items-center gap-2">
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
                {copiedUrl ? <Check className="w-3.5 h-3.5 text-emerald-400" /> : <Copy className="w-3.5 h-3.5" />}
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
        </div>

        {/* Tab Buttons */}
        <div className="flex border-b border-zinc-800 pt-2 gap-6">
          <button
            type="button"
            onClick={() => setActiveTab('builds')}
            className={`pb-3 text-xs font-semibold flex items-center gap-2 border-b-2 transition ${
              activeTab === 'builds'
                ? 'border-emerald-500 text-emerald-400'
                : 'border-transparent text-zinc-400 hover:text-zinc-200'
            }`}
          >
            <Terminal className="w-3.5 h-3.5" />
            Builds
            <span className="text-[10px] px-1.5 py-0.2 rounded-full bg-zinc-800 text-zinc-400">
              {deployments.length}
            </span>
          </button>

          <button
            type="button"
            onClick={() => setActiveTab('code')}
            className={`pb-3 text-xs font-semibold flex items-center gap-2 border-b-2 transition ${
              activeTab === 'code'
                ? 'border-emerald-500 text-emerald-400'
                : 'border-transparent text-zinc-400 hover:text-zinc-200'
            }`}
          >
            <Code className="w-3.5 h-3.5" />
            Code Editor
          </button>

          <button
            type="button"
            onClick={() => setActiveTab('domains')}
            className={`pb-3 text-xs font-semibold flex items-center gap-2 border-b-2 transition ${
              activeTab === 'domains'
                ? 'border-emerald-500 text-emerald-400'
                : 'border-transparent text-zinc-400 hover:text-zinc-200'
            }`}
          >
            <Globe className="w-3.5 h-3.5" />
            Domain Setup
            <span className="text-[10px] px-1.5 py-0.2 rounded-full bg-zinc-800 text-zinc-400">
              {appDomains.length + 1}
            </span>
          </button>

          <button
            type="button"
            onClick={() => setActiveTab('bindings')}
            className={`pb-3 text-xs font-semibold flex items-center gap-2 border-b-2 transition ${
              activeTab === 'bindings'
                ? 'border-emerald-500 text-emerald-400'
                : 'border-transparent text-zinc-400 hover:text-zinc-200'
            }`}
          >
            <Link2 className="w-3.5 h-3.5" />
            Service Bindings
            <span className={`text-[10px] px-1.5 py-0.2 rounded-full font-bold ${
              totalBindingsCount > 0 ? 'bg-indigo-950 text-indigo-300 border border-indigo-800/60' : 'bg-zinc-800 text-zinc-400'
            }`}>
              {totalBindingsCount}
            </span>
          </button>

          <button
            type="button"
            onClick={() => setActiveTab('settings')}
            className={`pb-3 text-xs font-semibold flex items-center gap-2 border-b-2 transition ${
              activeTab === 'settings'
                ? 'border-emerald-500 text-emerald-400'
                : 'border-transparent text-zinc-400 hover:text-zinc-200'
            }`}
          >
            <Settings className="w-3.5 h-3.5" />
            Settings
          </button>
        </div>
      </div>

      {/* Tab Contents */}
      <div className="flex-1 overflow-y-auto p-8">
        {/* Tab 1: Builds */}
        {activeTab === 'builds' && (
          <div className="grid grid-cols-12 gap-6 h-full min-h-[450px]">
            {/* Deployments List (Left Column) */}
            <div className="col-span-4 bg-zinc-900/30 border border-zinc-800/80 rounded-xl p-4 space-y-2.5 overflow-y-auto max-h-[600px]">
              <div className="flex items-center justify-between text-xs font-semibold uppercase tracking-wider text-zinc-400 mb-2">
                <span>Deployment Versions ({deployments.length})</span>
                <button
                  type="button"
                  onClick={() => refetchDeployments()}
                  className="text-zinc-400 hover:text-zinc-200 transition"
                >
                  <RefreshCw className="w-3.5 h-3.5" />
                </button>
              </div>

              {deployments.length === 0 ? (
                <div className="text-xs text-zinc-500 italic p-3">No deployments found for this application.</div>
              ) : (
                deployments.map(dep => {
                  const isSelected = selectedDepId === dep.id;
                  const d = new Date(dep.createdAt);
                  const dateStr = isNaN(d.getTime()) ? '' : d.toLocaleString();
                  return (
                    <div
                      key={dep.id}
                      onClick={() => setSelectedDepId(dep.id)}
                      className={`p-3.5 rounded-xl border cursor-pointer transition space-y-1.5 ${
                        isSelected
                          ? 'bg-zinc-800 border-emerald-500 text-zinc-100 shadow-sm'
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
                      <div className="flex items-center gap-1.5 text-[10px] text-zinc-500 font-mono">
                        <Clock className="w-3 h-3 text-zinc-600" />
                        <span>{dateStr}</span>
                      </div>
                    </div>
                  );
                })
              )}
            </div>

            {/* Build Logs Console (Right Column) */}
            <div className="col-span-8 flex flex-col space-y-2 min-h-0">
              <div className="flex items-center justify-between text-xs text-zinc-400">
                <div className="flex items-center gap-2.5">
                  <span className="font-semibold uppercase tracking-wider">
                    {selectedDep ? `Build Output (v${selectedDep.buildVersion || 1})` : 'Build Output'}
                  </span>
                  {selectedDep && (
                    <span className={`text-[10px] px-2 py-0.5 rounded-full font-bold uppercase ${getDeploymentStatusBadge(selectedDep.status)}`}>
                      {selectedDep.status}
                    </span>
                  )}
                </div>
                {selectedDepId && (
                  <span className="font-mono text-[10px] text-zinc-500">{selectedDepId}</span>
                )}
              </div>

              <div className="flex-1 bg-black/90 rounded-xl p-5 font-mono text-xs text-zinc-300 overflow-y-auto border border-zinc-900 space-y-2 shadow-inner min-h-[400px]">
                {isLoadingLogs ? (
                  <div className="flex items-center gap-2 text-zinc-500 py-4">
                    <RefreshCw className="w-3.5 h-3.5 animate-spin text-emerald-400" />
                    <span>Loading build logs...</span>
                  </div>
                ) : depLogs.length === 0 ? (
                  <div className="text-zinc-600 italic">No log entries available for this build version.</div>
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
                      <div key={idx} className="flex items-start gap-3 leading-relaxed">
                        <span className="text-zinc-600 shrink-0 select-none">{timeStr}</span>
                        <span className="text-emerald-500 font-bold uppercase text-[10px] shrink-0 select-none">
                          [{l.step}]
                        </span>
                        <span className={l.level === 'error' ? 'text-rose-400 font-semibold' : 'text-zinc-200'}>
                          {l.message}
                        </span>
                      </div>
                    );
                  })
                )}
              </div>
            </div>
          </div>
        )}

        {/* Tab 2: Code Editor */}
        {activeTab === 'code' && (
          <div className="space-y-4 max-w-5xl">
            {app.sourceType === 'inline' ? (
              <>
                <div className="flex items-center justify-between">
                  <div>
                    <h3 className="text-sm font-semibold text-zinc-200">Worker Source Code</h3>
                    <p className="text-xs text-zinc-400">
                      Standard Cloudflare Worker ES module handling fetch events.
                    </p>
                  </div>

                  <div className="flex items-center gap-2">
                    {codeSaveStatus === 'saved' && (
                      <span className="text-xs text-emerald-400 font-semibold flex items-center gap-1">
                        <Check className="w-3.5 h-3.5" /> Saved!
                      </span>
                    )}
                    {codeSaveStatus === 'error' && (
                      <span className="text-xs text-rose-400 font-semibold">Failed to save code</span>
                    )}

                    <button
                      type="button"
                      onClick={() => handleSaveCode(false)}
                      disabled={isSavingCode || editedCode === app.inlineCode}
                      className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg border border-zinc-700 bg-zinc-900 hover:bg-zinc-800 disabled:opacity-50 text-zinc-200 text-xs font-semibold transition"
                    >
                      <Save className="w-3.5 h-3.5 text-zinc-400" />
                      Save
                    </button>

                    <button
                      type="button"
                      onClick={() => handleSaveCode(true)}
                      disabled={isSavingCode || isDeploying}
                      className="flex items-center gap-1.5 px-4 py-1.5 rounded-lg bg-emerald-500 hover:bg-emerald-600 disabled:opacity-50 text-zinc-950 text-xs font-bold transition shadow"
                    >
                      {isSavingCode || isDeploying ? (
                        <RefreshCw className="w-3.5 h-3.5 animate-spin" />
                      ) : (
                        <Play className="w-3.5 h-3.5 fill-current" />
                      )}
                      Save & Deploy
                    </button>
                  </div>
                </div>

                <CodeEditor
                  value={editedCode}
                  onChange={setEditedCode}
                  minHeight="420px"
                />
              </>
            ) : (
              <div className="p-6 rounded-xl border border-zinc-800 bg-zinc-900/30 space-y-4">
                <div>
                  <h3 className="text-base font-semibold text-zinc-200">Git Repository Integration</h3>
                  <p className="text-xs text-zinc-400 mt-1">
                    This application is linked to a remote Git repository. Builds are automatically compiled and bundled from Git commits.
                  </p>
                </div>

                <div className="space-y-3 font-mono text-xs">
                  <div>
                    <label className="text-zinc-500 block mb-1">Repository URL</label>
                    <div className="flex items-center gap-2">
                      <input
                        type="text"
                        readOnly
                        value={app.gitRepo || ''}
                        className="flex-1 bg-zinc-950 border border-zinc-800 rounded-lg p-2.5 text-zinc-200 select-all"
                      />
                      {app.gitRepo && (
                        <a
                          href={app.gitRepo}
                          target="_blank"
                          rel="noopener noreferrer"
                          className="px-3 py-2 rounded-lg border border-zinc-800 hover:bg-zinc-800 text-zinc-300 transition flex items-center gap-1"
                        >
                          <ExternalLink className="w-3.5 h-3.5" /> GitHub
                        </a>
                      )}
                    </div>
                  </div>

                  <div>
                    <label className="text-zinc-500 block mb-1">Production Branch</label>
                    <div className="flex items-center gap-2">
                      <input
                        type="text"
                        value={gitBranch}
                        onChange={e => setGitBranch(e.target.value)}
                        className="w-64 bg-zinc-950 border border-zinc-800 rounded-lg p-2 text-zinc-200 outline-none focus:border-emerald-500"
                      />
                      <button
                        type="button"
                        onClick={handleSaveBranch}
                        disabled={isSavingBranch || gitBranch === app.branch}
                        className="px-3 py-1.5 rounded-lg bg-zinc-800 hover:bg-zinc-700 disabled:opacity-50 text-zinc-200 text-xs font-semibold transition"
                      >
                        {isSavingBranch ? 'Updating...' : 'Update Branch'}
                      </button>
                    </div>
                  </div>
                </div>
              </div>
            )}
          </div>
        )}

        {/* Tab 3: Domain Setup */}
        {activeTab === 'domains' && (
          <div className="space-y-6 max-w-4xl">
            {/* Default Traefik Subdomain Route */}
            <div className="p-5 rounded-xl border border-zinc-800 bg-zinc-900/20 space-y-3">
              <div className="flex items-start justify-between">
                <div>
                  <h3 className="text-sm font-semibold text-zinc-200">Default Subdomain Ingress</h3>
                  <p className="text-xs text-zinc-400 mt-0.5">
                    Automatically assigned and synchronized with the Traefik fleet proxy.
                  </p>
                </div>
                <span className="text-xs font-semibold text-emerald-400 bg-emerald-950/60 border border-emerald-800/80 px-2.5 py-0.5 rounded-full flex items-center gap-1">
                  <ShieldCheck className="w-3.5 h-3.5" /> Active
                </span>
              </div>

              <div className="grid grid-cols-2 gap-3 pt-1">
                <div className="p-3 rounded-lg bg-zinc-950 border border-zinc-800/80 font-mono text-xs flex items-center justify-between">
                  <span className="text-emerald-400">{subdomain}.localhost</span>
                  <span className="text-[10px] text-zinc-500">RFC 6761 Wildcard</span>
                </div>
                <div className="p-3 rounded-lg bg-zinc-950 border border-zinc-800/80 font-mono text-xs flex items-center justify-between">
                  <span className="text-emerald-400">{subdomain}.cubit.local</span>
                  <span className="text-[10px] text-zinc-500">Local Traefik Mesh</span>
                </div>
              </div>
            </div>

            {/* Custom Domains */}
            <div className="space-y-4">
              <div className="flex items-center justify-between">
                <div>
                  <h3 className="text-sm font-semibold text-zinc-200">Custom Domains</h3>
                  <p className="text-xs text-zinc-400">
                    Route external hostnames with automatic Let's Encrypt TLS certificates.
                  </p>
                </div>
              </div>

              {appDomains.length === 0 ? (
                <div className="p-6 rounded-xl border border-dashed border-zinc-800 text-center text-xs text-zinc-500">
                  No custom domains configured for this application yet.
                </div>
              ) : (
                <div className="space-y-2">
                  {appDomains.map(d => (
                    <div
                      key={d.id}
                      className="p-4 rounded-xl border border-zinc-800 bg-zinc-900/30 flex items-center justify-between"
                    >
                      <div className="flex items-center gap-3">
                        <Globe className="w-4 h-4 text-emerald-400" />
                        <div>
                          <div className="font-mono text-xs font-semibold text-zinc-200">{d.hostname}</div>
                          <div className="text-[10px] text-zinc-500">Path prefix: {d.pathPrefix}</div>
                        </div>
                      </div>

                      <div className="flex items-center gap-2 text-xs text-emerald-400 bg-emerald-950/40 border border-emerald-900 px-3 py-1 rounded-full">
                        <ShieldCheck className="w-3.5 h-3.5" />
                        <span>Let's Encrypt Active</span>
                      </div>
                    </div>
                  ))}
                </div>
              )}

              {/* Add Custom Domain Form */}
              <form onSubmit={handleAddDomain} className="p-4 rounded-xl border border-zinc-800 bg-zinc-900/20 space-y-3">
                <span className="text-xs font-semibold text-zinc-300 block">Add New Domain</span>
                <div className="flex gap-2">
                  <input
                    type="text"
                    required
                    placeholder="api.yourdomain.com"
                    value={newHostname}
                    onChange={e => setNewHostname(e.target.value)}
                    className="flex-1 bg-zinc-950 border border-zinc-800 rounded-lg px-3 py-2 text-xs font-mono text-zinc-100 outline-none focus:border-emerald-500"
                  />
                  <button
                    type="submit"
                    disabled={isCreatingDomain}
                    className="px-4 py-2 rounded-lg bg-emerald-500 hover:bg-emerald-600 disabled:opacity-50 text-zinc-950 text-xs font-semibold transition flex items-center gap-1.5"
                  >
                    <Plus className="w-3.5 h-3.5" />
                    {isCreatingDomain ? 'Adding...' : 'Add Domain'}
                  </button>
                </div>
              </form>
            </div>
          </div>
        )}

        {/* Tab 4: Settings */}
        {activeTab === 'settings' && (
          <div className="space-y-6 max-w-4xl">
            {/* General Info */}
            <div className="p-5 rounded-xl border border-zinc-800 bg-zinc-900/20 space-y-4">
              <h3 className="text-sm font-semibold text-zinc-200">Application Identity</h3>
              <div className="grid grid-cols-2 gap-4 text-xs font-mono">
                <div>
                  <label className="text-zinc-500 block mb-1">ID</label>
                  <input
                    type="text"
                    readOnly
                    value={app.id}
                    className="w-full bg-zinc-950 border border-zinc-800 rounded-lg p-2 text-zinc-400 select-all"
                  />
                </div>
                <div>
                  <label className="text-zinc-500 block mb-1">Name</label>
                  <input
                    type="text"
                    readOnly
                    value={app.name}
                    className="w-full bg-zinc-950 border border-zinc-800 rounded-lg p-2 text-zinc-200"
                  />
                </div>
              </div>
            </div>

            {/* Environment Variables Editor */}
            <div className="p-5 rounded-xl border border-zinc-800 bg-zinc-900/20 space-y-4">
              <div className="flex items-center justify-between">
                <div>
                  <h3 className="text-sm font-semibold text-zinc-200">Environment Variables</h3>
                  <p className="text-xs text-zinc-400">
                    Passed to the Cloudflare Worker <code className="text-emerald-400 font-mono">env</code> parameter.
                  </p>
                </div>
                <div className="flex items-center gap-2">
                  {envSaveStatus === 'saved' && (
                    <span className="text-xs text-emerald-400 font-semibold flex items-center gap-1">
                      <Check className="w-3.5 h-3.5" /> Variables Saved!
                    </span>
                  )}
                  <button
                    type="button"
                    onClick={addEnvVarRow}
                    className="flex items-center gap-1 px-3 py-1.5 rounded-lg border border-zinc-800 bg-zinc-900 hover:bg-zinc-800 text-zinc-300 text-xs font-medium transition"
                  >
                    <Plus className="w-3.5 h-3.5" /> Add Variable
                  </button>
                  <button
                    type="button"
                    onClick={handleSaveEnvVars}
                    disabled={isSavingEnv}
                    className="px-3.5 py-1.5 rounded-lg bg-emerald-500 hover:bg-emerald-600 disabled:opacity-50 text-zinc-950 text-xs font-semibold transition"
                  >
                    {isSavingEnv ? 'Saving...' : 'Save Variables'}
                  </button>
                </div>
              </div>

              {envVars.length === 0 ? (
                <div className="p-6 rounded-xl border border-dashed border-zinc-800 text-center text-xs text-zinc-500">
                  No environment variables configured.
                </div>
              ) : (
                <div className="space-y-2">
                  {envVars.map((e, idx) => {
                    const isSecretRevealed = showSecretValues[idx];
                    return (
                      <div key={idx} className="flex items-center gap-2">
                        <input
                          type="text"
                          placeholder="KEY"
                          value={e.key}
                          onChange={ev => updateEnvVarRow(idx, 'key', ev.target.value)}
                          className="w-1/3 bg-zinc-950 border border-zinc-800 rounded-lg p-2 text-xs font-mono text-zinc-200 outline-none focus:border-emerald-500 uppercase"
                        />
                        <div className="relative flex-1">
                          <input
                            type={e.isSecret && !isSecretRevealed ? 'password' : 'text'}
                            placeholder="VALUE"
                            value={e.value}
                            onChange={ev => updateEnvVarRow(idx, 'value', ev.target.value)}
                            className="w-full bg-zinc-950 border border-zinc-800 rounded-lg p-2 pr-8 text-xs font-mono text-zinc-200 outline-none focus:border-emerald-500"
                          />
                          {e.isSecret && (
                            <button
                              type="button"
                              onClick={() => setShowSecretValues({ ...showSecretValues, [idx]: !isSecretRevealed })}
                              className="absolute right-2 top-2.5 text-zinc-500 hover:text-zinc-300"
                            >
                              {isSecretRevealed ? <EyeOff className="w-3.5 h-3.5" /> : <Eye className="w-3.5 h-3.5" />}
                            </button>
                          )}
                        </div>

                        <label className="flex items-center gap-1.5 text-xs text-zinc-400 select-none cursor-pointer px-2">
                          <input
                            type="checkbox"
                            checked={e.isSecret}
                            onChange={ev => updateEnvVarRow(idx, 'isSecret', ev.target.checked)}
                            className="rounded bg-zinc-950 border-zinc-800 text-emerald-500 focus:ring-0"
                          />
                          <span>Secret</span>
                        </label>

                        <button
                          type="button"
                          onClick={() => removeEnvVarRow(idx)}
                          className="p-2 rounded-lg text-zinc-500 hover:text-rose-400 hover:bg-rose-950/40 transition"
                        >
                          <Trash2 className="w-4 h-4" />
                        </button>
                      </div>
                    );
                  })}
                </div>
              )}
            </div>
          </div>
        )}

        {/* Tab 5: Service Bindings */}
        {activeTab === 'bindings' && (
          <div className="max-w-4xl space-y-8">
            {/* Status & Summary Card */}
            <div className={`p-5 rounded-2xl border ${
              totalBindingsCount > 0
                ? 'bg-gradient-to-r from-indigo-950/30 via-zinc-900/50 to-zinc-900/30 border-indigo-800/50'
                : 'bg-zinc-900/40 border-zinc-800/80'
            }`}>
              <div className="flex items-start justify-between gap-4 flex-wrap sm:flex-nowrap">
                <div className="flex items-start gap-3.5">
                  <div className={`p-2.5 rounded-xl border ${
                    totalBindingsCount > 0
                      ? 'bg-indigo-950/80 border-indigo-700/80 text-indigo-400'
                      : 'bg-zinc-800 border-zinc-700 text-zinc-400'
                  }`}>
                    <Link2 className="w-5 h-5" />
                  </div>
                  <div className="space-y-1">
                    <div className="flex items-center gap-2">
                      <h2 className="text-base font-bold text-zinc-100">
                        {totalBindingsCount > 0
                          ? `Worker Bound to ${totalBindingsCount} Service${totalBindingsCount > 1 ? 's' : ''}`
                          : 'No Active Service Bindings'}
                      </h2>
                      <span className={`text-[10px] px-2 py-0.5 rounded-full font-bold uppercase ${
                        totalBindingsCount > 0
                          ? 'bg-indigo-950 text-indigo-300 border border-indigo-800'
                          : 'bg-zinc-800 text-zinc-500 border border-zinc-700'
                      }`}>
                        {totalBindingsCount > 0 ? 'Connected' : 'Standalone'}
                      </span>
                    </div>
                    <p className="text-xs text-zinc-400 leading-relaxed">
                      {totalBindingsCount > 0 ? (
                        <>
                          This worker is actively bound to <strong className="text-zinc-200">{bindings.length} injected Cloudflare resource{bindings.length === 1 ? '' : 's'}</strong> (accessible via <code className="font-mono text-emerald-400 text-[11px]">env.*</code>) and <strong className="text-zinc-200">{totalAttachedFleet} attached fleet service{totalAttachedFleet === 1 ? '' : 's'}</strong>. Click on any service card below to open its control panel with that resource pre-selected.
                        </>
                      ) : (
                        <>
                          This worker is currently running standalone without any service bindings or attached fleet triggers. You can inject KV namespaces, D1 databases, R2 buckets, queues, or workflows below.
                        </>
                      )}
                    </p>
                  </div>
                </div>

                <div className="flex items-center gap-2">
                  <button
                    type="button"
                    onClick={fetchFleetServices}
                    disabled={isLoadingFleet}
                    className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg border border-zinc-800 bg-zinc-900 hover:bg-zinc-800 text-zinc-300 text-xs font-semibold transition"
                    title="Refresh bindings & attached fleet services"
                  >
                    <RefreshCw className={`w-3.5 h-3.5 ${isLoadingFleet ? 'animate-spin text-emerald-400' : 'text-zinc-400'}`} />
                    Refresh
                  </button>
                  <button
                    type="button"
                    onClick={() => {
                      handleTypeChange('kv_namespace');
                      setShowAddBindingModal(true);
                    }}
                    className="flex items-center gap-1.5 px-3.5 py-1.5 rounded-lg bg-emerald-500 hover:bg-emerald-600 text-zinc-950 text-xs font-bold transition shadow-sm"
                  >
                    <Plus className="w-3.5 h-3.5" />
                    Add Binding
                  </button>
                </div>
              </div>
            </div>

            {/* Section 1: Injected Cloudflare Resource Bindings (env.*) */}
            <div className="space-y-4">
              <div className="flex items-center justify-between">
                <div>
                  <h3 className="text-sm font-bold text-zinc-200 flex items-center gap-2">
                    <Database className="w-4 h-4 text-emerald-400" />
                    Injected Cloudflare Resource Bindings
                  </h3>
                  <p className="text-xs text-zinc-500">
                    Services injected into this Worker's runtime context under <code className="font-mono text-emerald-400 text-[11px]">env.&lt;NAME&gt;</code>.
                  </p>
                </div>
                <button
                  type="button"
                  onClick={() => {
                    handleTypeChange('kv_namespace');
                    setShowAddBindingModal(true);
                  }}
                  className="text-xs font-semibold text-emerald-400 hover:text-emerald-300 flex items-center gap-1 transition"
                >
                  <Plus className="w-3.5 h-3.5" />
                  Add Resource Binding
                </button>
              </div>

              {bindings.length === 0 ? (
                <div className="p-8 rounded-xl border border-dashed border-zinc-800 bg-zinc-950/40 text-center space-y-3">
                  <div className="w-10 h-10 rounded-full bg-zinc-900 border border-zinc-800 mx-auto flex items-center justify-center text-zinc-500">
                    <Database className="w-5 h-5" />
                  </div>
                  <div className="space-y-1">
                    <h4 className="text-xs font-bold text-zinc-300">No Injected Resource Bindings</h4>
                    <p className="text-xs text-zinc-500 max-w-md mx-auto">
                      Bind KV namespaces, D1 databases, R2 storage buckets, queues, or workflows to access them inside your worker code.
                    </p>
                  </div>
                  <button
                    type="button"
                    onClick={() => {
                      handleTypeChange('kv_namespace');
                      setShowAddBindingModal(true);
                    }}
                    className="inline-flex items-center gap-1.5 px-3.5 py-1.5 rounded-lg border border-zinc-800 bg-zinc-900 hover:bg-zinc-800 text-zinc-200 text-xs font-semibold transition"
                  >
                    <Plus className="w-3.5 h-3.5 text-emerald-400" />
                    Add First Binding
                  </button>
                </div>
              ) : (
                <div className="grid grid-cols-1 md:grid-cols-2 gap-3.5">
                  {bindings.map((b, idx) => {
                    const tabKey = getServiceTabForBindingType(b.type);
                    const label = getServiceLabelForBindingType(b.type);
                    return (
                      <div
                        key={idx}
                        className="bg-zinc-900/60 border border-zinc-800/80 rounded-xl p-4 flex flex-col justify-between gap-3 hover:border-zinc-700/80 transition shadow-sm"
                      >
                        <div className="flex items-start justify-between gap-2">
                          <div className="flex items-center gap-2">
                            {renderBindingTypeIcon(b.type)}
                            <span className="text-xs font-bold text-zinc-200">{label}</span>
                          </div>
                          <button
                            type="button"
                            onClick={() => handleDeleteBinding(idx)}
                            disabled={isSavingBindings}
                            title="Delete this binding"
                            className="p-1 rounded text-zinc-500 hover:text-rose-400 hover:bg-rose-950/30 transition"
                          >
                            <Trash2 className="w-3.5 h-3.5" />
                          </button>
                        </div>

                        <div className="space-y-1.5 bg-zinc-950/60 rounded-lg p-2.5 border border-zinc-800/60">
                          <div className="flex items-center justify-between text-[11px]">
                            <span className="text-zinc-500">Variable:</span>
                            <span className="font-mono font-bold text-emerald-400">env.{b.name}</span>
                          </div>
                          <div className="flex items-center justify-between text-[11px]">
                            <span className="text-zinc-500">Target Resource:</span>
                            <span className="font-mono text-zinc-300 truncate max-w-[180px]" title={b.resourceId || 'Default'}>
                              {b.resourceId || 'Default'}
                            </span>
                          </div>
                        </div>

                        <button
                          type="button"
                          onClick={() => onNavigateToService?.(tabKey, b.resourceId)}
                          className="w-full flex items-center justify-center gap-1.5 py-2 px-3 rounded-lg bg-zinc-800 hover:bg-zinc-700 text-zinc-200 text-xs font-semibold transition"
                        >
                          <span>Open in {getServicePageName(b.type)}</span>
                          <ArrowRight className="w-3.5 h-3.5 text-emerald-400" />
                        </button>
                      </div>
                    );
                  })}
                </div>
              )}
            </div>

            {/* Section 2: Attached Fleet Services (Reverse Bindings) */}
            <div className="space-y-4 pt-4 border-t border-zinc-800/80">
              <div>
                <h3 className="text-sm font-bold text-zinc-200 flex items-center gap-2">
                  <Cpu className="w-4 h-4 text-purple-400" />
                  Attached Fleet Services & Triggers
                </h3>
                <p className="text-xs text-zinc-500">
                  Services across the Cubit fleet configured to invoke, dispatch messages to, or host logic inside this Worker.
                </p>
              </div>

              {totalAttachedFleet === 0 ? (
                <div className="p-6 rounded-xl border border-dashed border-zinc-800 bg-zinc-950/40 text-center space-y-2">
                  <p className="text-xs text-zinc-500">
                    No fleet services currently target this Worker. When a Cron Trigger, Queue Consumer, Durable Object, or Workflow targets <code className="font-mono text-zinc-400">{app.name}</code> (or ID <code className="font-mono text-zinc-400">{app.id}</code>), it will automatically link here with direct navigation.
                  </p>
                </div>
              ) : (
                <div className="grid grid-cols-1 md:grid-cols-2 gap-3.5">
                  {/* Cron Triggers */}
                  {attachedCron.map((cron, idx) => (
                    <div
                      key={`cron-${idx}`}
                      className="bg-zinc-900/60 border border-zinc-800/80 rounded-xl p-4 flex flex-col justify-between gap-3 hover:border-zinc-700/80 transition"
                    >
                      <div className="flex items-center justify-between">
                        <div className="flex items-center gap-2">
                          <Clock className="w-4 h-4 text-amber-400" />
                          <span className="text-xs font-bold text-zinc-200">Cron Trigger</span>
                        </div>
                        <span className="text-[10px] font-mono px-2 py-0.5 rounded bg-amber-950/80 text-amber-300 border border-amber-800/60">
                          {cron.cronExpression || cron.cron || 'Scheduled'}
                        </span>
                      </div>
                      <div className="space-y-1 bg-zinc-950/60 rounded-lg p-2.5 border border-zinc-800/60 text-[11px]">
                        <div className="flex justify-between">
                          <span className="text-zinc-500">Name:</span>
                          <span className="font-semibold text-zinc-200">{cron.name}</span>
                        </div>
                        <div className="flex justify-between">
                          <span className="text-zinc-500">Status:</span>
                          <span className="text-emerald-400 font-semibold">{cron.status || (cron.active ? 'active' : 'inactive')}</span>
                        </div>
                      </div>
                      <button
                        type="button"
                        onClick={() => onNavigateToService?.('cron', cron.id)}
                        className="w-full flex items-center justify-center gap-1.5 py-2 px-3 rounded-lg bg-zinc-800 hover:bg-zinc-700 text-zinc-200 text-xs font-semibold transition"
                      >
                        <span>Open in Cron Triggers</span>
                        <ArrowRight className="w-3.5 h-3.5 text-amber-400" />
                      </button>
                    </div>
                  ))}

                  {/* Queues (as Consumer) */}
                  {attachedQueues.map((queue, idx) => (
                    <div
                      key={`queue-${idx}`}
                      className="bg-zinc-900/60 border border-zinc-800/80 rounded-xl p-4 flex flex-col justify-between gap-3 hover:border-zinc-700/80 transition"
                    >
                      <div className="flex items-center justify-between">
                        <div className="flex items-center gap-2">
                          <Inbox className="w-4 h-4 text-emerald-400" />
                          <span className="text-xs font-bold text-zinc-200">Queue Consumer</span>
                        </div>
                        <span className="text-[10px] font-mono px-2 py-0.5 rounded bg-emerald-950/80 text-emerald-300 border border-emerald-800/60">
                          Max Retries: {queue.maxRetries}
                        </span>
                      </div>
                      <div className="space-y-1 bg-zinc-950/60 rounded-lg p-2.5 border border-zinc-800/60 text-[11px]">
                        <div className="flex justify-between">
                          <span className="text-zinc-500">Queue Name:</span>
                          <span className="font-semibold text-zinc-200">{queue.name}</span>
                        </div>
                        <div className="flex justify-between">
                          <span className="text-zinc-500">Backlog:</span>
                          <span className="text-zinc-300">{queue.backlogCount ?? queue.messageBacklog ?? 0} messages</span>
                        </div>
                      </div>
                      <button
                        type="button"
                        onClick={() => onNavigateToService?.('queues', queue.id)}
                        className="w-full flex items-center justify-center gap-1.5 py-2 px-3 rounded-lg bg-zinc-800 hover:bg-zinc-700 text-zinc-200 text-xs font-semibold transition"
                      >
                        <span>Open in Queues</span>
                        <ArrowRight className="w-3.5 h-3.5 text-emerald-400" />
                      </button>
                    </div>
                  ))}

                  {/* Durable Objects */}
                  {attachedDO.map((doCls, idx) => (
                    <div
                      key={`do-${idx}`}
                      className="bg-zinc-900/60 border border-zinc-800/80 rounded-xl p-4 flex flex-col justify-between gap-3 hover:border-zinc-700/80 transition"
                    >
                      <div className="flex items-center justify-between">
                        <div className="flex items-center gap-2">
                          <Box className="w-4 h-4 text-purple-400" />
                          <span className="text-xs font-bold text-zinc-200">Durable Object Class</span>
                        </div>
                        <span className="text-[10px] font-mono px-2 py-0.5 rounded bg-purple-950/80 text-purple-300 border border-purple-800/60">
                          {doCls.instancesCount ?? 0} instances
                        </span>
                      </div>
                      <div className="space-y-1 bg-zinc-950/60 rounded-lg p-2.5 border border-zinc-800/60 text-[11px]">
                        <div className="flex justify-between">
                          <span className="text-zinc-500">Class Name:</span>
                          <span className="font-semibold text-zinc-200">{doCls.name}</span>
                        </div>
                        <div className="flex justify-between">
                          <span className="text-zinc-500">Facets:</span>
                          <span className="text-zinc-300">{doCls.facets?.map((f: any) => f.name).join(', ') || 'None'}</span>
                        </div>
                      </div>
                      <button
                        type="button"
                        onClick={() => onNavigateToService?.('durable-objects', doCls.id)}
                        className="w-full flex items-center justify-center gap-1.5 py-2 px-3 rounded-lg bg-zinc-800 hover:bg-zinc-700 text-zinc-200 text-xs font-semibold transition"
                      >
                        <span>Open in Durable Objects</span>
                        <ArrowRight className="w-3.5 h-3.5 text-purple-400" />
                      </button>
                    </div>
                  ))}

                  {/* Workflows */}
                  {attachedWorkflows.map((wf, idx) => (
                    <div
                      key={`wf-${idx}`}
                      className="bg-zinc-900/60 border border-zinc-800/80 rounded-xl p-4 flex flex-col justify-between gap-3 hover:border-zinc-700/80 transition"
                    >
                      <div className="flex items-center justify-between">
                        <div className="flex items-center gap-2">
                          <GitMerge className="w-4 h-4 text-pink-400" />
                          <span className="text-xs font-bold text-zinc-200">Workflow Target</span>
                        </div>
                        <span className="text-[10px] font-mono px-2 py-0.5 rounded bg-pink-950/80 text-pink-300 border border-pink-800/60">
                          {wf.steps?.length || 0} steps
                        </span>
                      </div>
                      <div className="space-y-1 bg-zinc-950/60 rounded-lg p-2.5 border border-zinc-800/60 text-[11px]">
                        <div className="flex justify-between">
                          <span className="text-zinc-500">Workflow Name:</span>
                          <span className="font-semibold text-zinc-200">{wf.name}</span>
                        </div>
                        <div className="flex justify-between">
                          <span className="text-zinc-500">Steps:</span>
                          <span className="text-zinc-300 truncate max-w-[180px]">{wf.steps?.join(' ➔ ') || 'None'}</span>
                        </div>
                      </div>
                      <button
                        type="button"
                        onClick={() => onNavigateToService?.('workflows', wf.id)}
                        className="w-full flex items-center justify-center gap-1.5 py-2 px-3 rounded-lg bg-zinc-800 hover:bg-zinc-700 text-zinc-200 text-xs font-semibold transition"
                      >
                        <span>Open in Workflows</span>
                        <ArrowRight className="w-3.5 h-3.5 text-pink-400" />
                      </button>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>
        )}
      </div>

      {/* Add Resource Binding Modal */}
      {showAddBindingModal && (
        <div className="fixed inset-0 bg-black/70 backdrop-blur-sm flex items-center justify-center p-4 z-50">
          <div className="bg-zinc-900 border border-zinc-800 rounded-2xl max-w-md w-full p-6 space-y-5 shadow-2xl">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <Link2 className="w-5 h-5 text-emerald-400" />
                <h3 className="text-lg font-bold text-zinc-100">Add Resource Binding</h3>
              </div>
              <button
                type="button"
                onClick={() => setShowAddBindingModal(false)}
                className="text-zinc-500 hover:text-zinc-300 transition"
              >
                <X className="w-5 h-5" />
              </button>
            </div>

            <form onSubmit={handleAddBinding} className="space-y-4">
              {/* Type selector */}
              <div className="space-y-1.5">
                <label className="text-xs font-semibold text-zinc-400">Binding Type</label>
                <select
                  value={newBindingType}
                  onChange={e => handleTypeChange(e.target.value as ResourceBindingType)}
                  className="w-full bg-zinc-950 border border-zinc-800 rounded-lg p-2.5 text-xs text-zinc-200 outline-none focus:border-emerald-500"
                >
                  <option value="kv_namespace">KV Namespace (Key-Value Storage)</option>
                  <option value="d1_database">D1 Database (Serverless SQLite)</option>
                  <option value="r2_bucket">R2 Bucket (Object Storage)</option>
                  <option value="queue">Queue (Message Queue Producer)</option>
                  <option value="workflow">Workflow (Stateful Durable Execution)</option>
                </select>
              </div>

              {/* Resource Target */}
              <div className="space-y-1.5">
                <label className="text-xs font-semibold text-zinc-400">Target Fleet Resource</label>
                <select
                  value={newBindingResourceId}
                  onChange={e => setNewBindingResourceId(e.target.value)}
                  className="w-full bg-zinc-950 border border-zinc-800 rounded-lg p-2.5 text-xs text-zinc-200 outline-none focus:border-emerald-500"
                >
                  <option value="">-- Select fleet resource or custom --</option>
                  {newBindingType === 'kv_namespace' && fleetKV.map(k => (
                    <option key={k.id} value={k.name || k.id}>{k.name} ({k.id.substring(0, 8)}...)</option>
                  ))}
                  {newBindingType === 'd1_database' && fleetD1.map(d => (
                    <option key={d.id} value={d.name || d.id}>{d.name} ({d.id.substring(0, 8)}...)</option>
                  ))}
                  {newBindingType === 'r2_bucket' && fleetR2.map(b => (
                    <option key={b.name} value={b.name}>{b.name}</option>
                  ))}
                  {newBindingType === 'queue' && fleetQueues.map(q => (
                    <option key={q.id} value={q.name || q.id}>{q.name}</option>
                  ))}
                  {newBindingType === 'workflow' && fleetWorkflows.map(w => (
                    <option key={w.id} value={w.name || w.id}>{w.name}</option>
                  ))}
                  <option value="__custom__">-- Enter custom ID / name --</option>
                </select>

                {newBindingResourceId === '__custom__' && (
                  <input
                    type="text"
                    placeholder="Enter custom resource name or ID"
                    value={customResourceId}
                    onChange={e => setCustomResourceId(e.target.value)}
                    className="w-full mt-2 bg-zinc-950 border border-zinc-800 rounded-lg p-2.5 text-xs text-zinc-200 outline-none focus:border-emerald-500 font-mono"
                    required
                  />
                )}
              </div>

              {/* Binding variable name */}
              <div className="space-y-1.5">
                <label className="text-xs font-semibold text-zinc-400">Environment Variable Name</label>
                <div className="flex items-center gap-2">
                  <span className="text-xs font-mono text-zinc-500">env.</span>
                  <input
                    type="text"
                    placeholder="MY_BINDING"
                    value={newBindingName}
                    onChange={e => setNewBindingName(e.target.value.toUpperCase().replace(/[^A-Z0-9_]/g, ''))}
                    className="flex-1 bg-zinc-950 border border-zinc-800 rounded-lg p-2.5 text-xs text-zinc-200 font-mono outline-none focus:border-emerald-500 uppercase"
                    required
                  />
                </div>
                <p className="text-[11px] text-zinc-500">
                  Access in worker: <code className="font-mono text-emerald-400">env.{newBindingName || 'MY_BINDING'}</code>
                </p>
              </div>

              <div className="flex items-center justify-end gap-3 pt-3 border-t border-zinc-800">
                <button
                  type="button"
                  onClick={() => setShowAddBindingModal(false)}
                  className="px-4 py-2 rounded-lg text-xs font-semibold text-zinc-400 hover:text-zinc-200 hover:bg-zinc-800 transition"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  disabled={isSavingBindings || !newBindingName.trim()}
                  className="px-5 py-2 rounded-lg text-xs font-bold bg-emerald-500 hover:bg-emerald-600 disabled:opacity-50 text-zinc-950 transition shadow-sm"
                >
                  {isSavingBindings ? 'Adding...' : 'Add Binding'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}

// Helpers for bindings
export function isServiceAttachedToApp(target: string | undefined | null, app: { id: string; name: string; subdomain?: string }): boolean {
  if (!target) return false;
  return target === app.id || target === app.name || (!!app.subdomain && target === app.subdomain);
}

export function getServiceTabForBindingType(type: ResourceBindingType): ActiveTab {
  switch (type) {
    case 'kv_namespace':
      return 'kv';
    case 'd1_database':
      return 'd1';
    case 'r2_bucket':
      return 'r2';
    case 'queue':
      return 'queues';
    case 'workflow':
      return 'workflows';
    default:
      return 'apps';
  }
}

export function getServiceLabelForBindingType(type: ResourceBindingType): string {
  switch (type) {
    case 'kv_namespace':
      return 'KV Namespace';
    case 'd1_database':
      return 'D1 Database';
    case 'r2_bucket':
      return 'R2 Bucket';
    case 'queue':
      return 'Queue Producer';
    case 'workflow':
      return 'Workflow';
    default:
      return type;
  }
}

export function getServicePageName(type: ResourceBindingType): string {
  switch (type) {
    case 'kv_namespace':
      return 'KV Namespaces';
    case 'd1_database':
      return 'D1 Databases';
    case 'r2_bucket':
      return 'R2 Buckets';
    case 'queue':
      return 'Queues';
    case 'workflow':
      return 'Workflows';
    default:
      return 'Service';
  }
}

function renderBindingTypeIcon(type: ResourceBindingType) {
  switch (type) {
    case 'kv_namespace':
      return <Database className="w-4 h-4 text-amber-400" />;
    case 'd1_database':
      return <Database className="w-4 h-4 text-sky-400" />;
    case 'r2_bucket':
      return <HardDrive className="w-4 h-4 text-purple-400" />;
    case 'queue':
      return <Inbox className="w-4 h-4 text-emerald-400" />;
    case 'workflow':
      return <GitMerge className="w-4 h-4 text-pink-400" />;
    default:
      return <Link2 className="w-4 h-4 text-zinc-400" />;
  }
}
