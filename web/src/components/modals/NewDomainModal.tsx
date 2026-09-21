import React, { useState } from 'react';
import { useDashboard } from '../../context/DashboardContext';

export function NewDomainModal() {
  const { showNewDomainModal, setShowNewDomainModal, apps, handleCreateDomain } = useDashboard();
  const [selectedAppId, setSelectedAppId] = useState('');
  const [domainHost, setDomainHost] = useState('');

  if (!showNewDomainModal) return null;

  const onSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    await handleCreateDomain(selectedAppId, domainHost);
  };

  return (
    <div className="fixed inset-0 bg-black/60 backdrop-blur-sm flex items-center justify-center p-4 z-50">
      <div className="bg-zinc-900 border border-zinc-800 rounded-2xl max-w-md w-full p-6 space-y-5 shadow-2xl">
        <h3 className="text-lg font-bold">Add Custom Domain</h3>
        <form onSubmit={onSubmit} className="space-y-4">
          <div>
            <label className="text-xs font-medium text-zinc-400 mb-1 block">Target Application</label>
            <select
              required
              value={selectedAppId}
              onChange={e => setSelectedAppId(e.target.value)}
              className="w-full bg-zinc-950 border border-zinc-800 rounded-lg px-3 py-2 text-sm text-zinc-100 focus:outline-none focus:border-emerald-500"
            >
              <option value="">Select an application...</option>
              {apps.map(a => (
                <option key={a.id} value={a.id}>{a.name}</option>
              ))}
            </select>
          </div>

          <div>
            <label className="text-xs font-medium text-zinc-400 mb-1 block">Hostname</label>
            <input
              type="text"
              required
              placeholder="api.example.com"
              value={domainHost}
              onChange={e => setDomainHost(e.target.value)}
              className="w-full bg-zinc-950 border border-zinc-800 rounded-lg px-3 py-2 text-sm text-zinc-100 focus:outline-none focus:border-emerald-500 font-mono"
            />
          </div>

          <div className="flex justify-end gap-3 pt-2">
            <button
              type="button"
              onClick={() => setShowNewDomainModal(false)}
              className="px-4 py-2 rounded-lg border border-zinc-800 text-sm font-medium hover:bg-zinc-800 transition"
            >
              Cancel
            </button>
            <button
              type="submit"
              className="px-4 py-2 rounded-lg bg-emerald-500 hover:bg-emerald-600 text-zinc-950 text-sm font-semibold transition"
            >
              Create Domain Route
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
