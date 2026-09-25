import { describe, it, expect, beforeEach } from 'vitest';
import { renderHook } from '@testing-library/react';
import { usePreferencesStore, usePreferencesActions } from './usePreferencesStore';

describe('Given the global usePreferencesStore with persist middleware', () => {
  beforeEach(() => {
    localStorage.clear();
    usePreferencesStore.getState().actions.resetPreferences();
  });

  describe('When modifying preferences', () => {
    it('Then updates theme and persists to localStorage', () => {
      const { setTheme } = usePreferencesStore.getState().actions;
      setTheme('light');

      expect(usePreferencesStore.getState().theme).toBe('light');

      const stored = JSON.parse(localStorage.getItem('cubit:preferences') || '{}');
      expect(stored.state?.theme).toBe('light');
    });

    it('Then toggles sidebarCollapsed flag', () => {
      const { toggleSidebar } = usePreferencesStore.getState().actions;
      const initial = usePreferencesStore.getState().sidebarCollapsed;

      toggleSidebar();
      expect(usePreferencesStore.getState().sidebarCollapsed).toBe(!initial);

      toggleSidebar();
      expect(usePreferencesStore.getState().sidebarCollapsed).toBe(initial);
    });

    it('Then updates editor preferences (tabSize and wordWrap)', () => {
      const { setEditorTabSize, toggleEditorWordWrap } = usePreferencesStore.getState().actions;

      setEditorTabSize(4);
      expect(usePreferencesStore.getState().editorTabSize).toBe(4);

      const initialWrap = usePreferencesStore.getState().editorWordWrap;
      toggleEditorWordWrap();
      expect(usePreferencesStore.getState().editorWordWrap).toBe(!initialWrap);
    });

    it('Then resetPreferences restores defaults', () => {
      const { setTheme, setEditorTabSize, resetPreferences } =
        usePreferencesStore.getState().actions;

      setTheme('light');
      setEditorTabSize(8);

      resetPreferences();

      expect(usePreferencesStore.getState().theme).toBe('dark');
      expect(usePreferencesStore.getState().editorTabSize).toBe(2);
    });
  });

  describe('When accessing usePreferencesActions', () => {
    it('Then actions reference is stable across state changes', () => {
      const { result, rerender } = renderHook(() => usePreferencesActions());
      const initialActions = result.current;

      usePreferencesStore.getState().actions.toggleSidebar();
      rerender();

      expect(result.current).toBe(initialActions);
    });
  });
});
