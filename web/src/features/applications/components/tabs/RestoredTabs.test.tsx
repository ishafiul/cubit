import { describe, it, expect, vi, afterEach, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor, cleanup } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import type { Application } from '../../../../api/model';
import { ApplicationStoreProvider } from '../../stores/applicationStore';
import { OverviewTab } from './OverviewTab';
import { CodeEditorTab } from './CodeEditorTab';
import { TriggersTab } from './TriggersTab';
import { SettingsTab } from './SettingsTab';
import { BuildsTab } from './BuildsTab';
import { BindingsTab } from './BindingsTab';

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
});

const mockApp: Application = {
  id: 'app-test-123',
  name: 'prod-worker',
  subdomain: 'prod-worker',
  sourceType: 'inline',
  inlineCode: 'export default { fetch() { return new Response("hello"); } };',
  status: 'running',
  compatibilityDate: '2024-09-23',
  compatibilityFlags: ['nodejs_compat'],
  memoryLimitMb: 256,
  maxDurationMs: 150,
  activeDeploymentId: 'dep-1',
  bindings: [
    { name: 'USERS_KV', type: 'kv_namespace', resourceId: 'kv-users' },
    { name: 'ANALYTICS_DB', type: 'd1_database', resourceId: 'db-analytics' },
  ],
  envVars: [
    { key: 'API_URL', value: 'https://api.example.com', isSecret: false },
    { key: 'JWT_SECRET', value: 'super-secret-token', isSecret: true },
  ],
  createdAt: '2026-09-25T10:00:00Z',
  updatedAt: '2026-09-25T10:00:00Z',
};

function renderWithProviders(ui: React.ReactElement, app: Application = mockApp) {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
    },
  });

  return render(
    <QueryClientProvider client={queryClient}>
      <ApplicationStoreProvider
        key={app.id}
        appId={app.id}
        initialTab="overview"
        initialCode={app.inlineCode || ''}
      >
        {ui}
      </ApplicationStoreProvider>
    </QueryClientProvider>
  );
}

