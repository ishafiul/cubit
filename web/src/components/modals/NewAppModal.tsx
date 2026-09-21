import React, { useState } from 'react';
import { X, Zap, GitBranch } from 'lucide-react';
import { useDashboard } from '../../context/DashboardContext';

export function NewAppModal() {
  const { showNewAppModal, setShowNewAppModal, handleCreateApp } = useDashboard();

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

  if (!showNewAppModal) return null;

  const onSubmit = async (e: React.FormEvent) => {
    await handleCreateApp(e, {
      name: appName,
      sourceType,
      gitRepo,
      gitBranch,
      inlineCode,
      autoDeploy,
    });
  };

  return (
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

        <form onSubmit={onSubmit} className="space-y-4">
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
  );
}
