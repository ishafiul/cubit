import { render, screen, fireEvent, cleanup } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import React from 'react';
import { NewAppModal } from './NewAppModal';
import { useModalStore } from '../../shared/stores/useModalStore';

vi.mock('@tanstack/react-router', () => ({
  useNavigate: () => vi.fn(),
}));

vi.mock('../../api/generated/applications/applications', () => ({
  useCreateApplication: () => ({ mutateAsync: vi.fn() }),
  getListApplicationsQueryKey: () => ['applications'],
}));

vi.mock('../../shared/hooks/useDeploymentManager', () => ({
  useDeploymentManager: () => ({ deployApp: vi.fn() }),
}));

describe('NewAppModal', () => {
  let queryClient: QueryClient;

  beforeEach(() => {
    queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    useModalStore.getState().actions.closeAllModals();
    useModalStore.getState().actions.setShowNewAppModal(true);
    vi.clearAllMocks();
  });

  afterEach(() => {
    cleanup();
  });

  const wrapper = ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );

  it('Given NewAppModal is open When switching to Python template Then textarea updates to celld v0.6.2 Python code', () => {
    const { container } = render(<NewAppModal />, { wrapper });

    // Initially JavaScript ESM
    const textarea = container.querySelector('textarea') as HTMLTextAreaElement;
    expect(textarea.value).toContain('export default');

    // Click Python button
    const pythonBtn = screen.getByText('Python');
    fireEvent.click(pythonBtn);

    // Now contains Python worker template
    expect(textarea.value).toContain('from js import Response');
    expect(textarea.value).toContain('async def fetch(request, env)');
    expect(textarea.value).toContain('celld v0.6.2');

    // Click JavaScript button to switch back
    const jsBtn = screen.getByText('JavaScript (ESM)');
    fireEvent.click(jsBtn);
    expect(textarea.value).toContain('export default');
  });
});
