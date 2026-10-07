import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { render, screen, fireEvent, waitFor, cleanup } from '@testing-library/react';
import { AccessManagementView } from './AccessManagementView';
import { useAuthStore } from '../../shared/stores/useAuthStore';

describe('Given the AccessManagementView component', () => {
  let originalFetch: typeof global.fetch;

  const mockTokens = [
    {
      id: 'tok-1',
      userId: 'usr-1',
      name: 'CI Deployment Token',
      roleId: 'developer',
      expiresAt: '2026-12-31T00:00:00Z',
      lastUsedAt: '2026-06-01T12:00:00Z',
      createdAt: '2026-01-01T00:00:00Z',
    },
  ];

  const mockRoles = [
    {
      id: 'admin',
      name: 'Admin',
      description: 'Root administrator',
      isSystem: true,
      permissions: ['*'],
      createdAt: '2026-01-01T00:00:00Z',
      updatedAt: '2026-01-01T00:00:00Z',
    },
    {
      id: 'role-deployer',
      name: 'Custom Deployer',
      description: 'Deploy only role',
      isSystem: false,
      permissions: ['apps:deploy', 'deployments:*'],
      createdAt: '2026-02-01T00:00:00Z',
      updatedAt: '2026-02-01T00:00:00Z',
    },
  ];

  const mockPermissions = [
    'apps:read',
    'apps:create',
    'apps:deploy',
    'deployments:*',
    'tokens:*',
    'roles:*',
  ];

  beforeEach(() => {
    originalFetch = global.fetch;
    vi.spyOn(window, 'confirm').mockReturnValue(true);
    useAuthStore.getState().actions.clearAuth();
    useAuthStore.getState().actions.setAuth({
      accessToken: 'valid-test-jwt',
      permissions: ['*'],
    });

    global.fetch = vi.fn().mockImplementation((url: string, opts?: RequestInit) => {
      if (url.includes('/api/v1/tokens')) {
        if (opts?.method === 'POST') {
          return Promise.resolve(
            new Response(
              JSON.stringify({
                token: 'cbt_supersecrettoken_abc123',
                apiToken: {
                  id: 'tok-2',
                  userId: 'usr-1',
                  name: 'New Worker Token',
                  roleId: 'developer',
                  createdAt: '2026-06-01T00:00:00Z',
                },
              }),
              { status: 201 }
            )
          );
        }
        if (opts?.method === 'DELETE') {
          return Promise.resolve(new Response(JSON.stringify({ status: 'deleted' }), { status: 200 }));
        }
        return Promise.resolve(new Response(JSON.stringify(mockTokens), { status: 200 }));
      }

      if (url.includes('/api/v1/roles')) {
        if (opts?.method === 'POST') {
          return Promise.resolve(
            new Response(
              JSON.stringify({
                id: 'role-new',
                name: 'New Custom Role',
                description: 'Created via test',
                isSystem: false,
                permissions: ['apps:read'],
                createdAt: '2026-06-01T00:00:00Z',
                updatedAt: '2026-06-01T00:00:00Z',
              }),
              { status: 201 }
            )
          );
        }
        if (opts?.method === 'DELETE') {
          return Promise.resolve(new Response(JSON.stringify({ status: 'deleted' }), { status: 200 }));
        }
        return Promise.resolve(new Response(JSON.stringify(mockRoles), { status: 200 }));
      }

      if (url.includes('/api/v1/permissions')) {
        return Promise.resolve(
          new Response(JSON.stringify({ permissions: mockPermissions }), { status: 200 })
        );
      }

      return Promise.reject(new Error(`unexpected fetch: ${url}`));
    });
  });

  afterEach(() => {
    cleanup();
    global.fetch = originalFetch;
    vi.restoreAllMocks();
  });

  describe('When viewing API Tokens tab', () => {
    it('Then renders the list of active API tokens', async () => {
      render(<AccessManagementView />);

      await waitFor(() => {
        expect(screen.getByText('CI Deployment Token')).toBeDefined();
        expect(screen.getByText('developer')).toBeDefined();
      });
    });

    it('Then allows generating a new token and displays the plaintext token once', async () => {
      render(<AccessManagementView />);

      await waitFor(() => {
        expect(screen.getByText('CI Deployment Token')).toBeDefined();
      });

      fireEvent.click(screen.getByRole('button', { name: /generate api token/i }));

      expect(screen.getByLabelText(/token name/i)).toBeDefined();

      fireEvent.change(screen.getByLabelText(/token name/i), {
        target: { value: 'New Worker Token' },
      });

      fireEvent.click(screen.getByRole('button', { name: /^generate$/i }));

      await waitFor(() => {
        expect(screen.getByText('cbt_supersecrettoken_abc123')).toBeDefined();
        expect(screen.getByText(/copy your personal access token now/i)).toBeDefined();
      });
    });

    it('Then allows revoking an active token', async () => {
      render(<AccessManagementView />);

      await waitFor(() => {
        expect(screen.getByText('CI Deployment Token')).toBeDefined();
      });

      fireEvent.click(screen.getByRole('button', { name: /revoke/i }));

      await waitFor(() => {
        expect(global.fetch).toHaveBeenCalledWith(
          expect.stringContaining('/api/v1/tokens/tok-1'),
          expect.objectContaining({ method: 'DELETE' })
        );
      });
    });
  });

  describe('When switching to Roles & Permissions tab', () => {
    it('Then displays system roles and custom roles', async () => {
      render(<AccessManagementView />);

      fireEvent.click(screen.getByRole('button', { name: /roles & permissions/i }));

      await waitFor(() => {
        expect(screen.getByText('Admin')).toBeDefined();
        expect(screen.getByText('Custom Deployer')).toBeDefined();
        expect(screen.getByText(/root administrator/i)).toBeDefined();
      });
    });

    it('Then allows creating a new custom role', async () => {
      render(<AccessManagementView />);

      fireEvent.click(screen.getByRole('button', { name: /roles & permissions/i }));

      await waitFor(() => {
        expect(screen.getByText('Admin')).toBeDefined();
      });

      fireEvent.click(screen.getByRole('button', { name: /create custom role/i }));

      fireEvent.change(screen.getByLabelText(/role name/i), {
        target: { value: 'New Custom Role' },
      });
      fireEvent.change(screen.getByLabelText(/description/i), {
        target: { value: 'Created via test' },
      });

      // Select a permission checkbox
      fireEvent.click(screen.getByLabelText('apps:read'));

      fireEvent.click(screen.getByRole('button', { name: /^create role$/i }));

      await waitFor(() => {
        expect(global.fetch).toHaveBeenCalledWith(
          expect.stringContaining('/api/v1/roles'),
          expect.objectContaining({
            method: 'POST',
            body: expect.stringContaining('New Custom Role'),
          })
        );
      });
    });
  });
});
