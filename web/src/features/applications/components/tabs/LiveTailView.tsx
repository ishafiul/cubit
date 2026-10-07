import { useState, useEffect, useMemo } from 'react';
import {
  Radio,
  Pause,
  Play,
  Trash2,
  Search,
  X,
  ChevronDown,
  ChevronRight,
  Copy,
  Check,
  Send,
  ListFilter,
  FileText,
  Terminal,
  Globe,
  Braces,
} from 'lucide-react';
import type { Application, RequestLogEvent } from '../../../../api/model';
import {
  useExpandedLogId,
  useApplicationActions,
} from '../../stores/applicationStore';
import { getSubdomainUrl } from '../../../../shared/utils/subdomain';

export interface CfTelemetry {
  country?: string;
  city?: string;
  colo?: string;
  asn?: number | string;
  httpProtocol?: string;
  tlsVersion?: string;
}

export type OtelSeverity = 'DEBUG' | 'INFO' | 'WARN' | 'ERROR';

export interface OtelSeverityBadge {
  label: OtelSeverity;
  className: string;
}

export function getOtelSeverity(level?: string): OtelSeverityBadge {
  const norm = (level || 'info').toLowerCase();
  if (norm === 'error' || norm === 'fatal') {
    return {
      label: 'ERROR',
      className: 'bg-rose-950 text-rose-400 border border-rose-800',
    };
  }
  if (norm === 'warn' || norm === 'warning') {
    return {
      label: 'WARN',
      className: 'bg-amber-950 text-amber-400 border border-amber-800',
    };
  }
  if (norm === 'debug' || norm === 'trace') {
    return {
      label: 'DEBUG',
      className: 'bg-sky-950 text-sky-400 border border-sky-800',
    };
  }
  return {
    label: 'INFO',
    className: 'bg-emerald-950 text-emerald-400 border border-emerald-800',
  };
}

export interface LiveTailViewProps {
  app: Application;
  onTestApp?: (app: Application) => void;
}

