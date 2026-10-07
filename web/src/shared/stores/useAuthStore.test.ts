import { describe, it, expect, beforeEach } from 'vitest';
import { useAuthStore, matchPermission, hasPermission } from './useAuthStore';

describe('Given useAuthStore and permission matcher', () => {
  beforeEach(() => {
    useAuthStore.getState().actions.clearAuth();
  });

  describe('When evaluating matchPermission and hasPermission', () => {
    it('Then wildcard matches any required permission', () => {
      expect(matchPermission('*', 'apps:deploy')).toBe(true);
      expect(matchPermission('*', 'nodes:write')).toBe(true);
    });

    it('Then prefix wildcard matches matching domain actions', () => {
      expect(matchPermission('apps:*', 'apps:read')).toBe(true);
      expect(matchPermission('apps:*', 'apps:deploy')).toBe(true);
      expect(matchPermission('apps:*', 'services:read')).toBe(false);
    });

    it('Then suffix wildcard matches matching action types', () => {
      expect(matchPermission('*:read', 'apps:read')).toBe(true);
      expect(matchPermission('*:read', 'nodes:read')).toBe(true);
      expect(matchPermission('*:read', 'apps:deploy')).toBe(false);
    });

    it('Then exact match succeeds', () => {
      expect(matchPermission('apps:deploy', 'apps:deploy')).toBe(true);
      expect(matchPermission('apps:deploy', 'apps:read')).toBe(false);
    });

    it('Then hasPermission evaluates arrays of user permissions', () => {
      const perms = ['apps:read', 'nodes:*'];
      expect(hasPermission(perms, 'apps:read')).toBe(true);
      expect(hasPermission(perms, 'nodes:write')).toBe(true);
      expect(hasPermission(perms, 'apps:deploy')).toBe(false);
    });
  });

  describe('When the store is in its initial unauthenticated state', () => {
    it('Then state indicates no authenticated session', () => {
      const state = useAuthStore.getState();
      expect(state.accessToken).toBeNull();
      expect(state.refreshToken).toBeNull();
      expect(state.user).toBeNull();
      expect(state.role).toBeNull();
      expect(state.permissions).toEqual([]);
      expect(state.isAuthenticated).toBe(false);
      expect(state.actions.hasPermission('apps:read')).toBe(false);
    });
  });

  describe('When setting authentication credentials via setAuth', () => {
    const mockUser = {
      id: 'usr-admin',
      name: 'Root Administrator',
      email: 'admin@cubit.dev',
      roleId: 'admin',
      isActive: true,
      createdAt: '2026-01-01T00:00:00Z',
      updatedAt: '2026-01-01T00:00:00Z',
    };

    const mockRole = {
      id: 'admin',
      name: 'Admin',
      description: 'Superuser',
      isSystem: true,
      permissions: ['*'],
      createdAt: '2026-01-01T00:00:00Z',
      updatedAt: '2026-01-01T00:00:00Z',
    };

    it('Then sets tokens, user, role and marks user as authenticated', () => {
      useAuthStore.getState().actions.setAuth({
        accessToken: 'access-jwt-123',
        refreshToken: 'refresh-rot-456',
        user: mockUser,
        role: mockRole,
        permissions: ['*'],
      });

      const state = useAuthStore.getState();
      expect(state.accessToken).toBe('access-jwt-123');
      expect(state.refreshToken).toBe('refresh-rot-456');
      expect(state.user).toEqual(mockUser);
      expect(state.role).toEqual(mockRole);
      expect(state.permissions).toEqual(['*']);
      expect(state.isAuthenticated).toBe(true);
      expect(state.actions.hasPermission('deployments:rollback')).toBe(true);
    });

    it('Then updates accessToken and refreshToken independently', () => {
      useAuthStore.getState().actions.setAuth({
        accessToken: 'initial-access',
        refreshToken: 'initial-refresh',
        user: mockUser,
      });

      useAuthStore.getState().actions.setAccessToken('new-access-jwt');
      expect(useAuthStore.getState().accessToken).toBe('new-access-jwt');
      expect(useAuthStore.getState().user).toEqual(mockUser);

      useAuthStore.getState().actions.setRefreshToken('new-refresh-token');
      expect(useAuthStore.getState().refreshToken).toBe('new-refresh-token');
    });
  });

  describe('When logging out or clearing session via clearAuth', () => {
    it('Then resets all tokens and user metadata to initial values', () => {
      useAuthStore.getState().actions.setAuth({
        accessToken: 'valid-token',
        refreshToken: 'valid-refresh',
        user: {
          id: 'usr-1',
          name: 'Developer',
          email: 'dev@cubit.dev',
          roleId: 'developer',
          isActive: true,
          createdAt: '',
          updatedAt: '',
        },
        permissions: ['apps:*'],
      });

      expect(useAuthStore.getState().isAuthenticated).toBe(true);

      useAuthStore.getState().actions.clearAuth();

      const state = useAuthStore.getState();
      expect(state.accessToken).toBeNull();
      expect(state.refreshToken).toBeNull();
      expect(state.user).toBeNull();
      expect(state.role).toBeNull();
      expect(state.permissions).toEqual([]);
      expect(state.isAuthenticated).toBe(false);
      expect(state.actions.hasPermission('apps:read')).toBe(false);
    });
  });
});
