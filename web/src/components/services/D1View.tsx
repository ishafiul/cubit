import React, { useState, useEffect } from 'react';
import { Database, Plus, Trash2, Play, RefreshCw, AlertCircle, CheckCircle2, Clock } from 'lucide-react';

interface D1Database {
  id: string;
  name: string;
  sizeBytes: number;
  tablesCount: number;
  createdAt: string;
}

interface D1QueryResult {
  columns: string[];
  rows: Array<Record<string, any>>;
  rowsAffected: number;
  durationMs: number;
}

export function D1View({ initialSelectedId }: { initialSelectedId?: string } = {}) {
  const [databases, setDatabases] = useState<D1Database[]>([]);
  const [selectedDb, setSelectedDb] = useState<D1Database | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Query state
  const [sqlQuery, setSqlQuery] = useState<string>(
    `-- Celld 0.5.1 Serverless SQLite D1 Query
CREATE TABLE IF NOT EXISTS users (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL,
  email TEXT UNIQUE,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO users (name, email) VALUES ('Ada Lovelace', 'ada@cubit.local');
SELECT * FROM users;`
  );
  const [executing, setExecuting] = useState(false);
  const [queryResult, setQueryResult] = useState<D1QueryResult | null>(null);
  const [queryError, setQueryError] = useState<string | null>(null);

  // Modals
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [newDbName, setNewDbName] = useState('');

  const fetchDatabases = async () => {
    setLoading(true);
    try {
      const res = await fetch('/api/v1/d1/databases');
      if (!res.ok) throw new Error('Failed to fetch D1 databases');
      const data = await res.json();
      const list = Array.isArray(data) ? data : [];
      setDatabases(list);
      if (initialSelectedId) {
        const found = list.find(d => d.id === initialSelectedId || d.name === initialSelectedId);
        if (found) {
          setSelectedDb(found);
          return;
        }
      }
      if (list.length > 0 && !selectedDb) {
        setSelectedDb(list[0]);
      }
    } catch (err: any) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (initialSelectedId && databases.length > 0) {
      const found = databases.find(d => d.id === initialSelectedId || d.name === initialSelectedId);
      if (found) {
        setSelectedDb(found);
      }
    }
  }, [initialSelectedId, databases]);

  useEffect(() => {
    fetchDatabases();
  }, []);

  const handleCreateDatabase = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newDbName.trim()) return;
    try {
      const res = await fetch('/api/v1/d1/databases', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name: newDbName.trim() }),
      });
      if (!res.ok) throw new Error('Failed to create D1 database');
      const created = await res.json();
      setShowCreateModal(false);
      setNewDbName('');
      await fetchDatabases();
      setSelectedDb(created);
    } catch (err: any) {
      alert(err.message);
    }
  };

  const handleDeleteDatabase = async (id: string) => {
    if (!confirm('Are you sure you want to delete this D1 database and all its tables?')) return;
    try {
      const res = await fetch(`/api/v1/d1/databases/${id}`, { method: 'DELETE' });
      if (!res.ok) throw new Error('Failed to delete database');
      if (selectedDb?.id === id) {
        setSelectedDb(null);
        setQueryResult(null);
      }
      await fetchDatabases();
    } catch (err: any) {
      alert(err.message);
    }
  };

  const handleExecuteQuery = async () => {
    if (!selectedDb || !sqlQuery.trim()) return;
    setExecuting(true);
    setQueryError(null);
    try {
      const res = await fetch(`/api/v1/d1/databases/${selectedDb.id}/query`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ sql: sqlQuery }),
      });
      const data = await res.json();
      if (!res.ok) {
        throw new Error(data.message || 'Query execution failed');
      }
      setQueryResult(data);
      await fetchDatabases(); // refresh table count
    } catch (err: any) {
      setQueryError(err.message);
      setQueryResult(null);
    } finally {
      setExecuting(false);
    }
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <div className="flex items-center gap-3">
            <h2 className="text-2xl font-bold tracking-tight">D1 SQL Databases</h2>
            <span className="text-xs bg-emerald-950 text-emerald-400 border border-emerald-800 px-2.5 py-0.5 rounded-full font-mono font-medium">
              celld service: Yes
            </span>
          </div>
          <p className="text-sm text-zinc-400 mt-1">
            Serverless SQL database engine powered by isolated SQLite stores with zero cold starts.
          </p>
        </div>
        <button
          onClick={() => setShowCreateModal(true)}
          className="flex items-center gap-2 px-4 py-2 rounded-lg bg-emerald-500 hover:bg-emerald-600 text-zinc-950 text-sm font-semibold transition"
        >
          <Plus className="w-4 h-4" />
          Create D1 Database
        </button>
      </div>

      {error && (
        <div className="p-4 rounded-xl border border-red-900/50 bg-red-950/20 text-red-400 flex items-center gap-3">
          <AlertCircle className="w-5 h-5 flex-shrink-0" />
          <span className="text-sm">{error}</span>
        </div>
      )}

      {/* Main Grid: Left = Databases List, Right = Interactive SQL Console */}
      <div className="grid grid-cols-12 gap-6 min-h-[550px]">
        {/* Databases Sidebar */}
        <div className="col-span-4 border border-zinc-800/80 bg-zinc-900/30 rounded-2xl p-4 flex flex-col justify-between">
          <div className="space-y-3">
            <div className="flex items-center justify-between pb-2 border-b border-zinc-800">
              <span className="text-xs font-semibold text-zinc-400 uppercase tracking-wider">Databases</span>
              <button onClick={fetchDatabases} className="text-zinc-500 hover:text-zinc-300 transition">
                <RefreshCw className={`w-3.5 h-3.5 ${loading ? 'animate-spin' : ''}`} />
              </button>
            </div>

            {databases.length === 0 ? (
              <div className="text-center py-10 text-zinc-500 text-sm">
                No D1 databases created yet.
              </div>
            ) : (
              <div className="space-y-1.5 overflow-y-auto max-h-[480px]">
                {databases.map(db => (
                  <div
                    key={db.id}
                    onClick={() => {
                      setSelectedDb(db);
                      setQueryResult(null);
                      setQueryError(null);
                    }}
                    className={`p-3 rounded-xl cursor-pointer flex items-center justify-between border transition ${
                      selectedDb?.id === db.id
                        ? 'bg-zinc-800/90 border-emerald-500/50 text-white'
                        : 'border-transparent hover:bg-zinc-800/40 text-zinc-300'
                    }`}
                  >
                    <div className="flex items-center gap-3 min-w-0">
                      <Database className={`w-4 h-4 flex-shrink-0 ${selectedDb?.id === db.id ? 'text-emerald-400' : 'text-zinc-500'}`} />
                      <div className="truncate">
                        <div className="font-semibold text-sm truncate">{db.name}</div>
                        <div className="text-[11px] text-zinc-500 font-mono truncate">{db.id}</div>
                      </div>
                    </div>
                    <div className="flex items-center gap-2">
                      <span className="text-xs bg-zinc-800 px-2 py-0.5 rounded font-mono text-zinc-400">
                        {db.tablesCount} tables
                      </span>
                      <button
                        onClick={(e) => {
                          e.stopPropagation();
                          handleDeleteDatabase(db.id);
                        }}
                        className="p-1 hover:text-red-400 text-zinc-500 transition"
                        title="Delete Database"
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

        {/* Interactive SQL Console & Results */}
        <div className="col-span-8 border border-zinc-800/80 bg-zinc-900/30 rounded-2xl p-6 flex flex-col justify-between">
          {selectedDb ? (
            <div className="space-y-4">
              <div className="flex items-center justify-between border-b border-zinc-800 pb-3">
                <div>
                  <div className="flex items-center gap-2">
                    <h3 className="text-lg font-bold">{selectedDb.name}</h3>
                    <span className="text-xs font-mono text-zinc-500">ID: {selectedDb.id}</span>
                  </div>
                  <p className="text-xs text-zinc-400 mt-0.5">Isolated SQLite database file: <span className="font-mono text-zinc-300">.data/d1/{selectedDb.id}.db</span></p>
                </div>
                <div className="flex items-center gap-2">
                  <button
                    onClick={() => setSqlQuery('SELECT name, type FROM sqlite_master WHERE type=\'table\';')}
                    className="text-xs text-zinc-400 hover:text-zinc-200 bg-zinc-800 px-2.5 py-1 rounded transition"
                  >
                    Schema
                  </button>
                  <button
                    onClick={handleExecuteQuery}
                    disabled={executing}
                    className="flex items-center gap-2 px-3.5 py-1.5 rounded-lg bg-emerald-500 hover:bg-emerald-600 disabled:opacity-50 text-zinc-950 text-xs font-semibold transition"
                  >
                    <Play className={`w-3.5 h-3.5 fill-current ${executing ? 'animate-spin' : ''}`} />
                    {executing ? 'Running...' : 'Execute SQL'}
                  </button>
                </div>
              </div>

              {/* SQL Query Editor */}
              <div className="space-y-1.5">
                <div className="flex items-center justify-between text-xs text-zinc-400 font-semibold uppercase tracking-wider">
                  <span>SQL Query Console</span>
                  <span className="text-[11px] text-zinc-500 font-normal">Supports full SQLite DDL, DML, and multi-statements</span>
                </div>
                <textarea
                  rows={6}
                  value={sqlQuery}
                  onChange={(e) => setSqlQuery(e.target.value)}
                  placeholder="Enter SQL statement..."
                  className="w-full p-3.5 rounded-xl bg-zinc-950/80 border border-zinc-800 font-mono text-xs text-zinc-200 focus:outline-none focus:border-emerald-500/60 leading-relaxed"
                />
              </div>

              {/* Execution Error Banner */}
              {queryError && (
                <div className="p-3 rounded-xl border border-red-900/50 bg-red-950/20 text-red-400 text-xs flex items-center gap-2 font-mono">
                  <AlertCircle className="w-4 h-4 flex-shrink-0" />
                  <span>{queryError}</span>
                </div>
              )}

              {/* Execution Results View */}
              {queryResult && (
                <div className="space-y-2">
                  <div className="flex items-center justify-between text-xs text-zinc-400">
                    <div className="flex items-center gap-3">
                      <span className="flex items-center gap-1.5 text-emerald-400 font-medium">
                        <CheckCircle2 className="w-3.5 h-3.5" />
                        Query Executed
                      </span>
                      <span>Rows: {queryResult.rows.length}</span>
                      {queryResult.rowsAffected > 0 && <span>Rows Affected: {queryResult.rowsAffected}</span>}
                    </div>
                    <div className="flex items-center gap-1.5 font-mono text-zinc-500">
                      <Clock className="w-3 h-3" />
                      <span>{queryResult.durationMs.toFixed(2)} ms</span>
                    </div>
                  </div>

                  <div className="overflow-x-auto max-h-[220px] rounded-xl border border-zinc-800 bg-zinc-950/60">
                    {queryResult.columns && queryResult.columns.length > 0 ? (
                      <table className="w-full text-left text-xs">
                        <thead className="border-b border-zinc-800 bg-zinc-900/60 text-zinc-400 font-semibold sticky top-0">
                          <tr>
                            {queryResult.columns.map(col => (
                              <th key={col} className="p-2.5 font-mono">{col}</th>
                            ))}
                          </tr>
                        </thead>
                        <tbody className="divide-y divide-zinc-800/60 font-mono">
                          {queryResult.rows.length === 0 ? (
                            <tr>
                              <td colSpan={queryResult.columns.length} className="p-4 text-center text-zinc-500 font-sans">
                                Query returned 0 rows.
                              </td>
                            </tr>
                          ) : (
                            queryResult.rows.map((row, idx) => (
                              <tr key={idx} className="hover:bg-zinc-800/30 transition">
                                {queryResult.columns.map(col => (
                                  <td key={col} className="p-2.5 text-zinc-300">
                                    {row[col] !== null && row[col] !== undefined ? String(row[col]) : <span className="text-zinc-600">NULL</span>}
                                  </td>
                                ))}
                              </tr>
                            ))
                          )}
                        </tbody>
                      </table>
                    ) : (
                      <div className="p-6 text-center text-zinc-500 text-xs">
                        Statement executed successfully. (0 result columns returned)
                      </div>
                    )}
                  </div>
                </div>
              )}
            </div>
          ) : (
            <div className="flex flex-col items-center justify-center h-full text-center py-20 text-zinc-500">
              <Database className="w-12 h-12 stroke-1 text-zinc-700 mb-3" />
              <p className="text-sm">Select or create a D1 database to start running queries</p>
            </div>
          )}
        </div>
      </div>

      {/* Create Database Modal */}
      {showCreateModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm p-4">
          <div className="w-full max-w-md bg-zinc-900 border border-zinc-800 rounded-2xl p-6 shadow-2xl space-y-4">
            <h3 className="text-lg font-bold">Create D1 Database</h3>
            <form onSubmit={handleCreateDatabase} className="space-y-4">
              <div>
                <label className="text-xs font-semibold text-zinc-400 uppercase tracking-wider block mb-1">
                  Database Name
                </label>
                <input
                  type="text"
                  required
                  placeholder="e.g. production_db, analytics_store"
                  value={newDbName}
                  onChange={(e) => setNewDbName(e.target.value)}
                  className="w-full px-3 py-2 rounded-xl bg-zinc-950 border border-zinc-800 text-sm focus:outline-none focus:border-emerald-500"
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
                  Create
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
