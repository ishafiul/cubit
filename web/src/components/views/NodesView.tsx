import { Server, Radio, ShieldCheck, HardDrive } from 'lucide-react';
import { useDashboard } from '../../context/DashboardContext';

export function NodesView() {
  const { nodes, apps, domains } = useDashboard();

  return (
    <div className="space-y-6">
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

      {/* Node Cards */}
      <div className="space-y-3">
        <h3 className="text-sm font-semibold text-zinc-400 uppercase tracking-wider">Bare-Metal Node Instances</h3>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {nodes.map(node => (
            <div key={node.id} className="p-5 rounded-xl border border-zinc-800/80 bg-zinc-900/20 space-y-4 hover:border-zinc-700 transition">
              <div className="flex items-start justify-between">
                <div>
                  <div className="flex items-center gap-2">
                    <h4 className="font-semibold text-base">{node.name}</h4>
                    <span className={`text-[10px] px-2 py-0.5 rounded-full font-bold uppercase ${
                      node.status === 'active' ? 'bg-emerald-950 text-emerald-400 border border-emerald-800' : 'bg-amber-950 text-amber-400 border border-amber-800'
                    }`}>
                      {node.status}
                    </span>
                  </div>
                  <p className="text-xs text-zinc-500 font-mono mt-0.5">{node.ipAddress}</p>
                </div>
                <span className="text-xs font-mono text-zinc-400 bg-zinc-800 px-2 py-1 rounded">
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
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
