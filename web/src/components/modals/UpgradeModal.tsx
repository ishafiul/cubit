import React, { useState } from 'react';
import { useDashboard } from '../../context/DashboardContext';

export function UpgradeModal() {
  const { showUpgradeModal, setShowUpgradeModal, handleUpgradeCelld } = useDashboard();
  const [targetVersion, setTargetVersion] = useState('0.3.0');

  if (!showUpgradeModal) return null;

  const onSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    await handleUpgradeCelld(targetVersion);
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

        <form onSubmit={onSubmit} className="space-y-4">
          <div>
            <label className="text-xs font-medium text-zinc-400 mb-1 block">Target Version</label>
            <input
              type="text"
              required
              placeholder="0.3.0"
              value={targetVersion}
              onChange={e => setTargetVersion(e.target.value)}
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
              className="px-4 py-2 rounded-lg bg-emerald-500 hover:bg-emerald-600 text-zinc-950 text-sm font-semibold transition"
            >
              Start Rolling Upgrade
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
