import React, { useState } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { useModalStore, useModalActions } from '../../shared/stores/useModalStore';
import {
  useUpgradeCelldDaemon,
  getGetRuntimeStatusQueryKey,
} from '../../api/generated/runtime/runtime';
import { useToastActions } from '../../shared/stores/useToastStore';

export function UpgradeModal() {
  const showUpgradeModal = useModalStore((state) => state.showUpgradeModal);
  const { setShowUpgradeModal } = useModalActions();
  const upgradeCelldMutation = useUpgradeCelldDaemon();
  const queryClient = useQueryClient();
  const { addToast } = useToastActions();

  const [targetVersion, setTargetVersion] = useState('0.6.1');
  const [isSubmitting, setIsSubmitting] = useState(false);

  if (!showUpgradeModal) return null;

  const onSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!targetVersion.trim()) return;

    setIsSubmitting(true);
    try {
      await upgradeCelldMutation.mutateAsync({
        data: { targetVersion: targetVersion.trim() },
      });
      await queryClient.invalidateQueries({ queryKey: getGetRuntimeStatusQueryKey() });
      setShowUpgradeModal(false);
      addToast({
        title: 'Upgrade Initiated',
        description: `Rolling upgrade to v${targetVersion} started across fleet.`,
        variant: 'info',
      });
    } catch (err: unknown) {
      const errorMsg =
        (err as { response?: { data?: { error?: string } }; message?: string })?.response?.data
          ?.error ||
        (err as { message?: string })?.message ||
        'Failed to start upgrade';
      addToast({
        title: 'Upgrade Failed',
        description: errorMsg,
        variant: 'error',
      });
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div className="fixed inset-0 bg-black/60 backdrop-blur-sm flex items-center justify-center p-4 z-50">
      <div className="bg-zinc-900 border border-zinc-800 rounded-2xl max-w-md w-full p-6 space-y-5 shadow-2xl">
        <div>
          <h3 className="text-lg font-bold">Rolling Upgrade celld Daemon</h3>
          <p className="text-xs text-zinc-400 mt-1">
            Upgrades the celld runtime container one node at a time with zero dropped requests.
          </p>
        </div>

        <div className="bg-emerald-500/10 border border-emerald-500/20 rounded-xl p-3 flex items-start gap-2.5">
          <div className="w-2 h-2 rounded-full bg-emerald-400 mt-1.5 shrink-0" />
          <div className="text-xs text-zinc-300">
            <span className="font-semibold text-emerald-400">Bucket Durability Active (Garage S3)</span>
            <p className="text-zinc-400 mt-0.5">
              celld v0.6.1 supports zero-downtime rolling node updates when backed by bucket storage durability.
            </p>
          </div>
        </div>

        <div className="bg-sky-500/10 border border-sky-500/20 rounded-xl p-3 flex items-start gap-2.5">
          <div className="w-2 h-2 rounded-full bg-sky-400 mt-1.5 shrink-0" />
          <div className="text-xs text-zinc-300">
            <span className="font-semibold text-sky-400">LTX Epoch Retention & GC (celld v0.6.1)</span>
            <p className="text-zinc-400 mt-0.5">
              Set <code className="text-sky-300 font-mono">CELLD_LTX_RETENTION_SECS</code> to automatically prune superseded epochs from bucket storage. Run <code className="text-sky-300 font-mono">celld cell gc --dry-run</code> on nodes to preview reclaimable storage.
            </p>
          </div>
        </div>

        <form onSubmit={onSubmit} className="space-y-4">
          <div>
            <label className="text-xs font-medium text-zinc-400 mb-1 block">Target Version</label>
            <input
              type="text"
              required
              placeholder="0.6.1"
              value={targetVersion}
              onChange={(e) => setTargetVersion(e.target.value)}
              className="w-full bg-zinc-950 border border-zinc-800 rounded-lg px-3 py-2 text-sm text-zinc-100 focus:outline-none focus:border-emerald-500 font-mono"
            />
          </div>

          <div className="flex justify-end gap-3 pt-2">
            <button
              type="button"
              onClick={() => setShowUpgradeModal(false)}
              className="px-4 py-2 rounded-lg border border-zinc-800 text-sm font-medium hover:bg-zinc-800 transition"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={isSubmitting}
              className={`px-4 py-2 rounded-lg bg-emerald-500 hover:bg-emerald-600 text-zinc-950 text-sm font-semibold transition ${
                isSubmitting ? 'opacity-50 cursor-not-allowed' : ''
              }`}
            >
              {isSubmitting ? 'Starting...' : 'Start Rolling Upgrade'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