export function LiveTailView({ app, onTestApp }: LiveTailViewProps) {
  const expandedLogId = useExpandedLogId();
  const { setExpandedLogId } = useApplicationActions();

  const [liveLogs, setLiveLogs] = useState<RequestLogEvent[]>([]);
  const [isLiveStreaming, setIsLiveStreaming] = useState(true);
  const [liveConnected, setLiveConnected] = useState(false);
  const [logSearchTerm, setLogSearchTerm] = useState('');
  const [logMethodFilter, setLogMethodFilter] = useState<'ALL' | 'GET' | 'POST' | 'PUT' | 'DELETE'>('ALL');
  const [logStatusFilter, setLogStatusFilter] = useState<'ALL' | '2xx' | '4xx' | '5xx'>('ALL');
  const [activeLogDetailTab, setActiveLogDetailTab] = useState<'headers' | 'payload' | 'logs' | 'cf' | 'json'>('headers');
  const [copiedLogSection, setCopiedLogSection] = useState<string | null>(null);

  const testUrl = getSubdomainUrl(app.subdomain || app.name, app.testUrl);

  const handleCopyLogSection = (text: string, sectionKey: string) => {
    navigator.clipboard.writeText(text);
    setCopiedLogSection(sectionKey);
    setTimeout(() => {
      setCopiedLogSection((prev) => (prev === sectionKey ? null : prev));
    }, 2000);
  };

  const formatJsonPayload = (raw: unknown): string => {
    if (raw === undefined || raw === null) return '';
    if (typeof raw === 'object') {
      try {
        return JSON.stringify(raw, null, 2);
      } catch {
        return String(raw);
      }
    }
    if (typeof raw === 'string') {
      try {
        const parsed = JSON.parse(raw);
        return JSON.stringify(parsed, null, 2);
      } catch {
        return raw;
      }
    }
    return String(raw);
  };

  // Real-time SSE connection to /api/v1/applications/:id/logs/stream
  useEffect(() => {
    if (!isLiveStreaming || !app.id) {
      setLiveConnected(false);
      return;
    }

    if (typeof EventSource === 'undefined') {
      return;
    }

    const es = new EventSource(`/api/v1/applications/${app.id}/logs/stream`);
    es.onopen = () => {
      setLiveConnected(true);
    };
    es.onmessage = (e) => {
      try {
        const item: RequestLogEvent = JSON.parse(e.data);
        setLiveLogs((prev) => [item, ...prev].slice(0, 200));
      } catch (err) {
        console.error('Error parsing live log event:', err);
      }
    };
    es.onerror = () => {
      setLiveConnected(false);
    };

    return () => {
      es.close();
      setLiveConnected(false);
    };
  }, [app.id, isLiveStreaming]);

  const filteredLogs = useMemo(() => {
    return liveLogs.filter((log) => {
      if (logMethodFilter !== 'ALL' && log.method.toUpperCase() !== logMethodFilter) return false;
      const code = log.statusCode ?? 200;
      if (logStatusFilter === '2xx' && (code < 200 || code >= 300)) return false;
      if (logStatusFilter === '4xx' && (code < 400 || code >= 500)) return false;
      if (logStatusFilter === '5xx' && code < 500) return false;
      if (logSearchTerm.trim()) {
        const q = logSearchTerm.toLowerCase().trim();
        const path = (log.path || '').toLowerCase();
        const url = (log.url || '').toLowerCase();
        const id = (log.id || '').toLowerCase();
        const ip = (log.clientIp || '').toLowerCase();
        const country = String(log.cf?.country || '').toLowerCase();
        const colo = String(log.cf?.colo || '').toLowerCase();
        const codeStr = String(code);
        return (
          path.includes(q) ||
          url.includes(q) ||
          id.includes(q) ||
          ip.includes(q) ||
          codeStr.includes(q) ||
          country.includes(q) ||
          colo.includes(q)
        );
      }
      return true;
    });
  }, [liveLogs, logMethodFilter, logStatusFilter, logSearchTerm]);

  return (
    <div data-testid="live-tail-view" className="space-y-4">
      {/* Live Tail Toolbar */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 bg-zinc-900/60 p-3 rounded-xl border border-zinc-800">
        <div className="flex items-center gap-3">
          <div className="flex items-center gap-2">
            <span
              className={`w-2.5 h-2.5 rounded-full ${
                liveConnected ? 'bg-emerald-500 animate-pulse' : 'bg-zinc-600'
              }`}
            />
            <span
              data-testid="tail-status-indicator"
              className={`text-xs font-semibold font-mono ${
                liveConnected ? 'text-emerald-400' : 'text-zinc-500'
              }`}
            >
              {liveConnected ? 'Connected (Live SSE)' : 'Connecting...'}
            </span>
          </div>
        </div>

        <div className="flex items-center gap-2">
          <button
            type="button"
            data-testid="tail-pause-resume-btn"
            onClick={() => setIsLiveStreaming(!isLiveStreaming)}
            className="px-3 py-1.5 rounded-lg bg-zinc-800 hover:bg-zinc-700 text-zinc-300 text-xs font-semibold transition flex items-center gap-1.5"
          >
            {isLiveStreaming ? (
              <Pause className="w-3.5 h-3.5 text-amber-400" />
            ) : (
              <Play className="w-3.5 h-3.5 text-emerald-400" />
            )}
            <span>{isLiveStreaming ? 'Pause Stream' : 'Resume Stream'}</span>
          </button>
          <button
            type="button"
            data-testid="tail-clear-btn"
            onClick={() => setLiveLogs([])}
            className="px-3 py-1.5 rounded-lg bg-zinc-800 hover:bg-zinc-700 text-zinc-400 hover:text-zinc-200 text-xs font-semibold transition flex items-center gap-1.5"
            title="Clear live tail logs"
          >
            <Trash2 className="w-3.5 h-3.5" />
            <span>Clear</span>
          </button>
        </div>
      </div>

      {/* Filter and Search Bar */}
      <div className="flex flex-wrap items-center justify-between gap-3 p-3 bg-zinc-900/40 rounded-xl border border-zinc-800 text-xs">
        <div className="flex flex-wrap items-center gap-2">
          {/* Free Text Search */}
          <div className="relative min-w-[220px]">
            <Search className="w-3.5 h-3.5 absolute left-2.5 top-1/2 -translate-y-1/2 text-zinc-500" />
            <input
              type="text"
              placeholder="Filter by path, IP, Ray ID..."
              value={logSearchTerm}
              onChange={(e) => setLogSearchTerm(e.target.value)}
              className="w-full pl-8 pr-7 py-1.5 rounded-lg bg-zinc-950 border border-zinc-800 text-xs text-zinc-200 placeholder-zinc-500 focus:outline-none focus:border-emerald-500 font-mono"
            />
            {logSearchTerm && (
              <button
                type="button"
                onClick={() => setLogSearchTerm('')}
                className="absolute right-2 top-1/2 -translate-y-1/2 text-zinc-400 hover:text-zinc-200"
              >
                <X className="w-3.5 h-3.5" />
              </button>
            )}
          </div>

          {/* Method Filter Pills */}
          <div className="flex items-center gap-1 bg-black/60 p-1 rounded-lg border border-zinc-800 text-[11px]">
            {(['ALL', 'GET', 'POST', 'PUT', 'DELETE'] as const).map((m) => (
              <button
                key={m}
                type="button"
                onClick={() => setLogMethodFilter(m)}
                className={`px-2 py-0.5 rounded font-semibold transition ${
                  logMethodFilter === m
                    ? 'bg-zinc-800 text-zinc-100 shadow-sm'
                    : 'text-zinc-400 hover:text-zinc-200'
                }`}
              >
                {m}
              </button>
            ))}
          </div>

          {/* Status Code Filter Pills */}
          <div className="flex items-center gap-1 bg-black/60 p-1 rounded-lg border border-zinc-800 text-[11px]">
            {(['ALL', '2xx', '4xx', '5xx'] as const).map((st) => (
              <button
                key={st}
                type="button"
                onClick={() => setLogStatusFilter(st)}
                className={`px-2 py-0.5 rounded font-semibold transition ${
                  logStatusFilter === st
                    ? st === '2xx'
                      ? 'bg-emerald-950 text-emerald-400 border border-emerald-800'
                      : st === '4xx'
                      ? 'bg-amber-950 text-amber-400 border border-amber-800'
                      : st === '5xx'
                      ? 'bg-rose-950 text-rose-400 border border-rose-800'
                      : 'bg-zinc-800 text-zinc-100'
                    : 'text-zinc-400 hover:text-zinc-200'
                }`}
              >
                {st}
              </button>
            ))}
          </div>
        </div>

        <div className="flex items-center gap-2 text-[11px] text-zinc-400">
          <span>
            Showing {filteredLogs.length} of {liveLogs.length} events
          </span>
          {(logSearchTerm || logMethodFilter !== 'ALL' || logStatusFilter !== 'ALL') && (
            <button
              type="button"
              onClick={() => {
                setLogSearchTerm('');
                setLogMethodFilter('ALL');
                setLogStatusFilter('ALL');
              }}
              className="text-xs text-sky-400 hover:underline ml-1"
            >
              Reset filters
            </button>
          )}
        </div>
      </div>

      {/* Table Container */}
      <div className="bg-zinc-900/60 border border-zinc-800 rounded-xl overflow-hidden">
        {/* Table Header */}
        <div className="grid grid-cols-12 gap-2 px-4 py-2.5 border-b border-zinc-800 text-zinc-500 text-[11px] font-semibold uppercase tracking-wider bg-zinc-950/60">
          <span className="col-span-2">TIMESTAMP</span>
          <span className="col-span-1">METHOD</span>
          <span className="col-span-3">PATH / URL</span>
          <span className="col-span-1 text-center">STATUS</span>
          <span className="col-span-1 text-right">LATENCY</span>
          <span className="col-span-2 text-center">EDGE & CLIENT IP</span>
          <span className="col-span-2 text-right">RAY ID</span>
        </div>

        {/* Log Events List */}
        <div className="divide-y divide-zinc-800/60 max-h-[600px] overflow-y-auto">
          {filteredLogs.length === 0 ? (
            <div className="py-16 text-center space-y-3 text-zinc-500 font-sans p-6">
              <Radio className="w-8 h-8 mx-auto text-emerald-500 animate-pulse" />
              <p className="text-sm font-semibold text-zinc-300">
                {liveLogs.length === 0 ? 'Live Request Tail Active' : 'No requests matched your filter'}
              </p>
              <p className="text-xs text-zinc-500 max-w-md mx-auto">
                {liveLogs.length === 0
                  ? `Waiting for incoming HTTP requests to ${testUrl}. Invocations will stream here in real time.`
                  : 'Try adjusting your search query, HTTP method, or response code filter.'}
              </p>
              {liveLogs.length === 0 && onTestApp && (
                <button
                  type="button"
                  data-testid="tail-send-test-btn"
                  onClick={() => onTestApp(app)}
                  className="mt-3 inline-flex items-center gap-1.5 px-4 py-2 rounded-lg border border-emerald-800 bg-emerald-950/60 hover:bg-emerald-950 text-emerald-400 text-xs font-semibold transition"
                >
                  <Send className="w-3.5 h-3.5" /> Send Test Request
                </button>
              )}
            </div>
          ) : (
            filteredLogs.map((entry, idx) => {
              const logId = entry.id || `req-${idx}`;
              const isExpanded = expandedLogId === logId;
              const code = entry.statusCode ?? 200;

              return (
                <div
                  key={logId}
                  data-testid={`log-row-${logId}`}
                  className={`transition ${
                    isExpanded ? 'bg-zinc-900/90' : 'hover:bg-zinc-850/40 bg-zinc-950/40'
                  }`}
                >
                  {/* Row Summary */}
                  <div
                    onClick={() => setExpandedLogId(isExpanded ? null : logId)}
                    className="grid grid-cols-12 gap-2 items-center px-4 py-2.5 cursor-pointer select-none text-[11px]"
                  >
                    <div className="col-span-2 flex items-center gap-1.5 text-zinc-400">
                      {isExpanded ? (
                        <ChevronDown className="w-3.5 h-3.5 text-emerald-400 shrink-0" />
                      ) : (
                        <ChevronRight className="w-3.5 h-3.5 text-zinc-500 shrink-0" />
                      )}
                      <span className="text-zinc-400 font-mono">
                        {new Date(entry.timestamp).toLocaleTimeString()}
                      </span>
                    </div>

                    <div className="col-span-1">
                      <span
                        className={`text-[10px] px-1.5 py-0.5 rounded font-bold uppercase ${
                          entry.method === 'GET'
                            ? 'bg-sky-950 text-sky-400 border border-sky-800'
                            : entry.method === 'POST'
                            ? 'bg-emerald-950 text-emerald-400 border border-emerald-800'
                            : entry.method === 'PUT' || entry.method === 'PATCH'
                            ? 'bg-amber-950 text-amber-400 border border-amber-800'
                            : entry.method === 'DELETE'
                            ? 'bg-rose-950 text-rose-400 border border-rose-800'
                            : 'bg-zinc-800 text-zinc-300'
                        }`}
                      >
                        {entry.method}
                      </span>
                    </div>

                    <div
                      className="col-span-3 truncate font-mono text-zinc-200"
                      title={entry.url || entry.path}
                    >
                      <span>{entry.path || '/'}</span>
                    </div>

                    <div className="col-span-1 text-center font-bold">
                      <span
                        className={`${
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

                    <div className="col-span-1 text-right text-zinc-400 font-mono">
                      {Math.round(entry.durationMs)}ms
                    </div>

                    <div
                      className="col-span-2 flex items-center justify-center gap-1 font-mono text-[10px] truncate"
                      title={entry.clientIp}
                    >
                      {Boolean((entry.cf as CfTelemetry)?.country) && (
                        <span
                          className="px-1.5 py-0.2 rounded bg-sky-950/80 text-sky-300 border border-sky-800/80 font-bold shrink-0"
                          title={`Country: ${String((entry.cf as CfTelemetry)?.country)}, PoP: ${String(
                            (entry.cf as CfTelemetry)?.colo || 'SFO'
                          )}`}
                        >
                          {String((entry.cf as CfTelemetry)?.country)} ·{' '}
                          {String((entry.cf as CfTelemetry)?.colo || 'SFO')}
                        </span>
                      )}
                      <span className="text-zinc-500 truncate">{entry.clientIp || '127.0.0.1'}</span>
                    </div>

                    <div
                      className="col-span-2 text-right text-zinc-500 font-mono text-[10px] truncate"
                      title={entry.id}
                    >
                      {entry.id ? entry.id.replace('ray_', '') : 'N/A'}
                    </div>
                  </div>

                  {/* Expanded Cloudflare-Style Log Inspector Drawer */}
                  {isExpanded && (
                    <LogInspectorDrawer
                      entry={entry}
                      logId={logId}
                      activeTab={activeLogDetailTab}
                      onTabChange={setActiveLogDetailTab}
                      copiedSection={copiedLogSection}
                      onCopySection={handleCopyLogSection}
                      formatJson={formatJsonPayload}
                    />
                  )}
                </div>
              );
            })
          )}
        </div>
      </div>
    </div>
  );
}

interface LogInspectorDrawerProps {
  entry: RequestLogEvent;
  logId: string;
  activeTab: 'headers' | 'payload' | 'logs' | 'cf' | 'json';
  onTabChange: (tab: 'headers' | 'payload' | 'logs' | 'cf' | 'json') => void;
  copiedSection: string | null;
  onCopySection: (text: string, sectionKey: string) => void;
  formatJson: (raw: unknown) => string;
}

function LogInspectorDrawer({
  entry,
  logId,
  activeTab,
  onTabChange,
  copiedSection,
  onCopySection,
  formatJson,
}: LogInspectorDrawerProps) {
  const reqHeaders = (entry.requestHeaders || {}) as Record<string, string>;
  const respHeaders = (entry.responseHeaders || {}) as Record<string, string>;
  const reqHeaderCount = Object.keys(reqHeaders).length;
  const respHeaderCount = Object.keys(respHeaders).length;
  const totalLogsCount = entry.logs ? entry.logs.length : 0;

  return (
    <div
      data-testid="log-inspector-drawer"
      className="p-4 border-t border-zinc-800 bg-black/80 space-y-4 font-sans text-xs"
    >
      {/* Top Info Bar */}
      <div className="flex flex-wrap items-center justify-between gap-3 p-3 bg-zinc-900/70 rounded-lg border border-zinc-800">
        <div className="flex flex-wrap items-center gap-3">
          <div className="flex items-center gap-1.5">
            <span className="text-[11px] text-zinc-500 font-semibold uppercase">Ray ID:</span>
            <code className="text-zinc-200 font-mono text-[11px] bg-zinc-950 px-2 py-0.5 rounded border border-zinc-800">
              {entry.id || 'N/A'}
            </code>
            {entry.id && (
              <button
                type="button"
                onClick={() => onCopySection(entry.id, `ray-${logId}`)}
                className="text-zinc-400 hover:text-zinc-200 transition"
                title="Copy Ray ID"
              >
                {copiedSection === `ray-${logId}` ? (
                  <Check className="w-3.5 h-3.5 text-emerald-400" />
                ) : (
                  <Copy className="w-3.5 h-3.5" />
                )}
              </button>
            )}
          </div>

          <div className="flex items-center gap-1.5 max-w-md truncate">
            <span className="text-[11px] text-zinc-500 font-semibold uppercase">URL:</span>
            <code
              className="text-emerald-400 font-mono text-[11px] truncate"
              title={entry.url || entry.path}
            >
              {entry.url || entry.path}
            </code>
            <button
              type="button"
              onClick={() => onCopySection(entry.url || entry.path, `url-${logId}`)}
              className="text-zinc-400 hover:text-zinc-200 transition"
              title="Copy URL"
            >
              {copiedSection === `url-${logId}` ? (
                <Check className="w-3.5 h-3.5 text-emerald-400" />
              ) : (
                <Copy className="w-3.5 h-3.5" />
              )}
            </button>
          </div>

          <span
            className={`px-2 py-0.5 rounded-full font-bold text-[10px] uppercase border ${
              (entry.statusCode ?? 200) < 400
                ? 'bg-emerald-950 text-emerald-400 border-emerald-800'
                : (entry.statusCode ?? 200) < 500
                ? 'bg-amber-950 text-amber-400 border-amber-800'
                : 'bg-rose-950 text-rose-400 border-rose-800'
            }`}
          >
            {entry.outcome || ((entry.statusCode ?? 200) < 400 ? 'Success' : 'Error')}
          </span>
        </div>

        <button
          type="button"
          onClick={() => onCopySection(JSON.stringify(entry, null, 2), `full-${logId}`)}
          className="text-xs text-zinc-400 hover:text-zinc-200 flex items-center gap-1.5 px-2.5 py-1 rounded bg-zinc-800 hover:bg-zinc-700 transition"
        >
          {copiedSection === `full-${logId}` ? (
            <>
              <Check className="w-3.5 h-3.5 text-emerald-400" />
              <span className="text-emerald-400">Copied Full Event</span>
            </>
          ) : (
            <>
              <Copy className="w-3.5 h-3.5" />
              <span>Copy JSON</span>
            </>
          )}
        </button>
      </div>

      {/* Inspector Sub-Tabs */}
      <div className="flex items-center gap-2 border-b border-zinc-800 pb-2">
        <button
          type="button"
          onClick={() => onTabChange('headers')}
          className={`px-3 py-1.5 rounded-md font-semibold text-xs transition flex items-center gap-1.5 ${
            activeTab === 'headers'
              ? 'bg-zinc-800 text-zinc-100 border border-zinc-700'
              : 'text-zinc-400 hover:text-zinc-200'
          }`}
        >
          <ListFilter className="w-3.5 h-3.5" />
          <span>Headers</span>
          <span className="text-[10px] bg-zinc-950 px-1.5 py-0.2 rounded border border-zinc-800 text-zinc-400">
            {reqHeaderCount + respHeaderCount}
          </span>
        </button>

        <button
          type="button"
          onClick={() => onTabChange('payload')}
          className={`px-3 py-1.5 rounded-md font-semibold text-xs transition flex items-center gap-1.5 ${
            activeTab === 'payload'
              ? 'bg-zinc-800 text-zinc-100 border border-zinc-700'
              : 'text-zinc-400 hover:text-zinc-200'
          }`}
        >
          <FileText className="w-3.5 h-3.5" />
          <span>Payload</span>
          {(entry.requestBody || entry.responseBody) && (
            <span className="w-2 h-2 rounded-full bg-emerald-500" />
          )}
        </button>

        <button
          type="button"
          onClick={() => onTabChange('logs')}
          className={`px-3 py-1.5 rounded-md font-semibold text-xs transition flex items-center gap-1.5 ${
            activeTab === 'logs'
              ? 'bg-zinc-800 text-zinc-100 border border-zinc-700'
              : 'text-zinc-400 hover:text-zinc-200'
          }`}
        >
          <Terminal className="w-3.5 h-3.5" />
          <span>Console Logs</span>
          {totalLogsCount > 0 && (
            <span className="text-[10px] bg-sky-950 text-sky-400 px-1.5 py-0.2 rounded border border-sky-800">
              {totalLogsCount}
            </span>
          )}
        </button>

        <button
          type="button"
          onClick={() => onTabChange('cf')}
          className={`px-3 py-1.5 rounded-md font-semibold text-xs transition flex items-center gap-1.5 ${
            activeTab === 'cf'
              ? 'bg-zinc-800 text-zinc-100 border border-zinc-700'
              : 'text-zinc-400 hover:text-zinc-200'
          }`}
        >
          <Globe className="w-3.5 h-3.5 text-sky-400" />
          <span>Edge / request.cf</span>
          {Boolean((entry.cf as CfTelemetry)?.country) && (
            <span className="text-[10px] bg-sky-950 text-sky-300 px-1.5 py-0.2 rounded border border-sky-800 font-mono">
              {String((entry.cf as CfTelemetry)?.country)} · {String((entry.cf as CfTelemetry)?.colo || 'SFO')}
            </span>
          )}
        </button>

        <button
          type="button"
          onClick={() => onTabChange('json')}
          className={`px-3 py-1.5 rounded-md font-semibold text-xs transition flex items-center gap-1.5 ${
            activeTab === 'json'
              ? 'bg-zinc-800 text-zinc-100 border border-zinc-700'
              : 'text-zinc-400 hover:text-zinc-200'
          }`}
        >
          <Braces className="w-3.5 h-3.5" />
          <span>Raw JSON</span>
        </button>
      </div>

      {/* Tab Content: Headers */}
      {activeTab === 'headers' && (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {/* Request Headers */}
          <div className="bg-zinc-950 rounded-lg border border-zinc-800/80 p-3.5 space-y-3">
            <div className="flex items-center justify-between border-b border-zinc-900 pb-2">
              <div className="flex items-center gap-2">
                <span className="text-xs font-bold text-zinc-200">Request Headers</span>
                <span className="text-[10px] bg-zinc-900 px-1.5 py-0.5 rounded text-zinc-400 font-mono">
                  {reqHeaderCount}
                </span>
              </div>
              {reqHeaderCount > 0 && (
                <button
                  type="button"
                  onClick={() => onCopySection(JSON.stringify(reqHeaders, null, 2), `reqh-${logId}`)}
                  className="text-[11px] text-zinc-400 hover:text-zinc-200 flex items-center gap-1 transition"
                >
                  {copiedSection === `reqh-${logId}` ? (
                    <span className="text-emerald-400 flex items-center gap-1">
                      <Check className="w-3 h-3" /> Copied
                    </span>
                  ) : (
                    <>
                      <Copy className="w-3 h-3" /> Copy
                    </>
                  )}
                </button>
              )}
            </div>

            {reqHeaderCount === 0 ? (
              <div className="text-xs text-zinc-500 italic py-2">No request headers recorded</div>
            ) : (
              <div className="space-y-1.5 max-h-56 overflow-y-auto font-mono text-[11px]">
                {Object.entries(reqHeaders).map(([k, v]) => (
                  <div key={k} className="flex items-start justify-between gap-2 p-1.5 rounded hover:bg-zinc-900/60">
                    <span className="text-emerald-400 shrink-0 font-medium">{k}:</span>
                    <span className="text-zinc-300 text-right break-all">{v}</span>
                  </div>
                ))}
              </div>
            )}
          </div>

          {/* Response Headers */}
          <div className="bg-zinc-950 rounded-lg border border-zinc-800/80 p-3.5 space-y-3">
            <div className="flex items-center justify-between border-b border-zinc-900 pb-2">
              <div className="flex items-center gap-2">
                <span className="text-xs font-bold text-zinc-200">Response Headers</span>
                <span className="text-[10px] bg-zinc-900 px-1.5 py-0.5 rounded text-zinc-400 font-mono">
                  {respHeaderCount}
                </span>
              </div>
              {respHeaderCount > 0 && (
                <button
                  type="button"
                  onClick={() => onCopySection(JSON.stringify(respHeaders, null, 2), `resph-${logId}`)}
                  className="text-[11px] text-zinc-400 hover:text-zinc-200 flex items-center gap-1 transition"
                >
                  {copiedSection === `resph-${logId}` ? (
                    <span className="text-emerald-400 flex items-center gap-1">
                      <Check className="w-3 h-3" /> Copied
                    </span>
                  ) : (
                    <>
                      <Copy className="w-3 h-3" /> Copy
                    </>
                  )}
                </button>
              )}
            </div>

            {respHeaderCount === 0 ? (
              <div className="text-xs text-zinc-500 italic py-2">No response headers recorded</div>
            ) : (
              <div className="space-y-1.5 max-h-56 overflow-y-auto font-mono text-[11px]">
                {Object.entries(respHeaders).map(([k, v]) => (
                  <div key={k} className="flex items-start justify-between gap-2 p-1.5 rounded hover:bg-zinc-900/60">
                    <span className="text-sky-400 shrink-0 font-medium">{k}:</span>
                    <span className="text-zinc-300 text-right break-all">{v}</span>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>
      )}

      {/* Tab Content: Payload */}
      {activeTab === 'payload' && (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {/* Request Body */}
          <div className="bg-zinc-950 rounded-lg border border-zinc-800/80 p-3.5 space-y-2">
            <div className="flex items-center justify-between border-b border-zinc-900 pb-2">
              <span className="text-xs font-bold text-zinc-200">Request Body</span>
              {entry.requestBody && (
                <button
                  type="button"
                  onClick={() => onCopySection(formatJson(entry.requestBody), `reqb-${logId}`)}
                  className="text-[11px] text-zinc-400 hover:text-zinc-200 flex items-center gap-1 transition"
                >
                  {copiedSection === `reqb-${logId}` ? (
                    <span className="text-emerald-400 flex items-center gap-1">
                      <Check className="w-3 h-3" /> Copied
                    </span>
                  ) : (
                    <>
                      <Copy className="w-3 h-3" /> Copy
                    </>
                  )}
                </button>
              )}
            </div>
            {!entry.requestBody ? (
              <div className="text-xs text-zinc-500 italic py-4">No request payload captured</div>
            ) : (
              <pre className="p-2.5 rounded bg-zinc-900/80 border border-zinc-800 font-mono text-[11px] text-zinc-300 max-h-60 overflow-y-auto whitespace-pre-wrap">
                {formatJson(entry.requestBody)}
              </pre>
            )}
          </div>

          {/* Response Body */}
          <div className="bg-zinc-950 rounded-lg border border-zinc-800/80 p-3.5 space-y-2">
            <div className="flex items-center justify-between border-b border-zinc-900 pb-2">
              <span className="text-xs font-bold text-zinc-200">Response Body</span>
              {entry.responseBody && (
                <button
                  type="button"
                  onClick={() => onCopySection(formatJson(entry.responseBody), `respb-${logId}`)}
                  className="text-[11px] text-zinc-400 hover:text-zinc-200 flex items-center gap-1 transition"
                >
                  {copiedSection === `respb-${logId}` ? (
                    <span className="text-emerald-400 flex items-center gap-1">
                      <Check className="w-3 h-3" /> Copied
                    </span>
                  ) : (
                    <>
                      <Copy className="w-3 h-3" /> Copy
                    </>
                  )}
                </button>
              )}
            </div>
            {!entry.responseBody ? (
              <div className="text-xs text-zinc-500 italic py-4">No response payload captured</div>
            ) : (
              <pre className="p-2.5 rounded bg-zinc-900/80 border border-zinc-800 font-mono text-[11px] text-zinc-300 max-h-60 overflow-y-auto whitespace-pre-wrap">
                {formatJson(entry.responseBody)}
              </pre>
            )}
          </div>
        </div>
      )}

      {/* Tab Content: Console Logs */}
      {activeTab === 'logs' && (
        <div className="bg-zinc-950 rounded-lg border border-zinc-800/80 p-3.5 space-y-3">
          <div className="flex items-center justify-between border-b border-zinc-900 pb-2">
            <div className="flex items-center gap-2">
              <Terminal className="w-4 h-4 text-emerald-400" />
              <span className="text-xs font-bold text-zinc-200">Worker Isolate Console Logs</span>
              <span className="text-[10px] bg-zinc-900 px-1.5 py-0.5 rounded text-zinc-400 font-mono">
                {totalLogsCount} logs
              </span>
            </div>
          </div>

          {!entry.logs || entry.logs.length === 0 ? (
            <div className="text-xs text-zinc-500 italic py-4">
              No console logs emitted by the isolate during this execution.
            </div>
          ) : (
            <div className="space-y-1.5 max-h-60 overflow-y-auto font-mono text-[11px]">
              {entry.logs.map((log, lIdx) => (
                <div
                  key={lIdx}
                  className="flex items-start gap-2 p-2 rounded bg-zinc-900/50 border border-zinc-850"
                >
                  <span className="text-zinc-500 text-[10px] shrink-0">
                    {new Date(log.timestamp).toLocaleTimeString()}
                  </span>
                  {(() => {
                    const otel = getOtelSeverity(log.level);
                    return (
                      <span
                        data-testid="otel-severity-badge"
                        className={`text-[9px] px-1.5 py-0.5 rounded font-mono font-bold uppercase shrink-0 ${otel.className}`}
                      >
                        {otel.label}
                      </span>
                    );
                  })()}
                  <span className="text-zinc-200 break-all">{log.message}</span>
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      {/* Tab Content: Edge context (request.cf) */}
      {activeTab === 'cf' && (
        <div className="bg-zinc-950 rounded-lg border border-zinc-800/80 p-4 space-y-4">
          <div className="flex items-center justify-between border-b border-zinc-900 pb-2">
            <div className="flex items-center gap-2">
              <Globe className="w-4 h-4 text-sky-400" />
              <span className="text-xs font-bold text-zinc-200">
                Cloudflare Edge Context (<code className="text-sky-300">request.cf</code>)
              </span>
            </div>
            {entry.cf && (
              <button
                type="button"
                onClick={() => onCopySection(JSON.stringify(entry.cf, null, 2), `cf-${logId}`)}
                className="text-[11px] text-zinc-400 hover:text-zinc-200 flex items-center gap-1 transition"
              >
                {copiedSection === `cf-${logId}` ? (
                  <span className="text-emerald-400 flex items-center gap-1">
                    <Check className="w-3 h-3" /> Copied
                  </span>
                ) : (
                  <>
                    <Copy className="w-3 h-3" /> Copy
                  </>
                )}
              </button>
            )}
          </div>

          {!entry.cf ? (
            <div className="text-xs text-zinc-500 italic py-4">
              No edge metadata attached to this request.
            </div>
          ) : (() => {
            const cf = entry.cf as CfTelemetry;
            return (
              <div className="grid grid-cols-2 sm:grid-cols-4 gap-3 text-xs">
                <div className="p-2.5 rounded-lg bg-zinc-900/60 border border-zinc-800/60">
                  <span className="text-[10px] text-zinc-500 uppercase font-semibold">Country</span>
                  <p className="font-bold text-zinc-200 mt-0.5">{String(cf.country || 'N/A')}</p>
                </div>
                <div className="p-2.5 rounded-lg bg-zinc-900/60 border border-zinc-800/60">
                  <span className="text-[10px] text-zinc-500 uppercase font-semibold">City</span>
                  <p className="font-bold text-zinc-200 mt-0.5">{String(cf.city || 'N/A')}</p>
                </div>
                <div className="p-2.5 rounded-lg bg-zinc-900/60 border border-zinc-800/60">
                  <span className="text-[10px] text-zinc-500 uppercase font-semibold">Colo / PoP</span>
                  <p className="font-bold text-sky-400 font-mono mt-0.5">{String(cf.colo || 'SFO')}</p>
                </div>
                <div className="p-2.5 rounded-lg bg-zinc-900/60 border border-zinc-800/60">
                  <span className="text-[10px] text-zinc-500 uppercase font-semibold">ASN</span>
                  <p className="font-bold text-zinc-200 font-mono mt-0.5">
                    AS{String(cf.asn ?? '13335')}
                  </p>
                </div>
                <div className="p-2.5 rounded-lg bg-zinc-900/60 border border-zinc-800/60">
                  <span className="text-[10px] text-zinc-500 uppercase font-semibold">Protocol</span>
                  <p className="font-bold text-zinc-200 font-mono mt-0.5">
                    {String(cf.httpProtocol || 'HTTP/2')}
                  </p>
                </div>
                <div className="p-2.5 rounded-lg bg-zinc-900/60 border border-zinc-800/60">
                  <span className="text-[10px] text-zinc-500 uppercase font-semibold">TLS Version</span>
                  <p className="font-bold text-zinc-200 font-mono mt-0.5">
                    {String(cf.tlsVersion || 'TLSv1.3')}
                  </p>
                </div>
                <div className="p-2.5 rounded-lg bg-zinc-900/60 border border-zinc-800/60">
                  <span className="text-[10px] text-zinc-500 uppercase font-semibold">Client IP</span>
                  <p className="font-bold text-zinc-200 font-mono mt-0.5">
                    {entry.clientIp || '127.0.0.1'}
                  </p>
                </div>
                <div className="p-2.5 rounded-lg bg-zinc-900/60 border border-zinc-800/60">
                  <span className="text-[10px] text-zinc-500 uppercase font-semibold">Ray ID</span>
                  <p className="font-bold text-zinc-200 font-mono text-[11px] truncate mt-0.5">
                    {entry.id || 'N/A'}
                  </p>
                </div>
              </div>
            );
          })()}
        </div>
      )}

      {/* Tab Content: Raw JSON */}
      {activeTab === 'json' && (
        <div className="space-y-2">
          <pre className="p-3 rounded-lg bg-zinc-950 border border-zinc-800 font-mono text-[11px] text-emerald-400 overflow-x-auto max-h-72">
            {JSON.stringify(entry, null, 2)}
          </pre>
        </div>
      )}
    </div>
  );
}
