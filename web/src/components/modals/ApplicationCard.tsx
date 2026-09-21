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
}

export function ApplicationCard({
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
