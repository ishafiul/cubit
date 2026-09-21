import { useState } from 'react';
import { Code, Play, AlertCircle, Clock, Sparkles, Copy, Check } from 'lucide-react';
import { CodeEditor } from '../CodeEditor';

const TEMPLATES: Record<string, { name: string; code: string; method: string; path: string }> = {
  hello: {
    name: 'Hello World',
    method: 'GET',
    path: '/',
    code: `export default {
  async fetch(request, env, ctx) {
    return new Response("Hello World from Dynamic Celld 0.5.1 Worker!", {
      headers: { "content-type": "text/plain; charset=utf-8" },
    });
  },
};`,
  },
  json: {
    name: 'JSON API Endpoint',
    method: 'GET',
    path: '/api/v1/status',
    code: `export default {
  async fetch(request, env, ctx) {
    const url = new URL(request.url);
    return Response.json({
      status: "online",
      celld_runtime: "v0.5.1",
      timestamp: new Date().toISOString(),
      path: url.pathname,
    });
  },
};`,
  },
  echo: {
    name: 'Header & Body Echo',
    method: 'POST',
    path: '/echo',
    code: `export default {
  async fetch(request, env, ctx) {
    const body = await request.text();
    const headers = Object.fromEntries(request.headers.entries());
    return Response.json({
      method: request.method,
      receivedHeaders: headers,
      body: body || null,
    });
  },
};`,
  },
  router: {
    name: 'REST Sub-Router',
    method: 'GET',
    path: '/items',
    code: `export default {
  async fetch(request, env, ctx) {
    const url = new URL(request.url);
    if (url.pathname === '/items') {
      return Response.json([{ id: 1, item: "D1 Database" }, { id: 2, item: "KV Store" }]);
    }
    return new Response("Not Found", { status: 404 });
  },
};`,
  },
};

