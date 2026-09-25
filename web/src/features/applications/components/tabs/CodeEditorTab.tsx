import { useState } from 'react';
import { Save, RotateCcw, AlertTriangle } from 'lucide-react';
import type { Application } from '../../../../api/model';
import { CodeEditor } from '../../../../components/CodeEditor';
import { useUpdateApplication } from '../../../../api/generated/applications/applications';
import {
  useDraftCode,
  useIsDirty,
  useApplicationActions,
} from '../../stores/applicationStore';
import { useToastActions } from '../../../../shared/stores/useToastStore';

export function CodeEditorTab({ app }: { app: Application }) {
  const draftCode = useDraftCode();
  const isDirty = useIsDirty();
  const { setDraftCode, markCodeSaved, resetDraftCode } = useApplicationActions();
  const { addToast } = useToastActions();

  const updateAppMutation = useUpdateApplication();
  const [isSaving, setIsSaving] = useState(false);

  const handleSave = async () => {
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
        title: 'Draft Saved',
        description: 'Worker script has been updated successfully.',
        variant: 'success',
      });
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

  return (
    <div data-testid="tab-content-code" className="p-6 max-w-6xl mx-auto w-full flex flex-col flex-1">
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
            Modify your ES module worker entrypoint code
          </p>
        </div>

        <div className="flex items-center gap-2">
          {isDirty && (
            <button
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
            data-testid="editor-save-btn"
            onClick={handleSave}
            disabled={!isDirty || isSaving}
            className={`flex items-center gap-1.5 px-4 py-1.5 rounded-lg text-xs font-semibold text-white transition ${
              !isDirty || isSaving
                ? 'bg-zinc-800 text-zinc-500 cursor-not-allowed border border-zinc-700/50'
                : 'bg-emerald-600 hover:bg-emerald-500 shadow-md shadow-emerald-950/40'
            }`}
          >
            <Save className={`w-3.5 h-3.5 ${isSaving ? 'animate-spin' : ''}`} />
            {isSaving ? 'Saving...' : 'Save Changes'}
          </button>
        </div>
      </div>

      {app.sourceType === 'git' && (
        <div className="mb-4 p-3 rounded-lg bg-blue-950/40 border border-blue-800/60 flex items-center gap-2 text-xs text-blue-300">
          <AlertTriangle className="w-4 h-4 text-blue-400 shrink-0" />
          <span>
            This application is connected to a Git repository (<code className="font-mono">{app.gitRepo}</code>).
            Saving inline code will override the bundle on the next manual deploy.
          </span>
        </div>
      )}

      <div className="flex-1 min-h-[420px] rounded-lg border border-zinc-800 overflow-hidden bg-zinc-950">
        <CodeEditor
          value={draftCode}
          onChange={setDraftCode}
          minHeight="420px"
        />
      </div>
    </div>
  );
}
