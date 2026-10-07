import { create } from 'zustand';
import { persist } from 'zustand/middleware';

export interface AuthUser {
  id: string;
  name: string;
  email: string;
  roleId: string;
  isActive: boolean;
  createdAt: string;
  updatedAt: string;
}

export interface AuthRole {
  id: string;
  name: string;
  description?: string;
  isSystem: boolean;
  permissions: string[];
  createdAt: string;
  updatedAt: string;
}

export function matchPermission(pattern: string, required: string): boolean {
  if (!pattern || !required) return false;
  if (pattern === '*' || pattern === required) return true;
  if (pattern.endsWith(':*')) {
    const prefix = pattern.slice(0, -2);
    if (required === prefix || required.startsWith(prefix + ':')) return true;
  }
  if (pattern.startsWith('*:')) {
    const suffix = pattern.slice(2);
    if (required === suffix || required.endsWith(':' + suffix)) return true;
  }
  return false;
}

export function hasPermission(userPerms: string[] = [], required: string): boolean {
  if (!required) return false;
  return userPerms.some((perm) => matchPermission(perm, required));
}

export interface AuthState {
  accessToken: string | null;
  refreshToken: string | null;
  user: AuthUser | null;
  role: AuthRole | null;
  permissions: string[];
  isAuthenticated: boolean;
}

export interface AuthActions {
  setAuth: (payload: {
    accessToken: string;
    refreshToken?: string | null;
    user?: AuthUser | null;
    role?: AuthRole | null;
    permissions?: string[];
  }) => void;
  setAccessToken: (accessToken: string) => void;
  setRefreshToken: (refreshToken: string | null) => void;
  setUser: (user: AuthUser | null) => void;
  setRole: (role: AuthRole | null) => void;
  clearAuth: () => void;
  hasPermission: (permission: string) => boolean;
}

export type AuthStore = AuthState & { actions: AuthActions };

const initialAuthState: AuthState = {
  accessToken: null,
  refreshToken: null,
  user: null,
  role: null,
  permissions: [],
  isAuthenticated: false,
};

export const useAuthStore = create<AuthStore>()(
  persist(
    (set, get) => {
      const actions: AuthActions = {
        setAuth: ({ accessToken, refreshToken, user, role, permissions }) => {
          const resolvedPermissions =
            permissions || role?.permissions || (user?.roleId === 'admin' ? ['*'] : []);
          set({
            accessToken,
            refreshToken: refreshToken !== undefined ? refreshToken : get().refreshToken,
            user: user !== undefined ? user : get().user,
            role: role !== undefined ? role : get().role,
            permissions: resolvedPermissions,
            isAuthenticated: Boolean(accessToken),
          });
        },
        setAccessToken: (accessToken) =>
          set({
            accessToken,
            isAuthenticated: Boolean(accessToken),
          }),
        setRefreshToken: (refreshToken) => set({ refreshToken }),
        setUser: (user) => set({ user }),
        setRole: (role) =>
          set({
            role,
            permissions: role?.permissions || get().permissions,
          }),
        clearAuth: () => set({ ...initialAuthState }),
        hasPermission: (permission: string) => {
          return hasPermission(get().permissions, permission);
        },
      };

      return {
        ...initialAuthState,
        actions,
      };
    },
    {
      name: 'cubit:auth',
      partialize: (state) => ({
        accessToken: state.accessToken,
        refreshToken: state.refreshToken,
        user: state.user,
        role: state.role,
        permissions: state.permissions,
        isAuthenticated: state.isAuthenticated,
      }),
    }
  )
);

export function useAuthActions(): AuthActions {
  return useAuthStore((state) => state.actions);
}

export function useCurrentUser(): AuthUser | null {
  return useAuthStore((state) => state.user);
}

export function useUserRole(): AuthRole | null {
  return useAuthStore((state) => state.role);
}

export function useIsAuthenticated(): boolean {
  return useAuthStore((state) => state.isAuthenticated);
}

export function useUserPermissions(): string[] {
  return useAuthStore((state) => state.permissions);
}

export function useHasPermission(permission: string): boolean {
  return useAuthStore((state) => state.actions.hasPermission(permission));
}
