import { GitBranch, Calendar, Radio } from 'lucide-react';
import type { Application } from '../../../../api/model';
import { MetricsPanel } from './MetricsPanel';

export function OverviewTab({ app }: { app: Application }) {
  return (
    <div data-testid="tab-content-overview" className="p-6 space-y-6 max-w-6xl mx-auto w-full">
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-4">
          <div className="flex items-center gap-2 text-zinc-400 text-xs font-medium">
            <Radio className="w-4 h-4 text-emerald-400" />
            Runtime Status
          </div>
          <p className="mt-2 text-xl font-bold text-zinc-100 capitalize">
            {app.status || 'Active'}
          </p>
          <p className="text-xs text-zinc-500 mt-1">Celld V8 isolate active</p>
        </div>

        <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-4">
          <div className="flex items-center gap-2 text-zinc-400 text-xs font-medium">
            <GitBranch className="w-4 h-4 text-blue-400" />
            Source Type
          </div>
          <p className="mt-2 text-xl font-bold text-zinc-100 capitalize">
            {app.sourceType || 'Inline'}
          </p>
          <p className="text-xs text-zinc-500 mt-1">
            {app.sourceType === 'git' ? app.gitRepo || 'Git repository' : 'Editor code bundle'}
          </p>
        </div>

        <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-4">
          <div className="flex items-center gap-2 text-zinc-400 text-xs font-medium">
            <Calendar className="w-4 h-4 text-purple-400" />
            Created At
          </div>
          <p className="mt-2 text-sm font-semibold text-zinc-200">
            {app.createdAt ? new Date(app.createdAt).toLocaleDateString() : 'Recently'}
          </p>
          <p className="text-xs text-zinc-500 mt-1 font-mono">
            Updated: {app.updatedAt ? new Date(app.updatedAt).toLocaleTimeString() : 'Recently'}
          </p>
        </div>
      </div>

      <MetricsPanel appId={app.id} />
    </div>
  );
}
