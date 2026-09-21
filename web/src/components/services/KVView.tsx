import React, { useState, useEffect } from 'react';
import { Database, Plus, Trash2, Key, RefreshCw, AlertCircle, Search, Edit3 } from 'lucide-react';

interface KVNamespace {
  id: string;
  name: string;
  keyCount: number;
  createdAt: string;
}

interface KVPair {
  namespaceId: string;
  key: string;
  value: string;
  expirationTtl?: number;
  metadata?: string;
  updatedAt: string;
}

export function KVView({ initialSelectedId }: { initialSelectedId?: string } = {}) {
  const [namespaces, setNamespaces] = useState<KVNamespace[]>([]);
  const [selectedNs, setSelectedNs] = useState<KVNamespace | null>(null);
  const [pairs, setPairs] = useState<KVPair[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Modals & Inputs
  const [showCreateNsModal, setShowCreateNsModal] = useState(false);
  const [newNsName, setNewNsName] = useState('');

  const [showPutKeyModal, setShowPutKeyModal] = useState(false);
  const [newKey, setNewKey] = useState('');
  const [newValue, setNewValue] = useState('');
  const [newTtl, setNewTtl] = useState(0);
  const [newMetadata, setNewMetadata] = useState('');

  const [searchQuery, setSearchQuery] = useState('');

  const fetchNamespaces = async () => {
    setLoading(true);
    try {
      const res = await fetch('/api/v1/kv/namespaces');
      if (!res.ok) throw new Error('Failed to fetch KV namespaces');
      const data = await res.json();
      const list = Array.isArray(data) ? data : [];
      setNamespaces(list);
      if (initialSelectedId) {
        const found = list.find(n => n.id === initialSelectedId || n.name === initialSelectedId);
        if (found) {
          setSelectedNs(found);
          return;
        }
      }
      if (list.length > 0 && !selectedNs) {
        setSelectedNs(list[0]);
      }
    } catch (err: any) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (initialSelectedId && namespaces.length > 0) {
      const found = namespaces.find(n => n.id === initialSelectedId || n.name === initialSelectedId);
      if (found) {
        setSelectedNs(found);
      }
    }
  }, [initialSelectedId, namespaces]);

  const fetchPairs = async (nsId: string) => {
    try {
      const res = await fetch(`/api/v1/kv/namespaces/${nsId}/keys`);
      if (!res.ok) throw new Error('Failed to fetch keys');
      const data = await res.json();
      setPairs(Array.isArray(data) ? data : []);
    } catch (err: any) {
      console.error(err);
    }
  };

  useEffect(() => {
    fetchNamespaces();
  }, []);

  useEffect(() => {
    if (selectedNs) {
      fetchPairs(selectedNs.id);
    } else {
      setPairs([]);
    }
  }, [selectedNs]);

  const handleCreateNamespace = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newNsName.trim()) return;
    try {
      const res = await fetch('/api/v1/kv/namespaces', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name: newNsName.trim() }),
      });
      if (!res.ok) throw new Error('Failed to create namespace');
      const created = await res.json();
      setShowCreateNsModal(false);
      setNewNsName('');
      await fetchNamespaces();
      setSelectedNs(created);
    } catch (err: any) {
      alert(err.message);
    }
  };

  const handleDeleteNamespace = async (id: string) => {
    if (!confirm('Are you sure you want to delete this KV namespace and all its keys?')) return;
    try {
      const res = await fetch(`/api/v1/kv/namespaces/${id}`, { method: 'DELETE' });
      if (!res.ok) throw new Error('Failed to delete namespace');
      if (selectedNs?.id === id) {
        setSelectedNs(null);
      }
      await fetchNamespaces();
    } catch (err: any) {
      alert(err.message);
    }
  };

  const handlePutKey = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!selectedNs || !newKey.trim()) return;
    try {
      const res = await fetch(`/api/v1/kv/namespaces/${selectedNs.id}/values/${encodeURIComponent(newKey.trim())}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          value: newValue,
          expiration_ttl: Number(newTtl) || 0,
          metadata: newMetadata,
        }),
      });
      if (!res.ok) throw new Error('Failed to save key-value pair');
      setShowPutKeyModal(false);
      setNewKey('');
      setNewValue('');
      setNewTtl(0);
      setNewMetadata('');
      await fetchPairs(selectedNs.id);
      await fetchNamespaces();
    } catch (err: any) {
      alert(err.message);
    }
  };

  const handleDeleteKey = async (key: string) => {
    if (!selectedNs) return;
    try {
      const res = await fetch(`/api/v1/kv/namespaces/${selectedNs.id}/values/${encodeURIComponent(key)}`, {
        method: 'DELETE',
      });
      if (!res.ok) throw new Error('Failed to delete key');
      await fetchPairs(selectedNs.id);
      await fetchNamespaces();
    } catch (err: any) {
      alert(err.message);
    }
  };

  const filteredPairs = pairs.filter(p => p.key.toLowerCase().includes(searchQuery.toLowerCase()));

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <div className="flex items-center gap-3">
            <h2 className="text-2xl font-bold tracking-tight">KV Namespaces</h2>
            <span className="text-xs bg-emerald-950 text-emerald-400 border border-emerald-800 px-2.5 py-0.5 rounded-full font-mono font-medium">
              celld service: Yes
            </span>
          </div>
          <p className="text-sm text-zinc-400 mt-1">
            Global ultra-low latency key-value storage backed by SQLite and fast local caching.
          </p>
        </div>
        <button
          onClick={() => setShowCreateNsModal(true)}
          className="flex items-center gap-2 px-4 py-2 rounded-lg bg-emerald-500 hover:bg-emerald-600 text-zinc-950 text-sm font-semibold transition"
        >
          <Plus className="w-4 h-4" />
          Create Namespace
        </button>
      </div>

      {error && (
        <div className="p-4 rounded-xl border border-red-900/50 bg-red-950/20 text-red-400 flex items-center gap-3">
          <AlertCircle className="w-5 h-5 flex-shrink-0" />
          <span className="text-sm">{error}</span>
        </div>
      )}

      {/* Main Layout: Left = Namespaces List, Right = Keys & Values Editor */}
      <div className="grid grid-cols-12 gap-6 min-h-[500px]">
        {/* Namespaces Sidebar */}
        <div className="col-span-4 border border-zinc-800/80 bg-zinc-900/30 rounded-2xl p-4 flex flex-col justify-between">
          <div className="space-y-3">
            <div className="flex items-center justify-between pb-2 border-b border-zinc-800">
              <span className="text-xs font-semibold text-zinc-400 uppercase tracking-wider">Namespaces</span>
              <button onClick={fetchNamespaces} className="text-zinc-500 hover:text-zinc-300 transition">
                <RefreshCw className={`w-3.5 h-3.5 ${loading ? 'animate-spin' : ''}`} />
              </button>
            </div>

            {namespaces.length === 0 ? (
              <div className="text-center py-10 text-zinc-500 text-sm">
                No KV namespaces created yet.
              </div>
            ) : (
              <div className="space-y-1.5 overflow-y-auto max-h-[480px]">
                {namespaces.map(ns => (
                  <div
                    key={ns.id}
                    onClick={() => setSelectedNs(ns)}
                    className={`p-3 rounded-xl cursor-pointer flex items-center justify-between border transition ${
                      selectedNs?.id === ns.id
                        ? 'bg-zinc-800/90 border-emerald-500/50 text-white'
                        : 'border-transparent hover:bg-zinc-800/40 text-zinc-300'
                    }`}
                  >
                    <div className="flex items-center gap-3 min-w-0">
                      <Database className={`w-4 h-4 flex-shrink-0 ${selectedNs?.id === ns.id ? 'text-emerald-400' : 'text-zinc-500'}`} />
                      <div className="truncate">
                        <div className="font-semibold text-sm truncate">{ns.name}</div>
                        <div className="text-[11px] text-zinc-500 font-mono truncate">{ns.id}</div>
                      </div>
                    </div>
                    <div className="flex items-center gap-2">
                      <span className="text-xs bg-zinc-800 px-2 py-0.5 rounded font-mono text-zinc-400">
                        {ns.keyCount} keys
                      </span>
                      <button
                        onClick={(e) => {
                          e.stopPropagation();
                          handleDeleteNamespace(ns.id);
                        }}
                        className="p-1 hover:text-red-400 text-zinc-500 transition"
                        title="Delete Namespace"
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

        {/* Keys & Value Browser */}
        <div className="col-span-8 border border-zinc-800/80 bg-zinc-900/30 rounded-2xl p-6 flex flex-col justify-between">
          {selectedNs ? (
            <div className="space-y-4">
              <div className="flex items-center justify-between border-b border-zinc-800 pb-4">
                <div>
                  <div className="flex items-center gap-2">
                    <h3 className="text-lg font-bold">{selectedNs.name}</h3>
                    <span className="text-xs font-mono text-zinc-500">ID: {selectedNs.id}</span>
                  </div>
                  <p className="text-xs text-zinc-400 mt-0.5">Direct KV store bindings available to all Celld workers.</p>
                </div>
                <button
                  onClick={() => {
                    setNewKey('');
                    setNewValue('');
                    setNewTtl(0);
                    setNewMetadata('');
                    setShowPutKeyModal(true);
                  }}
                  className="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-emerald-500/10 hover:bg-emerald-500/20 text-emerald-400 border border-emerald-500/30 text-xs font-medium transition"
                >
                  <Plus className="w-3.5 h-3.5" />
                  Add Key / Value
                </button>
              </div>

              {/* Search Keys */}
              <div className="relative">
                <Search className="w-4 h-4 absolute left-3.5 top-3 text-zinc-500" />
                <input
                  type="text"
                  placeholder="Filter keys in this namespace..."
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  className="w-full pl-10 pr-4 py-2 rounded-xl bg-zinc-950/60 border border-zinc-800 text-xs text-zinc-200 placeholder-zinc-500 focus:outline-none focus:border-emerald-500/60"
                />
              </div>

              {/* Keys Table */}
              <div className="overflow-x-auto rounded-xl border border-zinc-800 bg-zinc-950/40">
                <table className="w-full text-left text-xs">
                  <thead className="border-b border-zinc-800 bg-zinc-900/50 text-zinc-400 font-semibold">
                    <tr>
                      <th className="p-3">Key Name</th>
                      <th className="p-3">Stored Value</th>
                      <th className="p-3">TTL</th>
                      <th className="p-3 text-right">Actions</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-zinc-800/60 font-mono">
                    {filteredPairs.length === 0 ? (
                      <tr>
                        <td colSpan={4} className="p-8 text-center text-zinc-500 font-sans">
                          No key-value pairs stored in this namespace.
                        </td>
                      </tr>
                    ) : (
                      filteredPairs.map(p => (
                        <tr key={p.key} className="hover:bg-zinc-800/30 transition">
                          <td className="p-3 font-semibold text-emerald-400">
                            <div className="flex items-center gap-2">
                              <Key className="w-3.5 h-3.5 text-zinc-500 flex-shrink-0" />
                              <span>{p.key}</span>
                            </div>
                          </td>
                          <td className="p-3 max-w-[260px] truncate text-zinc-300 font-sans">
                            {p.value}
                          </td>
                          <td className="p-3 text-zinc-400">
                            {p.expirationTtl && p.expirationTtl > 0 ? `${p.expirationTtl}s` : 'None'}
                          </td>
                          <td className="p-3 text-right space-x-2">
                            <button
                              onClick={() => {
                                setNewKey(p.key);
                                setNewValue(p.value);
                                setNewTtl(p.expirationTtl || 0);
                                setNewMetadata(p.metadata || '');
                                setShowPutKeyModal(true);
                              }}
                              className="p-1 hover:text-emerald-400 text-zinc-400 transition"
                              title="Edit Value"
                            >
                              <Edit3 className="w-3.5 h-3.5" />
                            </button>
                            <button
                              onClick={() => handleDeleteKey(p.key)}
                              className="p-1 hover:text-red-400 text-zinc-500 transition"
                              title="Delete Key"
                            >
                              <Trash2 className="w-3.5 h-3.5" />
                            </button>
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
              <Database className="w-12 h-12 stroke-1 text-zinc-700 mb-3" />
              <p className="text-sm">Select a KV namespace from the list to inspect keys and values</p>
            </div>
          )}
        </div>
      </div>

      {/* Create Namespace Modal */}
      {showCreateNsModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm p-4">
          <div className="w-full max-w-md bg-zinc-900 border border-zinc-800 rounded-2xl p-6 shadow-2xl space-y-4">
            <h3 className="text-lg font-bold">Create KV Namespace</h3>
            <form onSubmit={handleCreateNamespace} className="space-y-4">
              <div>
                <label className="text-xs font-semibold text-zinc-400 uppercase tracking-wider block mb-1">
                  Namespace Name
                </label>
                <input
                  type="text"
                  required
                  placeholder="e.g. user_sessions, cache_store"
                  value={newNsName}
                  onChange={(e) => setNewNsName(e.target.value)}
                  className="w-full px-3 py-2 rounded-xl bg-zinc-950 border border-zinc-800 text-sm focus:outline-none focus:border-emerald-500"
                />
              </div>
              <div className="flex justify-end gap-3 pt-2">
                <button
                  type="button"
                  onClick={() => setShowCreateNsModal(false)}
                  className="px-4 py-2 rounded-lg text-sm text-zinc-400 hover:text-white transition"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  className="px-4 py-2 rounded-lg bg-emerald-500 hover:bg-emerald-600 text-zinc-950 text-sm font-semibold transition"
                >
                  Create
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Put Key Modal */}
      {showPutKeyModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm p-4">
          <div className="w-full max-w-lg bg-zinc-900 border border-zinc-800 rounded-2xl p-6 shadow-2xl space-y-4">
            <h3 className="text-lg font-bold">Write KV Pair</h3>
            <form onSubmit={handlePutKey} className="space-y-4">
              <div>
                <label className="text-xs font-semibold text-zinc-400 uppercase tracking-wider block mb-1">
                  Key
                </label>
                <input
                  type="text"
                  required
                  placeholder="e.g. session:user_42"
                  value={newKey}
                  onChange={(e) => setNewKey(e.target.value)}
                  className="w-full px-3 py-2 rounded-xl bg-zinc-950 border border-zinc-800 font-mono text-xs focus:outline-none focus:border-emerald-500"
                />
              </div>
              <div>
                <label className="text-xs font-semibold text-zinc-400 uppercase tracking-wider block mb-1">
                  Value
                </label>
                <textarea
                  rows={4}
                  required
                  placeholder="Arbitrary text, JSON string, or serialized payload..."
                  value={newValue}
                  onChange={(e) => setNewValue(e.target.value)}
                  className="w-full px-3 py-2 rounded-xl bg-zinc-950 border border-zinc-800 font-mono text-xs focus:outline-none focus:border-emerald-500"
                />
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="text-xs font-semibold text-zinc-400 uppercase tracking-wider block mb-1">
                    Expiration TTL (seconds)
                  </label>
                  <input
                    type="number"
                    min={0}
                    placeholder="0 = never expires"
                    value={newTtl}
                    onChange={(e) => setNewTtl(parseInt(e.target.value) || 0)}
                    className="w-full px-3 py-2 rounded-xl bg-zinc-950 border border-zinc-800 text-xs focus:outline-none focus:border-emerald-500"
                  />
                </div>
                <div>
                  <label className="text-xs font-semibold text-zinc-400 uppercase tracking-wider block mb-1">
                    Metadata (optional)
                  </label>
                  <input
                    type="text"
                    placeholder="e.g. {'role': 'admin'}"
                    value={newMetadata}
                    onChange={(e) => setNewMetadata(e.target.value)}
                    className="w-full px-3 py-2 rounded-xl bg-zinc-950 border border-zinc-800 text-xs focus:outline-none focus:border-emerald-500"
                  />
                </div>
              </div>
              <div className="flex justify-end gap-3 pt-2">
                <button
                  type="button"
                  onClick={() => setShowPutKeyModal(false)}
                  className="px-4 py-2 rounded-lg text-sm text-zinc-400 hover:text-white transition"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  className="px-4 py-2 rounded-lg bg-emerald-500 hover:bg-emerald-600 text-zinc-950 text-sm font-semibold transition"
                >
                  Save Key
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
