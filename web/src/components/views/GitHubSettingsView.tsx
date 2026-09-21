import { useState, useEffect } from 'react';
import {
  GitBranch,
  ShieldCheck,
  ExternalLink,
  Copy,
  Check,
  RefreshCw,
  Trash2,
  Plus,
  Lock,
  Globe,
  AlertCircle,
  Key,
} from 'lucide-react';
import { useDashboard } from '../../context/DashboardContext';

interface GitHubSettings {
  id: string;
  appId: string;
  appName: string;
  clientId: string;
  installationId: string;
  isConfigured: boolean;
  hasPrivateKey: boolean;
  hasWebhookSecret: boolean;
  webhookUrl: string;
}

interface GitHubRepoItem {
  id: number;
  name: string;
  fullName: string;
  defaultBranch: string;
  private: boolean;
  cloneUrl: string;
  htmlUrl: string;
}

export function GitHubSettingsView() {
  const { setShowNewAppModal } = useDashboard();

  const [settings, setSettings] = useState<GitHubSettings | null>(null);
  const [repositories, setRepositories] = useState<GitHubRepoItem[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [isLoadingRepos, setIsLoadingRepos] = useState(false);
  const [manifestData, setManifestData] = useState<any>(null);
  const [copiedWebhook, setCopiedWebhook] = useState(false);

  // Manual configuration form state
  const [showManualModal, setShowManualModal] = useState(false);
  const [appId, setAppId] = useState('');
  const [appName, setAppName] = useState('');
  const [clientId, setClientId] = useState('');
  const [clientSecret, setClientSecret] = useState('');
  const [webhookSecret, setWebhookSecret] = useState('');
  const [privateKey, setPrivateKey] = useState('');
  const [installationId, setInstallationId] = useState('');
  const [isSaving, setIsSaving] = useState(false);

  // Manifest callback conversion code
  const [exchangeCode, setExchangeCode] = useState('');
  const [isExchanging, setIsExchanging] = useState(false);
  const [exchangeError, setExchangeError] = useState<string | null>(null);

  const fetchSettings = async () => {
    setIsLoading(true);
    try {
      const res = await fetch('/api/v1/github/settings');
      if (res.ok) {
        const data = await res.json();
        setSettings(data);
        if (data.isConfigured) {
          fetchRepositories();
        }
      }
    } catch (_) {
      // ignore
    } finally {
      setIsLoading(false);
    }
  };

  const fetchManifest = async () => {
    try {
      const res = await fetch('/api/v1/github/manifest');
      if (res.ok) {
        const data = await res.json();
        setManifestData(data);
      }
    } catch (_) {
      // ignore
    }
  };

  const fetchRepositories = async () => {
    setIsLoadingRepos(true);
    try {
      const res = await fetch('/api/v1/github/repositories');
      if (res.ok) {
        const data = await res.json();
        setRepositories(Array.isArray(data) ? data : []);
      }
    } catch (_) {
      setRepositories([]);
    } finally {
      setIsLoadingRepos(false);
    }
  };

  useEffect(() => {
    fetchSettings();
    fetchManifest();
  }, []);

  const handleCopyWebhook = () => {
    if (!settings?.webhookUrl) return;
    navigator.clipboard.writeText(settings.webhookUrl);
    setCopiedWebhook(true);
    setTimeout(() => setCopiedWebhook(false), 2000);
  };

  const handleSaveManual = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsSaving(true);
    try {
      const res = await fetch('/api/v1/github/settings', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          appId,
          appName,
          clientId,
          clientSecret,
          webhookSecret,
          privateKey,
          installationId,
        }),
      });
      if (res.ok) {
        setShowManualModal(false);
        fetchSettings();
      }
    } finally {
      setIsSaving(false);
    }
  };

  const handleExchangeCode = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!exchangeCode.trim()) return;
    setIsExchanging(true);
    setExchangeError(null);
    try {
      const res = await fetch('/api/v1/github/manifest/exchange', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ code: exchangeCode.trim() }),
      });
      if (res.ok) {
        setExchangeCode('');
        fetchSettings();
      } else {
        const errText = await res.text();
        setExchangeError(errText || 'Failed to complete GitHub App setup');
      }
    } catch (err: any) {
      setExchangeError(err.message || 'Network error');
    } finally {
      setIsExchanging(false);
    }
  };

  const handleDisconnect = async () => {
    if (!confirm('Are you sure you want to disconnect this GitHub App? Webhooks and automatic deployments will be disabled.')) {
      return;
    }
    await fetch('/api/v1/github/settings', { method: 'DELETE' });
    setSettings(null);
    setRepositories([]);
    fetchSettings();
  };

  return (
    <div className="space-y-8 max-w-5xl">
      {/* Header Info */}
      <div>
        <div className="flex items-center gap-3 mb-1">
          <h3 className="text-lg font-bold text-zinc-100 flex items-center gap-2">
            <GitBranch className="w-5 h-5 text-purple-400" />
            GitHub App & Push-to-Deploy CI/CD
          </h3>
          {settings?.isConfigured && (
            <span className="text-[11px] px-2.5 py-0.5 rounded-full bg-emerald-950/80 text-emerald-400 border border-emerald-800 font-semibold flex items-center gap-1.5">
              <span className="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse" />
              Connected
            </span>
          )}
        </div>
        <p className="text-xs text-zinc-400">
          Connect your GitHub account or organization via a Dokploy-style GitHub App to automatically build, version, and deploy Cloudflare Workers on every <code className="text-emerald-400 font-mono">git push</code>.
        </p>
      </div>

      {isLoading ? (
        <div className="p-12 text-center text-zinc-500 flex items-center justify-center gap-2">
          <RefreshCw className="w-4 h-4 animate-spin text-emerald-400" />
          <span>Checking GitHub App status...</span>
        </div>
      ) : !settings?.isConfigured ? (
        /* Setup / Connection Options */
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
          {/* 1-Click Manifest Flow */}
          <div className="p-6 rounded-2xl border border-zinc-800/80 bg-zinc-900/30 flex flex-col justify-between space-y-5">
            <div className="space-y-3">
              <div className="w-10 h-10 rounded-xl bg-purple-950/60 border border-purple-800/80 flex items-center justify-center text-purple-400">
                <GitBranch className="w-5 h-5" />
              </div>
              <div>
                <h4 className="font-bold text-base text-zinc-200">1-Click GitHub App (Recommended)</h4>
                <p className="text-xs text-zinc-400 mt-1 leading-relaxed">
                  Automatically create and configure a GitHub App with read-only repository permissions and webhook events configured.
                </p>
              </div>

              {manifestData && (
                <form
                  action="https://github.com/settings/apps/new"
                  method="post"
                  target="_blank"
                  className="pt-2"
                >
                  <input
                    type="hidden"
                    name="manifest"
                    value={JSON.stringify(manifestData)}
                  />
                  <button
                    type="submit"
                    className="w-full flex items-center justify-center gap-2 px-4 py-2.5 rounded-xl bg-purple-600 hover:bg-purple-500 text-white text-xs font-bold transition shadow-sm"
                  >
                    <ExternalLink className="w-4 h-4" />
                    Create GitHub App on GitHub
                  </button>
                </form>
              )}
            </div>

            {/* Complete Setup with Code */}
            <div className="pt-4 border-t border-zinc-800/80 space-y-3">
              <span className="text-[11px] font-semibold text-zinc-400 uppercase tracking-wider block">
                Complete Setup via Code
              </span>
              <form onSubmit={handleExchangeCode} className="space-y-2">
                <div className="flex gap-2">
                  <input
                    type="text"
                    placeholder="Enter callback code from GitHub..."
                    value={exchangeCode}
                    onChange={e => setExchangeCode(e.target.value)}
                    className="flex-1 bg-zinc-950 border border-zinc-800 rounded-lg px-3 py-1.5 text-xs text-zinc-200 focus:outline-none focus:border-purple-500 font-mono"
                  />
                  <button
                    type="submit"
                    disabled={isExchanging || !exchangeCode.trim()}
                    className="px-3 py-1.5 rounded-lg bg-zinc-800 hover:bg-zinc-700 disabled:opacity-50 text-zinc-200 text-xs font-semibold transition shrink-0"
                  >
                    {isExchanging ? <RefreshCw className="w-3.5 h-3.5 animate-spin" /> : 'Connect'}
                  </button>
                </div>
                {exchangeError && (
                  <p className="text-[11px] text-rose-400 flex items-center gap-1.5">
                    <AlertCircle className="w-3.5 h-3.5 shrink-0" />
                    {exchangeError}
                  </p>
                )}
              </form>
            </div>
          </div>

          {/* Manual Credentials or Token */}
          <div className="p-6 rounded-2xl border border-zinc-800/80 bg-zinc-900/30 flex flex-col justify-between space-y-5">
            <div className="space-y-3">
              <div className="w-10 h-10 rounded-xl bg-zinc-800/80 border border-zinc-700/80 flex items-center justify-center text-zinc-300">
                <Key className="w-5 h-5" />
              </div>
              <div>
                <h4 className="font-bold text-base text-zinc-200">Manual Setup / Access Token</h4>
                <p className="text-xs text-zinc-400 mt-1 leading-relaxed">
                  Provide an existing GitHub App ID, Private Key, and Webhook Secret, or enter a Personal Access Token (PAT) for quick testing.
                </p>
              </div>
            </div>

            <div className="pt-4 border-t border-zinc-800/80">
              <button
                type="button"
                onClick={() => setShowManualModal(true)}
                className="w-full flex items-center justify-center gap-2 px-4 py-2.5 rounded-xl border border-zinc-700 bg-zinc-900 hover:bg-zinc-800 text-xs font-semibold transition"
              >
                Enter App Credentials Manually
              </button>
            </div>
          </div>
        </div>
      ) : (
        /* Connected GitHub App View */
        <div className="space-y-6">
          {/* Status & Configuration Card */}
          <div className="p-6 rounded-2xl border border-zinc-800/80 bg-zinc-900/30 space-y-5">
            <div className="flex items-center justify-between flex-wrap gap-4">
              <div>
                <h4 className="text-base font-bold text-zinc-100 flex items-center gap-2">
                  <span>{settings.appName || 'Connected GitHub App'}</span>
                  {settings.appId && (
                    <span className="text-xs font-mono text-zinc-400 bg-zinc-800/80 px-2 py-0.5 rounded">
                      ID: {settings.appId}
                    </span>
                  )}
                </h4>
                <p className="text-xs text-zinc-400 mt-0.5">
                  Receiving push events and deploying matching worker applications automatically.
                </p>
              </div>

              <div className="flex items-center gap-3">
                {settings.appName && (
                  <a
                    href={`https://github.com/apps/${settings.appName}/installations/new`}
                    target="_blank"
                    rel="noreferrer"
                    className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg border border-purple-500/50 bg-purple-950/30 text-purple-300 hover:bg-purple-900/40 text-xs font-semibold transition"
                  >
                    <ExternalLink className="w-3.5 h-3.5" />
                    Install on More Repos
                  </a>
                )}
                <button
                  type="button"
                  onClick={handleDisconnect}
                  className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg border border-rose-900/60 bg-rose-950/30 text-rose-400 hover:bg-rose-900/40 text-xs font-semibold transition"
                >
                  <Trash2 className="w-3.5 h-3.5" />
                  Disconnect
                </button>
              </div>
            </div>

            {/* Webhook Endpoint Info */}
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4 pt-2">
              <div className="p-3.5 rounded-xl bg-zinc-950/60 border border-zinc-800/80 space-y-1">
                <span className="text-[11px] font-semibold text-zinc-400 uppercase tracking-wider block">
                  Webhook Ingress URL
                </span>
                <div className="flex items-center justify-between gap-2">
                  <code className="text-xs text-emerald-400 font-mono truncate">
                    {settings.webhookUrl}
                  </code>
                  <button
                    type="button"
                    onClick={handleCopyWebhook}
                    className="text-zinc-400 hover:text-zinc-200 p-1 transition shrink-0"
                    title="Copy Webhook URL"
                  >
                    {copiedWebhook ? <Check className="w-3.5 h-3.5 text-emerald-400" /> : <Copy className="w-3.5 h-3.5" />}
                  </button>
                </div>
              </div>

              <div className="p-3.5 rounded-xl bg-zinc-950/60 border border-zinc-800/80 space-y-1">
                <span className="text-[11px] font-semibold text-zinc-400 uppercase tracking-wider block">
                  Security & HMAC Verification
                </span>
                <div className="flex items-center gap-2">
                  <span className="text-xs text-emerald-400 flex items-center gap-1 font-semibold">
                    <ShieldCheck className="w-3.5 h-3.5" />
                    {settings.hasWebhookSecret ? 'HMAC-SHA256 Secret Configured' : 'No Secret (Permissive)'}
                  </span>
                  {settings.hasPrivateKey && (
                    <span className="text-[10px] bg-zinc-800 text-zinc-400 px-2 py-0.5 rounded font-mono">
                      RSA Key Active
                    </span>
                  )}
                </div>
              </div>
            </div>
          </div>

          {/* Accessible Repositories List */}
          <div className="space-y-3">
            <div className="flex items-center justify-between">
              <div>
                <h4 className="text-sm font-semibold text-zinc-300 uppercase tracking-wider">
                  Accessible Repositories ({repositories.length})
                </h4>
                <p className="text-xs text-zinc-500">
                  Repositories permitted by your GitHub App installation. You can deploy any of these as a Cloudflare Worker isolate.
                </p>
              </div>
              <button
                type="button"
                onClick={fetchRepositories}
                disabled={isLoadingRepos}
                className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg border border-zinc-800 bg-zinc-900 hover:bg-zinc-800 text-xs font-semibold text-zinc-300 transition"
              >
                <RefreshCw className={`w-3.5 h-3.5 ${isLoadingRepos ? 'animate-spin text-emerald-400' : ''}`} />
                Refresh
              </button>
            </div>

            {isLoadingRepos ? (
              <div className="p-8 text-center text-zinc-500 flex items-center justify-center gap-2">
                <RefreshCw className="w-4 h-4 animate-spin text-emerald-400" />
                <span>Fetching repositories from GitHub...</span>
              </div>
            ) : repositories.length === 0 ? (
              <div className="p-8 rounded-xl border border-dashed border-zinc-800 text-center space-y-2">
                <p className="text-xs text-zinc-500">
                  No repositories found. Ensure your GitHub App is installed on an account with repository access.
                </p>
                {settings.appName && (
                  <a
                    href={`https://github.com/apps/${settings.appName}/installations/new`}
                    target="_blank"
                    rel="noreferrer"
                    className="inline-flex items-center gap-1 text-xs text-purple-400 hover:underline"
                  >
                    Install app on repositories <ExternalLink className="w-3 h-3" />
                  </a>
                )}
              </div>
            ) : (
              <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
                {repositories.map(repo => (
                  <div
                    key={repo.id}
                    className="p-4 rounded-xl border border-zinc-800/80 bg-zinc-900/20 hover:border-zinc-700 transition flex items-center justify-between"
                  >
                    <div className="space-y-1 min-w-0 pr-3">
                      <div className="flex items-center gap-2">
                        <span className="font-semibold text-xs text-zinc-200 truncate">{repo.fullName}</span>
                        {repo.private ? (
                          <span className="text-[10px] px-1.5 py-0.2 rounded bg-zinc-800 text-zinc-400 flex items-center gap-1">
                            <Lock className="w-2.5 h-2.5" /> Private
                          </span>
                        ) : (
                          <span className="text-[10px] px-1.5 py-0.2 rounded bg-zinc-800 text-zinc-400 flex items-center gap-1">
                            <Globe className="w-2.5 h-2.5" /> Public
                          </span>
                        )}
                      </div>
                      <div className="text-[11px] text-zinc-500 font-mono">
                        Default branch: <span className="text-emerald-400">{repo.defaultBranch || 'main'}</span>
                      </div>
                    </div>

                    <button
                      type="button"
                      onClick={() => setShowNewAppModal(true)}
                      className="px-2.5 py-1.5 rounded-lg bg-emerald-500/10 hover:bg-emerald-500/20 border border-emerald-500/30 text-emerald-400 text-xs font-semibold transition shrink-0 flex items-center gap-1"
                    >
                      <Plus className="w-3.5 h-3.5" />
                      Create Worker
                    </button>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>
      )}

      {/* Manual Configuration Modal */}
      {showManualModal && (
        <div className="fixed inset-0 bg-black/60 backdrop-blur-sm flex items-center justify-center p-4 z-50">
          <div className="bg-zinc-900 border border-zinc-800 rounded-2xl max-w-lg w-full p-6 space-y-4 shadow-2xl overflow-y-auto max-h-[90vh]">
            <h3 className="text-base font-bold text-zinc-100">Manual GitHub Configuration</h3>
            <p className="text-xs text-zinc-400">
              Enter GitHub App credentials or a Personal Access Token (for quick local development).
            </p>

            <form onSubmit={handleSaveManual} className="space-y-3">
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="text-xs font-medium text-zinc-400 mb-1 block">App Name</label>
                  <input
                    type="text"
                    placeholder="cubit-fleet"
                    value={appName}
                    onChange={e => setAppName(e.target.value)}
                    className="w-full bg-zinc-950 border border-zinc-800 rounded-lg px-3 py-1.5 text-xs text-zinc-200 focus:outline-none focus:border-emerald-500"
                  />
                </div>
                <div>
                  <label className="text-xs font-medium text-zinc-400 mb-1 block">Client ID</label>
                  <input
                    type="text"
                    placeholder="Iv1.xxxxxxxxxxxx"
                    value={clientId}
                    onChange={e => setClientId(e.target.value)}
                    className="w-full bg-zinc-950 border border-zinc-800 rounded-lg px-3 py-1.5 text-xs text-zinc-200 focus:outline-none focus:border-emerald-500 font-mono"
                  />
                </div>
              </div>

              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="text-xs font-medium text-zinc-400 mb-1 block">App ID</label>
                  <input
                    type="text"
                    placeholder="123456"
                    value={appId}
                    onChange={e => setAppId(e.target.value)}
                    className="w-full bg-zinc-950 border border-zinc-800 rounded-lg px-3 py-1.5 text-xs text-zinc-200 focus:outline-none focus:border-emerald-500 font-mono"
                  />
                </div>
                <div>
                  <label className="text-xs font-medium text-zinc-400 mb-1 block">Installation ID</label>
                  <input
                    type="text"
                    placeholder="987654"
                    value={installationId}
                    onChange={e => setInstallationId(e.target.value)}
                    className="w-full bg-zinc-950 border border-zinc-800 rounded-lg px-3 py-1.5 text-xs text-zinc-200 focus:outline-none focus:border-emerald-500 font-mono"
                  />
                </div>
              </div>

              <div>
                <label className="text-xs font-medium text-zinc-400 mb-1 block">Webhook Secret</label>
                <input
                  type="text"
                  placeholder="secret-token-for-hmac-verification"
                  value={webhookSecret}
                  onChange={e => setWebhookSecret(e.target.value)}
                  className="w-full bg-zinc-950 border border-zinc-800 rounded-lg px-3 py-1.5 text-xs text-zinc-200 focus:outline-none focus:border-emerald-500 font-mono"
                />
              </div>

              <div>
                <label className="text-xs font-medium text-zinc-400 mb-1 block">Client Secret / PAT Token</label>
                <input
                  type="password"
                  placeholder="ghp_... or client secret"
                  value={clientSecret}
                  onChange={e => setClientSecret(e.target.value)}
                  className="w-full bg-zinc-950 border border-zinc-800 rounded-lg px-3 py-1.5 text-xs text-zinc-200 focus:outline-none focus:border-emerald-500 font-mono"
                />
              </div>

              <div>
                <label className="text-xs font-medium text-zinc-400 mb-1 block">
                  GitHub App Private Key (PEM format)
                </label>
                <textarea
                  rows={4}
                  placeholder="-----BEGIN RSA PRIVATE KEY-----&#10;...&#10;-----END RSA PRIVATE KEY-----"
                  value={privateKey}
                  onChange={e => setPrivateKey(e.target.value)}
                  className="w-full bg-zinc-950 border border-zinc-800 rounded-lg p-2.5 text-[11px] font-mono text-zinc-300 focus:outline-none focus:border-emerald-500 resize-none leading-relaxed"
                />
              </div>

              <div className="flex justify-end gap-3 pt-2">
                <button
                  type="button"
                  onClick={() => setShowManualModal(false)}
                  className="px-3.5 py-1.5 rounded-lg border border-zinc-800 text-xs font-semibold hover:bg-zinc-800 transition"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  disabled={isSaving}
                  className="px-4 py-1.5 rounded-lg bg-emerald-500 hover:bg-emerald-600 text-zinc-950 text-xs font-bold transition"
                >
                  {isSaving ? 'Saving...' : 'Save Configuration'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
