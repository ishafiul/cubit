import React, { useState, useEffect } from 'react';
import { GitMerge, Play, Plus, Trash2, RefreshCw, AlertCircle, ArrowRight, Activity } from 'lucide-react';

interface Workflow {
  id: string;
  name: string;
  targetAppId: string;
  steps: string[];
  createdAt: string;
}

interface WorkflowRun {
  id: string;
  workflowId: string;
  status: string;
  currentStep: string;
  logs: string[];
  startedAt: string;
  finishedAt?: string;
}

export function WorkflowsView({ initialSelectedId }: { initialSelectedId?: string } = {}) {
  const [workflows, setWorkflows] = useState<Workflow[]>([]);
  const [selectedWf, setSelectedWf] = useState<Workflow | null>(null);
  const [runs, setRuns] = useState<WorkflowRun[]>([]);
  const [loading, setLoading] = useState(false);
  const [triggering, setTriggering] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Modals
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [name, setName] = useState('');
  const [targetAppId, setTargetAppId] = useState('');
  const [stepsInput, setStepsInput] = useState('validate_input, process_payment, notify_user');

  const fetchWorkflows = async () => {
    setLoading(true);
    try {
      const res = await fetch('/api/v1/workflows');
      if (!res.ok) throw new Error('Failed to fetch workflows');
      const data = await res.json();
      const list = Array.isArray(data) ? data : [];
      setWorkflows(list);
      if (initialSelectedId) {
        const found = list.find(w => w.id === initialSelectedId || w.name === initialSelectedId);
        if (found) {
          setSelectedWf(found);
          return;
        }
      }
      if (list.length > 0 && !selectedWf) {
        setSelectedWf(list[0]);
      }
    } catch (err: any) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (initialSelectedId && workflows.length > 0) {
      const found = workflows.find(w => w.id === initialSelectedId || w.name === initialSelectedId);
      if (found) {
        setSelectedWf(found);
      }
    }
  }, [initialSelectedId, workflows]);

  const fetchRuns = async (wfId: string) => {
    try {
      const res = await fetch(`/api/v1/workflows/${wfId}/runs`);
      if (!res.ok) throw new Error('Failed to fetch workflow runs');
      const data = await res.json();
      setRuns(Array.isArray(data) ? data : []);
    } catch (err: any) {
      console.error(err);
    }
  };

  useEffect(() => {
    fetchWorkflows();
  }, []);

  useEffect(() => {
    if (selectedWf) {
      fetchRuns(selectedWf.id);
    } else {
      setRuns([]);
    }
  }, [selectedWf]);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim()) return;
    const steps = stepsInput.split(',').map(s => s.trim()).filter(Boolean);
    try {
      const res = await fetch('/api/v1/workflows', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          name: name.trim(),
          target_app_id: targetAppId.trim(),
          steps,
        }),
      });
      if (!res.ok) throw new Error('Failed to create workflow');
      const created = await res.json();
      setShowCreateModal(false);
      setName('');
      setTargetAppId('');
      setStepsInput('validate_input, process_payment, notify_user');
      await fetchWorkflows();
      setSelectedWf(created);
    } catch (err: any) {
      alert(err.message);
    }
  };

  const handleDelete = async (id: string) => {
    if (!confirm('Are you sure you want to delete this workflow?')) return;
    try {
      const res = await fetch(`/api/v1/workflows/${id}`, { method: 'DELETE' });
      if (!res.ok) throw new Error('Failed to delete workflow');
      if (selectedWf?.id === id) {
        setSelectedWf(null);
      }
      await fetchWorkflows();
    } catch (err: any) {
      alert(err.message);
    }
  };

  const handleTrigger = async (id: string) => {
    setTriggering(true);
    try {
      const res = await fetch(`/api/v1/workflows/${id}/trigger`, { method: 'POST' });
      if (!res.ok) throw new Error('Failed to trigger workflow');
      await fetchRuns(id);
    } catch (err: any) {
      alert(err.message);
    } finally {
      setTriggering(false);
    }
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <div className="flex items-center gap-3">
            <h2 className="text-2xl font-bold tracking-tight">Workflows</h2>
            <span className="text-xs bg-emerald-950 text-emerald-400 border border-emerald-800 px-2.5 py-0.5 rounded-full font-mono font-medium">
              celld service: Yes
            </span>
          </div>
          <p className="text-sm text-zinc-400 mt-1">
            Durable multi-step state machines with automatic step checkpoints, retries, and sleep intervals.
          </p>
        </div>
        <button
          onClick={() => setShowCreateModal(true)}
          className="flex items-center gap-2 px-4 py-2 rounded-lg bg-emerald-500 hover:bg-emerald-600 text-zinc-950 text-sm font-semibold transition"
        >
          <Plus className="w-4 h-4" />
          Create Workflow
        </button>
      </div>

      {error && (
        <div className="p-4 rounded-xl border border-red-900/50 bg-red-950/20 text-red-400 flex items-center gap-3">
          <AlertCircle className="w-5 h-5 flex-shrink-0" />
          <span className="text-sm">{error}</span>
        </div>
      )}

      {/* Main Grid: Left = Workflows List, Right = Steps & Execution Logs */}
      <div className="grid grid-cols-12 gap-6 min-h-[520px]">
        {/* Left Column: Workflows List */}
        <div className="col-span-5 border border-zinc-800/80 bg-zinc-900/30 rounded-2xl p-4 flex flex-col justify-between">
          <div className="space-y-3">
            <div className="flex items-center justify-between pb-2 border-b border-zinc-800">
              <span className="text-xs font-semibold text-zinc-400 uppercase tracking-wider">Workflows</span>
              <button onClick={fetchWorkflows} className="text-zinc-500 hover:text-zinc-300 transition">
                <RefreshCw className={`w-3.5 h-3.5 ${loading ? 'animate-spin' : ''}`} />
              </button>
            </div>

            {workflows.length === 0 ? (
              <div className="text-center py-10 text-zinc-500 text-sm">
                No durable workflows defined yet.
              </div>
            ) : (
              <div className="space-y-2 overflow-y-auto max-h-[480px]">
                {workflows.map(wf => (
                  <div
                    key={wf.id}
                    onClick={() => setSelectedWf(wf)}
                    className={`p-3.5 rounded-xl cursor-pointer border transition space-y-2 ${
                      selectedWf?.id === wf.id
                        ? 'bg-zinc-800/90 border-emerald-500/50 text-white'
                        : 'border-transparent hover:bg-zinc-800/40 text-zinc-300'
                    }`}
                  >
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-2">
                        <GitMerge className={`w-4 h-4 ${selectedWf?.id === wf.id ? 'text-emerald-400' : 'text-zinc-500'}`} />
                        <span className="font-semibold text-sm">{wf.name}</span>
                      </div>
                      <span className="text-xs font-mono bg-zinc-800 px-2 py-0.5 rounded text-emerald-400">
                        {wf.steps.length} steps
                      </span>
                    </div>

                    <div className="flex items-center justify-between text-xs text-zinc-400">
                      <span className="truncate max-w-[180px]">Worker: {wf.targetAppId || 'General'}</span>
                      <button
                        onClick={(e) => {
                          e.stopPropagation();
                          handleDelete(wf.id);
                        }}
                        className="p-1 hover:text-red-400 text-zinc-500 transition"
                        title="Delete Workflow"
                      >
                        <Trash2 className="w-3.5 h-3.5" />
                      </button>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>

        {/* Right Column: Workflow Steps & Execution Runs */}
        <div className="col-span-7 border border-zinc-800/80 bg-zinc-900/30 rounded-2xl p-6 flex flex-col justify-between space-y-5">
          {selectedWf ? (
            <div className="space-y-5">
              <div className="flex items-center justify-between border-b border-zinc-800 pb-3">
                <div>
                  <div className="flex items-center gap-2">
                    <h3 className="text-lg font-bold">{selectedWf.name}</h3>
                    <span className="text-xs font-mono text-zinc-500">ID: {selectedWf.id}</span>
                  </div>
                  <p className="text-xs text-zinc-400 mt-0.5">Durable workflow target: <span className="font-mono text-zinc-300">{selectedWf.targetAppId || 'default'}</span></p>
                </div>
                <button
                  onClick={() => handleTrigger(selectedWf.id)}
                  disabled={triggering}
                  className="flex items-center gap-2 px-3.5 py-1.5 rounded-lg bg-emerald-500 hover:bg-emerald-600 disabled:opacity-50 text-zinc-950 text-xs font-semibold transition"
                >
                  <Play className={`w-3.5 h-3.5 fill-current ${triggering ? 'animate-spin' : ''}`} />
                  {triggering ? 'Triggering...' : 'Trigger Workflow'}
                </button>
              </div>

              {/* Step Sequence Visualization */}
              <div className="space-y-2">
                <span className="text-xs font-semibold text-zinc-400 uppercase tracking-wider block">
                  Workflow Step Sequence
                </span>
                <div className="flex items-center flex-wrap gap-2 p-3.5 rounded-xl bg-zinc-950/60 border border-zinc-800">
                  {selectedWf.steps.map((step, idx) => (
                    <React.Fragment key={step}>
                      <div className="flex items-center gap-2 px-3 py-1 rounded-lg bg-zinc-900 border border-zinc-800 text-xs font-mono text-emerald-400">
                        <span className="w-4 h-4 rounded-full bg-emerald-950 border border-emerald-800 flex items-center justify-center text-[10px] text-emerald-400 font-bold">
                          {idx + 1}
                        </span>
                        <span>{step}</span>
                      </div>
                      {idx < selectedWf.steps.length - 1 && (
                        <ArrowRight className="w-3.5 h-3.5 text-zinc-600 flex-shrink-0" />
                      )}
                    </React.Fragment>
                  ))}
                </div>
              </div>

              {/* Runs List & Step Logs */}
              <div className="space-y-2">
                <div className="flex items-center justify-between text-xs text-zinc-400 font-semibold uppercase tracking-wider">
                  <div className="flex items-center gap-2">
                    <Activity className="w-4 h-4 text-emerald-400" />
                    <span>Workflow Runs ({runs.length})</span>
                  </div>
                  <button onClick={() => fetchRuns(selectedWf.id)} className="text-zinc-500 hover:text-zinc-300">
                    <RefreshCw className="w-3 h-3" />
                  </button>
                </div>

                <div className="space-y-3 max-h-[260px] overflow-y-auto">
                  {runs.length === 0 ? (
                    <div className="p-8 text-center text-zinc-500 text-xs rounded-xl border border-zinc-800 bg-zinc-950/40">
                      No executions yet. Click "Trigger Workflow" to run this state machine.
                    </div>
                  ) : (
                    runs.map(run => (
                      <div key={run.id} className="p-3.5 rounded-xl border border-zinc-800 bg-zinc-950/60 space-y-2">
                        <div className="flex items-center justify-between text-xs">
                          <div className="flex items-center gap-2">
                            <span className="font-mono text-zinc-400">{run.id.slice(0, 8)}...</span>
                            <span className="text-[10px] px-2 py-0.5 rounded-full font-bold uppercase bg-emerald-950 text-emerald-400 border border-emerald-800">
                              {run.status}
                            </span>
                          </div>
                          <span className="text-zinc-500 text-[11px]">
                            {new Date(run.startedAt).toLocaleTimeString()}
                          </span>
                        </div>

                        {run.logs && run.logs.length > 0 && (
                          <div className="p-2.5 rounded-lg bg-black/40 border border-zinc-900 font-mono text-[11px] text-zinc-300 space-y-1">
                            {run.logs.map((log, lIdx) => (
                              <div key={lIdx} className="text-zinc-400">{log}</div>
                            ))}
                          </div>
                        )}
                      </div>
                    ))
                  )}
                </div>
              </div>
            </div>
          ) : (
            <div className="flex flex-col items-center justify-center h-full text-center py-20 text-zinc-500">
              <GitMerge className="w-12 h-12 stroke-1 text-zinc-700 mb-3" />
              <p className="text-sm">Select a workflow to inspect steps and execution states</p>
            </div>
          )}
        </div>
      </div>

      {/* Create Workflow Modal */}
      {showCreateModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm p-4">
          <div className="w-full max-w-md bg-zinc-900 border border-zinc-800 rounded-2xl p-6 shadow-2xl space-y-4">
            <h3 className="text-lg font-bold">Create Durable Workflow</h3>
            <form onSubmit={handleCreate} className="space-y-4">
              <div>
                <label className="text-xs font-semibold text-zinc-400 uppercase tracking-wider block mb-1">
                  Workflow Name
                </label>
                <input
                  type="text"
                  required
                  placeholder="e.g. order-processing, user-onboarding"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  className="w-full px-3 py-2 rounded-xl bg-zinc-950 border border-zinc-800 text-sm focus:outline-none focus:border-emerald-500"
                />
              </div>
              <div>
                <label className="text-xs font-semibold text-zinc-400 uppercase tracking-wider block mb-1">
                  Target Worker ID / Subdomain (optional)
                </label>
                <input
                  type="text"
                  placeholder="e.g. ecommerce-worker"
                  value={targetAppId}
                  onChange={(e) => setTargetAppId(e.target.value)}
                  className="w-full px-3 py-2 rounded-xl bg-zinc-950 border border-zinc-800 text-xs focus:outline-none focus:border-emerald-500"
                />
              </div>
              <div>
                <label className="text-xs font-semibold text-zinc-400 uppercase tracking-wider block mb-1">
                  Steps (comma-separated list)
                </label>
                <input
                  type="text"
                  required
                  placeholder="validate_input, process_order, send_receipt"
                  value={stepsInput}
                  onChange={(e) => setStepsInput(e.target.value)}
                  className="w-full px-3 py-2 rounded-xl bg-zinc-950 border border-zinc-800 font-mono text-xs focus:outline-none focus:border-emerald-500"
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
                  Create Workflow
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
