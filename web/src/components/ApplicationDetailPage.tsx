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
  Activity,
  Radio,
  Pause,
  Flame,
  AlertTriangle,
  Sliders,
  ChevronDown,
  ChevronRight,
  Search,
  FileText,
  Braces,
  ListFilter,
  RotateCcw,
  Download,
  Lock,
  Edit2,
} from 'lucide-react';
import type { Application, ResourceBinding, ResourceBindingType, ApplicationMetrics, RequestLogEvent, Deployment } from '../api/model';
import type { ActiveTab } from '../App';
import { useListDeployments, useRollbackApplication } from '../api/generated/deployments/deployments';
import { useUpdateApplication, useDeleteApplication, useGetApplicationMetrics } from '../api/generated/applications/applications';
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

export type DetailTab = 'overview' | 'code' | 'builds' | 'triggers' | 'bindings' | 'settings';

export interface ApplicationDetailPageProps {
  app: Application;
  allApps?: Application[];
  onBack: () => void;
  onDeploy: (appId: string) => Promise<void> | void;
  isDeploying: boolean;
  onTestApp: (app: Application) => void;
  onRefreshApps: () => void;
  onNavigateToService?: (targetTab: ActiveTab, resourceId?: string) => void;
  initialTab?: DetailTab;
  onTabChange?: (tab: DetailTab) => void;
}

