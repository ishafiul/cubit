import { useState } from 'react';
import {
  Server,
  Radio,
  ShieldCheck,
  HardDrive,
  Plus,
  Play,
  Pause,
  Trash2,
  RefreshCw,
  Shield,
  Lock,
} from 'lucide-react';
import { useQueryClient } from '@tanstack/react-query';
import {
  useListNodes,
  useDrainNode,
  useActivateNode,
  useDeleteNode,
  getListNodesQueryKey,
} from '../../api/generated/nodes/nodes';
import { useListApplications } from '../../api/generated/applications/applications';
import { useListDomains } from '../../api/generated/domains/domains';
import { useModalActions } from '../../shared/stores/useModalStore';
import { useToastActions } from '../../shared/stores/useToastStore';

export function NodesView() {
  const queryClient = useQueryClient();
  const { data: nodesData } = useListNodes();
  const { data: appsData } = useListApplications();
  const { data: domainsData } = useListDomains();
  const nodes = Array.isArray(nodesData) ? nodesData : [];
  const apps = Array.isArray(appsData) ? appsData : [];
  const domains = Array.isArray(domainsData) ? domainsData : [];

  const { setShowAddNodeModal } = useModalActions();
  const { addToast } = useToastActions();

  const drainNodeMutation = useDrainNode();
  const activateNodeMutation = useActivateNode();
  const deleteNodeMutation = useDeleteNode();

  const [loadingNodeId, setLoadingNodeId] = useState<string | null>(null);

  const onDrain = async (nodeId: string, nodeName: string) => {
    if (!window.confirm(`Are you sure you want to drain node "${nodeName}"? Existing Durable Object leases will safely migrate to other fleet nodes.`)) {
      return;
    }
    setLoadingNodeId(nodeId);
    try {
      await drainNodeMutation.mutateAsync({ id: nodeId });
      await queryClient.invalidateQueries({ queryKey: getListNodesQueryKey() });
      addToast({
        title: 'Node Drained',
        description: `Node "${nodeName}" drained successfully.`,
        variant: 'info',
      });
    } catch (err: unknown) {
      console.error('Failed to drain node', err);
      addToast({
        title: 'Drain Failed',
        description: 'Failed to drain node',
        variant: 'error',
      });
    } finally {
      setLoadingNodeId(null);
    }
  };

  const onActivate = async (nodeId: string) => {
    setLoadingNodeId(nodeId);
    try {
      await activateNodeMutation.mutateAsync({ id: nodeId });
      await queryClient.invalidateQueries({ queryKey: getListNodesQueryKey() });
      addToast({
        title: 'Node Activated',
        description: 'Node activated successfully.',
        variant: 'success',
      });
    } catch (err: unknown) {
      console.error('Failed to activate node', err);
      addToast({
        title: 'Activation Failed',
        description: 'Failed to activate node',
        variant: 'error',
      });
    } finally {
      setLoadingNodeId(null);
    }
  };

  const onDelete = async (nodeId: string, nodeName: string) => {
    if (!window.confirm(`Deregister node "${nodeName}" from the fleet? This will remove its upstream proxy routes.`)) {
      return;
    }
    setLoadingNodeId(nodeId);
    try {
      await deleteNodeMutation.mutateAsync({ id: nodeId });
      await queryClient.invalidateQueries({ queryKey: getListNodesQueryKey() });
      addToast({
        title: 'Node Deregistered',
        description: `Node "${nodeName}" removed from fleet.`,
        variant: 'info',
      });
    } catch (err: unknown) {
      console.error('Failed to delete node', err);
      addToast({
        title: 'Deregister Failed',
        description: 'Failed to deregister node',
        variant: 'error',
      });
    } finally {
      setLoadingNodeId(null);
    }
  };

  return (
    <div className="space-y-6">
      {/* Fleet Overview Stats */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-5">
        <div className="p-5 rounded-2xl border border-zinc-800/80 bg-zinc-900/30">
          <div className="flex items-center justify-between text-zinc-400 text-sm mb-2">
            <span>Fleet Nodes</span>
            <Server className="w-4 h-4 text-emerald-400" />
          </div>
          <div className="text-3xl font-bold">{nodes.length}</div>
        </div>

        <div className="p-5 rounded-2xl border border-zinc-800/80 bg-zinc-900/30">
          <div className="flex items-center justify-between text-zinc-400 text-sm mb-2">
            <span>Active Workers</span>
            <Radio className="w-4 h-4 text-emerald-400" />
          </div>
          <div className="text-3xl font-bold">{apps.filter(a => a.status === 'running').length}</div>
        </div>

        <div className="p-5 rounded-2xl border border-zinc-800/80 bg-zinc-900/30">
          <div className="flex items-center justify-between text-zinc-400 text-sm mb-2">
            <span>Traefik SSL Routes</span>
            <ShieldCheck className="w-4 h-4 text-emerald-400" />
          </div>
          <div className="text-3xl font-bold">{domains.length}</div>
        </div>
      </div>

      {/* Node Instances Header & List */}
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <div>
            <h3 className="text-sm font-semibold text-zinc-300 uppercase tracking-wider">
              Bare-Metal Node Instances
            </h3>
            <p className="text-xs text-zinc-500 mt-0.5">
              Distributed celld runtime daemons powering Workers & Durable Objects
            </p>
          </div>
          <button
            type="button"
            onClick={() => setShowAddNodeModal(true)}
            className="flex items-center gap-2 px-3.5 py-1.5 rounded-lg bg-emerald-500 hover:bg-emerald-600 text-zinc-950 text-xs font-semibold transition shadow-sm"
          >
            <Plus className="w-3.5 h-3.5" />
            <span>Add Fleet Node</span>
          </button>
        </div>

        {nodes.length === 0 ? (
          <div className="p-10 rounded-2xl border border-zinc-800/80 bg-zinc-900/20 text-center space-y-4">
            <div className="w-12 h-12 rounded-xl bg-zinc-800/60 border border-zinc-700/60 flex items-center justify-center mx-auto text-zinc-400">
              <Server className="w-6 h-6" />
            </div>
            <div className="max-w-md mx-auto">
              <h4 className="font-semibold text-base text-zinc-200">No Fleet Nodes Connected</h4>
              <p className="text-xs text-zinc-400 mt-1">
                You haven't attached any VPS or bare-metal servers yet. Add a node to start running distributed edge workers with automated Traefik routing.
              </p>
            </div>
            <button
              type="button"
              onClick={() => setShowAddNodeModal(true)}
              className="inline-flex items-center gap-2 px-4 py-2 rounded-lg bg-emerald-500 hover:bg-emerald-600 text-zinc-950 text-xs font-bold transition shadow-sm"
            >
              <Plus className="w-4 h-4" />
              <span>Add Your First Node</span>
            </button>
          </div>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            {nodes.map(node => {
              const isLoading = loadingNodeId === node.id;
              const isDraining = node.status === 'draining';
              const isOffline = node.status === 'offline';
              const isActive = node.status === 'active';
              const isMainNode = Boolean(node.isProtected || node.name === 'worker-node-01');

              return (
                <div
                  key={node.id}
                  className="p-5 rounded-xl border border-zinc-800/80 bg-zinc-900/20 space-y-4 hover:border-zinc-700 transition relative"
                >
                  <div className="flex items-start justify-between">
                    <div>
                      <div className="flex items-center gap-2">
                        <h4 className="font-semibold text-base text-zinc-100">{node.name}</h4>
                        <span
                          className={`text-[10px] px-2 py-0.5 rounded-full font-bold uppercase tracking-wider ${
                            isActive
                              ? 'bg-emerald-950/80 text-emerald-400 border border-emerald-800/80'
                              : isDraining
                              ? 'bg-amber-950/80 text-amber-400 border border-amber-800/80'
                              : 'bg-red-950/80 text-red-400 border border-red-800/80'
                          }`}
                        >
                          {node.status}
                        </span>
                        {isMainNode && (
                          <span className="text-[10px] px-2 py-0.5 rounded-full font-bold uppercase tracking-wider bg-sky-950/80 text-sky-400 border border-sky-800/80 flex items-center gap-1">
                            <Shield className="w-2.5 h-2.5" />
                            <span>Main Node</span>
                          </span>
                        )}
                      </div>
                      <p className="text-xs text-zinc-400 font-mono mt-0.5">{node.ipAddress}</p>
                    </div>
                    <span className="text-xs font-mono text-zinc-400 bg-zinc-800/80 px-2 py-1 rounded border border-zinc-700/50">
                      celld v{node.celldVersion}
                    </span>
                  </div>

                  <div className="grid grid-cols-2 gap-2 text-xs text-zinc-400">
                    <div className="p-2.5 rounded bg-zinc-950/40 border border-zinc-900 flex items-center gap-2">
                      <Radio className="w-3.5 h-3.5 text-zinc-500" />
                      <span>Worker Port: {node.workerPort}</span>
                    </div>
                    <div className="p-2.5 rounded bg-zinc-950/40 border border-zinc-900 flex items-center gap-2">
                      <HardDrive className="w-3.5 h-3.5 text-zinc-500" />
                      <span>Peer Port: {node.internalPort}</span>
                    </div>
                  </div>

                  {/* Node Management Actions */}
                  <div className="flex items-center justify-between pt-2 border-t border-zinc-850">
                    {isMainNode ? (
                      <div className="flex items-center justify-between w-full">
                        <div className="flex items-center gap-1.5 text-xs text-zinc-400 font-medium py-1">
                          <Lock className="w-3.5 h-3.5 text-sky-400" />
                          <span>Protected Main Node (Cannot be drained or removed)</span>
                        </div>
                        <span className="text-[10px] font-mono text-zinc-500 uppercase">Core Cluster Host</span>
                      </div>
                    ) : (
                      <>
                        <div className="flex items-center gap-2">
                          {isActive && (
                            <button
                              type="button"
                              disabled={isLoading}
                              onClick={() => onDrain(node.id, node.name)}
                              className="flex items-center gap-1.5 px-2.5 py-1 rounded-md text-xs font-medium border border-amber-800/60 bg-amber-950/30 text-amber-400 hover:bg-amber-900/40 transition disabled:opacity-50"
                              title="Drain traffic from this node"
                            >
                              {isLoading ? (
                                <RefreshCw className="w-3 h-3 animate-spin" />
                              ) : (
                                <Pause className="w-3 h-3" />
                              )}
                              <span>Drain</span>
                            </button>
                          )}

                          {isDraining && (
                            <button
                              type="button"
                              disabled={isLoading}
                              onClick={() => onActivate(node.id)}
                              className="flex items-center gap-1.5 px-2.5 py-1 rounded-md text-xs font-medium border border-emerald-800/60 bg-emerald-950/30 text-emerald-400 hover:bg-emerald-900/40 transition disabled:opacity-50"
                              title="Reactivate node to accept traffic"
                            >
                              {isLoading ? (
                                <RefreshCw className="w-3 h-3 animate-spin" />
                              ) : (
                                <Play className="w-3 h-3" />
                              )}
                              <span>Activate</span>
                            </button>
                          )}

                          {isOffline && (
                            <button
                              type="button"
                              disabled={isLoading}
                              onClick={() => onActivate(node.id)}
                              className="flex items-center gap-1.5 px-2.5 py-1 rounded-md text-xs font-medium border border-emerald-800/60 bg-emerald-950/30 text-emerald-400 hover:bg-emerald-900/40 transition disabled:opacity-50"
                              title="Restore node to active state"
                            >
                              {isLoading ? (
                                <RefreshCw className="w-3 h-3 animate-spin" />
                              ) : (
                                <Play className="w-3 h-3" />
                              )}
                              <span>Activate</span>
                            </button>
                          )}
                        </div>

                        <button
                          type="button"
                          disabled={isLoading}
                          onClick={() => onDelete(node.id, node.name)}
                          className="flex items-center gap-1 px-2 py-1 rounded-md text-xs text-zinc-500 hover:text-red-400 hover:bg-red-950/30 transition disabled:opacity-50"
                          title="Deregister node"
                        >
                          <Trash2 className="w-3.5 h-3.5" />
                          <span>Remove</span>
                        </button>
                      </>
                    )}
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
}

