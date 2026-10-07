// Custom Fetch client mutator for Orval with Bearer Auth injection & Silent 401 Refresh
import { useAuthStore } from '../shared/stores/useAuthStore';

export interface CustomInstanceConfig {
  url: string;
  method?: string;
  params?: Record<string, string | number | boolean | undefined>;
  data?: unknown;
  headers?: Record<string, string>;
  signal?: AbortSignal;
  responseType?: string;
  _retry?: boolean;
}

export type AuthRedirectHandler = (to: string) => void;

let redirectHandler: AuthRedirectHandler = (to: string) => {
  if (typeof window !== 'undefined' && window.location) {
    if (!window.location.pathname.startsWith('/login') && !window.location.pathname.startsWith('/setup')) {
      window.location.href = to;
    }
  }
};

export const setAuthRedirectHandler = (handler: AuthRedirectHandler) => {
  redirectHandler = handler;
};

// Global shared promise to deduplicate concurrent silent token refresh requests
let refreshPromise: Promise<string | null> | null = null;

function isAuthEndpoint(url: string): boolean {
  return (
    url.includes('/auth/login') ||
    url.includes('/auth/setup') ||
    url.includes('/auth/refresh')
  );
}

export const customInstance = async <T>(
  config: CustomInstanceConfig
): Promise<T> => {
  const { url, method = 'GET', params, headers, data, signal, responseType, _retry } = config;

  let requestUrl = url;
  if (!requestUrl.startsWith('http://') && !requestUrl.startsWith('https://')) {
    if (!requestUrl.startsWith('/api/v1')) {
      requestUrl = `/api/v1${requestUrl.startsWith('/') ? '' : '/'}${requestUrl}`;
    }
  }
  if (params) {
    const searchParams = new URLSearchParams();
    Object.entries(params).forEach(([key, value]) => {
      if (value !== undefined) {
        searchParams.append(key, String(value));
      }
    });
    const queryString = searchParams.toString();
    if (queryString) {
      requestUrl += (requestUrl.includes('?') ? '&' : '?') + queryString;
    }
  }

  const isFormData = typeof FormData !== 'undefined' && data instanceof FormData;
  const baseHeaders: Record<string, string> = isFormData ? {} : { 'Content-Type': 'application/json' };

  // Inject Authorization Bearer token from auth store if present and not overridden
  const authHeaders: Record<string, string> = {};
  const currentAccessToken = useAuthStore.getState().accessToken;
  if (currentAccessToken && (!headers || !headers['Authorization'])) {
    authHeaders['Authorization'] = `Bearer ${currentAccessToken}`;
  }

  const response = await fetch(requestUrl, {
    method,
    headers: {
      ...baseHeaders,
      ...authHeaders,
      ...headers,
    },
    body: data !== undefined ? (isFormData || typeof data === 'string' ? (data as BodyInit) : JSON.stringify(data)) : undefined,
    signal,
  });

  if (!response.ok) {
    // 401 Unauthorized handling with silent refresh retry
    if (response.status === 401 && !_retry && !isAuthEndpoint(requestUrl)) {
      const refreshToken = useAuthStore.getState().refreshToken;
      if (refreshToken) {
        if (!refreshPromise) {
          refreshPromise = (async () => {
            try {
              const refreshRes = await fetch('/api/v1/auth/refresh', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ refreshToken }),
              });

              if (!refreshRes.ok) {
                useAuthStore.getState().actions.clearAuth();
                redirectHandler('/login');
                return null;
              }

              const refreshData = await refreshRes.json();
              if (refreshData.accessToken) {
                useAuthStore.getState().actions.setAccessToken(refreshData.accessToken);
                if (refreshData.refreshToken) {
                  useAuthStore.getState().actions.setRefreshToken(refreshData.refreshToken);
                }
                return refreshData.accessToken as string;
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
        }

        const newAccessToken = await refreshPromise;
        if (newAccessToken) {
          return customInstance<T>({
            ...config,
            headers: {
              ...headers,
              Authorization: `Bearer ${newAccessToken}`,
            },
            _retry: true,
          });
        }
      } else {
        useAuthStore.getState().actions.clearAuth();
        redirectHandler('/login');
      }
    }

    let errorData;
    try {
      errorData = await response.json();
    } catch {
      errorData = { message: response.statusText || 'An unexpected error occurred' };
    }
    throw new Error(errorData.error || errorData.message || `HTTP ${response.status}`);
  }

  if (response.status === 204) {
    return {} as T;
  }

  if (responseType === 'blob') {
    return response.blob() as Promise<T>;
  }
  if (responseType === 'text') {
    return response.text() as Promise<T>;
  }

  return response.json() as Promise<T>;
};
