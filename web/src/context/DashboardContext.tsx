import { createContext, useContext, useState, useEffect, ReactNode } from 'react';
import type { Application, Node, Domain } from '../api/model';
import {
  useListNodes,
  useCreateNode,
  useDrainNode,
  useActivateNode,
  useDeleteNode,
} from '../api/generated/nodes/nodes';
import {
  useListApplications,
  useCreateApplication,
} from '../api/generated/applications/applications';
import { useDeployApplication } from '../api/generated/deployments/deployments';
import { useListDomains, useCreateDomain } from '../api/generated/domains/domains';
import { useGetRuntimeStatus, useUpgradeCelldDaemon } from '../api/generated/runtime/runtime';

export interface LogEntry {
  timestamp: string;
  step: string;
  message: string;
  level: string;
}

export interface DashboardContextType {
  nodes: Node[];
  apps: Application[];
  domains: Domain[];
  runtimeStatus: any;
  refetchAll: () => void;
  refetchApps: () => void;
  refetchDomains: () => void;
  refetchNodes: () => void;
  refetchRuntime: () => void;

  // Modals state
  showNewAppModal: boolean;
  setShowNewAppModal: (show: boolean) => void;
  showAddNodeModal: boolean;
  setShowAddNodeModal: (show: boolean) => void;
  showUpgradeModal: boolean;
  setShowUpgradeModal: (show: boolean) => void;
  showNewDomainModal: boolean;
  setShowNewDomainModal: (show: boolean) => void;
  testingApp: Application | null;
  setTestingApp: (app: Application | null) => void;
  historyApp: Application | null;
  setHistoryApp: (app: Application | null) => void;
  viewingCodeApp: any;
  setViewingCodeApp: (app: any) => void;

  // Deployment & live logs state
  handleDeployApp: (appId: string) => Promise<void>;
  isDeploying: string | null;
  activeDeploymentId: string | null;
  setActiveDeploymentId: (id: string | null) => void;
  deployStatus: string;
  logs: LogEntry[];

  // Form actions
  handleCreateApp: (e: React.FormEvent, data: { name: string; sourceType: 'inline' | 'git'; gitRepo: string; gitBranch: string; rootDir?: string; inlineCode: string; autoDeploy: boolean }) => Promise<string | undefined>;
  handleUpgradeCelld: (targetVersion: string) => Promise<void>;
  handleCreateDomain: (appId: string, host: string) => Promise<void>;
  handleCreateNode: (data: { name: string; ipAddress: string; workerPort?: number; internalPort?: number }) => Promise<void>;
  handleDrainNode: (nodeId: string) => Promise<void>;
  handleActivateNode: (nodeId: string) => Promise<void>;
  handleDeleteNode: (nodeId: string) => Promise<void>;
}

const DashboardContext = createContext<DashboardContextType | null>(null);

export function useDashboard() {
  const ctx = useContext(DashboardContext);
  if (!ctx) {
    throw new Error('useDashboard must be used within a DashboardProvider');
  }
  return ctx;
}

