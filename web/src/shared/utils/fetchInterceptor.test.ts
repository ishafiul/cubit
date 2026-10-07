import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import {
  installFetchInterceptor,
  uninstallFetchInterceptor,
  setFetchInterceptorRedirectHandler,
  isApiUrl,
  isAuthBypassUrl,
} from './fetchInterceptor';
import { useAuthStore } from '../stores/useAuthStore';

describe('Given fetchInterceptor utility functions', () => {
  it('When checking isAuthBypassUrl with login/setup/refresh/status Then returns true', () => {
    expect(isAuthBypassUrl('/api/v1/auth/login')).toBe(true);
    expect(isAuthBypassUrl('/api/v1/auth/setup')).toBe(true);
    expect(isAuthBypassUrl('/api/v1/auth/refresh')).toBe(true);
    expect(isAuthBypassUrl('/api/v1/cluster/status')).toBe(true);
    expect(isAuthBypassUrl('/health')).toBe(true);
    expect(isAuthBypassUrl('/api/v1/kv/namespaces')).toBe(false);
  });

  it('When checking isApiUrl with relative or same-origin paths Then correctly distinguishes API paths', () => {
    expect(isApiUrl('/api/v1/kv/namespaces')).toBe(true);
    expect(isApiUrl('/api/v1/d1/databases')).toBe(true);
    expect(isApiUrl('/assets/index.js')).toBe(false);
    expect(isApiUrl('https://api.github.com/repos')).toBe(false);
  });
});

