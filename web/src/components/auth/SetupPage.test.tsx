import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { render, screen, fireEvent, waitFor, cleanup } from '@testing-library/react';
import { SetupPage } from './SetupPage';
import { useAuthStore } from '../../shared/stores/useAuthStore';

// Mock tanstack router navigation
const mockNavigate = vi.fn();
vi.mock('@tanstack/react-router', () => ({
  useNavigate: () => mockNavigate,
  Link: ({ children, to }: { children: React.ReactNode; to: string }) => (
    <a href={to}>{children}</a>
  ),
}));

describe('Given the SetupPage component (Initial Cluster Onboarding)', () => {
  let originalFetch: typeof global.fetch;

  beforeEach(() => {
    originalFetch = global.fetch;
    useAuthStore.getState().actions.clearAuth();
    mockNavigate.mockReset();
    vi.clearAllMocks();
  });

  afterEach(() => {
    cleanup();
    global.fetch = originalFetch;
  });

  describe('When the setup wizard renders', () => {
    it('Then displays name, email, password, and confirm password fields', () => {
      render(<SetupPage />);

      expect(screen.getByLabelText(/full name/i)).toBeDefined();
      expect(screen.getByLabelText(/email address/i)).toBeDefined();
      expect(screen.getByLabelText(/^password$/i)).toBeDefined();
      expect(screen.getByLabelText(/confirm password/i)).toBeDefined();
      expect(screen.getByRole('button', { name: /create administrator/i })).toBeDefined();
    });
  });

  describe('When submitting with mismatched passwords', () => {
    it('Then displays validation error and does not call network', async () => {
      global.fetch = vi.fn();
      render(<SetupPage />);

      fireEvent.change(screen.getByLabelText(/full name/i), {
        target: { value: 'Admin User' },
      });
      fireEvent.change(screen.getByLabelText(/email address/i), {
        target: { value: 'admin@cubit.dev' },
      });
      fireEvent.change(screen.getByLabelText(/^password$/i), {
        target: { value: 'password123' },
      });
      fireEvent.change(screen.getByLabelText(/confirm password/i), {
        target: { value: 'mismatched123' },
      });

      fireEvent.click(screen.getByRole('button', { name: /create administrator/i }));

      await waitFor(() => {
        expect(screen.getByText(/passwords do not match/i)).toBeDefined();
      });

      expect(global.fetch).not.toHaveBeenCalled();
    });
  });

  describe('When successfully completing initial setup', () => {
    it('Then creates root admin account, stores session, and navigates to /apps', async () => {
      global.fetch = vi.fn().mockImplementation((url: string) => {
        if (url.includes('/api/v1/auth/setup')) {
          return Promise.resolve(
            new Response(
              JSON.stringify({
                accessToken: 'admin-jwt-token-789',
                refreshToken: 'admin-refresh-token-012',
                expiresIn: 900,
                user: {
                  id: 'usr-root-admin',
                  name: 'System Admin',
                  email: 'root@cubit.dev',
                  roleId: 'admin',
                  isActive: true,
                },
              }),
              { status: 201 }
            )
          );
        }
        return Promise.reject(new Error(`unexpected ${url}`));
      });

      render(<SetupPage />);

      fireEvent.change(screen.getByLabelText(/full name/i), {
        target: { value: 'System Admin' },
      });
      fireEvent.change(screen.getByLabelText(/email address/i), {
        target: { value: 'root@cubit.dev' },
      });
      fireEvent.change(screen.getByLabelText(/^password$/i), {
        target: { value: 'supersecretpass123' },
      });
      fireEvent.change(screen.getByLabelText(/confirm password/i), {
        target: { value: 'supersecretpass123' },
      });

      fireEvent.click(screen.getByRole('button', { name: /create administrator/i }));

      await waitFor(() => {
        expect(useAuthStore.getState().isAuthenticated).toBe(true);
        expect(useAuthStore.getState().accessToken).toBe('admin-jwt-token-789');
        expect(useAuthStore.getState().user?.email).toBe('root@cubit.dev');
        expect(mockNavigate).toHaveBeenCalledWith({ to: '/apps' });
      });
    });
  });

  describe('When setup was already completed by another user', () => {
    it('Then displays conflict error message', async () => {
      global.fetch = vi.fn().mockImplementation((url: string) => {
        if (url.includes('/api/v1/auth/setup')) {
          return Promise.resolve(
            new Response(
              JSON.stringify({ error: 'setup has already been completed' }),
              { status: 409 }
            )
          );
        }
        return Promise.reject(new Error(`unexpected ${url}`));
      });

      render(<SetupPage />);

      fireEvent.change(screen.getByLabelText(/full name/i), {
        target: { value: 'Late Admin' },
      });
      fireEvent.change(screen.getByLabelText(/email address/i), {
        target: { value: 'late@cubit.dev' },
      });
      fireEvent.change(screen.getByLabelText(/^password$/i), {
        target: { value: 'validpassword123' },
      });
      fireEvent.change(screen.getByLabelText(/confirm password/i), {
        target: { value: 'validpassword123' },
      });

      fireEvent.click(screen.getByRole('button', { name: /create administrator/i }));

      await waitFor(() => {
        expect(screen.getByText(/setup has already been completed/i)).toBeDefined();
        expect(useAuthStore.getState().isAuthenticated).toBe(false);
      });
    });
  });
});
