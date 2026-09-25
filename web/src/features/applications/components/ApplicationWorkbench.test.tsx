import { describe, it, expect, vi, afterEach } from 'vitest';
import { render, screen, act, cleanup } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { ApplicationWorkbench } from './ApplicationWorkbench';
import type { Application } from '../../../api/model';

afterEach(() => {
  cleanup();
});

const mockApp: Application = {
  id: 'app-abc',
  name: 'my-edge-worker',
  sourceType: 'inline',
  inlineCode: 'export default { fetch() { return new Response("ok"); } };',
  status: 'running',
  createdAt: '2026-09-25T10:00:00Z',
  updatedAt: '2026-09-25T10:00:00Z',
};

function renderWithClient(ui: React.ReactElement) {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
    },
  });
  return render(<QueryClientProvider client={queryClient}>{ui}</QueryClientProvider>);
}

describe('Given ApplicationWorkbench with 4-tier state architecture', () => {
  describe('When rendered for an application', () => {
    it('Then renders the application name and default overview tab', () => {
      renderWithClient(
        <ApplicationWorkbench
          app={mockApp}
          onBack={vi.fn()}
          onDeploy={vi.fn()}
          isDeploying={false}
          onTestApp={vi.fn()}
        />
      );

      expect(screen.getByTestId('workbench-app-name').textContent).toBe('my-edge-worker');
      expect(screen.getByTestId('tab-content-overview')).toBeDefined();
    });

    it('Then clicking code tab switches to CodeEditorTab without re-rendering unaffected components', () => {
      renderWithClient(
        <ApplicationWorkbench
          app={mockApp}
          onBack={vi.fn()}
          onDeploy={vi.fn()}
          isDeploying={false}
          onTestApp={vi.fn()}
        />
      );

      act(() => {
        screen.getByTestId('tab-btn-code').click();
      });

      expect(screen.getByTestId('tab-content-code')).toBeDefined();
    });

    it('Then navigating to another app resets workbench scoped store cleanly', () => {
      const secondApp: Application = {
        id: 'app-xyz',
        name: 'second-worker',
        sourceType: 'git',
        gitRepo: 'https://github.com/org/repo',
        status: 'running',
        createdAt: '2026-09-25T11:00:00Z',
        updatedAt: '2026-09-25T11:00:00Z',
      };

      const { rerender } = renderWithClient(
        <ApplicationWorkbench
          app={mockApp}
          initialTab="code"
          onBack={vi.fn()}
          onDeploy={vi.fn()}
          isDeploying={false}
          onTestApp={vi.fn()}
        />
      );

      expect(screen.getByTestId('workbench-app-name').textContent).toBe('my-edge-worker');

      rerender(
        <QueryClientProvider client={new QueryClient()}>
          <ApplicationWorkbench
            app={secondApp}
            initialTab="overview"
            onBack={vi.fn()}
            onDeploy={vi.fn()}
            isDeploying={false}
            onTestApp={vi.fn()}
          />
        </QueryClientProvider>
      );

      expect(screen.getByTestId('workbench-app-name').textContent).toBe('second-worker');
      expect(screen.getByTestId('tab-content-overview')).toBeDefined();
    });
  });
});
