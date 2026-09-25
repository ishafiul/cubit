import React, { useState } from 'react';
import { Server, Terminal, Copy, Check, X, AlertCircle, Info } from 'lucide-react';
import { useQueryClient } from '@tanstack/react-query';
import { useModalStore, useModalActions } from '../../shared/stores/useModalStore';
import { useCreateNode, getListNodesQueryKey } from '../../api/generated/nodes/nodes';
import { useToastActions } from '../../shared/stores/useToastStore';

export function AddNodeModal() {
  const showAddNodeModal = useModalStore((state) => state.showAddNodeModal);
  const { setShowAddNodeModal } = useModalActions();
  const createNodeMutation = useCreateNode();
  const queryClient = useQueryClient();
  const { addToast } = useToastActions();

  const [activeTab, setActiveTab] = useState<'docker' | 'manual'>('docker');
  const [nodeName, setNodeName] = useState('');
  const [ipAddress, setIpAddress] = useState('');
  const [workerPort, setWorkerPort] = useState(8080);
  const [internalPort, setInternalPort] = useState(8081);
  const [storageHost, setStorageHost] = useState(
    typeof window !== 'undefined' && window.location.hostname !== 'localhost'
      ? window.location.hostname
      : '127.0.0.1'
  );
  const [copied, setCopied] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);

  if (!showAddNodeModal) return null;

  const targetIp = ipAddress.trim() || '<NODE_PUBLIC_IP>';
  const joinCommand = `docker run -d \\
  --name celld \\
  --restart always \\
  --net=host \\
  ghcr.io/denoland/celld:latest \\
  --advertise-ip ${targetIp} \\
  --port ${workerPort} \\
  --peer-port ${internalPort} \\
  --endpoint http://${storageHost.trim() || '127.0.0.1'}:3900 \\
  --bucket cubit-fleet`;

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(joinCommand);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch (err) {
      console.error('Failed to copy', err);
    }
  };

  const handleClose = () => {
    setShowAddNodeModal(false);
    setErrorMessage(null);
    setNodeName('');
    setIpAddress('');
  };

  const onSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!nodeName.trim()) {
      setErrorMessage('Node name is required.');
      return;
    }
    if (!ipAddress.trim() || ipAddress.includes('<')) {
      setErrorMessage('Please provide a valid public IP address or hostname.');
      return;
    }

    setIsSubmitting(true);
    setErrorMessage(null);

    try {
      await createNodeMutation.mutateAsync({
        data: {
          name: nodeName.trim(),
          ipAddress: ipAddress.trim(),
          workerPort: Number(workerPort) || 8080,
          internalPort: Number(internalPort) || 8081,
        },
      });
      await queryClient.invalidateQueries({ queryKey: getListNodesQueryKey() });
      addToast({
        title: 'Node Registered',
        description: `Node ${nodeName.trim()} registered in fleet successfully.`,
        variant: 'success',
      });
      handleClose();
    } catch (err: any) {
      console.error('Failed to register node:', err);
      setErrorMessage(err?.response?.data?.error || err?.message || 'Failed to register node in fleet');
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div className="fixed inset-0 bg-black/70 backdrop-blur-sm flex items-center justify-center p-4 z-50 animate-in fade-in duration-200">
      <div className="bg-zinc-900 border border-zinc-800 rounded-2xl max-w-xl w-full p-6 space-y-5 shadow-2xl relative text-zinc-100">
        {/* Header */}
        <div className="flex items-center justify-between pb-2 border-b border-zinc-800/80">
          <div className="flex items-center gap-2.5">
            <div className="w-8 h-8 rounded-lg bg-emerald-500/10 border border-emerald-500/20 flex items-center justify-center text-emerald-400">
              <Server className="w-4 h-4" />
            </div>
            <div>
              <h3 className="text-base font-bold text-zinc-100">Add Fleet Node</h3>
              <p className="text-xs text-zinc-400">Connect a new bare-metal VPS to your Cubit cluster</p>
            </div>
          </div>
          <button
            type="button"
            onClick={handleClose}
            className="p-1 rounded-lg text-zinc-400 hover:text-zinc-200 hover:bg-zinc-800 transition"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* Tab Toggle */}
        <div className="flex rounded-lg bg-zinc-950 p-1 border border-zinc-800">
          <button
            type="button"
            onClick={() => setActiveTab('docker')}
            className={`flex-1 flex items-center justify-center gap-2 py-1.5 px-3 rounded-md text-xs font-semibold transition ${
              activeTab === 'docker'
                ? 'bg-zinc-800 text-emerald-400 shadow-sm'
                : 'text-zinc-400 hover:text-zinc-200'
            }`}
          >
            <Terminal className="w-3.5 h-3.5" />
            <span>1-Click Docker Join</span>
          </button>
          <button
            type="button"
            onClick={() => setActiveTab('manual')}
            className={`flex-1 flex items-center justify-center gap-2 py-1.5 px-3 rounded-md text-xs font-semibold transition ${
              activeTab === 'manual'
                ? 'bg-zinc-800 text-emerald-400 shadow-sm'
                : 'text-zinc-400 hover:text-zinc-200'
            }`}
          >
            <Server className="w-3.5 h-3.5" />
            <span>Manual Registration</span>
          </button>
        </div>

        {errorMessage && (
          <div className="flex items-center gap-2.5 p-3 rounded-lg bg-red-950/40 border border-red-800/80 text-red-300 text-xs">
            <AlertCircle className="w-4 h-4 shrink-0 text-red-400" />
            <span>{errorMessage}</span>
          </div>
        )}

        <form onSubmit={onSubmit} className="space-y-4">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
            <div>
              <label className="text-xs font-medium text-zinc-400 mb-1 block">
                Node Name <span className="text-emerald-400">*</span>
              </label>
              <input
                type="text"
                required
                placeholder="e.g. lon-vps-1"
                value={nodeName}
                onChange={e => setNodeName(e.target.value)}
                className="w-full bg-zinc-950 border border-zinc-800 rounded-lg px-3 py-2 text-sm text-zinc-100 focus:outline-none focus:border-emerald-500 font-mono"
              />
            </div>

            <div>
              <label className="text-xs font-medium text-zinc-400 mb-1 block">
                Node Public IP / Hostname <span className="text-emerald-400">*</span>
              </label>
              <input
                type="text"
                required
                placeholder="e.g. 198.51.100.24"
                value={ipAddress}
                onChange={e => setIpAddress(e.target.value)}
                className="w-full bg-zinc-950 border border-zinc-800 rounded-lg px-3 py-2 text-sm text-zinc-100 focus:outline-none focus:border-emerald-500 font-mono"
              />
            </div>
          </div>

          {activeTab === 'docker' ? (
            <div className="space-y-3">
              <div>
                <label className="text-xs font-medium text-zinc-400 mb-1 block">
                  Garage / S3 Storage Host
                </label>
                <input
                  type="text"
                  placeholder="e.g. 192.168.1.50 or cluster.domain"
                  value={storageHost}
                  onChange={e => setStorageHost(e.target.value)}
                  className="w-full bg-zinc-950 border border-zinc-800 rounded-lg px-3 py-1.5 text-xs text-zinc-100 focus:outline-none focus:border-emerald-500 font-mono"
                />
                <p className="text-[11px] text-zinc-500 mt-1">
                  The host IP or domain where Garage S3 (port 3900) is accessible from the new node.
                </p>
              </div>

              <div className="space-y-1.5">
                <div className="flex items-center justify-between">
                  <span className="text-xs font-medium text-zinc-400">Run Command on Target Node:</span>
                  <button
                    type="button"
                    onClick={handleCopy}
                    className="flex items-center gap-1 text-xs text-emerald-400 hover:text-emerald-300 font-medium transition"
                  >
                    {copied ? (
                      <>
                        <Check className="w-3.5 h-3.5" />
                        <span>Copied!</span>
                      </>
                    ) : (
                      <>
                        <Copy className="w-3.5 h-3.5" />
                        <span>Copy Command</span>
                      </>
                    )}
                  </button>
                </div>
                <pre className="p-3 bg-zinc-950 rounded-lg border border-zinc-800 font-mono text-[11px] text-zinc-300 overflow-x-auto whitespace-pre leading-relaxed">
                  {joinCommand}
                </pre>
              </div>

              <div className="flex items-start gap-2 p-2.5 rounded-lg bg-zinc-950/60 border border-zinc-800/80 text-[11px] text-zinc-400">
                <Info className="w-4 h-4 shrink-0 text-emerald-400 mt-0.5" />
                <span>
                  After running the Docker command on your VPS, click <strong>Register Node in Fleet</strong> below. Cubit will sync Traefik upstream routes automatically.
                </span>
              </div>
            </div>
          ) : (
            <div className="space-y-3">
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="text-xs font-medium text-zinc-400 mb-1 block">
                    Worker Port
                  </label>
                  <input
                    type="number"
                    value={workerPort}
                    onChange={e => setWorkerPort(Number(e.target.value))}
                    className="w-full bg-zinc-950 border border-zinc-800 rounded-lg px-3 py-2 text-sm text-zinc-100 focus:outline-none focus:border-emerald-500 font-mono"
                  />
                  <p className="text-[10px] text-zinc-500 mt-1">Default HTTP worker port (8080)</p>
                </div>

                <div>
                  <label className="text-xs font-medium text-zinc-400 mb-1 block">
                    Peer Sync Port
                  </label>
                  <input
                    type="number"
                    value={internalPort}
                    onChange={e => setInternalPort(Number(e.target.value))}
                    className="w-full bg-zinc-950 border border-zinc-800 rounded-lg px-3 py-2 text-sm text-zinc-100 focus:outline-none focus:border-emerald-500 font-mono"
                  />
                  <p className="text-[10px] text-zinc-500 mt-1">Default peer gossip port (8081)</p>
                </div>
              </div>

              <div className="flex items-start gap-2 p-2.5 rounded-lg bg-zinc-950/60 border border-zinc-800/80 text-[11px] text-zinc-400">
                <Info className="w-4 h-4 shrink-0 text-emerald-400 mt-0.5" />
                <span>
                  Ensure firewall rules allow incoming TCP traffic on worker port {workerPort} from Traefik and peer port {internalPort} across cluster nodes.
                </span>
              </div>
            </div>
          )}

          {/* Action Buttons */}
          <div className="flex justify-end gap-3 pt-3 border-t border-zinc-800/80">
            <button
              type="button"
              onClick={handleClose}
              className="px-4 py-2 rounded-lg border border-zinc-800 text-sm font-medium hover:bg-zinc-800 transition"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={isSubmitting}
              className="flex items-center gap-2 px-4 py-2 rounded-lg bg-emerald-500 hover:bg-emerald-600 disabled:opacity-50 text-zinc-950 text-sm font-semibold transition shadow-sm"
            >
              {isSubmitting ? (
                <>
                  <span className="w-4 h-4 border-2 border-zinc-950 border-t-transparent rounded-full animate-spin" />
                  <span>Registering...</span>
                </>
              ) : (
                <>
                  <Server className="w-4 h-4" />
                  <span>Register Node in Fleet</span>
                </>
              )}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
