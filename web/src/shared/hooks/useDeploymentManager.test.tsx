import { renderHook, act, cleanup } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import React from 'react';
import { useDeploymentManager } from './useDeploymentManager';
import { useDeploymentTrackerStore } from '../stores/useDeploymentTrackerStore';
import { useToastStore } from '../stores/useToastStore';

const mockMutateAsync = vi.fn();

vi.mock('../../api/generated/deployments/deployments', () => ({
  useDeployApplication: () => ({
    mutateAsync: mockMutateAsync,
  }),
}));

describe('useDeploymentManager', () => {
  let queryClient: QueryClient;

  beforeEach(() => {
    queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    useDeploymentTrackerStore.getState().actions.resetDeploymentTracker();
    useToastStore.getState().actions.clearToasts();
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      json: vi.fn().mockResolvedValue([]),
    }));
    vi.clearAllMocks();
  });

  afterEach(() => {
    cleanup();
  });

  const wrapper = ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );

  it('Given an application ID When deployApp is called Then it sets activeDeploymentId and dispatches toast', async () => {
    mockMutateAsync.mockResolvedValueOnce({ id: 'dep-12345678' });

    const { result } = renderHook(() => useDeploymentManager(), { wrapper });

    await act(async () => {
      await result.current.deployApp('app-1');
    });

    expect(mockMutateAsync).toHaveBeenCalledWith({
      id: 'app-1',
      data: { commitHash: 'HEAD' },
    });
    expect(useDeploymentTrackerStore.getState().activeDeploymentId).toBe('dep-12345678');
    expect(useToastStore.getState().toasts).toHaveLength(1);
    expect(useToastStore.getState().toasts[0].variant).toBe('info');
  });

  it('Given a failed deployment mutation When deployApp is called Then it resets isDeploying and dispatches error toast', async () => {
    mockMutateAsync.mockRejectedValueOnce(new Error('Deployment failed on host'));

    const { result } = renderHook(() => useDeploymentManager(), { wrapper });

    await act(async () => {
      await expect(result.current.deployApp('app-err')).rejects.toThrow('Deployment failed on host');
    });

    expect(useDeploymentTrackerStore.getState().isDeploying).toBeNull();
    expect(useToastStore.getState().toasts).toHaveLength(1);
    expect(useToastStore.getState().toasts[0].variant).toBe('error');
  });
});
