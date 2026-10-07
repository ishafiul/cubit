import React, { useState } from 'react';
import { useNavigate, Link } from '@tanstack/react-router';
import { Shield, User, Mail, Lock, ArrowRight, AlertCircle, Loader2 } from 'lucide-react';
import { customInstance } from '../../api/custom-instance';
import { useAuthStore } from '../../shared/stores/useAuthStore';
import { useToastActions } from '../../shared/stores/useToastStore';
import { setCachedClusterStatus } from '../../shared/utils/clusterStatus';

interface SetupResponse {
  accessToken: string;
  refreshToken: string;
  expiresIn: number;
  user: {
    id: string;
    name: string;
    email: string;
    roleId: string;
    isActive: boolean;
    createdAt?: string;
    updatedAt?: string;
  };
}

export function SetupPage() {
  const navigate = useNavigate();
  const { setAuth } = useAuthStore((state) => state.actions);
  const { addToast } = useToastActions();

  const [name, setName] = useState('');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);

    const cleanName = name.trim();
    const cleanEmail = email.trim();

    if (!cleanName) {
      setError('Full name is required');
      return;
    }
    if (!cleanEmail) {
      setError('Email address is required');
      return;
    }
    if (!password) {
      setError('Password is required');
      return;
    }
    if (password.length < 8) {
      setError('Password must be at least 8 characters long');
      return;
    }
    if (password !== confirmPassword) {
      setError('Passwords do not match');
      return;
    }

    setIsSubmitting(true);
    try {
      const resp = await customInstance<SetupResponse>({
        url: '/auth/setup',
        method: 'POST',
        data: {
          name: cleanName,
          email: cleanEmail,
          password,
        },
      });

      setAuth({
        accessToken: resp.accessToken,
        refreshToken: resp.refreshToken,
        user: {
          id: resp.user.id,
          name: resp.user.name,
          email: resp.user.email,
          roleId: resp.user.roleId,
          isActive: resp.user.isActive,
          createdAt: resp.user.createdAt || new Date().toISOString(),
          updatedAt: resp.user.updatedAt || new Date().toISOString(),
        },
        permissions: ['*'],
      });

      // Crucial: Mark cluster as initialized so requireAuthGuard allows immediate access to /apps
      setCachedClusterStatus({ initialized: true, version: '1.3.0' });

      addToast({
        title: 'Root administrator account created successfully',
        variant: 'success',
      });

      navigate({ to: '/apps' });
      if (typeof window !== 'undefined' && window.location.pathname === '/setup') {
        window.location.href = '/apps';
      }
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Failed to initialize cluster';
      if (msg.toLowerCase().includes('already initialized')) {
        setCachedClusterStatus({ initialized: true, version: '1.3.0' });
        setError('Cluster is already initialized. Please sign in with your administrator credentials.');
      } else {
        setError(msg);
      }
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div className="min-h-screen bg-zinc-950 flex flex-col justify-center py-12 sm:px-6 lg:px-8">
      <div className="sm:mx-auto sm:w-full sm:max-w-md">
        {/* Shield Icon / Onboarding Title */}
        <div className="flex items-center justify-center gap-3">
          <div className="w-10 h-10 rounded-xl bg-emerald-500/10 border border-emerald-500/20 flex items-center justify-center text-emerald-400 shadow-lg shadow-emerald-500/5">
            <Shield className="w-5 h-5" />
          </div>
          <div>
            <h1 className="text-xl font-bold tracking-tight text-zinc-100">CUBIT</h1>
            <p className="text-xs text-zinc-400">Cluster Initialization Wizard</p>
          </div>
        </div>

        <h2 className="mt-6 text-center text-2xl font-semibold tracking-tight text-zinc-100">
          Initialize Cubit Cluster
        </h2>
        <p className="mt-1 text-center text-sm text-zinc-400">
          Create the root administrator account to bootstrap cluster security
        </p>
      </div>

      <div className="mt-8 sm:mx-auto sm:w-full sm:max-w-md">
        <div className="bg-zinc-900/60 backdrop-blur border border-zinc-800/80 py-8 px-4 shadow-xl sm:rounded-2xl sm:px-10">
          <form className="space-y-4" onSubmit={handleSubmit} noValidate>
            {error && (
              <div
                role="alert"
                className="p-3.5 rounded-xl border border-red-900/60 bg-red-950/40 flex items-start gap-3 text-red-200 text-xs animate-in fade-in duration-200"
              >
                <AlertCircle className="w-4 h-4 text-red-400 shrink-0 mt-0.5" />
                <span className="flex-1 font-medium">{error}</span>
              </div>
            )}

            <div>
              <label
                htmlFor="setup-name"
                className="block text-xs font-semibold text-zinc-300 uppercase tracking-wider mb-1.5"
              >
                Full Name
              </label>
              <div className="relative">
                <div className="absolute inset-y-0 left-0 pl-3.5 flex items-center pointer-events-none text-zinc-500">
                  <User className="w-4 h-4" />
                </div>
                <input
                  id="setup-name"
                  type="text"
                  autoComplete="name"
                  required
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  placeholder="Primary Operator"
                  className="w-full pl-10 pr-3.5 py-2.5 rounded-xl bg-zinc-950/80 border border-zinc-800 text-sm text-zinc-100 placeholder-zinc-500 focus:outline-none focus:border-emerald-500/80 focus:ring-1 focus:ring-emerald-500/80 transition"
                />
              </div>
            </div>

            <div>
              <label
                htmlFor="setup-email"
                className="block text-xs font-semibold text-zinc-300 uppercase tracking-wider mb-1.5"
              >
                Email Address
              </label>
              <div className="relative">
                <div className="absolute inset-y-0 left-0 pl-3.5 flex items-center pointer-events-none text-zinc-500">
                  <Mail className="w-4 h-4" />
                </div>
                <input
                  id="setup-email"
                  type="email"
                  autoComplete="email"
                  required
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  placeholder="admin@cubit.dev"
                  className="w-full pl-10 pr-3.5 py-2.5 rounded-xl bg-zinc-950/80 border border-zinc-800 text-sm text-zinc-100 placeholder-zinc-500 focus:outline-none focus:border-emerald-500/80 focus:ring-1 focus:ring-emerald-500/80 transition"
                />
              </div>
            </div>

            <div>
              <label
                htmlFor="setup-password"
                className="block text-xs font-semibold text-zinc-300 uppercase tracking-wider mb-1.5"
              >
                Password
              </label>
              <div className="relative">
                <div className="absolute inset-y-0 left-0 pl-3.5 flex items-center pointer-events-none text-zinc-500">
                  <Lock className="w-4 h-4" />
                </div>
                <input
                  id="setup-password"
                  type="password"
                  autoComplete="new-password"
                  required
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  placeholder="Minimum 8 characters"
                  className="w-full pl-10 pr-3.5 py-2.5 rounded-xl bg-zinc-950/80 border border-zinc-800 text-sm text-zinc-100 placeholder-zinc-500 focus:outline-none focus:border-emerald-500/80 focus:ring-1 focus:ring-emerald-500/80 transition"
                />
              </div>
            </div>

            <div>
              <label
                htmlFor="setup-confirm-password"
                className="block text-xs font-semibold text-zinc-300 uppercase tracking-wider mb-1.5"
              >
                Confirm Password
              </label>
              <div className="relative">
                <div className="absolute inset-y-0 left-0 pl-3.5 flex items-center pointer-events-none text-zinc-500">
                  <Lock className="w-4 h-4" />
                </div>
                <input
                  id="setup-confirm-password"
                  type="password"
                  autoComplete="new-password"
                  required
                  value={confirmPassword}
                  onChange={(e) => setConfirmPassword(e.target.value)}
                  placeholder="Repeat password"
                  className="w-full pl-10 pr-3.5 py-2.5 rounded-xl bg-zinc-950/80 border border-zinc-800 text-sm text-zinc-100 placeholder-zinc-500 focus:outline-none focus:border-emerald-500/80 focus:ring-1 focus:ring-emerald-500/80 transition"
                />
              </div>
            </div>

            <div className="pt-2">
              <button
                type="submit"
                disabled={isSubmitting}
                className="w-full flex items-center justify-center gap-2 py-2.5 px-4 rounded-xl text-sm font-semibold text-zinc-950 bg-emerald-400 hover:bg-emerald-300 active:scale-[0.99] disabled:opacity-50 disabled:cursor-not-allowed transition shadow-sm"
              >
                {isSubmitting ? (
                  <>
                    <Loader2 className="w-4 h-4 animate-spin text-zinc-950" />
                    <span>Bootstrapping...</span>
                  </>
                ) : (
                  <>
                    <span>Create Administrator</span>
                    <ArrowRight className="w-4 h-4" />
                  </>
                )}
              </button>
            </div>
          </form>

          <div className="mt-6 pt-5 border-t border-zinc-800/80 flex items-center justify-between text-xs text-zinc-400">
            <span>Cluster already bootstrapped?</span>
            <Link
              to="/login"
              className="text-emerald-400 hover:text-emerald-300 font-medium transition"
            >
              Sign In Instead
            </Link>
          </div>
        </div>
      </div>
    </div>
  );
}