describe('Given Restored Application Workbench Tabs', () => {
  beforeEach(() => {
    // Mock global fetch for fleet services and telemetry
    vi.stubGlobal(
      'fetch',
      vi.fn().mockImplementation((url: string) => {
        if (url.includes('/api/v1/cron')) {
          return Promise.resolve({
            ok: true,
            json: () =>
              Promise.resolve([
                {
                  id: 'cron-1',
                  name: 'Nightly Sync',
                  cron: '0 0 * * *',
                  target_app_id: 'app-test-123',
                  status: 'active',
                  last_run_at: '2026-09-26T00:00:00Z',
                },
              ]),
          });
        }
        if (url.includes('/api/v1/queues')) {
          return Promise.resolve({
            ok: true,
            json: () =>
              Promise.resolve([
                {
                  id: 'queue-1',
                  name: 'order-queue',
                  consumer_app_id: 'app-test-123',
                  max_batch_size: 20,
                  max_retries: 5,
                },
              ]),
          });
        }
        if (url.includes('/deployments')) {
          if (url.includes('/logs')) {
            return Promise.resolve({
              ok: true,
              json: () =>
                Promise.resolve([
                  {
                    timestamp: '2026-09-26T12:00:01Z',
                    step: 'BUILD',
                    message: 'Compiling worker bundle with esbuild',
                    level: 'info',
                  },
                ]),
            });
          }
          return Promise.resolve({
            ok: true,
            json: () =>
              Promise.resolve([
                {
                  id: 'dep-1',
                  buildVersion: 2,
                  status: 'active',
                  commitHash: 'abcdef123456',
                  commitMessage: 'feat: add telemetry metrics',
                  bundleSize: 1048576,
                  createdAt: '2026-09-26T12:00:00Z',
                },
                {
                  id: 'dep-0',
                  buildVersion: 1,
                  status: 'superseded',
                  commitHash: '0123456789ab',
                  commitMessage: 'chore: initial release',
                  bundleSize: 524288,
                  createdAt: '2026-09-25T12:00:00Z',
                },
              ]),
          });
        }
        return Promise.resolve({
          ok: true,
          json: () => Promise.resolve([]),
          text: () => Promise.resolve('export default {};'),
        });
      })
    );
  });

  describe('When OverviewTab is rendered', () => {
    it('Then displays the architecture configuration card and quotas', async () => {
      renderWithProviders(<OverviewTab app={mockApp} />);

      expect(screen.getByText('Worker Architecture & Configuration')).toBeDefined();
      expect(screen.getByText('2024-09-23')).toBeDefined();
      expect(screen.getByText(/256 MB RAM \/ 150 ms CPU/)).toBeDefined();
      expect(screen.getByText('nodejs_compat enabled')).toBeDefined();
      expect(screen.getByTestId('quick-action-code')).toBeDefined();
      expect(screen.getByTestId('quick-action-tail')).toBeDefined();
      expect(screen.getByTestId('overview-test-worker-btn')).toBeDefined();
    });

    it('Then clicking Test Worker Invocation calls onTestApp', () => {
      const onTestApp = vi.fn();
      renderWithProviders(<OverviewTab app={mockApp} onTestApp={onTestApp} />);

      const testBtn = screen.getByTestId('overview-test-worker-btn');
      fireEvent.click(testBtn);
      expect(onTestApp).toHaveBeenCalledWith(mockApp);
    });
  });

  describe('When CodeEditorTab is rendered', () => {
    it('Then renders Save & Deploy button for inline workers', () => {
      const onDeploy = vi.fn();
      renderWithProviders(<CodeEditorTab app={mockApp} onDeploy={onDeploy} />);

      expect(screen.getByTestId('editor-save-and-deploy-btn')).toBeDefined();
    });

    it('Then renders dedicated Git Repository Integration panel for git workers', () => {
      const gitApp: Application = {
        ...mockApp,
        sourceType: 'git',
        gitRepo: 'https://github.com/cubit/sample-worker',
        branch: 'staging',
        rootDir: 'services/worker',
      };

      renderWithProviders(<CodeEditorTab app={gitApp} />, gitApp);

      expect(screen.getByText('Git Repository Integration')).toBeDefined();
      expect(screen.getByTestId('update-branch-btn')).toBeDefined();
      expect(screen.getByTestId('update-rootdir-btn')).toBeDefined();
      expect(screen.getByText('Push-to-Deploy CI/CD Active')).toBeDefined();
    });
  });

  describe('When TriggersTab is rendered', () => {
    it('Then displays default subdomain ingress routes and add cron button', async () => {
      renderWithProviders(<TriggersTab app={mockApp} />);

      expect(screen.getByText('Default Subdomain Ingress')).toBeDefined();
      expect(screen.getByText('prod-worker.localhost')).toBeDefined();
      expect(screen.getByText('prod-worker.cubit.local')).toBeDefined();
      expect(screen.getByTestId('add-cron-trigger-btn')).toBeDefined();
    });

    it('Then clicking Add Cron Trigger opens the modal dialog', async () => {
      renderWithProviders(<TriggersTab app={mockApp} />);

      const addBtn = screen.getByTestId('add-cron-trigger-btn');
      fireEvent.click(addBtn);

      expect(screen.getByText('Trigger Name')).toBeDefined();
      expect(screen.getByText('Cron Expression (5-part format)')).toBeDefined();
    });
  });

  describe('When SettingsTab is rendered', () => {
    it('Then displays editable Runtime & Compatibility configuration form', () => {
      renderWithProviders(<SettingsTab app={mockApp} />);

      expect(screen.getByText('Runtime & Compatibility Settings')).toBeDefined();
      expect(screen.getByTestId('runtime-compat-date-input')).toBeDefined();
      expect(screen.getByTestId('runtime-memory-select')).toBeDefined();
      expect(screen.getByTestId('runtime-cpu-timeout-input')).toBeDefined();
      expect(screen.getByTestId('runtime-nodejs-compat-checkbox')).toBeDefined();
      expect(screen.getByTestId('save-runtime-settings-btn')).toBeDefined();
    });

    it('Then filters environment variables by search input and type filter pills', async () => {
      renderWithProviders(<SettingsTab app={mockApp} />);

      expect(screen.getByTestId('env-row-API_URL')).toBeDefined();
      expect(screen.getByTestId('env-row-JWT_SECRET')).toBeDefined();

      // Click Plain Text filter pill
      const plainBtn = screen.getByTestId('filter-plain-btn');
      fireEvent.click(plainBtn);

      expect(screen.getByTestId('env-row-API_URL')).toBeDefined();
      expect(screen.queryByTestId('env-row-JWT_SECRET')).toBeNull();

      // Click Secret filter pill
      const secretBtn = screen.getByTestId('filter-secret-btn');
      fireEvent.click(secretBtn);

      expect(screen.queryByTestId('env-row-API_URL')).toBeNull();
      expect(screen.getByTestId('env-row-JWT_SECRET')).toBeDefined();

      // Reset to All and filter by search term
      fireEvent.click(screen.getByTestId('filter-all-btn'));
      const searchInput = screen.getByTestId('var-search-input');
      fireEvent.change(searchInput, { target: { value: 'API' } });

      expect(screen.getByTestId('env-row-API_URL')).toBeDefined();
      expect(screen.queryByTestId('env-row-JWT_SECRET')).toBeNull();
    });
  });

  describe('When BuildsTab is rendered', () => {
    it('Then displays two-column deployment history and bundle download button', async () => {
      renderWithProviders(<BuildsTab app={mockApp} />);

      await waitFor(() => {
        expect(screen.getByTestId('download-bundle-btn')).toBeDefined();
      });
    });

    it('Then clicking Rollback button on superseded deployment opens confirmation modal', async () => {
      renderWithProviders(<BuildsTab app={mockApp} />);

      await waitFor(() => {
        expect(screen.getByTestId('rollback-btn-dep-0')).toBeDefined();
      });

      fireEvent.click(screen.getByTestId('rollback-btn-dep-0'));

      expect(screen.getByText('Confirm Zero-Downtime Rollback')).toBeDefined();
      expect(screen.getByTestId('confirm-rollback-btn')).toBeDefined();
    });
  });

  describe('When BindingsTab is rendered', () => {
    it('Then displays injected bindings and attached fleet services section', async () => {
      const onNavigateToService = vi.fn();
      renderWithProviders(
        <BindingsTab app={mockApp} onNavigateToService={onNavigateToService} />
      );

      expect(screen.getByText('Cloudflare Compatible Resource Bindings (env.*)')).toBeDefined();
      expect(screen.getByTestId('binding-row-USERS_KV')).toBeDefined();
      expect(screen.getByTestId('binding-row-ANALYTICS_DB')).toBeDefined();
      expect(screen.getByText('Attached Fleet Services & Triggers')).toBeDefined();
    });
  });
});
