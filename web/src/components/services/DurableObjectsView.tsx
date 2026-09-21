import React, { useState, useEffect } from 'react';
import { Box, Plus, Trash2, RefreshCw, AlertCircle, Cpu, Code2 } from 'lucide-react';

interface DOFacet {
  name: string;
  method: string;
  description: string;
}

interface DurableObjectClass {
  id: string;
  name: string;
  appId: string;
  facets: DOFacet[];
  createdAt: string;
}

interface DurableObjectInstance {
  id: string;
  classId: string;
  objectId: string;
  status: string;
  storageKeysCount: number;
  createdAt: string;
}

export function DurableObjectsView() {
  const [classes, setClasses] = useState<DurableObjectClass[]>([]);
  const [selectedClass, setSelectedClass] = useState<DurableObjectClass | null>(null);
  const [instances, setInstances] = useState<DurableObjectInstance[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Modals
  const [showCreateClassModal, setShowCreateClassModal] = useState(false);
  const [name, setName] = useState('');
  const [appId, setAppId] = useState('');
  const [facetMethod, setFacetMethod] = useState('increment');
  const [facetDesc, setFacetDesc] = useState('Increments internal counter atomically');

  const [showCreateInstModal, setShowCreateInstModal] = useState(false);
  const [objectId, setObjectId] = useState('');

  const fetchClasses = async () => {
    setLoading(true);
    try {
      const res = await fetch('/api/v1/durable-objects');
      if (!res.ok) throw new Error('Failed to fetch DO classes');
      const data = await res.json();
      const list = Array.isArray(data) ? data : [];
      setClasses(list);
      if (list.length > 0 && !selectedClass) {
        setSelectedClass(list[0]);
      }
    } catch (err: any) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  const fetchInstances = async (classId: string) => {
    try {
      const res = await fetch(`/api/v1/durable-objects/${classId}/instances`);
      if (!res.ok) throw new Error('Failed to fetch DO instances');
      const data = await res.json();
      setInstances(Array.isArray(data) ? data : []);
    } catch (err: any) {
      console.error(err);
    }
  };

  useEffect(() => {
    fetchClasses();
  }, []);

  useEffect(() => {
    if (selectedClass) {
      fetchInstances(selectedClass.id);
    } else {
      setInstances([]);
    }
  }, [selectedClass]);

  const handleCreateClass = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim()) return;
    const facets: DOFacet[] = [];
    if (facetMethod.trim()) {
      facets.push({
        name: facetMethod.trim(),
        method: facetMethod.trim(),
        description: facetDesc.trim(),
      });
    }
    try {
      const res = await fetch('/api/v1/durable-objects', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          name: name.trim(),
          app_id: appId.trim(),
          facets,
        }),
      });
      if (!res.ok) throw new Error('Failed to create DO class');
      const created = await res.json();
      setShowCreateClassModal(false);
      setName('');
      setAppId('');
      await fetchClasses();
      setSelectedClass(created);
    } catch (err: any) {
      alert(err.message);
    }
  };

  const handleDeleteClass = async (id: string) => {
    if (!confirm('Are you sure you want to delete this Durable Object class?')) return;
    try {
      const res = await fetch(`/api/v1/durable-objects/${id}`, { method: 'DELETE' });
      if (!res.ok) throw new Error('Failed to delete DO class');
      if (selectedClass?.id === id) {
        setSelectedClass(null);
      }
      await fetchClasses();
    } catch (err: any) {
      alert(err.message);
    }
  };

  const handleCreateInstance = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!selectedClass || !objectId.trim()) return;
    try {
      const res = await fetch(`/api/v1/durable-objects/${selectedClass.id}/instances`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ object_id: objectId.trim() }),
      });
      if (!res.ok) throw new Error('Failed to create DO instance');
      setShowCreateInstModal(false);
      setObjectId('');
      await fetchInstances(selectedClass.id);
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
            <h2 className="text-2xl font-bold tracking-tight">Durable Objects & Facets</h2>
            <span className="text-xs bg-emerald-950 text-emerald-400 border border-emerald-800 px-2.5 py-0.5 rounded-full font-mono font-medium">
              celld service: Yes
            </span>
          </div>
          <p className="text-sm text-zinc-400 mt-1">
            Stateful, globally unique in-memory actor instances with local transactional storage and RPC facets.
          </p>
        </div>
        <button
          onClick={() => setShowCreateClassModal(true)}
          className="flex items-center gap-2 px-4 py-2 rounded-lg bg-emerald-500 hover:bg-emerald-600 text-zinc-950 text-sm font-semibold transition"
        >
          <Plus className="w-4 h-4" />
          Define DO Class
        </button>
      </div>

      {error && (
        <div className="p-4 rounded-xl border border-red-900/50 bg-red-950/20 text-red-400 flex items-center gap-3">
          <AlertCircle className="w-5 h-5 flex-shrink-0" />
          <span className="text-sm">{error}</span>
        </div>
      )}

      {/* Main Grid: Left = DO Classes, Right = Instances & Facets */}
      <div className="grid grid-cols-12 gap-6 min-h-[500px]">
        {/* Left Column: DO Classes List */}
        <div className="col-span-5 border border-zinc-800/80 bg-zinc-900/30 rounded-2xl p-4 flex flex-col justify-between">
          <div className="space-y-3">
            <div className="flex items-center justify-between pb-2 border-b border-zinc-800">
              <span className="text-xs font-semibold text-zinc-400 uppercase tracking-wider">DO Actor Classes</span>
              <button onClick={fetchClasses} className="text-zinc-500 hover:text-zinc-300 transition">
                <RefreshCw className={`w-3.5 h-3.5 ${loading ? 'animate-spin' : ''}`} />
              </button>
            </div>

            {classes.length === 0 ? (
              <div className="text-center py-10 text-zinc-500 text-sm">
                No Durable Object classes defined yet.
              </div>
            ) : (
              <div className="space-y-2 overflow-y-auto max-h-[480px]">
                {classes.map(c => (
                  <div
                    key={c.id}
                    onClick={() => setSelectedClass(c)}
                    className={`p-3.5 rounded-xl cursor-pointer border transition space-y-2 ${
                      selectedClass?.id === c.id
                        ? 'bg-zinc-800/90 border-emerald-500/50 text-white'
                        : 'border-transparent hover:bg-zinc-800/40 text-zinc-300'
                    }`}
                  >
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-2">
                        <Box className={`w-4 h-4 ${selectedClass?.id === c.id ? 'text-emerald-400' : 'text-zinc-500'}`} />
                        <span className="font-semibold text-sm">{c.name}</span>
                      </div>
                      <span className="text-xs font-mono bg-zinc-800 px-2 py-0.5 rounded text-emerald-400">
                        {c.facets ? c.facets.length : 0} facets
                      </span>
                    </div>

                    <div className="flex items-center justify-between text-xs text-zinc-400">
                      <span className="truncate max-w-[180px]">Worker: {c.appId || 'default'}</span>
                      <button
                        onClick={(e) => {
                          e.stopPropagation();
                          handleDeleteClass(c.id);
                        }}
                        className="p-1 hover:text-red-400 text-zinc-500 transition"
                        title="Delete Class"
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

        {/* Right Column: DO Facets & Live Instances */}
        <div className="col-span-7 border border-zinc-800/80 bg-zinc-900/30 rounded-2xl p-6 flex flex-col justify-between space-y-5">
          {selectedClass ? (
            <div className="space-y-5">
              <div className="flex items-center justify-between border-b border-zinc-800 pb-3">
                <div>
                  <div className="flex items-center gap-2">
                    <h3 className="text-lg font-bold">{selectedClass.name}</h3>
                    <span className="text-xs font-mono text-zinc-500">ID: {selectedClass.id}</span>
                  </div>
                  <p className="text-xs text-zinc-400 mt-0.5">Attached worker binding: <span className="font-mono text-zinc-300">{selectedClass.appId || 'Global'}</span></p>
                </div>
                <button
                  onClick={() => setShowCreateInstModal(true)}
                  className="flex items-center gap-2 px-3.5 py-1.5 rounded-lg bg-emerald-500/10 hover:bg-emerald-500/20 text-emerald-400 border border-emerald-500/30 text-xs font-semibold transition"
                >
                  <Plus className="w-3.5 h-3.5" />
                  Instantiate Actor
                </button>
              </div>

              {/* Durable Object Facets Section */}
              <div className="space-y-2">
                <span className="text-xs font-semibold text-zinc-400 uppercase tracking-wider flex items-center gap-2">
                  <Code2 className="w-4 h-4 text-emerald-400" />
                  Exported RPC Facets
                </span>
                {selectedClass.facets && selectedClass.facets.length > 0 ? (
                  <div className="grid grid-cols-2 gap-3">
                    {selectedClass.facets.map(f => (
                      <div key={f.method} className="p-3 rounded-xl border border-zinc-800 bg-zinc-950/60 space-y-1">
                        <div className="font-mono text-xs font-semibold text-emerald-400">{f.method}()</div>
                        <p className="text-[11px] text-zinc-400">{f.description || 'RPC facet callable via stub'}</p>
                      </div>
                    ))}
                  </div>
                ) : (
                  <div className="p-3 rounded-xl border border-zinc-800 bg-zinc-950/30 text-xs text-zinc-500">
                    No explicit RPC facets declared on this DO class. Default fetch() method applies.
                  </div>
                )}
              </div>

              {/* Active Instances List */}
              <div className="space-y-2">
                <div className="flex items-center justify-between text-xs text-zinc-400 font-semibold uppercase tracking-wider">
                  <div className="flex items-center gap-2">
                    <Cpu className="w-4 h-4 text-emerald-400" />
                    <span>Active In-Memory Instances ({instances.length})</span>
                  </div>
                  <button onClick={() => fetchInstances(selectedClass.id)} className="text-zinc-500 hover:text-zinc-300">
                    <RefreshCw className="w-3 h-3" />
                  </button>
                </div>

                <div className="overflow-x-auto rounded-xl border border-zinc-800 bg-zinc-950/50">
                  <table className="w-full text-left text-xs">
                    <thead className="border-b border-zinc-800 bg-zinc-900/50 text-zinc-400 font-semibold">
                      <tr>
                        <th className="p-3">Object ID (Actor Key)</th>
                        <th className="p-3">Status</th>
                        <th className="p-3">Keys Count</th>
                        <th className="p-3">Created</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-zinc-800/60 font-mono">
                      {instances.length === 0 ? (
                        <tr>
                          <td colSpan={4} className="p-6 text-center text-zinc-500 font-sans">
                            No actor instances running. Click "Instantiate Actor" to initialize an ID.
                          </td>
                        </tr>
                      ) : (
                        instances.map(inst => (
                          <tr key={inst.id} className="hover:bg-zinc-800/30 transition">
                            <td className="p-3 font-semibold text-emerald-400">
                              {inst.objectId}
                            </td>
                            <td className="p-3">
                              <span className="text-[10px] px-2 py-0.5 rounded-full font-bold uppercase bg-emerald-950 text-emerald-400 border border-emerald-800">
                                {inst.status}
                              </span>
                            </td>
                            <td className="p-3 text-zinc-300">
                              {inst.storageKeysCount} keys
                            </td>
                            <td className="p-3 text-zinc-500 text-[11px] font-sans">
                              {new Date(inst.createdAt).toLocaleTimeString()}
                            </td>
                          </tr>
                        ))
                      )}
                    </tbody>
                  </table>
                </div>
              </div>
            </div>
          ) : (
            <div className="flex flex-col items-center justify-center h-full text-center py-20 text-zinc-500">
              <Box className="w-12 h-12 stroke-1 text-zinc-700 mb-3" />
              <p className="text-sm">Select a Durable Object class to inspect facets and actor state</p>
            </div>
          )}
        </div>
      </div>

      {/* Define Class Modal */}
      {showCreateClassModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm p-4">
          <div className="w-full max-w-md bg-zinc-900 border border-zinc-800 rounded-2xl p-6 shadow-2xl space-y-4">
            <h3 className="text-lg font-bold">Define Durable Object Class</h3>
            <form onSubmit={handleCreateClass} className="space-y-4">
              <div>
                <label className="text-xs font-semibold text-zinc-400 uppercase tracking-wider block mb-1">
                  Class Name
                </label>
                <input
                  type="text"
                  required
                  placeholder="e.g. CounterDO, ChatRoomActor"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  className="w-full px-3 py-2 rounded-xl bg-zinc-950 border border-zinc-800 text-sm focus:outline-none focus:border-emerald-500"
                />
              </div>
              <div>
                <label className="text-xs font-semibold text-zinc-400 uppercase tracking-wider block mb-1">
                  Attached Worker App ID (optional)
                </label>
                <input
                  type="text"
                  placeholder="e.g. main-api-worker"
                  value={appId}
                  onChange={(e) => setAppId(e.target.value)}
                  className="w-full px-3 py-2 rounded-xl bg-zinc-950 border border-zinc-800 text-xs focus:outline-none focus:border-emerald-500"
                />
              </div>
              <div className="p-3 rounded-xl bg-zinc-950 border border-zinc-800 space-y-3">
                <span className="text-xs font-semibold text-zinc-300">Initial RPC Facet</span>
                <div>
                  <input
                    type="text"
                    placeholder="Facet Method Name (e.g. increment)"
                    value={facetMethod}
                    onChange={(e) => setFacetMethod(e.target.value)}
                    className="w-full px-2.5 py-1.5 rounded-lg bg-zinc-900 border border-zinc-800 font-mono text-xs focus:outline-none focus:border-emerald-500"
                  />
                </div>
                <div>
                  <input
                    type="text"
                    placeholder="Facet Description"
                    value={facetDesc}
                    onChange={(e) => setFacetDesc(e.target.value)}
                    className="w-full px-2.5 py-1.5 rounded-lg bg-zinc-900 border border-zinc-800 text-xs focus:outline-none focus:border-emerald-500"
                  />
                </div>
              </div>
              <div className="flex justify-end gap-3 pt-2">
                <button
                  type="button"
                  onClick={() => setShowCreateClassModal(false)}
                  className="px-4 py-2 rounded-lg text-sm text-zinc-400 hover:text-white transition"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  className="px-4 py-2 rounded-lg bg-emerald-500 hover:bg-emerald-600 text-zinc-950 text-sm font-semibold transition"
                >
                  Save Class
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Create Instance Modal */}
      {showCreateInstModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm p-4">
          <div className="w-full max-w-md bg-zinc-900 border border-zinc-800 rounded-2xl p-6 shadow-2xl space-y-4">
            <h3 className="text-lg font-bold">Instantiate Durable Actor</h3>
            <form onSubmit={handleCreateInstance} className="space-y-4">
              <div>
                <label className="text-xs font-semibold text-zinc-400 uppercase tracking-wider block mb-1">
                  Actor Key / ObjectID (String or UUID)
                </label>
                <input
                  type="text"
                  required
                  placeholder="e.g. room-general or user:99"
                  value={objectId}
                  onChange={(e) => setObjectId(e.target.value)}
                  className="w-full px-3 py-2 rounded-xl bg-zinc-950 border border-zinc-800 font-mono text-xs focus:outline-none focus:border-emerald-500"
                />
              </div>
              <div className="flex justify-end gap-3 pt-2">
                <button
                  type="button"
                  onClick={() => setShowCreateInstModal(false)}
                  className="px-4 py-2 rounded-lg text-sm text-zinc-400 hover:text-white transition"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  className="px-4 py-2 rounded-lg bg-emerald-500 hover:bg-emerald-600 text-zinc-950 text-sm font-semibold transition"
                >
                  Instantiate
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
