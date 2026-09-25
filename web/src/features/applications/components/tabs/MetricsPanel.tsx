import { Activity, Clock } from 'lucide-react';
import { useGetApplicationMetrics } from '../../../../api/generated/applications/applications';
import {
  useActiveMetricsInterval,
  useApplicationActions,
  MetricsInterval,
} from '../../stores/applicationStore';

export function MetricsPanel({ appId }: { appId: string }) {
  const interval = useActiveMetricsInterval();
  const { setActiveMetricsInterval } = useApplicationActions();

  const { data: metrics, isLoading } = useGetApplicationMetrics(appId);

  const intervals: MetricsInterval[] = ['1h', '24h', '7d', '30d'];

  return (
    <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-6">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 mb-6">
        <div>
          <h3 className="text-sm font-bold text-zinc-100 flex items-center gap-2">
            <Activity className="w-4 h-4 text-emerald-400" />
            Performance & Request Telemetry
          </h3>
          <p className="text-xs text-zinc-400 mt-0.5">
            Aggregated metrics for workers running in Celld isolates
          </p>
        </div>

        <div className="flex items-center gap-1 bg-zinc-950 p-1 rounded-lg border border-zinc-800">
          <Clock className="w-3.5 h-3.5 text-zinc-500 ml-1 mr-0.5" />
          {intervals.map((i) => (
            <button
              key={i}
              onClick={() => setActiveMetricsInterval(i)}
              className={`px-2.5 py-1 text-xs font-semibold rounded ${
                interval === i
                  ? 'bg-zinc-800 text-emerald-400'
                  : 'text-zinc-400 hover:text-zinc-200'
              }`}
            >
              {i}
            </button>
          ))}
        </div>
      </div>

      {isLoading ? (
        <div className="h-40 flex items-center justify-center text-xs text-zinc-500">
          Loading metrics...
        </div>
      ) : (
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
          <div className="bg-zinc-950/60 p-4 rounded-lg border border-zinc-800/60">
            <span className="text-xs text-zinc-400">Total Requests</span>
            <p className="text-2xl font-bold text-zinc-100 mt-1">
              {metrics?.totalRequests ?? 0}
            </p>
          </div>
          <div className="bg-zinc-950/60 p-4 rounded-lg border border-zinc-800/60">
            <span className="text-xs text-zinc-400">Error Count (4xx/5xx)</span>
            <p className="text-2xl font-bold text-rose-400 mt-1">
              {(metrics?.status4xx ?? 0) + (metrics?.status5xx ?? 0)}
            </p>
          </div>
          <div className="bg-zinc-950/60 p-4 rounded-lg border border-zinc-800/60">
            <span className="text-xs text-zinc-400">Avg Duration</span>
            <p className="text-2xl font-bold text-blue-400 mt-1">
              {metrics?.avgDurationMs ? `${metrics.avgDurationMs.toFixed(1)}ms` : '0ms'}
            </p>
          </div>
        </div>
      )}
    </div>
  );
}
