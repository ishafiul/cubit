import React, { useState } from 'react';
import { Send, RefreshCw, Copy, Check, X } from 'lucide-react';
import type { Application } from '../../api/model';
import { useTestApplication } from '../../api/generated/applications/applications';

export interface TestAppModalProps {
  app: Application;
  onClose: () => void;
}

export function TestAppModal({ app, onClose }: TestAppModalProps) {
  const [method, setMethod] = useState<'GET' | 'POST' | 'PUT' | 'DELETE'>('GET');
  const [path, setPath] = useState('/');
  const [body, setBody] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [testResult, setTestResult] = useState<{
    statusCode: number;
    headers?: Record<string, string>;
    body: string;
    durationMs?: number;
  } | null>(null);
  const [copiedCurl, setCopiedCurl] = useState(false);

  const testAppMutation = useTestApplication();
  const subdomain = app.subdomain || app.name;
  const testUrl = app.testUrl || `http://${subdomain}.localhost:8000`;
  const curlCommand = `curl -i -X ${method} -H "Host: ${subdomain}.localhost:8000" http://localhost:8000${path}${
    method !== 'GET' && body ? ` -d '${body}'` : ''
  }`;

  const handleRunTest = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsLoading(true);
    const start = performance.now();
    try {
      const res = await testAppMutation.mutateAsync({
        id: app.id,
        data: {
          method,
          path: path.startsWith('/') ? path : `/${path}`,
          body: method !== 'GET' && body ? body : undefined,
        },
      });
      const duration = Math.round(performance.now() - start);
      setTestResult({
        statusCode: res.statusCode,
        headers: res.headers as Record<string, string> | undefined,
        body: res.body,
        durationMs: duration,
      });
    } catch (err: any) {
      setTestResult({
        statusCode: 500,
        body: err?.message || 'Failed to invoke test runner',
        durationMs: Math.round(performance.now() - start),
      });
    } finally {
      setIsLoading(false);
    }
  };

  const copyCurl = () => {
    navigator.clipboard.writeText(curlCommand);
    setCopiedCurl(true);
    setTimeout(() => setCopiedCurl(false), 2000);
  };

  return (
    <div className="fixed inset-0 bg-black/60 backdrop-blur-sm flex items-center justify-center p-4 z-50">
      <div className="bg-zinc-900 border border-zinc-800 rounded-2xl max-w-2xl w-full p-6 space-y-5 shadow-2xl overflow-hidden flex flex-col max-h-[90vh]">
        {/* Header */}
        <div className="flex items-center justify-between border-b border-zinc-800/80 pb-4">
          <div>
            <div className="flex items-center gap-2.5">
              <h3 className="text-lg font-bold">Test Application</h3>
              <span className="text-xs px-2.5 py-0.5 rounded-full font-mono bg-emerald-950/80 text-emerald-400 border border-emerald-800/80">
                {app.name}
              </span>
            </div>
            <p className="text-xs text-zinc-400 mt-0.5">
              Live HTTP testing via subdomain route{' '}
              <code className="text-emerald-400 font-mono">{testUrl}</code>
            </p>
          </div>
          <button
            type="button"
            onClick={onClose}
            className="text-zinc-500 hover:text-zinc-300 transition"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* Form */}
        <form onSubmit={handleRunTest} className="space-y-4">
          <div className="flex items-center gap-2">
            <select
              value={method}
              onChange={e => setMethod(e.target.value as any)}
              className="bg-zinc-950 border border-zinc-800 rounded-lg px-3 py-2 text-xs font-mono font-bold text-zinc-200 outline-none focus:border-emerald-500"
            >
              <option value="GET">GET</option>
              <option value="POST">POST</option>
              <option value="PUT">PUT</option>
              <option value="DELETE">DELETE</option>
            </select>
            <div className="relative flex-1">
              <span className="absolute left-3 top-2.5 text-xs text-zinc-500 font-mono">/</span>
              <input
                type="text"
                value={path.startsWith('/') ? path.slice(1) : path}
                onChange={e => setPath('/' + e.target.value)}
                placeholder="api/users or hello"
                className="w-full bg-zinc-950 border border-zinc-800 rounded-lg py-2 pl-6 pr-3 text-xs font-mono text-zinc-200 outline-none focus:border-emerald-500"
              />
            </div>
            <button
              type="submit"
              disabled={isLoading}
              className="flex items-center gap-2 px-5 py-2 rounded-lg bg-emerald-500 hover:bg-emerald-600 disabled:opacity-50 text-zinc-950 text-xs font-bold transition shadow-sm"
            >
              {isLoading ? <RefreshCw className="w-3.5 h-3.5 animate-spin" /> : <Send className="w-3.5 h-3.5" />}
              Send
            </button>
          </div>

          {method !== 'GET' && (
            <div className="space-y-1.5">
              <label className="text-xs text-zinc-400 font-medium">Request Body (JSON / text):</label>
              <textarea
                value={body}
                onChange={e => setBody(e.target.value)}
                rows={3}
                placeholder='{"message": "hello"}'
                className="w-full bg-zinc-950 border border-zinc-800 rounded-lg p-2.5 text-xs font-mono text-zinc-200 outline-none focus:border-emerald-500 resize-none"
              />
            </div>
          )}
        </form>

        {/* cURL snippet */}
        <div className="bg-zinc-950 border border-zinc-800/80 rounded-xl p-3 flex items-center justify-between gap-3 text-xs font-mono text-zinc-400">
          <span className="truncate">{curlCommand}</span>
          <button
            type="button"
            onClick={copyCurl}
            className="flex items-center gap-1 text-zinc-400 hover:text-zinc-200 shrink-0 transition"
          >
            {copiedCurl ? <Check className="w-3.5 h-3.5 text-emerald-400" /> : <Copy className="w-3.5 h-3.5" />}
            <span>{copiedCurl ? 'Copied' : 'Copy cURL'}</span>
          </button>
        </div>

        {/* Response Box */}
        {testResult && (
          <div className="space-y-2 flex-1 min-h-0 flex flex-col">
            <div className="flex items-center justify-between text-xs">
              <div className="flex items-center gap-2">
                <span className="font-semibold text-zinc-400">Response:</span>
                <span className={`px-2 py-0.5 rounded font-mono font-bold ${
                  testResult.statusCode >= 200 && testResult.statusCode < 300
                    ? 'bg-emerald-950 text-emerald-400 border border-emerald-800'
                    : testResult.statusCode >= 400
                    ? 'bg-rose-950 text-rose-400 border border-rose-800'
                    : 'bg-zinc-800 text-zinc-300'
                }`}>
                  HTTP {testResult.statusCode}
                </span>
                {testResult.durationMs !== undefined && (
                  <span className="text-zinc-500 font-mono">{testResult.durationMs}ms</span>
                )}
              </div>
            </div>

            <div className="flex-1 bg-black/80 rounded-xl p-3.5 font-mono text-xs text-zinc-300 overflow-y-auto border border-zinc-800 shadow-inner max-h-48 whitespace-pre-wrap">
              {testResult.body || '<Empty Response Body>'}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
