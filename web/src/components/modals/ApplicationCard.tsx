import React, { useState } from 'react';
import {
  Globe,
  ExternalLink,
  Copy,
  Check,
  Zap,
  GitBranch,
  Play,
  RefreshCw,
  Send,
  Code,
  History,
  Trash2,
  AlertTriangle,
  X,
} from 'lucide-react';
import type { Application } from '../../api/model';
import { useListDeployments } from '../../api/generated/deployments/deployments';

export interface ApplicationCardProps {
  app: Application;
  onOpenApp: (app: Application) => void;
  onDeploy: (appId: string) => void;
  isDeploying: boolean;
  onViewCode: (app: Application) => void;
  onTestApp: (app: Application) => void;
  onViewHistory: (app: Application) => void;
  onDelete?: (appId: string) => Promise<void>;
}

export function ApplicationCard({
  app,
  onOpenApp,
  onDeploy,
  isDeploying,
  onViewCode,
  onTestApp,
  onViewHistory,
  onDelete,
}: ApplicationCardProps) {
  const { data: deploymentsData } = useListDeployments(app.id);
  const deployments = Array.isArray(deploymentsData) ? deploymentsData : [];
  const activeDep = deployments.find(d => d.id === app.activeDeploymentId) || (deployments.length > 0 ? deployments[0] : null);
  const buildVersion = activeDep?.buildVersion;
  const subdomain = app.subdomain || app.name;
  const testUrl = app.testUrl || `http://${subdomain}.localhost:8000`;
  const [copied, setCopied] = useState(false);
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);
  const [isDeleting, setIsDeleting] = useState(false);

  const copyUrl = (e: React.MouseEvent) => {
    e.stopPropagation();
    navigator.clipboard.writeText(testUrl);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const handleConfirmDelete = async () => {
    if (!onDelete) return;
    setIsDeleting(true);
    try {
      await onDelete(app.id);
    } catch (err) {
      console.error('Failed deleting application:', err);
      alert('Failed to delete worker application');
    } finally {
      setIsDeleting(false);
      setShowDeleteConfirm(false);
    }
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
            <p className="text-xs text-zinc-400 font-mono">
              {app.gitRepo} ({app.branch || 'main'}{app.rootDir ? ` • ${app.rootDir}` : ''})
            </p>
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

          {onDelete && (
            <button
              type="button"
              onClick={() => setShowDeleteConfirm(true)}
              disabled={isDeleting}
              title={`Delete worker ${app.name}`}
              className="p-1.5 rounded-lg border border-zinc-800 bg-zinc-900/80 hover:bg-rose-950/50 hover:border-rose-800/80 text-zinc-400 hover:text-rose-400 transition"
            >
              <Trash2 className="w-3.5 h-3.5" />
            </button>
          )}
        </div>
      </div>

      {/* Delete Confirmation Modal */}
      {showDeleteConfirm && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/70 backdrop-blur-sm animate-in fade-in">
          <div className="w-full max-w-md bg-zinc-900 border border-zinc-800 rounded-2xl p-6 shadow-2xl space-y-4">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-3">
                <div className="p-2.5 rounded-full bg-rose-950/80 border border-rose-800/80 text-rose-400">
                  <AlertTriangle className="w-5 h-5" />
                </div>
                <h3 className="text-base font-bold text-zinc-100">Delete Worker?</h3>
              </div>
              <button
                type="button"
                onClick={() => setShowDeleteConfirm(false)}
                className="p-1 rounded-lg text-zinc-400 hover:text-zinc-200 hover:bg-zinc-800 transition"
              >
                <X className="w-4 h-4" />
              </button>
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
                onClick={handleConfirmDelete}
                disabled={isDeleting}
                className="px-5 py-2 rounded-lg text-xs font-bold bg-rose-600 hover:bg-rose-500 disabled:opacity-50 text-white transition shadow-sm"
              >
                {isDeleting ? 'Deleting...' : 'Confirm Delete'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
