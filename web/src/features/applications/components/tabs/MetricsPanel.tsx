import {
  Activity,
  Clock,
  ShieldCheck,
  Flame,
  Globe,
  Radio,
} from 'lucide-react';
import { useGetApplicationMetrics } from '../../../../api/generated/applications/applications';
import {
  useActiveMetricsInterval,
  useApplicationActions,
  MetricsInterval,
} from '../../stores/applicationStore';
import type { ApplicationMetrics } from '../../../../api/model';

export function MetricsPanel({ appId }: { appId: string }) {
  const interval = useActiveMetricsInterval();
  const { setActiveMetricsInterval } = useApplicationActions();

  const { data: metricsData, isLoading } = useGetApplicationMetrics(appId, {
    query: {
      refetchInterval: 5000,
    },
  });
  const metrics = metricsData as ApplicationMetrics | undefined;

  const intervals: MetricsInterval[] = ['1h', '24h', '7d', '30d'];

  return (
    <div data-testid="metrics-panel" className="space-y-6">
      {/* Metrics Card Row */}
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
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-5 gap-4">
            {/* Total Requests */}
            <div className="bg-zinc-950/60 p-4 rounded-lg border border-zinc-800/60 space-y-1">
              <div className="flex items-center justify-between text-xs text-zinc-400">
                <span className="font-semibold uppercase tracking-wider text-[10px]">Total Requests</span>
                <Activity className="w-3.5 h-3.5 text-emerald-400" />
              </div>
              <p className="text-2xl font-bold font-mono text-zinc-100 mt-1">
                {(metrics?.totalRequests ?? 0).toLocaleString()}
              </p>
              <p className="text-[10px] text-zinc-500">Tracked via celld runtime</p>
            </div>

            {/* Success Rate */}
            <div className="bg-zinc-950/60 p-4 rounded-lg border border-zinc-800/60 space-y-1">
              <div className="flex items-center justify-between text-xs text-zinc-400">
                <span className="font-semibold uppercase tracking-wider text-[10px]">Success Rate</span>
                <ShieldCheck className="w-3.5 h-3.5 text-emerald-400" />
              </div>
              <p className="text-2xl font-bold font-mono text-emerald-400 mt-1">
                {metrics?.successRate !== undefined ? metrics.successRate.toFixed(1) : '100.0'}%
              </p>
              <div className="flex items-center gap-1.5 text-[10px] text-zinc-500">
                <span
                  className={`w-1.5 h-1.5 rounded-full ${
                    (metrics?.errorRate ?? 0) === 0 ? 'bg-emerald-500' : 'bg-amber-500'
                  }`}
                />
                <span>
                  {(metrics?.errorRate ?? 0) === 0 ? 'Healthy edge availability' : 'Errors detected'}
                </span>
              </div>
            </div>

            {/* HTTP Status Breakdown */}
            <div className="bg-zinc-950/60 p-4 rounded-lg border border-zinc-800/60 space-y-2">
              <div className="flex items-center justify-between text-xs text-zinc-400">
                <span className="font-semibold uppercase tracking-wider text-[10px]">Response Codes</span>
                <Activity className="w-3.5 h-3.5 text-sky-400" />
              </div>
              <div className="flex items-center gap-2 text-xs font-mono">
                <span className="text-emerald-400 font-bold">{metrics?.status2xx ?? 0}</span>
                <span className="text-[10px] text-zinc-500">2xx</span>
                <span className="text-amber-400 font-bold ml-1">{metrics?.status4xx ?? 0}</span>
                <span className="text-[10px] text-zinc-500">4xx</span>
                <span className="text-rose-400 font-bold ml-1">{metrics?.status5xx ?? 0}</span>
                <span className="text-[10px] text-zinc-500">5xx</span>
              </div>
              <div className="w-full h-1.5 bg-zinc-850 rounded-full overflow-hidden flex">
                {(metrics?.totalRequests ?? 0) > 0 ? (
                  <>
                    <div
                      style={{
                        width: `${((metrics?.status2xx ?? 0) / (metrics?.totalRequests || 1)) * 100}%`,
                      }}
                      className="bg-emerald-500 h-full"
                      title={`2xx: ${metrics?.status2xx ?? 0}`}
                    />
                    <div
                      style={{
                        width: `${((metrics?.status4xx ?? 0) / (metrics?.totalRequests || 1)) * 100}%`,
                      }}
                      className="bg-amber-500 h-full"
                      title={`4xx: ${metrics?.status4xx ?? 0}`}
                    />
                    <div
                      style={{
                        width: `${((metrics?.status5xx ?? 0) / (metrics?.totalRequests || 1)) * 100}%`,
                      }}
                      className="bg-rose-500 h-full"
                      title={`5xx: ${metrics?.status5xx ?? 0}`}
                    />
                  </>
                ) : (
                  <div className="bg-zinc-700 h-full w-full" />
                )}
              </div>
            </div>

            {/* Latency */}
            <div className="bg-zinc-950/60 p-4 rounded-lg border border-zinc-800/60 space-y-1">
              <div className="flex items-center justify-between text-xs text-zinc-400">
                <span className="font-semibold uppercase tracking-wider text-[10px]">Latency</span>
                <Clock className="w-3.5 h-3.5 text-purple-400" />
              </div>
              <div className="flex items-baseline gap-2 font-mono mt-1">
                <div>
                  <span className="text-xl font-bold text-zinc-100">
                    {metrics?.avgDurationMs ? Math.round(metrics.avgDurationMs) : 0}ms
                  </span>
                  <span className="text-[10px] text-zinc-500 ml-0.5">avg</span>
                </div>
                <span className="text-zinc-600">/</span>
                <div>
                  <span className="text-xl font-bold text-purple-400">
                    {metrics?.p99DurationMs ? Math.round(metrics.p99DurationMs) : 0}ms
                  </span>
                  <span className="text-[10px] text-zinc-500 ml-0.5">p99</span>
                </div>
              </div>
              <p className="text-[10px] text-zinc-500">Isolate execution time</p>
            </div>

            {/* Last Invocation */}
            <div className="bg-zinc-950/60 p-4 rounded-lg border border-zinc-800/60 space-y-1">
              <div className="flex items-center justify-between text-xs text-zinc-400">
                <span className="font-semibold uppercase tracking-wider text-[10px]">Last Invocation</span>
                <Flame className="w-3.5 h-3.5 text-amber-400" />
              </div>
              <p className="text-sm font-semibold font-mono text-zinc-200 mt-2 truncate">
                {metrics?.lastInvokedAt
                  ? new Date(metrics.lastInvokedAt).toLocaleTimeString()
                  : 'Never'}
              </p>
              <div className="flex items-center gap-1.5 text-[10px] text-zinc-500 mt-1">
                <span className="w-1.5 h-1.5 rounded-full bg-emerald-500 animate-pulse" />
                <span>celld worker active</span>
              </div>
            </div>
          </div>
        )}
      </div>

      {/* Edge Geolocation & Datacenter PoPs Grid */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Traffic by Geolocation (Country) */}
        <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-5 space-y-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <Globe className="w-4 h-4 text-sky-400" />
              <h3 className="text-sm font-bold text-zinc-200">Traffic by Geolocation (request.cf)</h3>
            </div>
            <span className="text-xs text-zinc-500 font-mono">
              {Object.keys(metrics?.requestsByCountry || {}).length} Countries
            </span>
          </div>

          {Object.keys(metrics?.requestsByCountry || {}).length === 0 ? (
            <div className="text-xs text-zinc-500 italic py-6 text-center border border-dashed border-zinc-800 rounded-lg">
              No edge geolocation traffic recorded yet. Requests to this worker will stream country telemetry here.
            </div>
          ) : (
            <div className="space-y-3">
              {Object.entries(metrics?.requestsByCountry || {})
                .sort(([, a], [, b]) => (b as number) - (a as number))
                .slice(0, 5)
                .map(([countryCode, count]) => {
                  const total = metrics?.totalRequests || 1;
                  const pct = Math.round(((count as number) / total) * 100);
                  return (
                    <div key={countryCode} className="space-y-1">
                      <div className="flex items-center justify-between text-xs">
                        <div className="flex items-center gap-2">
                          <span className="font-bold font-mono px-1.5 py-0.5 rounded bg-zinc-950 border border-zinc-800 text-sky-300 text-[10px]">
                            {countryCode}
                          </span>
                          <span className="text-zinc-300 font-medium">
                            {countryCode === 'US'
                              ? 'United States'
                              : countryCode === 'GB'
                              ? 'United Kingdom'
                              : countryCode === 'DE'
                              ? 'Germany'
                              : countryCode === 'JP'
                              ? 'Japan'
                              : countryCode === 'FR'
                              ? 'France'
                              : countryCode}
                          </span>
                        </div>
                        <div className="flex items-center gap-2 font-mono text-zinc-400">
                          <span>{(count as number).toLocaleString()} reqs</span>
                          <span className="text-zinc-500 text-[11px]">({pct}%)</span>
                        </div>
                      </div>
                      <div className="w-full h-1.5 bg-zinc-950 rounded-full overflow-hidden border border-zinc-800/40">
                        <div
                          style={{ width: `${Math.max(pct, 2)}%` }}
                          className="bg-sky-500 h-full rounded-full transition-all duration-500"
                        />
                      </div>
                    </div>
                  );
                })}
            </div>
          )}
        </div>

        {/* Edge Points of Presence (Colos) */}
        <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-5 space-y-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <Radio className="w-4 h-4 text-emerald-400" />
              <h3 className="text-sm font-bold text-zinc-200">Edge Points of Presence (Colos)</h3>
            </div>
            <span className="text-xs text-zinc-500 font-mono">
              {Object.keys(metrics?.requestsByColo || {}).length} PoPs
            </span>
          </div>

          {Object.keys(metrics?.requestsByColo || {}).length === 0 ? (
            <div className="text-xs text-zinc-500 italic py-6 text-center border border-dashed border-zinc-800 rounded-lg">
              No edge datacenter traffic recorded yet. Requests will record Cloudflare-compatible airport code PoPs.
            </div>
          ) : (
            <div className="space-y-3">
              {Object.entries(metrics?.requestsByColo || {})
                .sort(([, a], [, b]) => (b as number) - (a as number))
                .slice(0, 5)
                .map(([colo, count]) => {
                  const total = metrics?.totalRequests || 1;
                  const pct = Math.round(((count as number) / total) * 100);
                  return (
                    <div key={colo} className="space-y-1">
                      <div className="flex items-center justify-between text-xs">
                        <div className="flex items-center gap-2">
                          <span className="font-bold font-mono px-1.5 py-0.5 rounded bg-zinc-950 border border-zinc-800 text-emerald-400 text-[10px]">
                            {colo}
                          </span>
                          <span className="text-zinc-300 font-medium">
                            {colo === 'SFO'
                              ? 'San Francisco, CA'
                              : colo === 'IAD'
                              ? 'Washington, DC'
                              : colo === 'LHR'
                              ? 'London, UK'
                              : colo === 'FRA'
                              ? 'Frankfurt, DE'
                              : `${colo} Edge Node`}
                          </span>
                        </div>
                        <div className="flex items-center gap-2 font-mono text-zinc-400">
                          <span>{(count as number).toLocaleString()} reqs</span>
                          <span className="text-zinc-500 text-[11px]">({pct}%)</span>
                        </div>
                      </div>
                      <div className="w-full h-1.5 bg-zinc-950 rounded-full overflow-hidden border border-zinc-800/40">
                        <div
                          style={{ width: `${Math.max(pct, 2)}%` }}
                          className="bg-emerald-500 h-full rounded-full transition-all duration-500"
                        />
                      </div>
                    </div>
                  );
                })}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
