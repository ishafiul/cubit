import React, { useState } from 'react';
import { Globe, Plus, Clock, ShieldCheck } from 'lucide-react';
import type { Application } from '../../../../api/model';
import { useListDomains, useCreateDomain } from '../../../../api/generated/domains/domains';
import { useToastActions } from '../../../../shared/stores/useToastStore';
import { domainSchema, validateSchema } from '../../../../shared/utils/validation';
import { isOk } from '../../../../shared/utils/result';

export function TriggersTab({ app }: { app: Application }) {
  const { data: domains, refetch: refetchDomains, isLoading } = useListDomains();
  const createDomainMutation = useCreateDomain();
  const { addToast } = useToastActions();

  const [hostname, setHostname] = useState('');
  const [validationError, setValidationError] = useState<string | null>(null);

  const appDomains = Array.isArray(domains)
    ? domains.filter((d) => d.applicationId === app.id)
    : [];

  const handleAddDomain = async (e: React.FormEvent) => {
    e.preventDefault();
    setValidationError(null);

    const validationResult = validateSchema(domainSchema, {
      hostname: hostname.trim(),
      pathPrefix: '/',
    });

    if (!isOk(validationResult)) {
      setValidationError(validationResult.error.issues[0]?.message || 'Invalid domain hostname');
      return;
    }

    try {
      await createDomainMutation.mutateAsync({
        data: {
          applicationId: app.id,
          hostname: validationResult.value.hostname,
          pathPrefix: '/',
        },
      });
      addToast({
        title: 'Domain Attached',
        description: `Successfully registered ${validationResult.value.hostname}`,
        variant: 'success',
      });
      setHostname('');
      refetchDomains();
    } catch (err: any) {
      addToast({
        title: 'Failed to Register Domain',
        description: err?.message || 'Could not register custom domain.',
        variant: 'error',
      });
    }
  };

  return (
    <div data-testid="tab-content-triggers" className="p-6 max-w-6xl mx-auto w-full space-y-6">
      {/* Custom Domains */}
      <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-6">
        <h3 className="text-sm font-bold text-zinc-100 flex items-center gap-2 mb-1">
          <Globe className="w-4 h-4 text-emerald-400" />
          Custom Domain & Ingress Routing
        </h3>
        <p className="text-xs text-zinc-400 mb-4">
          Route edge traffic through Traefik v3 directly into this Celld worker isolate
        </p>

        <form onSubmit={handleAddDomain} className="flex gap-2 max-w-lg mb-6">
          <input
            type="text"
            placeholder="api.example.com"
            value={hostname}
            onChange={(e) => setHostname(e.target.value)}
            className="flex-1 px-3 py-1.5 rounded-lg bg-zinc-950 border border-zinc-800 text-xs text-zinc-200 focus:outline-none focus:border-emerald-500 font-mono"
          />
          <button
            type="submit"
            className="flex items-center gap-1.5 px-3.5 py-1.5 rounded-lg bg-emerald-600 hover:bg-emerald-500 text-xs font-semibold text-white transition"
          >
            <Plus className="w-3.5 h-3.5" />
            Add Route
          </button>
        </form>

        {validationError && (
          <p className="text-xs text-rose-400 -mt-4 mb-4">{validationError}</p>
        )}

        {isLoading ? (
          <div className="text-xs text-zinc-500">Loading domains...</div>
        ) : appDomains.length === 0 ? (
          <div className="p-4 rounded-lg bg-zinc-950/60 border border-zinc-800/60 text-xs text-zinc-500">
            No custom domains configured. Worker is accessible via default cluster ingress.
          </div>
        ) : (
          <div className="space-y-2">
            {appDomains.map((domain) => (
              <div
                key={domain.id}
                className="flex items-center justify-between p-3 rounded-lg bg-zinc-950/60 border border-zinc-800/60"
              >
                <div className="flex items-center gap-2">
                  <Globe className="w-4 h-4 text-zinc-500" />
                  <span className="text-xs font-mono font-medium text-zinc-200">
                    {domain.hostname}
                  </span>
                  <span className="text-[10px] text-zinc-500 font-mono">
                    Path: {domain.pathPrefix || '/'}
                  </span>
                </div>
                <div className="flex items-center gap-2 text-xs">
                  <span className="flex items-center gap-1 text-emerald-400">
                    <ShieldCheck className="w-3.5 h-3.5" />
                    Active Ingress
                  </span>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Cron Triggers */}
      <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-6">
        <h3 className="text-sm font-bold text-zinc-100 flex items-center gap-2 mb-1">
          <Clock className="w-4 h-4 text-purple-400" />
          Scheduled Cron Triggers
        </h3>
        <p className="text-xs text-zinc-400 mb-4">
          Automated crons scheduled via Cloudflare Worker standard syntax
        </p>

        <div className="p-4 rounded-lg bg-zinc-950/60 border border-zinc-800/60 text-xs text-zinc-400">
          Configure crons in your <code className="font-mono text-emerald-400">wrangler.jsonc</code> triggers:
          <pre className="mt-2 p-2 rounded bg-zinc-900 font-mono text-[11px] text-zinc-300">
{`"triggers": {
  "crons": ["*/5 * * * *"]
}`}
          </pre>
        </div>
      </div>
    </div>
  );
}
