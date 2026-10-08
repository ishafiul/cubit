import { render, screen } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { DurableObjectsView } from './DurableObjectsView';

// Mock fetch globally for testing
beforeEach(() => {
  vi.stubGlobal('fetch', vi.fn().mockImplementation((url: string) => {
    if (url === '/api/v1/durable-objects') {
      return Promise.resolve({
        ok: true,
        json: () => Promise.resolve([
          {
            id: 'class-1',
            name: 'ChatRoom',
            className: 'ChatRoom',
            rpcFacets: ['room', 'session'],
            methods: ['fetch', 'join'],
            createdAt: '2026-10-08T00:00:00Z',
          },
        ]),
      });
    }
    if (url.includes('/instances')) {
      return Promise.resolve({
        ok: true,
        json: () => Promise.resolve([]),
      });
    }
    return Promise.resolve({ ok: true, json: () => Promise.resolve({}) });
  }));
});

describe('DurableObjectsView', () => {
  it('Given DurableObjectsView When rendered Then displays celld v0.6.2 guidance on plain classes and facet WebSockets', async () => {
    render(<DurableObjectsView />);

    expect(await screen.findByText(/Durable Objects & Facets/i)).toBeDefined();
    expect(await screen.findByText(/celld v0.6.2 Plain Classes & Facet WebSockets/i)).toBeDefined();
    expect(screen.getByText(/getDurableObjectClass\(\)/i)).toBeDefined();
    expect(screen.getByText(/accepts plain classes without extending runtime base/i)).toBeDefined();
    expect(screen.getByText(/WebSockets work in facets \(inbound and outbound\) natively\./i)).toBeDefined();
    expect(screen.getByText(/stub\.fetch\(\)/i)).toBeDefined();
    expect(screen.getByText(/rejects on owner failure without re-running\./i)).toBeDefined();
  });
});
