import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { customInstance } from './custom-instance';

describe('Given customInstance fetch client', () => {
  let originalFetch: typeof global.fetch;

  beforeEach(() => {
    originalFetch = global.fetch;
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
