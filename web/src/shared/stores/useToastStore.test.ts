import { describe, it, expect, beforeEach, vi } from 'vitest';
import { renderHook } from '@testing-library/react';
import { useToastStore, useToastActions } from './useToastStore';

describe('Given the global useToastStore', () => {
  beforeEach(() => {
    useToastStore.getState().actions.clearToasts();
    vi.useRealTimers();
  });

  describe('When adding toasts', () => {
    it('Then a new toast is appended with a generated id', () => {
      const { addToast } = useToastStore.getState().actions;
      const id = addToast({
        title: 'Deployment Succeeded',
        variant: 'success',
      });

      const toasts = useToastStore.getState().toasts;
      expect(toasts.length).toBe(1);
      expect(toasts[0].id).toBe(id);
      expect(toasts[0].title).toBe('Deployment Succeeded');
      expect(toasts[0].variant).toBe('success');
    });

    it('Then auto-dismisses after the specified duration', () => {
      vi.useFakeTimers();
      const { addToast } = useToastStore.getState().actions;

      addToast({
        title: 'Transient Notice',
        variant: 'info',
        duration: 3000,
      });

      expect(useToastStore.getState().toasts.length).toBe(1);

      vi.advanceTimersByTime(3000);

      expect(useToastStore.getState().toasts.length).toBe(0);
    });
  });

  describe('When dismissing toasts manually', () => {
    it('Then dismissToast removes the target toast by id', () => {
      const { addToast, dismissToast } = useToastStore.getState().actions;
      const id1 = addToast({ title: 'First', variant: 'info' });
      const id2 = addToast({ title: 'Second', variant: 'warning' });

      expect(useToastStore.getState().toasts.length).toBe(2);

      dismissToast(id1);

      const remaining = useToastStore.getState().toasts;
      expect(remaining.length).toBe(1);
      expect(remaining[0].id).toBe(id2);
    });

    it('Then clearToasts removes all active toasts', () => {
      const { addToast, clearToasts } = useToastStore.getState().actions;
      addToast({ title: 'A', variant: 'info' });
      addToast({ title: 'B', variant: 'error' });

      expect(useToastStore.getState().toasts.length).toBe(2);
      clearToasts();
      expect(useToastStore.getState().toasts.length).toBe(0);
    });
  });

  describe('When using useToastActions', () => {
    it('Then actions reference is stable across state changes', () => {
      const { result, rerender } = renderHook(() => useToastActions());
      const initialActions = result.current;

      useToastStore.getState().actions.addToast({ title: 'Test', variant: 'info' });
      rerender();

      expect(result.current).toBe(initialActions);
    });
  });
});
