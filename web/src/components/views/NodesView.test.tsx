import { render, screen, fireEvent, cleanup, waitFor } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import React from 'react';
import { NodesView } from './NodesView';
import { useModalStore } from '../../shared/stores/useModalStore';
import * as nodeApi from '../../api/generated/nodes/nodes';

const mockDrainMutateAsync = vi.fn();
const mockActivateMutateAsync = vi.fn();
const mockDeleteMutateAsync = vi.fn();

vi.mock('../../api/generated/nodes/nodes', async (importOriginal) => {
  const actual = await importOriginal<typeof nodeApi>();
  return {
    ...actual,
    useListNodes: vi.fn(),
    useDrainNode: () => ({ mutateAsync: mockDrainMutateAsync }),
    useActivateNode: () => ({ mutateAsync: mockActivateMutateAsync }),
    useDeleteNode: () => ({ mutateAsync: mockDeleteMutateAsync }),
  };
});

vi.mock('../../api/generated/applications/applications', () => ({
  useListApplications: () => ({ data: [] }),
}));

vi.mock('../../api/generated/domains/domains', () => ({
  useListDomains: () => ({ data: [] }),
}));

describe('NodesView', () => {
  let queryClient: QueryClient;

  beforeEach(() => {
    queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    useModalStore.getState().actions.closeAllModals();
    vi.clearAllMocks();
    vi.spyOn(window, 'confirm').mockReturnValue(true);
  });

  afterEach(() => {
    cleanup();
  });

  const wrapper = ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );

  it('Given nodes in fleet When NodesView renders Then displays node details and fleet stats', () => {
    vi.mocked(nodeApi.useListNodes).mockReturnValue({
      data: [
        {
          id: 'node-1',
          name: 'worker-node-02',
          ipAddress: '192.168.1.50',
          workerPort: 8080,
          internalPort: 8081,
          status: 'active',
          isProtected: false,
          createdAt: new Date().toISOString(),
          updatedAt: new Date().toISOString(),
        },
      ],
      isLoading: false,
    } as any);

    render(<NodesView />, { wrapper });

    expect(screen.getByText('worker-node-02')).toBeDefined();
    expect(screen.getByText('192.168.1.50')).toBeDefined();
    expect(screen.getByText('active')).toBeDefined();
  });

  it('Given user clicks Add Fleet Node button When clicked Then opens showAddNodeModal', () => {
    vi.mocked(nodeApi.useListNodes).mockReturnValue({
      data: [],
      isLoading: false,
    } as any);

    render(<NodesView />, { wrapper });

    const addBtn = screen.getByText('Add Fleet Node');
    fireEvent.click(addBtn);

    expect(useModalStore.getState().showAddNodeModal).toBe(true);
  });

  it('Given an active node When user clicks Drain Node Then drain mutation is invoked', async () => {
    vi.mocked(nodeApi.useListNodes).mockReturnValue({
      data: [
        {
          id: 'node-2',
          name: 'worker-node-02',
          ipAddress: '192.168.1.50',
          workerPort: 8080,
          internalPort: 8081,
          status: 'active',
          isProtected: false,
          createdAt: new Date().toISOString(),
          updatedAt: new Date().toISOString(),
        },
      ],
      isLoading: false,
    } as any);

    mockDrainMutateAsync.mockResolvedValueOnce({});

    render(<NodesView />, { wrapper });

    const drainBtn = screen.getByText('Drain');
    fireEvent.click(drainBtn);

    await waitFor(() => {
      expect(mockDrainMutateAsync).toHaveBeenCalledWith({ id: 'node-2' });
    });
  });
});