export function DashboardProvider({ children }: { children: ReactNode }) {
  // Orval TanStack Query hooks
  const { data: nodesData, refetch: refetchNodes } = useListNodes();
  const { data: appsData, refetch: refetchApps } = useListApplications();
  const { data: domainsData, refetch: refetchDomains } = useListDomains();
  const { data: runtimeData, refetch: refetchRuntime } = useGetRuntimeStatus();

  const nodes = Array.isArray(nodesData) ? nodesData : [];
  const apps = Array.isArray(appsData) ? appsData : [];
  const domains = Array.isArray(domainsData) ? domainsData : [];
  const runtimeStatus = runtimeData;

  const createApplicationMutation = useCreateApplication();
  const deployApplicationMutation = useDeployApplication();
  const upgradeCelldMutation = useUpgradeCelldDaemon();
  const createDomainMutation = useCreateDomain();
  const createNodeMutation = useCreateNode();
  const drainNodeMutation = useDrainNode();
  const activateNodeMutation = useActivateNode();
  const deleteNodeMutation = useDeleteNode();

  // Modals state
  const [showNewAppModal, setShowNewAppModal] = useState(false);
  const [showAddNodeModal, setShowAddNodeModal] = useState(false);
  const [showUpgradeModal, setShowUpgradeModal] = useState(false);
  const [showNewDomainModal, setShowNewDomainModal] = useState(false);
  const [testingApp, setTestingApp] = useState<Application | null>(null);
  const [historyApp, setHistoryApp] = useState<Application | null>(null);
  const [viewingCodeApp, setViewingCodeApp] = useState<any>(null);

  // Live log streaming
  const [activeDeploymentId, setActiveDeploymentId] = useState<string | null>(null);
  const [deployStatus, setDeployStatus] = useState<string>('');
  const [isDeploying, setIsDeploying] = useState<string | null>(null);
  const [logs, setLogs] = useState<LogEntry[]>([]);

  const refetchAll = () => {
    refetchNodes();
    refetchApps();
    refetchDomains();
    refetchRuntime();
  };

  // Poll runtime status every 5 seconds
  useEffect(() => {
    const interval = setInterval(refetchAll, 5000);
    return () => clearInterval(interval);
  }, []);

  // EventSource live log streaming
  useEffect(() => {
    if (!activeDeploymentId) return;

    setLogs([]);
    setDeployStatus('building');

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
      .catch(err => console.error('Failed fetching initial logs', err));

    fetch(`/api/v1/deployments/${activeDeploymentId}`)
      .then(r => r.json())
      .then((data: any) => {
        if (data && data.status) {
          setDeployStatus(data.status);
        }
      })
      .catch(err => console.error('Failed fetching deployment info', err));

    const es = new EventSource(`/api/v1/deployments/${activeDeploymentId}/logs/stream`);

    es.onmessage = (event) => {
      try {
        const entry = JSON.parse(event.data);
        const normalized: LogEntry = {
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
        console.error('Failed parsing log entry', e);
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
      refetchAll();
    });

    es.onerror = () => {
      es.close();
    };

    return () => {
      es.close();
    };
  }, [activeDeploymentId]);

  const handleDeployApp = async (appId: string) => {
    setIsDeploying(appId);
    try {
      const res = await deployApplicationMutation.mutateAsync({
        id: appId,
        data: { commitHash: 'HEAD' },
      });
      if (res && res.id) {
        setActiveDeploymentId(res.id);
        refetchApps();
      }
    } finally {
      setIsDeploying(null);
    }
  };

  const handleCreateApp = async (
    e: React.FormEvent,
    data: { name: string; sourceType: 'inline' | 'git'; gitRepo: string; gitBranch: string; rootDir?: string; inlineCode: string; autoDeploy: boolean }
  ) => {
    e.preventDefault();
    const payload = data.sourceType === 'git'
      ? { name: data.name, sourceType: 'git' as const, gitRepo: data.gitRepo, branch: data.gitBranch || 'main', rootDir: data.rootDir || '' }
      : { name: data.name, sourceType: 'inline' as const, inlineCode: data.inlineCode || undefined };

    const res = await createApplicationMutation.mutateAsync({
      data: payload,
    });
    setShowNewAppModal(false);
    refetchApps();

    if (res && res.id) {
      if (data.autoDeploy) {
        handleDeployApp(res.id);
      }
      return res.id;
    }
    return undefined;
  };

  const handleUpgradeCelld = async (targetVersion: string) => {
    await upgradeCelldMutation.mutateAsync({
      data: { targetVersion },
    });
    setShowUpgradeModal(false);
    refetchRuntime();
  };

  const handleCreateDomain = async (appId: string, host: string) => {
    await createDomainMutation.mutateAsync({
      data: { applicationId: appId, hostname: host, pathPrefix: '/' },
    });
    setShowNewDomainModal(false);
    refetchDomains();
  };

  const handleCreateNode = async (data: { name: string; ipAddress: string; workerPort?: number; internalPort?: number }) => {
    await createNodeMutation.mutateAsync({
      data: {
        name: data.name,
        ipAddress: data.ipAddress,
        workerPort: data.workerPort || 8080,
        internalPort: data.internalPort || 8081,
      },
    });
    setShowAddNodeModal(false);
    refetchNodes();
  };

  const handleDrainNode = async (nodeId: string) => {
    await drainNodeMutation.mutateAsync({ id: nodeId });
    refetchNodes();
  };

  const handleActivateNode = async (nodeId: string) => {
    await activateNodeMutation.mutateAsync({ id: nodeId });
    refetchNodes();
  };

  const handleDeleteNode = async (nodeId: string) => {
    await deleteNodeMutation.mutateAsync({ id: nodeId });
    refetchNodes();
  };

  const value: DashboardContextType = {
    nodes,
    apps,
    domains,
    runtimeStatus,
    refetchAll,
    refetchApps,
    refetchDomains,
    refetchNodes,
    refetchRuntime,
    showNewAppModal,
    setShowNewAppModal,
    showAddNodeModal,
    setShowAddNodeModal,
    showUpgradeModal,
    setShowUpgradeModal,
    showNewDomainModal,
    setShowNewDomainModal,
    testingApp,
    setTestingApp,
    historyApp,
    setHistoryApp,
    viewingCodeApp,
    setViewingCodeApp,
    handleDeployApp,
    isDeploying,
    activeDeploymentId,
    setActiveDeploymentId,
    deployStatus,
    logs,
    handleCreateApp,
    handleUpgradeCelld,
    handleCreateDomain,
    handleCreateNode,
    handleDrainNode,
    handleActivateNode,
    handleDeleteNode,
  };

  return <DashboardContext.Provider value={value}>{children}</DashboardContext.Provider>;
}
