import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { render, screen, fireEvent, waitFor, cleanup } from '@testing-library/react';
import { LoginPage } from './LoginPage';
import { useAuthStore } from '../../shared/stores/useAuthStore';

// Mock tanstack router navigation
const mockNavigate = vi.fn();
vi.mock('@tanstack/react-router', () => ({
  useNavigate: () => mockNavigate,
  Link: ({ children, to }: { children: React.ReactNode; to: string }) => (
    <a href={to}>{children}</a>
  ),
}));

describe('Given the LoginPage component', () => {
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

  describe('When the login page renders', () => {
    it('Then displays email, password fields and sign in button', () => {
      render(<LoginPage />);

      expect(screen.getByLabelText(/email address/i)).toBeDefined();
      expect(screen.getByLabelText(/password/i)).toBeDefined();
      expect(screen.getByRole('button', { name: /sign in/i })).toBeDefined();
    });
  });

  describe('When submitting with empty credentials', () => {
    it('Then shows validation errors without making a network request', async () => {
      global.fetch = vi.fn();
      render(<LoginPage />);

      fireEvent.click(screen.getByRole('button', { name: /sign in/i }));

      await waitFor(() => {
        expect(screen.getByText(/email is required/i)).toBeDefined();
      });

      expect(global.fetch).not.toHaveBeenCalled();
    });
  });

  describe('When submitting valid credentials successfully', () => {
    it('Then calls auth API, updates useAuthStore and navigates to /apps', async () => {
      global.fetch = vi.fn().mockImplementation((url: string) => {
        if (url.includes('/api/v1/auth/login')) {
          return Promise.resolve(
            new Response(
              JSON.stringify({
                accessToken: 'jwt-access-token-123',
                refreshToken: 'refresh-token-456',
                expiresIn: 900,
                user: {
                  id: 'usr-1',
                  name: 'Alice',
                  email: 'alice@cubit.dev',
                  roleId: 'developer',
                  isActive: true,
                },
              }),
              { status: 200 }
            )
          );
        }
        return Promise.reject(new Error(`unexpected ${url}`));
      });

      render(<LoginPage />);

      fireEvent.change(screen.getByLabelText(/email address/i), {
        target: { value: 'alice@cubit.dev' },
      });
      fireEvent.change(screen.getByLabelText(/password/i), {
        target: { value: 'secretpassword123' },
      });

      fireEvent.click(screen.getByRole('button', { name: /sign in/i }));

      await waitFor(() => {
        expect(useAuthStore.getState().isAuthenticated).toBe(true);
        expect(useAuthStore.getState().accessToken).toBe('jwt-access-token-123');
        expect(useAuthStore.getState().user?.email).toBe('alice@cubit.dev');
        expect(mockNavigate).toHaveBeenCalledWith({ to: '/apps' });
      });
    });
  });

  describe('When backend returns 401 invalid credentials', () => {
    it('Then displays the server error message to the user', async () => {
      global.fetch = vi.fn().mockImplementation((url: string) => {
        if (url.includes('/api/v1/auth/login')) {
          return Promise.resolve(
            new Response(
              JSON.stringify({ error: 'invalid email or password' }),
              { status: 401 }
            )
          );
        }
        return Promise.reject(new Error(`unexpected ${url}`));
      });

      render(<LoginPage />);

      fireEvent.change(screen.getByLabelText(/email address/i), {
        target: { value: 'alice@cubit.dev' },
      });
      fireEvent.change(screen.getByLabelText(/password/i), {
        target: { value: 'wrongpass' },
      });

      fireEvent.click(screen.getByRole('button', { name: /sign in/i }));

      await waitFor(() => {
        expect(screen.getByText(/invalid email or password/i)).toBeDefined();
        expect(useAuthStore.getState().isAuthenticated).toBe(false);
        expect(mockNavigate).not.toHaveBeenCalled();
      });
    });
  });
});
