import { render, screen, fireEvent, cleanup, waitFor } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { KVView } from './KVView';

describe('KVView Import Flow', () => {
  const mockNamespaces = [
    {
      id: 'ns-123',
      name: 'user_cache',
      keyCount: 2,
      createdAt: '2026-09-25T10:00:00Z',
    },
  ];

  const mockPairs = [
    {
      namespaceId: 'ns-123',
      key: 'user:1',
      value: 'Alice',
      expirationTtl: 0,
      updatedAt: '2026-09-25T10:00:00Z',
    },
  ];

  beforeEach(() => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockImplementation((url: string, opts?: any) => {
        if (url === '/api/v1/kv/namespaces') {
          return Promise.resolve({
            ok: true,
            json: () => Promise.resolve(mockNamespaces),
          });
        }
        if (url === '/api/v1/kv/namespaces/ns-123/keys') {
          return Promise.resolve({
            ok: true,
            json: () => Promise.resolve(mockPairs),
          });
        }
        if (url === '/api/v1/kv/namespaces/ns-123/import' && opts?.method === 'POST') {
          return Promise.resolve({
            ok: true,
            json: () => Promise.resolve({ namespace_id: 'ns-123', total: 2, imported: 2 }),
          });
        }
        return Promise.resolve({
          ok: true,
          json: () => Promise.resolve({}),
        });
      })
    );
  });

  afterEach(() => {
    cleanup();
    vi.restoreAllMocks();
  });

  it('Given selected namespace When clicking Import from Cloudflare Then opens modal and allows importing JSON', async () => {
    render(<KVView initialSelectedId="ns-123" />);

    // Wait for namespace to load
    await waitFor(() => {
      expect(screen.getAllByText('user_cache').length).toBeGreaterThan(0);
    });

    // Find and click "Import from Cloudflare"
    const importBtn = screen.getByTitle('Import data from Cloudflare KV');
    expect(importBtn).toBeDefined();
    fireEvent.click(importBtn);

    // Modal title should appear
    expect(screen.getByText('Import Cloudflare KV Data')).toBeDefined();

    // Fill JSON dump into textarea
    const textarea = screen.getByPlaceholderText(/user:1/);
    fireEvent.change(textarea, {
      target: { value: JSON.stringify([{ key: 'user:2', value: 'Bob' }]) },
    });

    // Submit import
    const submitBtn = screen.getByRole('button', { name: /Import Keys/i });
    fireEvent.click(submitBtn);

    await waitFor(() => {
      expect(global.fetch).toHaveBeenCalledWith(
        '/api/v1/kv/namespaces/ns-123/import',
        expect.objectContaining({
          method: 'POST',
        })
      );
    });

    await waitFor(() => {
      expect(screen.getByText(/Successfully imported 2 keys/)).toBeDefined();
    });
  });
});
