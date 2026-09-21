import { X } from 'lucide-react';
import { useDashboard } from '../../context/DashboardContext';

export function ViewCodeModal() {
  const { viewingCodeApp, setViewingCodeApp } = useDashboard();

  if (!viewingCodeApp) return null;

  return (
    <div className="fixed inset-0 bg-black/60 backdrop-blur-sm flex items-center justify-center p-4 z-50">
      <div className="bg-zinc-900 border border-zinc-800 rounded-2xl max-w-lg w-full p-6 space-y-4 shadow-2xl">
        <div className="flex items-center justify-between">
          <div>
            <h3 className="text-lg font-bold">{viewingCodeApp.name}</h3>
            <p className="text-xs text-zinc-400">Inline Worker Script</p>
          </div>
          <button
            type="button"
            onClick={() => setViewingCodeApp(null)}
            className="text-zinc-500 hover:text-zinc-300 transition"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        <pre className="p-4 rounded-xl bg-zinc-950 border border-zinc-800/80 text-xs font-mono text-zinc-200 overflow-x-auto max-h-80 leading-relaxed">
          <code>{viewingCodeApp.inlineCode || '// No inline code found'}</code>
        </pre>

        <div className="flex justify-end">
          <button
            type="button"
            onClick={() => setViewingCodeApp(null)}
            className="px-4 py-2 rounded-lg border border-zinc-800 text-sm font-medium hover:bg-zinc-800 transition"
          >
            Close
          </button>
        </div>
      </div>
    </div>
  );
}
