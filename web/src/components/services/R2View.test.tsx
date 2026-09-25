import { render, screen, fireEvent, cleanup, waitFor } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { R2View } from './R2View';

describe('R2View Batch Import Flow', () => {
  const mockBuckets = [
    {
      name: 'user-uploads',
      objectsCount: 0,
      sizeBytes: 0,
      createdAt: '2026-09-25T10:00:00Z',
      isSystem: false,
    },
  ];

  beforeEach(() => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockImplementation((url: string, opts?: any) => {
        if (url === '/api/v1/r2/buckets') {
          return Promise.resolve({
            ok: true,
            json: () => Promise.resolve(mockBuckets),
          });
        }
        if (url === '/api/v1/r2/buckets/user-uploads/objects') {
          return Promise.resolve({
            ok: true,
            json: () => Promise.resolve([]),
          });
        }
        if (url === '/api/v1/r2/buckets/user-uploads/import' && opts?.method === 'POST') {
          return Promise.resolve({
            ok: true,
            json: () => Promise.resolve({ bucket: 'user-uploads', imported: 2 }),
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

  it('Given custom bucket When clicking Migrate / Import Then opens modal and posts batch manifest', async () => {
    render(<R2View initialSelectedId="user-uploads" />);

    // Wait for bucket to load
    await waitFor(() => {
      expect(screen.getAllByText('user-uploads').length).toBeGreaterThan(0);
    });

    // Find and click "Migrate / Import"
    const importBtn = screen.getByTitle('Bulk import objects or migrate from Cloudflare R2');
    expect(importBtn).toBeDefined();
    fireEvent.click(importBtn);

    // Modal title should appear
    expect(screen.getByText('Import / Migrate R2 Objects')).toBeDefined();

    // Fill JSON manifest into textarea
    const textarea = screen.getByPlaceholderText(/docs\/readme\.txt/);
    fireEvent.change(textarea, {
      target: {
        value: JSON.stringify([
          { key: 'avatar.png', content: 'base64data', content_type: 'image/png' },
          { key: 'report.pdf', content: 'pdfdata', content_type: 'application/pdf' },
        ]),
      },
    });

    // Submit import
    const submitBtn = screen.getByRole('button', { name: /Import Objects/i });
    fireEvent.click(submitBtn);

    await waitFor(() => {
      expect(global.fetch).toHaveBeenCalledWith(
        '/api/v1/r2/buckets/user-uploads/import',
        expect.objectContaining({
          method: 'POST',
        })
      );
    });

    await waitFor(() => {
      expect(screen.getByText(/Imported 2 objects into bucket user-uploads/)).toBeDefined();
    });
  });
});
