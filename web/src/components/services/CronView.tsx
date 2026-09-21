import React, { useState, useEffect } from 'react';
import { Clock, Play, Plus, Trash2, RefreshCw, AlertCircle, History } from 'lucide-react';

interface CronTrigger {
  id: string;
  name: string;
  cron: string;
  targetAppId: string;
  active: boolean;
  createdAt: string;
}

interface CronRun {
  id: string;
  triggerId: string;
  status: string;
  statusCode: number;
  durationMs: number;
  executedAt: string;
}

export function CronView() {
  const [triggers, setTriggers] = useState<CronTrigger[]>([]);
  const [selectedTrigger, setSelectedTrigger] = useState<CronTrigger | null>(null);
  const [runs, setRuns] = useState<CronRun[]>([]);
  const [loading, setLoading] = useState(false);
  const [runningId, setRunningId] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  // Modal
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [name, setName] = useState('');
  const [cron, setCron] = useState('*/15 * * * *');
  const [targetAppId, setTargetAppId] = useState('');

  const fetchTriggers = async () => {
    setLoading(true);
    try {
      const res = await fetch('/api/v1/cron');
      if (!res.ok) throw new Error('Failed to fetch cron triggers');
      const data = await res.json();
      const list = Array.isArray(data) ? data : [];
      setTriggers(list);
      if (list.length > 0 && !selectedTrigger) {
        setSelectedTrigger(list[0]);
      }
    } catch (err: any) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  const fetchRuns = async (triggerId: string) => {
    try {
      const res = await fetch(`/api/v1/cron/${triggerId}/runs`);
      if (!res.ok) throw new Error('Failed to fetch runs');
      const data = await res.json();
      setRuns(Array.isArray(data) ? data : []);
    } catch (err: any) {
      console.error(err);
    }
  };

  useEffect(() => {
    fetchTriggers();
  }, []);

  useEffect(() => {
    if (selectedTrigger) {
      fetchRuns(selectedTrigger.id);
    } else {
      setRuns([]);
    }
  }, [selectedTrigger]);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim() || !cron.trim()) return;
    try {
      const res = await fetch('/api/v1/cron', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          name: name.trim(),
          cron: cron.trim(),
          target_app_id: targetAppId.trim(),
        }),
      });
      if (!res.ok) throw new Error('Failed to create cron trigger');
      const created = await res.json();
      setShowCreateModal(false);
      setName('');
      setCron('*/15 * * * *');
      setTargetAppId('');
      await fetchTriggers();
      setSelectedTrigger(created);
    } catch (err: any) {
      alert(err.message);
    }
  };

  const handleDelete = async (id: string) => {
    if (!confirm('Are you sure you want to delete this cron trigger?')) return;
    try {
      const res = await fetch(`/api/v1/cron/${id}`, { method: 'DELETE' });
      if (!res.ok) throw new Error('Failed to delete cron trigger');
      if (selectedTrigger?.id === id) {
        setSelectedTrigger(null);
      }
      await fetchTriggers();
    } catch (err: any) {
      alert(err.message);
    }
  };

  const handleRunNow = async (id: string) => {
    setRunningId(id);
    try {
      const res = await fetch(`/api/v1/cron/${id}/run`, { method: 'POST' });
      if (!res.ok) throw new Error('Failed to trigger execution');
      await fetchRuns(id);
    } catch (err: any) {
      alert(err.message);
    } finally {
      setRunningId(null);
    }
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <div className="flex items-center gap-3">
            <h2 className="text-2xl font-bold tracking-tight">Cron Triggers</h2>
            <span className="text-xs bg-emerald-950 text-emerald-400 border border-emerald-800 px-2.5 py-0.5 rounded-full font-mono font-medium">
              celld service: Yes
            </span>
          </div>
          <p className="text-sm text-zinc-400 mt-1">
            Automated scheduled task runners that dispatch synthetic invocation events to workers.
          </p>
        </div>
        <button
          onClick={() => setShowCreateModal(true)}
          className="flex items-center gap-2 px-4 py-2 rounded-lg bg-emerald-500 hover:bg-emerald-600 text-zinc-950 text-sm font-semibold transition"
        >
          <Plus className="w-4 h-4" />
          Add Cron Trigger
        </button>
      </div>

      {error && (
        <div className="p-4 rounded-xl border border-red-900/50 bg-red-950/20 text-red-400 flex items-center gap-3">
          <AlertCircle className="w-5 h-5 flex-shrink-0" />
          <span className="text-sm">{error}</span>
        </div>
      )}

      {/* Main Grid: Left = Triggers List, Right = Run History */}
      <div className="grid grid-cols-12 gap-6 min-h-[500px]">
        {/* Left Column: Triggers List */}
        <div className="col-span-5 border border-zinc-800/80 bg-zinc-900/30 rounded-2xl p-4 flex flex-col justify-between">
          <div className="space-y-3">
            <div className="flex items-center justify-between pb-2 border-b border-zinc-800">
              <span className="text-xs font-semibold text-zinc-400 uppercase tracking-wider">Scheduled Triggers</span>
              <button onClick={fetchTriggers} className="text-zinc-500 hover:text-zinc-300 transition">
                <RefreshCw className={`w-3.5 h-3.5 ${loading ? 'animate-spin' : ''}`} />
              </button>
            </div>

            {triggers.length === 0 ? (
              <div className="text-center py-10 text-zinc-500 text-sm">
                No scheduled cron triggers configured.
              </div>
            ) : (
              <div className="space-y-2 overflow-y-auto max-h-[480px]">
                {triggers.map(t => (
                  <div
                    key={t.id}
                    onClick={() => setSelectedTrigger(t)}
                    className={`p-3.5 rounded-xl cursor-pointer border transition space-y-2 ${
                      selectedTrigger?.id === t.id
                        ? 'bg-zinc-800/90 border-emerald-500/50 text-white'
                        : 'border-transparent hover:bg-zinc-800/40 text-zinc-300'
                    }`}
                  >
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-2">
                        <Clock className={`w-4 h-4 ${selectedTrigger?.id === t.id ? 'text-emerald-400' : 'text-zinc-500'}`} />
                        <span className="font-semibold text-sm">{t.name}</span>
                      </div>
                      <span className="text-xs font-mono bg-zinc-800 px-2 py-0.5 rounded text-emerald-400">
                        {t.cron}
                      </span>
                    </div>

                    <div className="flex items-center justify-between text-xs text-zinc-400">
                      <span className="truncate max-w-[180px]">Target: {t.targetAppId || 'All Workers'}</span>
                      <div className="flex items-center gap-2">
                        <button
                          onClick={(e) => {
                            e.stopPropagation();
                            handleRunNow(t.id);
                          }}
                          disabled={runningId === t.id}
                          className="flex items-center gap-1 text-[11px] font-semibold px-2 py-1 rounded bg-emerald-500/20 hover:bg-emerald-500/30 text-emerald-400 transition"
                        >
                          <Play className={`w-3 h-3 fill-current ${runningId === t.id ? 'animate-spin' : ''}`} />
                          {runningId === t.id ? 'Running...' : 'Run Now'}
                        </button>
                        <button
                          onClick={(e) => {
                            e.stopPropagation();
                            handleDelete(t.id);
                          }}
                          className="p-1 hover:text-red-400 text-zinc-500 transition"
                          title="Delete Trigger"
                        >
                          <Trash2 className="w-3.5 h-3.5" />
                        </button>
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>

        {/* Right Column: Execution History */}
        <div className="col-span-7 border border-zinc-800/80 bg-zinc-900/30 rounded-2xl p-6 flex flex-col justify-between">
          {selectedTrigger ? (
            <div className="space-y-4">
              <div className="flex items-center justify-between border-b border-zinc-800 pb-3">
                <div>
                  <div className="flex items-center gap-2">
                    <h3 className="text-lg font-bold">{selectedTrigger.name}</h3>
                    <span className="text-xs font-mono bg-zinc-800 px-2 py-0.5 rounded text-emerald-400">
                      {selectedTrigger.cron}
                    </span>
                  </div>
                  <p className="text-xs text-zinc-400 mt-0.5">Dispatches synthetic invocation events to: <span className="font-mono text-zinc-300">{selectedTrigger.targetAppId || 'default'}</span></p>
                </div>
                <button
                  onClick={() => handleRunNow(selectedTrigger.id)}
                  disabled={runningId === selectedTrigger.id}
                  className="flex items-center gap-2 px-3.5 py-1.5 rounded-lg bg-emerald-500 hover:bg-emerald-600 disabled:opacity-50 text-zinc-950 text-xs font-semibold transition"
                >
                  <Play className={`w-3.5 h-3.5 fill-current ${runningId === selectedTrigger.id ? 'animate-spin' : ''}`} />
                  Trigger Now
                </button>
              </div>

              <div className="flex items-center justify-between text-xs text-zinc-400 font-semibold uppercase tracking-wider">
                <div className="flex items-center gap-2">
                  <History className="w-4 h-4 text-emerald-400" />
                  <span>Execution Runs ({runs.length})</span>
                </div>
                <button onClick={() => fetchRuns(selectedTrigger.id)} className="text-zinc-500 hover:text-zinc-300">
                  <RefreshCw className="w-3 h-3" />
                </button>
              </div>

              <div className="overflow-x-auto rounded-xl border border-zinc-800 bg-zinc-950/50">
                <table className="w-full text-left text-xs">
                  <thead className="border-b border-zinc-800 bg-zinc-900/50 text-zinc-400 font-semibold">
                    <tr>
                      <th className="p-3">Run ID</th>
                      <th className="p-3">Status</th>
                      <th className="p-3">HTTP Code</th>
                      <th className="p-3">Duration</th>
                      <th className="p-3">Timestamp</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-zinc-800/60 font-mono">
                    {runs.length === 0 ? (
                      <tr>
                        <td colSpan={5} className="p-8 text-center text-zinc-500 font-sans">
                          No execution history recorded yet. Click "Trigger Now" to test.
                        </td>
                      </tr>
                    ) : (
                      runs.map(run => (
                        <tr key={run.id} className="hover:bg-zinc-800/30 transition">
                          <td className="p-3 text-zinc-400 font-mono text-[11px]">
                            {run.id.slice(0, 8)}...
                          </td>
                          <td className="p-3">
                            <span className={`text-[10px] px-2 py-0.5 rounded-full font-bold uppercase ${
                              run.status === 'success'
                                ? 'bg-emerald-950 text-emerald-400 border border-emerald-800'
                                : 'bg-red-950 text-red-400 border border-red-800'
                            }`}>
                              {run.status}
                            </span>
                          </td>
                          <td className="p-3 text-zinc-200">
                            {run.statusCode}
                          </td>
                          <td className="p-3 text-zinc-400">
                            {run.durationMs.toFixed(1)} ms
                          </td>
                          <td className="p-3 text-zinc-500 text-[11px] font-sans">
                            {new Date(run.executedAt).toLocaleTimeString()}
                          </td>
                        </tr>
                      ))
                    )}
                  </tbody>
                </table>
              </div>
            </div>
          ) : (
            <div className="flex flex-col items-center justify-center h-full text-center py-20 text-zinc-500">
              <Clock className="w-12 h-12 stroke-1 text-zinc-700 mb-3" />
              <p className="text-sm">Select a cron trigger to view execution logs</p>
            </div>
          )}
        </div>
      </div>

      {/* Create Modal */}
      {showCreateModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm p-4">
          <div className="w-full max-w-md bg-zinc-900 border border-zinc-800 rounded-2xl p-6 shadow-2xl space-y-4">
            <h3 className="text-lg font-bold">Add Cron Trigger</h3>
            <form onSubmit={handleCreate} className="space-y-4">
              <div>
                <label className="text-xs font-semibold text-zinc-400 uppercase tracking-wider block mb-1">
                  Trigger Name
                </label>
                <input
                  type="text"
                  required
                  placeholder="e.g. database-cleanup, sync-cache"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  className="w-full px-3 py-2 rounded-xl bg-zinc-950 border border-zinc-800 text-sm focus:outline-none focus:border-emerald-500"
                />
              </div>
              <div>
                <label className="text-xs font-semibold text-zinc-400 uppercase tracking-wider block mb-1">
                  Cron Schedule Expression
                </label>
                <input
                  type="text"
                  required
                  placeholder="*/15 * * * * or @daily"
                  value={cron}
                  onChange={(e) => setCron(e.target.value)}
                  className="w-full px-3 py-2 rounded-xl bg-zinc-950 border border-zinc-800 font-mono text-xs focus:outline-none focus:border-emerald-500"
                />
              </div>
              <div>
                <label className="text-xs font-semibold text-zinc-400 uppercase tracking-wider block mb-1">
                  Target Worker ID / Subdomain (optional)
                </label>
                <input
                  type="text"
                  placeholder="Leave blank to broadcast to active workers"
                  value={targetAppId}
                  onChange={(e) => setTargetAppId(e.target.value)}
                  className="w-full px-3 py-2 rounded-xl bg-zinc-950 border border-zinc-800 text-xs focus:outline-none focus:border-emerald-500"
                />
              </div>
              <div className="flex justify-end gap-3 pt-2">
                <button
                  type="button"
                  onClick={() => setShowCreateModal(false)}
                  className="px-4 py-2 rounded-lg text-sm text-zinc-400 hover:text-white transition"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  className="px-4 py-2 rounded-lg bg-emerald-500 hover:bg-emerald-600 text-zinc-950 text-sm font-semibold transition"
                >
                  Create Trigger
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