export function ApplicationDetailPage({
  app,
  allApps = [],
  onBack,
  onDeploy,
  isDeploying,
  onTestApp,
  onRefreshApps,
  onNavigateToService,
  initialTab,
  onTabChange,
}: ApplicationDetailPageProps) {
  const [activeTab, setActiveTab] = useState<DetailTab>(initialTab || 'overview');

  useEffect(() => {
    if (initialTab && initialTab !== activeTab) {
      setActiveTab(initialTab);
    }
  }, [initialTab]);

  const switchTab = (t: DetailTab) => {
    setActiveTab(t);
    onTabChange?.(t);
  };

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

  // Rollback and Bundle states
  const rollbackMutation = useRollbackApplication();
  const [showRollbackModal, setShowRollbackModal] = useState(false);
  const [rollbackTargetDep, setRollbackTargetDep] = useState<Deployment | null>(null);
  const [isRollingBack, setIsRollingBack] = useState(false);
  const [rollbackStatus, setRollbackStatus] = useState<'idle' | 'success' | 'error'>('idle');
  const [rollbackError, setRollbackError] = useState('');
  const [isDownloadingBundle, setIsDownloadingBundle] = useState(false);

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

  // Environment variables & Secrets suite state
  const [envVars, setEnvVars] = useState<Array<{ key: string; value: string; isSecret: boolean }>>(
    app.envVars ? app.envVars.map(e => ({ key: e.key || (e as any).Key || '', value: e.value || (e as any).Value || '', isSecret: !!(e.isSecret || (e as any).IsSecret) })) : []
  );
  const [showSecretValues, setShowSecretValues] = useState<Record<number, boolean>>({});
  const [isSavingEnv, setIsSavingEnv] = useState(false);
  const [envSaveStatus, setEnvSaveStatus] = useState<'idle' | 'saved' | 'error'>('idle');

  // Enhanced Variables & Secrets suite filters and modals
  const [varSearchTerm, setVarSearchTerm] = useState('');
  const [varTypeFilter, setVarTypeFilter] = useState<'ALL' | 'PLAIN' | 'SECRET'>('ALL');
  const [showAddVarModal, setShowAddVarModal] = useState(false);
  const [newVarKey, setNewVarKey] = useState('');
  const [newVarValue, setNewVarValue] = useState('');
  const [newVarIsSecret, setNewVarIsSecret] = useState(false);
  const [showNewVarValue, setShowNewVarValue] = useState(false);
  const [editingVarIndex, setEditingVarIndex] = useState<number | null>(null);
  const [copiedVarKey, setCopiedVarKey] = useState<string | null>(null);
  const [copiedVarValue, setCopiedVarValue] = useState<string | null>(null);
  const [varDeleteConfirmIndex, setVarDeleteConfirmIndex] = useState<number | null>(null);

  useEffect(() => {
    if (app.envVars) {
      setEnvVars(
        app.envVars.map(e => ({
          key: e.key || (e as any).Key || '',
          value: e.value || (e as any).Value || '',
          isSecret: !!(e.isSecret || (e as any).IsSecret),
        }))
      );
    }
  }, [app.envVars, app.id]);

  const updateAppMutation = useUpdateApplication();
  const deleteAppMutation = useDeleteApplication();
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);
  const [isDeletingApp, setIsDeletingApp] = useState(false);

  // Application Metrics Query (polled every 5s)
  const { data: metricsData } = useGetApplicationMetrics(app.id, {
    query: {
      refetchInterval: 5000,
    },
  });
  const metrics = metricsData as ApplicationMetrics | undefined;

  // Builds & Live Logs Sub-View State
  const [buildsViewMode, setBuildsViewMode] = useState<'history' | 'tail'>('history');
  const [liveLogs, setLiveLogs] = useState<RequestLogEvent[]>([]);
  const [isLiveStreaming, setIsLiveStreaming] = useState(true);
  const [liveConnected, setLiveConnected] = useState(false);
  const [logSearchTerm, setLogSearchTerm] = useState('');
  const [logMethodFilter, setLogMethodFilter] = useState<'ALL' | 'GET' | 'POST' | 'PUT' | 'DELETE'>('ALL');
  const [logStatusFilter, setLogStatusFilter] = useState<'ALL' | '2xx' | '4xx' | '5xx'>('ALL');
  const [expandedLogId, setExpandedLogId] = useState<string | null>(null);
  const [activeLogDetailTab, setActiveLogDetailTab] = useState<'headers' | 'payload' | 'logs' | 'json'>('headers');
  const [copiedLogSection, setCopiedLogSection] = useState<string | null>(null);

  const handleCopyLogSection = (text: string, sectionKey: string) => {
    navigator.clipboard.writeText(text);
    setCopiedLogSection(sectionKey);
    setTimeout(() => {
      setCopiedLogSection(prev => (prev === sectionKey ? null : prev));
    }, 2000);
  };

  const formatJsonPayload = (raw: any): string => {
    if (raw === undefined || raw === null) return '';
    if (typeof raw === 'object') {
      try {
        return JSON.stringify(raw, null, 2);
      } catch {
        return String(raw);
      }
    }
    if (typeof raw === 'string') {
      try {
        const parsed = JSON.parse(raw);
        return JSON.stringify(parsed, null, 2);
      } catch {
        return raw;
      }
    }
    return String(raw);
  };

  useEffect(() => {
    if (activeTab !== 'builds' || buildsViewMode !== 'tail' || !isLiveStreaming) {
      setLiveConnected(false);
      return;
    }
    const es = new EventSource(`/api/v1/applications/${app.id}/logs/stream`);
    es.onopen = () => {
      setLiveConnected(true);
    };
    es.onmessage = (e) => {
      try {
        const item: RequestLogEvent = JSON.parse(e.data);
        setLiveLogs(prev => [item, ...prev].slice(0, 200));
      } catch (err) {
        console.error('Error parsing live log event:', err);
      }
    };
    es.onerror = () => {
      setLiveConnected(false);
    };
    return () => {
      es.close();
      setLiveConnected(false);
    };
  }, [activeTab, buildsViewMode, isLiveStreaming, app.id]);

  // Cron Triggers State
  const [showAddCronModal, setShowAddCronModal] = useState(false);
  const [newCronName, setNewCronName] = useState('');
  const [newCronExpr, setNewCronExpr] = useState('*/15 * * * *');
  const [isCreatingCron, setIsCreatingCron] = useState(false);
  const [runningCronId, setRunningCronId] = useState<string | null>(null);

  const handleCreateCron = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newCronName.trim() || !newCronExpr.trim()) return;
    setIsCreatingCron(true);
    try {
      const res = await fetch('/api/v1/cron', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          name: newCronName.trim(),
          cron: newCronExpr.trim(),
          target_app_id: app.id,
        }),
      });
      if (!res.ok) throw new Error('Failed to create cron trigger');
      setShowAddCronModal(false);
      setNewCronName('');
      setNewCronExpr('*/15 * * * *');
      await fetchFleetServices();
    } catch (err: any) {
      alert(err.message || 'Failed to create cron trigger');
    } finally {
      setIsCreatingCron(false);
    }
  };

  const handleRunCronNow = async (cronId: string) => {
    setRunningCronId(cronId);
    try {
      await fetch(`/api/v1/cron/${cronId}/run`, { method: 'POST' });
      await fetchFleetServices();
    } catch (err) {
      console.error('Failed to trigger cron execution:', err);
    } finally {
      setRunningCronId(null);
    }
  };

  // Runtime & Compatibility Settings State
  const [compatDate, setCompatDate] = useState(app.compatibilityDate || '2024-04-03');
  const [nodejsCompat, setNodejsCompat] = useState((app.compatibilityFlags || []).includes('nodejs_compat'));
  const [memoryLimitMB, setMemoryLimitMB] = useState(app.memoryLimitMb || 128);
  const [maxDurationMs, setMaxDurationMs] = useState(app.maxDurationMs || 50);
  const [isSavingRuntime, setIsSavingRuntime] = useState(false);
  const [runtimeSaveStatus, setRuntimeSaveStatus] = useState<'idle' | 'saved' | 'error'>('idle');

  useEffect(() => {
    setCompatDate(app.compatibilityDate || '2024-04-03');
    setNodejsCompat((app.compatibilityFlags || []).includes('nodejs_compat'));
    setMemoryLimitMB(app.memoryLimitMb || 128);
    setMaxDurationMs(app.maxDurationMs || 50);
  }, [app.compatibilityDate, app.compatibilityFlags, app.memoryLimitMb, app.maxDurationMs]);

  const handleSaveRuntime = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsSavingRuntime(true);
    setRuntimeSaveStatus('idle');
    try {
      await updateAppMutation.mutateAsync({
        id: app.id,
        data: {
          compatibilityDate: compatDate.trim() || undefined,
          compatibilityFlags: nodejsCompat ? ['nodejs_compat'] : [],
          memoryLimitMb: Number(memoryLimitMB),
          maxDurationMs: Number(maxDurationMs),
        },
      });
      setRuntimeSaveStatus('saved');
      onRefreshApps();
      setTimeout(() => setRuntimeSaveStatus('idle'), 3000);
    } catch (err) {
      console.error('Failed saving runtime configuration:', err);
      setRuntimeSaveStatus('error');
    } finally {
      setIsSavingRuntime(false);
    }
  };

  const handleDeleteApp = async () => {
    setIsDeletingApp(true);
    try {
      await deleteAppMutation.mutateAsync({ id: app.id });
      onRefreshApps();
      onBack();
    } catch (err) {
      console.error('Failed deleting application:', err);
      alert('Failed to delete application');
    } finally {
      setIsDeletingApp(false);
      setShowDeleteConfirm(false);
    }
  };

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
    } else if (t === 'service') {
      setNewBindingName('MY_WORKER');
      const other = allApps.find(a => a.id !== app.id);
      setNewBindingResourceId(other?.name || other?.id || '');
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

  // Save Environment Variables list helper
  const handleSaveEnvVarsList = async (updatedList: Array<{ key: string; value: string; isSecret: boolean }>) => {
    setIsSavingEnv(true);
    setEnvSaveStatus('idle');
    try {
      const validEnvs = updatedList.filter(e => e.key.trim() !== '');
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
      setEnvVars(validEnvs);
      setEnvSaveStatus('saved');
      onRefreshApps();
      setTimeout(() => setEnvSaveStatus('idle'), 3000);
      return true;
    } catch (err) {
      console.error('Failed saving env vars:', err);
      setEnvSaveStatus('error');
      return false;
    } finally {
      setIsSavingEnv(false);
    }
  };

  const handleSaveEnvVars = async () => {
    await handleSaveEnvVarsList(envVars);
  };

  const handleCopyVar = (text: string, type: 'key' | 'value', id: string) => {
    navigator.clipboard.writeText(text);
    if (type === 'key') {
      setCopiedVarKey(id);
      setTimeout(() => setCopiedVarKey(null), 2000);
    } else {
      setCopiedVarValue(id);
      setTimeout(() => setCopiedVarValue(null), 2000);
    }
  };

  const handleAddVariableSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    const cleanKey = newVarKey.trim().toUpperCase().replace(/[^A-Z0-9_]/g, '_');
    if (!cleanKey) return;
    const existingIndex = envVars.findIndex(v => v.key.toUpperCase() === cleanKey);
    let updated: Array<{ key: string; value: string; isSecret: boolean }>;
    if (existingIndex >= 0) {
      updated = envVars.map((v, i) =>
        i === existingIndex ? { key: cleanKey, value: newVarValue, isSecret: newVarIsSecret } : v
      );
    } else {
      updated = [...envVars, { key: cleanKey, value: newVarValue, isSecret: newVarIsSecret }];
    }
    const success = await handleSaveEnvVarsList(updated);
    if (success) {
      setShowAddVarModal(false);
      setNewVarKey('');
      setNewVarValue('');
      setNewVarIsSecret(false);
      setShowNewVarValue(false);
    }
  };

  const handleDeleteVariable = async (index: number) => {
    const updated = envVars.filter((_, i) => i !== index);
    await handleSaveEnvVarsList(updated);
    setVarDeleteConfirmIndex(null);
  };

  const updateEnvVarRow = (index: number, field: 'key' | 'value' | 'isSecret', val: any) => {
    setEnvVars(envVars.map((row, i) => (i === index ? { ...row, [field]: val } : row)));
  };

  // Rollback Deployment Execution
  const handleRollback = async () => {
    if (!rollbackTargetDep) return;
    setIsRollingBack(true);
    setRollbackStatus('idle');
    setRollbackError('');
    try {
      await rollbackMutation.mutateAsync({
        id: app.id,
        data: {
          deploymentId: rollbackTargetDep.id,
        },
      });
      setRollbackStatus('success');
      await refetchDeployments();
      onRefreshApps();
      setTimeout(() => {
        setShowRollbackModal(false);
        setRollbackStatus('idle');
        setRollbackTargetDep(null);
      }, 1200);
    } catch (err: any) {
      console.error('Failed to rollback deployment:', err);
      setRollbackStatus('error');
      setRollbackError(err?.message || 'Failed to rollback deployment');
    } finally {
      setIsRollingBack(false);
    }
  };

  // Download Compiled Bundle
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
    } catch (err) {
      console.error('Download bundle error:', err);
      alert('Failed to download compiled worker bundle');
    } finally {
      setIsDownloadingBundle(false);
    }
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
                onClick={() => switchTab('bindings')}
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
        <div className="flex border-b border-zinc-800 pt-2 gap-6 overflow-x-auto">
          <button
            type="button"
            onClick={() => switchTab('overview')}
            className={`pb-3 text-xs font-semibold flex items-center gap-2 border-b-2 transition shrink-0 ${
              activeTab === 'overview'
                ? 'border-emerald-500 text-emerald-400'
                : 'border-transparent text-zinc-400 hover:text-zinc-200'
            }`}
          >
            <Activity className="w-3.5 h-3.5" />
            Overview
          </button>

          <button
            type="button"
            onClick={() => switchTab('code')}
            className={`pb-3 text-xs font-semibold flex items-center gap-2 border-b-2 transition shrink-0 ${
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
            onClick={() => switchTab('builds')}
            className={`pb-3 text-xs font-semibold flex items-center gap-2 border-b-2 transition shrink-0 ${
              activeTab === 'builds'
                ? 'border-emerald-500 text-emerald-400'
                : 'border-transparent text-zinc-400 hover:text-zinc-200'
            }`}
          >
            <Terminal className="w-3.5 h-3.5" />
            Builds & Logs
            <span className="text-[10px] px-1.5 py-0.2 rounded-full bg-zinc-800 text-zinc-400">
              {deployments.length}
            </span>
          </button>

          <button
            type="button"
            onClick={() => switchTab('triggers')}
            className={`pb-3 text-xs font-semibold flex items-center gap-2 border-b-2 transition shrink-0 ${
              activeTab === 'triggers'
                ? 'border-emerald-500 text-emerald-400'
                : 'border-transparent text-zinc-400 hover:text-zinc-200'
            }`}
          >
            <Zap className="w-3.5 h-3.5" />
            Triggers
            <span className="text-[10px] px-1.5 py-0.2 rounded-full bg-zinc-800 text-zinc-400">
              {appDomains.length + 1 + attachedCron.length + attachedQueues.length}
            </span>
          </button>

          <button
            type="button"
            onClick={() => switchTab('bindings')}
            className={`pb-3 text-xs font-semibold flex items-center gap-2 border-b-2 transition shrink-0 ${
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
            onClick={() => switchTab('settings')}
            className={`pb-3 text-xs font-semibold flex items-center gap-2 border-b-2 transition shrink-0 ${
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
        {/* Tab 1: Overview */}
        {activeTab === 'overview' && (
          <div className="space-y-6 max-w-6xl">
            {/* Top Metrics Cards */}
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
              {/* Total Invocations */}
              <div className="bg-zinc-900/40 border border-zinc-800/80 rounded-xl p-4 space-y-2">
                <div className="flex items-center justify-between text-xs text-zinc-400">
                  <span className="font-semibold uppercase tracking-wider">Total Requests</span>
                  <Activity className="w-4 h-4 text-emerald-400" />
                </div>
                <div className="flex items-baseline gap-2">
                  <span className="text-2xl font-bold font-mono text-zinc-100">
                    {(metrics?.totalRequests ?? 0).toLocaleString()}
                  </span>
                  <span className="text-[11px] text-zinc-500">invocations</span>
                </div>
                <p className="text-[11px] text-zinc-500">
                  Telemetry tracked via celld runtime
                </p>
              </div>

              {/* HTTP Status Breakdown */}
              <div className="bg-zinc-900/40 border border-zinc-800/80 rounded-xl p-4 space-y-2">
                <div className="flex items-center justify-between text-xs text-zinc-400">
                  <span className="font-semibold uppercase tracking-wider">Response Codes</span>
                  <ShieldCheck className="w-4 h-4 text-sky-400" />
                </div>
                <div className="flex items-center gap-3 text-xs font-mono">
                  <div className="flex items-center gap-1 text-emerald-400">
                    <span className="font-bold">{metrics?.status2xx ?? 0}</span>
                    <span className="text-[10px] text-zinc-500">2xx</span>
                  </div>
                  <div className="flex items-center gap-1 text-amber-400">
                    <span className="font-bold">{metrics?.status4xx ?? 0}</span>
                    <span className="text-[10px] text-zinc-500">4xx</span>
                  </div>
                  <div className="flex items-center gap-1 text-rose-400">
                    <span className="font-bold">{metrics?.status5xx ?? 0}</span>
                    <span className="text-[10px] text-zinc-500">5xx</span>
                  </div>
                </div>
                <div className="w-full h-1.5 bg-zinc-800 rounded-full overflow-hidden flex">
                  {(metrics?.totalRequests ?? 0) > 0 ? (
                    <>
                      <div
                        style={{ width: `${((metrics?.status2xx ?? 0) / (metrics?.totalRequests || 1)) * 100}%` }}
                        className="bg-emerald-500 h-full"
                        title={`2xx: ${metrics?.status2xx ?? 0}`}
                      />
                      <div
                        style={{ width: `${((metrics?.status4xx ?? 0) / (metrics?.totalRequests || 1)) * 100}%` }}
                        className="bg-amber-500 h-full"
                        title={`4xx: ${metrics?.status4xx ?? 0}`}
                      />
                      <div
                        style={{ width: `${((metrics?.status5xx ?? 0) / (metrics?.totalRequests || 1)) * 100}%` }}
                        className="bg-rose-500 h-full"
                        title={`5xx: ${metrics?.status5xx ?? 0}`}
                      />
                    </>
                  ) : (
                    <div className="bg-zinc-700 h-full w-full" />
                  )}
                </div>
              </div>

              {/* Latency */}
              <div className="bg-zinc-900/40 border border-zinc-800/80 rounded-xl p-4 space-y-2">
                <div className="flex items-center justify-between text-xs text-zinc-400">
                  <span className="font-semibold uppercase tracking-wider">Latency</span>
                  <Clock className="w-4 h-4 text-purple-400" />
                </div>
                <div className="flex items-baseline gap-3 font-mono">
                  <div>
                    <span className="text-xl font-bold text-zinc-100">
                      {metrics?.avgDurationMs ? Math.round(metrics.avgDurationMs) : 0}ms
                    </span>
                    <span className="text-[10px] text-zinc-500 ml-1">avg</span>
                  </div>
                  <span className="text-zinc-600">/</span>
                  <div>
                    <span className="text-xl font-bold text-purple-400">
                      {metrics?.p99DurationMs ? Math.round(metrics.p99DurationMs) : 0}ms
                    </span>
                    <span className="text-[10px] text-zinc-500 ml-1">p99</span>
                  </div>
                </div>
                <p className="text-[11px] text-zinc-500">
                  Isolate execution time
                </p>
              </div>

              {/* Last Activity */}
              <div className="bg-zinc-900/40 border border-zinc-800/80 rounded-xl p-4 space-y-2">
                <div className="flex items-center justify-between text-xs text-zinc-400">
                  <span className="font-semibold uppercase tracking-wider">Last Invocation</span>
                  <Flame className="w-4 h-4 text-amber-400" />
                </div>
                <div className="text-sm font-semibold font-mono text-zinc-200 truncate">
                  {(metrics as any)?.lastInvokedAt
                    ? new Date((metrics as any).lastInvokedAt).toLocaleTimeString()
                    : app.updatedAt
                      ? new Date(app.updatedAt).toLocaleTimeString()
                      : 'Never'}
                </div>
                <div className="flex items-center gap-1.5 text-[11px] text-zinc-500">
                  <span className="w-2 h-2 rounded-full bg-emerald-500 animate-pulse" />
                  <span>celld worker active</span>
                </div>
              </div>
            </div>

            {/* Architecture & Quotas Card */}
            <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
              <div className="md:col-span-2 bg-zinc-900/30 border border-zinc-800/80 rounded-xl p-5 space-y-4">
                <div className="flex items-center justify-between">
                  <h3 className="text-sm font-bold text-zinc-200">Worker Architecture & Configuration</h3>
                  <span className="text-xs text-zinc-500 font-mono">ID: {app.id.substring(0, 8)}...</span>
                </div>

                <div className="grid grid-cols-1 sm:grid-cols-2 gap-3.5 text-xs">
                  <div className="p-3 bg-zinc-950/60 rounded-lg border border-zinc-800/60 space-y-1">
                    <span className="text-[11px] text-zinc-500 uppercase tracking-wider block">Ingress Route</span>
                    <span className="font-mono text-emerald-400 font-semibold truncate block">{testUrl}</span>
                    <span className="text-[10px] text-zinc-500">Traefik mesh route</span>
                  </div>

                  <div className="p-3 bg-zinc-950/60 rounded-lg border border-zinc-800/60 space-y-1">
                    <span className="text-[11px] text-zinc-500 uppercase tracking-wider block">Source Deployment</span>
                    <span className="font-mono text-zinc-200 font-semibold block">
                      {app.sourceType === 'inline' ? 'Inline Worker (Web IDE)' : `Git Repo (${app.branch || 'main'})`}
                    </span>
                    <span className="text-[10px] text-zinc-500">Active version: v{activeBuildVersion}</span>
                  </div>

                  <div className="p-3 bg-zinc-950/60 rounded-lg border border-zinc-800/60 space-y-1">
                    <span className="text-[11px] text-zinc-500 uppercase tracking-wider block">Compatibility Date</span>
                    <span className="font-mono text-zinc-200 font-semibold block">{app.compatibilityDate || '2024-04-03'}</span>
                    <span className="text-[10px] text-zinc-500">
                      {app.compatibilityFlags?.includes('nodejs_compat') ? 'nodejs_compat enabled' : 'standard V8 runtime'}
                    </span>
                  </div>

                  <div className="p-3 bg-zinc-950/60 rounded-lg border border-zinc-800/60 space-y-1">
                    <span className="text-[11px] text-zinc-500 uppercase tracking-wider block">Resource Quotas</span>
                    <span className="font-mono text-zinc-200 font-semibold block">
                      {app.memoryLimitMb || 128} MB RAM / {app.maxDurationMs || 50} ms CPU
                    </span>
                    <span className="text-[10px] text-zinc-500">Isolate execution limits</span>
                  </div>
                </div>

                {/* Quick actions strip */}
                <div className="pt-2 flex flex-wrap gap-2.5">
                  <button
                    type="button"
                    onClick={() => switchTab('code')}
                    className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-zinc-800 hover:bg-zinc-700 text-zinc-200 text-xs font-semibold transition"
                  >
                    <Code className="w-3.5 h-3.5 text-emerald-400" />
                    Edit Code
                  </button>
                  <button
                    type="button"
                    onClick={() => {
                      switchTab('builds');
                      setBuildsViewMode('tail');
                    }}
                    className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-zinc-800 hover:bg-zinc-700 text-zinc-200 text-xs font-semibold transition"
                  >
                    <Radio className="w-3.5 h-3.5 text-sky-400" />
                    Live Tail
                  </button>
                  <button
                    type="button"
                    onClick={() => switchTab('triggers')}
                    className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-zinc-800 hover:bg-zinc-700 text-zinc-200 text-xs font-semibold transition"
                  >
                    <Zap className="w-3.5 h-3.5 text-amber-400" />
                    Triggers ({appDomains.length + 1 + attachedCron.length + attachedQueues.length})
                  </button>
                  <button
                    type="button"
                    onClick={() => switchTab('bindings')}
                    className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-zinc-800 hover:bg-zinc-700 text-zinc-200 text-xs font-semibold transition"
                  >
                    <Link2 className="w-3.5 h-3.5 text-indigo-400" />
                    Bindings ({totalBindingsCount})
                  </button>
                  <button
                    type="button"
                    onClick={() => switchTab('settings')}
                    className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-zinc-800 hover:bg-zinc-700 text-zinc-200 text-xs font-semibold transition"
                  >
                    <Settings className="w-3.5 h-3.5 text-zinc-400" />
                    Settings
                  </button>
                </div>
              </div>

              {/* Connected Services Card */}
              <div className="bg-zinc-900/30 border border-zinc-800/80 rounded-xl p-5 space-y-3 flex flex-col justify-between">
                <div>
                  <div className="flex items-center justify-between mb-2">
                    <h3 className="text-sm font-bold text-zinc-200">Connected Fleet</h3>
                    <span className="text-[10px] font-bold px-2 py-0.5 rounded-full bg-indigo-950 text-indigo-300 border border-indigo-800">
                      {totalBindingsCount} Bound
                    </span>
                  </div>
                  <p className="text-xs text-zinc-400 leading-relaxed">
                    Resources injected via <code className="font-mono text-emerald-400">env.*</code> or triggering this worker.
                  </p>

                  <div className="mt-4 space-y-2 text-xs">
                    <div className="flex items-center justify-between p-2 rounded-lg bg-zinc-950/60 border border-zinc-800/60">
                      <span className="text-zinc-400 flex items-center gap-1.5">
                        <Database className="w-3.5 h-3.5 text-amber-400" /> Injected Resources
                      </span>
                      <span className="font-mono font-bold text-zinc-200">{bindings.length}</span>
                    </div>
                    <div className="flex items-center justify-between p-2 rounded-lg bg-zinc-950/60 border border-zinc-800/60">
                      <span className="text-zinc-400 flex items-center gap-1.5">
                        <Clock className="w-3.5 h-3.5 text-sky-400" /> Cron Triggers
                      </span>
                      <span className="font-mono font-bold text-zinc-200">{attachedCron.length}</span>
                    </div>
                    <div className="flex items-center justify-between p-2 rounded-lg bg-zinc-950/60 border border-zinc-800/60">
                      <span className="text-zinc-400 flex items-center gap-1.5">
                        <Inbox className="w-3.5 h-3.5 text-emerald-400" /> Queue Consumers
                      </span>
                      <span className="font-mono font-bold text-zinc-200">{attachedQueues.length}</span>
                    </div>
                    <div className="flex items-center justify-between p-2 rounded-lg bg-zinc-950/60 border border-zinc-800/60">
                      <span className="text-zinc-400 flex items-center gap-1.5">
                        <Globe className="w-3.5 h-3.5 text-purple-400" /> Custom Domains
                      </span>
                      <span className="font-mono font-bold text-zinc-200">{appDomains.length}</span>
                    </div>
                  </div>
                </div>

                <button
                  type="button"
                  onClick={() => onTestApp(app)}
                  className="w-full mt-3 py-2 px-3 rounded-lg border border-emerald-900/60 bg-emerald-950/40 hover:bg-emerald-950/70 text-emerald-400 text-xs font-semibold transition flex items-center justify-center gap-1.5"
                >
                  <Send className="w-3.5 h-3.5" />
                  Test Worker Invocation
                </button>
              </div>
            </div>
          </div>
        )}

        {/* Tab 2: Builds & Live Logs */}
        {activeTab === 'builds' && (
          <div className="space-y-4 max-w-6xl">
            {/* View Mode Switcher */}
            <div className="flex items-center justify-between border-b border-zinc-800/80 pb-3">
              <div className="flex items-center gap-2">
                <button
                  type="button"
                  onClick={() => setBuildsViewMode('history')}
                  className={`px-3 py-1.5 rounded-lg text-xs font-semibold flex items-center gap-2 transition ${
                    buildsViewMode === 'history'
                      ? 'bg-zinc-800 text-zinc-100 border border-zinc-700'
                      : 'text-zinc-400 hover:text-zinc-200 hover:bg-zinc-900'
                  }`}
                >
                  <Terminal className="w-3.5 h-3.5" />
                  Deployment History ({deployments.length})
                </button>
                <button
                  type="button"
                  onClick={() => setBuildsViewMode('tail')}
                  className={`px-3 py-1.5 rounded-lg text-xs font-semibold flex items-center gap-2 transition ${
                    buildsViewMode === 'tail'
                      ? 'bg-emerald-950/80 text-emerald-400 border border-emerald-800'
                      : 'text-zinc-400 hover:text-zinc-200 hover:bg-zinc-900'
                  }`}
                >
                  <Radio className="w-3.5 h-3.5 text-emerald-400 animate-pulse" />
                  Live Request Tail (Realtime SSE)
                </button>
              </div>

              {buildsViewMode === 'tail' && (
                <div className="flex items-center gap-3">
                  <div className="flex items-center gap-1.5 text-[11px]">
                    <span className={`w-2 h-2 rounded-full ${liveConnected ? 'bg-emerald-500 animate-ping' : 'bg-zinc-600'}`} />
                    <span className={liveConnected ? 'text-emerald-400 font-mono' : 'text-zinc-500'}>
                      {liveConnected ? 'Connected' : 'Connecting...'}
                    </span>
                  </div>
                  <button
                    type="button"
                    onClick={() => setIsLiveStreaming(!isLiveStreaming)}
                    className="px-2.5 py-1 rounded bg-zinc-800 hover:bg-zinc-700 text-zinc-300 text-xs font-semibold transition flex items-center gap-1"
                  >
                    {isLiveStreaming ? <Pause className="w-3 h-3 text-amber-400" /> : <Play className="w-3 h-3 text-emerald-400" />}
                    {isLiveStreaming ? 'Pause' : 'Resume'}
                  </button>
                  <button
                    type="button"
                    onClick={() => setLiveLogs([])}
                    className="px-2.5 py-1 rounded bg-zinc-800 hover:bg-zinc-700 text-zinc-400 hover:text-zinc-200 text-xs font-semibold transition flex items-center gap-1"
                    title="Clear live tail logs"
                  >
                    <Trash2 className="w-3 h-3" />
                    Clear
                  </button>
                </div>
              )}
            </div>

            {buildsViewMode === 'history' ? (
              <div className="grid grid-cols-12 gap-6 min-h-[450px]">
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
                      const isLive = app.activeDeploymentId === dep.id;
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
                          <div className="flex items-center justify-between gap-1.5 flex-wrap">
                            <div className="flex items-center gap-1.5">
                              <span className="font-bold text-xs text-emerald-400 font-mono">
                                v{dep.buildVersion || 1}
                              </span>
                              {isLive && (
                                <span className="text-[10px] px-1.5 py-0.5 rounded-full font-bold uppercase bg-emerald-500/20 text-emerald-300 border border-emerald-500/50 flex items-center gap-1">
                                  <span className="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse" /> LIVE
                                </span>
                              )}
                            </div>
                            <span className={`text-[10px] px-2 py-0.5 rounded-full font-bold uppercase ${getDeploymentStatusBadge(dep.status)}`}>
                              {dep.status}
                            </span>
                          </div>
                          <div className="text-[11px] text-zinc-400 truncate font-mono">
                            {dep.commitHash ? dep.commitHash.slice(0, 7) : 'HEAD'} - {dep.commitMessage || 'Manual deployment'}
                          </div>
                          <div className="flex items-center justify-between text-[10px] text-zinc-500 font-mono">
                            <div className="flex items-center gap-1.5">
                              <Clock className="w-3 h-3 text-zinc-600" />
                              <span>{dateStr}</span>
                            </div>
                            {dep.bundleSize != null && dep.bundleSize > 0 && (
                              <span>{((dep.bundleSize) / 1024).toFixed(1)} KB</span>
                            )}
                          </div>
                        </div>
                      );
                    })
                  )}
                </div>

                {/* Build Logs Console (Right Column) */}
                <div className="col-span-8 flex flex-col space-y-2 min-h-0">
                  <div className="flex flex-wrap items-center justify-between gap-2 text-xs text-zinc-400 pb-1">
                    <div className="flex items-center gap-2.5">
                      <span className="font-semibold uppercase tracking-wider">
                        {selectedDep ? `Build Output (v${selectedDep.buildVersion || 1})` : 'Build Output'}
                      </span>
                      {selectedDep && (
                        selectedDep.id === app.activeDeploymentId ? (
                          <span className="text-[10px] px-2 py-0.5 rounded-full font-bold uppercase bg-emerald-500/20 text-emerald-300 border border-emerald-500/50 flex items-center gap-1">
                            <span className="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse" /> CURRENTLY ACTIVE
                          </span>
                        ) : (
                          <span className={`text-[10px] px-2 py-0.5 rounded-full font-bold uppercase ${getDeploymentStatusBadge(selectedDep.status)}`}>
                            {selectedDep.status}
                          </span>
                        )
                      )}
                    </div>

                    <div className="flex items-center gap-2">
                      {/* Download Bundle button */}
                      {selectedDep && (
                        <button
                          type="button"
                          onClick={() => handleDownloadBundle(selectedDep.id, selectedDep.buildVersion)}
                          disabled={isDownloadingBundle}
                          className="px-2.5 py-1 rounded-lg bg-zinc-900 hover:bg-zinc-800 text-zinc-300 border border-zinc-700/80 text-xs font-semibold flex items-center gap-1.5 transition"
                          title="Download compiled worker bundle"
                        >
                          <Download className="w-3.5 h-3.5 text-zinc-400" />
                          <span>{isDownloadingBundle ? 'Downloading...' : 'Download Bundle'}</span>
                        </button>
                      )}

                      {/* Rollback button if not active and not failed */}
                      {selectedDep && selectedDep.id !== app.activeDeploymentId && selectedDep.status !== 'failed' && (
                        <button
                          type="button"
                          onClick={() => {
                            setRollbackTargetDep(selectedDep);
                            setShowRollbackModal(true);
                          }}
                          className="px-2.5 py-1 rounded-lg bg-amber-500/10 hover:bg-amber-500/20 text-amber-300 border border-amber-500/40 text-xs font-semibold flex items-center gap-1.5 transition"
                          title="Rollback live isolate traffic to this deployment version"
                        >
                          <RotateCcw className="w-3.5 h-3.5 text-amber-400" />
                          <span>Rollback to v{selectedDep.buildVersion || 1}</span>
                        </button>
                      )}

                      {selectedDepId && (
                        <span className="font-mono text-[10px] text-zinc-500 hidden sm:inline">{selectedDepId.slice(0, 8)}...</span>
                      )}
                    </div>
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
            ) : (
              /* Cloudflare-Style Live Request Tail Console */
              <div className="bg-zinc-950/90 rounded-xl border border-zinc-800 p-4 space-y-3 font-mono text-xs shadow-inner min-h-[550px] flex flex-col">
                {/* Search & Filter Bar */}
                <div className="flex flex-wrap items-center justify-between gap-3 p-3 bg-zinc-900/60 rounded-lg border border-zinc-800/80">
                  <div className="flex flex-wrap items-center gap-2.5 flex-1 min-w-[280px]">
                    <div className="relative flex-1 max-w-sm">
                      <Search className="w-3.5 h-3.5 text-zinc-500 absolute left-2.5 top-1/2 -translate-y-1/2" />
                      <input
                        type="text"
                        value={logSearchTerm}
                        onChange={e => setLogSearchTerm(e.target.value)}
                        placeholder="Search path, URL, ray ID, IP, or status..."
                        className="w-full pl-8 pr-3 py-1.5 bg-black/70 border border-zinc-700/80 rounded-md text-zinc-200 placeholder-zinc-500 text-xs focus:outline-none focus:border-emerald-500 font-sans"
                      />
                      {logSearchTerm && (
                        <button
                          type="button"
                          onClick={() => setLogSearchTerm('')}
                          className="absolute right-2 top-1/2 -translate-y-1/2 text-zinc-400 hover:text-zinc-200"
                        >
                          <X className="w-3.5 h-3.5" />
                        </button>
                      )}
                    </div>

                    {/* Method Filter Pills */}
                    <div className="flex items-center gap-1 bg-black/60 p-0.5 rounded-md border border-zinc-800 text-[11px] font-sans">
                      {(['ALL', 'GET', 'POST', 'PUT', 'DELETE'] as const).map(m => (
                        <button
                          key={m}
                          type="button"
                          onClick={() => setLogMethodFilter(m)}
                          className={`px-2 py-0.5 rounded font-semibold transition ${
                            logMethodFilter === m
                              ? 'bg-zinc-800 text-zinc-100 shadow-xs'
                              : 'text-zinc-400 hover:text-zinc-200'
                          }`}
                        >
                          {m}
                        </button>
                      ))}
                    </div>

                    {/* Status Code Filter Pills */}
                    <div className="flex items-center gap-1 bg-black/60 p-0.5 rounded-md border border-zinc-800 text-[11px] font-sans">
                      {(['ALL', '2xx', '4xx', '5xx'] as const).map(st => (
                        <button
                          key={st}
                          type="button"
                          onClick={() => setLogStatusFilter(st)}
                          className={`px-2 py-0.5 rounded font-semibold transition ${
                            logStatusFilter === st
                              ? st === '2xx' ? 'bg-emerald-950 text-emerald-400 border border-emerald-800'
                                : st === '4xx' ? 'bg-amber-950 text-amber-400 border border-amber-800'
                                : st === '5xx' ? 'bg-rose-950 text-rose-400 border border-rose-800'
                                : 'bg-zinc-800 text-zinc-100'
                              : 'text-zinc-400 hover:text-zinc-200'
                          }`}
                        >
                          {st}
                        </button>
                      ))}
                    </div>
                  </div>

                  <div className="flex items-center gap-2 text-[11px] text-zinc-400 font-sans">
                    <span>Showing {liveLogs.filter(log => {
                      if (logMethodFilter !== 'ALL' && log.method.toUpperCase() !== logMethodFilter) return false;
                      const code = log.statusCode ?? (log as any).status ?? 200;
                      if (logStatusFilter === '2xx' && (code < 200 || code >= 300)) return false;
                      if (logStatusFilter === '4xx' && (code < 400 || code >= 500)) return false;
                      if (logStatusFilter === '5xx' && code < 500) return false;
                      if (logSearchTerm.trim()) {
                        const q = logSearchTerm.toLowerCase().trim();
                        const path = (log.path || '').toLowerCase();
                        const url = (log.url || '').toLowerCase();
                        const id = (log.id || '').toLowerCase();
                        const ip = (log.clientIp || '').toLowerCase();
                        const codeStr = String(code);
                        return path.includes(q) || url.includes(q) || id.includes(q) || ip.includes(q) || codeStr.includes(q);
                      }
                      return true;
                    }).length} of {liveLogs.length} events</span>
                    {(logSearchTerm || logMethodFilter !== 'ALL' || logStatusFilter !== 'ALL') && (
                      <button
                        type="button"
                        onClick={() => {
                          setLogSearchTerm('');
                          setLogMethodFilter('ALL');
                          setLogStatusFilter('ALL');
                        }}
                        className="text-xs text-sky-400 hover:underline ml-1"
                      >
                        Reset filters
                      </button>
                    )}
                  </div>
                </div>

                {/* Table Header */}
                <div className="grid grid-cols-12 gap-2 px-3 py-2 border-b border-zinc-800/80 text-zinc-500 text-[11px] font-semibold uppercase tracking-wider">
                  <span className="col-span-2">TIMESTAMP</span>
                  <span className="col-span-1">METHOD</span>
                  <span className="col-span-4">PATH / URL</span>
                  <span className="col-span-1 text-center">STATUS</span>
                  <span className="col-span-1 text-right">LATENCY</span>
                  <span className="col-span-1 text-center">CLIENT IP</span>
                  <span className="col-span-2 text-right">RAY ID</span>
                </div>

                {/* Log Event Stream */}
                <div className="flex-1 overflow-y-auto space-y-2 max-h-[600px] pr-1">
                  {(() => {
                    const filtered = liveLogs.filter(log => {
                      if (logMethodFilter !== 'ALL' && log.method.toUpperCase() !== logMethodFilter) return false;
                      const code = log.statusCode ?? (log as any).status ?? 200;
                      if (logStatusFilter === '2xx' && (code < 200 || code >= 300)) return false;
                      if (logStatusFilter === '4xx' && (code < 400 || code >= 500)) return false;
                      if (logStatusFilter === '5xx' && code < 500) return false;
                      if (logSearchTerm.trim()) {
                        const q = logSearchTerm.toLowerCase().trim();
                        const path = (log.path || '').toLowerCase();
                        const url = (log.url || '').toLowerCase();
                        const id = (log.id || '').toLowerCase();
                        const ip = (log.clientIp || '').toLowerCase();
                        const codeStr = String(code);
                        return path.includes(q) || url.includes(q) || id.includes(q) || ip.includes(q) || codeStr.includes(q);
                      }
                      return true;
                    });

                    if (filtered.length === 0) {
                      return (
                        <div className="py-16 text-center space-y-3 text-zinc-500 font-sans">
                          <Radio className="w-8 h-8 mx-auto text-emerald-500 animate-pulse" />
                          <p className="text-sm font-semibold text-zinc-300">
                            {liveLogs.length === 0 ? 'Live Request Tail Active' : 'No requests matched your filter'}
                          </p>
                          <p className="text-xs text-zinc-500 max-w-md mx-auto">
                            {liveLogs.length === 0
                              ? `Waiting for incoming HTTP requests to ${testUrl}. Invocations will stream here in real time.`
                              : 'Try adjusting your search query, HTTP method, or response code filter.'}
                          </p>
                          {liveLogs.length === 0 && (
                            <button
                              type="button"
                              onClick={() => onTestApp(app)}
                              className="mt-3 inline-flex items-center gap-1.5 px-4 py-2 rounded-lg border border-emerald-800 bg-emerald-950/60 hover:bg-emerald-950 text-emerald-400 text-xs font-semibold transition"
                            >
                              <Send className="w-3.5 h-3.5" /> Send Test Request
                            </button>
                          )}
                        </div>
                      );
                    }

                    return filtered.map((entry, idx) => {
                      const logId = entry.id || `log-${idx}`;
                      const isExpanded = expandedLogId === logId;
                      const code = entry.statusCode ?? (entry as any).status ?? 200;
                      const reqHeaders = (entry.requestHeaders || {}) as Record<string, string>;
                      const respHeaders = (entry.responseHeaders || {}) as Record<string, string>;
                      const reqHeaderCount = Object.keys(reqHeaders).length;
                      const respHeaderCount = Object.keys(respHeaders).length;
                      const totalLogsCount = (entry.logs?.length || 0) + (entry.exceptions?.length || 0);

                      return (
                        <div
                          key={logId}
                          className={`rounded-lg border transition ${
                            isExpanded
                              ? 'bg-zinc-900/90 border-emerald-500/80 shadow-md'
                              : 'bg-zinc-950/70 border-zinc-800/80 hover:border-zinc-700 hover:bg-zinc-900/40'
                          }`}
                        >
                          {/* Row Summary */}
                          <div
                            onClick={() => setExpandedLogId(isExpanded ? null : logId)}
                            className="grid grid-cols-12 gap-2 items-center px-3 py-2 cursor-pointer select-none text-[11px]"
                          >
                            <div className="col-span-2 flex items-center gap-1.5 text-zinc-400">
                              {isExpanded ? (
                                <ChevronDown className="w-3.5 h-3.5 text-emerald-400 shrink-0" />
                              ) : (
                                <ChevronRight className="w-3.5 h-3.5 text-zinc-500 shrink-0" />
                              )}
                              <span className="text-zinc-400 font-mono">
                                {new Date(entry.timestamp).toLocaleTimeString()}
                              </span>
                            </div>

                            <div className="col-span-1">
                              <span
                                className={`text-[10px] px-1.5 py-0.5 rounded font-bold uppercase ${
                                  entry.method === 'GET'
                                    ? 'bg-sky-950 text-sky-400 border border-sky-800'
                                    : entry.method === 'POST'
                                      ? 'bg-emerald-950 text-emerald-400 border border-emerald-800'
                                      : entry.method === 'PUT' || entry.method === 'PATCH'
                                        ? 'bg-amber-950 text-amber-400 border border-amber-800'
                                        : entry.method === 'DELETE'
                                          ? 'bg-rose-950 text-rose-400 border border-rose-800'
                                          : 'bg-zinc-800 text-zinc-300'
                                }`}
                              >
                                {entry.method}
                              </span>
                            </div>

                            <div className="col-span-4 truncate font-mono text-zinc-200" title={entry.url || entry.path}>
                              <span>{entry.path || '/'}</span>
                            </div>

                            <div className="col-span-1 text-center font-bold">
                              <span
                                className={`${
                                  code >= 200 && code < 300
                                    ? 'text-emerald-400'
                                    : code >= 400 && code < 500
                                      ? 'text-amber-400'
                                      : code >= 500
                                        ? 'text-rose-400'
                                        : 'text-zinc-400'
                                }`}
                              >
                                {code}
                              </span>
                            </div>

                            <div className="col-span-1 text-right text-zinc-400 font-mono">
                              {Math.round(entry.durationMs)}ms
                            </div>

                            <div className="col-span-1 text-center text-zinc-500 truncate" title={entry.clientIp}>
                              {entry.clientIp || '127.0.0.1'}
                            </div>

                            <div className="col-span-2 text-right text-zinc-500 font-mono text-[10px] truncate" title={entry.id}>
                              {entry.id ? entry.id.replace('ray_', '') : 'N/A'}
                            </div>
                          </div>

                          {/* Expanded Cloudflare-Style Log Inspector */}
                          {isExpanded && (
                            <div className="p-4 border-t border-zinc-800/80 bg-black/80 space-y-4 font-sans text-xs">
                              {/* Inspector Top Info Bar */}
                              <div className="flex flex-wrap items-center justify-between gap-3 p-3 bg-zinc-900/70 rounded-lg border border-zinc-800">
                                <div className="flex flex-wrap items-center gap-3">
                                  {/* Ray ID */}
                                  <div className="flex items-center gap-1.5">
                                    <span className="text-[11px] text-zinc-500 font-semibold uppercase">Ray ID:</span>
                                    <code className="text-zinc-200 font-mono text-[11px] bg-zinc-950 px-2 py-0.5 rounded border border-zinc-800">
                                      {entry.id || 'N/A'}
                                    </code>
                                    {entry.id && (
                                      <button
                                        type="button"
                                        onClick={() => handleCopyLogSection(entry.id!, `ray-${logId}`)}
                                        className="text-zinc-400 hover:text-zinc-200 transition"
                                        title="Copy Ray ID"
                                      >
                                        {copiedLogSection === `ray-${logId}` ? (
                                          <Check className="w-3.5 h-3.5 text-emerald-400" />
                                        ) : (
                                          <Copy className="w-3.5 h-3.5" />
                                        )}
                                      </button>
                                    )}
                                  </div>

                                  {/* Full URL */}
                                  <div className="flex items-center gap-1.5 max-w-md truncate">
                                    <span className="text-[11px] text-zinc-500 font-semibold uppercase">URL:</span>
                                    <code className="text-emerald-400 font-mono text-[11px] truncate" title={entry.url || entry.path}>
                                      {entry.url || entry.path}
                                    </code>
                                    <button
                                      type="button"
                                      onClick={() => handleCopyLogSection(entry.url || entry.path, `url-${logId}`)}
                                      className="text-zinc-400 hover:text-zinc-200 transition"
                                      title="Copy URL"
                                    >
                                      {copiedLogSection === `url-${logId}` ? (
                                        <Check className="w-3.5 h-3.5 text-emerald-400" />
                                      ) : (
                                        <Copy className="w-3.5 h-3.5" />
                                      )}
                                    </button>
                                  </div>

                                  {/* Outcome */}
                                  <span
                                    className={`text-[10px] px-2 py-0.5 rounded-full font-bold uppercase ${
                                      entry.outcome === 'exception' || code >= 500
                                        ? 'bg-rose-950 text-rose-400 border border-rose-800'
                                        : 'bg-emerald-950 text-emerald-400 border border-emerald-800'
                                    }`}
                                  >
                                    {entry.outcome || (code >= 500 ? 'exception' : 'ok')}
                                  </span>
                                </div>

                                {/* Action Buttons: Copy JSON */}
                                <div className="flex items-center gap-2">
                                  <button
                                    type="button"
                                    onClick={() => handleCopyLogSection(JSON.stringify(entry, null, 2), `json-${logId}`)}
                                    className="px-3 py-1.5 bg-emerald-950 hover:bg-emerald-900/80 text-emerald-400 border border-emerald-800 rounded-md font-semibold text-xs transition flex items-center gap-1.5"
                                    title="Copy raw Cloudflare JSON event"
                                  >
                                    {copiedLogSection === `json-${logId}` ? (
                                      <>
                                        <Check className="w-3.5 h-3.5 text-emerald-300" />
                                        <span>Copied JSON!</span>
                                      </>
                                    ) : (
                                      <>
                                        <Copy className="w-3.5 h-3.5" />
                                        <span>Copy JSON</span>
                                      </>
                                    )}
                                  </button>
                                </div>
                              </div>

                              {/* Inspector Sub-Tabs */}
                              <div className="flex items-center gap-2 border-b border-zinc-800 pb-2">
                                <button
                                  type="button"
                                  onClick={() => setActiveLogDetailTab('headers')}
                                  className={`px-3 py-1.5 rounded-md font-semibold text-xs transition flex items-center gap-1.5 ${
                                    activeLogDetailTab === 'headers'
                                      ? 'bg-zinc-800 text-zinc-100 border border-zinc-700'
                                      : 'text-zinc-400 hover:text-zinc-200'
                                  }`}
                                >
                                  <ListFilter className="w-3.5 h-3.5" />
                                  <span>Headers</span>
                                  <span className="text-[10px] bg-zinc-950 px-1.5 py-0.2 rounded border border-zinc-800 text-zinc-400">
                                    {reqHeaderCount + respHeaderCount}
                                  </span>
                                </button>

                                <button
                                  type="button"
                                  onClick={() => setActiveLogDetailTab('payload')}
                                  className={`px-3 py-1.5 rounded-md font-semibold text-xs transition flex items-center gap-1.5 ${
                                    activeLogDetailTab === 'payload'
                                      ? 'bg-zinc-800 text-zinc-100 border border-zinc-700'
                                      : 'text-zinc-400 hover:text-zinc-200'
                                  }`}
                                >
                                  <FileText className="w-3.5 h-3.5" />
                                  <span>Payload</span>
                                  {(entry.requestBody || entry.responseBody) && (
                                    <span className="w-2 h-2 rounded-full bg-emerald-500" />
                                  )}
                                </button>

                                <button
                                  type="button"
                                  onClick={() => setActiveLogDetailTab('logs')}
                                  className={`px-3 py-1.5 rounded-md font-semibold text-xs transition flex items-center gap-1.5 ${
                                    activeLogDetailTab === 'logs'
                                      ? 'bg-zinc-800 text-zinc-100 border border-zinc-700'
                                      : 'text-zinc-400 hover:text-zinc-200'
                                  }`}
                                >
                                  <Terminal className="w-3.5 h-3.5" />
                                  <span>Console Logs</span>
                                  {totalLogsCount > 0 && (
                                    <span className="text-[10px] bg-sky-950 text-sky-400 px-1.5 py-0.2 rounded border border-sky-800">
                                      {totalLogsCount}
                                    </span>
                                  )}
                                </button>

                                <button
                                  type="button"
                                  onClick={() => setActiveLogDetailTab('json')}
                                  className={`px-3 py-1.5 rounded-md font-semibold text-xs transition flex items-center gap-1.5 ${
                                    activeLogDetailTab === 'json'
                                      ? 'bg-zinc-800 text-zinc-100 border border-zinc-700'
                                      : 'text-zinc-400 hover:text-zinc-200'
                                  }`}
                                >
                                  <Braces className="w-3.5 h-3.5" />
                                  <span>Raw JSON</span>
                                </button>
                              </div>

                              {/* Tab Content: Headers */}
                              {activeLogDetailTab === 'headers' && (
                                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                                  {/* Request Headers */}
                                  <div className="bg-zinc-950 rounded-lg border border-zinc-800/80 p-3.5 space-y-3">
                                    <div className="flex items-center justify-between border-b border-zinc-900 pb-2">
                                      <div className="flex items-center gap-2">
                                        <span className="text-xs font-bold text-zinc-200">Request Headers</span>
                                        <span className="text-[10px] bg-zinc-900 px-1.5 py-0.5 rounded text-zinc-400 font-mono">
                                          {reqHeaderCount}
                                        </span>
                                      </div>
                                      {reqHeaderCount > 0 && (
                                        <button
                                          type="button"
                                          onClick={() => handleCopyLogSection(JSON.stringify(reqHeaders, null, 2), `reqh-${logId}`)}
                                          className="text-[11px] text-zinc-400 hover:text-zinc-200 flex items-center gap-1 transition"
                                        >
                                          {copiedLogSection === `reqh-${logId}` ? (
                                            <span className="text-emerald-400 flex items-center gap-1">
                                              <Check className="w-3 h-3" /> Copied
                                            </span>
                                          ) : (
                                            <>
                                              <Copy className="w-3 h-3" /> Copy
                                            </>
                                          )}
                                        </button>
                                      )}
                                    </div>

                                    {reqHeaderCount === 0 ? (
                                      <div className="text-xs text-zinc-500 italic py-2">No request headers recorded</div>
                                    ) : (
                                      <div className="space-y-1.5 max-h-[300px] overflow-y-auto pr-1">
                                        {Object.entries(reqHeaders).map(([key, val]) => (
                                          <div
                                            key={key}
                                            className="p-1.5 rounded bg-zinc-900/50 hover:bg-zinc-900 transition flex flex-col font-mono text-[11px]"
                                          >
                                            <span className="text-sky-400 font-semibold">{key}:</span>
                                            <span className="text-zinc-300 break-all pl-2">{val}</span>
                                          </div>
                                        ))}
                                      </div>
                                    )}
                                  </div>

                                  {/* Response Headers */}
                                  <div className="bg-zinc-950 rounded-lg border border-zinc-800/80 p-3.5 space-y-3">
                                    <div className="flex items-center justify-between border-b border-zinc-900 pb-2">
                                      <div className="flex items-center gap-2">
                                        <span className="text-xs font-bold text-zinc-200">Response Headers</span>
                                        <span className="text-[10px] bg-zinc-900 px-1.5 py-0.5 rounded text-zinc-400 font-mono">
                                          {respHeaderCount}
                                        </span>
                                      </div>
                                      {respHeaderCount > 0 && (
                                        <button
                                          type="button"
                                          onClick={() => handleCopyLogSection(JSON.stringify(respHeaders, null, 2), `resph-${logId}`)}
                                          className="text-[11px] text-zinc-400 hover:text-zinc-200 flex items-center gap-1 transition"
                                        >
                                          {copiedLogSection === `resph-${logId}` ? (
                                            <span className="text-emerald-400 flex items-center gap-1">
                                              <Check className="w-3 h-3" /> Copied
                                            </span>
                                          ) : (
                                            <>
                                              <Copy className="w-3 h-3" /> Copy
                                            </>
                                          )}
                                        </button>
                                      )}
                                    </div>

                                    {respHeaderCount === 0 ? (
                                      <div className="text-xs text-zinc-500 italic py-2">No response headers recorded</div>
                                    ) : (
                                      <div className="space-y-1.5 max-h-[300px] overflow-y-auto pr-1">
                                        {Object.entries(respHeaders).map(([key, val]) => (
                                          <div
                                            key={key}
                                            className="p-1.5 rounded bg-zinc-900/50 hover:bg-zinc-900 transition flex flex-col font-mono text-[11px]"
                                          >
                                            <span className="text-emerald-400 font-semibold">{key}:</span>
                                            <span className="text-zinc-300 break-all pl-2">{val}</span>
                                          </div>
                                        ))}
                                      </div>
                                    )}
                                  </div>
                                </div>
                              )}

                              {/* Tab Content: Payload */}
                              {activeLogDetailTab === 'payload' && (
                                <div className="space-y-4">
                                  {/* Request Body */}
                                  <div className="bg-zinc-950 rounded-lg border border-zinc-800/80 p-3.5 space-y-2">
                                    <div className="flex items-center justify-between border-b border-zinc-900 pb-2">
                                      <span className="text-xs font-bold text-zinc-200">
                                        Request Body {entry.requestBody ? `(${entry.requestBody.length} bytes)` : ''}
                                      </span>
                                      {entry.requestBody && (
                                        <button
                                          type="button"
                                          onClick={() => handleCopyLogSection(entry.requestBody!, `reqb-${logId}`)}
                                          className="text-[11px] text-zinc-400 hover:text-zinc-200 flex items-center gap-1 transition"
                                        >
                                          {copiedLogSection === `reqb-${logId}` ? (
                                            <span className="text-emerald-400 flex items-center gap-1">
                                              <Check className="w-3 h-3" /> Copied
                                            </span>
                                          ) : (
                                            <>
                                              <Copy className="w-3 h-3" /> Copy
                                            </>
                                          )}
                                        </button>
                                      )}
                                    </div>

                                    {entry.requestBody ? (
                                      <pre className="text-[11px] font-mono text-zinc-300 bg-zinc-900/70 p-3 rounded overflow-x-auto max-h-[250px] whitespace-pre-wrap">
                                        {formatJsonPayload(entry.requestBody)}
                                      </pre>
                                    ) : (
                                      <p className="text-xs text-zinc-500 italic py-2">
                                        (No request body payload sent)
                                      </p>
                                    )}
                                  </div>

                                  {/* Response Body */}
                                  <div className="bg-zinc-950 rounded-lg border border-zinc-800/80 p-3.5 space-y-2">
                                    <div className="flex items-center justify-between border-b border-zinc-900 pb-2">
                                      <span className="text-xs font-bold text-zinc-200">
                                        Response Body {entry.responseBody ? `(${entry.responseBody.length} bytes)` : ''}
                                      </span>
                                      {entry.responseBody && (
                                        <button
                                          type="button"
                                          onClick={() => handleCopyLogSection(entry.responseBody!, `respb-${logId}`)}
                                          className="text-[11px] text-zinc-400 hover:text-zinc-200 flex items-center gap-1 transition"
                                        >
                                          {copiedLogSection === `respb-${logId}` ? (
                                            <span className="text-emerald-400 flex items-center gap-1">
                                              <Check className="w-3 h-3" /> Copied
                                            </span>
                                          ) : (
                                            <>
                                              <Copy className="w-3 h-3" /> Copy
                                            </>
                                          )}
                                        </button>
                                      )}
                                    </div>

                                    {entry.responseBody ? (
                                      <pre className="text-[11px] font-mono text-emerald-300/90 bg-zinc-900/70 p-3 rounded overflow-x-auto max-h-[250px] whitespace-pre-wrap">
                                        {formatJsonPayload(entry.responseBody)}
                                      </pre>
                                    ) : (
                                      <p className="text-xs text-zinc-500 italic py-2">
                                        (Empty response body returned by worker)
                                      </p>
                                    )}
                                  </div>
                                </div>
                              )}

                              {/* Tab Content: Console Logs & Exceptions */}
                              {activeLogDetailTab === 'logs' && (
                                <div className="space-y-3">
                                  {/* Exceptions */}
                                  {entry.exceptions && entry.exceptions.length > 0 && (
                                    <div className="p-3 bg-rose-950/60 border border-rose-900/80 rounded-lg space-y-2">
                                      <div className="flex items-center gap-2 text-rose-400 font-bold text-xs">
                                        <AlertTriangle className="w-4 h-4" />
                                        <span>Uncaught Runtime Exceptions ({entry.exceptions.length})</span>
                                      </div>
                                      {entry.exceptions.map((exc, eIdx) => (
                                        <pre
                                          key={eIdx}
                                          className="text-[11px] font-mono text-rose-300 bg-rose-950/40 p-2.5 rounded border border-rose-900/50 overflow-x-auto whitespace-pre-wrap"
                                        >
                                          {exc}
                                        </pre>
                                      ))}
                                    </div>
                                  )}

                                  {/* Console Logs */}
                                  <div className="bg-zinc-950 rounded-lg border border-zinc-800/80 p-3.5 space-y-2">
                                    <span className="text-xs font-bold text-zinc-200 block border-b border-zinc-900 pb-2">
                                      Isolate Console Output ({entry.logs?.length || 0})
                                    </span>

                                    {!entry.logs || entry.logs.length === 0 ? (
                                      <p className="text-xs text-zinc-500 italic py-2">
                                        No console.log(), console.info(), or console.error() statements executed for this request.
                                      </p>
                                    ) : (
                                      <div className="space-y-1.5 font-mono text-[11px] max-h-[300px] overflow-y-auto">
                                        {entry.logs.map((log, lIdx) => (
                                          <div
                                            key={lIdx}
                                            className="flex items-start gap-2.5 p-1.5 rounded bg-zinc-900/60 hover:bg-zinc-900"
                                          >
                                            <span
                                              className={`text-[9px] px-1.5 py-0.5 rounded font-bold uppercase shrink-0 ${
                                                log.level === 'error'
                                                  ? 'bg-rose-950 text-rose-400 border border-rose-800'
                                                  : log.level === 'warn'
                                                    ? 'bg-amber-950 text-amber-400 border border-amber-800'
                                                    : log.level === 'info'
                                                      ? 'bg-sky-950 text-sky-400 border border-sky-800'
                                                      : 'bg-zinc-800 text-zinc-300'
                                              }`}
                                            >
                                              {log.level}
                                            </span>
                                            <span className="text-zinc-300 whitespace-pre-wrap flex-1 break-all">
                                              {log.message}
                                            </span>
                                          </div>
                                        ))}
                                      </div>
                                    )}
                                  </div>
                                </div>
                              )}

                              {/* Tab Content: Raw JSON */}
                              {activeLogDetailTab === 'json' && (
                                <div className="space-y-2">
                                  <div className="flex items-center justify-between text-xs text-zinc-400">
                                    <span>Cloudflare Structured Telemetry JSON</span>
                                    <button
                                      type="button"
                                      onClick={() => handleCopyLogSection(JSON.stringify(entry, null, 2), `rawjson-${logId}`)}
                                      className="px-2.5 py-1 bg-zinc-800 hover:bg-zinc-700 text-zinc-300 rounded font-semibold text-xs transition flex items-center gap-1"
                                    >
                                      {copiedLogSection === `rawjson-${logId}` ? (
                                        <span className="text-emerald-400 flex items-center gap-1">
                                          <Check className="w-3 h-3" /> Copied JSON
                                        </span>
                                      ) : (
                                        <>
                                          <Copy className="w-3 h-3" /> Copy JSON
                                        </>
                                      )}
                                    </button>
                                  </div>
                                  <pre className="text-[11px] font-mono text-zinc-300 bg-zinc-950 p-4 rounded-lg border border-zinc-800/80 overflow-x-auto max-h-[400px] whitespace-pre-wrap">
                                    {JSON.stringify(entry, null, 2)}
                                  </pre>
                                </div>
                              )}
                            </div>
                          )}
                        </div>
                      );
                    });
                  })()}
                </div>
              </div>
            )}
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

                  <div className="p-3 rounded-xl bg-purple-950/20 border border-purple-900/40 space-y-1">
                    <div className="flex items-center gap-2 text-xs font-semibold text-purple-300">
                      <ShieldCheck className="w-3.5 h-3.5" />
                      <span>Push-to-Deploy CI/CD Active</span>
                    </div>
                    <p className="text-[11px] text-zinc-400">
                      Incoming commits pushed to <code className="text-zinc-200">{app.branch || 'main'}</code> trigger automatic builds and releases via the GitHub App webhook.
                    </p>
                  </div>
                </div>
              </div>
            )}
          </div>
        )}

        {/* Tab 3: Triggers */}
        {activeTab === 'triggers' && (
          <div className="space-y-8 max-w-4xl">
            {/* Section 1: Ingress & Custom Domains */}
            <div className="space-y-4">
              <div className="flex items-center justify-between">
                <div>
                  <h3 className="text-sm font-bold text-zinc-200 flex items-center gap-2">
                    <Globe className="w-4 h-4 text-purple-400" />
                    HTTP & Custom Domains
                  </h3>
                  <p className="text-xs text-zinc-400">
                    Hostnames routed directly to this worker through Traefik and edge proxies.
                  </p>
                </div>
              </div>

              {/* Default Traefik Subdomain Route */}
              <div className="p-5 rounded-xl border border-zinc-800 bg-zinc-900/20 space-y-3">
                <div className="flex items-start justify-between">
                  <div>
                    <h4 className="text-xs font-semibold text-zinc-200">Default Subdomain Ingress</h4>
                    <p className="text-[11px] text-zinc-500 mt-0.5">
                      Automatically assigned and synchronized with the Traefik fleet proxy.
                    </p>
                  </div>
                  <span className="text-xs font-semibold text-emerald-400 bg-emerald-950/60 border border-emerald-800/80 px-2.5 py-0.5 rounded-full flex items-center gap-1">
                    <ShieldCheck className="w-3.5 h-3.5" /> Active
                  </span>
                </div>

                <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 pt-1">
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

              {/* Custom Domains list */}
              {appDomains.length > 0 && (
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
                <span className="text-xs font-semibold text-zinc-300 block">Add New Custom Domain</span>
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

            {/* Section 2: Cron Triggers (scheduled()) */}
            <div className="space-y-4 pt-4 border-t border-zinc-800/80">
              <div className="flex items-center justify-between">
                <div>
                  <h3 className="text-sm font-bold text-zinc-200 flex items-center gap-2">
                    <Clock className="w-4 h-4 text-sky-400" />
                    Cron Triggers (scheduled())
                  </h3>
                  <p className="text-xs text-zinc-400">
                    Execute your worker's exported <code className="font-mono text-emerald-400">scheduled(event, env, ctx)</code> handler on an automated schedule.
                  </p>
                </div>
                <button
                  type="button"
                  onClick={() => setShowAddCronModal(true)}
                  className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-sky-600 hover:bg-sky-500 text-white text-xs font-semibold transition shadow-sm"
                >
                  <Plus className="w-3.5 h-3.5" />
                  Add Cron Trigger
                </button>
              </div>

              {attachedCron.length === 0 ? (
                <div className="p-6 rounded-xl border border-dashed border-zinc-800 bg-zinc-950/40 text-center space-y-2">
                  <p className="text-xs text-zinc-400">No cron triggers scheduled for this worker.</p>
                  <p className="text-[11px] text-zinc-600 max-w-sm mx-auto">
                    Add a cron schedule to periodically invoke maintenance tasks, cache warming, or scheduled jobs.
                  </p>
                </div>
              ) : (
                <div className="space-y-2.5">
                  {attachedCron.map((cron) => (
                    <div
                      key={cron.id}
                      className="p-4 rounded-xl border border-zinc-800 bg-zinc-900/40 flex items-center justify-between gap-4 flex-wrap"
                    >
                      <div className="flex items-center gap-3">
                        <div className="p-2 rounded-lg bg-sky-950/80 border border-sky-800/80 text-sky-400">
                          <Clock className="w-4 h-4" />
                        </div>
                        <div>
                          <div className="flex items-center gap-2">
                            <span className="text-xs font-bold text-zinc-200">{cron.name}</span>
                            <span className="font-mono text-[11px] px-2 py-0.5 rounded bg-zinc-800 text-emerald-400 border border-zinc-700">
                              {cron.cronExpression || cron.cron}
                            </span>
                            <span className="text-[10px] px-2 py-0.5 rounded-full font-bold uppercase bg-emerald-950 text-emerald-400 border border-emerald-800">
                              Active
                            </span>
                          </div>
                          <div className="text-[10px] text-zinc-500 font-mono mt-1">
                            Last run: {cron.lastRunAt ? new Date(cron.lastRunAt).toLocaleString() : 'Never'}
                          </div>
                        </div>
                      </div>

                      <div className="flex items-center gap-2">
                        <button
                          type="button"
                          onClick={() => handleRunCronNow(cron.id)}
                          disabled={runningCronId === cron.id}
                          className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg border border-sky-800 bg-sky-950/60 hover:bg-sky-900/80 text-sky-300 text-xs font-semibold transition"
                          title="Trigger immediate execution"
                        >
                          {runningCronId === cron.id ? <RefreshCw className="w-3.5 h-3.5 animate-spin" /> : <Play className="w-3.5 h-3.5" />}
                          Run Now
                        </button>
                        <button
                          type="button"
                          onClick={() => onNavigateToService?.('cron', cron.id)}
                          className="flex items-center gap-1 px-3 py-1.5 rounded-lg bg-zinc-800 hover:bg-zinc-700 text-zinc-300 text-xs font-semibold transition"
                        >
                          <span>Open in Cron</span>
                          <ArrowRight className="w-3.5 h-3.5 text-zinc-400" />
                        </button>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>

            {/* Section 3: Queue Consumers (queue()) */}
            <div className="space-y-4 pt-4 border-t border-zinc-800/80">
              <div className="flex items-center justify-between">
                <div>
                  <h3 className="text-sm font-bold text-zinc-200 flex items-center gap-2">
                    <Inbox className="w-4 h-4 text-emerald-400" />
                    Queue Consumers (queue())
                  </h3>
                  <p className="text-xs text-zinc-400">
                    Process batches of messages via your worker's exported <code className="font-mono text-emerald-400">queue(batch, env, ctx)</code> handler.
                  </p>
                </div>
                <button
                  type="button"
                  onClick={() => onNavigateToService?.('queues')}
                  className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-zinc-800 hover:bg-zinc-700 text-zinc-300 text-xs font-semibold transition"
                >
                  <span>Manage Queues</span>
                  <ArrowRight className="w-3.5 h-3.5 text-emerald-400" />
                </button>
              </div>

              {attachedQueues.length === 0 ? (
                <div className="p-6 rounded-xl border border-dashed border-zinc-800 bg-zinc-950/40 text-center space-y-2">
                  <p className="text-xs text-zinc-400">No queues configured to send messages to this worker.</p>
                  <p className="text-[11px] text-zinc-600 max-w-sm mx-auto">
                    To process messages, bind this worker as a consumer in the Queues control panel.
                  </p>
                </div>
              ) : (
                <div className="space-y-2.5">
                  {attachedQueues.map((q) => (
                    <div
                      key={q.id}
                      className="p-4 rounded-xl border border-zinc-800 bg-zinc-900/40 flex items-center justify-between"
                    >
                      <div className="flex items-center gap-3">
                        <div className="p-2 rounded-lg bg-emerald-950/80 border border-emerald-800/80 text-emerald-400">
                          <Inbox className="w-4 h-4" />
                        </div>
                        <div>
                          <div className="flex items-center gap-2">
                            <span className="text-xs font-bold text-zinc-200">{q.name}</span>
                            <span className="text-[10px] px-2 py-0.5 rounded-full font-bold uppercase bg-emerald-950 text-emerald-400 border border-emerald-800">
                              Consumer Active
                            </span>
                          </div>
                          <div className="text-[11px] text-zinc-500 font-mono mt-0.5">
                            Batch size: {q.maxBatchSize || 10} • Retries: {q.maxRetries || 3}
                          </div>
                        </div>
                      </div>

                      <button
                        type="button"
                        onClick={() => onNavigateToService?.('queues', q.id)}
                        className="flex items-center gap-1 px-3 py-1.5 rounded-lg bg-zinc-800 hover:bg-zinc-700 text-zinc-300 text-xs font-semibold transition"
                      >
                        <span>Open Queue</span>
                        <ArrowRight className="w-3.5 h-3.5 text-emerald-400" />
                      </button>
                    </div>
                  ))}
                </div>
              )}
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

            {/* Runtime & Compatibility Configuration */}
            <form onSubmit={handleSaveRuntime} className="p-5 rounded-xl border border-zinc-800 bg-zinc-900/20 space-y-4">
              <div className="flex items-center justify-between">
                <div>
                  <h3 className="text-sm font-semibold text-zinc-200 flex items-center gap-2">
                    <Sliders className="w-4 h-4 text-emerald-400" />
                    Runtime & Compatibility Settings
                  </h3>
                  <p className="text-xs text-zinc-400">
                    Point-in-time compatibility date, Node.js API support, and celld execution resource limits.
                  </p>
                </div>
                <div className="flex items-center gap-2">
                  {runtimeSaveStatus === 'saved' && (
                    <span className="text-xs text-emerald-400 font-semibold flex items-center gap-1">
                      <Check className="w-3.5 h-3.5" /> Runtime Saved!
                    </span>
                  )}
                  {runtimeSaveStatus === 'error' && (
                    <span className="text-xs text-rose-400 font-semibold">Failed to save runtime</span>
                  )}
                  <button
                    type="submit"
                    disabled={isSavingRuntime}
                    className="px-3.5 py-1.5 rounded-lg bg-emerald-500 hover:bg-emerald-600 disabled:opacity-50 text-zinc-950 text-xs font-semibold transition"
                  >
                    {isSavingRuntime ? 'Saving...' : 'Save Runtime Settings'}
                  </button>
                </div>
              </div>

              <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 text-xs font-mono">
                <div>
                  <label className="text-zinc-400 block mb-1 font-sans font-medium">Compatibility Date</label>
                  <input
                    type="text"
                    value={compatDate}
                    onChange={e => setCompatDate(e.target.value)}
                    placeholder="2024-04-03"
                    className="w-full bg-zinc-950 border border-zinc-800 rounded-lg p-2.5 text-zinc-200 outline-none focus:border-emerald-500"
                  />
                  <span className="text-[11px] text-zinc-500 font-sans block mt-1">
                    Standard Cloudflare Worker point-in-time runtime snapshot (YYYY-MM-DD).
                  </span>
                </div>

                <div>
                  <label className="text-zinc-400 block mb-1 font-sans font-medium">Memory Limit (Isolate)</label>
                  <select
                    value={memoryLimitMB}
                    onChange={e => setMemoryLimitMB(Number(e.target.value))}
                    className="w-full bg-zinc-950 border border-zinc-800 rounded-lg p-2.5 text-zinc-200 outline-none focus:border-emerald-500 font-sans"
                  >
                    <option value={64}>64 MB (Lightweight)</option>
                    <option value={128}>128 MB (Default standard)</option>
                    <option value={256}>256 MB (High capacity)</option>
                    <option value={512}>512 MB (Data intensive)</option>
                    <option value={1024}>1024 MB (Max limit)</option>
                  </select>
                  <span className="text-[11px] text-zinc-500 font-sans block mt-1">
                    Maximum heap memory allocated per V8 isolate.
                  </span>
                </div>

                <div>
                  <label className="text-zinc-400 block mb-1 font-sans font-medium">CPU Execution Timeout (ms)</label>
                  <input
                    type="number"
                    min={5}
                    max={30000}
                    value={maxDurationMs}
                    onChange={e => setMaxDurationMs(Number(e.target.value))}
                    className="w-full bg-zinc-950 border border-zinc-800 rounded-lg p-2.5 text-zinc-200 outline-none focus:border-emerald-500"
                  />
                  <span className="text-[11px] text-zinc-500 font-sans block mt-1">
                    Allowed CPU time per request before execution is terminated.
                  </span>
                </div>

                <div className="flex flex-col justify-center pt-2">
                  <label className="flex items-center gap-2 cursor-pointer select-none">
                    <input
                      type="checkbox"
                      checked={nodejsCompat}
                      onChange={e => setNodejsCompat(e.target.checked)}
                      className="rounded bg-zinc-950 border-zinc-800 text-emerald-500 focus:ring-0 w-4 h-4"
                    />
                    <span className="text-xs font-semibold text-zinc-200 font-sans">
                      Enable Node.js Compatibility (nodejs_compat)
                    </span>
                  </label>
                  <span className="text-[11px] text-zinc-500 font-sans mt-1 ml-6">
                    Injects <code className="text-emerald-400 font-mono">node:buffer</code>, <code className="text-emerald-400 font-mono">node:crypto</code>, <code className="text-emerald-400 font-mono">node:events</code>, etc.
                  </span>
                </div>
              </div>
            </form>

            {/* Cloudflare-Style Variables & Secrets Suite */}
            <div className="p-6 rounded-xl border border-zinc-800 bg-zinc-900/30 space-y-5">
              {/* Header & Controls */}
              <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 border-b border-zinc-800/80 pb-4">
                <div>
                  <div className="flex items-center gap-2.5">
                    <h3 className="text-sm font-bold text-zinc-100 flex items-center gap-2">
                      <ShieldCheck className="w-4 h-4 text-emerald-400" />
                      Variables & Secrets
                    </h3>
                    <span className="text-[11px] px-2 py-0.5 rounded-full font-medium bg-zinc-800 text-zinc-300 border border-zinc-700">
                      {envVars.length} total
                    </span>
                    {envVars.filter(v => v.isSecret).length > 0 && (
                      <span className="text-[11px] px-2 py-0.5 rounded-full font-medium bg-amber-950/80 text-amber-400 border border-amber-800/80 flex items-center gap-1">
                        <Lock className="w-3 h-3" />
                        {envVars.filter(v => v.isSecret).length} secrets
                      </span>
                    )}
                  </div>
                  <p className="text-xs text-zinc-400 mt-1">
                    Environment variables and encrypted secrets injected directly into your isolate runtime context via <code className="text-emerald-400 font-mono">env</code> in <code className="text-emerald-400 font-mono">worker.fetch(request, env, ctx)</code>.
                  </p>
                </div>

                <div className="flex items-center gap-2 shrink-0">
                  {envSaveStatus === 'saved' && (
                    <span className="text-xs text-emerald-400 font-semibold flex items-center gap-1">
                      <Check className="w-3.5 h-3.5" /> Variables Saved!
                    </span>
                  )}
                  {envSaveStatus === 'error' && (
                    <span className="text-xs text-rose-400 font-semibold">Failed to save variables</span>
                  )}
                  <button
                    type="button"
                    onClick={() => {
                      setNewVarKey('');
                      setNewVarValue('');
                      setNewVarIsSecret(false);
                      setShowNewVarValue(false);
                      setShowAddVarModal(true);
                    }}
                    className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-emerald-500 hover:bg-emerald-400 text-zinc-950 text-xs font-bold transition shadow-xs"
                  >
                    <Plus className="w-3.5 h-3.5" />
                    Add Variable or Secret
                  </button>
                  <button
                    type="button"
                    onClick={handleSaveEnvVars}
                    disabled={isSavingEnv}
                    className="px-3 py-1.5 rounded-lg border border-zinc-700 bg-zinc-800 hover:bg-zinc-700 disabled:opacity-50 text-zinc-200 text-xs font-medium transition"
                  >
                    {isSavingEnv ? 'Saving...' : 'Save Changes'}
                  </button>
                </div>
              </div>

              {/* Search & Type Filter Bar */}
              <div className="flex flex-wrap items-center justify-between gap-3">
                <div className="relative flex-1 max-w-sm">
                  <Search className="w-3.5 h-3.5 text-zinc-500 absolute left-2.5 top-1/2 -translate-y-1/2" />
                  <input
                    type="text"
                    value={varSearchTerm}
                    onChange={e => setVarSearchTerm(e.target.value)}
                    placeholder="Filter by variable name..."
                    className="w-full pl-8 pr-3 py-1.5 bg-black/60 border border-zinc-800 rounded-lg text-zinc-200 placeholder-zinc-500 text-xs focus:outline-none focus:border-emerald-500"
                  />
                  {varSearchTerm && (
                    <button
                      type="button"
                      onClick={() => setVarSearchTerm('')}
                      className="absolute right-2 top-1/2 -translate-y-1/2 text-zinc-400 hover:text-zinc-200"
                    >
                      <X className="w-3.5 h-3.5" />
                    </button>
                  )}
                </div>

                <div className="flex items-center gap-1 bg-black/60 p-0.5 rounded-lg border border-zinc-800 text-[11px]">
                  <button
                    type="button"
                    onClick={() => setVarTypeFilter('ALL')}
                    className={`px-2.5 py-1 rounded-md font-semibold transition ${
                      varTypeFilter === 'ALL'
                        ? 'bg-zinc-800 text-zinc-100 shadow-xs'
                        : 'text-zinc-400 hover:text-zinc-200'
                    }`}
                  >
                    All ({envVars.length})
                  </button>
                  <button
                    type="button"
                    onClick={() => setVarTypeFilter('PLAIN')}
                    className={`px-2.5 py-1 rounded-md font-semibold transition ${
                      varTypeFilter === 'PLAIN'
                        ? 'bg-zinc-800 text-zinc-100 shadow-xs'
                        : 'text-zinc-400 hover:text-zinc-200'
                    }`}
                  >
                    Plaintext ({envVars.filter(v => !v.isSecret).length})
                  </button>
                  <button
                    type="button"
                    onClick={() => setVarTypeFilter('SECRET')}
                    className={`px-2.5 py-1 rounded-md font-semibold transition flex items-center gap-1 ${
                      varTypeFilter === 'SECRET'
                        ? 'bg-amber-950/80 text-amber-300 border border-amber-800/80 shadow-xs'
                        : 'text-zinc-400 hover:text-zinc-200'
                    }`}
                  >
                    <Lock className="w-3 h-3" />
                    Secrets ({envVars.filter(v => v.isSecret).length})
                  </button>
                </div>
              </div>

              {/* Table or Empty State */}
              {envVars.length === 0 ? (
                <div className="p-8 rounded-xl border border-dashed border-zinc-800 text-center space-y-3">
                  <div className="w-10 h-10 rounded-full bg-zinc-800/60 border border-zinc-700/60 flex items-center justify-center mx-auto text-zinc-400">
                    <ShieldCheck className="w-5 h-5" />
                  </div>
                  <div className="space-y-1">
                    <div className="text-xs font-bold text-zinc-200">No environment variables or secrets configured</div>
                    <p className="text-[11px] text-zinc-500 max-w-md mx-auto">
                      Variables and secrets configure your isolate runtime with API keys, database URLs, and service tokens.
                    </p>
                  </div>
                  <button
                    type="button"
                    onClick={() => {
                      setNewVarKey('');
                      setNewVarValue('');
                      setNewVarIsSecret(false);
                      setShowNewVarValue(false);
                      setShowAddVarModal(true);
                    }}
                    className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-emerald-500/10 hover:bg-emerald-500/20 text-emerald-400 border border-emerald-500/30 text-xs font-semibold transition"
                  >
                    <Plus className="w-3.5 h-3.5" />
                    Add your first variable
                  </button>
                </div>
              ) : (
                <div className="overflow-hidden border border-zinc-800/80 rounded-xl bg-black/40">
                  <table className="w-full text-left text-xs">
                    <thead>
                      <tr className="border-b border-zinc-800 bg-zinc-950/80 text-zinc-400 font-semibold uppercase tracking-wider text-[11px]">
                        <th className="py-2.5 px-4">Variable Name</th>
                        <th className="py-2.5 px-4 w-36">Type</th>
                        <th className="py-2.5 px-4">Value</th>
                        <th className="py-2.5 px-4 w-28 text-right">Actions</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-zinc-800/60">
                      {envVars
                        .map((e, originalIndex) => ({ ...e, originalIndex }))
                        .filter(e => {
                          if (varTypeFilter === 'PLAIN' && e.isSecret) return false;
                          if (varTypeFilter === 'SECRET' && !e.isSecret) return false;
                          if (varSearchTerm.trim()) {
                            const q = varSearchTerm.toLowerCase().trim();
                            return e.key.toLowerCase().includes(q) || (!e.isSecret && e.value.toLowerCase().includes(q));
                          }
                          return true;
                        })
                        .map((e) => {
                          const idx = e.originalIndex;
                          const isRevealed = showSecretValues[idx];
                          const isEditing = editingVarIndex === idx;
                          const isConfirmingDelete = varDeleteConfirmIndex === idx;

                          if (isEditing) {
                            return (
                              <tr key={idx} className="bg-zinc-900/60">
                                <td className="py-2 px-4">
                                  <input
                                    type="text"
                                    value={e.key}
                                    onChange={ev => updateEnvVarRow(idx, 'key', ev.target.value.toUpperCase())}
                                    className="w-full bg-zinc-950 border border-zinc-700 rounded-lg p-1.5 text-xs font-mono text-zinc-100 uppercase outline-none focus:border-emerald-500"
                                  />
                                </td>
                                <td className="py-2 px-4">
                                  <label className="flex items-center gap-1.5 cursor-pointer select-none">
                                    <input
                                      type="checkbox"
                                      checked={e.isSecret}
                                      onChange={ev => updateEnvVarRow(idx, 'isSecret', ev.target.checked)}
                                      className="rounded bg-zinc-950 border-zinc-700 text-amber-500 focus:ring-0"
                                    />
                                    <span className="text-xs text-zinc-300">Secret</span>
                                  </label>
                                </td>
                                <td className="py-2 px-4">
                                  <input
                                    type={e.isSecret && !isRevealed ? 'password' : 'text'}
                                    value={e.value}
                                    onChange={ev => updateEnvVarRow(idx, 'value', ev.target.value)}
                                    className="w-full bg-zinc-950 border border-zinc-700 rounded-lg p-1.5 text-xs font-mono text-zinc-100 outline-none focus:border-emerald-500"
                                  />
                                </td>
                                <td className="py-2 px-4 text-right">
                                  <div className="flex items-center justify-end gap-1">
                                    <button
                                      type="button"
                                      onClick={() => {
                                        setEditingVarIndex(null);
                                        handleSaveEnvVars();
                                      }}
                                      className="p-1.5 rounded-md bg-emerald-500/20 text-emerald-400 hover:bg-emerald-500/30 transition"
                                      title="Save row"
                                    >
                                      <Check className="w-3.5 h-3.5" />
                                    </button>
                                    <button
                                      type="button"
                                      onClick={() => setEditingVarIndex(null)}
                                      className="p-1.5 rounded-md bg-zinc-800 text-zinc-400 hover:text-zinc-200 transition"
                                      title="Done"
                                    >
                                      <X className="w-3.5 h-3.5" />
                                    </button>
                                  </div>
                                </td>
                              </tr>
                            );
                          }

                          return (
                            <tr key={idx} className="hover:bg-zinc-900/40 transition">
                              <td className="py-3 px-4">
                                <div className="flex items-center gap-2">
                                  <span className="font-mono font-bold text-xs text-zinc-100 bg-zinc-800/80 px-2 py-1 rounded border border-zinc-700/60">
                                    {e.key}
                                  </span>
                                  <button
                                    type="button"
                                    onClick={() => handleCopyVar(e.key, 'key', `k-${idx}`)}
                                    className="text-zinc-500 hover:text-zinc-300 transition"
                                    title="Copy variable name"
                                  >
                                    {copiedVarKey === `k-${idx}` ? (
                                      <Check className="w-3 h-3 text-emerald-400" />
                                    ) : (
                                      <Copy className="w-3 h-3" />
                                    )}
                                  </button>
                                </div>
                              </td>

                              <td className="py-3 px-4">
                                {e.isSecret ? (
                                  <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[10px] font-semibold bg-amber-950/80 text-amber-400 border border-amber-800/80">
                                    <Lock className="w-2.5 h-2.5" />
                                    Secret
                                  </span>
                                ) : (
                                  <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[10px] font-semibold bg-zinc-800/80 text-zinc-300 border border-zinc-700/60">
                                    <Code className="w-2.5 h-2.5 text-zinc-400" />
                                    Plaintext
                                  </span>
                                )}
                              </td>

                              <td className="py-3 px-4">
                                <div className="flex items-center gap-2">
                                  <span className="font-mono text-xs text-zinc-300 truncate max-w-sm">
                                    {e.isSecret && !isRevealed ? '••••••••••••••••' : e.value || '<empty>'}
                                  </span>
                                  {e.isSecret && (
                                    <button
                                      type="button"
                                      onClick={() => setShowSecretValues({ ...showSecretValues, [idx]: !isRevealed })}
                                      className="p-1 rounded text-zinc-500 hover:text-zinc-300 transition"
                                      title={isRevealed ? 'Mask secret' : 'Reveal secret'}
                                    >
                                      {isRevealed ? <EyeOff className="w-3.5 h-3.5 text-amber-400" /> : <Eye className="w-3.5 h-3.5" />}
                                    </button>
                                  )}
                                  <button
                                    type="button"
                                    onClick={() => handleCopyVar(e.value, 'value', `v-${idx}`)}
                                    className="p-1 rounded text-zinc-500 hover:text-zinc-300 transition"
                                    title="Copy value"
                                  >
                                    {copiedVarValue === `v-${idx}` ? (
                                      <Check className="w-3.5 h-3.5 text-emerald-400" />
                                    ) : (
                                      <Copy className="w-3.5 h-3.5" />
                                    )}
                                  </button>
                                </div>
                              </td>

                              <td className="py-3 px-4 text-right">
                                {isConfirmingDelete ? (
                                  <div className="flex items-center justify-end gap-1.5">
                                    <span className="text-[10px] text-rose-400 font-semibold">Delete?</span>
                                    <button
                                      type="button"
                                      onClick={() => handleDeleteVariable(idx)}
                                      className="px-2 py-0.5 rounded bg-rose-600 hover:bg-rose-500 text-white text-[11px] font-bold transition"
                                    >
                                      Yes
                                    </button>
                                    <button
                                      type="button"
                                      onClick={() => setVarDeleteConfirmIndex(null)}
                                      className="px-1.5 py-0.5 rounded text-zinc-400 hover:text-zinc-200 text-[11px]"
                                    >
                                      No
                                    </button>
                                  </div>
                                ) : (
                                  <div className="flex items-center justify-end gap-1">
                                    <button
                                      type="button"
                                      onClick={() => setEditingVarIndex(idx)}
                                      className="p-1.5 rounded-lg text-zinc-400 hover:text-zinc-200 hover:bg-zinc-800 transition"
                                      title="Edit variable"
                                    >
                                      <Edit2 className="w-3.5 h-3.5" />
                                    </button>
                                    <button
                                      type="button"
                                      onClick={() => setVarDeleteConfirmIndex(idx)}
                                      className="p-1.5 rounded-lg text-zinc-400 hover:text-rose-400 hover:bg-rose-950/40 transition"
                                      title="Delete variable"
                                    >
                                      <Trash2 className="w-3.5 h-3.5" />
                                    </button>
                                  </div>
                                )}
                              </td>
                            </tr>
                          );
                        })}
                    </tbody>
                  </table>
                </div>
              )}

              {/* Developer Code Snippet Guide */}
              <div className="p-4 rounded-xl bg-black/60 border border-zinc-800/80 space-y-2 text-xs">
                <div className="flex items-center justify-between text-zinc-400">
                  <span className="font-semibold text-zinc-300 flex items-center gap-1.5">
                    <Code className="w-3.5 h-3.5 text-emerald-400" />
                    Runtime Access in Worker Code
                  </span>
                  <span className="text-[11px] text-zinc-500 font-mono">fetch(request, env, ctx)</span>
                </div>
                <pre className="font-mono text-[11px] text-zinc-300 bg-zinc-950 p-3 rounded-lg border border-zinc-800/80 overflow-x-auto">
{`export default {
  async fetch(request, env, ctx) {
    const apiKey = env.API_KEY;         // Secrets & variables injected via env
    const dbUrl = env.DATABASE_URL;
    return new Response(\`Loaded API key: \${apiKey ? 'present' : 'none'}\`);
  }
};`}
                </pre>
              </div>
            </div>

            {/* Danger Zone: Delete Application */}
            <div className="p-5 rounded-xl border border-rose-900/60 bg-rose-950/10 space-y-4">
              <div className="flex items-start justify-between">
                <div>
                  <h3 className="text-sm font-semibold text-rose-300 flex items-center gap-2">
                    <AlertTriangle className="w-4 h-4 text-rose-400" />
                    Danger Zone
                  </h3>
                  <p className="text-xs text-zinc-400 mt-1">
                    Permanently delete this worker application and all its deployment records from the cluster.
                  </p>
                </div>
                <button
                  type="button"
                  onClick={() => setShowDeleteConfirm(true)}
                  className="px-3.5 py-1.5 rounded-lg bg-rose-600 hover:bg-rose-500 text-white text-xs font-semibold transition"
                >
                  Delete Worker
                </button>
              </div>
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
                  <option value="service">Service Binding (Worker-to-Worker RPC)</option>
                </select>
              </div>

              {/* Resource Target */}
              <div className="space-y-1.5">
                <label className="text-xs font-semibold text-zinc-400">
                  {newBindingType === 'service' ? 'Target Worker' : 'Target Fleet Resource'}
                </label>
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
                  {newBindingType === 'service' && allApps.filter(a => a.id !== app.id).map(a => (
                    <option key={a.id} value={a.name || a.id}>{a.name} ({a.subdomain || a.id.substring(0, 8)})</option>
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
                    placeholder={newBindingType === 'service' ? 'MY_WORKER' : 'MY_BINDING'}
                    value={newBindingName}
                    onChange={e => setNewBindingName(e.target.value.toUpperCase().replace(/[^A-Z0-9_]/g, ''))}
                    className="flex-1 bg-zinc-950 border border-zinc-800 rounded-lg p-2.5 text-xs text-zinc-200 font-mono outline-none focus:border-emerald-500 uppercase"
                    required
                  />
                </div>
                <p className="text-[11px] text-zinc-500">
                  Access in worker: <code className="font-mono text-emerald-400">env.{newBindingName || (newBindingType === 'service' ? 'MY_WORKER' : 'MY_BINDING')}</code>
                  {newBindingType === 'service' && (
                    <span className="block mt-0.5 text-zinc-400">
                      RPC fetch: <code className="font-mono text-zinc-300">await env.{newBindingName || 'MY_WORKER'}.fetch(request)</code>
                    </span>
                  )}
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

      {/* Add Cron Trigger Modal */}
      {showAddCronModal && (
        <div className="fixed inset-0 bg-black/70 backdrop-blur-sm flex items-center justify-center p-4 z-50">
          <div className="bg-zinc-900 border border-zinc-800 rounded-2xl max-w-md w-full p-6 space-y-5 shadow-2xl">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <Clock className="w-5 h-5 text-sky-400" />
                <h3 className="text-lg font-bold text-zinc-100">Add Cron Trigger</h3>
              </div>
              <button
                type="button"
                onClick={() => setShowAddCronModal(false)}
                className="text-zinc-500 hover:text-zinc-300 transition"
              >
                <X className="w-5 h-5" />
              </button>
            </div>

            <form onSubmit={handleCreateCron} className="space-y-4">
              <div className="space-y-1.5">
                <label className="text-xs font-semibold text-zinc-400">Trigger Name</label>
                <input
                  type="text"
                  required
                  placeholder="daily-cleanup"
                  value={newCronName}
                  onChange={e => setNewCronName(e.target.value)}
                  className="w-full bg-zinc-950 border border-zinc-800 rounded-lg p-2.5 text-xs text-zinc-200 outline-none focus:border-sky-500 font-mono"
                />
              </div>

              <div className="space-y-1.5">
                <label className="text-xs font-semibold text-zinc-400">Cron Expression</label>
                <input
                  type="text"
                  required
                  placeholder="*/15 * * * *"
                  value={newCronExpr}
                  onChange={e => setNewCronExpr(e.target.value)}
                  className="w-full bg-zinc-950 border border-zinc-800 rounded-lg p-2.5 text-xs text-zinc-200 outline-none focus:border-sky-500 font-mono"
                />
                <div className="flex flex-wrap gap-1.5 pt-1">
                  {[
                    { label: 'Every min', expr: '* * * * *' },
                    { label: 'Every 5m', expr: '*/5 * * * *' },
                    { label: 'Every 15m', expr: '*/15 * * * *' },
                    { label: 'Hourly', expr: '0 * * * *' },
                    { label: 'Daily (Midnight)', expr: '0 0 * * *' },
                  ].map((preset) => (
                    <button
                      key={preset.expr}
                      type="button"
                      onClick={() => setNewCronExpr(preset.expr)}
                      className="text-[10px] px-2 py-0.5 rounded bg-zinc-800 hover:bg-zinc-700 text-zinc-300 transition"
                    >
                      {preset.label}
                    </button>
                  ))}
                </div>
              </div>

              <div className="space-y-1.5">
                <label className="text-xs font-semibold text-zinc-400">Target Worker</label>
                <input
                  type="text"
                  readOnly
                  value={`${app.name} (${app.id})`}
                  className="w-full bg-zinc-950/60 border border-zinc-800 rounded-lg p-2.5 text-xs text-zinc-400 font-mono select-all"
                />
                <p className="text-[11px] text-zinc-500">
                  Invokes the exported <code className="font-mono text-emerald-400">scheduled(event, env, ctx)</code> handler.
                </p>
              </div>

              <div className="flex items-center justify-end gap-3 pt-3 border-t border-zinc-800">
                <button
                  type="button"
                  onClick={() => setShowAddCronModal(false)}
                  className="px-4 py-2 rounded-lg text-xs font-semibold text-zinc-400 hover:text-zinc-200 hover:bg-zinc-800 transition"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  disabled={isCreatingCron || !newCronName.trim() || !newCronExpr.trim()}
                  className="px-5 py-2 rounded-lg text-xs font-bold bg-sky-600 hover:bg-sky-500 disabled:opacity-50 text-white transition shadow-sm"
                >
                  {isCreatingCron ? 'Scheduling...' : 'Create Trigger'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Rollback Deployment Confirmation Modal */}
      {showRollbackModal && rollbackTargetDep && (
        <div className="fixed inset-0 bg-black/70 backdrop-blur-sm flex items-center justify-center p-4 z-50">
          <div className="bg-zinc-900 border border-amber-900/60 rounded-2xl max-w-md w-full p-6 space-y-4 shadow-2xl">
            <div className="flex items-center gap-3 text-amber-400">
              <div className="p-2.5 rounded-full bg-amber-950/80 border border-amber-800/80">
                <RotateCcw className="w-5 h-5" />
              </div>
              <div>
                <h3 className="text-base font-bold text-zinc-100">Rollback Deployment</h3>
                <p className="text-xs text-zinc-400">Restore previous stable worker release</p>
              </div>
            </div>

            <div className="p-3.5 rounded-xl bg-zinc-950 border border-zinc-800 space-y-2 text-xs">
              <div className="flex items-center justify-between">
                <span className="text-zinc-500 font-medium">Target Version:</span>
                <span className="font-bold text-emerald-400 font-mono text-sm">v{rollbackTargetDep.buildVersion || 1}</span>
              </div>
              <div className="flex items-center justify-between">
                <span className="text-zinc-500 font-medium">Commit:</span>
                <span className="font-mono text-zinc-300">{rollbackTargetDep.commitHash?.slice(0, 7) || 'HEAD'}</span>
              </div>
              <div className="flex items-center justify-between">
                <span className="text-zinc-500 font-medium">Message:</span>
                <span className="text-zinc-300 truncate max-w-[220px]">{rollbackTargetDep.commitMessage || 'Manual deployment'}</span>
              </div>
              <div className="flex items-center justify-between">
                <span className="text-zinc-500 font-medium">Created:</span>
                <span className="text-zinc-400">{new Date(rollbackTargetDep.createdAt).toLocaleString()}</span>
              </div>
            </div>

            <p className="text-xs text-zinc-400 leading-relaxed">
              This will immediately re-point live HTTP isolate traffic for <code className="text-emerald-400 font-mono">{app.subdomain}.localhost:8000</code> to this compiled bundle. Zero downtime, no rebuild required.
            </p>

            {rollbackStatus === 'success' && (
              <div className="p-3 rounded-lg bg-emerald-950/80 border border-emerald-800 text-emerald-300 text-xs font-semibold flex items-center gap-2">
                <Check className="w-4 h-4 text-emerald-400" />
                <span>Successfully rolled back to v{rollbackTargetDep.buildVersion || 1}!</span>
              </div>
            )}

            {rollbackStatus === 'error' && (
              <div className="p-3 rounded-lg bg-rose-950/80 border border-rose-800 text-rose-300 text-xs font-semibold flex items-center gap-2">
                <AlertTriangle className="w-4 h-4 text-rose-400" />
                <span>{rollbackError || 'Failed to rollback deployment'}</span>
              </div>
            )}

            <div className="flex items-center justify-end gap-3 pt-3 border-t border-zinc-800">
              <button
                type="button"
                onClick={() => {
                  setShowRollbackModal(false);
                  setRollbackTargetDep(null);
                  setRollbackStatus('idle');
                }}
                className="px-4 py-2 rounded-lg text-xs font-semibold text-zinc-400 hover:text-zinc-200 hover:bg-zinc-800 transition"
              >
                Cancel
              </button>
              <button
                type="button"
                onClick={handleRollback}
                disabled={isRollingBack || rollbackStatus === 'success'}
                className="px-5 py-2 rounded-lg text-xs font-bold bg-amber-500 hover:bg-amber-400 disabled:opacity-50 text-zinc-950 transition shadow-sm flex items-center gap-1.5"
              >
                {isRollingBack ? (
                  <>
                    <RefreshCw className="w-3.5 h-3.5 animate-spin" />
                    <span>Rolling back...</span>
                  </>
                ) : (
                  <>
                    <RotateCcw className="w-3.5 h-3.5" />
                    <span>Confirm Rollback</span>
                  </>
                )}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Add Variable or Secret Modal */}
      {showAddVarModal && (
        <div className="fixed inset-0 bg-black/70 backdrop-blur-sm flex items-center justify-center p-4 z-50">
          <div className="bg-zinc-900 border border-zinc-800 rounded-2xl max-w-lg w-full p-6 space-y-5 shadow-2xl">
            <div className="flex items-center justify-between border-b border-zinc-800 pb-3">
              <div className="flex items-center gap-2.5">
                <div className="p-2 rounded-lg bg-emerald-950/80 border border-emerald-800 text-emerald-400">
                  <Lock className="w-4 h-4" />
                </div>
                <div>
                  <h3 className="text-sm font-bold text-zinc-100">Add Environment Variable or Secret</h3>
                  <p className="text-xs text-zinc-400">Injected into isolate runtime via <code className="text-emerald-400 font-mono">env</code></p>
                </div>
              </div>
              <button
                type="button"
                onClick={() => {
                  setShowAddVarModal(false);
                  setNewVarKey('');
                  setNewVarValue('');
                  setNewVarIsSecret(false);
                }}
                className="text-zinc-500 hover:text-zinc-300"
              >
                <X className="w-4 h-4" />
              </button>
            </div>

            <form onSubmit={handleAddVariableSubmit} className="space-y-4">
              {/* Key Input */}
              <div>
                <label className="text-xs font-semibold text-zinc-300 block mb-1">
                  Variable Name <span className="text-rose-400">*</span>
                </label>
                <input
                  type="text"
                  required
                  placeholder="e.g. API_KEY, DATABASE_URL, STRIPE_SECRET"
                  value={newVarKey}
                  onChange={e => setNewVarKey(e.target.value.toUpperCase())}
                  className="w-full bg-zinc-950 border border-zinc-800 rounded-lg p-2.5 text-xs font-mono text-zinc-200 outline-none focus:border-emerald-500 uppercase"
                />
                <span className="text-[11px] text-zinc-500 mt-1 block">
                  Must contain only uppercase alphanumeric characters and underscores.
                </span>
              </div>

              {/* Type Selection */}
              <div>
                <label className="text-xs font-semibold text-zinc-300 block mb-1.5">Variable Type</label>
                <div className="grid grid-cols-2 gap-3">
                  <div
                    onClick={() => setNewVarIsSecret(false)}
                    className={`p-3 rounded-xl border cursor-pointer transition flex items-start gap-2.5 ${
                      !newVarIsSecret
                        ? 'bg-zinc-800/80 border-emerald-500 text-zinc-100'
                        : 'bg-zinc-950 border-zinc-800 text-zinc-400 hover:border-zinc-700'
                    }`}
                  >
                    <Code className="w-4 h-4 text-emerald-400 shrink-0 mt-0.5" />
                    <div>
                      <div className="text-xs font-bold">Plaintext Variable</div>
                      <div className="text-[10px] text-zinc-400 mt-0.5">Visible in dashboard and logs</div>
                    </div>
                  </div>

                  <div
                    onClick={() => setNewVarIsSecret(true)}
                    className={`p-3 rounded-xl border cursor-pointer transition flex items-start gap-2.5 ${
                      newVarIsSecret
                        ? 'bg-zinc-800/80 border-amber-500 text-zinc-100'
                        : 'bg-zinc-950 border-zinc-800 text-zinc-400 hover:border-zinc-700'
                    }`}
                  >
                    <Lock className="w-4 h-4 text-amber-400 shrink-0 mt-0.5" />
                    <div>
                      <div className="text-xs font-bold">Encrypted Secret</div>
                      <div className="text-[10px] text-zinc-400 mt-0.5">Masked value for credentials & keys</div>
                    </div>
                  </div>
                </div>
              </div>

              {/* Value Input */}
              <div>
                <div className="flex items-center justify-between mb-1">
                  <label className="text-xs font-semibold text-zinc-300">
                    Value <span className="text-rose-400">*</span>
                  </label>
                  {newVarIsSecret && (
                    <button
                      type="button"
                      onClick={() => setShowNewVarValue(!showNewVarValue)}
                      className="text-[11px] text-zinc-400 hover:text-zinc-200 flex items-center gap-1"
                    >
                      {showNewVarValue ? <EyeOff className="w-3 h-3" /> : <Eye className="w-3 h-3" />}
                      <span>{showNewVarValue ? 'Hide' : 'Reveal'}</span>
                    </button>
                  )}
                </div>
                <div className="relative">
                  <input
                    type={newVarIsSecret && !showNewVarValue ? 'password' : 'text'}
                    required
                    placeholder={newVarIsSecret ? '••••••••••••••••' : 'Value string'}
                    value={newVarValue}
                    onChange={e => setNewVarValue(e.target.value)}
                    className="w-full bg-zinc-950 border border-zinc-800 rounded-lg p-2.5 text-xs font-mono text-zinc-200 outline-none focus:border-emerald-500"
                  />
                </div>
              </div>

              <div className="flex items-center justify-end gap-3 pt-3 border-t border-zinc-800">
                <button
                  type="button"
                  onClick={() => {
                    setShowAddVarModal(false);
                    setNewVarKey('');
                    setNewVarValue('');
                    setNewVarIsSecret(false);
                  }}
                  className="px-4 py-2 rounded-lg text-xs font-semibold text-zinc-400 hover:text-zinc-200 hover:bg-zinc-800 transition"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  disabled={isSavingEnv || !newVarKey.trim()}
                  className="px-5 py-2 rounded-lg text-xs font-bold bg-emerald-500 hover:bg-emerald-400 disabled:opacity-50 text-zinc-950 transition shadow-sm flex items-center gap-1.5"
                >
                  {isSavingEnv ? (
                    <>
                      <RefreshCw className="w-3.5 h-3.5 animate-spin" />
                      <span>Saving...</span>
                    </>
                  ) : (
                    <>
                      <Save className="w-3.5 h-3.5" />
                      <span>Save Variable</span>
                    </>
                  )}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Delete Application Confirmation Modal */}
      {showDeleteConfirm && (
        <div className="fixed inset-0 bg-black/70 backdrop-blur-sm flex items-center justify-center p-4 z-50">
          <div className="bg-zinc-900 border border-rose-900/60 rounded-2xl max-w-md w-full p-6 space-y-4 shadow-2xl">
            <div className="flex items-center gap-3 text-rose-400">
              <div className="p-2.5 rounded-full bg-rose-950/80 border border-rose-800/80">
                <AlertTriangle className="w-5 h-5" />
              </div>
              <h3 className="text-base font-bold text-zinc-100">Delete Application?</h3>
            </div>

            <p className="text-xs text-zinc-400 leading-relaxed">
              Are you sure you want to permanently delete <strong className="text-zinc-200">{app.name}</strong>? This action cannot be undone and will terminate all running isolates and remove deployment history.
            </p>

            <div className="flex items-center justify-end gap-3 pt-3 border-t border-zinc-800">
              <button
                type="button"
                onClick={() => setShowDeleteConfirm(false)}
                className="px-4 py-2 rounded-lg text-xs font-semibold text-zinc-400 hover:text-zinc-200 hover:bg-zinc-800 transition"
              >
                Cancel
              </button>
              <button
                type="button"
                onClick={handleDeleteApp}
                disabled={isDeletingApp}
                className="px-5 py-2 rounded-lg text-xs font-bold bg-rose-600 hover:bg-rose-500 disabled:opacity-50 text-white transition shadow-sm"
              >
                {isDeletingApp ? 'Deleting...' : 'Confirm Delete'}
              </button>
            </div>
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
    case 'service':
      return 'apps';
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
    case 'service':
      return 'Worker RPC';
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
    case 'service':
      return 'Workers';
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
    case 'service':
      return <Cpu className="w-4 h-4 text-emerald-400" />;
    default:
      return <Link2 className="w-4 h-4 text-zinc-400" />;
  }
}
