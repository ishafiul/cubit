import { describe, it, expect, beforeEach } from 'vitest';
import { renderHook } from '@testing-library/react';
import { useModalStore, useModalActions } from './useModalStore';

describe('Given the global useModalStore coordinator', () => {
  beforeEach(() => {
    useModalStore.getState().actions.closeAllModals();
  });

  describe('When managing modal visibility', () => {
    it('Then opens and closes newAppModal', () => {
      const { setShowNewAppModal } = useModalStore.getState().actions;

      expect(useModalStore.getState().showNewAppModal).toBe(false);

      setShowNewAppModal(true);
      expect(useModalStore.getState().showNewAppModal).toBe(true);

      setShowNewAppModal(false);
      expect(useModalStore.getState().showNewAppModal).toBe(false);
    });

    it('Then manages entity payloads for testingApp and historyApp', () => {
      const { setTestingApp, setHistoryApp } = useModalStore.getState().actions;
      const fakeApp = { id: 'app-1', name: 'worker-one' } as any;

      setTestingApp(fakeApp);
      expect(useModalStore.getState().testingApp).toEqual(fakeApp);

      setHistoryApp(fakeApp);
      expect(useModalStore.getState().historyApp).toEqual(fakeApp);

      setTestingApp(null);
      expect(useModalStore.getState().testingApp).toBeNull();
    });

    it('Then closeAllModals resets all modal flags and payloads to initial state', () => {
      const { setShowNewAppModal, setShowAddNodeModal, setTestingApp, closeAllModals } =
        useModalStore.getState().actions;

      setShowNewAppModal(true);
      setShowAddNodeModal(true);
      setTestingApp({ id: 'app-99', name: 'app' } as any);

      closeAllModals();

      const state = useModalStore.getState();
      expect(state.showNewAppModal).toBe(false);
      expect(state.showAddNodeModal).toBe(false);
      expect(state.showUpgradeModal).toBe(false);
      expect(state.showNewDomainModal).toBe(false);
      expect(state.testingApp).toBeNull();
      expect(state.historyApp).toBeNull();
      expect(state.viewingCodeApp).toBeNull();
    });
  });

  describe('When calling useModalActions', () => {
    it('Then actions reference is stable across state updates', () => {
      const { result, rerender } = renderHook(() => useModalActions());
      const initialActions = result.current;

      useModalStore.getState().actions.setShowUpgradeModal(true);
      rerender();

      expect(result.current).toBe(initialActions);
    });
  });
});
