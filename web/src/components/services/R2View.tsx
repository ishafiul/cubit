import React, { useState, useEffect } from 'react';
import { HardDrive, RefreshCw, AlertCircle, FileText, Upload, Folder } from 'lucide-react';

interface R2Bucket {
  name: string;
  objectsCount: number;
  sizeBytes: number;
  createdAt: string;
}

interface R2Object {
  key: string;
  sizeBytes: number;
  contentType: string;
  etag: string;
  lastModified: string;
}

export function R2View() {
  const [buckets, setBuckets] = useState<R2Bucket[]>([]);
  const [selectedBucket, setSelectedBucket] = useState<R2Bucket | null>(null);
  const [objects, setObjects] = useState<R2Object[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Modals
  const [showUploadModal, setShowUploadModal] = useState(false);
  const [uploadKey, setUploadKey] = useState('');
  const [uploadContent, setUploadContent] = useState('');
  const [uploading, setUploading] = useState(false);

  const fetchBuckets = async () => {
    setLoading(true);
    try {
      const res = await fetch('/api/v1/r2/buckets');
      if (!res.ok) throw new Error('Failed to fetch R2 buckets');
      const data = await res.json();
      const list = Array.isArray(data) ? data : [];
      setBuckets(list);
      if (list.length > 0 && !selectedBucket) {
        setSelectedBucket(list[0]);
      }
    } catch (err: any) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

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

  const handleUpload = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!selectedBucket || !uploadKey.trim()) return;
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
      if (!res.ok) throw new Error('Failed to upload object');
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
              <button onClick={fetchBuckets} className="text-zinc-500 hover:text-zinc-300 transition">
                <RefreshCw className={`w-3.5 h-3.5 ${loading ? 'animate-spin' : ''}`} />
              </button>
            </div>

            <div className="space-y-2 overflow-y-auto max-h-[480px]">
              {buckets.map(b => (
                <div
                  key={b.name}
                  onClick={() => setSelectedBucket(b)}
                  className={`p-3.5 rounded-xl cursor-pointer border transition space-y-1 ${
                    selectedBucket?.name === b.name
                      ? 'bg-zinc-800/90 border-emerald-500/50 text-white'
                      : 'border-transparent hover:bg-zinc-800/40 text-zinc-300'
                  }`}
                >
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-2">
                      <Folder className={`w-4 h-4 ${selectedBucket?.name === b.name ? 'text-emerald-400' : 'text-zinc-500'}`} />
                      <span className="font-semibold text-sm">{b.name}</span>
                    </div>
                    <span className="text-xs font-mono bg-zinc-800 px-2 py-0.5 rounded text-emerald-400">
                      {(b.sizeBytes / 1024).toFixed(0)} KB
                    </span>
                  </div>
                  <div className="text-xs text-zinc-500 font-mono">
                    {b.objectsCount} objects indexed
                  </div>
                </div>
              ))}
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
                    <span className="text-xs font-mono bg-zinc-800 px-2 py-0.5 rounded text-zinc-400">
                      S3 Protocol: Active
                    </span>
                  </div>
                  <p className="text-xs text-zinc-400 mt-0.5">Bucket path: <span className="font-mono text-zinc-300">s3://{selectedBucket.name}</span></p>
                </div>
                <button
                  onClick={() => setShowUploadModal(true)}
                  className="flex items-center gap-2 px-3.5 py-1.5 rounded-lg bg-emerald-500/10 hover:bg-emerald-500/20 text-emerald-400 border border-emerald-500/30 text-xs font-semibold transition"
                >
                  <Upload className="w-3.5 h-3.5" />
                  Upload Object
                </button>
              </div>

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
    </div>
  );
}
