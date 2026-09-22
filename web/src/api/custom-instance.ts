// Custom Fetch client mutator for Orval

export interface CustomInstanceConfig {
  url: string;
  method?: string;
  params?: Record<string, string | number | boolean | undefined>;
  data?: unknown;
  headers?: Record<string, string>;
  signal?: AbortSignal;
  responseType?: string;
}

export const customInstance = async <T>(
  config: CustomInstanceConfig
): Promise<T> => {
  const { url, method = 'GET', params, headers, data, signal, responseType } = config;

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

  const response = await fetch(requestUrl, {
    method,
    headers: {
      ...baseHeaders,
      ...headers,
    },
    body: data !== undefined ? (isFormData || typeof data === 'string' ? (data as BodyInit) : JSON.stringify(data)) : undefined,
    signal,
  });

  if (!response.ok) {
    let errorData;
    try {
      errorData = await response.json();
    } catch {
      errorData = { message: response.statusText || 'An unexpected error occurred' };
    }
    throw new Error(errorData.message || `HTTP ${response.status}`);
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
