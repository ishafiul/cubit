import { render, screen, fireEvent, cleanup } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import React from 'react';
import { WorkersView } from './WorkersView';
import { useModalStore } from '../../shared/stores/useModalStore';
import { useDeploymentTrackerStore } from '../../shared/stores/useDeploymentTrackerStore';
import * as appApi from '../../api/generated/applications/applications';

vi.mock('@tanstack/react-router', () => ({
  useNavigate: () => vi.fn(),
}));

vi.mock('../../api/generated/applications/applications', async (importOriginal) => {
  const actual = await importOriginal<typeof appApi>();
  return {
    ...actual,
    useListApplications: vi.fn(),
    useDeleteApplication: () => ({ mutateAsync: vi.fn() }),
  };
});

describe('WorkersView', () => {
  let queryClient: QueryClient;

  beforeEach(() => {
    queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    useModalStore.getState().actions.closeAllModals();
    useDeploymentTrackerStore.getState().actions.resetDeploymentTracker();
    vi.clearAllMocks();
  });

  afterEach(() => {
    cleanup();
  });

  const wrapper = ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );

  it('Given no applications When WorkersView renders Then displays empty state and opens NewAppModal on click', () => {
    vi.mocked(appApi.useListApplications).mockReturnValue({
      data: [],
      isLoading: false,
    } as any);

    render(<WorkersView />, { wrapper });

    expect(screen.getByText('No Cloudflare Workers Created')).toBeDefined();
    const createBtn = screen.getByText('Create Worker');
    fireEvent.click(createBtn);

    expect(useModalStore.getState().showNewAppModal).toBe(true);
  });

  it('Given existing applications When WorkersView renders Then lists application cards', () => {
    vi.mocked(appApi.useListApplications).mockReturnValue({
      data: [
        {
          id: 'app-1',
          name: 'payment-api',
          sourceType: 'inline',
          status: 'running',
          createdAt: new Date().toISOString(),
          updatedAt: new Date().toISOString(),
        },
      ],
      isLoading: false,
    } as any);

    render(<WorkersView />, { wrapper });

    expect(screen.getByText('payment-api')).toBeDefined();
  });

  it('Given active deployment logs When WorkersView renders Then shows live build stream', async () => {
    vi.mocked(appApi.useListApplications).mockReturnValue({
      data: [],
      isLoading: false,
    } as any);

    useDeploymentTrackerStore.getState().actions.setActiveDeploymentId('dep-abc');
    vi.stubGlobal(
      'fetch',
      vi.fn().mockImplementation((url: string) => {
        if (url.includes('/logs')) {
          return Promise.resolve({
            json: () =>
              Promise.resolve([
                {
                  timestamp: new Date().toISOString(),
                  step: 'esbuild',
                  message: 'Bundle generated successfully',
                  level: 'info',
                },
              ]),
          });
        }
        return Promise.resolve({
          json: () => Promise.resolve({ status: 'building' }),
        });
      })
    );

    render(<WorkersView />, { wrapper });

    expect(screen.getByText('Live Build & Deployment Stream')).toBeDefined();
    expect(await screen.findByText('Bundle generated successfully')).toBeDefined();
  });
});
