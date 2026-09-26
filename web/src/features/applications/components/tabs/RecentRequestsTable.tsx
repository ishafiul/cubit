import { Radio, ArrowRight, Send } from 'lucide-react';
import type { Application } from '../../../../api/model';
import { useGetApplicationMetrics } from '../../../../api/generated/applications/applications';
import { useApplicationActions } from '../../stores/applicationStore';

export interface RecentRequestsTableProps {
  app: Application;
  onTestApp?: (app: Application) => void;
}

export function RecentRequestsTable({ app, onTestApp }: RecentRequestsTableProps) {
  const { setActiveTab, setBuildsViewMode, setExpandedLogId } = useApplicationActions();

  const { data: metrics } = useGetApplicationMetrics(app.id, {
    query: {
      refetchInterval: 5000,
    },
  });

  const recentEvents = metrics?.recentEvents || [];

  const handleOpenTail = (logId?: string) => {
    setActiveTab('builds');
    setBuildsViewMode('tail');
    if (logId) {
      setExpandedLogId(logId);
    }
  };

  return (
    <div
      data-testid="recent-requests-table"
      className="bg-zinc-900 border border-zinc-800 rounded-xl p-5 space-y-4"
    >
      <div className="flex items-center justify-between border-b border-zinc-800/80 pb-3">
        <div className="flex items-center gap-2">
          <Radio className="w-4 h-4 text-emerald-400" />
          <h3 className="text-sm font-bold text-zinc-200">Recent Request Telemetry</h3>
        </div>
        <button
          type="button"
          data-testid="open-full-live-tail-btn"
          onClick={() => handleOpenTail()}
          className="text-xs text-sky-400 hover:text-sky-300 hover:underline flex items-center gap-1 font-semibold transition"
        >
          <span>Open Full Live Tail</span>
          <ArrowRight className="w-3.5 h-3.5" />
        </button>
      </div>

      {recentEvents.length === 0 ? (
        <div className="text-xs text-zinc-500 italic py-8 text-center border border-dashed border-zinc-800 rounded-lg space-y-2">
          <p>No recent request invocations recorded yet.</p>
          {onTestApp && (
            <button
              type="button"
              data-testid="recent-send-test-btn"
              onClick={() => onTestApp(app)}
              className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg border border-emerald-800 bg-emerald-950/60 hover:bg-emerald-950 text-emerald-400 text-xs font-semibold transition"
            >
              <Send className="w-3 h-3" /> Send Test Request
            </button>
          )}
        </div>
      ) : (
        <div className="space-y-2">
          {recentEvents.slice(0, 5).map((ev, idx) => {
            const code = ev.statusCode ?? 200;
            const logId = ev.id || `rec-${idx}`;
            return (
              <div
                key={logId}
                data-testid={`recent-row-${logId}`}
                onClick={() => handleOpenTail(ev.id)}
                className="flex items-center justify-between p-3 rounded-lg bg-zinc-950/70 border border-zinc-800/70 hover:border-zinc-700 hover:bg-zinc-850/50 cursor-pointer transition text-xs select-none"
              >
                <div className="flex items-center gap-3">
                  <span
                    className={`text-[10px] px-1.5 py-0.5 rounded font-bold uppercase ${
                      ev.method === 'GET'
                        ? 'bg-sky-950 text-sky-400 border border-sky-800'
                        : ev.method === 'POST'
                        ? 'bg-emerald-950 text-emerald-400 border border-emerald-800'
                        : ev.method === 'PUT' || ev.method === 'PATCH'
                        ? 'bg-amber-950 text-amber-400 border border-amber-800'
                        : ev.method === 'DELETE'
                        ? 'bg-rose-950 text-rose-400 border border-rose-800'
                        : 'bg-zinc-800 text-zinc-300'
                    }`}
                  >
                    {ev.method}
                  </span>
                  <span className="font-mono text-zinc-200 truncate max-w-xs" title={ev.url || ev.path}>
                    {ev.path || '/'}
                  </span>
                  <span
                    className={`font-mono font-bold text-[11px] ${
                      code >= 200 && code < 300
                        ? 'text-emerald-400'
                        : code >= 400 && code < 500
                        ? 'text-amber-400'
                        : code >= 500
                        ? 'text-rose-400'
                        : 'text-zinc-400'
                    }`}
                  >
                    {code}
                  </span>
                </div>

                <div className="flex items-center gap-4 text-[11px] font-mono text-zinc-400">
                  {Boolean(ev.cf && (ev.cf as any).country) && (
                    <span className="hidden sm:inline px-1.5 py-0.2 rounded bg-sky-950/80 text-sky-300 border border-sky-800/80 text-[10px] font-bold">
                      {String((ev.cf as any).country)} · {String((ev.cf as any).colo || 'SFO')}
                    </span>
                  )}
                  <span>{Math.round(ev.durationMs)}ms</span>
                  <span className="text-zinc-500">
                    {new Date(ev.timestamp).toLocaleTimeString()}
                  </span>
                </div>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
