import { useAuthStore } from '../stores/useAuthStore';

let isInstalled = false;
let originalFetchBackup: typeof fetch | null = null;
let refreshPromise: Promise<string | null> | null = null;

export type AuthRedirectHandler = (to: string) => void;

let redirectHandler: AuthRedirectHandler = (to: string) => {
  if (typeof window !== 'undefined' && window.location) {
    if (!window.location.pathname.startsWith('/login') && !window.location.pathname.startsWith('/setup')) {
      window.location.href = to;
    }
  }
};

export const setFetchInterceptorRedirectHandler = (handler: AuthRedirectHandler) => {
  redirectHandler = handler;
};

export function isAuthBypassUrl(url: string): boolean {
  return (
    url.includes('/auth/login') ||
    url.includes('/auth/setup') ||
    url.includes('/auth/refresh') ||
    url.includes('/cluster/status') ||
    url.includes('/health')
  );
}

export function isApiUrl(url: string): boolean {
  if (url.startsWith('/api/') || url.startsWith('/api/v1')) {
    return true;
  }
  if (typeof window !== 'undefined' && window.location) {
    try {
      const parsed = new URL(url, window.location.origin);
      if (parsed.origin === window.location.origin) {
        return parsed.pathname.startsWith('/api/') || parsed.pathname.startsWith('/api/v1');
      }
      return false;
    } catch {
      return false;
    }
  }
  if (url.startsWith('http://localhost') || url.startsWith('http://127.0.0.1')) {
    return url.includes('/api/');
  }
  return false;
}

export async function silentRefreshToken(): Promise<string | null> {
  const refreshToken = useAuthStore.getState().refreshToken;
  if (!refreshToken) {
    useAuthStore.getState().actions.clearAuth();
    redirectHandler('/login');
    return null;
  }

  if (refreshPromise) {
    return refreshPromise;
  }

  refreshPromise = (async () => {
    try {
      const fetchFn = originalFetchBackup || (typeof window !== 'undefined' ? window.fetch : globalThis.fetch);
      const res = await fetchFn('/api/v1/auth/refresh', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ refreshToken }),
      });

      if (!res.ok) {
        useAuthStore.getState().actions.clearAuth();
        redirectHandler('/login');
        return null;
      }

      const data = await res.json();
      if (data.accessToken) {
        useAuthStore.getState().actions.setAccessToken(data.accessToken);
        if (data.refreshToken) {
          useAuthStore.getState().actions.setRefreshToken(data.refreshToken);
        }
        return data.accessToken as string;
      }
      return null;
    } catch {
      useAuthStore.getState().actions.clearAuth();
      redirectHandler('/login');
      return null;
    } finally {
      refreshPromise = null;
    }
  })();

  return refreshPromise;
}

export function installFetchInterceptor(
  target: { fetch: typeof fetch } = typeof window !== 'undefined' ? window : globalThis
) {
  if (isInstalled) return;

  const originalFetch = target.fetch.bind(target);
  originalFetchBackup = originalFetch;

  target.fetch = async (input: RequestInfo | URL, init?: RequestInit): Promise<Response> => {
    let url: string;
    if (typeof input === 'string') {
      url = input;
    } else if (input instanceof URL) {
      url = input.toString();
    } else if (typeof Request !== 'undefined' && input instanceof Request) {
      url = input.url;
    } else {
      url = String(input);
    }

    const needsAuth = isApiUrl(url) && !isAuthBypassUrl(url);

    if (!needsAuth) {
      return originalFetch(input, init);
    }

    // Extract or construct Headers
    let headers: Headers;
    if (init?.headers) {
      headers = new Headers(init.headers);
    } else if (typeof Request !== 'undefined' && input instanceof Request) {
      headers = new Headers(input.headers);
    } else {
      headers = new Headers();
    }

    const token = useAuthStore.getState().accessToken;
    if (token && !headers.has('Authorization')) {
      headers.set('Authorization', `Bearer ${token}`);
    }

    let modifiedInput = input;
    let modifiedInit = init;

    if (typeof Request !== 'undefined' && input instanceof Request) {
      modifiedInput = new Request(input, { headers });
    } else {
      modifiedInit = { ...init, headers };
    }

    const response = await originalFetch(modifiedInput, modifiedInit);

    // Silent refresh on 401 Unauthorized for protected API routes
    const isRetry = Boolean((init as any)?._isRetry);
    if (response.status === 401 && !isRetry && !url.includes('/auth/refresh')) {
      const newToken = await silentRefreshToken();
      if (newToken) {
        headers.set('Authorization', `Bearer ${newToken}`);
        if (typeof Request !== 'undefined' && input instanceof Request) {
          try {
            const retriedRequest = new Request(input, { headers });
            return await originalFetch(retriedRequest);
          } catch {
            return response;
          }
        } else {
          return await originalFetch(input, { ...init, headers, _isRetry: true } as RequestInit);
        }
      }
    }

    return response;
  };

  isInstalled = true;
}

export function uninstallFetchInterceptor(
  target: { fetch: typeof fetch } = typeof window !== 'undefined' ? window : globalThis,
  originalFetch?: typeof fetch
) {
  if (originalFetch) {
    target.fetch = originalFetch;
  } else if (originalFetchBackup) {
    target.fetch = originalFetchBackup;
  }
  originalFetchBackup = null;
  isInstalled = false;
  refreshPromise = null;
}
