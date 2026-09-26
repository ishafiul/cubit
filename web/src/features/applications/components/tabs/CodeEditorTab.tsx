import { useState, useEffect } from 'react';
import {
  Save,
  RotateCcw,
  Play,
  RefreshCw,
  ExternalLink,
  ShieldCheck,
  GitBranch,
} from 'lucide-react';
import type { Application } from '../../../../api/model';
import { CodeEditor } from '../../../../components/CodeEditor';
import { useUpdateApplication } from '../../../../api/generated/applications/applications';
import {
  useDraftCode,
  useIsDirty,
  useApplicationActions,
} from '../../stores/applicationStore';
import { useToastActions } from '../../../../shared/stores/useToastStore';

export interface CodeEditorTabProps {
  app: Application;
  onDeploy?: (appId: string) => Promise<void> | void;
  isDeploying?: boolean;
  onRefreshApps?: () => void;
}

export function CodeEditorTab({
  app,
  onDeploy,
  isDeploying = false,
  onRefreshApps,
}: CodeEditorTabProps) {
  const draftCode = useDraftCode();
  const isDirty = useIsDirty();
  const { setDraftCode, markCodeSaved, resetDraftCode } = useApplicationActions();
  const { addToast } = useToastActions();

  const updateAppMutation = useUpdateApplication();
  const [isSaving, setIsSaving] = useState(false);

  // Git repository configuration state
  const [gitBranch, setGitBranch] = useState(app.branch || 'main');
  const [isSavingBranch, setIsSavingBranch] = useState(false);

  const [rootDir, setRootDir] = useState(app.rootDir || '');
  const [isSavingRootDir, setIsSavingRootDir] = useState(false);

  useEffect(() => {
    setGitBranch(app.branch || 'main');
    setRootDir(app.rootDir || '');
  }, [app.branch, app.rootDir]);

  const handleSave = async (andDeploy = false) => {
    setIsSaving(true);
    try {
      await updateAppMutation.mutateAsync({
        id: app.id,
        data: {
          inlineCode: draftCode,
        },
      });
      markCodeSaved(draftCode);
      addToast({
        title: andDeploy ? 'Code Saved' : 'Draft Saved',
        description: andDeploy
          ? 'Worker script updated. Starting deployment...'
          : 'Worker script has been updated successfully.',
        variant: 'success',
      });
      onRefreshApps?.();

      if (andDeploy && onDeploy) {
        await onDeploy(app.id);
      }
    } catch (err: any) {
      addToast({
        title: 'Failed to Save Code',
        description: err?.message || 'An error occurred while saving the worker script.',
        variant: 'error',
      });
    } finally {
      setIsSaving(false);
    }
  };

  const handleSaveBranch = async () => {
    setIsSavingBranch(true);
    try {
      await updateAppMutation.mutateAsync({
        id: app.id,
        data: { branch: gitBranch.trim() },
      });
      addToast({
        title: 'Branch Updated',
        description: `Production branch set to ${gitBranch.trim()}`,
        variant: 'success',
      });
      onRefreshApps?.();
    } catch (err: any) {
      addToast({
        title: 'Failed to Update Branch',
        description: err?.message || 'Could not update production branch.',
        variant: 'error',
      });
    } finally {
      setIsSavingBranch(false);
    }
  };

  const handleSaveRootDir = async () => {
    setIsSavingRootDir(true);
    try {
      await updateAppMutation.mutateAsync({
        id: app.id,
        data: { rootDir: rootDir.trim() },
      });
      addToast({
        title: 'Root Directory Updated',
        description: `Worker subfolder set to ${rootDir.trim() || 'root'}`,
        variant: 'success',
      });
      onRefreshApps?.();
    } catch (err: any) {
      addToast({
        title: 'Failed to Update Root Directory',
        description: err?.message || 'Could not update root directory.',
        variant: 'error',
      });
    } finally {
      setIsSavingRootDir(false);
    }
  };

  return (
    <div data-testid="tab-content-code" className="p-6 max-w-6xl mx-auto w-full flex flex-col flex-1">
      {app.sourceType === 'git' ? (
        <div className="p-6 rounded-xl border border-zinc-800 bg-zinc-900/30 space-y-5">
          <div className="flex items-center justify-between border-b border-zinc-800/80 pb-4">
            <div>
              <h3 className="text-base font-semibold text-zinc-100 flex items-center gap-2">
                <GitBranch className="w-5 h-5 text-blue-400" />
                Git Repository Integration
              </h3>
              <p className="text-xs text-zinc-400 mt-1">
                This application is linked to a remote Git repository. Builds are automatically compiled and bundled from Git commits.
              </p>
            </div>
            {app.gitRepo && (
              <a
                href={app.gitRepo}
                target="_blank"
                rel="noopener noreferrer"
                className="px-3.5 py-1.5 rounded-lg border border-zinc-700 bg-zinc-800 hover:bg-zinc-700 text-zinc-200 text-xs font-semibold transition flex items-center gap-1.5"
              >
                <ExternalLink className="w-3.5 h-3.5" /> Open GitHub
              </a>
            )}
          </div>

          <div className="space-y-4 font-mono text-xs">
            <div>
              <label className="text-zinc-400 block mb-1 font-sans text-xs font-semibold">Repository URL</label>
              <input
                type="text"
                readOnly
                value={app.gitRepo || ''}
                className="w-full bg-zinc-950 border border-zinc-800 rounded-lg p-2.5 text-zinc-300 select-all"
              />
            </div>

            <div>
              <label className="text-zinc-400 block mb-1 font-sans text-xs font-semibold">Production Branch</label>
              <div className="flex items-center gap-2">
                <input
                  type="text"
                  value={gitBranch}
                  onChange={(e) => setGitBranch(e.target.value)}
                  className="w-72 bg-zinc-950 border border-zinc-800 rounded-lg p-2 text-zinc-200 outline-none focus:border-emerald-500"
                />
                <button
                  type="button"
                  data-testid="update-branch-btn"
                  onClick={handleSaveBranch}
                  disabled={isSavingBranch || gitBranch.trim() === (app.branch || 'main')}
                  className="px-3.5 py-2 rounded-lg bg-zinc-800 hover:bg-zinc-700 disabled:opacity-50 text-zinc-200 text-xs font-semibold transition"
                >
                  {isSavingBranch ? 'Updating...' : 'Update Branch'}
                </button>
              </div>
            </div>

            <div>
              <label className="text-zinc-400 block mb-1 font-sans text-xs font-semibold">Root Directory / Subfolder</label>
              <div className="flex items-center gap-2">
                <input
                  type="text"
                  placeholder="e.g. packages/worker or empty for root"
                  value={rootDir}
                  onChange={(e) => setRootDir(e.target.value)}
                  className="w-72 bg-zinc-950 border border-zinc-800 rounded-lg p-2 text-zinc-200 outline-none focus:border-emerald-500"
                />
                <button
                  type="button"
                  data-testid="update-rootdir-btn"
                  onClick={handleSaveRootDir}
                  disabled={isSavingRootDir || rootDir.trim() === (app.rootDir || '')}
                  className="px-3.5 py-2 rounded-lg bg-zinc-800 hover:bg-zinc-700 disabled:opacity-50 text-zinc-200 text-xs font-semibold transition"
                >
                  {isSavingRootDir ? 'Updating...' : 'Update Root Dir'}
                </button>
              </div>
              <p className="text-[11px] text-zinc-500 mt-1 font-sans">
                For monorepos, specify the relative path containing your worker's <code className="font-mono text-zinc-400">wrangler.json</code> or source entrypoint.
              </p>
            </div>

            <div className="p-3.5 rounded-xl bg-purple-950/20 border border-purple-900/40 space-y-1 font-sans">
              <div className="flex items-center gap-2 text-xs font-semibold text-purple-300">
                <ShieldCheck className="w-4 h-4 text-purple-400" />
                <span>Push-to-Deploy CI/CD Active</span>
              </div>
              <p className="text-[11px] text-zinc-400">
                Incoming commits pushed to <code className="font-mono text-zinc-200 font-semibold">{app.branch || 'main'}</code> trigger automatic builds and releases via the Cubit webhook mesh.
              </p>
            </div>
          </div>
        </div>
      ) : (
        <>
          <div className="flex items-center justify-between mb-4">
            <div>
              <h3 className="text-sm font-bold text-zinc-100 flex items-center gap-2">
                Worker Script Editor
                {isDirty && (
                  <span className="text-[10px] font-semibold text-amber-400 bg-amber-950/80 px-2 py-0.5 rounded border border-amber-800/80">
                    Unsaved
                  </span>
                )}
              </h3>
              <p className="text-xs text-zinc-400 mt-0.5">
                Standard Cloudflare Worker ES module handling fetch and scheduled events
              </p>
            </div>

            <div className="flex items-center gap-2">
              {isDirty && (
                <button
                  type="button"
                  data-testid="editor-reset-btn"
                  onClick={resetDraftCode}
                  disabled={isSaving}
                  className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-zinc-800 hover:bg-zinc-700 text-xs font-semibold text-zinc-300 transition"
                >
                  <RotateCcw className="w-3.5 h-3.5" />
                  Revert
                </button>
              )}
              <button
                type="button"
                data-testid="editor-save-btn"
                onClick={() => handleSave(false)}
                disabled={!isDirty || isSaving}
                className={`flex items-center gap-1.5 px-3.5 py-1.5 rounded-lg text-xs font-semibold transition ${
                  !isDirty || isSaving
                    ? 'bg-zinc-800 text-zinc-500 cursor-not-allowed border border-zinc-700/50'
                    : 'bg-zinc-800 hover:bg-zinc-700 text-zinc-200 border border-zinc-700'
                }`}
              >
                <Save className={`w-3.5 h-3.5 ${isSaving ? 'animate-spin' : ''}`} />
                {isSaving ? 'Saving...' : 'Save Draft'}
              </button>
              <button
                type="button"
                data-testid="editor-save-and-deploy-btn"
                onClick={() => handleSave(true)}
                disabled={isSaving || isDeploying}
                className={`flex items-center gap-1.5 px-4 py-1.5 rounded-lg text-xs font-bold text-zinc-950 transition shadow ${
                  isSaving || isDeploying
                    ? 'bg-emerald-700 text-zinc-400 cursor-not-allowed'
                    : 'bg-emerald-500 hover:bg-emerald-400 shadow-md shadow-emerald-950/40'
                }`}
              >
                {isSaving || isDeploying ? (
                  <RefreshCw className="w-3.5 h-3.5 animate-spin" />
                ) : (
                  <Play className="w-3.5 h-3.5 fill-current" />
                )}
                <span>{isDeploying ? 'Deploying...' : isSaving ? 'Saving...' : 'Save & Deploy'}</span>
              </button>
            </div>
          </div>

          <div className="flex-1 min-h-[420px] rounded-lg border border-zinc-800 overflow-hidden bg-zinc-950">
            <CodeEditor
              value={draftCode}
              onChange={setDraftCode}
              minHeight="420px"
            />
          </div>
        </>
      )}
    </div>
  );
}
