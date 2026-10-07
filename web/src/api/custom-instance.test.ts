import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { customInstance, setAuthRedirectHandler } from './custom-instance';
import { useAuthStore } from '../shared/stores/useAuthStore';

describe('Given customInstance fetch client', () => {
  let originalFetch: typeof global.fetch;

  beforeEach(() => {
    originalFetch = global.fetch;
    useAuthStore.getState().actions.clearAuth();
  });

  afterEach(() => {
    global.fetch = originalFetch;
    vi.restoreAllMocks();
  });

  describe('When making a successful GET request with query params', () => {
    let result: unknown;
    let fetchCalledWithUrl: string;

    beforeEach(async () => {
      global.fetch = vi.fn().mockImplementation((url: string) => {
        fetchCalledWithUrl = url;
        return Promise.resolve(
          new Response(JSON.stringify({ status: 'ok', count: 42 }), {
            status: 200,
            headers: { 'Content-Type': 'application/json' },
          })
        );
      });

      result = await customInstance<{ status: string; count: number }>({
        url: '/applications',
        method: 'GET',
        params: { page: 1, filter: 'active' },
      });
    });

    it('Then it formats query parameters into the URL', () => {
      expect(fetchCalledWithUrl).toBe('/api/v1/applications?page=1&filter=active');
    });

    it('Then it parses and returns the JSON body', () => {
      expect(result).toEqual({ status: 'ok', count: 42 });
    });
  });

  describe('When making a POST request with json data payload', () => {
    let capturedOptions: RequestInit | undefined;

    beforeEach(async () => {
      global.fetch = vi.fn().mockImplementation((_url: string, opts?: RequestInit) => {
        capturedOptions = opts;
        return Promise.resolve(
          new Response(JSON.stringify({ id: 'app-1', name: 'my-worker' }), {
            status: 201,
            headers: { 'Content-Type': 'application/json' },
          })
        );
      });

      await customInstance({
        url: '/applications',
        method: 'POST',
        data: { name: 'my-worker', gitRepo: 'https://github.com/org/worker' },
      });
    });

    it('Then it sends serialized JSON in request body', () => {
      expect(capturedOptions?.body).toBe(
        JSON.stringify({ name: 'my-worker', gitRepo: 'https://github.com/org/worker' })
      );
    });

    it('Then it sets application/json content-type header', () => {
      expect(capturedOptions?.headers).toMatchObject({
        'Content-Type': 'application/json',
      });
    });
  });

  describe('When the user is authenticated in useAuthStore', () => {
    let capturedHeaders: Record<string, string> | undefined;

    beforeEach(async () => {
      useAuthStore.getState().actions.setAuth({
        accessToken: 'valid-test-access-token',
        refreshToken: 'valid-test-refresh-token',
      });

      global.fetch = vi.fn().mockImplementation((_url: string, opts?: RequestInit) => {
        capturedHeaders = opts?.headers as Record<string, string>;
        return Promise.resolve(
          new Response(JSON.stringify({ status: 'ok' }), {
            status: 200,
            headers: { 'Content-Type': 'application/json' },
          })
        );
      });

      await customInstance({ url: '/applications' });
    });

    it('Then it injects the Bearer token into the Authorization header', () => {
      expect(capturedHeaders?.['Authorization']).toBe('Bearer valid-test-access-token');
    });
  });

  describe('When a request has a custom Authorization header', () => {
    let capturedHeaders: Record<string, string> | undefined;

    beforeEach(async () => {
      useAuthStore.getState().actions.setAuth({
        accessToken: 'store-token',
      });

      global.fetch = vi.fn().mockImplementation((_url: string, opts?: RequestInit) => {
        capturedHeaders = opts?.headers as Record<string, string>;
        return Promise.resolve(
          new Response(JSON.stringify({ status: 'ok' }), { status: 200 })
        );
      });

      await customInstance({
        url: '/applications',
        headers: { Authorization: 'Bearer custom-override-token' },
      });
    });

    it('Then it preserves the custom Authorization header', () => {
      expect(capturedHeaders?.['Authorization']).toBe('Bearer custom-override-token');
    });
  });

  describe('When an API call returns 401 and a refresh token exists', () => {
    it('Then it transparently calls /refresh, updates the store, and retries the original request', async () => {
      useAuthStore.getState().actions.setAuth({
        accessToken: 'expired-token',
        refreshToken: 'valid-refresh-token',
      });

      const fetchMock = vi.fn().mockImplementation((url: string, opts?: RequestInit) => {
        if (url === '/api/v1/applications' && (opts?.headers as Record<string, string>)?.[
          'Authorization'
        ] === 'Bearer expired-token') {
          return Promise.resolve(
            new Response(JSON.stringify({ error: 'invalid or expired token' }), { status: 401 })
          );
        }

        if (url === '/api/v1/auth/refresh') {
          return Promise.resolve(
            new Response(
              JSON.stringify({
                accessToken: 'freshly-rotated-access-token',
                refreshToken: 'freshly-rotated-refresh-token',
              }),
              { status: 200 }
            )
          );
        }

        if (url === '/api/v1/applications' && (opts?.headers as Record<string, string>)?.[
          'Authorization'
        ] === 'Bearer freshly-rotated-access-token') {
          return Promise.resolve(
            new Response(JSON.stringify({ apps: ['app-a', 'app-b'] }), { status: 200 })
          );
        }

        return Promise.reject(new Error(`Unexpected call to ${url}`));
      });

      global.fetch = fetchMock;

      const result = await customInstance<{ apps: string[] }>({ url: '/applications' });

      expect(result).toEqual({ apps: ['app-a', 'app-b'] });
      expect(useAuthStore.getState().accessToken).toBe('freshly-rotated-access-token');
      expect(useAuthStore.getState().refreshToken).toBe('freshly-rotated-refresh-token');
      expect(fetchMock).toHaveBeenCalledTimes(3);
    });
  });

  describe('When an API call returns 401 and refresh also fails with 401', () => {
    it('Then it clears the session, triggers redirect to /login, and throws an error', async () => {
      useAuthStore.getState().actions.setAuth({
        accessToken: 'expired-token',
        refreshToken: 'revoked-refresh-token',
      });

      const redirectedUrls: string[] = [];
      setAuthRedirectHandler((to) => {
        redirectedUrls.push(to);
      });

      global.fetch = vi.fn().mockImplementation((url: string) => {
        if (url === '/api/v1/applications') {
          return Promise.resolve(
            new Response(JSON.stringify({ error: 'token expired' }), { status: 401 })
          );
        }
        if (url === '/api/v1/auth/refresh') {
          return Promise.resolve(
            new Response(JSON.stringify({ error: 'invalid refresh token' }), { status: 401 })
          );
        }
        return Promise.reject(new Error('unexpected'));
      });

      let caughtError: Error | null = null;
      try {
        await customInstance({ url: '/applications' });
      } catch (err) {
        caughtError = err as Error;
      }

      expect(caughtError).not.toBeNull();
      expect(useAuthStore.getState().isAuthenticated).toBe(false);
      expect(useAuthStore.getState().accessToken).toBeNull();
      expect(redirectedUrls).toContain('/login');
    });
  });

  describe('When an API call returns 401 and no refresh token is stored', () => {
    it('Then it clears auth, redirects to /login immediately without calling refresh', async () => {
      useAuthStore.getState().actions.setAuth({
        accessToken: 'orphaned-token',
        refreshToken: null,
      });

      const redirectedUrls: string[] = [];
      setAuthRedirectHandler((to) => {
        redirectedUrls.push(to);
      });

      const fetchMock = vi.fn().mockImplementation((url: string) => {
        if (url === '/api/v1/applications') {
          return Promise.resolve(
            new Response(JSON.stringify({ error: 'token expired' }), { status: 401 })
          );
        }
        return Promise.reject(new Error(`unexpected ${url}`));
      });
      global.fetch = fetchMock;

      let caughtError: Error | null = null;
      try {
        await customInstance({ url: '/applications' });
      } catch (err) {
        caughtError = err as Error;
      }

      expect(caughtError).not.toBeNull();
      expect(fetchMock).toHaveBeenCalledTimes(1);
      expect(useAuthStore.getState().isAuthenticated).toBe(false);
      expect(redirectedUrls).toContain('/login');
    });
  });

  describe('When an auth endpoint like /auth/login returns 401', () => {
    it('Then it throws the credential error directly without attempting to refresh or redirect', async () => {
      const redirectedUrls: string[] = [];
      setAuthRedirectHandler((to) => {
        redirectedUrls.push(to);
      });

      global.fetch = vi.fn().mockImplementation(() => {
        return Promise.resolve(
          new Response(JSON.stringify({ error: 'invalid email or password' }), { status: 401 })
        );
      });

      let caughtError: Error | null = null;
      try {
        await customInstance({ url: '/auth/login', method: 'POST', data: { email: 'a', password: 'b' } });
      } catch (err) {
        caughtError = err as Error;
      }

      expect(caughtError?.message).toBe('invalid email or password');
      expect(redirectedUrls).toHaveLength(0);
    });
  });

  describe('When the backend returns a 404 error response', () => {
    let caughtError: Error | null = null;

    beforeEach(async () => {
      global.fetch = vi.fn().mockImplementation(() => {
        return Promise.resolve(
          new Response(JSON.stringify({ message: 'application not found' }), {
            status: 404,
            headers: { 'Content-Type': 'application/json' },
          })
        );
      });

      try {
        await customInstance({ url: '/applications/non-existent' });
      } catch (err) {
        caughtError = err as Error;
      }
    });

    it('Then it throws an error containing the server error message', () => {
      expect(caughtError).not.toBeNull();
      expect(caughtError?.message).toBe('application not found');
    });
  });

  it('When request returns HTTP 204 No Content then it returns an empty object', async () => {
    global.fetch = vi.fn().mockImplementation(() => {
      return Promise.resolve(new Response(null, { status: 204 }));
    });

    const res = await customInstance({ url: '/nodes/node-1', method: 'DELETE' });
    expect(res).toEqual({});
  });
});
