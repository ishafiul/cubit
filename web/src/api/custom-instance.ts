// Custom Fetch client mutator for Orval

export interface CustomInstanceConfig {
  url: string;
  method?: string;
  params?: Record<string, string | number | boolean | undefined>;
  data?: unknown;
  headers?: Record<string, string>;
  signal?: AbortSignal;
}

export const customInstance = async <T>(
  config: CustomInstanceConfig
): Promise<T> => {
  const { url, method = 'GET', params, headers, data, signal } = config;

  let requestUrl = url;
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

  const response = await fetch(requestUrl, {
    method,
    headers: {
      'Content-Type': 'application/json',
      ...headers,
    },
    body: data !== undefined ? (typeof data === 'string' ? data : JSON.stringify(data)) : undefined,
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

  return response.json() as Promise<T>;
};
