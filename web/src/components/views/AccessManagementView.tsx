import React, { useState, useEffect } from 'react';
import {
  Key,
  Shield,
  Plus,
  Trash2,
  Copy,
  Check,
  AlertTriangle,
  Lock,
  Unlock,
  Clock,
  AlertCircle,
  Loader2,
  X,
} from 'lucide-react';
import { customInstance } from '../../api/custom-instance';
import { useAuthStore } from '../../shared/stores/useAuthStore';
import { useToastActions } from '../../shared/stores/useToastStore';

export interface APITokenItem {
  id: string;
  userId: string;
  name: string;
  roleId: string;
  expiresAt?: string | null;
  lastUsedAt?: string | null;
  createdAt: string;
}

export interface RoleItem {
  id: string;
  name: string;
  description?: string;
  isSystem: boolean;
  permissions: string[];
  createdAt: string;
  updatedAt: string;
}

export function AccessManagementView() {
  const [activeTab, setActiveTab] = useState<'tokens' | 'roles'>('tokens');

  // Token management state
  const [tokens, setTokens] = useState<APITokenItem[]>([]);
  const [loadingTokens, setLoadingTokens] = useState(false);
  const [showGenerateModal, setShowGenerateModal] = useState(false);
  const [generatedPlainToken, setGeneratedPlainToken] = useState<string | null>(null);
  const [tokenName, setTokenName] = useState('');
  const [tokenRole, setTokenRole] = useState('developer');
  const [tokenExpiryDays, setTokenExpiryDays] = useState(30);
  const [generatingToken, setGeneratingToken] = useState(false);
  const [tokenError, setTokenError] = useState<string | null>(null);
  const [copiedToken, setCopiedToken] = useState(false);

  // Role management state
  const [roles, setRoles] = useState<RoleItem[]>([]);
  const [permissionsCatalog, setPermissionsCatalog] = useState<string[]>([]);
  const [loadingRoles, setLoadingRoles] = useState(false);
  const [showCreateRoleModal, setShowCreateRoleModal] = useState(false);
  const [roleName, setRoleName] = useState('');
  const [roleDescription, setRoleDescription] = useState('');
  const [selectedPermissions, setSelectedPermissions] = useState<string[]>([]);
  const [creatingRole, setCreatingRole] = useState(false);
  const [roleError, setRoleError] = useState<string | null>(null);

  const { addToast } = useToastActions();
  const permissions = useAuthStore((state) => state.permissions);

  const isSuperuser = permissions.includes('*');
  const canManageTokens = isSuperuser || permissions.includes('tokens:*') || permissions.includes('tokens:manage');
  const canManageRoles = isSuperuser || permissions.includes('roles:*') || permissions.includes('roles:manage');

  // Fetch Tokens
  const fetchTokens = async () => {
    setLoadingTokens(true);
    try {
      const data = await customInstance<APITokenItem[]>({ url: '/tokens' });
      setTokens(Array.isArray(data) ? data : []);
    } catch (err) {
      console.error('Failed to load tokens', err);
    } finally {
      setLoadingTokens(false);
    }
  };

  // Fetch Roles & Catalog
  const fetchRolesAndCatalog = async () => {
    setLoadingRoles(true);
    try {
      const [rolesData, permsData] = await Promise.all([
        customInstance<RoleItem[]>({ url: '/roles' }),
        customInstance<{ permissions: string[] }>({ url: '/permissions' }),
      ]);
      setRoles(Array.isArray(rolesData) ? rolesData : []);
      if (permsData && Array.isArray(permsData.permissions)) {
        setPermissionsCatalog(permsData.permissions);
      }
    } catch (err) {
      console.error('Failed to load roles/permissions', err);
    } finally {
      setLoadingRoles(false);
    }
  };

  useEffect(() => {
    fetchTokens();
    fetchRolesAndCatalog();
  }, []);

  // Handle Token Generation
  const handleGenerateToken = async (e: React.FormEvent) => {
    e.preventDefault();
    setTokenError(null);
    if (!tokenName.trim()) {
      setTokenError('Token name is required');
      return;
    }

    setGeneratingToken(true);
    try {
      const res = await customInstance<{ token: string; apiToken: APITokenItem }>({
        url: '/tokens',
        method: 'POST',
        data: {
          name: tokenName.trim(),
          roleId: tokenRole,
          expiresInDays: tokenExpiryDays,
        },
      });

      setGeneratedPlainToken(res.token);
      setTokenName('');
      addToast({ title: 'API Token generated successfully', variant: 'success' });
      fetchTokens();
    } catch (err) {
      setTokenError(err instanceof Error ? err.message : 'Failed to generate token');
    } finally {
      setGeneratingToken(false);
    }
  };

  // Handle Token Revocation
  const handleRevokeToken = async (id: string, name: string) => {
    if (!window.confirm(`Are you sure you want to revoke "${name}"? Any automated workflows using this token will fail.`)) {
      return;
    }

    try {
      await customInstance({ url: `/tokens/${id}`, method: 'DELETE' });
      addToast({ title: 'Token revoked', variant: 'info' });
      fetchTokens();
    } catch (err) {
      addToast({
        title: err instanceof Error ? err.message : 'Failed to revoke token',
        variant: 'error',
      });
    }
  };

  // Handle Copy Token
  const handleCopyToken = () => {
    if (generatedPlainToken) {
      navigator.clipboard.writeText(generatedPlainToken);
      setCopiedToken(true);
      setTimeout(() => setCopiedToken(false), 2000);
    }
  };

  // Handle Role Creation
  const handleCreateRole = async (e: React.FormEvent) => {
    e.preventDefault();
    setRoleError(null);
    if (!roleName.trim()) {
      setRoleError('Role name is required');
      return;
    }

    setCreatingRole(true);
    try {
      await customInstance<RoleItem>({
        url: '/roles',
        method: 'POST',
        data: {
          name: roleName.trim(),
          description: roleDescription.trim(),
          permissions: selectedPermissions,
        },
      });

      addToast({ title: 'Custom role created', variant: 'success' });
      setShowCreateRoleModal(false);
      setRoleName('');
      setRoleDescription('');
      setSelectedPermissions([]);
      fetchRolesAndCatalog();
    } catch (err) {
      setRoleError(err instanceof Error ? err.message : 'Failed to create role');
    } finally {
      setCreatingRole(false);
    }
  };

  // Handle Role Deletion
  const handleDeleteRole = async (role: RoleItem) => {
    if (role.isSystem) return;
    if (!window.confirm(`Delete custom role "${role.name}"?`)) return;

    try {
      await customInstance({ url: `/roles/${role.id}`, method: 'DELETE' });
      addToast({ title: 'Role deleted', variant: 'info' });
      fetchRolesAndCatalog();
    } catch (err) {
      addToast({
        title: err instanceof Error ? err.message : 'Failed to delete role',
        variant: 'error',
      });
    }
  };

  const togglePermission = (perm: string) => {
    setSelectedPermissions((prev) =>
      prev.includes(perm) ? prev.filter((p) => p !== perm) : [...prev, perm]
    );
  };

  const formatDate = (dateStr?: string | null) => {
    if (!dateStr) return 'Never';
    try {
      return new Date(dateStr).toLocaleDateString(undefined, {
        year: 'numeric',
        month: 'short',
        day: 'numeric',
      });
    } catch {
      return dateStr;
    }
  };

  return (
    <div className="space-y-6">
      {/* Top Header & Tabs */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 border-b border-zinc-800/80 pb-5">
        <div>
          <h2 className="text-xl font-bold tracking-tight text-zinc-100 flex items-center gap-2.5">
            <Key className="w-5 h-5 text-emerald-400" />
            Access Management
          </h2>
          <p className="text-xs text-zinc-400 mt-1">
            Manage personal automation API tokens, custom RBAC roles, and platform permissions
          </p>
        </div>

        {/* Tab Buttons */}
        <div className="flex items-center gap-2 bg-zinc-900/80 p-1 rounded-xl border border-zinc-800">
          <button
            type="button"
            onClick={() => setActiveTab('tokens')}
            className={`px-3.5 py-1.5 rounded-lg text-xs font-semibold transition ${
              activeTab === 'tokens'
                ? 'bg-zinc-800 text-emerald-400 shadow-sm'
                : 'text-zinc-400 hover:text-zinc-200'
            }`}
          >
            Personal API Tokens
          </button>
          <button
            type="button"
            onClick={() => setActiveTab('roles')}
            className={`px-3.5 py-1.5 rounded-lg text-xs font-semibold transition ${
              activeTab === 'roles'
                ? 'bg-zinc-800 text-emerald-400 shadow-sm'
                : 'text-zinc-400 hover:text-zinc-200'
            }`}
          >
            Roles & Permissions
          </button>
        </div>
      </div>

      {/* 1. API TOKENS TAB */}
      {activeTab === 'tokens' && (
        <div className="space-y-4">
          <div className="flex items-center justify-between">
            <span className="text-xs text-zinc-400">
              Personal API tokens authorize CLI automation and Wrangler deployments with <code className="text-emerald-400">cbt_</code> bearer headers.
            </span>
            {canManageTokens && (
              <button
                type="button"
                onClick={() => {
                  setGeneratedPlainToken(null);
                  setShowGenerateModal(true);
                }}
                className="flex items-center gap-2 px-3.5 py-1.5 rounded-xl bg-emerald-500 hover:bg-emerald-600 text-zinc-950 text-xs font-semibold transition shadow-sm"
              >
                <Plus className="w-4 h-4" />
                Generate API Token
              </button>
            )}
          </div>

          {loadingTokens ? (
            <div className="p-12 text-center text-zinc-500 text-xs flex items-center justify-center gap-2">
              <Loader2 className="w-4 h-4 animate-spin text-emerald-400" />
              Loading API tokens...
            </div>
          ) : tokens.length === 0 ? (
            <div className="p-12 text-center rounded-2xl border border-dashed border-zinc-800 bg-zinc-900/20">
              <Key className="w-8 h-8 text-zinc-600 mx-auto mb-3" />
              <h3 className="text-sm font-semibold text-zinc-300">No Personal API Tokens</h3>
              <p className="text-xs text-zinc-500 mt-1 max-w-sm mx-auto">
                Generate an API token to deploy workers via Wrangler or integrate CI/CD workflows.
              </p>
            </div>
          ) : (
            <div className="rounded-xl border border-zinc-800 overflow-hidden bg-zinc-900/30">
              <table className="w-full text-left text-xs text-zinc-300">
                <thead className="bg-zinc-900/70 border-b border-zinc-800 text-[11px] font-semibold text-zinc-400 uppercase tracking-wider">
                  <tr>
                    <th className="px-4 py-3">Token Name</th>
                    <th className="px-4 py-3">Role Scope</th>
                    <th className="px-4 py-3">Created</th>
                    <th className="px-4 py-3">Last Used</th>
                    <th className="px-4 py-3">Expires</th>
                    <th className="px-4 py-3 text-right">Actions</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-zinc-800/80">
                  {tokens.map((tok) => (
                    <tr key={tok.id} className="hover:bg-zinc-800/30 transition">
                      <td className="px-4 py-3 font-medium text-zinc-200 flex items-center gap-2">
                        <Key className="w-3.5 h-3.5 text-zinc-500" />
                        <span>{tok.name}</span>
                      </td>
                      <td className="px-4 py-3">
                        <span className="px-2 py-0.5 rounded-full bg-zinc-800 text-zinc-300 font-mono text-[10px] border border-zinc-700/60">
                          {tok.roleId}
                        </span>
                      </td>
                      <td className="px-4 py-3 text-zinc-400">{formatDate(tok.createdAt)}</td>
                      <td className="px-4 py-3 text-zinc-400">{formatDate(tok.lastUsedAt)}</td>
                      <td className="px-4 py-3 text-zinc-400">
                        {tok.expiresAt ? (
                          <span className="flex items-center gap-1.5">
                            <Clock className="w-3 h-3 text-zinc-500" />
                            {formatDate(tok.expiresAt)}
                          </span>
                        ) : (
                          <span className="text-zinc-500">Never</span>
                        )}
                      </td>
                      <td className="px-4 py-3 text-right">
                        {canManageTokens && (
                          <button
                            type="button"
                            onClick={() => handleRevokeToken(tok.id, tok.name)}
                            className="px-2.5 py-1 rounded-lg text-red-400 hover:text-red-300 hover:bg-red-950/40 border border-transparent hover:border-red-900/60 font-medium transition"
                          >
                            Revoke
                          </button>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      )}

      {/* 2. ROLES & PERMISSIONS TAB */}
      {activeTab === 'roles' && (
        <div className="space-y-4">
          <div className="flex items-center justify-between">
            <span className="text-xs text-zinc-400">
              Role-Based Access Control grants discrete permissions across applications, services, and fleet nodes.
            </span>
            {canManageRoles && (
              <button
                type="button"
                onClick={() => {
                  setRoleError(null);
                  setShowCreateRoleModal(true);
                }}
                className="flex items-center gap-2 px-3.5 py-1.5 rounded-xl bg-emerald-500 hover:bg-emerald-600 text-zinc-950 text-xs font-semibold transition shadow-sm"
              >
                <Plus className="w-4 h-4" />
                Create Custom Role
              </button>
            )}
          </div>

          {loadingRoles ? (
            <div className="p-12 text-center text-zinc-500 text-xs flex items-center justify-center gap-2">
              <Loader2 className="w-4 h-4 animate-spin text-emerald-400" />
              Loading roles...
            </div>
          ) : (
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
              {roles.map((r) => (
                <div
                  key={r.id}
                  className="rounded-xl border border-zinc-800 bg-zinc-900/40 p-4 flex flex-col justify-between hover:border-zinc-700/80 transition"
                >
                  <div>
                    <div className="flex items-center justify-between gap-2 mb-2">
                      <h3 className="text-sm font-semibold text-zinc-100 flex items-center gap-2">
                        {r.isSystem ? (
                          <Lock className="w-3.5 h-3.5 text-zinc-500" />
                        ) : (
                          <Unlock className="w-3.5 h-3.5 text-emerald-400" />
                        )}
                        {r.name}
                      </h3>
                      <span
                        className={`text-[10px] px-2 py-0.5 rounded-full font-medium border ${
                          r.isSystem
                            ? 'bg-zinc-800 text-zinc-400 border-zinc-700/60'
                            : 'bg-emerald-950/60 text-emerald-300 border-emerald-800/80'
                        }`}
                      >
                        {r.isSystem ? 'System' : 'Custom'}
                      </span>
                    </div>

                    <p className="text-xs text-zinc-400 mb-4 min-h-[32px]">
                      {r.description || 'No description provided.'}
                    </p>

                    <div>
                      <span className="text-[10px] font-semibold text-zinc-500 uppercase tracking-wider block mb-2">
                        Permissions ({r.permissions.length})
                      </span>
                      <div className="flex flex-wrap gap-1.5 max-h-28 overflow-y-auto pr-1">
                        {r.permissions.map((perm) => (
                          <span
                            key={perm}
                            className="px-2 py-0.5 rounded bg-zinc-800/80 text-zinc-300 font-mono text-[10px] border border-zinc-700/40"
                          >
                            {perm}
                          </span>
                        ))}
                      </div>
                    </div>
                  </div>

                  <div className="pt-4 mt-4 border-t border-zinc-800/60 flex items-center justify-between text-xs text-zinc-500">
                    <span>ID: {r.id}</span>
                    {!r.isSystem && canManageRoles && (
                      <button
                        type="button"
                        onClick={() => handleDeleteRole(r)}
                        className="text-red-400 hover:text-red-300 transition"
                        title="Delete Role"
                      >
                        <Trash2 className="w-3.5 h-3.5" />
                      </button>
                    )}
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      {/* GENERATE TOKEN MODAL */}
      {showGenerateModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-zinc-950/80 backdrop-blur-sm animate-in fade-in duration-200">
          <div className="bg-zinc-900 border border-zinc-800 rounded-2xl w-full max-w-lg p-6 shadow-2xl space-y-5">
            <div className="flex items-center justify-between border-b border-zinc-800/80 pb-3">
              <h3 className="text-base font-semibold text-zinc-100 flex items-center gap-2">
                <Key className="w-4 h-4 text-emerald-400" />
                {generatedPlainToken ? 'Token Generated' : 'Generate Personal API Token'}
              </h3>
              <button
                type="button"
                onClick={() => setShowGenerateModal(false)}
                className="text-zinc-500 hover:text-zinc-300 transition"
              >
                <X className="w-4 h-4" />
              </button>
            </div>

            {generatedPlainToken ? (
              <div className="space-y-4">
                <div className="p-3.5 rounded-xl border border-amber-900/60 bg-amber-950/30 flex items-start gap-3 text-amber-200 text-xs">
                  <AlertTriangle className="w-4 h-4 text-amber-400 shrink-0 mt-0.5" />
                  <span>
                    Make sure to copy your personal access token now. You will not be able to see it again.
                  </span>
                </div>

                <div className="space-y-1.5">
                  <label className="block text-xs font-semibold text-zinc-400 uppercase tracking-wider">
                    Plaintext API Token
                  </label>
                  <div className="p-3 rounded-xl bg-zinc-950 border border-zinc-800 font-mono text-xs text-emerald-300 break-all select-all flex items-center justify-between gap-2">
                    <span>{generatedPlainToken}</span>
                    <button
                      type="button"
                      onClick={handleCopyToken}
                      className="p-1.5 rounded-lg bg-zinc-800 hover:bg-zinc-700 text-zinc-200 transition shrink-0"
                      title="Copy to clipboard"
                    >
                      {copiedToken ? <Check className="w-4 h-4 text-emerald-400" /> : <Copy className="w-4 h-4" />}
                    </button>
                  </div>
                </div>

                <button
                  type="button"
                  onClick={() => setShowGenerateModal(false)}
                  className="w-full py-2.5 rounded-xl bg-emerald-500 hover:bg-emerald-600 text-zinc-950 text-xs font-semibold transition"
                >
                  Done
                </button>
              </div>
            ) : (
              <form onSubmit={handleGenerateToken} className="space-y-4" noValidate>
                {tokenError && (
                  <div className="p-3 rounded-xl border border-red-900/60 bg-red-950/30 text-red-200 text-xs flex items-center gap-2">
                    <AlertCircle className="w-4 h-4 text-red-400" />
                    <span>{tokenError}</span>
                  </div>
                )}

                <div>
                  <label htmlFor="token-name" className="block text-xs font-semibold text-zinc-300 mb-1">
                    Token Name
                  </label>
                  <input
                    id="token-name"
                    type="text"
                    required
                    value={tokenName}
                    onChange={(e) => setTokenName(e.target.value)}
                    placeholder="e.g. GitHub Actions CI Deployer"
                    className="w-full px-3 py-2 rounded-xl bg-zinc-950 border border-zinc-800 text-xs text-zinc-100 placeholder-zinc-500 focus:outline-none focus:border-emerald-500"
                  />
                </div>

                <div>
                  <label htmlFor="token-role" className="block text-xs font-semibold text-zinc-300 mb-1">
                    Role Scope
                  </label>
                  <select
                    id="token-role"
                    value={tokenRole}
                    onChange={(e) => setTokenRole(e.target.value)}
                    className="w-full px-3 py-2 rounded-xl bg-zinc-950 border border-zinc-800 text-xs text-zinc-100 focus:outline-none focus:border-emerald-500"
                  >
                    {roles.map((r) => (
                      <option key={r.id} value={r.id}>
                        {r.name} ({r.id})
                      </option>
                    ))}
                  </select>
                </div>

                <div>
                  <label htmlFor="token-expiry" className="block text-xs font-semibold text-zinc-300 mb-1">
                    Expiration
                  </label>
                  <select
                    id="token-expiry"
                    value={tokenExpiryDays}
                    onChange={(e) => setTokenExpiryDays(Number(e.target.value))}
                    className="w-full px-3 py-2 rounded-xl bg-zinc-950 border border-zinc-800 text-xs text-zinc-100 focus:outline-none focus:border-emerald-500"
                  >
                    <option value={7}>7 Days</option>
                    <option value={30}>30 Days</option>
                    <option value={90}>90 Days</option>
                    <option value={365}>1 Year</option>
                    <option value={0}>No Expiration</option>
                  </select>
                </div>

                <div className="flex items-center justify-end gap-3 pt-3 border-t border-zinc-800/80">
                  <button
                    type="button"
                    onClick={() => setShowGenerateModal(false)}
                    className="px-4 py-2 rounded-xl text-xs font-medium text-zinc-400 hover:text-zinc-200 transition"
                  >
                    Cancel
                  </button>
                  <button
                    type="submit"
                    disabled={generatingToken}
                    className="px-4 py-2 rounded-xl bg-emerald-500 hover:bg-emerald-600 text-zinc-950 text-xs font-semibold disabled:opacity-50 transition"
                  >
                    {generatingToken ? 'Generating...' : 'Generate'}
                  </button>
                </div>
              </form>
            )}
          </div>
        </div>
      )}

      {/* CREATE ROLE MODAL */}
      {showCreateRoleModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-zinc-950/80 backdrop-blur-sm animate-in fade-in duration-200">
          <div className="bg-zinc-900 border border-zinc-800 rounded-2xl w-full max-w-xl p-6 shadow-2xl space-y-5 max-h-[90vh] flex flex-col">
            <div className="flex items-center justify-between border-b border-zinc-800/80 pb-3">
              <h3 className="text-base font-semibold text-zinc-100 flex items-center gap-2">
                <Shield className="w-4 h-4 text-emerald-400" />
                Create Custom Role
              </h3>
              <button
                type="button"
                onClick={() => setShowCreateRoleModal(false)}
                className="text-zinc-500 hover:text-zinc-300 transition"
              >
                <X className="w-4 h-4" />
              </button>
            </div>

            <form onSubmit={handleCreateRole} className="space-y-4 flex-1 overflow-y-auto pr-1" noValidate>
              {roleError && (
                <div className="p-3 rounded-xl border border-red-900/60 bg-red-950/30 text-red-200 text-xs flex items-center gap-2">
                  <AlertCircle className="w-4 h-4 text-red-400" />
                  <span>{roleError}</span>
                </div>
              )}

              <div>
                <label htmlFor="role-name" className="block text-xs font-semibold text-zinc-300 mb-1">
                  Role Name
                </label>
                <input
                  id="role-name"
                  type="text"
                  required
                  value={roleName}
                  onChange={(e) => setRoleName(e.target.value)}
                  placeholder="e.g. Release Engineer"
                  className="w-full px-3 py-2 rounded-xl bg-zinc-950 border border-zinc-800 text-xs text-zinc-100 placeholder-zinc-500 focus:outline-none focus:border-emerald-500"
                />
              </div>

              <div>
                <label htmlFor="role-desc" className="block text-xs font-semibold text-zinc-300 mb-1">
                  Description
                </label>
                <input
                  id="role-desc"
                  type="text"
                  value={roleDescription}
                  onChange={(e) => setRoleDescription(e.target.value)}
                  placeholder="Role capabilities and responsibilities"
                  className="w-full px-3 py-2 rounded-xl bg-zinc-950 border border-zinc-800 text-xs text-zinc-100 placeholder-zinc-500 focus:outline-none focus:border-emerald-500"
                />
              </div>

              <div>
                <label className="block text-xs font-semibold text-zinc-300 mb-2">
                  Permissions ({selectedPermissions.length} selected)
                </label>
                <div className="grid grid-cols-2 gap-2 p-3 rounded-xl bg-zinc-950/60 border border-zinc-800 max-h-56 overflow-y-auto">
                  {permissionsCatalog.map((perm) => (
                    <label
                      key={perm}
                      className="flex items-center gap-2 text-xs text-zinc-300 hover:text-zinc-100 cursor-pointer p-1 rounded hover:bg-zinc-900"
                    >
                      <input
                        type="checkbox"
                        checked={selectedPermissions.includes(perm)}
                        onChange={() => togglePermission(perm)}
                        className="rounded border-zinc-700 bg-zinc-900 text-emerald-500 focus:ring-0 focus:outline-none"
                      />
                      <span className="font-mono text-[11px] truncate">{perm}</span>
                    </label>
                  ))}
                </div>
              </div>

              <div className="flex items-center justify-end gap-3 pt-3 border-t border-zinc-800/80">
                <button
                  type="button"
                  onClick={() => setShowCreateRoleModal(false)}
                  className="px-4 py-2 rounded-xl text-xs font-medium text-zinc-400 hover:text-zinc-200 transition"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  disabled={creatingRole}
                  className="px-4 py-2 rounded-xl bg-emerald-500 hover:bg-emerald-600 text-zinc-950 text-xs font-semibold disabled:opacity-50 transition"
                >
                  {creatingRole ? 'Creating...' : 'Create Role'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
