import { ShieldCheck, Plus, Globe } from 'lucide-react';
import { useDashboard } from '../../context/DashboardContext';

export function DomainsView() {
  const { domains, setShowNewDomainModal } = useDashboard();

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h3 className="text-sm font-semibold text-zinc-400 uppercase tracking-wider">
            Traefik Ingress Routes & Domains
          </h3>
          <p className="text-xs text-zinc-500">
            Automatic TLS and SNI-based hostname routing configured via Traefik.
          </p>
        </div>
        <button
          type="button"
          onClick={() => setShowNewDomainModal(true)}
          className="flex items-center gap-2 px-3.5 py-1.5 rounded-lg bg-emerald-500 hover:bg-emerald-600 text-zinc-950 text-xs font-bold transition shadow-sm"
        >
          <Plus className="w-3.5 h-3.5" />
          Add Domain
        </button>
      </div>

      {domains.length === 0 ? (
        <div className="p-12 rounded-2xl border border-dashed border-zinc-800 bg-zinc-950/40 text-center space-y-3">
          <div className="w-12 h-12 rounded-full bg-zinc-900 border border-zinc-800 mx-auto flex items-center justify-center text-zinc-500">
            <Globe className="w-6 h-6" />
          </div>
          <div className="space-y-1">
            <h4 className="text-base font-bold text-zinc-200">No Custom Domains Configured</h4>
            <p className="text-xs text-zinc-500 max-w-md mx-auto">
              Attach a custom hostname to any of your running Cloudflare Workers to route production web traffic.
            </p>
          </div>
          <button
            type="button"
            onClick={() => setShowNewDomainModal(true)}
            className="inline-flex items-center gap-1.5 px-4 py-2 rounded-lg bg-emerald-500 hover:bg-emerald-600 text-zinc-950 text-xs font-bold transition"
          >
            <Plus className="w-3.5 h-3.5" />
            Add First Domain
          </button>
        </div>
      ) : (
        <div className="grid grid-cols-1 gap-3">
          {domains.map(dom => (
            <div
              key={dom.id}
              className="p-5 rounded-xl border border-zinc-800/80 bg-zinc-900/20 flex items-center justify-between hover:border-zinc-700 transition"
            >
              <div className="space-y-1">
                <div className="flex items-center gap-3">
                  <h4 className="font-mono text-base text-emerald-400 font-semibold">{dom.hostname}</h4>
                  <span className="text-xs bg-zinc-800 px-2 py-0.5 rounded text-zinc-400">{dom.pathPrefix}</span>
                </div>
                <p className="text-xs text-zinc-500">Traefik Ingress Route</p>
              </div>

              <div className="flex items-center gap-2 text-xs text-emerald-400 bg-emerald-950/40 border border-emerald-900 px-3 py-1 rounded-full">
                <ShieldCheck className="w-4 h-4" />
                Let's Encrypt Active
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
