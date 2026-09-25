import React from 'react';
import {
  ArrowLeft,
  Play,
  Terminal,
  Code,
  Globe,
  Settings,
  ShieldCheck,
  Zap,
  Layers,
} from 'lucide-react';
import type { Application } from '../../../api/model';
import {
  useActiveTab,
  useApplicationActions,
  useIsDirty,
  DetailTab,
} from '../stores/applicationStore';

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

export interface WorkbenchHeaderProps {
  app: Application;
  onBack: () => void;
  onDeploy: (appId: string) => Promise<void> | void;
  isDeploying: boolean;
  onTestApp: (app: Application) => void;
  onTabChange?: (tab: DetailTab) => void;
}

export function WorkbenchHeader({
  app,
  onBack,
  onDeploy,
  isDeploying,
  onTestApp,
  onTabChange,
}: WorkbenchHeaderProps) {
  const activeTab = useActiveTab();
  const isDirty = useIsDirty();
  const { setActiveTab } = useApplicationActions();

  const handleTabClick = (tab: DetailTab) => {
    setActiveTab(tab);
    onTabChange?.(tab);
  };

  const tabs: { id: DetailTab; label: string; icon: React.ComponentType<{ className?: string }> }[] = [
    { id: 'overview', label: 'Overview', icon: Zap },
    { id: 'code', label: 'Code', icon: Code },
    { id: 'builds', label: 'Builds & Logs', icon: Terminal },
    { id: 'triggers', label: 'Triggers & Domains', icon: Globe },
    { id: 'bindings', label: 'Bindings', icon: Layers },
    { id: 'settings', label: 'Settings', icon: Settings },
  ];

  return (
    <div className="bg-zinc-900/90 border-b border-zinc-800 shrink-0 backdrop-blur">
      <div className="p-4 flex flex-col md:flex-row md:items-center justify-between gap-4">
        <div className="flex items-center gap-3">
          <button
            onClick={onBack}
            data-testid="header-back-btn"
            className="p-1.5 rounded-lg border border-zinc-800 hover:bg-zinc-800 text-zinc-400 hover:text-zinc-200 transition-colors"
          >
            <ArrowLeft className="w-4 h-4" />
          </button>
          <div>
            <div className="flex items-center gap-2">
              <h2
                data-testid="workbench-app-name"
                className="text-lg font-bold text-zinc-100 flex items-center gap-2"
              >
                {app.name}
              </h2>
              <span
                className={`text-[10px] uppercase font-bold tracking-wider px-2 py-0.5 rounded-full ${getDeploymentStatusBadge(
                  app.status || 'pending'
                )}`}
              >
                {app.status || 'pending'}
              </span>
              {isDirty && (
                <span className="text-[10px] font-bold px-2 py-0.5 rounded-full bg-amber-950 text-amber-400 border border-amber-800/80">
                  Unsaved Changes
                </span>
              )}
            </div>
            <p className="text-xs text-zinc-400 font-mono mt-0.5">
              ID: {app.id} • Source: {app.sourceType}
            </p>
          </div>
        </div>

        <div className="flex items-center gap-2">
          <button
            data-testid="header-test-btn"
            onClick={() => onTestApp(app)}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-zinc-800 hover:bg-zinc-700 border border-zinc-700 text-xs font-semibold text-zinc-200 transition"
          >
            <ShieldCheck className="w-3.5 h-3.5 text-blue-400" />
            Test Endpoint
          </button>
          <button
            data-testid="header-deploy-btn"
            onClick={() => onDeploy(app.id)}
            disabled={isDeploying}
            className={`flex items-center gap-1.5 px-3.5 py-1.5 rounded-lg text-xs font-semibold text-white transition ${
              isDeploying
                ? 'bg-zinc-700 cursor-not-allowed opacity-60'
                : 'bg-emerald-600 hover:bg-emerald-500 shadow-md shadow-emerald-950/40'
            }`}
          >
            <Play className={`w-3.5 h-3.5 ${isDeploying ? 'animate-spin' : ''}`} />
            {isDeploying ? 'Deploying...' : 'Deploy Worker'}
          </button>
        </div>
      </div>

      <div className="px-4 flex gap-1 border-t border-zinc-800/60 overflow-x-auto scrollbar-none">
        {tabs.map((tab) => {
          const Icon = tab.icon;
          const isActive = activeTab === tab.id;
          return (
            <button
              key={tab.id}
              data-testid={`tab-btn-${tab.id}`}
              onClick={() => handleTabClick(tab.id)}
              className={`flex items-center gap-2 py-2.5 px-3 text-xs font-medium border-b-2 whitespace-nowrap transition-colors ${
                isActive
                  ? 'border-emerald-500 text-emerald-400 font-semibold'
                  : 'border-transparent text-zinc-400 hover:text-zinc-200 hover:border-zinc-700'
              }`}
            >
              <Icon className="w-3.5 h-3.5" />
              {tab.label}
              {tab.id === 'code' && isDirty && (
                <span className="w-1.5 h-1.5 rounded-full bg-amber-400" />
              )}
            </button>
          );
        })}
      </div>
    </div>
  );
}
