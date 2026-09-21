import React, { useState, useEffect } from 'react';
import { Plus, Trash2, RefreshCw, AlertCircle, FileCode, ExternalLink } from 'lucide-react';

interface StaticSite {
  id: string;
  name: string;
  subdomain: string;
  indexDocument: string;
  assetsCount: number;
  createdAt: string;
}

export function StaticAssetsView() {
  const [sites, setSites] = useState<StaticSite[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Modal
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [name, setName] = useState('');
  const [subdomain, setSubdomain] = useState('');

  const fetchSites = async () => {
    setLoading(true);
    try {
      const res = await fetch('/api/v1/static-assets');
      if (!res.ok) throw new Error('Failed to fetch static sites');
      const data = await res.json();
      setSites(Array.isArray(data) ? data : []);
    } catch (err: any) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchSites();
  }, []);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim()) return;
    try {
      const res = await fetch('/api/v1/static-assets', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          name: name.trim(),
          subdomain: (subdomain || name).trim().toLowerCase().replace(/[^a-z0-9-]/g, '-'),
        }),
      });
      if (!res.ok) throw new Error('Failed to create static site');
      setShowCreateModal(false);
      setName('');
      setSubdomain('');
      await fetchSites();
    } catch (err: any) {
      alert(err.message);
    }
  };

  const handleDelete = async (id: string) => {
    if (!confirm('Are you sure you want to delete this static site?')) return;
    try {
      const res = await fetch(`/api/v1/static-assets/${id}`, { method: 'DELETE' });
      if (!res.ok) throw new Error('Failed to delete static site');
      await fetchSites();
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
            <h2 className="text-2xl font-bold tracking-tight">Static Assets</h2>
            <span className="text-xs bg-emerald-950 text-emerald-400 border border-emerald-800 px-2.5 py-0.5 rounded-full font-mono font-medium">
              celld service: Yes
            </span>
          </div>
          <p className="text-sm text-zinc-400 mt-1">
            Serve SPAs, HTML, client JS, CSS, and media directly from fast R2 edge cache storage with automatic asset hashing.
          </p>
        </div>
        <button
          onClick={() => setShowCreateModal(true)}
          className="flex items-center gap-2 px-4 py-2 rounded-lg bg-emerald-500 hover:bg-emerald-600 text-zinc-950 text-sm font-semibold transition"
        >
          <Plus className="w-4 h-4" />
          Deploy Static Site
        </button>
      </div>

      {error && (
        <div className="p-4 rounded-xl border border-red-900/50 bg-red-950/20 text-red-400 flex items-center gap-3">
          <AlertCircle className="w-5 h-5 flex-shrink-0" />
          <span className="text-sm">{error}</span>
        </div>
      )}

      {/* Sites Grid */}
      <div className="space-y-3">
        <div className="flex items-center justify-between">
          <span className="text-xs font-semibold text-zinc-400 uppercase tracking-wider">
            Static Asset Sites ({sites.length})
          </span>
          <button onClick={fetchSites} className="text-zinc-500 hover:text-zinc-300">
            <RefreshCw className={`w-3.5 h-3.5 ${loading ? 'animate-spin' : ''}`} />
          </button>
        </div>

        {sites.length === 0 ? (
          <div className="p-12 text-center rounded-2xl border border-zinc-800/80 bg-zinc-900/20 text-zinc-500">
            <FileCode className="w-12 h-12 stroke-1 text-zinc-700 mx-auto mb-3" />
            <p className="text-sm">No static sites configured yet.</p>
            <p className="text-xs text-zinc-600 mt-1">Deploy Vite, React, Astro, or static HTML/CSS sites with instant global edge delivery.</p>
          </div>
        ) : (
          <div className="grid grid-cols-2 gap-4">
            {sites.map(s => (
              <div key={s.id} className="p-5 rounded-2xl border border-zinc-800/80 bg-zinc-900/30 space-y-4 hover:border-zinc-700 transition">
                <div className="flex items-start justify-between">
                  <div>
                    <div className="flex items-center gap-2">
                      <h4 className="font-semibold text-base">{s.name}</h4>
                      <span className="text-[10px] px-2 py-0.5 rounded-full font-bold uppercase bg-emerald-950 text-emerald-400 border border-emerald-800">
                        Active
                      </span>
                    </div>
                    <p className="text-xs text-zinc-500 font-mono mt-0.5">{s.id}</p>
                  </div>
                  <button
                    onClick={() => handleDelete(s.id)}
                    className="p-1.5 hover:text-red-400 text-zinc-500 transition"
                    title="Delete Site"
                  >
                    <Trash2 className="w-4 h-4" />
                  </button>
                </div>

                <div className="space-y-2 text-xs text-zinc-400">
                  <div className="flex items-center justify-between p-2.5 rounded-xl bg-zinc-950/60 border border-zinc-800/60">
                    <span className="text-zinc-500">Live URL:</span>
                    <a
                      href={`http://${s.subdomain}.localhost:8000`}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="font-mono text-emerald-400 hover:underline flex items-center gap-1.5"
                    >
                      <span>http://{s.subdomain}.localhost:8000</span>
                      <ExternalLink className="w-3 h-3" />
                    </a>
                  </div>

                  <div className="grid grid-cols-2 gap-2 text-zinc-400 font-mono text-[11px]">
                    <div className="p-2 rounded-lg bg-zinc-950/40 border border-zinc-800/40">
                      Index: <span className="text-zinc-300">{s.indexDocument || 'index.html'}</span>
                    </div>
                    <div className="p-2 rounded-lg bg-zinc-950/40 border border-zinc-800/40">
                      Files: <span className="text-zinc-300">{s.assetsCount} assets</span>
                    </div>
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Deploy Static Site Modal */}
      {showCreateModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm p-4">
          <div className="w-full max-w-md bg-zinc-900 border border-zinc-800 rounded-2xl p-6 shadow-2xl space-y-4">
            <h3 className="text-lg font-bold">Deploy Static Asset Site</h3>
            <form onSubmit={handleCreate} className="space-y-4">
              <div>
                <label className="text-xs font-semibold text-zinc-400 uppercase tracking-wider block mb-1">
                  Site Name
                </label>
                <input
                  type="text"
                  required
                  placeholder="e.g. docs-site, frontend-app"
                  value={name}
                  onChange={(e) => {
                    setName(e.target.value);
                    if (!subdomain) {
                      setSubdomain(e.target.value.toLowerCase().replace(/[^a-z0-9-]/g, '-'));
                    }
                  }}
                  className="w-full px-3 py-2 rounded-xl bg-zinc-950 border border-zinc-800 text-sm focus:outline-none focus:border-emerald-500"
                />
              </div>
              <div>
                <label className="text-xs font-semibold text-zinc-400 uppercase tracking-wider block mb-1">
                  Subdomain Preview
                </label>
                <input
                  type="text"
                  required
                  placeholder="e.g. docs"
                  value={subdomain}
                  onChange={(e) => setSubdomain(e.target.value)}
                  className="w-full px-3 py-2 rounded-xl bg-zinc-950 border border-zinc-800 font-mono text-xs focus:outline-none focus:border-emerald-500"
                />
                <p className="text-[11px] text-zinc-500 mt-1">Accessible at: <span className="text-emerald-400 font-mono">{subdomain || 'name'}.localhost:8000</span></p>
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
                  Deploy Site
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
