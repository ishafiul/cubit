import { render, screen, fireEvent, cleanup } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import React from 'react';
import { UpgradeModal } from './UpgradeModal';
import { useModalStore } from '../../shared/stores/useModalStore';
import * as runtimeApi from '../../api/generated/runtime/runtime';

vi.mock('../../api/generated/runtime/runtime', () => ({
  useUpgradeCelldDaemon: vi.fn(),
  getGetRuntimeStatusQueryKey: () => ['runtime', 'status'],
}));

describe('UpgradeModal', () => {
  let queryClient: QueryClient;
  const mutateAsyncMock = vi.fn();

  beforeEach(() => {
    queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    vi.mocked(runtimeApi.useUpgradeCelldDaemon).mockReturnValue({
      mutateAsync: mutateAsyncMock,
    } as any);
    useModalStore.getState().actions.closeAllModals();
    useModalStore.getState().actions.setShowUpgradeModal(true);
    vi.clearAllMocks();
  });

  afterEach(() => {
    cleanup();
  });

  const wrapper = ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );

  it('Given UpgradeModal is open When rendered Then displays target version 0.6.1 and LTX epoch GC guidance', () => {
    render(<UpgradeModal />, { wrapper });

    // Target version input defaults to 0.6.1
    const input = screen.getByRole('textbox') as HTMLInputElement;
    expect(input.value).toBe('0.6.1');
    expect(input.placeholder).toBe('0.6.1');

    // LTX epoch retention guidance is visible
    expect(screen.getByText(/LTX Epoch Retention & GC \(celld v0.6.1\)/)).toBeDefined();
    expect(screen.getByText(/CELLD_LTX_RETENTION_SECS/)).toBeDefined();
    expect(screen.getByText(/celld cell gc --dry-run/)).toBeDefined();
  });

  it('Given UpgradeModal When submitting form Then invokes upgradeCelldMutation with 0.6.1', async () => {
    mutateAsyncMock.mockResolvedValueOnce({ success: true });
    render(<UpgradeModal />, { wrapper });

    const submitBtn = screen.getByRole('button', { name: /Start Rolling Upgrade/i });
    fireEvent.click(submitBtn);

    expect(mutateAsyncMock).toHaveBeenCalledWith({
      data: { targetVersion: '0.6.1' },
    });
  });
});
