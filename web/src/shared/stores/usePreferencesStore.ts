import { create } from 'zustand';
import { persist } from 'zustand/middleware';

export type ThemeMode = 'dark' | 'light' | 'system';

export interface PreferencesState {
  theme: ThemeMode;
  sidebarCollapsed: boolean;
  editorTabSize: number;
  editorWordWrap: boolean;
  logAutoScroll: boolean;
}

export interface PreferencesActions {
  setTheme: (theme: ThemeMode) => void;
  toggleSidebar: () => void;
  setEditorTabSize: (size: number) => void;
  toggleEditorWordWrap: () => void;
  toggleLogAutoScroll: () => void;
  resetPreferences: () => void;
}

export type PreferencesStore = PreferencesState & { actions: PreferencesActions };

const defaultPreferences: PreferencesState = {
  theme: 'dark',
  sidebarCollapsed: false,
  editorTabSize: 2,
  editorWordWrap: true,
  logAutoScroll: true,
};

export const usePreferencesStore = create<PreferencesStore>()(
  persist(
    (set) => {
      const actions: PreferencesActions = {
        setTheme: (theme) => set({ theme }),
        toggleSidebar: () => set((state) => ({ sidebarCollapsed: !state.sidebarCollapsed })),
        setEditorTabSize: (editorTabSize) => set({ editorTabSize }),
        toggleEditorWordWrap: () =>
          set((state) => ({ editorWordWrap: !state.editorWordWrap })),
        toggleLogAutoScroll: () =>
          set((state) => ({ logAutoScroll: !state.logAutoScroll })),
        resetPreferences: () => set({ ...defaultPreferences }),
      };

      return {
        ...defaultPreferences,
        actions,
      };
    },
    {
      name: 'cubit:preferences',
      partialize: (state) => ({
        theme: state.theme,
        sidebarCollapsed: state.sidebarCollapsed,
        editorTabSize: state.editorTabSize,
        editorWordWrap: state.editorWordWrap,
        logAutoScroll: state.logAutoScroll,
      }),
    }
  )
);

export function usePreferencesActions(): PreferencesActions {
  return usePreferencesStore((state) => state.actions);
}

export function useTheme(): ThemeMode {
  return usePreferencesStore((state) => state.theme);
}

export function useSidebarCollapsed(): boolean {
  return usePreferencesStore((state) => state.sidebarCollapsed);
}

export function useEditorPreferences() {
  const editorTabSize = usePreferencesStore((state) => state.editorTabSize);
  const editorWordWrap = usePreferencesStore((state) => state.editorWordWrap);
  return { editorTabSize, editorWordWrap };
}