export function DynamicWorkersView() {
  const [activeTemplate, setActiveTemplate] = useState('hello');
  const [code, setCode] = useState(TEMPLATES.hello.code);
  const [method, setMethod] = useState(TEMPLATES.hello.method);
  const [path, setPath] = useState(TEMPLATES.hello.path);
  const [reqHeaders, setReqHeaders] = useState('{\n  "X-Custom-Header": "Cubit-Dynamic"\n}');
  const [reqBody, setReqBody] = useState('');

  // Execution state
  const [executing, setExecuting] = useState(false);
  const [respStatus, setRespStatus] = useState<number | null>(null);
  const [respHeaders, setRespHeaders] = useState<Record<string, string> | null>(null);
  const [respBody, setRespBody] = useState<string | null>(null);
  const [durationMs, setDurationMs] = useState<number | null>(null);
  const [execError, setExecError] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);

  const handleSelectTemplate = (key: string) => {
    setActiveTemplate(key);
    const tmpl = TEMPLATES[key];
    setCode(tmpl.code);
    setMethod(tmpl.method);
    setPath(tmpl.path);
    setRespStatus(null);
    setRespBody(null);
    setExecError(null);
  };

  const handleExecute = async () => {
    setExecuting(true);
    setExecError(null);
    const start = performance.now();

    let parsedHeaders: Record<string, string> = {};
    try {
      if (reqHeaders.trim()) {
        parsedHeaders = JSON.parse(reqHeaders);
      }
    } catch {
      setExecError("Request headers must be valid JSON");
      setExecuting(false);
      return;
    }

    try {
      const res = await fetch('/api/v1/dynamic-workers/eval', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          code,
          method,
          path,
          headers: parsedHeaders,
          body: reqBody,
        }),
      });

      const data = await res.json();
      const elapsed = performance.now() - start;
      setDurationMs(elapsed);

      if (!res.ok) {
        throw new Error(data.message || 'Worker evaluation failed');
      }

      setRespStatus(data.status);
      setRespHeaders(data.headers);
      setRespBody(data.body);
    } catch (err: any) {
      setExecError(err.message);
      setRespStatus(null);
      setRespBody(null);
    } finally {
      setExecuting(false);
    }
  };

  const copyResponse = () => {
    if (respBody) {
      navigator.clipboard.writeText(respBody);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    }
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <div className="flex items-center gap-3">
            <h2 className="text-2xl font-bold tracking-tight">Dynamic Workers</h2>
            <span className="text-xs bg-emerald-950 text-emerald-400 border border-emerald-800 px-2.5 py-0.5 rounded-full font-mono font-medium">
              celld service: Yes
            </span>
          </div>
          <p className="text-sm text-zinc-400 mt-1">
            Execute dynamic, programmatic worker code snippets with zero cold starts via Celld isolate workers.
          </p>
        </div>
        <div className="flex items-center gap-2">
          {Object.entries(TEMPLATES).map(([key, tmpl]) => (
            <button
              key={key}
              onClick={() => handleSelectTemplate(key)}
              className={`px-3 py-1.5 rounded-lg text-xs font-medium transition ${
                activeTemplate === key
                  ? 'bg-emerald-500/20 text-emerald-400 border border-emerald-500/40'
                  : 'bg-zinc-900 border border-zinc-800 text-zinc-400 hover:text-zinc-200'
              }`}
            >
              {tmpl.name}
            </button>
          ))}
        </div>
      </div>

      {/* Main 2-Column Split: Left = Code Editor, Right = Request & Live Response Simulator */}
      <div className="grid grid-cols-12 gap-6 min-h-[600px]">
        {/* Left Column: Worker Code Editor */}
        <div className="col-span-7 border border-zinc-800/80 bg-zinc-900/30 rounded-2xl p-5 flex flex-col justify-between">
          <div className="space-y-3">
            <div className="flex items-center justify-between pb-2 border-b border-zinc-800">
              <div className="flex items-center gap-2 text-xs font-semibold text-zinc-400 uppercase tracking-wider">
                <Code className="w-4 h-4 text-emerald-400" />
                <span>Worker Script (ES Modules)</span>
              </div>
              <span className="text-[11px] font-mono text-zinc-500">celld 0.5.1 Runtime</span>
            </div>

            <div className="rounded-xl overflow-hidden border border-zinc-800">
              <CodeEditor
                value={code}
                onChange={setCode}
                minHeight="450px"
              />
            </div>
          </div>

          <div className="pt-4 flex items-center justify-between border-t border-zinc-800">
            <span className="text-xs text-zinc-500">
              Supports <span className="text-zinc-400 font-mono">export default &#123; fetch &#125;</span> standard Workers API
            </span>
            <button
              onClick={handleExecute}
              disabled={executing}
              className="flex items-center gap-2 px-5 py-2 rounded-lg bg-emerald-500 hover:bg-emerald-600 disabled:opacity-50 text-zinc-950 text-sm font-semibold transition shadow-lg shadow-emerald-950"
            >
              <Play className={`w-4 h-4 fill-current ${executing ? 'animate-spin' : ''}`} />
              {executing ? 'Evaluating...' : 'Run Dynamic Worker'}
            </button>
          </div>
        </div>

        {/* Right Column: Request Parameters & Response Inspector */}
        <div className="col-span-5 border border-zinc-800/80 bg-zinc-900/30 rounded-2xl p-5 flex flex-col justify-between space-y-4">
          {/* Request Configuration */}
          <div className="space-y-3">
            <div className="flex items-center justify-between pb-2 border-b border-zinc-800">
              <span className="text-xs font-semibold text-zinc-400 uppercase tracking-wider">HTTP Request Test</span>
              <span className="text-xs font-mono text-emerald-400">Direct Socket</span>
            </div>

            <div className="flex items-center gap-2">
              <select
                value={method}
                onChange={(e) => setMethod(e.target.value)}
                className="px-3 py-1.5 rounded-lg bg-zinc-950 border border-zinc-800 text-xs font-bold text-emerald-400 focus:outline-none focus:border-emerald-500"
              >
                <option value="GET">GET</option>
                <option value="POST">POST</option>
                <option value="PUT">PUT</option>
                <option value="DELETE">DELETE</option>
              </select>

              <input
                type="text"
                value={path}
                onChange={(e) => setPath(e.target.value)}
                placeholder="/"
                className="flex-1 px-3 py-1.5 rounded-lg bg-zinc-950 border border-zinc-800 text-xs font-mono text-zinc-200 focus:outline-none focus:border-emerald-500"
              />
            </div>

            <div>
              <label className="text-[11px] font-semibold text-zinc-400 block mb-1">Request Headers (JSON)</label>
              <textarea
                rows={2}
                value={reqHeaders}
                onChange={(e) => setReqHeaders(e.target.value)}
                placeholder='{"Content-Type": "application/json"}'
                className="w-full p-2 rounded-lg bg-zinc-950 border border-zinc-800 text-xs font-mono text-zinc-300 focus:outline-none"
              />
            </div>

            {method !== 'GET' && (
              <div>
                <label className="text-[11px] font-semibold text-zinc-400 block mb-1">Request Body</label>
                <textarea
                  rows={2}
                  value={reqBody}
                  onChange={(e) => setReqBody(e.target.value)}
                  placeholder="Request body payload..."
                  className="w-full p-2 rounded-lg bg-zinc-950 border border-zinc-800 text-xs font-mono text-zinc-300 focus:outline-none"
                />
              </div>
            )}
          </div>

          {/* Response Inspector */}
          <div className="flex-1 flex flex-col justify-between border-t border-zinc-800 pt-3 space-y-3">
            <div className="flex items-center justify-between">
              <span className="text-xs font-semibold text-zinc-400 uppercase tracking-wider">Live Response</span>
              {durationMs !== null && (
                <div className="flex items-center gap-1.5 text-xs text-zinc-400 font-mono">
                  <Clock className="w-3.5 h-3.5 text-zinc-500" />
                  <span>{durationMs.toFixed(1)} ms</span>
                </div>
              )}
            </div>

            {execError && (
              <div className="p-3 rounded-xl border border-red-900/50 bg-red-950/20 text-red-400 text-xs flex items-center gap-2 font-mono">
                <AlertCircle className="w-4 h-4 flex-shrink-0" />
                <span>{execError}</span>
              </div>
            )}

            {respStatus !== null ? (
              <div className="flex-1 flex flex-col space-y-2">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <span
                      className={`text-xs px-2.5 py-0.5 rounded-full font-bold font-mono ${
                        respStatus >= 200 && respStatus < 300
                          ? 'bg-emerald-950 text-emerald-400 border border-emerald-800'
                          : 'bg-amber-950 text-amber-400 border border-amber-800'
                      }`}
                    >
                      HTTP {respStatus}
                    </span>
                    <span className="text-xs text-zinc-400 font-medium">OK</span>
                  </div>

                  <button
                    onClick={copyResponse}
                    className="flex items-center gap-1 text-[11px] text-zinc-400 hover:text-white bg-zinc-800 px-2 py-0.5 rounded transition"
                  >
                    {copied ? <Check className="w-3 h-3 text-emerald-400" /> : <Copy className="w-3 h-3" />}
                    {copied ? 'Copied' : 'Copy'}
                  </button>
                </div>

                {respHeaders && Object.keys(respHeaders).length > 0 && (
                  <div className="text-[10px] font-mono text-zinc-500 truncate">
                    Headers: {Object.entries(respHeaders).map(([k, v]) => `${k}: ${v}`).join(', ')}
                  </div>
                )}

                <div className="flex-1 min-h-[140px] p-3 rounded-xl bg-zinc-950 border border-zinc-800 overflow-y-auto">
                  <pre className="text-xs font-mono text-zinc-200 whitespace-pre-wrap break-all leading-relaxed">
                    {respBody || '<empty body>'}
                  </pre>
                </div>
              </div>
            ) : (
              <div className="flex-1 flex flex-col items-center justify-center text-center p-8 text-zinc-500">
                <Sparkles className="w-8 h-8 text-zinc-700 mb-2 stroke-1" />
                <p className="text-xs">Click "Run Dynamic Worker" to evaluate in a fresh Celld isolate</p>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
