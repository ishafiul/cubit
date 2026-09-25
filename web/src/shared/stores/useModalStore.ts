import { create } from 'zustand';
import type { Application } from '../../api/model';

export interface ModalState {
  showNewAppModal: boolean;
  showAddNodeModal: boolean;
  showUpgradeModal: boolean;
  showNewDomainModal: boolean;
  testingApp: Application | null;
  historyApp: Application | null;
  viewingCodeApp: any;
}

export interface ModalActions {
  setShowNewAppModal: (show: boolean) => void;
  setShowAddNodeModal: (show: boolean) => void;
  setShowUpgradeModal: (show: boolean) => void;
  setShowNewDomainModal: (show: boolean) => void;
  setTestingApp: (app: Application | null) => void;
  setHistoryApp: (app: Application | null) => void;
  setViewingCodeApp: (app: any) => void;
  closeAllModals: () => void;
}

export type ModalStore = ModalState & { actions: ModalActions };

const defaultModalState: ModalState = {
  showNewAppModal: false,
  showAddNodeModal: false,
  showUpgradeModal: false,
  showNewDomainModal: false,
  testingApp: null,
  historyApp: null,
  viewingCodeApp: null,
};

export const useModalStore = create<ModalStore>((set) => {
  const actions: ModalActions = {
    setShowNewAppModal: (showNewAppModal) => set({ showNewAppModal }),
    setShowAddNodeModal: (showAddNodeModal) => set({ showAddNodeModal }),
    setShowUpgradeModal: (showUpgradeModal) => set({ showUpgradeModal }),
    setShowNewDomainModal: (showNewDomainModal) => set({ showNewDomainModal }),
    setTestingApp: (testingApp) => set({ testingApp }),
    setHistoryApp: (historyApp) => set({ historyApp }),
    setViewingCodeApp: (viewingCodeApp) => set({ viewingCodeApp }),
    closeAllModals: () => set({ ...defaultModalState }),
  };

  return {
    ...defaultModalState,
    actions,
  };
});

export function useModalActions(): ModalActions {
  return useModalStore((state) => state.actions);
}

export function useIsModalOpen(modalKey: keyof Omit<ModalState, 'testingApp' | 'historyApp' | 'viewingCodeApp'>): boolean {
  return useModalStore((state) => Boolean(state[modalKey]));
}
