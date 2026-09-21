import React, { useState, useEffect } from 'react';
import { HardDrive, RefreshCw, AlertCircle, FileText, Upload, Folder, Plus, Trash2, Lock } from 'lucide-react';

export interface R2Bucket {
  name: string;
  objectsCount: number;
  sizeBytes: number;
  createdAt: string;
  isSystem?: boolean;
}

export function isSystemBucket(bucket?: { name: string; isSystem?: boolean } | null): boolean {
  if (!bucket) return false;
  return Boolean(bucket.isSystem || bucket.name === 'cubit-fleet');
}

export function isValidBucketName(name: string): boolean {
  const s3Regex = /^[a-z0-9][a-z0-9.-]{1,61}[a-z0-9]$/;
  return s3Regex.test(name);
}

interface R2Object {
  key: string;
  sizeBytes: number;
  contentType: string;
  etag: string;
  lastModified: string;
}

export function R2View({ initialSelectedId }: { initialSelectedId?: string } = {}) {
  const [buckets, setBuckets] = useState<R2Bucket[]>([]);
  const [selectedBucket, setSelectedBucket] = useState<R2Bucket | null>(null);
  const [objects, setObjects] = useState<R2Object[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Modals & form state
  const [showUploadModal, setShowUploadModal] = useState(false);
  const [uploadKey, setUploadKey] = useState('');
  const [uploadContent, setUploadContent] = useState('');
  const [uploading, setUploading] = useState(false);

  const [showCreateBucketModal, setShowCreateBucketModal] = useState(false);
  const [newBucketName, setNewBucketName] = useState('');
  const [creatingBucket, setCreatingBucket] = useState(false);
  const [createBucketError, setCreateBucketError] = useState<string | null>(null);
  const [deletingBucketName, setDeletingBucketName] = useState<string | null>(null);

  const fetchBuckets = async () => {
    setLoading(true);
    try {
      const res = await fetch('/api/v1/r2/buckets');
      if (!res.ok) throw new Error('Failed to fetch R2 buckets');
      const data = await res.json();
      const list = Array.isArray(data) ? data : [];
      setBuckets(list);
      if (initialSelectedId) {
        const found = list.find(b => b.name === initialSelectedId);
        if (found) {
          setSelectedBucket(found);
          return;
        }
      }
      if (list.length > 0 && !selectedBucket) {
        setSelectedBucket(list[0]);
      }
    } catch (err: any) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (initialSelectedId && buckets.length > 0) {
      const found = buckets.find(b => b.name === initialSelectedId);
      if (found) {
        setSelectedBucket(found);
      }
    }
  }, [initialSelectedId, buckets]);

  const fetchObjects = async (bucketName: string) => {
    try {
      const res = await fetch(`/api/v1/r2/buckets/${bucketName}/objects`);
      if (!res.ok) throw new Error('Failed to fetch objects');
      const data = await res.json();
      setObjects(Array.isArray(data) ? data : []);
    } catch (err: any) {
      console.error(err);
    }
  };

  useEffect(() => {
    fetchBuckets();
  }, []);

  useEffect(() => {
    if (selectedBucket) {
      fetchObjects(selectedBucket.name);
    } else {
      setObjects([]);
    }
  }, [selectedBucket]);

  const handleCreateBucket = async (e: React.FormEvent) => {
    e.preventDefault();
    const name = newBucketName.trim().toLowerCase();
    if (!name) return;
    if (!isValidBucketName(name)) {
      setCreateBucketError('Bucket name must be 3-63 characters (lowercase letters, numbers, hyphens, and dots) and cannot start or end with a hyphen or dot.');
      return;
    }

    setCreatingBucket(true);
    setCreateBucketError(null);
    try {
      const res = await fetch('/api/v1/r2/buckets', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name }),
      });
      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        throw new Error(data.message || `Failed to create bucket (${res.status})`);
      }
      const created = await res.json();
      setShowCreateBucketModal(false);
      setNewBucketName('');
      await fetchBuckets();
      setSelectedBucket(created);
    } catch (err: any) {
      setCreateBucketError(err.message);
    } finally {
      setCreatingBucket(false);
    }
  };

  const handleDeleteBucket = async (bucket: R2Bucket) => {
    if (isSystemBucket(bucket)) {
      alert('The cubit-fleet system bucket is protected and cannot be deleted.');
      return;
    }
    if (!window.confirm(`Are you sure you want to permanently delete bucket "${bucket.name}" and all objects stored inside it?`)) {
      return;
    }
    setDeletingBucketName(bucket.name);
    try {
      const res = await fetch(`/api/v1/r2/buckets/${bucket.name}`, {
        method: 'DELETE',
      });
      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        throw new Error(data.message || `Failed to delete bucket (${res.status})`);
      }
      if (selectedBucket?.name === bucket.name) {
        setSelectedBucket(null);
      }
      await fetchBuckets();
    } catch (err: any) {
      alert(err.message);
    } finally {
      setDeletingBucketName(null);
    }
  };

  const handleUpload = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!selectedBucket || !uploadKey.trim()) return;
    if (isSystemBucket(selectedBucket)) {
      alert('Manual uploads to the cubit-fleet system bucket are restricted.');
      return;
    }
    setUploading(true);
    try {
      const res = await fetch(`/api/v1/r2/buckets/${selectedBucket.name}/upload`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          key: uploadKey.trim(),
          content: uploadContent,
        }),
      });
      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        throw new Error(data.message || 'Failed to upload object');
      }
      setShowUploadModal(false);
      setUploadKey('');
      setUploadContent('');
      await fetchObjects(selectedBucket.name);
    } catch (err: any) {
      alert(err.message);
    } finally {
      setUploading(false);
    }
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <div className="flex items-center gap-3">
            <h2 className="text-2xl font-bold tracking-tight">R2 Object Storage</h2>
            <span className="text-xs bg-emerald-950 text-emerald-400 border border-emerald-800 px-2.5 py-0.5 rounded-full font-mono font-medium">
              celld service: Yes
            </span>
          </div>
          <p className="text-sm text-zinc-400 mt-1">
            Zero-egress fee S3-compatible object storage powered by Garage and local block disks.
          </p>
        </div>
        <button
          onClick={() => {
            setCreateBucketError(null);
            setNewBucketName('');
            setShowCreateBucketModal(true);
          }}
          className="flex items-center gap-2 px-4 py-2 rounded-xl bg-emerald-500 hover:bg-emerald-600 text-zinc-950 font-semibold text-sm transition shadow-lg shadow-emerald-950/20"
        >
          <Plus className="w-4 h-4" />
          Create Bucket
        </button>
      </div>

      {error && (
        <div className="p-4 rounded-xl border border-red-900/50 bg-red-950/20 text-red-400 flex items-center gap-3">
          <AlertCircle className="w-5 h-5 flex-shrink-0" />
          <span className="text-sm">{error}</span>
        </div>
      )}

      {/* Main Grid: Left = Buckets List, Right = Objects Browser */}
      <div className="grid grid-cols-12 gap-6 min-h-[500px]">
        {/* Left Column: Buckets */}
        <div className="col-span-4 border border-zinc-800/80 bg-zinc-900/30 rounded-2xl p-4 flex flex-col justify-between">
          <div className="space-y-3">
            <div className="flex items-center justify-between pb-2 border-b border-zinc-800">
              <span className="text-xs font-semibold text-zinc-400 uppercase tracking-wider">Fleet Buckets</span>
              <div className="flex items-center gap-2">
                <button
                  onClick={() => {
                    setCreateBucketError(null);
                    setNewBucketName('');
                    setShowCreateBucketModal(true);
                  }}
                  className="flex items-center gap-1 px-2.5 py-1 rounded-lg bg-emerald-500/10 hover:bg-emerald-500/20 text-emerald-400 border border-emerald-500/30 text-xs font-medium transition"
                  title="Create new R2 bucket"
                >
                  <Plus className="w-3.5 h-3.5" />
                  <span>New</span>
                </button>
                <button onClick={fetchBuckets} className="text-zinc-500 hover:text-zinc-300 transition" title="Refresh buckets">
                  <RefreshCw className={`w-3.5 h-3.5 ${loading ? 'animate-spin' : ''}`} />
                </button>
              </div>
            </div>

            <div className="space-y-2 overflow-y-auto max-h-[480px]">
              {buckets.map(b => {
                const isSystem = isSystemBucket(b);
                return (
                  <div
                    key={b.name}
                    onClick={() => setSelectedBucket(b)}
                    className={`group p-3.5 rounded-xl cursor-pointer border transition space-y-1.5 ${
                      selectedBucket?.name === b.name
                        ? 'bg-zinc-800/90 border-emerald-500/50 text-white'
                        : 'border-transparent hover:bg-zinc-800/40 text-zinc-300'
                    }`}
                  >
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-2 min-w-0">
                        <Folder className={`w-4 h-4 flex-shrink-0 ${selectedBucket?.name === b.name ? 'text-emerald-400' : 'text-zinc-500'}`} />
                        <span className="font-semibold text-sm truncate">{b.name}</span>
                        {isSystem && (
                          <span className="inline-flex items-center gap-1 text-[10px] font-medium bg-amber-500/10 text-amber-400 border border-amber-500/20 px-1.5 py-0.2 rounded flex-shrink-0">
                            <Lock className="w-2.5 h-2.5" />
                            System
                          </span>
                        )}
                      </div>
                      <div className="flex items-center gap-2">
                        <span className="text-xs font-mono bg-zinc-800 px-2 py-0.5 rounded text-emerald-400">
                          {(b.sizeBytes / 1024).toFixed(0)} KB
                        </span>
                        {!isSystem && (
                          <button
                            type="button"
                            title={`Delete bucket ${b.name}`}
                            disabled={deletingBucketName === b.name}
                            onClick={(e) => {
                              e.stopPropagation();
                              handleDeleteBucket(b);
                            }}
                            className="p-1 rounded text-zinc-500 hover:text-red-400 hover:bg-red-500/10 transition opacity-0 group-hover:opacity-100"
                          >
                            <Trash2 className="w-3.5 h-3.5" />
                          </button>
                        )}
                      </div>
                    </div>
                    <div className="text-xs text-zinc-500 font-mono">
                      {b.objectsCount} objects indexed
                    </div>
                  </div>
                );
              })}
            </div>
          </div>
        </div>

        {/* Right Column: Objects Browser */}
        <div className="col-span-8 border border-zinc-800/80 bg-zinc-900/30 rounded-2xl p-6 flex flex-col justify-between">
          {selectedBucket ? (
            <div className="space-y-4">
              <div className="flex items-center justify-between border-b border-zinc-800 pb-3">
                <div>
                  <div className="flex items-center gap-2">
                    <h3 className="text-lg font-bold">{selectedBucket.name}</h3>
                    {isSystemBucket(selectedBucket) ? (
                      <span className="inline-flex items-center gap-1 text-xs font-medium bg-amber-500/10 text-amber-400 border border-amber-500/20 px-2 py-0.5 rounded-full font-mono">
                        <Lock className="w-3 h-3" />
                        Fleet System Bucket (Locked)
                      </span>
                    ) : (
                      <span className="text-xs font-mono bg-zinc-800 px-2 py-0.5 rounded text-zinc-400">
                        S3 Protocol: Active
                      </span>
                    )}
                  </div>
                  <p className="text-xs text-zinc-400 mt-0.5">Bucket path: <span className="font-mono text-zinc-300">s3://{selectedBucket.name}</span></p>
                </div>
                {isSystemBucket(selectedBucket) ? (
                  <div className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-zinc-800/60 border border-zinc-700/50 text-zinc-400 text-xs font-mono">
                    <Lock className="w-3.5 h-3.5 text-amber-400" />
                    <span>Uploads Restricted</span>
                  </div>
                ) : (
                  <button
                    onClick={() => setShowUploadModal(true)}
                    className="flex items-center gap-2 px-3.5 py-1.5 rounded-lg bg-emerald-500/10 hover:bg-emerald-500/20 text-emerald-400 border border-emerald-500/30 text-xs font-semibold transition"
                  >
                    <Upload className="w-3.5 h-3.5" />
                    Upload Object
                  </button>
                )}
              </div>

              {isSystemBucket(selectedBucket) && (
                <div className="p-3.5 rounded-xl border border-amber-500/30 bg-amber-500/10 text-amber-300 text-xs flex items-center gap-3">
                  <Lock className="w-4 h-4 flex-shrink-0 text-amber-400" />
                  <div>
                    <span className="font-semibold">Protected Fleet System Storage:</span> Managed automatically by Cubit for internal worker bundles and Durable Object snapshots. Direct uploads and bucket deletion are restricted to protect fleet integrity.
                  </div>
                </div>
              )}

              <div className="flex items-center justify-between text-xs text-zinc-400 font-semibold uppercase tracking-wider">
                <div className="flex items-center gap-2">
                  <FileText className="w-4 h-4 text-emerald-400" />
                  <span>Stored Objects ({objects.length})</span>
                </div>
                <button onClick={() => fetchObjects(selectedBucket.name)} className="text-zinc-500 hover:text-zinc-300">
                  <RefreshCw className="w-3 h-3" />
                </button>
              </div>

              <div className="overflow-x-auto rounded-xl border border-zinc-800 bg-zinc-950/50">
                <table className="w-full text-left text-xs">
                  <thead className="border-b border-zinc-800 bg-zinc-900/50 text-zinc-400 font-semibold">
                    <tr>
                      <th className="p-3">Object Key</th>
                      <th className="p-3">Size</th>
                      <th className="p-3">MIME Type</th>
                      <th className="p-3">ETag</th>
                      <th className="p-3">Last Modified</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-zinc-800/60 font-mono">
                    {objects.length === 0 ? (
                      <tr>
                        <td colSpan={5} className="p-8 text-center text-zinc-500 font-sans">
                          No objects found in this bucket.
                        </td>
                      </tr>
                    ) : (
                      objects.map(obj => (
                        <tr key={obj.key} className="hover:bg-zinc-800/30 transition">
                          <td className="p-3 text-emerald-400 font-semibold">
                            {obj.key}
                          </td>
                          <td className="p-3 text-zinc-300">
                            {(obj.sizeBytes / 1024).toFixed(1)} KB
                          </td>
                          <td className="p-3 text-zinc-400">
                            {obj.contentType}
                          </td>
                          <td className="p-3 text-zinc-500 text-[11px]">
                            {obj.etag}
                          </td>
                          <td className="p-3 text-zinc-500 text-[11px] font-sans">
                            {new Date(obj.lastModified).toLocaleDateString()}
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
              <HardDrive className="w-12 h-12 stroke-1 text-zinc-700 mb-3" />
              <p className="text-sm">Select an R2 bucket to browse files and bundles</p>
            </div>
          )}
        </div>
      </div>

      {/* Upload Modal */}
      {showUploadModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm p-4">
          <div className="w-full max-w-md bg-zinc-900 border border-zinc-800 rounded-2xl p-6 shadow-2xl space-y-4">
            <h3 className="text-lg font-bold">Upload Object to R2</h3>
            <form onSubmit={handleUpload} className="space-y-4">
              <div>
                <label className="text-xs font-semibold text-zinc-400 uppercase tracking-wider block mb-1">
                  Object Key (Path)
                </label>
                <input
                  type="text"
                  required
                  placeholder="e.g. assets/config.json or bundles/app.js"
                  value={uploadKey}
                  onChange={(e) => setUploadKey(e.target.value)}
                  className="w-full px-3 py-2 rounded-xl bg-zinc-950 border border-zinc-800 font-mono text-xs focus:outline-none focus:border-emerald-500"
                />
              </div>
              <div>
                <label className="text-xs font-semibold text-zinc-400 uppercase tracking-wider block mb-1">
                  Content / Payload
                </label>
                <textarea
                  rows={4}
                  required
                  placeholder="Object content text..."
                  value={uploadContent}
                  onChange={(e) => setUploadContent(e.target.value)}
                  className="w-full p-3 rounded-xl bg-zinc-950 border border-zinc-800 font-mono text-xs focus:outline-none focus:border-emerald-500"
                />
              </div>
              <div className="flex justify-end gap-3 pt-2">
                <button
                  type="button"
                  onClick={() => setShowUploadModal(false)}
                  className="px-4 py-2 rounded-lg text-sm text-zinc-400 hover:text-white transition"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  disabled={uploading}
                  className="px-4 py-2 rounded-lg bg-emerald-500 hover:bg-emerald-600 disabled:opacity-50 text-zinc-950 text-sm font-semibold transition"
                >
                  {uploading ? 'Uploading...' : 'Upload'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Create Bucket Modal */}
      {showCreateBucketModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm p-4">
          <div className="w-full max-w-md bg-zinc-900 border border-zinc-800 rounded-2xl p-6 shadow-2xl space-y-4">
            <div className="flex items-center justify-between">
              <h3 className="text-lg font-bold">Create R2 Storage Bucket</h3>
              <button
                type="button"
                onClick={() => setShowCreateBucketModal(false)}
                className="text-zinc-500 hover:text-zinc-300 text-sm"
              >
                ✕
              </button>
            </div>
            <p className="text-xs text-zinc-400">
              Create an S3-compatible object storage bucket with zero egress fees. Custom buckets can be bound directly to your Workers in application settings.
            </p>

            {createBucketError && (
              <div className="p-3 rounded-lg border border-red-900/50 bg-red-950/20 text-red-400 text-xs flex items-center gap-2">
                <AlertCircle className="w-4 h-4 flex-shrink-0" />
                <span>{createBucketError}</span>
              </div>
            )}

            <form onSubmit={handleCreateBucket} className="space-y-4">
              <div>
                <label className="text-xs font-semibold text-zinc-400 uppercase tracking-wider block mb-1">
                  Bucket Name
                </label>
                <input
                  type="text"
                  required
                  autoFocus
                  placeholder="e.g. app-assets or user-uploads"
                  value={newBucketName}
                  onChange={(e) => setNewBucketName(e.target.value.toLowerCase())}
                  className="w-full px-3 py-2 rounded-xl bg-zinc-950 border border-zinc-800 font-mono text-xs focus:outline-none focus:border-emerald-500 text-white"
                />
                <p className="text-[11px] text-zinc-500 mt-1">
                  3-63 characters, lowercase letters, numbers, hyphens, and dots.
                </p>
              </div>
              <div className="flex justify-end gap-3 pt-2">
                <button
                  type="button"
                  onClick={() => setShowCreateBucketModal(false)}
                  className="px-4 py-2 rounded-lg text-sm text-zinc-400 hover:text-white transition"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  disabled={creatingBucket || !newBucketName.trim()}
                  className="px-4 py-2 rounded-lg bg-emerald-500 hover:bg-emerald-600 disabled:opacity-50 text-zinc-950 text-sm font-semibold transition"
                >
                  {creatingBucket ? 'Creating...' : 'Create Bucket'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