describe('Given installFetchInterceptor', () => {
  let mockFetch: any;
  let redirectedTo: string | null = null;
  let originalFetch: typeof fetch;

  beforeEach(() => {
    redirectedTo = null;
    useAuthStore.getState().actions.clearAuth();
    setFetchInterceptorRedirectHandler((to) => {
      redirectedTo = to;
    });

    originalFetch = window.fetch;
    mockFetch = vi.fn();
    window.fetch = mockFetch;
    installFetchInterceptor(window);
  });

  afterEach(() => {
    uninstallFetchInterceptor(window, originalFetch);
    window.fetch = originalFetch;
    vi.restoreAllMocks();
  });

  describe('When user is authenticated and requests a protected service endpoint', () => {
    it('Then injects Authorization: Bearer token header', async () => {
      useAuthStore.getState().actions.setAuth({
        accessToken: 'mock-access-token-123',
        refreshToken: 'mock-refresh-token-456',
        user: { id: 'u1', email: 'admin@cubit.dev', name: 'Admin', roleId: 'admin', isActive: true, createdAt: '', updatedAt: '' },
      });

      mockFetch.mockResolvedValueOnce(new Response(JSON.stringify([{ id: 'ns-1' }]), { status: 200 }));

      const res = await window.fetch('/api/v1/kv/namespaces');
      expect(res.status).toBe(200);

      expect(mockFetch).toHaveBeenCalledTimes(1);
      const callArgs = mockFetch.mock.calls[0];
      const headers = callArgs[1]?.headers as Headers;
      expect(headers.get('Authorization')).toBe('Bearer mock-access-token-123');
    });
  });

  describe('When user requests an endpoint with pre-existing Authorization header', () => {
    it('Then preserves the existing Authorization header without overwriting', async () => {
      useAuthStore.getState().actions.setAuth({
        accessToken: 'store-token',
        refreshToken: 'mock-refresh-token',
      });

      mockFetch.mockResolvedValueOnce(new Response(JSON.stringify({ ok: true }), { status: 200 }));

      await window.fetch('/api/v1/kv/namespaces', {
        headers: { Authorization: 'Bearer custom-token' },
      });

      const callArgs = mockFetch.mock.calls[0];
      const headers = callArgs[1]?.headers as Headers;
      expect(headers.get('Authorization')).toBe('Bearer custom-token');
    });
  });

  describe('When requesting a public auth bypass endpoint', () => {
    it('Then does not inject Authorization header', async () => {
      useAuthStore.getState().actions.setAuth({
        accessToken: 'store-token',
        refreshToken: 'mock-refresh-token',
      });

      mockFetch.mockResolvedValueOnce(new Response(JSON.stringify({ initialized: true }), { status: 200 }));

      await window.fetch('/api/v1/cluster/status');

      const callArgs = mockFetch.mock.calls[0];
      const headers = callArgs[1]?.headers as Headers | undefined;
      expect(headers?.get?.('Authorization') ?? undefined).toBeUndefined();
    });
  });

  describe('When a protected endpoint returns 401 and refresh succeeds', () => {
    it('Then rotates tokens and retries request with new access token', async () => {
      useAuthStore.getState().actions.setAuth({
        accessToken: 'expired-access-token',
        refreshToken: 'valid-refresh-token',
      });

      // 1st call: initial protected endpoint returns 401
      // 2nd call: /api/v1/auth/refresh returns 200 with new tokens
      // 3rd call: retried protected endpoint returns 200
      mockFetch
        .mockResolvedValueOnce(new Response(JSON.stringify({ error: 'Unauthorized' }), { status: 401 }))
        .mockResolvedValueOnce(
          new Response(
            JSON.stringify({
              accessToken: 'fresh-new-access-token',
              refreshToken: 'fresh-new-refresh-token',
            }),
            { status: 200 }
          )
        )
        .mockResolvedValueOnce(new Response(JSON.stringify([{ id: 'ns-1' }]), { status: 200 }));

      const res = await window.fetch('/api/v1/kv/namespaces');
      expect(res.status).toBe(200);

      // Verify token in store updated
      expect(useAuthStore.getState().accessToken).toBe('fresh-new-access-token');

      // Verify retried call had fresh new token
      expect(mockFetch).toHaveBeenCalledTimes(3);
      const retriedHeaders = mockFetch.mock.calls[2][1]?.headers as Headers;
      expect(retriedHeaders.get('Authorization')).toBe('Bearer fresh-new-access-token');
    });
  });

  describe('When a protected endpoint returns 401 and refresh fails', () => {
    it('Then clears auth in store and redirects to /login', async () => {
      useAuthStore.getState().actions.setAuth({
        accessToken: 'expired-access-token',
        refreshToken: 'invalid-refresh-token',
      });

      mockFetch
        .mockResolvedValueOnce(new Response(JSON.stringify({ error: 'Unauthorized' }), { status: 401 }))
        .mockResolvedValueOnce(new Response(JSON.stringify({ error: 'Token expired' }), { status: 401 }));

      const res = await window.fetch('/api/v1/kv/namespaces');
      expect(res.status).toBe(401);

      expect(useAuthStore.getState().isAuthenticated).toBe(false);
      expect(useAuthStore.getState().accessToken).toBeNull();
      expect(redirectedTo).toBe('/login');
    });
  });

  describe('When multiple concurrent requests receive 401', () => {
    it('Then deduplicates token refresh to a single /api/v1/auth/refresh call', async () => {
      useAuthStore.getState().actions.setAuth({
        accessToken: 'expired-access-token',
        refreshToken: 'valid-refresh-token',
      });

      let refreshCalls = 0;
      mockFetch.mockImplementation(async (url: string) => {
        if (url === '/api/v1/auth/refresh') {
          refreshCalls++;
          return new Response(
            JSON.stringify({ accessToken: 'new-shared-token', refreshToken: 'new-refresh-token' }),
            { status: 200 }
          );
        }
        if (url.startsWith('/api/v1/')) {
          const authHeader = (mockFetch.mock.calls[mockFetch.mock.calls.length - 1][1]?.headers as Headers)?.get('Authorization');
          if (authHeader === 'Bearer new-shared-token') {
            return new Response(JSON.stringify([{ ok: true }]), { status: 200 });
          }
          return new Response(JSON.stringify({ error: 'Unauthorized' }), { status: 401 });
        }
        return new Response('{}', { status: 200 });
      });

      // Fire 3 simultaneous service requests
      const [res1, res2, res3] = await Promise.all([
        window.fetch('/api/v1/kv/namespaces'),
        window.fetch('/api/v1/r2/buckets'),
        window.fetch('/api/v1/d1/databases'),
      ]);

      expect(res1.status).toBe(200);
      expect(res2.status).toBe(200);
      expect(res3.status).toBe(200);

      // Verify /api/v1/auth/refresh was only called once
      expect(refreshCalls).toBe(1);
      expect(useAuthStore.getState().accessToken).toBe('new-shared-token');
    });
  });
});

