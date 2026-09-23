import React, { useState, useEffect } from 'react';
import { X, Zap, GitBranch, Globe, RefreshCw, AlertCircle, ExternalLink, Folder, Search } from 'lucide-react';
import { useNavigate } from '@tanstack/react-router';
import { useDashboard } from '../../context/DashboardContext';

interface GitHubRepoItem {
  id: number;
  name: string;
  fullName: string;
  defaultBranch: string;
  private: boolean;
  cloneUrl: string;
  htmlUrl: string;
}

interface GitHubFolderItem {
  path: string;
  name: string;
  hasWrangler: boolean;
  hasPackageJson: boolean;
}

export function NewAppModal() {
  const navigate = useNavigate();
  const { showNewAppModal, setShowNewAppModal, handleCreateApp } = useDashboard();

  const [appName, setAppName] = useState('');
  const [sourceType, setSourceType] = useState<'inline' | 'github' | 'git'>('inline');
  const [gitRepo, setGitRepo] = useState('');
  const [gitBranch, setGitBranch] = useState('main');
  const [rootDir, setRootDir] = useState('');
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

  // GitHub App integration states
  const [isGitHubConfigured, setIsGitHubConfigured] = useState(false);
  const [gitHubRepos, setGitHubRepos] = useState<GitHubRepoItem[]>([]);
  const [repoSearch, setRepoSearch] = useState('');
  const [isLoadingRepos, setIsLoadingRepos] = useState(false);
  const [availableBranches, setAvailableBranches] = useState<string[]>(['main']);
  const [isLoadingBranches, setIsLoadingBranches] = useState(false);
  const [availableFolders, setAvailableFolders] = useState<GitHubFolderItem[]>([
    { path: '', name: 'Root (/)', hasWrangler: false, hasPackageJson: false },
  ]);
  const [isLoadingFolders, setIsLoadingFolders] = useState(false);

  useEffect(() => {
    if (showNewAppModal) {
      setRepoSearch('');
      // Check GitHub App configuration status
      fetch('/api/v1/github/settings')
        .then(r => r.json())
        .then(data => {
          if (data && data.isConfigured) {
            setIsGitHubConfigured(true);
            loadGitHubRepos();
          } else {
            setIsGitHubConfigured(false);
          }
        })
        .catch(() => setIsGitHubConfigured(false));
    }
  }, [showNewAppModal]);

  const loadGitHubRepos = async () => {
    setIsLoadingRepos(true);
    try {
      const res = await fetch('/api/v1/github/repositories');
      if (res.ok) {
        const data = await res.json();
        setGitHubRepos(Array.isArray(data) ? data : []);
      }
    } catch (_) {
      setGitHubRepos([]);
    } finally {
      setIsLoadingRepos(false);
    }
  };

  const loadFolders = async (owner: string, repo: string, branch: string) => {
    setIsLoadingFolders(true);
    try {
      const res = await fetch(`/api/v1/github/repositories/${owner}/${repo}/folders?branch=${encodeURIComponent(branch)}`);
      if (res.ok) {
        const folders: GitHubFolderItem[] = await res.json();
        if (Array.isArray(folders) && folders.length > 0) {
          setAvailableFolders(folders);
          const wranglerFolder = folders.find(f => f.hasWrangler);
          if (wranglerFolder) {
            setRootDir(wranglerFolder.path);
          } else {
            setRootDir(folders[0].path);
          }
          return;
        }
      }
    } catch (_) {
      // fallback
    } finally {
      setIsLoadingFolders(false);
    }
    setAvailableFolders([{ path: '', name: 'Root (/)', hasWrangler: false, hasPackageJson: false }]);
    setRootDir('');
  };

  const handleSelectGitHubRepo = async (fullName: string) => {
    const selected = gitHubRepos.find(r => r.fullName === fullName);
    if (!selected) return;

    setGitRepo(selected.cloneUrl || `https://github.com/${selected.fullName}.git`);
    if (!appName || gitHubRepos.some(r => r.name === appName)) {
      setAppName(selected.name);
    }
    const defBranch = selected.defaultBranch || 'main';
    setGitBranch(defBranch);

    // Fetch branches and folders for this repository
    const parts = selected.fullName.split('/');
    if (parts.length === 2) {
      setIsLoadingBranches(true);
      try {
        const res = await fetch(`/api/v1/github/repositories/${parts[0]}/${parts[1]}/branches`);
        let chosenBranch = defBranch;
        if (res.ok) {
          const branches = await res.json();
          if (Array.isArray(branches) && branches.length > 0) {
            setAvailableBranches(branches);
            if (branches.includes(defBranch)) {
              chosenBranch = defBranch;
            } else {
              chosenBranch = branches[0];
            }
          }
        }
        setGitBranch(chosenBranch);
        loadFolders(parts[0], parts[1], chosenBranch);
      } catch (_) {
        setAvailableBranches([defBranch]);
        loadFolders(parts[0], parts[1], defBranch);
      } finally {
        setIsLoadingBranches(false);
      }
    }
  };

  const handleBranchChange = (newBranch: string) => {
    setGitBranch(newBranch);
    const selected = gitHubRepos.find(r => r.cloneUrl === gitRepo || `https://github.com/${r.fullName}.git` === gitRepo);
    if (selected) {
      const parts = selected.fullName.split('/');
      if (parts.length === 2) {
        loadFolders(parts[0], parts[1], newBranch);
      }
    }
  };

  if (!showNewAppModal) return null;

  const onSubmit = async (e: React.FormEvent) => {
    const actualSourceType = sourceType === 'inline' ? 'inline' : 'git';
    await handleCreateApp(e, {
      name: appName,
      sourceType: actualSourceType,
      gitRepo,
      gitBranch,
      rootDir,
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
        <div className="flex rounded-lg bg-zinc-950 p-1 border border-zinc-800 gap-1">
          <button
            type="button"
            onClick={() => setSourceType('inline')}
            className={`flex-1 py-1.5 px-2 rounded-md text-xs font-semibold flex items-center justify-center gap-1.5 transition ${
              sourceType === 'inline' ? 'bg-zinc-800 text-emerald-400 shadow' : 'text-zinc-400 hover:text-zinc-200'
            }`}
          >
            <Zap className="w-3.5 h-3.5 text-emerald-400" />
            Hello World
          </button>
          <button
            type="button"
            onClick={() => setSourceType('github')}
            className={`flex-1 py-1.5 px-2 rounded-md text-xs font-semibold flex items-center justify-center gap-1.5 transition ${
              sourceType === 'github' ? 'bg-zinc-800 text-purple-400 shadow' : 'text-zinc-400 hover:text-zinc-200'
            }`}
          >
            <GitBranch className="w-3.5 h-3.5 text-purple-400" />
            GitHub App
          </button>
          <button
            type="button"
            onClick={() => setSourceType('git')}
            className={`flex-1 py-1.5 px-2 rounded-md text-xs font-semibold flex items-center justify-center gap-1.5 transition ${
              sourceType === 'git' ? 'bg-zinc-800 text-blue-400 shadow' : 'text-zinc-400 hover:text-zinc-200'
            }`}
          >
            <Globe className="w-3.5 h-3.5 text-blue-400" />
            Custom Git
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

          {sourceType === 'inline' && (
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
          )}

          {sourceType === 'github' && (
            <>
              {!isGitHubConfigured ? (
                <div className="p-4 rounded-xl border border-dashed border-purple-800/80 bg-purple-950/20 space-y-3">
                  <div className="flex items-start gap-2.5">
                    <AlertCircle className="w-4 h-4 text-purple-400 shrink-0 mt-0.5" />
                    <div>
                      <h5 className="text-xs font-bold text-zinc-200">GitHub App Not Connected</h5>
                      <p className="text-[11px] text-zinc-400 mt-0.5 leading-relaxed">
                        Connect a GitHub App to browse repositories and enable automated deployments on every <code className="text-purple-300 font-mono">git push</code>.
                      </p>
                    </div>
                  </div>
                  <button
                    type="button"
                    onClick={() => {
                      setShowNewAppModal(false);
                      navigate({ to: '/github' });
                    }}
                    className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-purple-600 hover:bg-purple-500 text-white text-xs font-bold transition shadow"
                  >
                    Configure GitHub App <ExternalLink className="w-3.5 h-3.5" />
                  </button>
                </div>
              ) : (
                <>
                  <div>
                    <div className="flex items-center justify-between mb-1">
                      <label className="text-xs font-medium text-zinc-400">Select GitHub Repository</label>
                      <button
                        type="button"
                        onClick={loadGitHubRepos}
                        disabled={isLoadingRepos}
                        className="text-[11px] text-purple-400 hover:text-purple-300 flex items-center gap-1 transition"
                      >
                        <RefreshCw className={`w-3 h-3 ${isLoadingRepos ? 'animate-spin' : ''}`} />
                        Refresh
                      </button>
                    </div>

                    {isLoadingRepos ? (
                      <div className="p-3 text-center text-xs text-zinc-500">Loading repositories...</div>
                    ) : (
                      <div className="space-y-2">
                        {/* Search / Filter Input */}
                        <div className="relative">
                          <Search className="w-3.5 h-3.5 text-zinc-500 absolute left-2.5 top-1/2 -translate-y-1/2 pointer-events-none" />
                          <input
                            type="text"
                            placeholder="Search repositories (e.g. org/repo)..."
                            value={repoSearch}
                            onChange={e => setRepoSearch(e.target.value)}
                            className="w-full bg-zinc-950 border border-zinc-800 rounded-lg pl-8 pr-16 py-1.5 text-xs text-zinc-100 placeholder-zinc-500 focus:outline-none focus:border-purple-500 font-mono"
                          />
                          <div className="absolute right-2 top-1/2 -translate-y-1/2 flex items-center gap-1.5">
                            {repoSearch && (
                              <button
                                type="button"
                                onClick={() => setRepoSearch('')}
                                className="text-zinc-500 hover:text-zinc-300 p-0.5 rounded transition"
                                title="Clear filter"
                              >
                                <X className="w-3 h-3" />
                              </button>
                            )}
                            <span className="text-[10px] text-zinc-500 font-mono">
                              {gitHubRepos.filter(r =>
                                r.fullName.toLowerCase().includes(repoSearch.toLowerCase()) ||
                                r.name.toLowerCase().includes(repoSearch.toLowerCase())
                              ).length}/{gitHubRepos.length}
                            </span>
                          </div>
                        </div>

                        {/* Dropdown */}
                        {(() => {
                          const filtered = gitHubRepos.filter(r =>
                            r.fullName.toLowerCase().includes(repoSearch.toLowerCase()) ||
                            r.name.toLowerCase().includes(repoSearch.toLowerCase())
                          );
                          const selected = gitHubRepos.find(r => r.cloneUrl === gitRepo || `https://github.com/${r.fullName}.git` === gitRepo);

                          return (
                            <>
                              <select
                                required
                                value={selected?.fullName || ''}
                                onChange={e => handleSelectGitHubRepo(e.target.value)}
                                className="w-full bg-zinc-950 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-100 focus:outline-none focus:border-purple-500 font-mono"
                              >
                                <option value="">
                                  {filtered.length === 0 ? '-- No matching repositories --' : '-- Choose a repository --'}
                                </option>
                                {selected && !filtered.some(r => r.id === selected.id) && (
                                  <option key={selected.id} value={selected.fullName}>
                                    {selected.fullName} ({selected.private ? 'Private' : 'Public'}) [Selected]
                                  </option>
                                )}
                                {filtered.map(r => (
                                  <option key={r.id} value={r.fullName}>
                                    {r.fullName} ({r.private ? 'Private' : 'Public'})
                                  </option>
                                ))}
                              </select>

                              {selected && (
                                <div className="flex items-center justify-between text-[11px] px-2.5 py-1.5 rounded-md bg-purple-950/30 border border-purple-900/40 text-purple-200">
                                  <span className="truncate font-mono">
                                    <span className="text-zinc-400 font-sans">Selected: </span>
                                    {selected.fullName}
                                  </span>
                                  <span className={`px-1.5 py-0.5 rounded text-[10px] font-semibold ${selected.private ? 'bg-amber-950/80 text-amber-300 border border-amber-800/60' : 'bg-emerald-950/80 text-emerald-300 border border-emerald-800/60'}`}>
                                    {selected.private ? 'Private' : 'Public'}
                                  </span>
                                </div>
                              )}
                            </>
                          );
                        })()}
                      </div>
                    )}
                  </div>

                  <div>
                    <label className="text-xs font-medium text-zinc-400 mb-1 block">Production Branch</label>
                    {isLoadingBranches ? (
                      <div className="text-xs text-zinc-500 p-2">Loading branches...</div>
                    ) : availableBranches.length > 1 ? (
                      <select
                        value={gitBranch}
                        onChange={e => handleBranchChange(e.target.value)}
                        className="w-full bg-zinc-950 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-100 focus:outline-none focus:border-purple-500 font-mono"
                      >
                        {availableBranches.map(b => (
                          <option key={b} value={b}>{b}</option>
                        ))}
                      </select>
                    ) : (
                      <input
                        type="text"
                        value={gitBranch}
                        onChange={e => handleBranchChange(e.target.value)}
                        className="w-full bg-zinc-950 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-100 focus:outline-none focus:border-purple-500 font-mono"
                      />
                    )}
                  </div>

                  <div>
                    <div className="flex items-center justify-between mb-1">
                      <label className="text-xs font-medium text-zinc-400 flex items-center gap-1.5">
                        <Folder className="w-3.5 h-3.5 text-purple-400" />
                        Worker Root Directory / Subfolder
                      </label>
                      {availableFolders.length > 1 && (
                        <span className="text-[10px] text-purple-400 font-medium">
                          {availableFolders.some(f => f.hasWrangler) ? '⚡ Worker detected' : `${availableFolders.length} folders discovered`}
                        </span>
                      )}
                    </div>
                    {isLoadingFolders ? (
                      <div className="text-xs text-zinc-500 p-2">Scanning directories and wrangler configs...</div>
                    ) : (
                      <div className="space-y-2">
                        <select
                          value={rootDir}
                          onChange={e => setRootDir(e.target.value)}
                          className="w-full bg-zinc-950 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-100 focus:outline-none focus:border-purple-500 font-mono"
                        >
                          {availableFolders.map(f => (
                            <option key={f.path} value={f.path}>
                              {f.path === '' ? '/ (Repository Root)' : f.path}
                              {f.hasWrangler ? ' ⚡ (wrangler config)' : f.hasPackageJson ? ' 📦 (package.json)' : ''}
                            </option>
                          ))}
                        </select>
                        <div className="flex items-center gap-2">
                          <input
                            type="text"
                            placeholder="Custom subfolder path (e.g. packages/worker)"
                            value={rootDir}
                            onChange={e => setRootDir(e.target.value)}
                            className="flex-1 bg-zinc-950/60 border border-zinc-800/80 rounded-md px-2.5 py-1.5 text-[11px] text-zinc-300 placeholder-zinc-600 focus:outline-none focus:border-purple-500 font-mono"
                          />
                          <span className="text-[10px] text-zinc-500 whitespace-nowrap">or custom path</span>
                        </div>
                      </div>
                    )}
                    <p className="text-[10px] text-zinc-500 mt-1">
                      For monorepos, choose the subfolder containing your worker's <code className="text-zinc-400">wrangler.json</code> or source files.
                    </p>
                  </div>

                  <div className="p-3 rounded-xl bg-purple-950/20 border border-purple-900/40 space-y-2">
                    <label className="flex items-center gap-2.5 text-xs text-zinc-300 cursor-pointer select-none">
                      <input
                        type="checkbox"
                        checked={autoDeploy}
                        onChange={e => setAutoDeploy(e.target.checked)}
                        className="rounded bg-zinc-950 border-zinc-800 text-purple-500 focus:ring-0 focus:ring-offset-0"
                      />
                      <span className="font-semibold text-purple-300">
                        Auto-deploy on git push (CI/CD Webhook)
                      </span>
                    </label>
                    <p className="text-[10px] text-zinc-500 pl-6">
                      Every commit pushed to <code className="text-zinc-300">{gitBranch}</code> will automatically trigger a new versioned release with live build logs.
                    </p>
                  </div>
                </>
              )}
            </>
          )}

          {sourceType === 'git' && (
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

              <div>
                <label className="text-xs font-medium text-zinc-400 mb-1 block">Root Directory / Subfolder (Optional)</label>
                <input
                  type="text"
                  placeholder="e.g. packages/worker or leave empty for repository root"
                  value={rootDir}
                  onChange={e => setRootDir(e.target.value)}
                  className="w-full bg-zinc-950 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-100 focus:outline-none focus:border-emerald-500 font-mono"
                />
                <p className="text-[10px] text-zinc-500 mt-1">
                  For monorepos, specify the relative path to the worker folder.
                </p>
              </div>

              <label className="flex items-center gap-2.5 text-xs text-zinc-400 cursor-pointer select-none">
                <input
                  type="checkbox"
                  checked={autoDeploy}
                  onChange={e => setAutoDeploy(e.target.checked)}
                  className="rounded bg-zinc-950 border-zinc-800 text-emerald-500 focus:ring-0 focus:ring-offset-0"
                />
                <span>Auto-deploy on git push</span>
              </label>
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
              disabled={sourceType === 'github' && !isGitHubConfigured}
              className="px-4 py-2 rounded-lg bg-emerald-500 hover:bg-emerald-600 disabled:opacity-50 text-zinc-950 text-sm font-semibold transition"
            >
              Create & Deploy
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
