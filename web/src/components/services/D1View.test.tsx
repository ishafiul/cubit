import { render, screen, fireEvent, cleanup, waitFor } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { D1View } from './D1View';

describe('D1View Import Flow', () => {
  const mockDatabases = [
    {
      id: 'd1-123',
      name: 'analytics_db',
      sizeBytes: 4096,
      tablesCount: 1,
      createdAt: '2026-09-25T10:00:00Z',
    },
  ];

  beforeEach(() => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockImplementation((url: string, opts?: any) => {
        if (url === '/api/v1/d1/databases') {
          return Promise.resolve({
            ok: true,
            json: () => Promise.resolve(mockDatabases),
          });
        }
        if (url === '/api/v1/d1/databases/d1-123/import' && opts?.method === 'POST') {
          return Promise.resolve({
            ok: true,
            json: () => Promise.resolve({ database_id: 'd1-123', executed: 3, duration_ms: 12.5 }),
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

  it('Given selected database When clicking Import SQL Dump Then opens modal and executes migration', async () => {
    render(<D1View initialSelectedId="d1-123" />);

    // Wait for database to load
    await waitFor(() => {
      expect(screen.getAllByText('analytics_db').length).toBeGreaterThan(0);
    });

    // Find and click "Import SQL Dump"
    const importBtn = screen.getByTitle('Import Cloudflare D1 SQL dump');
    expect(importBtn).toBeDefined();
    fireEvent.click(importBtn);

    // Modal title should appear
    expect(screen.getByText('Import Cloudflare D1 SQL Dump')).toBeDefined();

    // Fill SQL into textarea
    const textarea = screen.getByPlaceholderText(/CREATE TABLE IF NOT EXISTS/);
    fireEvent.change(textarea, {
      target: { value: 'CREATE TABLE logs (id INT); INSERT INTO logs VALUES (1);' },
    });

    // Submit SQL migration
    const submitBtn = screen.getByRole('button', { name: /Run SQL Migration/i });
    fireEvent.click(submitBtn);

    await waitFor(() => {
      expect(global.fetch).toHaveBeenCalledWith(
        '/api/v1/d1/databases/d1-123/import',
        expect.objectContaining({
          method: 'POST',
        })
      );
    });

    await waitFor(() => {
      expect(screen.getByText(/Executed 3 SQL statements/)).toBeDefined();
    });
  });
});
