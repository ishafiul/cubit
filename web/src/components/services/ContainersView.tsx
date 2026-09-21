import React, { useState, useEffect } from 'react';
import { Cpu, Plus, Trash2, RefreshCw, AlertCircle, Box } from 'lucide-react';

interface ContainerWorkload {
  id: string;
  name: string;
  image: string;
  port: number;
  status: string;
  envVars: Record<string, string>;
  createdAt: string;
}

export function ContainersView() {
  const [containers, setContainers] = useState<ContainerWorkload[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Modal
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [name, setName] = useState('');
  const [image, setImage] = useState('redis:alpine');
  const [port, setPort] = useState(6379);

  const fetchContainers = async () => {
    setLoading(true);
    try {
      const res = await fetch('/api/v1/containers');
      if (!res.ok) throw new Error('Failed to fetch containers');
      const data = await res.json();
      setContainers(Array.isArray(data) ? data : []);
    } catch (err: any) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchContainers();
  }, []);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim() || !image.trim()) return;
    try {
      const res = await fetch('/api/v1/containers', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          name: name.trim(),
          image: image.trim(),
          port: Number(port) || 8080,
          env_vars: {},
        }),
      });
      if (!res.ok) throw new Error('Failed to launch container');
      setShowCreateModal(false);
      setName('');
      setImage('redis:alpine');
      setPort(6379);
      await fetchContainers();
    } catch (err: any) {
      alert(err.message);
    }
  };

  const handleDelete = async (id: string) => {
    if (!confirm('Are you sure you want to terminate this container?')) return;
    try {
      const res = await fetch(`/api/v1/containers/${id}`, { method: 'DELETE' });
      if (!res.ok) throw new Error('Failed to delete container');
      await fetchContainers();
    } catch (err: any) {
      alert(err.message);
    }
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <div className="flex items-center gap-3">
            <h2 className="text-2xl font-bold tracking-tight">Containers</h2>
            <span className="text-xs bg-amber-950 text-amber-400 border border-amber-800 px-2.5 py-0.5 rounded-full font-mono font-medium">
              celld service: Experimental
            </span>
          </div>
          <p className="text-sm text-zinc-400 mt-1">
            Run OCI and Docker micro-containers sidecar to Celld V8 worker isolates on bare-metal fleet nodes.
          </p>
        </div>
        <button
          onClick={() => setShowCreateModal(true)}
          className="flex items-center gap-2 px-4 py-2 rounded-lg bg-emerald-500 hover:bg-emerald-600 text-zinc-950 text-sm font-semibold transition"
        >
          <Plus className="w-4 h-4" />
          Deploy Container
        </button>
      </div>

      {error && (
        <div className="p-4 rounded-xl border border-red-900/50 bg-red-950/20 text-red-400 flex items-center gap-3">
          <AlertCircle className="w-5 h-5 flex-shrink-0" />
          <span className="text-sm">{error}</span>
        </div>
      )}

      {/* Containers Grid */}
      <div className="space-y-3">
        <div className="flex items-center justify-between">
          <span className="text-xs font-semibold text-zinc-400 uppercase tracking-wider">
            Active Container Workloads ({containers.length})
          </span>
          <button onClick={fetchContainers} className="text-zinc-500 hover:text-zinc-300">
            <RefreshCw className={`w-3.5 h-3.5 ${loading ? 'animate-spin' : ''}`} />
          </button>
        </div>

        {containers.length === 0 ? (
          <div className="p-12 text-center rounded-2xl border border-zinc-800/80 bg-zinc-900/20 text-zinc-500">
            <Cpu className="w-12 h-12 stroke-1 text-zinc-700 mx-auto mb-3" />
            <p className="text-sm">No container workloads deployed yet.</p>
            <p className="text-xs text-zinc-600 mt-1">Deploy Redis, Postgres, or custom OCI images.</p>
          </div>
        ) : (
          <div className="grid grid-cols-2 gap-4">
            {containers.map(ct => (
              <div key={ct.id} className="p-5 rounded-2xl border border-zinc-800/80 bg-zinc-900/30 space-y-4 hover:border-zinc-700 transition">
                <div className="flex items-start justify-between">
                  <div>
                    <div className="flex items-center gap-2">
                      <h4 className="font-semibold text-base">{ct.name}</h4>
                      <span className="text-[10px] px-2 py-0.5 rounded-full font-bold uppercase bg-emerald-950 text-emerald-400 border border-emerald-800">
                        {ct.status}
                      </span>
                    </div>
                    <p className="text-xs text-zinc-500 font-mono mt-0.5">{ct.id}</p>
                  </div>
                  <button
                    onClick={() => handleDelete(ct.id)}
                    className="p-1.5 hover:text-red-400 text-zinc-500 transition"
                    title="Terminate Container"
                  >
                    <Trash2 className="w-4 h-4" />
                  </button>
                </div>

                <div className="grid grid-cols-2 gap-2 text-xs text-zinc-400 font-mono">
                  <div className="p-2.5 rounded-xl bg-zinc-950/60 border border-zinc-800/60 flex items-center gap-2">
                    <Box className="w-3.5 h-3.5 text-zinc-500 flex-shrink-0" />
                    <span className="truncate">{ct.image}</span>
                  </div>
                  <div className="p-2.5 rounded-xl bg-zinc-950/60 border border-zinc-800/60 flex items-center gap-2">
                    <Cpu className="w-3.5 h-3.5 text-zinc-500 flex-shrink-0" />
                    <span>Port: {ct.port}</span>
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Deploy Container Modal */}
      {showCreateModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm p-4">
          <div className="w-full max-w-md bg-zinc-900 border border-zinc-800 rounded-2xl p-6 shadow-2xl space-y-4">
            <h3 className="text-lg font-bold">Deploy Container Workload</h3>
            <form onSubmit={handleCreate} className="space-y-4">
              <div>
                <label className="text-xs font-semibold text-zinc-400 uppercase tracking-wider block mb-1">
                  Container Name
                </label>
                <input
                  type="text"
                  required
                  placeholder="e.g. redis-cache, postgres-db"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  className="w-full px-3 py-2 rounded-xl bg-zinc-950 border border-zinc-800 text-sm focus:outline-none focus:border-emerald-500"
                />
              </div>
              <div>
                <label className="text-xs font-semibold text-zinc-400 uppercase tracking-wider block mb-1">
                  Docker / OCI Image
                </label>
                <input
                  type="text"
                  required
                  placeholder="e.g. redis:alpine or postgres:16"
                  value={image}
                  onChange={(e) => setImage(e.target.value)}
                  className="w-full px-3 py-2 rounded-xl bg-zinc-950 border border-zinc-800 font-mono text-xs focus:outline-none focus:border-emerald-500"
                />
              </div>
              <div>
                <label className="text-xs font-semibold text-zinc-400 uppercase tracking-wider block mb-1">
                  Internal Service Port
                </label>
                <input
                  type="number"
                  required
                  value={port}
                  onChange={(e) => setPort(parseInt(e.target.value) || 8080)}
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
                  Deploy
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
